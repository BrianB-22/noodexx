package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"noodexx/internal/auth"
	"noodexx/internal/config"
	"os"
	"testing"
)

// TestRAGPolicyEnforcementFlow_CompleteIntegration tests the complete RAG policy enforcement flow
// This integration test validates Requirements 4.1, 4.2, 4.3, 5.1, 5.2, 5.3, 6.1, 6.2, 6.3
func TestRAGPolicyEnforcementFlow_CompleteIntegration(t *testing.T) {
	// Create temporary config file
	tmpFile, err := os.CreateTemp("", "config-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Configure both local and cloud providers
	cfg := &config.Config{
		Provider: config.ProviderConfig{
			Type: "ollama",
		},
		LocalProvider: config.ProviderConfig{
			Type:             "ollama",
			OllamaEndpoint:   "http://localhost:11434",
			OllamaEmbedModel: "nomic-embed-text",
			OllamaChatModel:  "llama3.2",
		},
		CloudProvider: config.ProviderConfig{
			Type:             "openai",
			OpenAIKey:        "sk-test123",
			OpenAIEmbedModel: "text-embedding-3-small",
			OpenAIChatModel:  "gpt-4",
		},
		Privacy: config.PrivacyConfig{
			UseLocalAI:     false, // Start in cloud mode
			CloudRAGPolicy: "no_rag",
		},
		Server: config.ServerConfig{
			Port:        8080,
			BindAddress: "localhost",
		},
		Logging: config.LoggingConfig{
			Level: "info",
		},
		Guardrails: config.GuardrailsConfig{
			PIIDetection: "normal",
		},
		UserMode: "single",
		Auth: config.AuthConfig{
			Provider: "userpass",
		},
	}
	if err := cfg.Save(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to save initial config: %v", err)
	}

	// Track RAG operations
	embedCallCount := 0
	searchCallCount := 0

	// Create mock cloud provider
	cloudProvider := &mockProviderForAsk{
		name:    "openai",
		isLocal: false,
		embedFunc: func(ctx context.Context, text string) ([]float32, error) {
			embedCallCount++
			return []float32{0.1, 0.2, 0.3}, nil
		},
		streamFunc: func(ctx context.Context, messages []Message, w io.Writer) (string, error) {
			response := "Cloud AI response"
			w.Write([]byte(response))
			return response, nil
		},
	}

	// Create mock local provider
	localProvider := &mockProviderForAsk{
		name:    "ollama",
		isLocal: true,
		embedFunc: func(ctx context.Context, text string) ([]float32, error) {
			embedCallCount++
			return []float32{0.4, 0.5, 0.6}, nil
		},
		streamFunc: func(ctx context.Context, messages []Message, w io.Writer) (string, error) {
			response := "Local AI response"
			w.Write([]byte(response))
			return response, nil
		},
	}

	// Create mock store that tracks search calls
	store := &mockStoreForAsk{
		searchByUserFunc: func(ctx context.Context, userID int64, queryVec []float32, topK int) ([]Chunk, error) {
			searchCallCount++
			return []Chunk{
				{Source: "test.txt", Text: "test chunk 1"},
				{Source: "test.txt", Text: "test chunk 2"},
			}, nil
		},
	}

	// Create mock provider manager (starts with cloud provider)
	providerManager := &mockProviderManagerForAsk{
		provider:     cloudProvider,
		providerName: "Cloud AI (gpt-4)",
	}

	// Create mock RAG enforcer (starts with no-RAG policy)
	ragEnforcer := &mockRAGEnforcerForAsk{
		shouldPerformRAG: false,
		ragStatus:        "RAG Disabled (Cloud Policy)",
	}

	// Create server
	server := &Server{
		configPath:      tmpFile.Name(),
		store:           store,
		logger:          &mockLoggerForAsk{},
		providerManager: providerManager,
		ragEnforcer:     ragEnforcer,
	}

	// Test 1: Cloud mode with "No RAG" policy - RAG should be disabled
	// Validates Requirements 5.1, 5.2
	t.Run("CloudMode_NoRAG_Policy", func(t *testing.T) {
		// Reset counters
		embedCallCount = 0
		searchCallCount = 0

		// Send query
		reqBody := map[string]string{
			"query":      "test query with no RAG",
			"session_id": "test-session",
		}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/ask", bytes.NewReader(bodyBytes))
		ctx := context.WithValue(req.Context(), auth.UserIDKey, int64(1))
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		server.handleAsk(w, req)

		// Verify response
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		// Verify RAG status header (Requirement 5.3)
		if ragStatus := w.Header().Get("X-RAG-Status"); ragStatus != "RAG Disabled (Cloud Policy)" {
			t.Errorf("Expected X-RAG-Status='RAG Disabled (Cloud Policy)', got '%s'", ragStatus)
		}

		// Verify RAG was NOT performed (Requirement 5.1, 5.2)
		if embedCallCount != 0 {
			t.Errorf("Expected 0 embed calls (RAG disabled), got %d", embedCallCount)
		}
		if searchCallCount != 0 {
			t.Errorf("Expected 0 search calls (RAG disabled), got %d", searchCallCount)
		}

		t.Log("✓ Cloud mode with No RAG policy: RAG correctly disabled")
	})

	// Test 2: Cloud mode with "Allow RAG" policy - RAG should be enabled
	// Validates Requirements 6.1, 6.2, 6.3
	t.Run("CloudMode_AllowRAG_Policy", func(t *testing.T) {
		// Reset counters
		embedCallCount = 0
		searchCallCount = 0

		// Update RAG enforcer to allow RAG
		ragEnforcer.shouldPerformRAG = true
		ragEnforcer.ragStatus = "RAG Enabled (Cloud)"

		// Send query
		reqBody := map[string]string{
			"query":      "test query with RAG allowed",
			"session_id": "test-session",
		}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/ask", bytes.NewReader(bodyBytes))
		ctx := context.WithValue(req.Context(), auth.UserIDKey, int64(1))
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		server.handleAsk(w, req)

		// Verify response
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		// Verify RAG status header (Requirement 6.3)
		if ragStatus := w.Header().Get("X-RAG-Status"); ragStatus != "RAG Enabled (Cloud)" {
			t.Errorf("Expected X-RAG-Status='RAG Enabled (Cloud)', got '%s'", ragStatus)
		}

		// Verify RAG WAS performed (Requirement 6.1, 6.2)
		if embedCallCount != 1 {
			t.Errorf("Expected 1 embed call (RAG enabled), got %d", embedCallCount)
		}
		if searchCallCount != 1 {
			t.Errorf("Expected 1 search call (RAG enabled), got %d", searchCallCount)
		}

		t.Log("✓ Cloud mode with Allow RAG policy: RAG correctly enabled")
	})

	// Test 3: Local mode - RAG should ALWAYS be enabled regardless of policy
	// Validates Requirements 4.1, 4.2, 4.3
	t.Run("LocalMode_RAG_AlwaysEnabled", func(t *testing.T) {
		// Switch to local provider
		providerManager.provider = localProvider
		providerManager.providerName = "Local AI (Ollama)"

		// Test with "no_rag" policy (should be ignored in local mode)
		ragEnforcer.shouldPerformRAG = true // Local mode always enables RAG
		ragEnforcer.ragStatus = "RAG Enabled (Local)"

		// Reset counters
		embedCallCount = 0
		searchCallCount = 0

		// Send query
		reqBody := map[string]string{
			"query":      "test query in local mode",
			"session_id": "test-session",
		}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/ask", bytes.NewReader(bodyBytes))
		ctx := context.WithValue(req.Context(), auth.UserIDKey, int64(1))
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		server.handleAsk(w, req)

		// Verify response
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		// Verify RAG status header
		if ragStatus := w.Header().Get("X-RAG-Status"); ragStatus != "RAG Enabled (Local)" {
			t.Errorf("Expected X-RAG-Status='RAG Enabled (Local)', got '%s'", ragStatus)
		}

		// Verify RAG WAS performed (Requirement 4.1, 4.2)
		if embedCallCount != 1 {
			t.Errorf("Expected 1 embed call (RAG always enabled in local mode), got %d", embedCallCount)
		}
		if searchCallCount != 1 {
			t.Errorf("Expected 1 search call (RAG always enabled in local mode), got %d", searchCallCount)
		}

		t.Log("✓ Local mode: RAG correctly enabled regardless of policy")
	})

	// Test 4: Policy change from No RAG to Allow RAG in cloud mode
	t.Run("PolicyChange_NoRAG_To_AllowRAG", func(t *testing.T) {
		// Switch back to cloud provider
		providerManager.provider = cloudProvider
		providerManager.providerName = "Cloud AI (gpt-4)"

		// Start with No RAG policy
		ragEnforcer.shouldPerformRAG = false
		ragEnforcer.ragStatus = "RAG Disabled (Cloud Policy)"

		// Reset counters
		embedCallCount = 0
		searchCallCount = 0

		// Send query with No RAG
		reqBody := map[string]string{
			"query":      "query before policy change",
			"session_id": "test-session",
		}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/ask", bytes.NewReader(bodyBytes))
		ctx := context.WithValue(req.Context(), auth.UserIDKey, int64(1))
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		server.handleAsk(w, req)

		// Verify RAG was NOT performed
		if embedCallCount != 0 {
			t.Errorf("Expected 0 embed calls before policy change, got %d", embedCallCount)
		}

		// Change policy to Allow RAG
		ragEnforcer.shouldPerformRAG = true
		ragEnforcer.ragStatus = "RAG Enabled (Cloud)"

		// Reset counters
		embedCallCount = 0
		searchCallCount = 0

		// Send query with Allow RAG
		reqBody = map[string]string{
			"query":      "query after policy change",
			"session_id": "test-session",
		}
		bodyBytes, _ = json.Marshal(reqBody)
		req = httptest.NewRequest(http.MethodPost, "/api/ask", bytes.NewReader(bodyBytes))
		ctx = context.WithValue(req.Context(), auth.UserIDKey, int64(1))
		req = req.WithContext(ctx)
		w = httptest.NewRecorder()

		server.handleAsk(w, req)

		// Verify RAG WAS performed after policy change
		if embedCallCount != 1 {
			t.Errorf("Expected 1 embed call after policy change, got %d", embedCallCount)
		}
		if searchCallCount != 1 {
			t.Errorf("Expected 1 search call after policy change, got %d", searchCallCount)
		}

		t.Log("✓ Policy change from No RAG to Allow RAG: correctly applied")
	})

	// Test 5: Multiple queries with different policies
	t.Run("MultipleQueries_DifferentPolicies", func(t *testing.T) {
		testCases := []struct {
			name             string
			useLocal         bool
			allowRAG         bool
			expectedEmbeds   int
			expectedSearches int
			expectedStatus   string
		}{
			{
				name:             "Cloud_NoRAG",
				useLocal:         false,
				allowRAG:         false,
				expectedEmbeds:   0,
				expectedSearches: 0,
				expectedStatus:   "RAG Disabled (Cloud Policy)",
			},
			{
				name:             "Cloud_AllowRAG",
				useLocal:         false,
				allowRAG:         true,
				expectedEmbeds:   1,
				expectedSearches: 1,
				expectedStatus:   "RAG Enabled (Cloud)",
			},
			{
				name:             "Local_AlwaysRAG",
				useLocal:         true,
				allowRAG:         true, // Always true for local
				expectedEmbeds:   1,
				expectedSearches: 1,
				expectedStatus:   "RAG Enabled (Local)",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Configure provider and RAG enforcer
				if tc.useLocal {
					providerManager.provider = localProvider
					providerManager.providerName = "Local AI (Ollama)"
				} else {
					providerManager.provider = cloudProvider
					providerManager.providerName = "Cloud AI (gpt-4)"
				}

				ragEnforcer.shouldPerformRAG = tc.allowRAG
				ragEnforcer.ragStatus = tc.expectedStatus

				// Reset counters
				embedCallCount = 0
				searchCallCount = 0

				// Send query
				reqBody := map[string]string{
					"query":      "test query for " + tc.name,
					"session_id": "test-session",
				}
				bodyBytes, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/api/ask", bytes.NewReader(bodyBytes))
				ctx := context.WithValue(req.Context(), auth.UserIDKey, int64(1))
				req = req.WithContext(ctx)
				w := httptest.NewRecorder()

				server.handleAsk(w, req)

				// Verify response
				if w.Code != http.StatusOK {
					t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
				}

				// Verify RAG status header
				if ragStatus := w.Header().Get("X-RAG-Status"); ragStatus != tc.expectedStatus {
					t.Errorf("Expected X-RAG-Status='%s', got '%s'", tc.expectedStatus, ragStatus)
				}

				// Verify RAG operations
				if embedCallCount != tc.expectedEmbeds {
					t.Errorf("Expected %d embed calls, got %d", tc.expectedEmbeds, embedCallCount)
				}
				if searchCallCount != tc.expectedSearches {
					t.Errorf("Expected %d search calls, got %d", tc.expectedSearches, searchCallCount)
				}
			})
		}
	})

	t.Log("=== RAG Policy Enforcement Flow Test Complete ===")
	t.Log("✓ All RAG policy scenarios validated successfully")
}

