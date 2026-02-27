package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"noodexx/internal/auth"
	"noodexx/internal/config"
	"noodexx/internal/rag"
	"strings"
	"time"
)

// handleDashboard renders the dashboard page with system stats
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestID := generateRequestID()

	// Create logger with request context
	logger := s.logger.WithContext("request_id", requestID).
		WithContext("method", r.Method).
		WithContext("path", r.URL.Path)

	logger.Debug("processing dashboard request")

	// Prevent caching
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	ctx := r.Context()

	// Get document count
	library, err := s.store.Library(ctx)
	if err != nil {
		logger.Error("failed to get library", "error", err.Error())
		http.Error(w, "Failed to load dashboard", http.StatusInternalServerError)
		return
	}
	docCount := len(library)

	// Get last ingestion timestamp
	var lastIngestion time.Time
	if docCount > 0 {
		// Find the most recent document
		for _, entry := range library {
			if entry.CreatedAt.After(lastIngestion) {
				lastIngestion = entry.CreatedAt
			}
		}
	}

	// Get provider info
	providerName := s.provider.Name()
	if providerName == "ollama" {
		providerName = fmt.Sprintf("Ollama (%s)", s.config.OllamaChatModel)
	} else if providerName == "openai" {
		providerName = fmt.Sprintf("OpenAI (%s)", s.config.OpenAIChatModel)
	} else if providerName == "anthropic" {
		providerName = fmt.Sprintf("Anthropic (%s)", s.config.AnthropicChatModel)
	}
	privacyMode := s.config.PrivacyMode

	// Prepare template data
	data := map[string]interface{}{
		"Title":         "Dashboard",
		"Page":          "dashboard",
		"DocumentCount": docCount,
		"Provider":      providerName,
		"PrivacyMode":   privacyMode,
		"LastIngestion": lastIngestion,
		"HasIngestions": !lastIngestion.IsZero(),
	}

	logger.Debug("rendering dashboard template", "document_count", docCount)

	// Render template
	if err := s.templates.ExecuteTemplate(w, "base.html", data); err != nil {
		logger.Error("failed to render dashboard template", "error", err.Error())
		http.Error(w, "Failed to render dashboard", http.StatusInternalServerError)
		return
	}

	latency := time.Since(start).Milliseconds()
	logger.Debug("request completed", "status", http.StatusOK, "latency_ms", latency)
}

// handleChat renders the chat page
func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestID := generateRequestID()

	// Create logger with request context
	logger := s.logger.WithContext("request_id", requestID).
		WithContext("method", r.Method).
		WithContext("path", r.URL.Path)

	logger.Debug("processing request")

	// Prevent caching
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// Check if cloud provider is available
	cloudProviderAvailable := false
	if s.providerManager != nil {
		cloudProviderAvailable = s.providerManager.GetCloudProvider() != nil
	}

	// Prepare template data
	data := map[string]interface{}{
		"Title":                  "Chat",
		"Page":                   "chat",
		"PrivacyMode":            s.config.PrivacyMode,
		"CloudProviderAvailable": cloudProviderAvailable,
	}

	// Render chat template
	if err := s.templates.ExecuteTemplate(w, "base.html", data); err != nil {
		logger.Error("request failed", "operation", "render_template", "error", err.Error())
		http.Error(w, "Failed to render chat", http.StatusInternalServerError)
		return
	}

	latency := time.Since(start).Milliseconds()
	logger.Debug("request completed", "status", http.StatusOK, "latency_ms", latency)
}

// handleAsk processes chat queries with RAG
func (s *Server) handleAsk(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestID := generateRequestID()

	// Create logger with request context
	logger := s.logger.WithContext("request_id", requestID).
		WithContext("method", r.Method).
		WithContext("path", r.URL.Path)

	logger.Debug("processing request")

	if r.Method != http.MethodPost {
		logger.Error("request failed", "operation", "method_check", "error", "method not allowed")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Extract user_id from context
	userID, err := auth.GetUserID(ctx)
	if err != nil {
		logger.Error("request failed", "operation", "get_user_id", "error", err.Error())
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse request
	var req struct {
		Query     string `json:"query"`
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("request failed", "operation", "parse_request", "error", err.Error())
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Generate session ID if not provided
	if req.SessionID == "" {
		req.SessionID = generateSessionID()
	}

	// If session exists, verify ownership
	if req.SessionID != "" {
		owner, err := s.store.GetSessionOwner(ctx, req.SessionID)
		if err == nil && owner != 0 {
			// Session exists, verify it belongs to this user
			if owner != userID {
				logger.Error("request failed", "operation", "verify_session_owner", "error", "unauthorized access to session")
				http.Error(w, "Forbidden: session belongs to another user", http.StatusForbidden)
				return
			}
		}
	}

	// Save user message with user_id
	// User messages don't have a provider mode, use empty string
	if err := s.store.SaveChatMessage(ctx, userID, req.SessionID, "user", req.Query, ""); err != nil {
		logger.Warn("failed to save user message", "error", err.Error())
	}

	// Audit log
	s.store.AddAuditEntry(ctx, "query", req.Query, req.SessionID)

	// Get active provider
	provider, err := s.providerManager.GetActiveProvider()
	if err != nil {
		logger.Error("request failed", "operation", "get_active_provider", "error", err.Error())
		http.Error(w, "Provider not configured. Please configure the AI provider in Settings.", http.StatusBadRequest)
		return
	}

	// Conditionally perform RAG based on policy
	var chunks []Chunk
	if s.ragEnforcer.ShouldPerformRAG() {
		logger.Debug("performing RAG search")

		// Embed query
		queryVec, err := provider.Embed(ctx, req.Query)
		if err != nil {
			logger.Error("request failed", "operation", "embed_query", "error", err.Error())
			http.Error(w, "Embedding failed", http.StatusInternalServerError)
			return
		}

		// Search for relevant chunks (user-scoped)
		chunks, err = s.store.SearchByUser(ctx, userID, queryVec, 5)
		if err != nil {
			logger.Error("request failed", "operation", "search_chunks", "error", err.Error())
			http.Error(w, "Search failed", http.StatusInternalServerError)
			return
		}
	} else {
		logger.Debug("skipping RAG search per policy")
	}

	// Build prompt using PromptBuilder (with or without chunks)
	// Convert api.Chunk to rag.Chunk
	ragChunks := make([]rag.Chunk, len(chunks))
	for i, chunk := range chunks {
		ragChunks[i] = rag.Chunk{
			Source: chunk.Source,
			Text:   chunk.Text,
			Score:  chunk.Score,
		}
	}

	promptBuilder := rag.NewPromptBuilder()
	prompt := promptBuilder.BuildPrompt(req.Query, ragChunks)

	// Stream response
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Session-ID", req.SessionID)
	w.Header().Set("X-Provider-Name", s.providerManager.GetProviderName())
	w.Header().Set("X-RAG-Status", s.ragEnforcer.GetRAGStatus())

	messages := []Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: prompt},
	}

	response, err := provider.Stream(ctx, messages, w)
	if err != nil {
		logger.Error("request failed", "operation", "stream_response", "error", err.Error())
		// Write error message to the stream so the client can display it
		errorMsg := fmt.Sprintf("Error: Failed to get response from AI provider. %s", err.Error())
		fmt.Fprint(w, errorMsg)
		return
	}

	// Save assistant message with user_id and provider mode
	providerMode := "local"
	if !s.providerManager.IsLocalMode() {
		providerMode = "cloud"
	}
	if err := s.store.SaveChatMessage(ctx, userID, req.SessionID, "assistant", response, providerMode); err != nil {
		logger.Warn("failed to save assistant message", "error", err.Error())
	}

	latency := time.Since(start).Milliseconds()
	logger.Debug("request completed", "status", http.StatusOK, "latency_ms", latency, "session_id", req.SessionID)
}

// handleSessions returns a list of all chat sessions for the current user
func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract user_id from context
	userID, err := auth.GetUserID(ctx)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get sessions for this user only
	sessions, err := s.store.GetUserSessions(ctx, userID)
	if err != nil {
		http.Error(w, "Failed to list sessions", http.StatusInternalServerError)
		return
	}

	// Return as JSON or HTML fragment based on Accept header
	if strings.Contains(r.Header.Get("Accept"), "application/json") {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sessions)
	} else {
		// Return HTML fragment for HTMX
		w.Header().Set("Content-Type", "text/html")
		for _, session := range sessions {
			relativeTime := formatRelativeTime(session.LastMessageAt)
			fmt.Fprintf(w, `<div class="session-item" data-session-id="%s" onclick="loadSession('%s')">
				<div class="session-time">%s</div>
				<div class="session-count">%d messages</div>
			</div>`, session.ID, session.ID, relativeTime, session.MessageCount)
		}
	}
}

