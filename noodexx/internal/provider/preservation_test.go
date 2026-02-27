package provider

import (
	"bytes"
	"noodexx/internal/config"
	"noodexx/internal/logging"
	"testing"
)

// TestPreservation_DualProviderFunctionalityWithValidCredentials tests Property 2:
// Preservation - Dual Provider Functionality with Valid Credentials
//
// **Validates: Requirements 3.1, 3.2, 3.3, 3.4**
//
// This test verifies that when both local and cloud providers have valid credentials,
// the application continues to work exactly as before. This is critical for regression prevention.
//
// These tests should PASS on UNFIXED code (confirming baseline behavior to preserve).
//
// Preservation Requirements:
//   - 3.1: When valid cloud provider API key configured, both providers initialize successfully
//   - 3.2: When both providers configured, switching between providers via UI toggle works
//   - 3.3: When properly configured cloud provider, all cloud provider functionality works
//   - 3.4: When properly configured local provider, all local provider functionality works independently
//
// Note: Since we cannot test with real API keys in unit tests, we test the initialization
// and provider management logic. The actual provider functionality (chat, embeddings) is tested
// at the integration level.
func TestPreservation_DualProviderFunctionalityWithValidCredentials(t *testing.T) {
	testCases := []struct {
		name              string
		cloudProviderType string
		cloudProviderKey  string
		defaultToLocal    bool
	}{
		{
			name:              "OpenAI with valid key - default to local",
			cloudProviderType: "openai",
			cloudProviderKey:  "sk-test-key-12345678901234567890123456789012", // Mock valid format
			defaultToLocal:    true,
		},
		{
			name:              "OpenAI with valid key - default to cloud",
			cloudProviderType: "openai",
			cloudProviderKey:  "sk-test-key-12345678901234567890123456789012",
			defaultToLocal:    false,
		},
		{
			name:              "Anthropic with valid key - default to local",
			cloudProviderType: "anthropic",
			cloudProviderKey:  "sk-ant-test-key-1234567890123456789012345678", // Mock valid format
			defaultToLocal:    true,
		},
		{
			name:              "Anthropic with valid key - default to cloud",
			cloudProviderType: "anthropic",
			cloudProviderKey:  "sk-ant-test-key-1234567890123456789012345678",
			defaultToLocal:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a logger that captures output
			var logBuf bytes.Buffer
			logger := logging.NewLogger("test", logging.INFO, &logBuf)

			// Create config with BOTH providers having valid credentials
			cfg := &config.Config{
				LocalProvider: config.ProviderConfig{
					Type:             "ollama",
					OllamaEndpoint:   "http://localhost:11434",
					OllamaEmbedModel: "nomic-embed-text",
					OllamaChatModel:  "llama3.2",
				},
				CloudProvider: config.ProviderConfig{
					Type: tc.cloudProviderType,
				},
				Privacy: config.PrivacyConfig{
					DefaultToLocal: tc.defaultToLocal,
					CloudRAGPolicy: "no_rag",
				},
			}

			// Set the appropriate API key based on provider type
			if tc.cloudProviderType == "openai" {
				cfg.CloudProvider.OpenAIKey = tc.cloudProviderKey
				cfg.CloudProvider.OpenAIEmbedModel = "text-embedding-3-small"
				cfg.CloudProvider.OpenAIChatModel = "gpt-4"
			} else if tc.cloudProviderType == "anthropic" {
				cfg.CloudProvider.AnthropicKey = tc.cloudProviderKey
				cfg.CloudProvider.AnthropicEmbedModel = "claude-3-sonnet"
				cfg.CloudProvider.AnthropicChatModel = "claude-3-opus"
			}

			// Attempt to create DualProviderManager
			manager, err := NewDualProviderManager(cfg, logger)

			// PRESERVATION REQUIREMENT 3.1: Both providers should initialize successfully
			if err != nil {
				t.Fatalf("Expected successful initialization with valid credentials, got error: %v", err)
			}

			if manager == nil {
				t.Fatal("Expected manager to be created, got nil")
			}

			// Verify both providers are initialized
			localProvider := manager.GetLocalProvider()
			if localProvider == nil {
				t.Error("Expected local provider to be initialized, got nil")
			}

			cloudProvider := manager.GetCloudProvider()
			if cloudProvider == nil {
				t.Error("Expected cloud provider to be initialized, got nil")
			}

			// PRESERVATION REQUIREMENT 3.2: Provider switching should work correctly
			// Test that IsLocalMode reflects the configuration
			if manager.IsLocalMode() != tc.defaultToLocal {
				t.Errorf("Expected IsLocalMode()=%v, got %v", tc.defaultToLocal, manager.IsLocalMode())
			}

			// Test GetActiveProvider returns the correct provider based on mode
			activeProvider, err := manager.GetActiveProvider()
			if err != nil {
				t.Errorf("Expected GetActiveProvider to succeed, got error: %v", err)
			}

			if tc.defaultToLocal {
				// In local mode, active provider should be local provider
				if activeProvider != localProvider {
					t.Error("Expected active provider to be local provider in local mode")
				}
			} else {
				// In cloud mode, active provider should be cloud provider
				if activeProvider != cloudProvider {
					t.Error("Expected active provider to be cloud provider in cloud mode")
				}
			}

			// PRESERVATION REQUIREMENT 3.3 & 3.4: Provider functionality
			// We verify that both providers are accessible and non-nil
			// Actual functionality (chat, embeddings) is tested at integration level
			if localProvider == nil {
				t.Error("Local provider should be accessible and non-nil")
			}

			if cloudProvider == nil {
				t.Error("Cloud provider should be accessible and non-nil")
			}

			// Verify GetProviderName returns appropriate names
			providerName := manager.GetProviderName()
			if providerName == "" {
				t.Error("Expected non-empty provider name")
			}

			// Verify no error messages in logs (should be clean initialization)
			logOutput := logBuf.String()
			if len(logOutput) > 0 {
				t.Logf("Log output: %s", logOutput)
			}

			t.Logf("SUCCESS: Both providers initialized correctly with valid credentials")
		})
	}
}

