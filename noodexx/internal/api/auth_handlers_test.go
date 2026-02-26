package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"noodexx/internal/auth"
	"testing"
	"time"
)

// Mock implementations for testing

type mockAuthProvider struct {
	loginFunc         func(ctx context.Context, username, password string) (string, error)
	logoutFunc        func(ctx context.Context, token string) error
	validateTokenFunc func(ctx context.Context, token string) (int64, error)
	refreshTokenFunc  func(ctx context.Context, token string) (string, error)
}

func (m *mockAuthProvider) Login(ctx context.Context, username, password string) (string, error) {
	if m.loginFunc != nil {
		return m.loginFunc(ctx, username, password)
	}
	return "mock-token", nil
}

func (m *mockAuthProvider) Logout(ctx context.Context, token string) error {
	if m.logoutFunc != nil {
		return m.logoutFunc(ctx, token)
	}
	return nil
}

func (m *mockAuthProvider) ValidateToken(ctx context.Context, token string) (int64, error) {
	if m.validateTokenFunc != nil {
		return m.validateTokenFunc(ctx, token)
	}
	return 1, nil
}

func (m *mockAuthProvider) RefreshToken(ctx context.Context, token string) (string, error) {
	if m.refreshTokenFunc != nil {
		return m.refreshTokenFunc(ctx, token)
	}
	return "new-mock-token", nil
}

type mockStoreForAuth struct {
	getUserByUsernameFunc func(ctx context.Context, username string) (*User, error)
	createUserFunc        func(ctx context.Context, username, password, email string, isAdmin, mustChangePassword bool) (int64, error)
	updatePasswordFunc    func(ctx context.Context, userID int64, newPassword string) error
}

func (m *mockStoreForAuth) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	if m.getUserByUsernameFunc != nil {
		return m.getUserByUsernameFunc(ctx, username)
	}
	return &User{
		ID:                 1,
		Username:           username,
		Email:              "test@example.com",
		IsAdmin:            false,
		MustChangePassword: false,
		CreatedAt:          time.Now(),
	}, nil
}

func (m *mockStoreForAuth) CreateUser(ctx context.Context, username, password, email string, isAdmin, mustChangePassword bool) (int64, error) {
	if m.createUserFunc != nil {
		return m.createUserFunc(ctx, username, password, email, isAdmin, mustChangePassword)
	}
	return 1, nil
}

func (m *mockStoreForAuth) UpdatePassword(ctx context.Context, userID int64, newPassword string) error {
	if m.updatePasswordFunc != nil {
		return m.updatePasswordFunc(ctx, userID, newPassword)
	}
	return nil
}

