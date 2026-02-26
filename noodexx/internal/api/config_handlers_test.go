package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"noodexx/internal/config"
	"os"
	"strings"
	"testing"
)

// TestHandleConfig_ValidationErrors tests that validation errors are returned correctly
func TestHandleConfig_ValidationErrors(t *testing.T) {
	tests := []struct {
		name           string
		formData       url.Values
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Invalid local provider type",
			formData: url.Values{
				"local_provider_type":      {"openai"}, // Invalid for local
				"local_ollama_endpoint":    {"http://localhost:11434"},
				"local_ollama_embed_model": {"nomic-embed-text"},
				"local_ollama_chat_model":  {"llama3.2"},
				"cloud_rag_policy":         {"no_rag"},
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Local provider validation failed: local provider must be Ollama",
		},
		{
			name: "Invalid Ollama endpoint (not localhost)",
			formData: url.Values{
				"local_provider_type":      {"ollama"},
				"local_ollama_endpoint":    {"http://example.com:11434"},
				"local_ollama_embed_model": {"nomic-embed-text"},
				"local_ollama_chat_model":  {"llama3.2"},
				"cloud_rag_policy":         {"no_rag"},
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Local provider validation failed: local provider must use localhost endpoint",
		},
		{
			name: "Missing OpenAI API key",
			formData: url.Values{
				"cloud_provider_type":      {"openai"},
				"cloud_openai_embed_model": {"text-embedding-3-small"},
				"cloud_openai_chat_model":  {"gpt-4"},
				"cloud_rag_policy":         {"no_rag"},
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Cloud provider validation failed: OpenAI API key is required",
		},
		{
			name: "Missing OpenAI models",
			formData: url.Values{
				"cloud_provider_type": {"openai"},
				"cloud_openai_key":    {"sk-test123"},
				"cloud_rag_policy":    {"no_rag"},
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Cloud provider validation failed: OpenAI models are required",
		},
		{
			name: "Invalid cloud provider type",
			formData: url.Values{
				"cloud_provider_type": {"invalid"},
				"cloud_rag_policy":    {"no_rag"},
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Cloud provider validation failed: invalid cloud provider type: invalid",
		},
		{
			name: "Invalid RAG policy",
			formData: url.Values{
				"cloud_rag_policy": {"invalid_policy"},
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "RAG policy validation failed: invalid RAG policy: invalid_policy",
		},
		{
			name: "Valid local provider configuration",
			formData: url.Values{
				"local_provider_type":      {"ollama"},
				"local_ollama_endpoint":    {"http://localhost:11434"},
				"local_ollama_embed_model": {"nomic-embed-text"},
				"local_ollama_chat_model":  {"llama3.2"},
				"cloud_rag_policy":         {"no_rag"},
			},
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name: "Valid cloud provider configuration",
			formData: url.Values{
				"cloud_provider_type":      {"openai"},
				"cloud_openai_key":         {"sk-test123"},
				"cloud_openai_embed_model": {"text-embedding-3-small"},
				"cloud_openai_chat_model":  {"gpt-4"},
				"cloud_rag_policy":         {"allow_rag"},
			},
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary config file for each test
			tmpFile, err := os.CreateTemp("", "config-*.json")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			// Write a minimal valid config
			initialConfig := &config.Config{
				Provider: config.ProviderConfig{
					Type:             "ollama",
					OllamaEndpoint:   "http://localhost:11434",
					OllamaEmbedModel: "nomic-embed-text",
					OllamaChatModel:  "llama3.2",
				},
				Privacy: config.PrivacyConfig{
					Enabled:        true,
					UseLocalAI:     true,
					CloudRAGPolicy: "no_rag",
				},
				Logging: config.LoggingConfig{
					Level: "info",
				},
				Guardrails: config.GuardrailsConfig{
					PIIDetection: "normal",
				},
				Server: config.ServerConfig{
					Port:        8080,
					BindAddress: "127.0.0.1",
				},
				UserMode: "single",
				Auth: config.AuthConfig{
					Provider: "userpass",
				},
			}
			if err := initialConfig.Save(tmpFile.Name()); err != nil {
				t.Fatalf("Failed to save initial config: %v", err)
			}

			// Create server with test config
			server := &Server{
				configPath:      tmpFile.Name(),
				logger:          &mockLogger{},
				providerManager: &mockProviderManager{},
			}

			// Create request
			req := httptest.NewRequest(http.MethodPost, "/api/config", strings.NewReader(tt.formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call handler
			server.handleConfig(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, rr.Code, rr.Body.String())
			}

			// Check error message if expected
			if tt.expectedError != "" {
				body := rr.Body.String()
				if !strings.Contains(body, tt.expectedError) {
					t.Errorf("Expected error message to contain %q, got %q", tt.expectedError, body)
				}
			}

			// Check success message if no error expected
			if tt.expectedError == "" && tt.expectedStatus == http.StatusOK {
				body := rr.Body.String()
				if !strings.Contains(body, "success") {
					t.Errorf("Expected success message, got %q", body)
				}
			}
		})
	}
}