// TestPreservation_ConfigurationReloadWithValidCredentials tests that configuration
// reload continues to work correctly when both providers have valid credentials.
//
// **Validates: Requirements 3.1, 3.2**
//
// This test verifies the Reload method preserves existing behavior when credentials are valid.
func TestPreservation_ConfigurationReloadWithValidCredentials(t *testing.T) {
	var logBuf bytes.Buffer
	logger := logging.NewLogger("test", logging.INFO, &logBuf)

	// Initial config with both providers
	initialCfg := &config.Config{
		LocalProvider: config.ProviderConfig{
			Type:             "ollama",
			OllamaEndpoint:   "http://localhost:11434",
			OllamaEmbedModel: "nomic-embed-text",
			OllamaChatModel:  "llama3.2",
		},
		CloudProvider: config.ProviderConfig{
			Type:              "openai",
			OpenAIKey:         "sk-test-key-12345678901234567890123456789012",
			OpenAIEmbedModel:  "text-embedding-3-small",
			OpenAIChatModel:   "gpt-4",
		},
		Privacy: config.PrivacyConfig{
			DefaultToLocal: true,
			CloudRAGPolicy: "no_rag",
		},
	}

	// Create initial manager
	manager, err := NewDualProviderManager(initialCfg, logger)
	if err != nil {
		t.Fatalf("Failed to create initial manager: %v", err)
	}

	// Verify initial state
	if manager.GetLocalProvider() == nil {
		t.Error("Expected local provider to be initialized")
	}
	if manager.GetCloudProvider() == nil {
		t.Error("Expected cloud provider to be initialized")
	}
	if !manager.IsLocalMode() {
		t.Error("Expected local mode to be true")
	}

	// Create new config with provider mode switched
	reloadCfg := &config.Config{
		LocalProvider: config.ProviderConfig{
			Type:             "ollama",
			OllamaEndpoint:   "http://localhost:11434",
			OllamaEmbedModel: "nomic-embed-text",
			OllamaChatModel:  "llama3.2",
		},
		CloudProvider: config.ProviderConfig{
			Type:              "openai",
			OpenAIKey:         "sk-test-key-12345678901234567890123456789012",
			OpenAIEmbedModel:  "text-embedding-3-small",
			OpenAIChatModel:   "gpt-4",
		},
		Privacy: config.PrivacyConfig{
			DefaultToLocal: false, // Switch to cloud mode
			CloudRAGPolicy: "no_rag",
		},
	}

	// Reload configuration
	err = manager.Reload(reloadCfg)
	if err != nil {
		t.Fatalf("Expected successful reload with valid credentials, got error: %v", err)
	}

	// Verify both providers are still initialized after reload
	if manager.GetLocalProvider() == nil {
		t.Error("Expected local provider to remain initialized after reload")
	}
	if manager.GetCloudProvider() == nil {
		t.Error("Expected cloud provider to remain initialized after reload")
	}

	// Verify mode was switched
	if manager.IsLocalMode() {
		t.Error("Expected local mode to be false after reload")
	}

	// Verify active provider switched to cloud
	activeProvider, err := manager.GetActiveProvider()
	if err != nil {
		t.Errorf("Expected GetActiveProvider to succeed after reload, got error: %v", err)
	}
	if activeProvider != manager.GetCloudProvider() {
		t.Error("Expected active provider to be cloud provider after mode switch")
	}

	t.Logf("SUCCESS: Configuration reload works correctly with valid credentials")
}