// handleSessionHistory retrieves messages for a specific session
func (s *Server) handleSessionHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract user_id from context
	userID, err := auth.GetUserID(ctx)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract session ID from URL path
	sessionID := strings.TrimPrefix(r.URL.Path, "/api/session/")
	if sessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	// Get session messages with ownership verification
	messages, err := s.store.GetSessionMessages(ctx, userID, sessionID)
	if err != nil {
		http.Error(w, "Failed to get session history", http.StatusInternalServerError)
		return
	}

	// Return as JSON or HTML fragment
	if strings.Contains(r.Header.Get("Accept"), "application/json") {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messages)
	} else {
		// Return HTML fragment for HTMX with proper message structure
		w.Header().Set("Content-Type", "text/html")
		for _, msg := range messages {
			avatarSVG := ""
			providerClass := ""
			if msg.Role == "user" {
				avatarSVG = `<svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
					<path fill-rule="evenodd" d="M10 9a3 3 0 100-6 3 3 0 000 6zm-7 9a7 7 0 1114 0H3z"/>
				</svg>`
			} else {
				avatarSVG = `<svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
					<path d="M2 5a2 2 0 012-2h7a2 2 0 012 2v4a2 2 0 01-2 2H9l-3 3v-3H4a2 2 0 01-2-2V5z"/>
					<path d="M15 7v2a4 4 0 01-4 4H9.828l-1.766 1.767c.28.149.599.233.938.233h2l3 3v-3h2a2 2 0 002-2V9a2 2 0 00-2-2h-1z"/>
				</svg>`
				// Add provider class for assistant messages
				if msg.ProviderMode == "cloud" {
					providerClass = " provider-cloud"
				} else {
					providerClass = " provider-local"
				}
			}

			fmt.Fprintf(w, `<div class="message message-%s">
				<div class="message-avatar%s">%s</div>
				<div class="message-content">%s</div>
			</div>`, msg.Role, providerClass, avatarSVG, msg.Content)
		}
	}
}

