package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Mock implementations for testing

type mockStore struct{}

func (m *mockStore) SaveChunk(ctx context.Context, source, text string, embedding []float32, tags []string, summary string) error {
	return nil
}

func (m *mockStore) Search(ctx context.Context, queryVec []float32, topK int) ([]Chunk, error) {
	return []Chunk{}, nil
}

func (m *mockStore) Library(ctx context.Context) ([]LibraryEntry, error) {
	return []LibraryEntry{}, nil
}

func (m *mockStore) DeleteSource(ctx context.Context, source string) error {
	return nil
}

func (m *mockStore) SaveMessage(ctx context.Context, sessionID, role, content string) error {
	return nil
}

func (m *mockStore) GetSessionHistory(ctx context.Context, sessionID string) ([]ChatMessage, error) {
	return []ChatMessage{}, nil
}

func (m *mockStore) ListSessions(ctx context.Context) ([]Session, error) {
	return []Session{}, nil
}

func (m *mockStore) AddAuditEntry(ctx context.Context, opType, details, userCtx string) error {
	return nil
}

func (m *mockStore) GetAuditLog(ctx context.Context, opType string, from, to time.Time) ([]AuditEntry, error) {
	return []AuditEntry{}, nil
}

func (m *mockStore) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	return &User{ID: 1, Username: username}, nil
}

func (m *mockStore) CreateUser(ctx context.Context, username, password, email string, isAdmin, mustChangePassword bool) (int64, error) {
	return 1, nil
}

func (m *mockStore) UpdatePassword(ctx context.Context, userID int64, newPassword string) error {
	return nil
}

func (m *mockStore) GetUserByID(ctx context.Context, userID int64) (*User, error) {
	return &User{ID: userID, Username: "testuser"}, nil
}

func (m *mockStore) ListUsers(ctx context.Context) ([]User, error) {
	return []User{}, nil
}

func (m *mockStore) DeleteUser(ctx context.Context, userID int64) error {
	return nil
}

func (m *mockStore) SearchByUser(ctx context.Context, userID int64, queryVec []float32, topK int) ([]Chunk, error) {
	return []Chunk{}, nil
}

func (m *mockStore) LibraryByUser(ctx context.Context, userID int64) ([]LibraryEntry, error) {
	return []LibraryEntry{}, nil
}

func (m *mockStore) SaveChatMessage(ctx context.Context, userID int64, sessionID, role, content string) error {
	return nil
}

func (m *mockStore) GetUserSessions(ctx context.Context, userID int64) ([]Session, error) {
	return []Session{}, nil
}

func (m *mockStore) GetSessionOwner(ctx context.Context, sessionID string) (int64, error) {
	return 0, nil
}

func (m *mockStore) GetSessionMessages(ctx context.Context, userID int64, sessionID string) ([]ChatMessage, error) {
	return []ChatMessage{}, nil
}

func (m *mockStore) GetUserSkills(ctx context.Context, userID int64) ([]Skill, error) {
	return []Skill{}, nil
}

func (m *mockStore) GetWatchedFoldersByUser(ctx context.Context, userID int64) ([]WatchedFolder, error) {
	return []WatchedFolder{}, nil
}

// mockAuthProvider is defined in auth_handlers_test.go

type mockProvider struct{}

func (m *mockProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	return []float32{0.1, 0.2, 0.3}, nil
}

func (m *mockProvider) Stream(ctx context.Context, messages []Message, w io.Writer) (string, error) {
	return "test response", nil
}

func (m *mockProvider) Name() string {
	return "mock"
}

func (m *mockProvider) IsLocal() bool {
	return true
}

type mockIngester struct{}

func (m *mockIngester) IngestText(ctx context.Context, userID int64, source, text string, tags []string) error {
	return nil
}

func (m *mockIngester) IngestURL(ctx context.Context, userID int64, url string, tags []string) error {
	return nil
}

type mockSearcher struct{}

func (m *mockSearcher) Search(ctx context.Context, queryVec []float32, topK int) ([]Chunk, error) {
	return []Chunk{}, nil
}

type mockLogger struct{}