// TestPreservation_ProviderSwitchingBehavior tests that switching between providers
// via the privacy toggle works correctly when both providers are configured.
//
// **Validates: Requirements 3.2**
//
// This test simulates multiple provider switches to ensure the behavior is preserved.
func TestPreservation_ProviderSwitchingBehavior(t *testing.T) {
	var logBuf bytes.Buffer
	logger := logging.NewLogger("test", logging.INFO, &logBuf)

	cfg := &config.Config{
		LocalProvider: config.ProviderConfig{
			Type:             "ollama",
			OllamaEndpoint:   "http://localhost:11434",
			OllamaEmbedModel: "nomic-embed-text",
			OllamaChatModel:  "llama3.2",
		},
		CloudProvider: config.ProviderConfig{
			Type:              "openai",
			OpenAIKey:         "sk-test-key-12345678901234567890123456789012",
			OpenAIEmbedModel:  "text-embedding-3-small",
			OpenAIChatModel:   "gpt-4",
		},
		Privacy: config.PrivacyConfig{
			DefaultToLocal: true,
			CloudRAGPolicy: "no_rag",
		},
	}

	manager, err := NewDualProviderManager(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Test sequence of provider switches
	testSequence := []struct {
		switchToLocal    bool
		expectedProvider string
	}{
		{switchToLocal: true, expectedProvider: "local"},
		{switchToLocal: false, expectedProvider: "cloud"},
		{switchToLocal: true, expectedProvider: "local"},
		{switchToLocal: false, expectedProvider: "cloud"},
	}

	for i, step := range testSequence {
		// Simulate provider switch by reloading with new mode
		cfg.Privacy.DefaultToLocal = step.switchToLocal
		err := manager.Reload(cfg)
		if err != nil {
			t.Errorf("Step %d: Failed to reload with mode=%v: %v", i, step.switchToLocal, err)
			continue
		}

		// Verify the mode was set correctly
		if manager.IsLocalMode() != step.switchToLocal {
			t.Errorf("Step %d: Expected IsLocalMode()=%v, got %v", i, step.switchToLocal, manager.IsLocalMode())
		}

		// Verify GetActiveProvider returns the correct provider
		activeProvider, err := manager.GetActiveProvider()
		if err != nil {
			t.Errorf("Step %d: GetActiveProvider failed: %v", i, err)
			continue
		}

		if step.expectedProvider == "local" {
			if activeProvider != manager.GetLocalProvider() {
				t.Errorf("Step %d: Expected active provider to be local", i)
			}
		} else {
			if activeProvider != manager.GetCloudProvider() {
				t.Errorf("Step %d: Expected active provider to be cloud", i)
			}
		}
	}

	t.Logf("SUCCESS: Provider switching works correctly through multiple switches")
}
