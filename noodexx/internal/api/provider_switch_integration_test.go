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
	"time"
)

// TestProviderSwitchFlow_CompleteIntegration tests the complete provider switch flow
// This integration test validates Requirements 2.1, 2.2, 2.3, 7.1, 7.2, 7.3
func TestProviderSwitchFlow_CompleteIntegration(t *testing.T) {
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
			DefaultToLocal:     true, // Start in local mode
			CloudRAGPolicy: "allow_rag",
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

	// Track provider switches and query routing
	localQueryCount := 0
	cloudQueryCount := 0

	// Create mock local provider
	localProvider := &mockProviderForAsk{
		name:    "ollama",
		isLocal: true,
		embedFunc: func(ctx context.Context, text string) ([]float32, error) {
			return []float32{0.1, 0.2, 0.3}, nil
		},
		streamFunc: func(ctx context.Context, messages []Message, w io.Writer) (string, error) {
			localQueryCount++
			response := "Local AI response"
			w.Write([]byte(response))
			return response, nil
		},
	}

	// Create mock cloud provider
	cloudProvider := &mockProviderForAsk{
		name:    "openai",
		isLocal: false,
		embedFunc: func(ctx context.Context, text string) ([]float32, error) {
			return []float32{0.4, 0.5, 0.6}, nil
		},
		streamFunc: func(ctx context.Context, messages []Message, w io.Writer) (string, error) {
			cloudQueryCount++
			response := "Cloud AI response"
			w.Write([]byte(response))
			return response, nil
		},
	}

	// Create mock provider manager that switches providers
	providerManager := &mockProviderManagerForAsk{
		provider:     localProvider,
		providerName: "Local AI (Ollama)",
	}

	// Create mock RAG enforcer
	ragEnforcer := &mockRAGEnforcerForAsk{
		shouldPerformRAG: true,
		ragStatus:        "RAG Enabled (Local)",
	}

	// Create mock store
	store := &mockStoreForAsk{
		searchByUserFunc: func(ctx context.Context, userID int64, queryVec []float32, topK int) ([]Chunk, error) {
			return []Chunk{
				{Source: "test.txt", Text: "test chunk"},
			}, nil
		},
	}

	// Create server
	server := &Server{
		configPath:      tmpFile.Name(),
		store:           store,
		logger:          &mockLoggerForAsk{},
		providerManager: providerManager,
		ragEnforcer:     ragEnforcer,
	}

	// Test 1: Initial state - Local provider
	t.Run("Initial_LocalProvider", func(t *testing.T) {
		// Send query
		reqBody := map[string]string{
			"query":      "test query 1",
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

		// Verify provider name header
		if providerName := w.Header().Get("X-Provider-Name"); providerName != "Local AI (Ollama)" {
			t.Errorf("Expected X-Provider-Name='Local AI (Ollama)', got '%s'", providerName)
		}

		// Verify RAG status header
		if ragStatus := w.Header().Get("X-RAG-Status"); ragStatus != "RAG Enabled (Local)" {
			t.Errorf("Expected X-RAG-Status='RAG Enabled (Local)', got '%s'", ragStatus)
		}

		// Verify query routed to local provider
		if localQueryCount != 1 {
			t.Errorf("Expected 1 local query, got %d", localQueryCount)
		}
		if cloudQueryCount != 0 {
			t.Errorf("Expected 0 cloud queries, got %d", cloudQueryCount)
		}
	})

	// Test 2: Toggle to cloud provider
	t.Run("Toggle_ToCloud", func(t *testing.T) {
		// Switch provider manager to cloud
		providerManager.provider = cloudProvider
		providerManager.providerName = "Cloud AI (gpt-4)"
		ragEnforcer.ragStatus = "RAG Enabled (Cloud)"

		// Toggle to cloud mode
		toggleReq := map[string]string{"mode": "cloud"}
		toggleBytes, _ := json.Marshal(toggleReq)
		req := httptest.NewRequest(http.MethodPost, "/api/privacy-toggle", bytes.NewReader(toggleBytes))
		w := httptest.NewRecorder()

		startTime := time.Now()
		server.handlePrivacyToggle(w, req)
		elapsed := time.Since(startTime)

		// Verify toggle completes within 1 second (Requirement 2.3)
		if elapsed > time.Second {
			t.Errorf("Toggle took %v, expected < 1 second", elapsed)
		}

		// Verify response
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var resp map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Verify provider name in response
		if provider, ok := resp["provider"].(string); !ok || provider != "Cloud AI (gpt-4)" {
			t.Errorf("Expected provider='Cloud AI (gpt-4)', got %v", resp["provider"])
		}

		// Verify RAG status in response
		if ragStatus, ok := resp["rag_status"].(string); !ok || ragStatus != "RAG Enabled (Cloud)" {
			t.Errorf("Expected rag_status='RAG Enabled (Cloud)', got %v", resp["rag_status"])
		}

		// Verify config was updated
		loadedCfg, err := config.Load(tmpFile.Name())
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}
		if loadedCfg.Privacy.DefaultToLocal {
			t.Error("Expected DefaultToLocal=false after toggle to cloud")
		}
	})

	// Test 3: Send query to cloud provider
	t.Run("Query_CloudProvider", func(t *testing.T) {
		// Send query
		reqBody := map[string]string{
			"query":      "test query 2",
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

		// Verify provider name header
		if providerName := w.Header().Get("X-Provider-Name"); providerName != "Cloud AI (gpt-4)" {
			t.Errorf("Expected X-Provider-Name='Cloud AI (gpt-4)', got '%s'", providerName)
		}

		// Verify RAG status header
		if ragStatus := w.Header().Get("X-RAG-Status"); ragStatus != "RAG Enabled (Cloud)" {
			t.Errorf("Expected X-RAG-Status='RAG Enabled (Cloud)', got '%s'", ragStatus)
		}

		// Verify query routed to cloud provider
		if localQueryCount != 1 {
			t.Errorf("Expected 1 local query (unchanged), got %d", localQueryCount)
		}
		if cloudQueryCount != 1 {
			t.Errorf("Expected 1 cloud query, got %d", cloudQueryCount)
		}
	})

	// Test 4: Toggle back to local provider
	t.Run("Toggle_BackToLocal", func(t *testing.T) {
		// Switch provider manager back to local
		providerManager.provider = localProvider
		providerManager.providerName = "Local AI (Ollama)"
		ragEnforcer.ragStatus = "RAG Enabled (Local)"

		// Toggle to local mode
		toggleReq := map[string]string{"mode": "local"}
		toggleBytes, _ := json.Marshal(toggleReq)
		req := httptest.NewRequest(http.MethodPost, "/api/privacy-toggle", bytes.NewReader(toggleBytes))
		w := httptest.NewRecorder()

		startTime := time.Now()
		server.handlePrivacyToggle(w, req)
		elapsed := time.Since(startTime)

		// Verify toggle completes within 1 second (Requirement 2.3)
		if elapsed > time.Second {
			t.Errorf("Toggle took %v, expected < 1 second", elapsed)
		}

		// Verify response
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var resp map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Verify provider name in response
		if provider, ok := resp["provider"].(string); !ok || provider != "Local AI (Ollama)" {
			t.Errorf("Expected provider='Local AI (Ollama)', got %v", resp["provider"])
		}

		// Verify config was updated
		loadedCfg, err := config.Load(tmpFile.Name())
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}
		if !loadedCfg.Privacy.DefaultToLocal {
			t.Error("Expected DefaultToLocal=true after toggle to local")
		}
	})

	// Test 5: Send query to local provider again
	t.Run("Query_LocalProvider_Again", func(t *testing.T) {
		// Send query
		reqBody := map[string]string{
			"query":      "test query 3",
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

		// Verify provider name header
		if providerName := w.Header().Get("X-Provider-Name"); providerName != "Local AI (Ollama)" {
			t.Errorf("Expected X-Provider-Name='Local AI (Ollama)', got '%s'", providerName)
		}

		// Verify query routed to local provider
		if localQueryCount != 2 {
			t.Errorf("Expected 2 local queries, got %d", localQueryCount)
		}
		if cloudQueryCount != 1 {
			t.Errorf("Expected 1 cloud query (unchanged), got %d", cloudQueryCount)
		}
	})

	// Test 6: Multiple rapid toggles
	t.Run("Multiple_RapidToggles", func(t *testing.T) {
		toggleCount := 5
		for i := 0; i < toggleCount; i++ {
			mode := "cloud"
			expectedProvider := "Cloud AI (gpt-4)"
			if i%2 == 0 {
				mode = "local"
				expectedProvider = "Local AI (Ollama)"
			}

			// Update mock provider manager
			if mode == "local" {
				providerManager.provider = localProvider
				providerManager.providerName = "Local AI (Ollama)"
			} else {
				providerManager.provider = cloudProvider
				providerManager.providerName = "Cloud AI (gpt-4)"
			}

			// Toggle
			toggleReq := map[string]string{"mode": mode}
			toggleBytes, _ := json.Marshal(toggleReq)
			req := httptest.NewRequest(http.MethodPost, "/api/privacy-toggle", bytes.NewReader(toggleBytes))
			w := httptest.NewRecorder()

			startTime := time.Now()
			server.handlePrivacyToggle(w, req)
			elapsed := time.Since(startTime)

			// Verify toggle completes within 1 second
			if elapsed > time.Second {
				t.Errorf("Toggle %d took %v, expected < 1 second", i+1, elapsed)
			}

			// Verify response
			if w.Code != http.StatusOK {
				t.Errorf("Toggle %d: Expected status %d, got %d", i+1, http.StatusOK, w.Code)
			}

			var resp map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("Toggle %d: Failed to decode response: %v", i+1, err)
			}

			// Verify provider name
			if provider, ok := resp["provider"].(string); !ok || provider != expectedProvider {
				t.Errorf("Toggle %d: Expected provider='%s', got %v", i+1, expectedProvider, resp["provider"])
			}
		}
	})

	// Test 7: Verify status indicator updates within 500ms (Requirement 7.3)
	t.Run("StatusIndicator_UpdateSpeed", func(t *testing.T) {
		// Toggle to cloud
		providerManager.provider = cloudProvider
		providerManager.providerName = "Cloud AI (gpt-4)"
		ragEnforcer.ragStatus = "RAG Enabled (Cloud)"

		toggleReq := map[string]string{"mode": "cloud"}
		toggleBytes, _ := json.Marshal(toggleReq)
		req := httptest.NewRequest(http.MethodPost, "/api/privacy-toggle", bytes.NewReader(toggleBytes))
		w := httptest.NewRecorder()

		startTime := time.Now()
		server.handlePrivacyToggle(w, req)
		elapsed := time.Since(startTime)

		// Verify response includes status information within 500ms
		if elapsed > 500*time.Millisecond {
			t.Errorf("Status update took %v, expected < 500ms (Requirement 7.3)", elapsed)
		}

		// Verify response includes provider and RAG status
		var resp map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if _, ok := resp["provider"]; !ok {
			t.Error("Response missing 'provider' field for status indicator")
		}

		if _, ok := resp["rag_status"]; !ok {
			t.Error("Response missing 'rag_status' field for status indicator")
		}
	})

	// Final verification: Check total query routing
	t.Log("=== Final Query Routing Summary ===")
	t.Logf("Local queries: %d", localQueryCount)
	t.Logf("Cloud queries: %d", cloudQueryCount)
	t.Logf("Total queries: %d", localQueryCount+cloudQueryCount)

	// Verify queries were routed correctly
	if localQueryCount < 2 {
		t.Errorf("Expected at least 2 local queries, got %d", localQueryCount)
	}
	if cloudQueryCount < 1 {
		t.Errorf("Expected at least 1 cloud query, got %d", cloudQueryCount)
	}
}