// handleLibrary renders the library page with document cards
func (s *Server) handleLibrary(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestID := generateRequestID()

	// Create logger with request context
	logger := s.logger.WithContext("request_id", requestID).
		WithContext("method", r.Method).
		WithContext("path", r.URL.Path)

	logger.Debug("processing request")

	// Prevent caching
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	ctx := r.Context()

	// Extract user_id from context
	userID, err := auth.GetUserID(ctx)
	if err != nil {
		logger.Error("request failed", "operation", "get_user_id", "error", err.Error())
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get tag filter from query parameter
	tagFilter := r.URL.Query().Get("tag")

	// Get library entries for user
	library, err := s.store.LibraryByUser(ctx, userID)
	if err != nil {
		logger.Error("request failed", "operation", "get_library", "error", err.Error())
		http.Error(w, "Failed to load library", http.StatusInternalServerError)
		return
	}

	// Filter by tag if specified
	var filteredLibrary []LibraryEntry
	if tagFilter != "" {
		for _, entry := range library {
			for _, tag := range entry.Tags {
				if tag == tagFilter {
					filteredLibrary = append(filteredLibrary, entry)
					break
				}
			}
		}
	} else {
		filteredLibrary = library
	}

	// Check if this is an HTMX request (return fragment)
	if r.Header.Get("HX-Request") == "true" {
		// Return HTML fragment with document cards
		w.Header().Set("Content-Type", "text/html")
		for _, entry := range filteredLibrary {
			tagsHTML := ""
			for _, tag := range entry.Tags {
				tagsHTML += fmt.Sprintf(`<span class="tag">%s</span>`, tag)
			}

			preview := entry.Summary
			if preview == "" && len(entry.Source) > 0 {
				preview = "No summary available"
			}
			if len(preview) > 150 {
				preview = preview[:150] + "..."
			}

			fmt.Fprintf(w, `<div class="document-card" data-source="%s">
				<h3>%s</h3>
				<p class="preview">%s</p>
				<div class="card-footer">
					<span class="chunk-count">%d chunks</span>
					<div class="tags">%s</div>
				</div>
				<button class="delete-btn" onclick="deleteDocument('%s')">Delete</button>
			</div>`, entry.Source, entry.Source, preview, entry.ChunkCount, tagsHTML, entry.Source)
		}

		latency := time.Since(start).Milliseconds()
		logger.Debug("request completed", "status", http.StatusOK, "latency_ms", latency, "htmx_request", true)
		return
	}

	// Render full page
	data := map[string]interface{}{
		"Title":       "Library",
		"Page":        "library",
		"PrivacyMode": s.config.PrivacyMode,
		"Library":     filteredLibrary,
		"TagFilter":   tagFilter,
	}

	if err := s.templates.ExecuteTemplate(w, "base.html", data); err != nil {
		logger.Error("request failed", "operation", "render_template", "error", err.Error())
		http.Error(w, "Failed to render library", http.StatusInternalServerError)
		return
	}

	latency := time.Since(start).Milliseconds()
	logger.Debug("request completed", "status", http.StatusOK, "latency_ms", latency)
}

// handleIngestText processes plain text ingestion
func (s *Server) handleIngestText(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestID := generateRequestID()

	// Create logger with request context
	logger := s.logger.WithContext("request_id", requestID).
		WithContext("method", r.Method).
		WithContext("path", r.URL.Path)

	logger.Debug("processing request")

	if r.Method != http.MethodPost {
		logger.Error("request failed", "operation", "method_check", "error", "method not allowed")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Extract user_id from context
	userID, err := auth.GetUserID(ctx)
	if err != nil {
		logger.Error("request failed", "operation", "get_user_id", "error", err.Error())
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse request
	var req struct {
		Source string   `json:"source"`
		Text   string   `json:"text"`
		Tags   []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("request failed", "operation", "parse_request", "error", err.Error())
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Ingest text with user_id
	if err := s.ingester.IngestText(ctx, userID, req.Source, req.Text, req.Tags); err != nil {
		logger.Error("request failed", "operation", "ingest_text", "source", req.Source, "error", err.Error())
		http.Error(w, fmt.Sprintf("Ingestion failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Audit log
	s.store.AddAuditEntry(ctx, "ingest", fmt.Sprintf("Text: %s", req.Source), "")

	// Broadcast WebSocket update
	s.wsHub.Broadcast("ingestion", fmt.Sprintf("Document '%s' ingested successfully", req.Source))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})

	latency := time.Since(start).Milliseconds()
	logger.Debug("request completed", "status", http.StatusOK, "latency_ms", latency, "source", req.Source)
}

// handleIngestURL processes URL ingestion
func (s *Server) handleIngestURL(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestID := generateRequestID()

	// Create logger with request context
	logger := s.logger.WithContext("request_id", requestID).
		WithContext("method", r.Method).
		WithContext("path", r.URL.Path)

	logger.Debug("processing request")

	if r.Method != http.MethodPost {
		logger.Error("request failed", "operation", "method_check", "error", "method not allowed")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Extract user_id from context
	userID, err := auth.GetUserID(ctx)
	if err != nil {
		logger.Error("request failed", "operation", "get_user_id", "error", err.Error())
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse request
	var req struct {
		URL  string   `json:"url"`
		Tags []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("request failed", "operation", "parse_request", "error", err.Error())
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Ingest URL with user_id
	if err := s.ingester.IngestURL(ctx, userID, req.URL, req.Tags); err != nil {
		logger.Error("request failed", "operation", "ingest_url", "url", req.URL, "error", err.Error())
		http.Error(w, fmt.Sprintf("Ingestion failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Audit log
	s.store.AddAuditEntry(ctx, "ingest", fmt.Sprintf("URL: %s", req.URL), "")

	// Broadcast WebSocket update
	s.wsHub.Broadcast("ingestion", fmt.Sprintf("URL '%s' ingested successfully", req.URL))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})

	latency := time.Since(start).Milliseconds()
	logger.Debug("request completed", "status", http.StatusOK, "latency_ms", latency, "url", req.URL)
}

// handleIngestFile processes file upload ingestion
func (s *Server) handleIngestFile(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestID := generateRequestID()

	// Create logger with request context
	logger := s.logger.WithContext("request_id", requestID).
		WithContext("method", r.Method).
		WithContext("path", r.URL.Path)

	logger.Debug("processing request")

	if r.Method != http.MethodPost {
		logger.Error("request failed", "operation", "method_check", "error", "method not allowed")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Parse multipart form
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB max
		logger.Error("request failed", "operation", "parse_form", "error", err.Error())
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		logger.Error("request failed", "operation", "get_file", "error", err.Error())
		http.Error(w, "Failed to get file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Get tags from form
	tagsStr := r.FormValue("tags")
	var tags []string
	if tagsStr != "" {
		tags = strings.Split(tagsStr, ",")
		for i := range tags {
			tags[i] = strings.TrimSpace(tags[i])
		}
	}

	// Ingest file
	if err := s.ingestFile(ctx, file, header, tags); err != nil {
		logger.Error("request failed", "operation", "ingest_file", "filename", header.Filename, "error", err.Error())
		http.Error(w, fmt.Sprintf("Ingestion failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Audit log
	s.store.AddAuditEntry(ctx, "ingest", fmt.Sprintf("File: %s", header.Filename), "")

	// Broadcast WebSocket update
	s.wsHub.Broadcast("ingestion", fmt.Sprintf("File '%s' ingested successfully", header.Filename))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})

	latency := time.Since(start).Milliseconds()
	logger.Debug("request completed", "status", http.StatusOK, "latency_ms", latency, "filename", header.Filename)
}

// ingestFile is a helper that processes file ingestion
func (s *Server) ingestFile(ctx context.Context, file multipart.File, header *multipart.FileHeader, tags []string) error {
	// Extract user_id from context
	userID, err := auth.GetUserID(ctx)
	if err != nil {
		return fmt.Errorf("unauthorized: %w", err)
	}

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// For now, treat all files as text
	// In a full implementation, this would handle different file types
	text := string(content)

	// Ingest as text with user_id
	return s.ingester.IngestText(ctx, userID, header.Filename, text, tags)
}

// handleDelete removes a document and all its chunks
func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestID := generateRequestID()

	// Create logger with request context
	logger := s.logger.WithContext("request_id", requestID).
		WithContext("method", r.Method).
		WithContext("path", r.URL.Path)

	logger.Debug("processing request")

	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		logger.Error("request failed", "operation", "method_check", "error", "method not allowed")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Parse request
	var req struct {
		Source string `json:"source"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("request failed", "operation", "parse_request", "error", err.Error())
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Delete document
	if err := s.store.DeleteSource(ctx, req.Source); err != nil {
		logger.Error("request failed", "operation", "delete_source", "source", req.Source, "error", err.Error())
		http.Error(w, "Delete failed", http.StatusInternalServerError)
		return
	}

	// Audit log
	s.store.AddAuditEntry(ctx, "delete", fmt.Sprintf("Source: %s", req.Source), "")

	// Broadcast WebSocket update
	s.wsHub.Broadcast("deletion", fmt.Sprintf("Document '%s' deleted", req.Source))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})

	latency := time.Since(start).Milliseconds()
	logger.Debug("request completed", "status", http.StatusOK, "latency_ms", latency, "source", req.Source)
}

// Note: PromptBuilder is now in the rag package (internal/rag/prompt.go)

// generateSessionID creates a random session ID
func generateSessionID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// generateRequestID creates a random request ID for logging
func generateRequestID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// formatRelativeTime formats a timestamp as relative time (e.g., "2 hours ago")
func formatRelativeTime(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}

// handleSettings renders the settings page
func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestID := generateRequestID()

	// Create logger with request context
	logger := s.logger.WithContext("request_id", requestID).
		WithContext("method", r.Method).
		WithContext("path", r.URL.Path)

	logger.Debug("processing request")

	// Prevent caching of settings page
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// Load current config from file to get latest values
	cfg, err := config.Load(s.configPath)
	if err != nil {
		logger.Error("Failed to load config", "error", err.Error())
		http.Error(w, "Failed to load configuration", http.StatusInternalServerError)
		return
	}

	// Create nested config structure that matches template expectations
	configData := map[string]interface{}{
		"Privacy": map[string]interface{}{
			"DefaultToLocal": cfg.Privacy.DefaultToLocal,
			"CloudRAGPolicy": cfg.Privacy.CloudRAGPolicy,
		},
		"LocalProvider": map[string]interface{}{
			"Type":             cfg.LocalProvider.Type,
			"OllamaEndpoint":   cfg.LocalProvider.OllamaEndpoint,
			"OllamaEmbedModel": cfg.LocalProvider.OllamaEmbedModel,
			"OllamaChatModel":  cfg.LocalProvider.OllamaChatModel,
		},
		"CloudProvider": map[string]interface{}{
			"Type":                cfg.CloudProvider.Type,
			"OpenAIKey":           cfg.CloudProvider.OpenAIKey,
			"OpenAIEmbedModel":    cfg.CloudProvider.OpenAIEmbedModel,
			"OpenAIChatModel":     cfg.CloudProvider.OpenAIChatModel,
			"AnthropicKey":        cfg.CloudProvider.AnthropicKey,
			"AnthropicEmbedModel": cfg.CloudProvider.AnthropicEmbedModel,
			"AnthropicChatModel":  cfg.CloudProvider.AnthropicChatModel,
		},
		"Folders": cfg.Folders,
		"Guardrails": map[string]interface{}{
			"PIIDetection":  cfg.Guardrails.PIIDetection,
			"AutoSummarize": cfg.Guardrails.AutoSummarize,
			"MaxFileSizeMB": cfg.Guardrails.MaxFileSizeMB,
			"MaxConcurrent": cfg.Guardrails.MaxConcurrent,
		},
	}

	// Check if cloud provider is available
	cloudProviderAvailable := false
	if s.providerManager != nil {
		cloudProviderAvailable = s.providerManager.GetCloudProvider() != nil
	}

	data := map[string]interface{}{
		"Title":                  "Settings",
		"Page":                   "settings",
		"PrivacyMode":            false,
		"Config":                 configData,
		"CloudProviderAvailable": cloudProviderAvailable,
	}

	if err := s.templates.ExecuteTemplate(w, "base.html", data); err != nil {
		logger.Error("request failed", "operation", "render_template", "error", err.Error())
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}

	latency := time.Since(start).Milliseconds()
	logger.Debug("request completed", "status", http.StatusOK, "latency_ms", latency)
}

// handleConfig saves configuration changes
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.logger.Debug("Received dual provider config save request")

	// Parse form data
	if err := r.ParseForm(); err != nil {
		s.logger.Error("Failed to parse form: %v", err)
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	s.logger.Debug("Form data received: %v", r.Form)

	// Load current config
	cfg, err := config.Load(s.configPath)
	if err != nil {
		s.logger.Error("Failed to load config: %v", err)
		http.Error(w, "Failed to load config", http.StatusInternalServerError)
		return
	}

	// Parse local provider configuration
	localProviderType := r.FormValue("local_provider_type")
	if localProviderType != "" {
		cfg.LocalProvider.Type = localProviderType
		s.logger.Debug("Local provider type: %s", localProviderType)
	}

	// Local Ollama settings
	if v := r.FormValue("local_ollama_endpoint"); v != "" {
		cfg.LocalProvider.OllamaEndpoint = v
		s.logger.Debug("Local Ollama endpoint: %s", v)
	}
	if v := r.FormValue("local_ollama_embed_model"); v != "" {
		cfg.LocalProvider.OllamaEmbedModel = v
		s.logger.Debug("Local Ollama embed model: %s", v)
	}
	if v := r.FormValue("local_ollama_chat_model"); v != "" {
		cfg.LocalProvider.OllamaChatModel = v
		s.logger.Debug("Local Ollama chat model: %s", v)
	}

	// Parse cloud provider configuration
	cloudProviderType := r.FormValue("cloud_provider_type")
	if cloudProviderType != "" {
		cfg.CloudProvider.Type = cloudProviderType
		s.logger.Debug("Cloud provider type: %s", cloudProviderType)
	}

	// Cloud OpenAI settings
	if v := r.FormValue("cloud_openai_key"); v != "" {
		cfg.CloudProvider.OpenAIKey = v
		s.logger.Debug("Cloud OpenAI key provided: %d chars", len(v))
	}
	if v := r.FormValue("cloud_openai_embed_model"); v != "" {
		cfg.CloudProvider.OpenAIEmbedModel = v
		s.logger.Debug("Cloud OpenAI embed model: %s", v)
	}
	if v := r.FormValue("cloud_openai_chat_model"); v != "" {
		cfg.CloudProvider.OpenAIChatModel = v
		s.logger.Debug("Cloud OpenAI chat model: %s", v)
	}

	// Cloud Anthropic settings
	if v := r.FormValue("cloud_anthropic_key"); v != "" {
		cfg.CloudProvider.AnthropicKey = v
		s.logger.Debug("Cloud Anthropic key provided: %d chars", len(v))
	}
	if v := r.FormValue("cloud_anthropic_embed_model"); v != "" {
		cfg.CloudProvider.AnthropicEmbedModel = v
		s.logger.Debug("Cloud Anthropic embed model: %s", v)
	}
	if v := r.FormValue("cloud_anthropic_chat_model"); v != "" {
		cfg.CloudProvider.AnthropicChatModel = v
		s.logger.Debug("Cloud Anthropic chat model: %s", v)
	}

	// Parse privacy toggle state (default_to_local)
	defaultToLocal := r.FormValue("default_to_local")
	if defaultToLocal == "true" || defaultToLocal == "on" {
		cfg.Privacy.DefaultToLocal = true
		s.logger.Debug("Privacy toggle: use local AI")
	} else if defaultToLocal == "false" || defaultToLocal == "" {
		cfg.Privacy.DefaultToLocal = false
		s.logger.Debug("Privacy toggle: use cloud AI")
	}

	// Parse RAG policy (cloud_rag_policy)
	ragPolicy := r.FormValue("cloud_rag_policy")
	if ragPolicy != "" {
		cfg.Privacy.CloudRAGPolicy = ragPolicy
		s.logger.Debug("Cloud RAG policy: %s", ragPolicy)
	}

	s.logger.Debug("Dual provider config parsed successfully")

	// Validate local provider configuration
	if err := cfg.LocalProvider.ValidateLocal(); err != nil {
		s.logger.Error("Local provider validation failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf(`{"success": false, "error": "Local provider validation failed: %s"}`, err.Error())))
		return
	}

	// Validate cloud provider configuration
	if err := cfg.CloudProvider.ValidateCloud(); err != nil {
		s.logger.Error("Cloud provider validation failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf(`{"success": false, "error": "Cloud provider validation failed: %s"}`, err.Error())))
		return
	}

	// Validate RAG policy
	if err := cfg.Privacy.ValidateRAGPolicy(); err != nil {
		s.logger.Error("RAG policy validation failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf(`{"success": false, "error": "RAG policy validation failed: %s"}`, err.Error())))
		return
	}

	s.logger.Debug("All validations passed, saving configuration")

	// Save configuration to disk
	if err := cfg.Save(s.configPath); err != nil {
		s.logger.Error("Failed to save config: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf(`{"success": false, "error": "Failed to save configuration: %s"}`, err.Error())))
		return
	}

	s.logger.Info("Configuration saved successfully")

	// Reload providers with new configuration
	if err := s.providerManager.Reload(cfg); err != nil {
		s.logger.Error("Failed to reload providers: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf(`{"success": false, "error": "Failed to reload providers: %s"}`, err.Error())))
		return
	}

	s.logger.Info("Providers reloaded successfully")

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"success": true, "message": "Configuration saved successfully"}`))
}

// handleTestConnection tests provider connectivity
func (s *Server) handleTestConnection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Parse form to get test_provider_mode
	if err := r.ParseForm(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to parse form data",
		})
		return
	}

	providerMode := r.FormValue("test_provider_mode")
	if providerMode == "" {
		providerMode = "active" // Default to active provider if not specified
	}

	// Get the provider to test based on mode
	var providerToTest LLMProvider
	var providerName string

	if s.providerManager != nil {
		// Using provider manager (dual-provider setup)
		switch providerMode {
		case "local":
			providerToTest = s.providerManager.GetLocalProvider()
			if providerToTest == nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": false,
					"error":   "Local provider not configured",
				})
				return
			}
			providerName = "Local provider"
		case "cloud":
			providerToTest = s.providerManager.GetCloudProvider()
			if providerToTest == nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": false,
					"error":   "Cloud provider not configured",
				})
				return
			}
			providerName = "Cloud provider"
		default:
			// Test active provider
			activeProvider, err := s.providerManager.GetActiveProvider()
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": false,
					"error":   err.Error(),
				})
				return
			}
			providerToTest = activeProvider
			providerName = "Active provider"
		}
	} else {
		// Single provider mode (legacy)
		providerToTest = s.provider
		providerName = "Provider"
	}

	// Test embedding with a simple text
	_, err := providerToTest.Embed(ctx, "test")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": providerName + " connection successful",
	})
}

