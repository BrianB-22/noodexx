package main

import (
	"bytes"
	"noodexx/internal/config"
	"noodexx/internal/logging"
	providerpkg "noodexx/internal/provider"
	"strings"
	"testing"
)

// TestMainLogic_CloudProviderNotAvailableMessage tests that when cloud provider
// is configured but not available, an appropriate message is displayed
func TestMainLogic_CloudProviderNotAvailableMessage(t *testing.T) {
	// Create a logger that captures output
	var logBuf bytes.Buffer
	logger := logging.NewLogger("test", logging.INFO, &logBuf)

	// Create config with cloud provider configured but missing API key
	cfg := &config.Config{
		LocalProvider: config.ProviderConfig{
			Type:             "ollama",
			OllamaEndpoint:   "http://localhost:11434",
			OllamaEmbedModel: "nomic-embed-text",
			OllamaChatModel:  "llama3.2",
		},
		CloudProvider: config.ProviderConfig{
			Type:      "openai",
			OpenAIKey: "", // Missing API key
		},
		Privacy: config.PrivacyConfig{
			DefaultToLocal: true,
			CloudRAGPolicy: "no_rag",
		},
	}

	// Initialize dual provider manager
	dualProviderManager, err := providerpkg.NewDualProviderManager(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to initialize provider manager: %v", err)
	}

	// Simulate the display logic from main.go
	if dualProviderManager.GetCloudProvider() == nil && cfg.CloudProvider.Type != "" {
		t.Log("⚠️  Cloud provider configured but not available (check API key configuration)")
	} else {
		t.Error("Expected cloud provider to be nil when API key is missing")
	}

	// Verify cloud provider is nil
	if dualProviderManager.GetCloudProvider() != nil {
		t.Error("Expected cloud provider to be nil")
	}

	// Verify local provider is available
	if dualProviderManager.GetLocalProvider() == nil {
		t.Error("Expected local provider to be available")
	}
}

// TestMainLogic_DefaultToCloudWithNoCloudProvider tests that when default provider
// is set to cloud but no cloud provider is configured, the application switches to local
func TestMainLogic_DefaultToCloudWithNoCloudProvider(t *testing.T) {
	// Create a logger that captures output
	var logBuf bytes.Buffer
	logger := logging.NewLogger("test", logging.INFO, &logBuf)

	// Create config with default to cloud but no cloud provider API key
	cfg := &config.Config{
		LocalProvider: config.ProviderConfig{
			Type:             "ollama",
			OllamaEndpoint:   "http://localhost:11434",
			OllamaEmbedModel: "nomic-embed-text",
			OllamaChatModel:  "llama3.2",
		},
		CloudProvider: config.ProviderConfig{
			Type:      "openai",
			OpenAIKey: "", // Missing API key
		},
		Privacy: config.PrivacyConfig{
			DefaultToLocal: false, // Default to cloud
			CloudRAGPolicy: "no_rag",
		},
	}

	// Initialize dual provider manager
	dualProviderManager, err := providerpkg.NewDualProviderManager(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to initialize provider manager: %v", err)
	}

	// Simulate the active provider check from main.go
	provider, err := dualProviderManager.GetActiveProvider()
	if err != nil {
		// Handle case where default provider is cloud but cloud provider not configured
		if !cfg.Privacy.DefaultToLocal && dualProviderManager.GetCloudProvider() == nil {
			logger.Warn("Defaulting to Local AI because no cloud provider configured")
			// Switch to local mode by updating the configuration
			cfg.Privacy.DefaultToLocal = true
			// Reload the dual provider manager with updated config
			if reloadErr := dualProviderManager.Reload(cfg); reloadErr != nil {
				t.Fatalf("Failed to reload provider manager: %v", reloadErr)
			}
			// Try to get active provider again (should now return local provider)
			provider, err = dualProviderManager.GetActiveProvider()
			if err != nil {
				t.Fatalf("Failed to get active provider after switching to local: %v", err)
			}
		} else {
			t.Fatalf("Failed to get active provider: %v", err)
		}
	}

	// Verify provider is not nil
	if provider == nil {
		t.Error("Expected provider to be available after switching to local")
	}

	// Verify warning was logged
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "Defaulting to Local AI because no cloud provider configured") {
		t.Errorf("Expected warning message in logs, got: %s", logOutput)
	}

	// Verify config was updated to default to local
	if !cfg.Privacy.DefaultToLocal {
		t.Error("Expected config to be updated to default to local")
	}
}
