package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

// handleDashboard renders the dashboard page with system stats
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get document count
	library, err := s.store.Library(ctx)
	if err != nil {
		log.Printf("Failed to get library: %v", err)
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
	privacyMode := s.config.PrivacyMode

	// Prepare template data
	data := map[string]interface{}{
		"DocumentCount": docCount,
		"Provider":      providerName,
		"PrivacyMode":   privacyMode,
		"LastIngestion": lastIngestion,
		"HasIngestions": !lastIngestion.IsZero(),
	}

	// Render template
	if err := s.templates.ExecuteTemplate(w, "dashboard.html", data); err != nil {
		log.Printf("Failed to render dashboard template: %v", err)
		http.Error(w, "Failed to render dashboard", http.StatusInternalServerError)
	}
}

// handleChat renders the chat page
func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	// Render chat template
	if err := s.templates.ExecuteTemplate(w, "chat.html", nil); err != nil {
		log.Printf("Failed to render chat template: %v", err)
		http.Error(w, "Failed to render chat", http.StatusInternalServerError)
	}
}

// handleAsk processes chat queries with RAG
func (s *Server) handleAsk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Parse request
	var req struct {
		Query     string `json:"query"`
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Generate session ID if not provided
	if req.SessionID == "" {
		req.SessionID = generateSessionID()
	}

	// Save user message
	if err := s.store.SaveMessage(ctx, req.SessionID, "user", req.Query); err != nil {
		log.Printf("Failed to save user message: %v", err)
	}

	// Audit log
	s.store.AddAuditEntry(ctx, "query", req.Query, req.SessionID)

	// Embed query
	queryVec, err := s.provider.Embed(ctx, req.Query)
	if err != nil {
		log.Printf("Embedding failed: %v", err)
		http.Error(w, "Embedding failed", http.StatusInternalServerError)
		return
	}

	// Search for relevant chunks
	chunks, err := s.searcher.Search(ctx, queryVec, 5)
	if err != nil {
		log.Printf("Search failed: %v", err)
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}

	// Build prompt using PromptBuilder
	promptBuilder := &PromptBuilder{}
	prompt := promptBuilder.BuildPrompt(req.Query, chunks)

	// Stream response
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Session-ID", req.SessionID)

	messages := []Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: prompt},
	}

	response, err := s.provider.Stream(ctx, messages, w)
	if err != nil {
		log.Printf("Stream failed: %v", err)
		return
	}

	// Save assistant message
	if err := s.store.SaveMessage(ctx, req.SessionID, "assistant", response); err != nil {
		log.Printf("Failed to save assistant message: %v", err)
	}
}

// handleSessions returns a list of all chat sessions
func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sessions, err := s.store.ListSessions(ctx)
	if err != nil {
		log.Printf("Failed to list sessions: %v", err)
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
			fmt.Fprintf(w, `<div class="session-item" data-session-id="%s">
				<div class="session-time">%s</div>
				<div class="session-count">%d messages</div>
			</div>`, session.ID, relativeTime, session.MessageCount)
		}
	}
}

// handleSessionHistory retrieves messages for a specific session
func (s *Server) handleSessionHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract session ID from URL path
	sessionID := strings.TrimPrefix(r.URL.Path, "/api/session/")
	if sessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	messages, err := s.store.GetSessionHistory(ctx, sessionID)
	if err != nil {
		log.Printf("Failed to get session history: %v", err)
		http.Error(w, "Failed to get session history", http.StatusInternalServerError)
		return
	}

	// Return as JSON or HTML fragment
	if strings.Contains(r.Header.Get("Accept"), "application/json") {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messages)
	} else {
		// Return HTML fragment for HTMX
		w.Header().Set("Content-Type", "text/html")
		for _, msg := range messages {
			fmt.Fprintf(w, `<div class="message %s-message">
				<div class="message-content">%s</div>
			</div>`, msg.Role, msg.Content)
		}
	}
}

// handleLibrary renders the library page with document cards
func (s *Server) handleLibrary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get tag filter from query parameter
	tagFilter := r.URL.Query().Get("tag")

	// Get library entries
	library, err := s.store.Library(ctx)
	if err != nil {
		log.Printf("Failed to get library: %v", err)
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
		return
	}

	// Render full page
	data := map[string]interface{}{
		"Library":   filteredLibrary,
		"TagFilter": tagFilter,
	}

	if err := s.templates.ExecuteTemplate(w, "library.html", data); err != nil {
		log.Printf("Failed to render library template: %v", err)
		http.Error(w, "Failed to render library", http.StatusInternalServerError)
	}
}

