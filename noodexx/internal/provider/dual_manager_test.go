package provider

import (
	"bytes"
	"noodexx/internal/config"
	"noodexx/internal/logging"
	"testing"
)

// Helper function to create a test logger
func createTestLogger() *logging.Logger {
	var buf bytes.Buffer
	return logging.NewLogger("test", logging.INFO, &buf)
}

// Helper function to create a config with both providers configured
func createDualProviderConfig() *config.Config {
	return &config.Config{
		LocalProvider: config.ProviderConfig{
			Type:             "ollama",
			OllamaEndpoint:   "http://localhost:11434",
			OllamaEmbedModel: "nomic-embed-text",
			OllamaChatModel:  "llama3.2",
		},
		CloudProvider: config.ProviderConfig{
			Type:             "openai",
			OpenAIKey:        "test-key",
			OpenAIEmbedModel: "text-embedding-3-small",
			OpenAIChatModel:  "gpt-4",
		},
		Privacy: config.PrivacyConfig{
			DefaultToLocal:     true,
			CloudRAGPolicy: "no_rag",
		},
	}
}

// Helper function to create a config with only local provider
func createLocalOnlyConfig() *config.Config {
	return &config.Config{
		LocalProvider: config.ProviderConfig{
			Type:             "ollama",
			OllamaEndpoint:   "http://localhost:11434",
			OllamaEmbedModel: "nomic-embed-text",
			OllamaChatModel:  "llama3.2",
		},
		CloudProvider: config.ProviderConfig{
			Type: "", // Not configured
		},
		Privacy: config.PrivacyConfig{
			DefaultToLocal:     true,
			CloudRAGPolicy: "no_rag",
		},
	}
}

// Helper function to create a config with only cloud provider
func createCloudOnlyConfig() *config.Config {
	return &config.Config{
		LocalProvider: config.ProviderConfig{
			Type: "", // Not configured
		},
		CloudProvider: config.ProviderConfig{
			Type:             "openai",
			OpenAIKey:        "test-key",
			OpenAIEmbedModel: "text-embedding-3-small",
			OpenAIChatModel:  "gpt-4",
		},
		Privacy: config.PrivacyConfig{
			DefaultToLocal:     false,
			CloudRAGPolicy: "allow_rag",
		},
	}
}

// TestGetActiveProvider_LocalMode tests GetActiveProvider returns local provider when DefaultToLocal is true
func TestGetActiveProvider_LocalMode(t *testing.T) {
	cfg := createDualProviderConfig()
	cfg.Privacy.DefaultToLocal = true
	logger := createTestLogger()

	manager, err := NewDualProviderManager(cfg, logger)
	if err != nil {
		t.Fatalf("NewDualProviderManager() failed: %v", err)
	}

	provider, err := manager.GetActiveProvider()
	if err != nil {
		t.Fatalf("GetActiveProvider() returned error: %v", err)
	}

	if provider == nil {
		t.Fatal("GetActiveProvider() returned nil provider")
	}

	// Verify it's the local provider by checking it's not nil
	if manager.localProvider == nil {
		t.Fatal("Local provider should be initialized")
	}
	if provider != manager.localProvider {
		t.Error("GetActiveProvider() should return local provider when DefaultToLocal is true")
	}
}

// TestGetActiveProvider_CloudMode tests GetActiveProvider returns cloud provider when DefaultToLocal is false
func TestGetActiveProvider_CloudMode(t *testing.T) {
	cfg := createDualProviderConfig()
	cfg.Privacy.DefaultToLocal = false
	logger := createTestLogger()

	manager, err := NewDualProviderManager(cfg, logger)
	if err != nil {
		t.Fatalf("NewDualProviderManager() failed: %v", err)
	}

	provider, err := manager.GetActiveProvider()
	if err != nil {
		t.Fatalf("GetActiveProvider() returned error: %v", err)
	}

	if provider == nil {
		t.Fatal("GetActiveProvider() returned nil provider")
	}

	// Verify it's the cloud provider
	if manager.cloudProvider == nil {
		t.Fatal("Cloud provider should be initialized")
	}
	if provider != manager.cloudProvider {
		t.Error("GetActiveProvider() should return cloud provider when DefaultToLocal is false")
	}
}