// TestProviderSwitchFlow_StatusIndicatorHeaders tests that status indicators are correctly set in response headers
// This validates Requirements 7.1, 7.2, 7.3
func TestProviderSwitchFlow_StatusIndicatorHeaders(t *testing.T) {
	testCases := []struct {
		name                string
		providerName        string
		ragStatus           string
		isLocal             bool
		expectedProviderHdr string
		expectedRAGHdr      string
	}{
		{
			name:                "Local_Ollama",
			providerName:        "Local AI (Ollama)",
			ragStatus:           "RAG Enabled (Local)",
			isLocal:             true,
			expectedProviderHdr: "Local AI (Ollama)",
			expectedRAGHdr:      "RAG Enabled (Local)",
		},
		{
			name:                "Cloud_OpenAI",
			providerName:        "Cloud AI (gpt-4)",
			ragStatus:           "RAG Enabled (Cloud)",
			isLocal:             false,
			expectedProviderHdr: "Cloud AI (gpt-4)",
			expectedRAGHdr:      "RAG Enabled (Cloud)",
		},
		{
			name:                "Cloud_NoRAG",
			providerName:        "Cloud AI (gpt-4)",
			ragStatus:           "RAG Disabled (Cloud Policy)",
			isLocal:             false,
			expectedProviderHdr: "Cloud AI (gpt-4)",
			expectedRAGHdr:      "RAG Disabled (Cloud Policy)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock provider
			provider := &mockProviderForAsk{
				name:    tc.providerName,
				isLocal: tc.isLocal,
				embedFunc: func(ctx context.Context, text string) ([]float32, error) {
					return []float32{0.1, 0.2, 0.3}, nil
				},
				streamFunc: func(ctx context.Context, messages []Message, w io.Writer) (string, error) {
					response := "test response"
					w.Write([]byte(response))
					return response, nil
				},
			}

			// Create server
			server := &Server{
				store:           &mockStoreForAsk{},
				logger:          &mockLoggerForAsk{},
				providerManager: &mockProviderManagerForAsk{provider: provider, providerName: tc.providerName},
				ragEnforcer:     &mockRAGEnforcerForAsk{shouldPerformRAG: tc.ragStatus != "RAG Disabled (Cloud Policy)", ragStatus: tc.ragStatus},
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

			// Verify headers
			if providerHdr := w.Header().Get("X-Provider-Name"); providerHdr != tc.expectedProviderHdr {
				t.Errorf("Expected X-Provider-Name='%s', got '%s'", tc.expectedProviderHdr, providerHdr)
			}

			if ragHdr := w.Header().Get("X-RAG-Status"); ragHdr != tc.expectedRAGHdr {
				t.Errorf("Expected X-RAG-Status='%s', got '%s'", tc.expectedRAGHdr, ragHdr)
			}
		})
	}
}