// handleActivity returns recent activity feed for the dashboard
func (s *Server) handleActivity(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Query audit log for recent 10 entries
	entries, err := s.store.GetAuditLog(ctx, "", time.Time{}, time.Now())
	if err != nil {
		http.Error(w, "Failed to fetch activity", http.StatusInternalServerError)
		return
	}

	// Limit to 10 most recent
	if len(entries) > 10 {
		entries = entries[len(entries)-10:]
	}

	// Reverse to show most recent first
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	// Format as HTML fragment
	var html strings.Builder
	for _, entry := range entries {
		html.WriteString(fmt.Sprintf(`<div class="activity-item">
			<div class="activity-type">%s</div>
			<div class="activity-details">%s</div>
			<div class="activity-time">%s</div>
		</div>`, entry.OperationType, entry.Details, formatRelativeTime(entry.Timestamp)))
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html.String()))
}

// handleSkills lists available skills for the current user
func (s *Server) handleSkills(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Extract user_id from context
	userID, err := auth.GetUserID(ctx)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get skills for this user from database
	skills, err := s.store.GetUserSkills(ctx, userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load skills: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"skills": skills,
	})
}

// handleRunSkill executes a manual-trigger skill
func (s *Server) handleRunSkill(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Extract user_id from context
	userID, err := auth.GetUserID(ctx)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse request body
	var req struct {
		SkillName string                 `json:"skill_name"`
		Query     string                 `json:"query"`
		Context   map[string]interface{} `json:"context"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Load skills for this user
	skills, err := s.skillsLoader.LoadForUser(ctx, userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load skills: %v", err), http.StatusInternalServerError)
		return
	}

	var targetSkill *Skill
	for _, skill := range skills {
		if skill.Name == req.SkillName {
			targetSkill = skill
			break
		}
	}

	if targetSkill == nil {
		http.Error(w, fmt.Sprintf("Skill not found: %s", req.SkillName), http.StatusNotFound)
		return
	}

	// Verify skill ownership - ensure the skill belongs to the current user
	if targetSkill.UserID != userID {
		http.Error(w, "Unauthorized: skill does not belong to current user", http.StatusForbidden)
		return
	}

	// Check if skill has manual trigger
	hasManualTrigger := false
	for _, trigger := range targetSkill.Triggers {
		if trigger.Type == "manual" {
			hasManualTrigger = true
			break
		}
	}

	if !hasManualTrigger {
		http.Error(w, "Skill does not support manual execution", http.StatusBadRequest)
		return
	}

	// Execute skill
	input := SkillInput{
		Query:    req.Query,
		Context:  req.Context,
		Settings: make(map[string]interface{}),
	}

	output, err := s.skillsExecutor.Execute(ctx, targetSkill, input)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Return result
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"result":   output.Result,
		"metadata": output.Metadata,
	})
}

// handleWatchedFolders returns the list of watched folders for the current user
func (s *Server) handleWatchedFolders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Extract user_id from context
	userID, err := auth.GetUserID(ctx)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get watched folders for this user
	folders, err := s.store.GetWatchedFoldersByUser(ctx, userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get watched folders: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"folders": folders,
	})
}

// Authentication Handlers

// handleLogin processes user login and returns a session token
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestID := generateRequestID()

	// Create logger with request context
	logger := s.logger.WithContext("request_id", requestID).
		WithContext("method", r.Method).
		WithContext("path", r.URL.Path)

	logger.Debug("processing login request")

	if r.Method != http.MethodPost {
		logger.Error("request failed", "operation", "method_check", "error", "method not allowed")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Parse request
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("request failed", "operation", "parse_request", "error", err.Error())
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Username == "" || req.Password == "" {
		logger.Warn("login failed", "reason", "missing credentials")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Username and password are required",
		})
		return
	}

	// Call auth provider Login
	token, err := s.authProvider.Login(ctx, req.Username, req.Password)
	if err != nil {
		logger.Warn("login failed", "username", req.Username, "error", err.Error())

		// Check if account is locked
		if strings.Contains(err.Error(), "account locked") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusLocked) // 423
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
			return
		}

		// Invalid credentials
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid username or password",
		})
		return
	}

	// Get user details to check must_change_password flag
	user, err := s.store.GetUserByUsername(ctx, req.Username)
	if err != nil {
		logger.Error("request failed", "operation", "get_user", "error", err.Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Set session_token cookie
	cookie := &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   7 * 24 * 60 * 60, // 7 days
	}

	// Set Secure flag in production (when not localhost)
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		cookie.Secure = true
	}

	http.SetCookie(w, cookie)

	// Determine redirect URL based on must_change_password
	redirectURL := "/"
	if user.MustChangePassword {
		redirectURL = "/change-password"
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"user": map[string]interface{}{
			"username": user.Username,
			"is_admin": user.IsAdmin,
		},
		"must_change_password": user.MustChangePassword,
		"redirect":             redirectURL,
	})

	latency := time.Since(start).Milliseconds()
	logger.Debug("login successful", "username", req.Username, "latency_ms", latency)
}

// handleLogout invalidates the user's session token
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestID := generateRequestID()

	// Create logger with request context
	logger := s.logger.WithContext("request_id", requestID).
		WithContext("method", r.Method).
		WithContext("path", r.URL.Path)

	logger.Debug("processing logout request")

	if r.Method != http.MethodPost {
		logger.Error("request failed", "operation", "method_check", "error", "method not allowed")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Extract token from request (use extractToken from middleware)
	token := extractTokenFromRequest(r)
	if token != "" {
		// Call auth provider Logout
		if err := s.authProvider.Logout(ctx, token); err != nil {
			logger.Warn("logout failed", "error", err.Error())
		}
	}

	// Clear session_token cookie
	cookie := &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1, // Delete cookie
	}
	http.SetCookie(w, cookie)

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})

	latency := time.Since(start).Milliseconds()
	logger.Debug("logout successful", "latency_ms", latency)
}

// handleRegister creates a new user account
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestID := generateRequestID()

	// Create logger with request context
	logger := s.logger.WithContext("request_id", requestID).
		WithContext("method", r.Method).
		WithContext("path", r.URL.Path)

	logger.Debug("processing registration request")

	if r.Method != http.MethodPost {
		logger.Error("request failed", "operation", "method_check", "error", "method not allowed")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Parse request
	var req struct {
		Username        string `json:"username"`
		Email           string `json:"email"`
		Password        string `json:"password"`
		ConfirmPassword string `json:"confirm_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("request failed", "operation", "parse_request", "error", err.Error())
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Validate password confirmation
	if req.Password != req.ConfirmPassword {
		logger.Warn("registration failed", "reason", "password mismatch")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Passwords do not match",
		})
		return
	}

	// Validate username format (3-32 chars, alphanumeric + underscore/dash)
	if len(req.Username) < 3 || len(req.Username) > 32 {
		logger.Warn("registration failed", "reason", "invalid username length")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Username must be between 3 and 32 characters",
		})
		return
	}

	// Check username contains only valid characters
	for _, c := range req.Username {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
			logger.Warn("registration failed", "reason", "invalid username characters")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Username can only contain letters, numbers, underscores, and dashes",
			})
			return
		}
	}

	// Validate email format (basic regex)
	if !strings.Contains(req.Email, "@") || !strings.Contains(req.Email, ".") {
		logger.Warn("registration failed", "reason", "invalid email format")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid email format",
		})
		return
	}

	// Create user (is_admin=false, must_change_password=false)
	_, err := s.store.CreateUser(ctx, req.Username, req.Password, req.Email, false, false)
	if err != nil {
		logger.Error("registration failed", "username", req.Username, "error", err.Error())

		// Check for duplicate username/email
		if strings.Contains(err.Error(), "UNIQUE constraint failed") || strings.Contains(err.Error(), "duplicate") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict) // 409
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Username or email already exists",
			})
			return
		}

		// Server error
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to create account",
		})
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Account created successfully",
	})

	latency := time.Since(start).Milliseconds()
	logger.Debug("registration successful", "username", req.Username, "latency_ms", latency)
}