// TestGetActiveProvider_UnconfiguredLocalProvider tests error when local provider is not configured
func TestGetActiveProvider_UnconfiguredLocalProvider(t *testing.T) {
	cfg := createCloudOnlyConfig()
	cfg.Privacy.DefaultToLocal = true // Try to use local, but it's not configured
	logger := createTestLogger()

	manager, err := NewDualProviderManager(cfg, logger)
	if err != nil {
		t.Fatalf("NewDualProviderManager() failed: %v", err)
	}

	provider, err := manager.GetActiveProvider()
	if err == nil {
		t.Fatal("GetActiveProvider() should return error when local provider is not configured")
	}

	if provider != nil {
		t.Error("GetActiveProvider() should return nil provider when error occurs")
	}

	expectedError := "local provider not configured"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

// TestGetActiveProvider_UnconfiguredCloudProvider tests error when cloud provider is not configured
func TestGetActiveProvider_UnconfiguredCloudProvider(t *testing.T) {
	cfg := createLocalOnlyConfig()
	cfg.Privacy.DefaultToLocal = false // Try to use cloud, but it's not configured
	logger := createTestLogger()

	manager, err := NewDualProviderManager(cfg, logger)
	if err != nil {
		t.Fatalf("NewDualProviderManager() failed: %v", err)
	}

	provider, err := manager.GetActiveProvider()
	if err == nil {
		t.Fatal("GetActiveProvider() should return error when cloud provider is not configured")
	}

	if provider != nil {
		t.Error("GetActiveProvider() should return nil provider when error occurs")
	}

	expectedError := "cloud provider not configured"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

// TestIsLocalMode_LocalEnabled tests IsLocalMode returns true when DefaultToLocal is true
func TestIsLocalMode_LocalEnabled(t *testing.T) {
	cfg := createDualProviderConfig()
	cfg.Privacy.DefaultToLocal = true
	logger := createTestLogger()

	manager, err := NewDualProviderManager(cfg, logger)
	if err != nil {
		t.Fatalf("NewDualProviderManager() failed: %v", err)
	}

	if !manager.IsLocalMode() {
		t.Error("IsLocalMode() should return true when DefaultToLocal is true")
	}
}

// TestIsLocalMode_CloudEnabled tests IsLocalMode returns false when DefaultToLocal is false
func TestIsLocalMode_CloudEnabled(t *testing.T) {
	cfg := createDualProviderConfig()
	cfg.Privacy.DefaultToLocal = false
	logger := createTestLogger()

	manager, err := NewDualProviderManager(cfg, logger)
	if err != nil {
		t.Fatalf("NewDualProviderManager() failed: %v", err)
	}

	if manager.IsLocalMode() {
		t.Error("IsLocalMode() should return false when DefaultToLocal is false")
	}
}

// TestGetProviderName_LocalOllama tests GetProviderName returns correct name for local Ollama
func TestGetProviderName_LocalOllama(t *testing.T) {
	cfg := createDualProviderConfig()
	cfg.Privacy.DefaultToLocal = true
	logger := createTestLogger()

	manager, err := NewDualProviderManager(cfg, logger)
	if err != nil {
		t.Fatalf("NewDualProviderManager() failed: %v", err)
	}

	name := manager.GetProviderName()
	expected := "Local AI (ollama)"
	if name != expected {
		t.Errorf("Expected provider name '%s', got '%s'", expected, name)
	}
}

// TestGetProviderName_CloudOpenAI tests GetProviderName returns correct name for cloud OpenAI
func TestGetProviderName_CloudOpenAI(t *testing.T) {
	cfg := createDualProviderConfig()
	cfg.Privacy.DefaultToLocal = false
	logger := createTestLogger()

	manager, err := NewDualProviderManager(cfg, logger)
	if err != nil {
		t.Fatalf("NewDualProviderManager() failed: %v", err)
	}

	name := manager.GetProviderName()
	expected := "Cloud AI (gpt-4)"
	if name != expected {
		t.Errorf("Expected provider name '%s', got '%s'", expected, name)
	}
}

// TestGetProviderName_CloudAnthropic tests GetProviderName returns correct name for cloud Anthropic
func TestGetProviderName_CloudAnthropic(t *testing.T) {
	cfg := createDualProviderConfig()
	cfg.CloudProvider.Type = "anthropic"
	cfg.CloudProvider.AnthropicKey = "test-key"
	cfg.CloudProvider.AnthropicChatModel = "claude-3-opus"
	cfg.Privacy.DefaultToLocal = false
	logger := createTestLogger()

	manager, err := NewDualProviderManager(cfg, logger)
	if err != nil {
		t.Fatalf("NewDualProviderManager() failed: %v", err)
	}

	name := manager.GetProviderName()
	expected := "Cloud AI (claude-3-opus)"
	if name != expected {
		t.Errorf("Expected provider name '%s', got '%s'", expected, name)
	}
}

// TestGetProviderName_UnconfiguredLocal tests GetProviderName when local provider is not configured
func TestGetProviderName_UnconfiguredLocal(t *testing.T) {
	cfg := createCloudOnlyConfig()
	cfg.Privacy.DefaultToLocal = true // Try to use local, but it's not configured
	logger := createTestLogger()

	manager, err := NewDualProviderManager(cfg, logger)
	if err != nil {
		t.Fatalf("NewDualProviderManager() failed: %v", err)
	}

	name := manager.GetProviderName()
	expected := "Local AI (Not Configured)"
	if name != expected {
		t.Errorf("Expected provider name '%s', got '%s'", expected, name)
	}
}

// TestGetProviderName_UnconfiguredCloud tests GetProviderName when cloud provider is not configured
func TestGetProviderName_UnconfiguredCloud(t *testing.T) {
	cfg := createLocalOnlyConfig()
	cfg.Privacy.DefaultToLocal = false // Try to use cloud, but it's not configured
	logger := createTestLogger()

	manager, err := NewDualProviderManager(cfg, logger)
	if err != nil {
		t.Fatalf("NewDualProviderManager() failed: %v", err)
	}

	name := manager.GetProviderName()
	expected := "Cloud AI (Not Configured)"
	if name != expected {
		t.Errorf("Expected provider name '%s', got '%s'", expected, name)
	}
}

// TestGetProviderName_CloudOpenAIWithoutModel tests GetProviderName falls back to provider type
func TestGetProviderName_CloudOpenAIWithoutModel(t *testing.T) {
	cfg := createDualProviderConfig()
	cfg.CloudProvider.OpenAIChatModel = "" // No model specified
	cfg.Privacy.DefaultToLocal = false
	logger := createTestLogger()

	manager, err := NewDualProviderManager(cfg, logger)
	if err != nil {
		t.Fatalf("NewDualProviderManager() failed: %v", err)
	}

	name := manager.GetProviderName()
	expected := "Cloud AI (OpenAI)"
	if name != expected {
		t.Errorf("Expected provider name '%s', got '%s'", expected, name)
	}
}

// TestReload_UpdateLocalProvider tests Reload updates local provider configuration
func TestReload_UpdateLocalProvider(t *testing.T) {
	cfg := createDualProviderConfig()
	logger := createTestLogger()

	manager, err := NewDualProviderManager(cfg, logger)
	if err != nil {
		t.Fatalf("NewDualProviderManager() failed: %v", err)
	}

	// Update local provider configuration
	newCfg := createDualProviderConfig()
	newCfg.LocalProvider.OllamaChatModel = "llama3.3"

	err = manager.Reload(newCfg)
	if err != nil {
		t.Fatalf("Reload() failed: %v", err)
	}

	// Verify config was updated
	if manager.config.LocalProvider.OllamaChatModel != "llama3.3" {
		t.Errorf("Expected local chat model 'llama3.3', got '%s'", manager.config.LocalProvider.OllamaChatModel)
	}

	// Verify local provider is still initialized
	if manager.localProvider == nil {
		t.Error("Local provider should still be initialized after reload")
	}
}

// TestReload_UpdateCloudProvider tests Reload updates cloud provider configuration
func TestReload_UpdateCloudProvider(t *testing.T) {
	cfg := createDualProviderConfig()
	logger := createTestLogger()

	manager, err := NewDualProviderManager(cfg, logger)
	if err != nil {
		t.Fatalf("NewDualProviderManager() failed: %v", err)
	}

	// Update cloud provider configuration
	newCfg := createDualProviderConfig()
	newCfg.CloudProvider.OpenAIChatModel = "gpt-4-turbo"

	err = manager.Reload(newCfg)
	if err != nil {
		t.Fatalf("Reload() failed: %v", err)
	}

	// Verify config was updated
	if manager.config.CloudProvider.OpenAIChatModel != "gpt-4-turbo" {
		t.Errorf("Expected cloud chat model 'gpt-4-turbo', got '%s'", manager.config.CloudProvider.OpenAIChatModel)
	}

	// Verify cloud provider is still initialized
	if manager.cloudProvider == nil {
		t.Error("Cloud provider should still be initialized after reload")
	}
}

// TestReload_RemoveLocalProvider tests Reload handles removal of local provider
func TestReload_RemoveLocalProvider(t *testing.T) {
	cfg := createDualProviderConfig()
	logger := createTestLogger()

	manager, err := NewDualProviderManager(cfg, logger)
	if err != nil {
		t.Fatalf("NewDualProviderManager() failed: %v", err)
	}

	// Remove local provider from config
	newCfg := createCloudOnlyConfig()

	err = manager.Reload(newCfg)
	if err != nil {
		t.Fatalf("Reload() failed: %v", err)
	}

	// Verify local provider was removed
	if manager.localProvider != nil {
		t.Error("Local provider should be nil after removal from config")
	}

	// Verify cloud provider is still available
	if manager.cloudProvider == nil {
		t.Error("Cloud provider should still be initialized")
	}
}

// TestReload_RemoveCloudProvider tests Reload handles removal of cloud provider
func TestReload_RemoveCloudProvider(t *testing.T) {
	cfg := createDualProviderConfig()
	logger := createTestLogger()

	manager, err := NewDualProviderManager(cfg, logger)
	if err != nil {
		t.Fatalf("NewDualProviderManager() failed: %v", err)
	}

	// Remove cloud provider from config
	newCfg := createLocalOnlyConfig()

	err = manager.Reload(newCfg)
	if err != nil {
		t.Fatalf("Reload() failed: %v", err)
	}

	// Verify cloud provider was removed
	if manager.cloudProvider != nil {
		t.Error("Cloud provider should be nil after removal from config")
	}

	// Verify local provider is still available
	if manager.localProvider == nil {
		t.Error("Local provider should still be initialized")
	}
}

// TestReload_RemoveBothProviders tests Reload returns error when both providers are removed
func TestReload_RemoveBothProviders(t *testing.T) {
	cfg := createDualProviderConfig()
	logger := createTestLogger()

	manager, err := NewDualProviderManager(cfg, logger)
	if err != nil {
		t.Fatalf("NewDualProviderManager() failed: %v", err)
	}

	// Remove both providers from config
	newCfg := &config.Config{
		LocalProvider: config.ProviderConfig{Type: ""},
		CloudProvider: config.ProviderConfig{Type: ""},
		Privacy: config.PrivacyConfig{
			DefaultToLocal:     true,
			CloudRAGPolicy: "no_rag",
		},
	}

	err = manager.Reload(newCfg)
	if err == nil {
		t.Fatal("Reload() should return error when both providers are removed")
	}

	expectedError := "at least one provider (local or cloud) must be configured after reload"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

// TestReload_SwitchProviderTypes tests Reload handles switching provider types
func TestReload_SwitchProviderTypes(t *testing.T) {
	cfg := createDualProviderConfig()
	logger := createTestLogger()

	manager, err := NewDualProviderManager(cfg, logger)
	if err != nil {
		t.Fatalf("NewDualProviderManager() failed: %v", err)
	}

	// Switch cloud provider from OpenAI to Anthropic
	newCfg := createDualProviderConfig()
	newCfg.CloudProvider.Type = "anthropic"
	newCfg.CloudProvider.AnthropicKey = "test-anthropic-key"
	newCfg.CloudProvider.AnthropicChatModel = "claude-3-opus"
	newCfg.CloudProvider.OpenAIKey = ""
	newCfg.CloudProvider.OpenAIChatModel = ""

	err = manager.Reload(newCfg)
	if err != nil {
		t.Fatalf("Reload() failed: %v", err)
	}

	// Verify config was updated
	if manager.config.CloudProvider.Type != "anthropic" {
		t.Errorf("Expected cloud provider type 'anthropic', got '%s'", manager.config.CloudProvider.Type)
	}

	// Verify cloud provider is still initialized
	if manager.cloudProvider == nil {
		t.Error("Cloud provider should be initialized after reload")
	}
}

// TestReload_UpdatePrivacySettings tests Reload updates privacy settings
func TestReload_UpdatePrivacySettings(t *testing.T) {
	cfg := createDualProviderConfig()
	cfg.Privacy.DefaultToLocal = true
	logger := createTestLogger()

	manager, err := NewDualProviderManager(cfg, logger)
	if err != nil {
		t.Fatalf("NewDualProviderManager() failed: %v", err)
	}

	// Update privacy settings
	newCfg := createDualProviderConfig()
	newCfg.Privacy.DefaultToLocal = false
	newCfg.Privacy.CloudRAGPolicy = "allow_rag"

	err = manager.Reload(newCfg)
	if err != nil {
		t.Fatalf("Reload() failed: %v", err)
	}

	// Verify privacy settings were updated
	if manager.config.Privacy.DefaultToLocal != false {
		t.Error("Expected DefaultToLocal to be false after reload")
	}
	if manager.config.Privacy.CloudRAGPolicy != "allow_rag" {
		t.Errorf("Expected CloudRAGPolicy 'allow_rag', got '%s'", manager.config.Privacy.CloudRAGPolicy)
	}

	// Verify GetActiveProvider now returns cloud provider
	provider, err := manager.GetActiveProvider()
	if err != nil {
		t.Fatalf("GetActiveProvider() failed: %v", err)
	}
	if provider != manager.cloudProvider {
		t.Error("GetActiveProvider() should return cloud provider after privacy toggle change")
	}
}
