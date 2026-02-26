package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestAuthMiddleware_SingleUserMode tests middleware in single-user mode
func TestAuthMiddleware_SingleUserMode(t *testing.T) {
	store := NewMockStore()

	// Create local-default user
	hash, _ := hashPassword("password")
	store.users["local-default"] = &User{
		ID:           1,
		Username:     "local-default",
		PasswordHash: hash,
		IsAdmin:      true,
	}

	middleware := AuthMiddleware(store, "single")

	// Create a test handler that checks for user_id in context
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := GetUserID(r.Context())
		if err != nil {
			t.Errorf("Expected user_id in context, got error: %v", err)
		}
		if userID != 1 {
			t.Errorf("Expected userID 1, got %d", userID)
		}
		w.WriteHeader(http.StatusOK)
	})

	// Wrap handler with middleware
	wrappedHandler := middleware(handler)

	// Test request without any authentication
	req := httptest.NewRequest("GET", "/api/library", nil)
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestAuthMiddleware_MultiUserMode_ValidToken tests middleware with valid token
func TestAuthMiddleware_MultiUserMode_ValidToken(t *testing.T) {
	store := NewMockStore()

	// Create a test user
	hash, _ := hashPassword("password")
	store.users["testuser"] = &User{
		ID:           2,
		Username:     "testuser",
		PasswordHash: hash,
	}

	// Create a valid session token
	token := "valid-token-123"
	store.tokens[token] = &SessionToken{
		Token:     token,
		UserID:    2,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	middleware := AuthMiddleware(store, "multi")

	// Create a test handler that checks for user_id in context
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := GetUserID(r.Context())
		if err != nil {
			t.Errorf("Expected user_id in context, got error: %v", err)
		}
		if userID != 2 {
			t.Errorf("Expected userID 2, got %d", userID)
		}
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(handler)

	// Test with Authorization header
	req := httptest.NewRequest("GET", "/api/library", nil)
	req.Header.Set("Authorization", "Bearer valid-token-123")
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestAuthMiddleware_MultiUserMode_InvalidToken tests middleware with invalid token
func TestAuthMiddleware_MultiUserMode_InvalidToken(t *testing.T) {
	store := NewMockStore()

	middleware := AuthMiddleware(store, "multi")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called with invalid token")
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(handler)

	// Test with invalid token
	req := httptest.NewRequest("GET", "/api/library", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

// TestAuthMiddleware_MultiUserMode_NoToken tests middleware without token
func TestAuthMiddleware_MultiUserMode_NoToken(t *testing.T) {
	store := NewMockStore()

	middleware := AuthMiddleware(store, "multi")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called without token")
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(handler)

	// Test without token
	req := httptest.NewRequest("GET", "/api/library", nil)
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

// TestAuthMiddleware_PublicEndpoints tests that public endpoints bypass auth
func TestAuthMiddleware_PublicEndpoints(t *testing.T) {
	store := NewMockStore()

	middleware := AuthMiddleware(store, "multi")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(handler)

	publicPaths := []string{"/login", "/register", "/static/css/style.css"}

	for _, path := range publicPaths {
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Public endpoint %s should return 200, got %d", path, w.Code)
		}
	}
}

// TestExtractToken_FromHeader tests token extraction from Authorization header
func TestExtractToken_FromHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/library", nil)
	req.Header.Set("Authorization", "Bearer test-token-123")

	token := extractToken(req)
	if token != "test-token-123" {
		t.Errorf("Expected token 'test-token-123', got '%s'", token)
	}
}

// TestExtractToken_FromCookie tests token extraction from cookie
func TestExtractToken_FromCookie(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/library", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_token",
		Value: "cookie-token-456",
	})

	token := extractToken(req)
	if token != "cookie-token-456" {
		t.Errorf("Expected token 'cookie-token-456', got '%s'", token)
	}
}

// TestExtractToken_HeaderPriority tests that header takes priority over cookie
func TestExtractToken_HeaderPriority(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/library", nil)
	req.Header.Set("Authorization", "Bearer header-token")
	req.AddCookie(&http.Cookie{
		Name:  "session_token",
		Value: "cookie-token",
	})

	token := extractToken(req)
	if token != "header-token" {
		t.Errorf("Expected header token 'header-token', got '%s'", token)
	}
}

// TestExtractToken_NoToken tests extraction when no token is present
func TestExtractToken_NoToken(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/library", nil)

	token := extractToken(req)
	if token != "" {
		t.Errorf("Expected empty token, got '%s'", token)
	}
}

// TestExtractToken_InvalidHeaderFormat tests extraction with invalid header format
func TestExtractToken_InvalidHeaderFormat(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/library", nil)
	req.Header.Set("Authorization", "InvalidFormat token-123")

	token := extractToken(req)
	if token != "" {
		t.Errorf("Expected empty token for invalid format, got '%s'", token)
	}
}

// TestIsPublicEndpoint tests public endpoint detection
func TestIsPublicEndpoint(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/login", true},
		{"/register", true},
		{"/static/css/style.css", true},
		{"/static/js/app.js", true},
		{"/api/library", false},
		{"/api/search", false},
		{"/dashboard", false},
		{"/settings", false},
	}

	for _, tt := range tests {
		result := isPublicEndpoint(tt.path)
		if result != tt.expected {
			t.Errorf("isPublicEndpoint(%s) = %v, expected %v", tt.path, result, tt.expected)
		}
	}
}

