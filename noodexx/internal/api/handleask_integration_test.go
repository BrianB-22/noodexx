package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"noodexx/internal/auth"
	"testing"
	"time"
)

// Mock implementations for handleAsk integration tests

// mockProviderForAsk implements LLMProvider for testing handleAsk
type mockProviderForAsk struct {
	embedFunc  func(ctx context.Context, text string) ([]float32, error)
	streamFunc func(ctx context.Context, messages []Message, w io.Writer) (string, error)
	name       string
	isLocal    bool
}

func (m *mockProviderForAsk) Embed(ctx context.Context, text string) ([]float32, error) {
	if m.embedFunc != nil {
		return m.embedFunc(ctx, text)
	}
	return []float32{0.1, 0.2, 0.3}, nil
}

func (m *mockProviderForAsk) Stream(ctx context.Context, messages []Message, w io.Writer) (string, error) {
	if m.streamFunc != nil {
		return m.streamFunc(ctx, messages, w)
	}
	response := "test response"
	w.Write([]byte(response))
	return response, nil
}

func (m *mockProviderForAsk) Name() string {
	return m.name
}

func (m *mockProviderForAsk) IsLocal() bool {
	return m.isLocal
}

// mockProviderManagerForAsk implements ProviderManager for testing
type mockProviderManagerForAsk struct {
	provider     LLMProvider
	providerName string
	err          error
}

func (m *mockProviderManagerForAsk) GetActiveProvider() (LLMProvider, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.provider, nil
}

func (m *mockProviderManagerForAsk) IsLocalMode() bool {
	return m.provider != nil && m.provider.IsLocal()
}

func (m *mockProviderManagerForAsk) GetProviderName() string {
	return m.providerName
}

func (m *mockProviderManagerForAsk) Reload(cfg interface{}) error {
	return nil
}

// mockRAGEnforcerForAsk implements RAGEnforcer for testing
type mockRAGEnforcerForAsk struct {
	shouldPerformRAG bool
	ragStatus        string
}

func (m *mockRAGEnforcerForAsk) ShouldPerformRAG() bool {
	return m.shouldPerformRAG
}

func (m *mockRAGEnforcerForAsk) GetRAGStatus() string {
	return m.ragStatus
}

// mockStoreForAsk implements Store for testing handleAsk
type mockStoreForAsk struct {
	searchByUserFunc    func(ctx context.Context, userID int64, queryVec []float32, topK int) ([]Chunk, error)
	saveChatMessageFunc func(ctx context.Context, userID int64, sessionID, role, content, providerMode string) error
	getSessionOwnerFunc func(ctx context.Context, sessionID string) (int64, error)
	addAuditEntryFunc   func(ctx context.Context, opType, details, userCtx string) error
}

func (m *mockStoreForAsk) SearchByUser(ctx context.Context, userID int64, queryVec []float32, topK int) ([]Chunk, error) {
	if m.searchByUserFunc != nil {
		return m.searchByUserFunc(ctx, userID, queryVec, topK)
	}
	return []Chunk{
		{Source: "test.txt", Text: "test chunk 1"},
		{Source: "test.txt", Text: "test chunk 2"},
	}, nil
}

func (m *mockStoreForAsk) SaveChatMessage(ctx context.Context, userID int64, sessionID, role, content, providerMode string) error {
	if m.saveChatMessageFunc != nil {
		return m.saveChatMessageFunc(ctx, userID, sessionID, role, content, providerMode)
	}
	return nil
}

func (m *mockStoreForAsk) GetSessionOwner(ctx context.Context, sessionID string) (int64, error) {
	if m.getSessionOwnerFunc != nil {
		return m.getSessionOwnerFunc(ctx, sessionID)
	}
	return 0, nil
}

func (m *mockStoreForAsk) AddAuditEntry(ctx context.Context, opType, details, userCtx string) error {
	if m.addAuditEntryFunc != nil {
		return m.addAuditEntryFunc(ctx, opType, details, userCtx)
	}
	return nil
}

