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

// TestHandleRunSkill_OwnershipVerification tests that skill execution verifies ownership
func TestHandleRunSkill_OwnershipVerification(t *testing.T) {
	// Create mock store
	mockStore := &mockStore{}

	// Create mock skills loader that returns a skill owned by user 1
	mockSkillsLoader := &mockSkillsLoader{
		skills: []*Skill{
			{
				UserID:      1, // Owned by user 1
				Name:        "test-skill",
				Version:     "1.0.0",
				Description: "Test skill",
				Executable:  "/bin/echo",
				Triggers: []SkillTrigger{
					{Type: "manual"},
				},
				Timeout:     30 * time.Second,
				RequiresNet: false,
				Path:        "/test/path",
			},
		},
	}

	// Create mock skills executor
	mockSkillsExecutor := &mockSkillsExecutor{}

	// Create mock logger
	mockLogger := &mockLogger{}

	// Create mock auth provider
	mockAuthProvider := &mockAuthProvider{}

	// Create server
	server := &Server{
		store:          mockStore,
		skillsLoader:   mockSkillsLoader,
		skillsExecutor: mockSkillsExecutor,
		logger:         mockLogger,
		authProvider:   mockAuthProvider,
	}

	tests := []struct {
		name           string
		userID         int64
		skillName      string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "authorized user can execute their skill",
			userID:         1,
			skillName:      "test-skill",
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name:           "unauthorized user cannot execute another user's skill",
			userID:         2, // Different user
			skillName:      "test-skill",
			expectedStatus: http.StatusForbidden,
			expectedError:  "Unauthorized: skill does not belong to current user",
		},
		{
			name:           "skill not found returns 404",
			userID:         1,
			skillName:      "nonexistent-skill",
			expectedStatus: http.StatusNotFound,
			expectedError:  "Skill not found: nonexistent-skill",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request body
			reqBody := map[string]interface{}{
				"skill_name": tt.skillName,
				"query":      "test query",
				"context":    map[string]interface{}{},
			}
			bodyBytes, _ := json.Marshal(reqBody)

			// Create request
			req := httptest.NewRequest(http.MethodPost, "/api/skills/run", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			// Add user ID to context
			ctx := context.WithValue(req.Context(), auth.UserIDKey, tt.userID)
			req = req.WithContext(ctx)

			// Create response recorder
			w := httptest.NewRecorder()

			// Call handler
			server.handleRunSkill(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check error message if expected
			if tt.expectedError != "" {
				body := w.Body.String()
				if body != tt.expectedError+"\n" {
					t.Errorf("Expected error %q, got %q", tt.expectedError, body)
				}
			}

			// If success expected, verify execution was called
			if tt.expectedStatus == http.StatusOK {
				if !mockSkillsExecutor.executeCalled {
					t.Error("Expected Execute to be called, but it wasn't")
				}
			}
		})
	}
}

// mockSkillsLoader implements SkillsLoader interface for testing
type mockSkillsLoader struct {
	skills []*Skill
}

func (m *mockSkillsLoader) LoadAll() ([]*Skill, error) {
	return m.skills, nil
}

func (m *mockSkillsLoader) LoadForUser(ctx context.Context, userID int64) ([]*Skill, error) {
	// Return all skills (filtering is done by the handler)
	return m.skills, nil
}

// mockSkillsExecutor implements SkillsExecutor interface for testing
type mockSkillsExecutor struct {
	executeCalled bool
}

func (m *mockSkillsExecutor) Execute(ctx context.Context, skill *Skill, input SkillInput) (*SkillOutput, error) {
	m.executeCalled = true
	return &SkillOutput{
		Result:   "test result",
		Error:    "",
		Metadata: map[string]interface{}{},
	}, nil
}