// Stub methods for Store interface (not used in auth tests)
func (m *mockStoreForAuth) SaveChunk(ctx context.Context, source, text string, embedding []float32, tags []string, summary string) error {
	return nil
}
func (m *mockStoreForAuth) Search(ctx context.Context, queryVec []float32, topK int) ([]Chunk, error) {
	return nil, nil
}
func (m *mockStoreForAuth) Library(ctx context.Context) ([]LibraryEntry, error) {
	return nil, nil
}
func (m *mockStoreForAuth) DeleteSource(ctx context.Context, source string) error {
	return nil
}
func (m *mockStoreForAuth) SaveMessage(ctx context.Context, sessionID, role, content string) error {
	return nil
}
func (m *mockStoreForAuth) GetSessionHistory(ctx context.Context, sessionID string) ([]ChatMessage, error) {
	return nil, nil
}
func (m *mockStoreForAuth) ListSessions(ctx context.Context) ([]Session, error) {
	return nil, nil
}
func (m *mockStoreForAuth) AddAuditEntry(ctx context.Context, opType, details, userCtx string) error {
	return nil
}
func (m *mockStoreForAuth) GetAuditLog(ctx context.Context, opType string, from, to time.Time) ([]AuditEntry, error) {
	return nil, nil
}
func (m *mockStoreForAuth) GetUserByID(ctx context.Context, userID int64) (*User, error) {
	return &User{
		ID:       userID,
		Username: fmt.Sprintf("user%d", userID),
		IsAdmin:  false,
	}, nil
}
func (m *mockStoreForAuth) ListUsers(ctx context.Context) ([]User, error) {
	return nil, nil
}
func (m *mockStoreForAuth) DeleteUser(ctx context.Context, userID int64) error {
	return nil
}
func (m *mockStoreForAuth) SearchByUser(ctx context.Context, userID int64, queryVec []float32, topK int) ([]Chunk, error) {
	return nil, nil
}
func (m *mockStoreForAuth) LibraryByUser(ctx context.Context, userID int64) ([]LibraryEntry, error) {
	return nil, nil
}
func (m *mockStoreForAuth) SaveChatMessage(ctx context.Context, userID int64, sessionID, role, content, providerMode string) error {
	return nil
}
func (m *mockStoreForAuth) GetUserSessions(ctx context.Context, userID int64) ([]Session, error) {
	return nil, nil
}
func (m *mockStoreForAuth) GetSessionOwner(ctx context.Context, sessionID string) (int64, error) {
	return 0, nil
}
func (m *mockStoreForAuth) GetSessionMessages(ctx context.Context, userID int64, sessionID string) ([]ChatMessage, error) {
	return nil, nil
}
func (m *mockStoreForAuth) GetUserSkills(ctx context.Context, userID int64) ([]Skill, error) {
	return nil, nil
}
func (m *mockStoreForAuth) GetWatchedFoldersByUser(ctx context.Context, userID int64) ([]WatchedFolder, error) {
	return nil, nil
}

// mockLogger is defined in server_test.go

// Test handleLogin