// handleIngestText processes plain text ingestion
func (s *Server) handleIngestText(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Parse request
	var req struct {
		Source string   `json:"source"`
		Text   string   `json:"text"`
		Tags   []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Ingest text
	if err := s.ingester.IngestText(ctx, req.Source, req.Text, req.Tags); err != nil {
		log.Printf("Text ingestion failed: %v", err)
		http.Error(w, fmt.Sprintf("Ingestion failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Audit log
	s.store.AddAuditEntry(ctx, "ingest", fmt.Sprintf("Text: %s", req.Source), "")

	// Broadcast WebSocket update
	s.wsHub.Broadcast("ingestion", fmt.Sprintf("Document '%s' ingested successfully", req.Source))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// handleIngestURL processes URL ingestion
func (s *Server) handleIngestURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Parse request
	var req struct {
		URL  string   `json:"url"`
		Tags []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Ingest URL
	if err := s.ingester.IngestURL(ctx, req.URL, req.Tags); err != nil {
		log.Printf("URL ingestion failed: %v", err)
		http.Error(w, fmt.Sprintf("Ingestion failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Audit log
	s.store.AddAuditEntry(ctx, "ingest", fmt.Sprintf("URL: %s", req.URL), "")

	// Broadcast WebSocket update
	s.wsHub.Broadcast("ingestion", fmt.Sprintf("URL '%s' ingested successfully", req.URL))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// handleIngestFile processes file upload ingestion
func (s *Server) handleIngestFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Parse multipart form
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB max
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
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
		log.Printf("File ingestion failed: %v", err)
		http.Error(w, fmt.Sprintf("Ingestion failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Audit log
	s.store.AddAuditEntry(ctx, "ingest", fmt.Sprintf("File: %s", header.Filename), "")

	// Broadcast WebSocket update
	s.wsHub.Broadcast("ingestion", fmt.Sprintf("File '%s' ingested successfully", header.Filename))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// ingestFile is a helper that processes file ingestion
func (s *Server) ingestFile(ctx context.Context, file multipart.File, header *multipart.FileHeader, tags []string) error {
	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// For now, treat all files as text
	// In a full implementation, this would handle different file types
	text := string(content)

	// Ingest as text
	return s.ingester.IngestText(ctx, header.Filename, text, tags)
}

// handleDelete removes a document and all its chunks
func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Parse request
	var req struct {
		Source string `json:"source"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Delete document
	if err := s.store.DeleteSource(ctx, req.Source); err != nil {
		log.Printf("Delete failed: %v", err)
		http.Error(w, "Delete failed", http.StatusInternalServerError)
		return
	}

	// Audit log
	s.store.AddAuditEntry(ctx, "delete", fmt.Sprintf("Source: %s", req.Source), "")

	// Broadcast WebSocket update
	s.wsHub.Broadcast("deletion", fmt.Sprintf("Document '%s' deleted", req.Source))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// PromptBuilder is a simple prompt builder for RAG
type PromptBuilder struct{}

// BuildPrompt combines query and chunks into a RAG prompt
func (pb *PromptBuilder) BuildPrompt(query string, chunks []Chunk) string {
	var sb strings.Builder

	sb.WriteString("You are a helpful assistant. Use the following context to answer the user's question.\n\n")
	sb.WriteString("Context:\n")

	for i, chunk := range chunks {
		sb.WriteString(fmt.Sprintf("\n[%d] Source: %s\n%s\n", i+1, chunk.Source, chunk.Text))
	}

	sb.WriteString("\n\nUser Question: ")
	sb.WriteString(query)
	sb.WriteString("\n\nAnswer based on the context above:")

	return sb.String()
}

// generateSessionID creates a random session ID
func generateSessionID() string {
	bytes := make([]byte, 16)
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
	data := map[string]interface{}{
		"PrivacyMode": s.config.PrivacyMode,
		"Provider":    s.config.Provider,
	}

	if err := s.templates.ExecuteTemplate(w, "settings.html", data); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}
}

// handleConfig saves configuration changes
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// TODO: Implement configuration saving
	// This requires access to the full config object and the ability to save it
	// For now, return a placeholder response
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"success": true, "message": "Configuration saved (placeholder)"}`))
}

// handleTestConnection tests provider connectivity
func (s *Server) handleTestConnection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Test embedding with a simple text
	_, err := s.provider.Embed(ctx, "test")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf(`{"success": false, "error": "%s"}`, err.Error())))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"success": true, "message": "Connection successful"}`))
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

// handleSkills lists available skills
func (s *Server) handleSkills(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Load all skills
	skills, err := s.skillsLoader.LoadAll()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load skills: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert to JSON-friendly format
	type SkillInfo struct {
		Name        string   `json:"name"`
		Version     string   `json:"version"`
		Description string   `json:"description"`
		Triggers    []string `json:"triggers"`
		RequiresNet bool     `json:"requires_network"`
	}

	skillsInfo := make([]SkillInfo, 0, len(skills))
	for _, skill := range skills {
		triggers := make([]string, 0, len(skill.Triggers))
		for _, trigger := range skill.Triggers {
			triggers = append(triggers, trigger.Type)
		}

		skillsInfo = append(skillsInfo, SkillInfo{
			Name:        skill.Name,
			Version:     skill.Version,
			Description: skill.Description,
			Triggers:    triggers,
			RequiresNet: skill.RequiresNet,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"skills": skillsInfo,
	})
}

// handleRunSkill executes a manual-trigger skill
func (s *Server) handleRunSkill(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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

	// Load all skills and find the requested one
	skills, err := s.skillsLoader.LoadAll()
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
	ctx := r.Context()
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
