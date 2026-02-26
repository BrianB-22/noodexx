package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"noodexx/internal/config"
	"os"
	"testing"
)

// MockProviderManager for testing
type MockProviderManager struct {
	providerName string
}

func (m *MockProviderManager) GetActiveProvider() (LLMProvider, error) {
	return nil, nil
}

func (m *MockProviderManager) IsLocalMode() bool {
	return true
}

func (m *MockProviderManager) GetProviderName() string {
	return m.providerName
}

func (m *MockProviderManager) Reload(cfg interface{}) error {
	return nil
}

// MockRAGEnforcer for testing
type MockRAGEnforcer struct {
	ragStatus string
}

func (m *MockRAGEnforcer) ShouldPerformRAG() bool {
	return true
}

func (m *MockRAGEnforcer) GetRAGStatus() string {
	return m.ragStatus
}

// MockLogger for testing
type MockLogger struct{}

func (m *MockLogger) Debug(format string, args ...interface{})         {}
func (m *MockLogger) Info(format string, args ...interface{})          {}
func (m *MockLogger) Warn(format string, args ...interface{})          {}
func (m *MockLogger) Error(format string, args ...interface{})         {}
func (m *MockLogger) WithContext(key string, value interface{}) Logger { return m }
func (m *MockLogger) WithFields(fields map[string]interface{}) Logger  { return m }

// TestHandlePrivacyToggle_TogglingToLocal tests toggling to local mode
func TestHandlePrivacyToggle_TogglingToLocal(t *testing.T) {
	// Create temporary config file
	tmpFile, err := os.CreateTemp("", "config-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write initial config
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

	// Create server with mocks
	server := &Server{
		configPath:      tmpFile.Name(),
		logger:          &MockLogger{},
		providerManager: &MockProviderManager{providerName: "Ollama (llama3.2)"},
		ragEnforcer:     &MockRAGEnforcer{ragStatus: "RAG Enabled (Local)"},
	}

	// Create request
	reqBody := map[string]string{"mode": "local"}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/privacy-toggle", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	// Execute handler
	server.handlePrivacyToggle(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Parse response
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify response fields
	if success, ok := resp["success"].(bool); !ok || !success {
		t.Errorf("Expected success=true, got %v", resp["success"])
	}

	if mode, ok := resp["mode"].(string); !ok || mode != "local" {
		t.Errorf("Expected mode='local', got %v", resp["mode"])
	}

	if provider, ok := resp["provider"].(string); !ok || provider != "Ollama (llama3.2)" {
		t.Errorf("Expected provider='Ollama (llama3.2)', got %v", resp["provider"])
	}

	if ragStatus, ok := resp["rag_status"].(string); !ok || ragStatus != "RAG Enabled (Local)" {
		t.Errorf("Expected rag_status='RAG Enabled (Local)', got %v", resp["rag_status"])
	}

	// Verify config was updated
	loadedCfg, err := config.Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if !loadedCfg.Privacy.UseLocalAI {
		t.Errorf("Expected UseLocalAI=true, got false")
	}
}

// TestHandlePrivacyToggle_TogglingToCloud tests toggling to cloud mode
func TestHandlePrivacyToggle_TogglingToCloud(t *testing.T) {
	// Create temporary config file
	tmpFile, err := os.CreateTemp("", "config-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write initial config
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
			UseLocalAI:     true, // Start in local mode
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

	// Create server with mocks
	server := &Server{
		configPath:      tmpFile.Name(),
		logger:          &MockLogger{},
		providerManager: &MockProviderManager{providerName: "OpenAI (gpt-4)"},
		ragEnforcer:     &MockRAGEnforcer{ragStatus: "RAG Enabled (Cloud)"},
	}

	// Create request
	reqBody := map[string]string{"mode": "cloud"}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/privacy-toggle", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	// Execute handler
	server.handlePrivacyToggle(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Parse response
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify response fields
	if success, ok := resp["success"].(bool); !ok || !success {
		t.Errorf("Expected success=true, got %v", resp["success"])
	}

	if mode, ok := resp["mode"].(string); !ok || mode != "cloud" {
		t.Errorf("Expected mode='cloud', got %v", resp["mode"])
	}

	if provider, ok := resp["provider"].(string); !ok || provider != "OpenAI (gpt-4)" {
		t.Errorf("Expected provider='OpenAI (gpt-4)', got %v", resp["provider"])
	}

	// Verify config was updated
	loadedCfg, err := config.Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if loadedCfg.Privacy.UseLocalAI {
		t.Errorf("Expected UseLocalAI=false, got true")
	}
}

// TestHandlePrivacyToggle_InvalidMode tests invalid mode value
func TestHandlePrivacyToggle_InvalidMode(t *testing.T) {
	// Create temporary config file
	tmpFile, err := os.CreateTemp("", "config-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write initial config
	cfg := &config.Config{
		Provider: config.ProviderConfig{
			Type: "ollama",
		},
		Privacy: config.PrivacyConfig{
			UseLocalAI:     true,
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

	// Create server with mocks
	server := &Server{
		configPath:      tmpFile.Name(),
		logger:          &MockLogger{},
		providerManager: &MockProviderManager{providerName: "Ollama"},
		ragEnforcer:     &MockRAGEnforcer{ragStatus: "RAG Enabled"},
	}

	// Create request with invalid mode
	reqBody := map[string]string{"mode": "invalid"}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/privacy-toggle", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	// Execute handler
	server.handlePrivacyToggle(w, req)

	// Check response
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	// Parse response
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify error response
	if success, ok := resp["success"].(bool); !ok || success {
		t.Errorf("Expected success=false, got %v", resp["success"])
	}

	if errorMsg, ok := resp["error"].(string); !ok || errorMsg == "" {
		t.Errorf("Expected error message, got %v", resp["error"])
	}
}

// TestHandlePrivacyToggle_MethodNotAllowed tests non-POST method
func TestHandlePrivacyToggle_MethodNotAllowed(t *testing.T) {
	server := &Server{
		logger: &MockLogger{},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/privacy-toggle", nil)
	w := httptest.NewRecorder()

	server.handlePrivacyToggle(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

// TestHandlePrivacyToggle_InvalidJSON tests invalid JSON body
func TestHandlePrivacyToggle_InvalidJSON(t *testing.T) {
	server := &Server{
		logger: &MockLogger{},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/privacy-toggle", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	server.handlePrivacyToggle(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestHandlePrivacyToggle_ResponseTime tests that response completes within 1 second
func TestHandlePrivacyToggle_ResponseTime(t *testing.T) {
	// Create temporary config file
	tmpFile, err := os.CreateTemp("", "config-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write initial config
	cfg := &config.Config{
		Provider: config.ProviderConfig{
			Type: "ollama",
		},
		Privacy: config.PrivacyConfig{
			UseLocalAI:     false,
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

	// Create server with mocks
	server := &Server{
		configPath:      tmpFile.Name(),
		logger:          &MockLogger{},
		providerManager: &MockProviderManager{providerName: "Ollama"},
		ragEnforcer:     &MockRAGEnforcer{ragStatus: "RAG Enabled"},
	}

	// Create request
	reqBody := map[string]string{"mode": "local"}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/privacy-toggle", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	// Execute handler
	server.handlePrivacyToggle(w, req)

	// Parse response
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check latency
	if latency, ok := resp["latency_ms"].(float64); ok {
		if latency > 1000 {
			t.Errorf("Expected latency < 1000ms, got %.2fms", latency)
		}
	} else {
		t.Error("Expected latency_ms in response")
	}
}