func TestHandleLogin_Success(t *testing.T) {
	mockAuth := &mockAuthProvider{
		loginFunc: func(ctx context.Context, username, password string) (string, error) {
			if username == "testuser" && password == "testpass" {
				return "test-token-123", nil
			}
			return "", nil
		},
	}

	mockStore := &mockStoreForAuth{
		getUserByUsernameFunc: func(ctx context.Context, username string) (*User, error) {
			return &User{
				ID:                 1,
				Username:           "testuser",
				Email:              "test@example.com",
				IsAdmin:            false,
				MustChangePassword: false,
				CreatedAt:          time.Now(),
			}, nil
		},
	}

	server := &Server{
		authProvider: mockAuth,
		store:        mockStore,
		logger:       &mockLogger{},
	}

	reqBody := map[string]string{
		"username": "testuser",
		"password": "testpass",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleLogin(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["success"] != true {
		t.Errorf("Expected success=true, got %v", response["success"])
	}

	if response["redirect"] != "/" {
		t.Errorf("Expected redirect=/, got %v", response["redirect"])
	}

	// Check cookie was set
	cookies := w.Result().Cookies()
	found := false
	for _, cookie := range cookies {
		if cookie.Name == "session_token" && cookie.Value == "test-token-123" {
			found = true
			if !cookie.HttpOnly {
				t.Error("Expected HttpOnly cookie")
			}
			if cookie.SameSite != http.SameSiteLaxMode {
				t.Error("Expected SameSite=Lax")
			}
		}
	}
	if !found {
		t.Error("Expected session_token cookie to be set")
	}
}

func TestHandleLogin_MustChangePassword(t *testing.T) {
	mockAuth := &mockAuthProvider{
		loginFunc: func(ctx context.Context, username, password string) (string, error) {
			return "test-token-123", nil
		},
	}

	mockStore := &mockStoreForAuth{
		getUserByUsernameFunc: func(ctx context.Context, username string) (*User, error) {
			return &User{
				ID:                 1,
				Username:           "testuser",
				Email:              "test@example.com",
				IsAdmin:            false,
				MustChangePassword: true, // Must change password
				CreatedAt:          time.Now(),
			}, nil
		},
	}

	server := &Server{
		authProvider: mockAuth,
		store:        mockStore,
		logger:       &mockLogger{},
	}

	reqBody := map[string]string{
		"username": "testuser",
		"password": "testpass",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleLogin(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["must_change_password"] != true {
		t.Errorf("Expected must_change_password=true, got %v", response["must_change_password"])
	}

	if response["redirect"] != "/change-password" {
		t.Errorf("Expected redirect=/change-password, got %v", response["redirect"])
	}
}

func TestHandleLogin_InvalidCredentials(t *testing.T) {
	mockAuth := &mockAuthProvider{
		loginFunc: func(ctx context.Context, username, password string) (string, error) {
			return "", &mockError{msg: "invalid credentials"}
		},
	}

	server := &Server{
		authProvider: mockAuth,
		store:        &mockStoreForAuth{},
		logger:       &mockLogger{},
	}

	reqBody := map[string]string{
		"username": "testuser",
		"password": "wrongpass",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleLogin(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["success"] != false {
		t.Errorf("Expected success=false, got %v", response["success"])
	}
}

func TestHandleLogin_AccountLocked(t *testing.T) {
	mockAuth := &mockAuthProvider{
		loginFunc: func(ctx context.Context, username, password string) (string, error) {
			return "", &mockError{msg: "account locked until 2024-01-01"}
		},
	}

	server := &Server{
		authProvider: mockAuth,
		store:        &mockStoreForAuth{},
		logger:       &mockLogger{},
	}

	reqBody := map[string]string{
		"username": "testuser",
		"password": "testpass",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleLogin(w, req)

	if w.Code != http.StatusLocked {
		t.Errorf("Expected status 423, got %d", w.Code)
	}
}

func TestHandleLogin_MissingCredentials(t *testing.T) {
	server := &Server{
		authProvider: &mockAuthProvider{},
		store:        &mockStoreForAuth{},
		logger:       &mockLogger{},
	}

	reqBody := map[string]string{
		"username": "",
		"password": "",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleLogin(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// Test handleLogout

func TestHandleLogout_Success(t *testing.T) {
	mockAuth := &mockAuthProvider{
		logoutFunc: func(ctx context.Context, token string) error {
			return nil
		},
	}

	server := &Server{
		authProvider: mockAuth,
		logger:       &mockLogger{},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "test-token"})
	w := httptest.NewRecorder()

	server.handleLogout(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["success"] != true {
		t.Errorf("Expected success=true, got %v", response["success"])
	}

	// Check cookie was cleared
	cookies := w.Result().Cookies()
	found := false
	for _, cookie := range cookies {
		if cookie.Name == "session_token" {
			found = true
			if cookie.MaxAge != -1 {
				t.Errorf("Expected MaxAge=-1, got %d", cookie.MaxAge)
			}
		}
	}
	if !found {
		t.Error("Expected session_token cookie to be cleared")
	}
}

// Test handleRegister

func TestHandleRegister_Success(t *testing.T) {
	mockStore := &mockStoreForAuth{
		createUserFunc: func(ctx context.Context, username, password, email string, isAdmin, mustChangePassword bool) (int64, error) {
			if isAdmin || mustChangePassword {
				t.Error("Expected isAdmin=false and mustChangePassword=false")
			}
			return 1, nil
		},
	}

	server := &Server{
		store:  mockStore,
		logger: &mockLogger{},
	}

	reqBody := map[string]string{
		"username":         "newuser",
		"email":            "new@example.com",
		"password":         "password123",
		"confirm_password": "password123",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleRegister(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["success"] != true {
		t.Errorf("Expected success=true, got %v", response["success"])
	}
}

func TestHandleRegister_PasswordMismatch(t *testing.T) {
	server := &Server{
		store:  &mockStoreForAuth{},
		logger: &mockLogger{},
	}

	reqBody := map[string]string{
		"username":         "newuser",
		"email":            "new@example.com",
		"password":         "password123",
		"confirm_password": "different",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleRegister(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["success"] != false {
		t.Errorf("Expected success=false, got %v", response["success"])
	}
}

func TestHandleRegister_InvalidUsername(t *testing.T) {
	server := &Server{
		store:  &mockStoreForAuth{},
		logger: &mockLogger{},
	}

	testCases := []struct {
		name     string
		username string
	}{
		{"too short", "ab"},
		{"too long", "abcdefghijklmnopqrstuvwxyz1234567"},
		{"invalid chars", "user@name"},
		{"invalid chars", "user name"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqBody := map[string]string{
				"username":         tc.username,
				"email":            "new@example.com",
				"password":         "password123",
				"confirm_password": "password123",
			}
			body, _ := json.Marshal(reqBody)

			req := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewReader(body))
			w := httptest.NewRecorder()

			server.handleRegister(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d", w.Code)
			}
		})
	}
}

func TestHandleRegister_InvalidEmail(t *testing.T) {
	server := &Server{
		store:  &mockStoreForAuth{},
		logger: &mockLogger{},
	}

	reqBody := map[string]string{
		"username":         "newuser",
		"email":            "invalid-email",
		"password":         "password123",
		"confirm_password": "password123",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleRegister(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleRegister_DuplicateUser(t *testing.T) {
	mockStore := &mockStoreForAuth{
		createUserFunc: func(ctx context.Context, username, password, email string, isAdmin, mustChangePassword bool) (int64, error) {
			return 0, &mockError{msg: "UNIQUE constraint failed"}
		},
	}

	server := &Server{
		store:  mockStore,
		logger: &mockLogger{},
	}

	reqBody := map[string]string{
		"username":         "existinguser",
		"email":            "existing@example.com",
		"password":         "password123",
		"confirm_password": "password123",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleRegister(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", w.Code)
	}
}

// Test handleChangePassword

func TestHandleChangePassword_Success(t *testing.T) {
	mockStore := &mockStoreForAuth{
		updatePasswordFunc: func(ctx context.Context, userID int64, newPassword string) error {
			if userID != 1 {
				t.Errorf("Expected userID=1, got %d", userID)
			}
			return nil
		},
	}

	server := &Server{
		store:  mockStore,
		logger: &mockLogger{},
	}

	reqBody := map[string]string{
		"new_password":     "newpassword123",
		"confirm_password": "newpassword123",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/change-password", bytes.NewReader(body))
	// Add user_id to context (simulating auth middleware)
	ctx := context.WithValue(req.Context(), auth.UserIDKey, int64(1))
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	server.handleChangePassword(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["success"] != true {
		t.Errorf("Expected success=true, got %v", response["success"])
	}
}

func TestHandleChangePassword_PasswordMismatch(t *testing.T) {
	server := &Server{
		store:  &mockStoreForAuth{},
		logger: &mockLogger{},
	}

	reqBody := map[string]string{
		"new_password":     "newpassword123",
		"confirm_password": "different",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/change-password", bytes.NewReader(body))
	ctx := context.WithValue(req.Context(), auth.UserIDKey, int64(1))
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	server.handleChangePassword(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleChangePassword_PasswordTooShort(t *testing.T) {
	server := &Server{
		store:  &mockStoreForAuth{},
		logger: &mockLogger{},
	}

	reqBody := map[string]string{
		"new_password":     "short",
		"confirm_password": "short",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/change-password", bytes.NewReader(body))
	ctx := context.WithValue(req.Context(), auth.UserIDKey, int64(1))
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	server.handleChangePassword(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleChangePassword_Unauthorized(t *testing.T) {
	server := &Server{
		store:  &mockStoreForAuth{},
		logger: &mockLogger{},
	}

	reqBody := map[string]string{
		"new_password":     "newpassword123",
		"confirm_password": "newpassword123",
	}
	body, _ := json.Marshal(reqBody)

	// No user_id in context
	req := httptest.NewRequest(http.MethodPost, "/api/change-password", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleChangePassword(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

// Helper types

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}
