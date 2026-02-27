package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"noodexx/internal/auth"
	"testing"
	"time"
)

// mockLogger for testing
type mockLoggerForPreferences struct{}

func (m *mockLoggerForPreferences) Debug(format string, args ...interface{})         {}
func (m *mockLoggerForPreferences) Info(format string, args ...interface{})          {}
func (m *mockLoggerForPreferences) Warn(format string, args ...interface{})          {}
func (m *mockLoggerForPreferences) Error(format string, args ...interface{})         {}
func (m *mockLoggerForPreferences) WithContext(key string, value interface{}) Logger { return m }
func (m *mockLoggerForPreferences) WithFields(fields map[string]interface{}) Logger  { return m }

// mockStoreForPreferences implements the Store interface for preferences testing
type mockStoreForPreferences struct {
	updateUserDarkModeFunc func(ctx context.Context, userID int64, darkMode bool) error
}

func (m *mockStoreForPreferences) UpdateUserDarkMode(ctx context.Context, userID int64, darkMode bool) error {
	if m.updateUserDarkModeFunc != nil {
		return m.updateUserDarkModeFunc(ctx, userID, darkMode)
	}
	return nil
}

// Stub methods for Store interface (not used in preferences tests)
func (m *mockStoreForPreferences) SaveChunk(ctx context.Context, source, text string, embedding []float32, tags []string, summary string) error {
	return nil
}
func (m *mockStoreForPreferences) Search(ctx context.Context, queryVec []float32, topK int) ([]Chunk, error) {
	return nil, nil
}
func (m *mockStoreForPreferences) SearchByUser(ctx context.Context, userID int64, queryVec []float32, topK int) ([]Chunk, error) {
	return nil, nil
}
func (m *mockStoreForPreferences) Library(ctx context.Context) ([]LibraryEntry, error) {
	return nil, nil
}
func (m *mockStoreForPreferences) LibraryByUser(ctx context.Context, userID int64) ([]LibraryEntry, error) {
	return nil, nil
}
func (m *mockStoreForPreferences) DeleteSource(ctx context.Context, source string) error {
	return nil
}
func (m *mockStoreForPreferences) SaveMessage(ctx context.Context, sessionID, role, content string) error {
	return nil
}
func (m *mockStoreForPreferences) SaveChatMessage(ctx context.Context, userID int64, sessionID, role, content, providerMode string) error {
	return nil
}
func (m *mockStoreForPreferences) GetSessionHistory(ctx context.Context, sessionID string) ([]ChatMessage, error) {
	return nil, nil
}
func (m *mockStoreForPreferences) GetSessionMessages(ctx context.Context, userID int64, sessionID string) ([]ChatMessage, error) {
	return nil, nil
}
func (m *mockStoreForPreferences) ListSessions(ctx context.Context) ([]Session, error) {
	return nil, nil
}
func (m *mockStoreForPreferences) GetUserSessions(ctx context.Context, userID int64) ([]Session, error) {
	return nil, nil
}
func (m *mockStoreForPreferences) GetSessionOwner(ctx context.Context, sessionID string) (int64, error) {
	return 0, nil
}
func (m *mockStoreForPreferences) AddAuditEntry(ctx context.Context, opType, details, userCtx string) error {
	return nil
}
func (m *mockStoreForPreferences) GetAuditLog(ctx context.Context, opType string, from, to time.Time) ([]AuditEntry, error) {
	return nil, nil
}
func (m *mockStoreForPreferences) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	return nil, nil
}
func (m *mockStoreForPreferences) GetUserByID(ctx context.Context, userID int64) (*User, error) {
	return nil, nil
}
func (m *mockStoreForPreferences) CreateUser(ctx context.Context, username, password, email string, isAdmin, mustChangePassword bool) (int64, error) {
	return 0, nil
}
func (m *mockStoreForPreferences) UpdatePassword(ctx context.Context, userID int64, newPassword string) error {
	return nil
}
func (m *mockStoreForPreferences) ListUsers(ctx context.Context) ([]User, error) {
	return nil, nil
}
func (m *mockStoreForPreferences) DeleteUser(ctx context.Context, userID int64) error {
	return nil
}
func (m *mockStoreForPreferences) GetUserSkills(ctx context.Context, userID int64) ([]Skill, error) {
	return nil, nil
}
func (m *mockStoreForPreferences) GetWatchedFoldersByUser(ctx context.Context, userID int64) ([]WatchedFolder, error) {
	return nil, nil
}

func TestHandleUpdatePreferences(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           map[string]interface{}
		userID         int64
		mockError      error
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:   "successful dark mode update",
			method: http.MethodPost,
			body: map[string]interface{}{
				"dark_mode": true,
			},
			userID:         1,
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Preferences updated successfully",
			},
		},
		{
			name:   "successful dark mode disable",
			method: http.MethodPost,
			body: map[string]interface{}{
				"dark_mode": false,
			},
			userID:         2,
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Preferences updated successfully",
			},
		},
		{
			name:           "method not allowed",
			method:         http.MethodGet,
			body:           nil,
			userID:         1,
			mockError:      nil,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   nil,
		},
		{
			name:           "invalid request body",
			method:         http.MethodPost,
			body:           nil,
			userID:         1,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock store
			mockStore := &mockStoreForPreferences{
				updateUserDarkModeFunc: func(ctx context.Context, userID int64, darkMode bool) error {
					if userID != tt.userID {
						t.Errorf("expected userID %d, got %d", tt.userID, userID)
					}
					return tt.mockError
				},
			}

			// Create server with mock store
			server := &Server{
				store:  mockStore,
				logger: &mockLoggerForPreferences{},
			}

			// Create request
			var req *http.Request
			if tt.body != nil {
				bodyBytes, _ := json.Marshal(tt.body)
				req = httptest.NewRequest(tt.method, "/api/user/preferences", bytes.NewReader(bodyBytes))
			} else if tt.method == http.MethodPost {
				req = httptest.NewRequest(tt.method, "/api/user/preferences", bytes.NewReader([]byte("invalid json")))
			} else {
				req = httptest.NewRequest(tt.method, "/api/user/preferences", nil)
			}

			// Add user ID to context (simulating auth middleware)
			ctx := context.WithValue(req.Context(), auth.UserIDKey, tt.userID)
			req = req.WithContext(ctx)

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call handler
			server.handleUpdatePreferences(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			// Check response body if expected
			if tt.expectedBody != nil {
				var response map[string]interface{}
				if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if response["success"] != tt.expectedBody["success"] {
					t.Errorf("expected success %v, got %v", tt.expectedBody["success"], response["success"])
				}

				if response["message"] != tt.expectedBody["message"] {
					t.Errorf("expected message %v, got %v", tt.expectedBody["message"], response["message"])
				}
			}
		})
	}
}