// Stub methods for Store interface
func (m *mockStoreForAsk) SaveChunk(ctx context.Context, source, text string, embedding []float32, tags []string, summary string) error {
	return nil
}
func (m *mockStoreForAsk) Search(ctx context.Context, queryVec []float32, topK int) ([]Chunk, error) {
	return nil, nil
}
func (m *mockStoreForAsk) Library(ctx context.Context) ([]LibraryEntry, error) {
	return nil, nil
}
func (m *mockStoreForAsk) DeleteSource(ctx context.Context, source string) error {
	return nil
}
func (m *mockStoreForAsk) SaveMessage(ctx context.Context, sessionID, role, content string) error {
	return nil
}
func (m *mockStoreForAsk) GetSessionHistory(ctx context.Context, sessionID string) ([]ChatMessage, error) {
	return nil, nil
}
func (m *mockStoreForAsk) ListSessions(ctx context.Context) ([]Session, error) {
	return nil, nil
}
func (m *mockStoreForAsk) GetAuditLog(ctx context.Context, opType string, from, to time.Time) ([]AuditEntry, error) {
	return nil, nil
}
func (m *mockStoreForAsk) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	return nil, nil
}
func (m *mockStoreForAsk) CreateUser(ctx context.Context, username, password, email string, isAdmin, mustChangePassword bool) (int64, error) {
	return 0, nil
}
func (m *mockStoreForAsk) UpdatePassword(ctx context.Context, userID int64, newPassword string) error {
	return nil
}
func (m *mockStoreForAsk) GetUserByID(ctx context.Context, userID int64) (*User, error) {
	return nil, nil
}
func (m *mockStoreForAsk) ListUsers(ctx context.Context) ([]User, error) {
	return nil, nil
}
func (m *mockStoreForAsk) DeleteUser(ctx context.Context, userID int64) error {
	return nil
}
func (m *mockStoreForAsk) LibraryByUser(ctx context.Context, userID int64) ([]LibraryEntry, error) {
	return nil, nil
}
func (m *mockStoreForAsk) GetUserSessions(ctx context.Context, userID int64) ([]Session, error) {
	return nil, nil
}
func (m *mockStoreForAsk) GetSessionMessages(ctx context.Context, userID int64, sessionID string) ([]ChatMessage, error) {
	return nil, nil
}
func (m *mockStoreForAsk) GetUserSkills(ctx context.Context, userID int64) ([]Skill, error) {
	return nil, nil
}
func (m *mockStoreForAsk) GetWatchedFoldersByUser(ctx context.Context, userID int64) ([]WatchedFolder, error) {
	return nil, nil
}

// mockLoggerForAsk implements Logger for testing
type mockLoggerForAsk struct{}

func (m *mockLoggerForAsk) Debug(format string, args ...interface{})         {}
func (m *mockLoggerForAsk) Info(format string, args ...interface{})          {}
func (m *mockLoggerForAsk) Warn(format string, args ...interface{})          {}
func (m *mockLoggerForAsk) Error(format string, args ...interface{})         {}
func (m *mockLoggerForAsk) WithContext(key string, value interface{}) Logger { return m }
func (m *mockLoggerForAsk) WithFields(fields map[string]interface{}) Logger  { return m }