// TestGetUserID_Success tests successful user ID extraction
func TestGetUserID_Success(t *testing.T) {
	ctx := context.WithValue(context.Background(), UserIDKey, int64(42))

	userID, err := GetUserID(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if userID != 42 {
		t.Errorf("Expected userID 42, got %d", userID)
	}
}

// TestGetUserID_NotFound tests error when user ID not in context
func TestGetUserID_NotFound(t *testing.T) {
	ctx := context.Background()

	_, err := GetUserID(ctx)
	if err == nil {
		t.Error("Expected error when user_id not in context")
	}
	if err != ErrUserIDNotFound {
		t.Errorf("Expected ErrUserIDNotFound, got %v", err)
	}
}

// TestGetUserID_WrongType tests error when context value is wrong type
func TestGetUserID_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), UserIDKey, "not-an-int")

	_, err := GetUserID(ctx)
	if err == nil {
		t.Error("Expected error when user_id is wrong type")
	}
}

// TestAuthMiddleware_TokenFromCookie tests middleware with cookie-based auth
func TestAuthMiddleware_TokenFromCookie(t *testing.T) {
	store := NewMockStore()

	// Create a test user
	hash, _ := hashPassword("password")
	store.users["testuser"] = &User{
		ID:           3,
		Username:     "testuser",
		PasswordHash: hash,
	}

	// Create a valid session token
	token := "cookie-token-789"
	store.tokens[token] = &SessionToken{
		Token:     token,
		UserID:    3,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	middleware := AuthMiddleware(store, "multi")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := GetUserID(r.Context())
		if err != nil {
			t.Errorf("Expected user_id in context, got error: %v", err)
		}
		if userID != 3 {
			t.Errorf("Expected userID 3, got %d", userID)
		}
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(handler)

	// Test with cookie
	req := httptest.NewRequest("GET", "/api/library", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_token",
		Value: token,
	})
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestAuthMiddleware_ExpiredToken tests middleware with expired token
func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	store := NewMockStore()

	// Create an expired session token
	token := "expired-token"
	store.tokens[token] = &SessionToken{
		Token:     token,
		UserID:    4,
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
	}

	middleware := AuthMiddleware(store, "multi")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called with expired token")
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(handler)

	req := httptest.NewRequest("GET", "/api/library", nil)
	req.Header.Set("Authorization", "Bearer expired-token")
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for expired token, got %d", w.Code)
	}
}
