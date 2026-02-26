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

// mockStoreForAdmin extends mockStoreForAuth with admin-specific methods
type mockStoreForAdmin struct {
	mockStoreForAuth
	getUserByIDFunc func(ctx context.Context, userID int64) (*User, error)
	listUsersFunc   func(ctx context.Context) ([]User, error)
	deleteUserFunc  func(ctx context.Context, userID int64) error
}

func (m *mockStoreForAdmin) GetUserByID(ctx context.Context, userID int64) (*User, error) {
	if m.getUserByIDFunc != nil {
		return m.getUserByIDFunc(ctx, userID)
	}
	return &User{
		ID:                 userID,
		Username:           fmt.Sprintf("user%d", userID),
		Email:              fmt.Sprintf("user%d@example.com", userID),
		IsAdmin:            userID == 1, // User 1 is admin
		MustChangePassword: false,
		CreatedAt:          time.Now(),
		LastLogin:          time.Now(),
	}, nil
}

func (m *mockStoreForAdmin) ListUsers(ctx context.Context) ([]User, error) {
	if m.listUsersFunc != nil {
		return m.listUsersFunc(ctx)
	}
	return []User{
		{
			ID:                 1,
			Username:           "admin",
			Email:              "admin@example.com",
			IsAdmin:            true,
			MustChangePassword: false,
			CreatedAt:          time.Now(),
			LastLogin:          time.Now(),
		},
		{
			ID:                 2,
			Username:           "user2",
			Email:              "user2@example.com",
			IsAdmin:            false,
			MustChangePassword: false,
			CreatedAt:          time.Now(),
			LastLogin:          time.Now(),
		},
	}, nil
}

func (m *mockStoreForAdmin) DeleteUser(ctx context.Context, userID int64) error {
	if m.deleteUserFunc != nil {
		return m.deleteUserFunc(ctx, userID)
	}
	return nil
}

func TestHandleGetUsers(t *testing.T) {
	tests := []struct {
		name           string
		userID         int64
		isAdmin        bool
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:           "admin can list users",
			userID:         1,
			isAdmin:        true,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				users, ok := resp["users"].([]interface{})
				if !ok {
					t.Fatal("response missing users array")
				}
				if len(users) != 2 {
					t.Errorf("expected 2 users, got %d", len(users))
				}
			},
		},
		{
			name:           "non-admin cannot list users",
			userID:         2,
			isAdmin:        false,
			expectedStatus: http.StatusForbidden,
			checkResponse:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockStoreForAdmin{}
			store.getUserByIDFunc = func(ctx context.Context, userID int64) (*User, error) {
				return &User{
					ID:      userID,
					IsAdmin: tt.isAdmin,
				}, nil
			}

			server := &Server{
				store:  store,
				logger: &mockLogger{},
			}

			req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
			ctx := context.WithValue(req.Context(), auth.UserIDKey, tt.userID)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			server.handleGetUsers(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w.Body.Bytes())
			}
		})
	}
}

// TestHandleCreateUser tests the POST /api/users endpoint
func TestHandleCreateUser(t *testing.T) {
	tests := []struct {
		name           string
		userID         int64
		isAdmin        bool
		requestBody    map[string]interface{}
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:    "admin can create user",
			userID:  1,
			isAdmin: true,
			requestBody: map[string]interface{}{
				"username": "newuser",
				"email":    "newuser@example.com",
				"password": "password123",
				"is_admin": false,
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if !resp["success"].(bool) {
					t.Error("expected success to be true")
				}
				user, ok := resp["user"].(map[string]interface{})
				if !ok {
					t.Fatal("response missing user object")
				}
				if user["username"] != "newuser" {
					t.Errorf("expected username 'newuser', got %v", user["username"])
				}
			},
		},
		{
			name:    "non-admin cannot create user",
			userID:  2,
			isAdmin: false,
			requestBody: map[string]interface{}{
				"username": "newuser",
				"password": "password123",
			},
			expectedStatus: http.StatusForbidden,
			checkResponse:  nil,
		},
		{
			name:    "missing username returns 400",
			userID:  1,
			isAdmin: true,
			requestBody: map[string]interface{}{
				"password": "password123",
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse:  nil,
		},
		{
			name:    "short password returns 400",
			userID:  1,
			isAdmin: true,
			requestBody: map[string]interface{}{
				"username": "newuser",
				"password": "short",
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockStoreForAdmin{}
			store.getUserByIDFunc = func(ctx context.Context, userID int64) (*User, error) {
				if userID == 3 {
					// Return the newly created user
					username := "newuser"
					if tt.requestBody["username"] != nil {
						username = tt.requestBody["username"].(string)
					}
					email := ""
					if tt.requestBody["email"] != nil {
						email = tt.requestBody["email"].(string)
					}
					isAdmin := false
					if tt.requestBody["is_admin"] != nil {
						isAdmin = tt.requestBody["is_admin"].(bool)
					}
					return &User{
						ID:       userID,
						Username: username,
						Email:    email,
						IsAdmin:  isAdmin,
					}, nil
				}
				return &User{
					ID:      userID,
					IsAdmin: tt.isAdmin,
				}, nil
			}
			store.createUserFunc = func(ctx context.Context, username, password, email string, isAdmin, mustChangePassword bool) (int64, error) {
				return 3, nil
			}

			server := &Server{
				store:  store,
				logger: &mockLogger{},
			}

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewReader(body))
			ctx := context.WithValue(req.Context(), auth.UserIDKey, tt.userID)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			server.handleCreateUser(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w.Body.Bytes())
			}
		})
	}
}