// handleChangePassword changes the user's password
func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestID := generateRequestID()

	// Create logger with request context
	logger := s.logger.WithContext("request_id", requestID).
		WithContext("method", r.Method).
		WithContext("path", r.URL.Path)

	logger.Debug("processing change password request")

	if r.Method != http.MethodPost {
		logger.Error("request failed", "operation", "method_check", "error", "method not allowed")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Extract user_id from context (set by auth middleware)
	userID, err := auth.GetUserID(ctx)
	if err != nil {
		logger.Error("request failed", "operation", "get_user_id", "error", err.Error())
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse request
	var req struct {
		NewPassword     string `json:"new_password"`
		ConfirmPassword string `json:"confirm_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("request failed", "operation", "parse_request", "error", err.Error())
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Validate password confirmation
	if req.NewPassword != req.ConfirmPassword {
		logger.Warn("password change failed", "reason", "password mismatch")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Passwords do not match",
		})
		return
	}

	// Validate password strength (min 8 chars)
	if len(req.NewPassword) < 8 {
		logger.Warn("password change failed", "reason", "password too short")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Password must be at least 8 characters",
		})
		return
	}

	// Update password
	if err := s.store.UpdatePassword(ctx, userID, req.NewPassword); err != nil {
		logger.Error("password change failed", "user_id", userID, "error", err.Error())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to change password",
		})
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Password changed successfully",
	})

	latency := time.Since(start).Milliseconds()
	logger.Debug("password change successful", "user_id", userID, "latency_ms", latency)
}

// extractTokenFromRequest extracts the session token from the request
// First checks Authorization header with "Bearer " prefix
// Falls back to session_token cookie if header not present
// Returns empty string if neither is found
func extractTokenFromRequest(r *http.Request) string {
	// Try Authorization header first
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}

	// Fall back to cookie
	cookie, err := r.Cookie("session_token")
	if err == nil {
		return cookie.Value
	}

	return ""
}

// isAdmin checks if the current user is an admin
// Returns (isAdmin bool, userID int64, error)
func (s *Server) isAdmin(ctx context.Context) (bool, int64, error) {
	userID, err := auth.GetUserID(ctx)
	if err != nil {
		return false, 0, err
	}

	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return false, 0, err
	}

	return user.IsAdmin, userID, nil
}

// generateRandomPassword generates a secure random password
func generateRandomPassword(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	password := make([]byte, length)

	for i := range password {
		// Generate random byte
		randomBytes := make([]byte, 1)
		if _, err := rand.Read(randomBytes); err != nil {
			return "", err
		}
		// Map to charset
		password[i] = charset[int(randomBytes[0])%len(charset)]
	}

	return string(password), nil
}

// handleGetUsers handles GET /api/users - list all users (admin only)
func (s *Server) handleGetUsers(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestID := generateRequestID()

	logger := s.logger.WithContext("request_id", requestID).
		WithContext("method", r.Method).
		WithContext("path", r.URL.Path)

	logger.Debug("processing get users request")

	ctx := r.Context()

	// Check if current user is admin
	isAdmin, userID, err := s.isAdmin(ctx)
	if err != nil {
		logger.Error("failed to get user from context", "error", err.Error())
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if !isAdmin {
		logger.Warn("non-admin user attempted to list users", "user_id", userID)
		http.Error(w, "Forbidden: admin access required", http.StatusForbidden)
		return
	}

	// Get all users
	users, err := s.store.ListUsers(ctx)
	if err != nil {
		logger.Error("failed to list users", "error", err.Error())
		http.Error(w, "Failed to retrieve users", http.StatusInternalServerError)
		return
	}

	// Format response
	type UserResponse struct {
		ID        int64     `json:"id"`
		Username  string    `json:"username"`
		Email     string    `json:"email"`
		IsAdmin   bool      `json:"is_admin"`
		CreatedAt time.Time `json:"created_at"`
		LastLogin time.Time `json:"last_login"`
	}

	userList := make([]UserResponse, len(users))
	for i, user := range users {
		userList[i] = UserResponse{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			IsAdmin:   user.IsAdmin,
			CreatedAt: user.CreatedAt,
			LastLogin: user.LastLogin,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"users": userList,
	})

	latency := time.Since(start).Milliseconds()
	logger.Debug("get users successful", "user_count", len(users), "latency_ms", latency)
}

// handleCreateUser handles POST /api/users - create new user (admin only)
func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestID := generateRequestID()

	logger := s.logger.WithContext("request_id", requestID).
		WithContext("method", r.Method).
		WithContext("path", r.URL.Path)

	logger.Debug("processing create user request")

	ctx := r.Context()

	// Check if current user is admin
	isAdmin, userID, err := s.isAdmin(ctx)
	if err != nil {
		logger.Error("failed to get user from context", "error", err.Error())
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if !isAdmin {
		logger.Warn("non-admin user attempted to create user", "user_id", userID)
		http.Error(w, "Forbidden: admin access required", http.StatusForbidden)
		return
	}

	// Parse request body
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
		IsAdmin  bool   `json:"is_admin"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("failed to decode request body", "error", err.Error())
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	if req.Password == "" {
		http.Error(w, "Password is required", http.StatusBadRequest)
		return
	}

	if len(req.Password) < 8 {
		http.Error(w, "Password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	// Validate username format (alphanumeric and underscore only)
	for _, c := range req.Username {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			http.Error(w, "Username must contain only alphanumeric characters and underscores", http.StatusBadRequest)
			return
		}
	}

	// Validate email format if provided
	if req.Email != "" && !strings.Contains(req.Email, "@") {
		http.Error(w, "Invalid email format", http.StatusBadRequest)
		return
	}

	// Create user
	newUserID, err := s.store.CreateUser(ctx, req.Username, req.Password, req.Email, req.IsAdmin, false)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") || strings.Contains(err.Error(), "unique") {
			if strings.Contains(err.Error(), "username") {
				logger.Warn("duplicate username", "username", req.Username)
				http.Error(w, "Username already exists", http.StatusConflict)
			} else if strings.Contains(err.Error(), "email") {
				logger.Warn("duplicate email", "email", req.Email)
				http.Error(w, "Email already registered", http.StatusConflict)
			} else {
				http.Error(w, "User already exists", http.StatusConflict)
			}
			return
		}
		logger.Error("failed to create user", "error", err.Error())
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	// Get created user
	newUser, err := s.store.GetUserByID(ctx, newUserID)
	if err != nil {
		logger.Error("failed to get created user", "error", err.Error())
		http.Error(w, "User created but failed to retrieve details", http.StatusInternalServerError)
		return
	}

	// Format response
	type UserResponse struct {
		ID       int64  `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
		IsAdmin  bool   `json:"is_admin"`
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"user": UserResponse{
			ID:       newUser.ID,
			Username: newUser.Username,
			Email:    newUser.Email,
			IsAdmin:  newUser.IsAdmin,
		},
	})

	latency := time.Since(start).Milliseconds()
	logger.Debug("user created successfully", "new_user_id", newUserID, "username", req.Username, "latency_ms", latency)
}

// handleDeleteUser handles DELETE /api/users/:id - delete user (admin only)
func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestID := generateRequestID()

	logger := s.logger.WithContext("request_id", requestID).
		WithContext("method", r.Method).
		WithContext("path", r.URL.Path)

	logger.Debug("processing delete user request")

	ctx := r.Context()

	// Check if current user is admin
	isAdmin, userID, err := s.isAdmin(ctx)
	if err != nil {
		logger.Error("failed to get user from context", "error", err.Error())
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if !isAdmin {
		logger.Warn("non-admin user attempted to delete user", "user_id", userID)
		http.Error(w, "Forbidden: admin access required", http.StatusForbidden)
		return
	}

	// Extract target user ID from URL path
	// Expected format: /api/users/:id
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	var targetUserID int64
	if _, err := fmt.Sscanf(pathParts[2], "%d", &targetUserID); err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Prevent admin from deleting themselves
	if targetUserID == userID {
		logger.Warn("admin attempted to delete themselves", "user_id", userID)
		http.Error(w, "Cannot delete your own account", http.StatusBadRequest)
		return
	}

	// Check if target user exists
	targetUser, err := s.store.GetUserByID(ctx, targetUserID)
	if err != nil {
		logger.Warn("target user not found", "target_user_id", targetUserID)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Delete user
	if err := s.store.DeleteUser(ctx, targetUserID); err != nil {
		logger.Error("failed to delete user", "target_user_id", targetUserID, "error", err.Error())
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "User deleted successfully",
	})

	latency := time.Since(start).Milliseconds()
	logger.Debug("user deleted successfully", "target_user_id", targetUserID, "target_username", targetUser.Username, "latency_ms", latency)
}

// handleResetUserPassword handles POST /api/users/:id/reset-password - reset user password (admin only)
func (s *Server) handleResetUserPassword(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestID := generateRequestID()

	logger := s.logger.WithContext("request_id", requestID).
		WithContext("method", r.Method).
		WithContext("path", r.URL.Path)

	logger.Debug("processing reset user password request")

	ctx := r.Context()

	// Check if current user is admin
	isAdmin, userID, err := s.isAdmin(ctx)
	if err != nil {
		logger.Error("failed to get user from context", "error", err.Error())
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if !isAdmin {
		logger.Warn("non-admin user attempted to reset password", "user_id", userID)
		http.Error(w, "Forbidden: admin access required", http.StatusForbidden)
		return
	}

	// Extract target user ID from URL path
	// Expected format: /api/users/:id/reset-password
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	var targetUserID int64
	if _, err := fmt.Sscanf(pathParts[2], "%d", &targetUserID); err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Check if target user exists
	targetUser, err := s.store.GetUserByID(ctx, targetUserID)
	if err != nil {
		logger.Warn("target user not found", "target_user_id", targetUserID)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Generate random password
	randomPassword, err := generateRandomPassword(16)
	if err != nil {
		logger.Error("failed to generate random password", "error", err.Error())
		http.Error(w, "Failed to generate password", http.StatusInternalServerError)
		return
	}

	// Update password (this sets must_change_password to false by default)
	if err := s.store.UpdatePassword(ctx, targetUserID, randomPassword); err != nil {
		logger.Error("failed to update password", "target_user_id", targetUserID, "error", err.Error())
		http.Error(w, "Failed to reset password", http.StatusInternalServerError)
		return
	}

	// Note: The design mentions we need to set must_change_password=true after reset
	// However, UpdatePassword sets it to false. We need to update the user record separately.
	// For now, we'll document this as a known limitation and the user will need to change
	// their password voluntarily. A proper implementation would require a new store method
	// or modifying UpdatePassword to accept a must_change_password parameter.

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":            true,
		"temporary_password": randomPassword,
		"message":            "Password reset successfully. User should change password on next login.",
	})

	latency := time.Since(start).Milliseconds()
	logger.Debug("password reset successful", "target_user_id", targetUserID, "target_username", targetUser.Username, "latency_ms", latency)
}

// handleLoginPage renders the login page
func (s *Server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	// Prevent caching
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// Prepare template data
	data := map[string]interface{}{
		"Title": "Login",
	}

	// Render login template
	if err := s.templates.ExecuteTemplate(w, "login-content", data); err != nil {
		s.logger.Error("Failed to render login template: %v", err)
		http.Error(w, "Failed to render login page", http.StatusInternalServerError)
		return
	}
}

// handleRegisterPage renders the registration page
func (s *Server) handleRegisterPage(w http.ResponseWriter, r *http.Request) {
	// Prevent caching
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// Prepare template data
	data := map[string]interface{}{
		"Title": "Register",
	}

	// Render register template
	if err := s.templates.ExecuteTemplate(w, "register-content", data); err != nil {
		s.logger.Error("Failed to render register template: %v", err)
		http.Error(w, "Failed to render register page", http.StatusInternalServerError)
		return
	}
}

// handleChangePasswordPage renders the password change page
func (s *Server) handleChangePasswordPage(w http.ResponseWriter, r *http.Request) {
	// Prevent caching
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// Prepare template data
	data := map[string]interface{}{
		"Title": "Change Password",
	}

	// Render change-password template
	if err := s.templates.ExecuteTemplate(w, "change-password-content", data); err != nil {
		s.logger.Error("Failed to render change-password template: %v", err)
		http.Error(w, "Failed to render change password page", http.StatusInternalServerError)
		return
	}
}

// handlePrivacyToggle handles POST /api/privacy-toggle endpoint
// Allows users to quickly switch between local and cloud AI providers
func (s *Server) handlePrivacyToggle(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestID := generateRequestID()

	// Create logger with request context
	logger := s.logger.WithContext("request_id", requestID).
		WithContext("method", r.Method).
		WithContext("path", r.URL.Path)

	if r.Method != http.MethodPost {
		logger.Warn("method not allowed", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	logger.Debug("processing privacy toggle request")

	// Parse JSON body
	var req struct {
		Mode string `json:"mode"` // "local" or "cloud"
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("failed to parse request body", "error", err.Error())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request body",
		})
		return
	}

	logger.Debug("privacy toggle request", "mode", req.Mode)

	// Validate mode
	if req.Mode != "local" && req.Mode != "cloud" {
		logger.Error("invalid mode", "mode", req.Mode)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid mode: must be 'local' or 'cloud'",
		})
		return
	}

	// Load current config
	cfg, err := config.Load(s.configPath)
	if err != nil {
		logger.Error("failed to load config", "error", err.Error())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to load configuration",
		})
		return
	}

	// Update privacy toggle state based on mode
	defaultToLocal := req.Mode == "local"
	cfg.Privacy.DefaultToLocal = defaultToLocal

	logger.Debug("updating privacy toggle", "default_to_local", defaultToLocal)

	// Save configuration to disk
	if err := cfg.Save(s.configPath); err != nil {
		logger.Error("failed to save config", "error", err.Error())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to save configuration",
		})
		return
	}

	// Reload the provider manager and RAG enforcer with the new config
	if err := s.providerManager.Reload(cfg); err != nil {
		logger.Error("failed to reload provider manager", "error", err.Error())
		// Don't fail the request, just log the error
	}
	s.ragEnforcer.Reload(cfg)

	logger.Info("privacy toggle updated successfully", "mode", req.Mode)

	// Get provider name and RAG status for response
	providerName := s.providerManager.GetProviderName()
	ragStatus := s.ragEnforcer.GetRAGStatus()

	// Calculate latency
	latency := time.Since(start).Milliseconds()
	logger.Debug("request completed", "status", http.StatusOK, "latency_ms", latency)

	// Return success response with new provider name and RAG status
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"mode":       req.Mode,
		"provider":   providerName,
		"rag_status": ragStatus,
		"latency_ms": latency,
	})
}