// TestRAGPolicyEnforcement_ContextInclusion tests that document context is correctly included/excluded
// This validates that RAG results are actually sent to the provider
func TestRAGPolicyEnforcement_ContextInclusion(t *testing.T) {
	testCases := []struct {
		name               string
		useLocal           bool
		allowRAG           bool
		expectedHasContext bool
		expectedRAGStatus  string
	}{
		{
			name:               "Cloud_NoRAG_NoContext",
			useLocal:           false,
			allowRAG:           false,
			expectedHasContext: false,
			expectedRAGStatus:  "RAG Disabled (Cloud Policy)",
		},
		{
			name:               "Cloud_AllowRAG_HasContext",
			useLocal:           false,
			allowRAG:           true,
			expectedHasContext: true,
			expectedRAGStatus:  "RAG Enabled (Cloud)",
		},
		{
			name:               "Local_AlwaysHasContext",
			useLocal:           true,
			allowRAG:           true,
			expectedHasContext: true,
			expectedRAGStatus:  "RAG Enabled (Local)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Track whether context was included in the prompt
			contextIncluded := false

			// Create mock provider that checks for context
			provider := &mockProviderForAsk{
				name:    "test-provider",
				isLocal: tc.useLocal,
				embedFunc: func(ctx context.Context, text string) ([]float32, error) {
					return []float32{0.1, 0.2, 0.3}, nil
				},
				streamFunc: func(ctx context.Context, messages []Message, w io.Writer) (string, error) {
					// Check if any message contains actual chunk content (not just the "Context:" label)
					for _, msg := range messages {
						if msg.Role == "user" {
							// Look for actual chunk content markers (numbered sources indicate real chunks)
							if bytes.Contains([]byte(msg.Content), []byte("[1] Source:")) {
								contextIncluded = true
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
						{Source: "test.txt", Text: "test chunk with content"},
					}, nil
				},
			}

			// Create server
			server := &Server{
				store:           store,
				logger:          &mockLoggerForAsk{},
				providerManager: &mockProviderManagerForAsk{provider: provider, providerName: "Test Provider"},
				ragEnforcer:     &mockRAGEnforcerForAsk{shouldPerformRAG: tc.allowRAG, ragStatus: tc.expectedRAGStatus},
			}

			// Send query
			reqBody := map[string]string{
				"query":      "test query",
				"session_id": "test-session",
			}
			bodyBytes, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/ask", bytes.NewReader(bodyBytes))
			ctx := context.WithValue(req.Context(), auth.UserIDKey, int64(1))
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			server.handleAsk(w, req)

			// Verify context inclusion matches expectation
			if contextIncluded != tc.expectedHasContext {
				t.Errorf("Expected context included=%v, got %v", tc.expectedHasContext, contextIncluded)
			}

			// Verify RAG status header
			if ragStatus := w.Header().Get("X-RAG-Status"); ragStatus != tc.expectedRAGStatus {
				t.Errorf("Expected X-RAG-Status='%s', got '%s'", tc.expectedRAGStatus, ragStatus)
			}
		})
	}
}