// TestHandleDeleteUser tests the DELETE /api/users/:id endpoint
func TestHandleDeleteUser(t *testing.T) {
	tests := []struct {
		name           string
		userID         int64
		isAdmin        bool
		targetUserID   string
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:           "admin can delete other user",
			userID:         1,
			isAdmin:        true,
			targetUserID:   "2",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if !resp["success"].(bool) {
					t.Error("expected success to be true")
				}
			},
		},
		{
			name:           "admin cannot delete themselves",
			userID:         1,
			isAdmin:        true,
			targetUserID:   "1",
			expectedStatus: http.StatusBadRequest,
			checkResponse:  nil,
		},
		{
			name:           "non-admin cannot delete user",
			userID:         2,
			isAdmin:        false,
			targetUserID:   "3",
			expectedStatus: http.StatusForbidden,
			checkResponse:  nil,
		},
		{
			name:           "invalid user ID returns 400",
			userID:         1,
			isAdmin:        true,
			targetUserID:   "invalid",
			expectedStatus: http.StatusBadRequest,
			checkResponse:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockStoreForAdmin{}
			store.getUserByIDFunc = func(ctx context.Context, userID int64) (*User, error) {
				if userID == tt.userID {
					return &User{
						ID:      userID,
						IsAdmin: tt.isAdmin,
					}, nil
				}
				return &User{
					ID:       userID,
					Username: fmt.Sprintf("user%d", userID),
				}, nil
			}

			server := &Server{
				store:  store,
				logger: &mockLogger{},
			}

			req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/users/%s", tt.targetUserID), nil)
			ctx := context.WithValue(req.Context(), auth.UserIDKey, tt.userID)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			server.handleDeleteUser(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w.Body.Bytes())
			}
		})
	}
}

// TestHandleResetUserPassword tests the POST /api/users/:id/reset-password endpoint
func TestHandleResetUserPassword(t *testing.T) {
	tests := []struct {
		name           string
		userID         int64
		isAdmin        bool
		targetUserID   string
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:           "admin can reset user password",
			userID:         1,
			isAdmin:        true,
			targetUserID:   "2",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if !resp["success"].(bool) {
					t.Error("expected success to be true")
				}
				tempPassword, ok := resp["temporary_password"].(string)
				if !ok || tempPassword == "" {
					t.Error("expected temporary_password in response")
				}
				if len(tempPassword) != 16 {
					t.Errorf("expected password length 16, got %d", len(tempPassword))
				}
			},
		},
		{
			name:           "non-admin cannot reset password",
			userID:         2,
			isAdmin:        false,
			targetUserID:   "3",
			expectedStatus: http.StatusForbidden,
			checkResponse:  nil,
		},
		{
			name:           "invalid user ID returns 400",
			userID:         1,
			isAdmin:        true,
			targetUserID:   "invalid",
			expectedStatus: http.StatusBadRequest,
			checkResponse:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockStoreForAdmin{}
			store.getUserByIDFunc = func(ctx context.Context, userID int64) (*User, error) {
				if userID == tt.userID {
					return &User{
						ID:      userID,
						IsAdmin: tt.isAdmin,
					}, nil
				}
				return &User{
					ID:       userID,
					Username: fmt.Sprintf("user%d", userID),
				}, nil
			}

			server := &Server{
				store:  store,
				logger: &mockLogger{},
			}

			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/users/%s/reset-password", tt.targetUserID), nil)
			ctx := context.WithValue(req.Context(), auth.UserIDKey, tt.userID)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			server.handleResetUserPassword(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w.Body.Bytes())
			}
		})
	}
}
