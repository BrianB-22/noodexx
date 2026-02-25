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

func (m *mockIngester) IngestText(ctx context.Context, source, text string, tags []string) error {
	return nil
}

func (m *mockIngester) IngestURL(ctx context.Context, url string, tags []string) error {
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
	srv, err := NewServerWithTemplatePath(store, provider, ingester, searcher, config, nil, nil, logger, "../../web/templates/*.html")
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

	srv, err := NewServerWithTemplatePath(store, provider, ingester, searcher, config, nil, nil, logger, "../../web/templates/*.html")
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	// Test that routes are registered by making test requests
	testCases := []struct {
		path   string
		method string
	}{
		{"/", "GET"},
		{"/chat", "GET"},
		{"/library", "GET"},
		{"/settings", "GET"},
		{"/api/ask", "POST"},
		{"/api/ingest/text", "POST"},
		{"/api/ingest/url", "POST"},
		{"/api/ingest/file", "POST"},
		{"/api/delete", "POST"},
		{"/api/sessions", "GET"},
		{"/api/session/test", "GET"},
		{"/api/config", "POST"},
		{"/api/activity", "GET"},
	}

	for _, tc := range testCases {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		// We expect 501 Not Implemented for now since handlers are placeholders
		if w.Code != http.StatusNotImplemented {
			t.Errorf("Route %s %s: expected status 501, got %d", tc.method, tc.path, w.Code)
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

	srv, err := NewServerWithTemplatePath(store, provider, ingester, searcher, config, nil, nil, logger, "../../web/templates/*.html")
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