func (m *mockLogger) Debug(format string, args ...interface{})         {}
func (m *mockLogger) Info(format string, args ...interface{})          {}
func (m *mockLogger) Warn(format string, args ...interface{})          {}
func (m *mockLogger) Error(format string, args ...interface{})         {}
func (m *mockLogger) WithContext(key string, value interface{}) Logger { return m }
func (m *mockLogger) WithFields(fields map[string]interface{}) Logger  { return m }

// Tests

func TestNewServer(t *testing.T) {
	store := &mockStore{}
	provider := &mockProvider{}
	ingester := &mockIngester{}
	searcher := &mockSearcher{}
	logger := &mockLogger{}
	config := &ServerConfig{
		PrivacyMode: true,
		Provider:    "ollama",
	}

	// Use the correct path from the test's perspective (running from noodexx directory)
	srv, err := NewServerWithTemplatePath(store, provider, ingester, searcher, config, nil, nil, logger, &mockAuthProvider{}, "config.json", "../../web/templates/*.html")
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	if srv == nil {
		t.Fatal("NewServer returned nil server")
	}

	if srv.store != store {
		t.Error("Server store not set correctly")
	}

	if srv.provider != provider {
		t.Error("Server provider not set correctly")
	}

	if srv.ingester != ingester {
		t.Error("Server ingester not set correctly")
	}

	if srv.searcher != searcher {
		t.Error("Server searcher not set correctly")
	}

	if srv.config != config {
		t.Error("Server config not set correctly")
	}

	if srv.templates == nil {
		t.Error("Server templates not loaded")
	}
}

func TestRegisterRoutes(t *testing.T) {
	store := &mockStore{}
	provider := &mockProvider{}
	ingester := &mockIngester{}
	searcher := &mockSearcher{}
	logger := &mockLogger{}
	config := &ServerConfig{
		PrivacyMode: true,
		Provider:    "ollama",
	}

	srv, err := NewServerWithTemplatePath(store, provider, ingester, searcher, config, nil, nil, logger, &mockAuthProvider{}, "config.json", "../../web/templates/*.html")
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	// Test that routes are registered by making test requests
	// Note: These tests don't apply the auth middleware, so routes that require auth will return 401
	testCases := []struct {
		path           string
		method         string
		expectedStatus int
		description    string
	}{
		{"/", "GET", http.StatusOK, "dashboard should return 200"},
		{"/chat", "GET", http.StatusOK, "chat page should return 200"},
		{"/library", "GET", http.StatusUnauthorized, "library page requires auth"},
		{"/settings", "GET", http.StatusOK, "settings page should return 200"},
		{"/api/ask", "POST", http.StatusUnauthorized, "ask endpoint requires auth"},
		{"/api/ingest/text", "POST", http.StatusUnauthorized, "ingest text requires auth"},
		{"/api/ingest/url", "POST", http.StatusUnauthorized, "ingest url requires auth"},
		{"/api/sessions", "GET", http.StatusUnauthorized, "sessions endpoint requires auth"},
		{"/api/config", "POST", http.StatusOK, "config endpoint should return 200"},
		{"/api/activity", "GET", http.StatusOK, "activity endpoint should return 200"},
		{"/api/login", "POST", http.StatusBadRequest, "login endpoint should return 400 for empty request"},
		{"/api/register", "POST", http.StatusBadRequest, "register endpoint should return 400 for empty request"},
	}

	for _, tc := range testCases {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != tc.expectedStatus {
			t.Errorf("Route %s %s: %s - expected status %d, got %d", tc.method, tc.path, tc.description, tc.expectedStatus, w.Code)
		}
	}
}

func TestStaticFileServing(t *testing.T) {
	store := &mockStore{}
	provider := &mockProvider{}
	ingester := &mockIngester{}
	searcher := &mockSearcher{}
	logger := &mockLogger{}
	config := &ServerConfig{
		PrivacyMode: true,
		Provider:    "ollama",
	}

	srv, err := NewServerWithTemplatePath(store, provider, ingester, searcher, config, nil, nil, logger, &mockAuthProvider{}, "config.json", "../../web/templates/*.html")
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	// Test static file serving
	req := httptest.NewRequest("GET", "/static/style.css", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Should return 200 OK if file exists, or 404 if not found
	if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
		t.Errorf("Static file serving: expected status 200 or 404, got %d", w.Code)
	}
}
