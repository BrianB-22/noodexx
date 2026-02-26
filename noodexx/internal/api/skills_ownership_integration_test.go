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

// TestSkillOwnershipIntegration tests the complete skill ownership verification flow
func TestSkillOwnershipIntegration(t *testing.T) {
	// Create mock store
	mockStore := &mockStore{}

	// Create mock skills loader with skills owned by different users
	mockSkillsLoader := &mockSkillsLoader{
		skills: []*Skill{
			{
				UserID:      1,
				Name:        "user1-skill",
				Version:     "1.0.0",
				Description: "User 1's skill",
				Executable:  "/bin/echo",
				Triggers: []SkillTrigger{
					{Type: "manual"},
				},
				Timeout:     30 * time.Second,
				RequiresNet: false,
				Path:        "/test/user1-skill",
			},
			{
				UserID:      2,
				Name:        "user2-skill",
				Version:     "1.0.0",
				Description: "User 2's skill",
				Executable:  "/bin/echo",
				Triggers: []SkillTrigger{
					{Type: "manual"},
				},
				Timeout:     30 * time.Second,
				RequiresNet: false,
				Path:        "/test/user2-skill",
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

	t.Run("user can execute their own skill", func(t *testing.T) {
		// Reset executor state
		mockSkillsExecutor.executeCalled = false

		// Create request for user 1 to execute their skill
		reqBody := map[string]interface{}{
			"skill_name": "user1-skill",
			"query":      "test query",
			"context":    map[string]interface{}{},
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/skills/run", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), auth.UserIDKey, int64(1))
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		server.handleRunSkill(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		if !mockSkillsExecutor.executeCalled {
			t.Error("Expected Execute to be called for authorized user")
		}
	})

	t.Run("user cannot execute another user's skill", func(t *testing.T) {
		// Reset executor state
		mockSkillsExecutor.executeCalled = false

		// Create request for user 1 to execute user 2's skill
		reqBody := map[string]interface{}{
			"skill_name": "user2-skill",
			"query":      "test query",
			"context":    map[string]interface{}{},
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/skills/run", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), auth.UserIDKey, int64(1))
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		server.handleRunSkill(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d. Body: %s", w.Code, w.Body.String())
		}

		if mockSkillsExecutor.executeCalled {
			t.Error("Execute should not be called for unauthorized user")
		}

		expectedError := "Unauthorized: skill does not belong to current user\n"
		if w.Body.String() != expectedError {
			t.Errorf("Expected error %q, got %q", expectedError, w.Body.String())
		}
	})

	t.Run("user 2 can execute their own skill", func(t *testing.T) {
		// Reset executor state
		mockSkillsExecutor.executeCalled = false

		// Create request for user 2 to execute their skill
		reqBody := map[string]interface{}{
			"skill_name": "user2-skill",
			"query":      "test query",
			"context":    map[string]interface{}{},
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/skills/run", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), auth.UserIDKey, int64(2))
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		server.handleRunSkill(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		if !mockSkillsExecutor.executeCalled {
			t.Error("Expected Execute to be called for authorized user")
		}
	})

	t.Run("user 2 cannot execute user 1's skill", func(t *testing.T) {
		// Reset executor state
		mockSkillsExecutor.executeCalled = false

		// Create request for user 2 to execute user 1's skill
		reqBody := map[string]interface{}{
			"skill_name": "user1-skill",
			"query":      "test query",
			"context":    map[string]interface{}{},
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/skills/run", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), auth.UserIDKey, int64(2))
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		server.handleRunSkill(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d. Body: %s", w.Code, w.Body.String())
		}

		if mockSkillsExecutor.executeCalled {
			t.Error("Execute should not be called for unauthorized user")
		}
	})
}