// TestHandleAsk_LocalProviderWithRAG tests query with local provider and RAG enabled
func TestHandleAsk_LocalProviderWithRAG(t *testing.T) {
	// Track whether RAG was performed
	ragPerformed := false
	embedCalled := false

	// Create mock provider
	provider := &mockProviderForAsk{
		name:    "ollama",
		isLocal: true,
		embedFunc: func(ctx context.Context, text string) ([]float32, error) {
			embedCalled = true
			return []float32{0.1, 0.2, 0.3}, nil
		},
		streamFunc: func(ctx context.Context, messages []Message, w io.Writer) (string, error) {
			// Verify prompt contains context (RAG was performed)
			if len(messages) > 0 {
				for _, msg := range messages {
					if msg.Role == "user" && len(msg.Content) > 0 {
						// Check if prompt contains "Context:" which indicates RAG
						if bytes.Contains([]byte(msg.Content), []byte("Context:")) {
							ragPerformed = true
						}
					}
				}
			}
			response := "test response"
			w.Write([]byte(response))
			return response, nil
		},
	}

	// Create mock store
	store := &mockStoreForAsk{
		searchByUserFunc: func(ctx context.Context, userID int64, queryVec []float32, topK int) ([]Chunk, error) {
			return []Chunk{
				{Source: "test.txt", Text: "test chunk 1"},
				{Source: "test.txt", Text: "test chunk 2"},
			}, nil
		},
	}

	// Create server
	server := &Server{
		store:           store,
		logger:          &mockLoggerForAsk{},
		providerManager: &mockProviderManagerForAsk{provider: provider, providerName: "Ollama (llama3.2)"},
		ragEnforcer:     &mockRAGEnforcerForAsk{shouldPerformRAG: true, ragStatus: "RAG Enabled (Local)"},
	}

	// Create request
	reqBody := map[string]string{
		"query":      "test query",
		"session_id": "test-session",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/ask", bytes.NewReader(bodyBytes))

	// Add user context
	ctx := context.WithValue(req.Context(), auth.UserIDKey, int64(1))
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Execute handler
	server.handleAsk(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify headers
	if providerName := w.Header().Get("X-Provider-Name"); providerName != "Ollama (llama3.2)" {
		t.Errorf("Expected X-Provider-Name='Ollama (llama3.2)', got '%s'", providerName)
	}

	if ragStatus := w.Header().Get("X-RAG-Status"); ragStatus != "RAG Enabled (Local)" {
		t.Errorf("Expected X-RAG-Status='RAG Enabled (Local)', got '%s'", ragStatus)
	}

	// Verify RAG was performed
	if !embedCalled {
		t.Error("Expected Embed to be called for RAG, but it wasn't")
	}

	if !ragPerformed {
		t.Error("Expected RAG to be performed (prompt should contain context), but it wasn't")
	}
}

// TestHandleAsk_CloudProviderWithNoRAGPolicy tests query with cloud provider and no-RAG policy
func TestHandleAsk_CloudProviderWithNoRAGPolicy(t *testing.T) {
	// Track whether RAG was performed
	embedCalled := false
	hasChunkContent := false

	// Create mock provider
	provider := &mockProviderForAsk{
		name:    "openai",
		isLocal: false,
		embedFunc: func(ctx context.Context, text string) ([]float32, error) {
			embedCalled = true
			return []float32{0.1, 0.2, 0.3}, nil
		},
		streamFunc: func(ctx context.Context, messages []Message, w io.Writer) (string, error) {
			// Verify prompt does NOT contain chunk content (RAG was skipped)
			if len(messages) > 0 {
				for _, msg := range messages {
					if msg.Role == "user" && len(msg.Content) > 0 {
						// Check if prompt contains "[1] Source:" which indicates actual chunks
						if bytes.Contains([]byte(msg.Content), []byte("[1] Source:")) {
							hasChunkContent = true
						}
					}
				}
			}
			response := "test response"
			w.Write([]byte(response))
			return response, nil
		},
	}

	// Create mock store
	store := &mockStoreForAsk{}

	// Create server
	server := &Server{
		store:           store,
		logger:          &mockLoggerForAsk{},
		providerManager: &mockProviderManagerForAsk{provider: provider, providerName: "OpenAI (gpt-4)"},
		ragEnforcer:     &mockRAGEnforcerForAsk{shouldPerformRAG: false, ragStatus: "RAG Disabled (Cloud Policy)"},
	}

	// Create request
	reqBody := map[string]string{
		"query":      "test query",
		"session_id": "test-session",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/ask", bytes.NewReader(bodyBytes))

	// Add user context
	ctx := context.WithValue(req.Context(), auth.UserIDKey, int64(1))
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Execute handler
	server.handleAsk(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify headers
	if providerName := w.Header().Get("X-Provider-Name"); providerName != "OpenAI (gpt-4)" {
		t.Errorf("Expected X-Provider-Name='OpenAI (gpt-4)', got '%s'", providerName)
	}

	if ragStatus := w.Header().Get("X-RAG-Status"); ragStatus != "RAG Disabled (Cloud Policy)" {
		t.Errorf("Expected X-RAG-Status='RAG Disabled (Cloud Policy)', got '%s'", ragStatus)
	}

	// Verify RAG was NOT performed
	if embedCalled {
		t.Error("Expected Embed NOT to be called (RAG disabled), but it was")
	}

	if hasChunkContent {
		t.Error("Expected RAG NOT to be performed (prompt should not contain chunk content), but it was")
	}
}

// TestHandleAsk_CloudProviderWithAllowRAGPolicy tests query with cloud provider and allow-RAG policy
func TestHandleAsk_CloudProviderWithAllowRAGPolicy(t *testing.T) {
	// Track whether RAG was performed
	ragPerformed := false
	embedCalled := false

	// Create mock provider
	provider := &mockProviderForAsk{
		name:    "openai",
		isLocal: false,
		embedFunc: func(ctx context.Context, text string) ([]float32, error) {
			embedCalled = true
			return []float32{0.1, 0.2, 0.3}, nil
		},
		streamFunc: func(ctx context.Context, messages []Message, w io.Writer) (string, error) {
			// Verify prompt contains context (RAG was performed)
			if len(messages) > 0 {
				for _, msg := range messages {
					if msg.Role == "user" && len(msg.Content) > 0 {
						// Check if prompt contains "Context:" which indicates RAG
						if bytes.Contains([]byte(msg.Content), []byte("Context:")) {
							ragPerformed = true
						}
					}
				}
			}
			response := "test response"
			w.Write([]byte(response))
			return response, nil
		},
	}

	// Create mock store
	store := &mockStoreForAsk{
		searchByUserFunc: func(ctx context.Context, userID int64, queryVec []float32, topK int) ([]Chunk, error) {
			return []Chunk{
				{Source: "test.txt", Text: "test chunk 1"},
				{Source: "test.txt", Text: "test chunk 2"},
			}, nil
		},
	}

	// Create server
	server := &Server{
		store:           store,
		logger:          &mockLoggerForAsk{},
		providerManager: &mockProviderManagerForAsk{provider: provider, providerName: "OpenAI (gpt-4)"},
		ragEnforcer:     &mockRAGEnforcerForAsk{shouldPerformRAG: true, ragStatus: "RAG Enabled"},
	}

	// Create request
	reqBody := map[string]string{
		"query":      "test query",
		"session_id": "test-session",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/ask", bytes.NewReader(bodyBytes))

	// Add user context
	ctx := context.WithValue(req.Context(), auth.UserIDKey, int64(1))
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Execute handler
	server.handleAsk(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify headers
	if providerName := w.Header().Get("X-Provider-Name"); providerName != "OpenAI (gpt-4)" {
		t.Errorf("Expected X-Provider-Name='OpenAI (gpt-4)', got '%s'", providerName)
	}

	if ragStatus := w.Header().Get("X-RAG-Status"); ragStatus != "RAG Enabled" {
		t.Errorf("Expected X-RAG-Status='RAG Enabled', got '%s'", ragStatus)
	}

	// Verify RAG was performed
	if !embedCalled {
		t.Error("Expected Embed to be called for RAG, but it wasn't")
	}

	if !ragPerformed {
		t.Error("Expected RAG to be performed (prompt should contain context), but it wasn't")
	}
}

// TestHandleAsk_UnconfiguredLocalProvider tests error response for unconfigured local provider
func TestHandleAsk_UnconfiguredLocalProvider(t *testing.T) {
	// Create server with provider manager that returns error
	server := &Server{
		store:  &mockStoreForAsk{},
		logger: &mockLoggerForAsk{},
		providerManager: &mockProviderManagerForAsk{
			err: errors.New("local provider not configured"),
		},
		ragEnforcer: &mockRAGEnforcerForAsk{shouldPerformRAG: true, ragStatus: "RAG Enabled (Local)"},
	}

	// Create request
	reqBody := map[string]string{
		"query":      "test query",
		"session_id": "test-session",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/ask", bytes.NewReader(bodyBytes))

	// Add user context
	ctx := context.WithValue(req.Context(), auth.UserIDKey, int64(1))
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Execute handler
	server.handleAsk(w, req)

	// Verify error response
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	// Verify error message
	body := w.Body.String()
	if !bytes.Contains([]byte(body), []byte("Provider not configured")) {
		t.Errorf("Expected error message to contain 'Provider not configured', got: %s", body)
	}
}

// TestHandleAsk_UnconfiguredCloudProvider tests error response for unconfigured cloud provider
func TestHandleAsk_UnconfiguredCloudProvider(t *testing.T) {
	// Create server with provider manager that returns error
	server := &Server{
		store:  &mockStoreForAsk{},
		logger: &mockLoggerForAsk{},
		providerManager: &mockProviderManagerForAsk{
			err: errors.New("cloud provider not configured"),
		},
		ragEnforcer: &mockRAGEnforcerForAsk{shouldPerformRAG: false, ragStatus: "RAG Disabled (Cloud Policy)"},
	}

	// Create request
	reqBody := map[string]string{
		"query":      "test query",
		"session_id": "test-session",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/ask", bytes.NewReader(bodyBytes))

	// Add user context
	ctx := context.WithValue(req.Context(), auth.UserIDKey, int64(1))
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Execute handler
	server.handleAsk(w, req)

	// Verify error response
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	// Verify error message
	body := w.Body.String()
	if !bytes.Contains([]byte(body), []byte("Provider not configured")) {
		t.Errorf("Expected error message to contain 'Provider not configured', got: %s", body)
	}
}
