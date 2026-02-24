package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_DefaultConfig(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Load config (should create default)
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify defaults
	if cfg.Provider.Type != "ollama" {
		t.Errorf("Expected provider type 'ollama', got '%s'", cfg.Provider.Type)
	}
	if cfg.Privacy.Enabled != true {
		t.Errorf("Expected privacy mode enabled by default")
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", cfg.Server.Port)
	}
	if cfg.Server.BindAddress != "127.0.0.1" {
		t.Errorf("Expected bind address '127.0.0.1', got '%s'", cfg.Server.BindAddress)
	}

	// Verify config file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}
}

func TestLoad_ExistingConfig(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create a custom config
	customCfg := &Config{
		Provider: ProviderConfig{
			Type:             "ollama",
			OllamaEndpoint:   "http://localhost:11434",
			OllamaEmbedModel: "custom-model",
			OllamaChatModel:  "custom-chat",
		},
		Privacy: PrivacyConfig{
			Enabled: true,
		},
		Folders: []string{},
		Logging: LoggingConfig{
			Level: "debug",
		},
		Guardrails: GuardrailsConfig{
			MaxFileSizeMB:     20,
			AllowedExtensions: []string{".txt"},
			MaxConcurrent:     5,
			PIIDetection:      "strict",
			AutoSummarize:     false,
		},
		Server: ServerConfig{
			Port:        9090,
			BindAddress: "127.0.0.1",
		},
	}

	// Save custom config
	if err := customCfg.Save(configPath); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Load config
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify custom values
	if cfg.Provider.OllamaEmbedModel != "custom-model" {
		t.Errorf("Expected embed model 'custom-model', got '%s'", cfg.Provider.OllamaEmbedModel)
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("Expected log level 'debug', got '%s'", cfg.Logging.Level)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("Expected port 9090, got %d", cfg.Server.Port)
	}
	if cfg.Guardrails.MaxFileSizeMB != 20 {
		t.Errorf("Expected max file size 20, got %d", cfg.Guardrails.MaxFileSizeMB)
	}
}

func TestEnvOverrides(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Set environment variables
	os.Setenv("NOODEXX_PROVIDER", "openai")
	os.Setenv("NOODEXX_OPENAI_KEY", "test-key")
	os.Setenv("NOODEXX_PRIVACY_MODE", "false")
	os.Setenv("NOODEXX_LOG_LEVEL", "debug")
	os.Setenv("NOODEXX_SERVER_PORT", "9000")
	defer func() {
		os.Unsetenv("NOODEXX_PROVIDER")
		os.Unsetenv("NOODEXX_OPENAI_KEY")
		os.Unsetenv("NOODEXX_PRIVACY_MODE")
		os.Unsetenv("NOODEXX_LOG_LEVEL")
		os.Unsetenv("NOODEXX_SERVER_PORT")
	}()

	// Load config
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify environment overrides
	if cfg.Provider.Type != "openai" {
		t.Errorf("Expected provider type 'openai', got '%s'", cfg.Provider.Type)
	}
	if cfg.Provider.OpenAIKey != "test-key" {
		t.Errorf("Expected OpenAI key 'test-key', got '%s'", cfg.Provider.OpenAIKey)
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("Expected log level 'debug', got '%s'", cfg.Logging.Level)
	}
	if cfg.Server.Port != 9000 {
		t.Errorf("Expected port 9000, got %d", cfg.Server.Port)
	}
}

func TestValidate_PrivacyMode(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *Config
		expectError bool
	}{
		{
			name: "Valid privacy mode with Ollama",
			cfg: &Config{
				Provider: ProviderConfig{
					Type:           "ollama",
					OllamaEndpoint: "http://localhost:11434",
				},
				Privacy:    PrivacyConfig{Enabled: true},
				Logging:    LoggingConfig{Level: "info"},
				Guardrails: GuardrailsConfig{PIIDetection: "normal"},
				Server:     ServerConfig{Port: 8080},
			},
			expectError: false,
		},
		{
			name: "Invalid privacy mode with OpenAI",
			cfg: &Config{
				Provider: ProviderConfig{
					Type:      "openai",
					OpenAIKey: "test-key",
				},
				Privacy:    PrivacyConfig{Enabled: true},
				Logging:    LoggingConfig{Level: "info"},
				Guardrails: GuardrailsConfig{PIIDetection: "normal"},
				Server:     ServerConfig{Port: 8080},
			},
			expectError: true,
		},
		{
			name: "Invalid privacy mode with non-localhost Ollama",
			cfg: &Config{
				Provider: ProviderConfig{
					Type:           "ollama",
					OllamaEndpoint: "http://192.168.1.100:11434",
				},
				Privacy:    PrivacyConfig{Enabled: true},
				Logging:    LoggingConfig{Level: "info"},
				Guardrails: GuardrailsConfig{PIIDetection: "normal"},
				Server:     ServerConfig{Port: 8080},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.expectError && err == nil {
				t.Error("Expected validation error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestValidate_ProviderRequirements(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *Config
		expectError bool
	}{
		{
			name: "OpenAI without API key",
			cfg: &Config{
				Provider: ProviderConfig{
					Type: "openai",
				},
				Privacy:    PrivacyConfig{Enabled: false},
				Logging:    LoggingConfig{Level: "info"},
				Guardrails: GuardrailsConfig{PIIDetection: "normal"},
				Server:     ServerConfig{Port: 8080},
			},
			expectError: true,
		},
		{
			name: "Anthropic without API key",
			cfg: &Config{
				Provider: ProviderConfig{
					Type: "anthropic",
				},
				Privacy:    PrivacyConfig{Enabled: false},
				Logging:    LoggingConfig{Level: "info"},
				Guardrails: GuardrailsConfig{PIIDetection: "normal"},
				Server:     ServerConfig{Port: 8080},
			},
			expectError: true,
		},
		{
			name: "Unknown provider type",
			cfg: &Config{
				Provider: ProviderConfig{
					Type: "unknown",
				},
				Privacy:    PrivacyConfig{Enabled: false},
				Logging:    LoggingConfig{Level: "info"},
				Guardrails: GuardrailsConfig{PIIDetection: "normal"},
				Server:     ServerConfig{Port: 8080},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.expectError && err == nil {
				t.Error("Expected validation error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := &Config{
		Provider: ProviderConfig{
			Type:             "ollama",
			OllamaEndpoint:   "http://localhost:11434",
			OllamaEmbedModel: "test-model",
		},
		Privacy: PrivacyConfig{Enabled: true},
		Folders: []string{"/test/path"},
		Logging: LoggingConfig{Level: "info"},
		Guardrails: GuardrailsConfig{
			MaxFileSizeMB:     10,
			AllowedExtensions: []string{".txt"},
			MaxConcurrent:     3,
			PIIDetection:      "normal",
			AutoSummarize:     true,
		},
		Server: ServerConfig{
			Port:        8080,
			BindAddress: "127.0.0.1",
		},
	}

	// Save config
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Load and verify
	loadedCfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if loadedCfg.Provider.OllamaEmbedModel != "test-model" {
		t.Errorf("Expected embed model 'test-model', got '%s'", loadedCfg.Provider.OllamaEmbedModel)
	}
	if len(loadedCfg.Folders) != 1 || loadedCfg.Folders[0] != "/test/path" {
		t.Errorf("Expected folders ['/test/path'], got %v", loadedCfg.Folders)
	}
}
