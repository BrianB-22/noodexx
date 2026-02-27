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
	if cfg.Privacy.DefaultToLocal != true {
		t.Errorf("Expected default_to_local enabled by default")
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", cfg.Server.Port)
	}
	if cfg.Server.BindAddress != "127.0.0.1" {
		t.Errorf("Expected bind address '127.0.0.1', got '%s'", cfg.Server.BindAddress)
	}

	// Verify logging defaults
	if cfg.Logging.Level != "info" {
		t.Errorf("Expected log level 'info', got '%s'", cfg.Logging.Level)
	}
	if cfg.Logging.DebugEnabled != true {
		t.Errorf("Expected debug_enabled true by default, got %v", cfg.Logging.DebugEnabled)
	}
	if cfg.Logging.File != "debug.log" {
		t.Errorf("Expected log file 'debug.log', got '%s'", cfg.Logging.File)
	}
	if cfg.Logging.MaxSizeMB != 10 {
		t.Errorf("Expected max_size_mb 10, got %d", cfg.Logging.MaxSizeMB)
	}
	if cfg.Logging.MaxBackups != 3 {
		t.Errorf("Expected max_backups 3, got %d", cfg.Logging.MaxBackups)
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
		Privacy: PrivacyConfig{},
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
		UserMode: "single",
		Auth: AuthConfig{
			Provider:               "userpass",
			SessionExpiryDays:      7,
			LockoutThreshold:       5,
			LockoutDurationMinutes: 15,
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
	os.Setenv("NOODEXX_DEBUG_ENABLED", "false")
	os.Setenv("NOODEXX_LOG_FILE", "custom.log")
	os.Setenv("NOODEXX_SERVER_PORT", "9000")
	defer func() {
		os.Unsetenv("NOODEXX_PROVIDER")
		os.Unsetenv("NOODEXX_OPENAI_KEY")
		os.Unsetenv("NOODEXX_PRIVACY_MODE")
		os.Unsetenv("NOODEXX_LOG_LEVEL")
		os.Unsetenv("NOODEXX_DEBUG_ENABLED")
		os.Unsetenv("NOODEXX_LOG_FILE")
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
	if cfg.Logging.DebugEnabled != false {
		t.Errorf("Expected debug_enabled false, got %v", cfg.Logging.DebugEnabled)
	}
	if cfg.Logging.File != "custom.log" {
		t.Errorf("Expected log file 'custom.log', got '%s'", cfg.Logging.File)
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
				Privacy:    PrivacyConfig{DefaultToLocal: true},
				Logging:    LoggingConfig{Level: "info"},
				Guardrails: GuardrailsConfig{PIIDetection: "normal"},
				Server:     ServerConfig{Port: 8080},
				UserMode:   "single",
				Auth: AuthConfig{
					Provider:               "userpass",
					SessionExpiryDays:      7,
					LockoutThreshold:       5,
					LockoutDurationMinutes: 15,
				},
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
				Privacy:    PrivacyConfig{DefaultToLocal: true},
				Logging:    LoggingConfig{Level: "info"},
				Guardrails: GuardrailsConfig{PIIDetection: "normal"},
				Server:     ServerConfig{Port: 8080},
				UserMode:   "single",
				Auth: AuthConfig{
					Provider:               "userpass",
					SessionExpiryDays:      7,
					LockoutThreshold:       5,
					LockoutDurationMinutes: 15,
				},
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
				Privacy:    PrivacyConfig{DefaultToLocal: true},
				Logging:    LoggingConfig{Level: "info"},
				Guardrails: GuardrailsConfig{PIIDetection: "normal"},
				Server:     ServerConfig{Port: 8080},
				UserMode:   "single",
				Auth: AuthConfig{
					Provider:               "userpass",
					SessionExpiryDays:      7,
					LockoutThreshold:       5,
					LockoutDurationMinutes: 15,
				},
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
				Privacy:    PrivacyConfig{DefaultToLocal: false},
				Logging:    LoggingConfig{Level: "info"},
				Guardrails: GuardrailsConfig{PIIDetection: "normal"},
				Server:     ServerConfig{Port: 8080},
				UserMode:   "single",
				Auth: AuthConfig{
					Provider:               "userpass",
					SessionExpiryDays:      7,
					LockoutThreshold:       5,
					LockoutDurationMinutes: 15,
				},
			},
			expectError: true,
		},
		{
			name: "Anthropic without API key",
			cfg: &Config{
				Provider: ProviderConfig{
					Type: "anthropic",
				},
				Privacy:    PrivacyConfig{DefaultToLocal: false},
				Logging:    LoggingConfig{Level: "info"},
				Guardrails: GuardrailsConfig{PIIDetection: "normal"},
				Server:     ServerConfig{Port: 8080},
				UserMode:   "single",
				Auth: AuthConfig{
					Provider:               "userpass",
					SessionExpiryDays:      7,
					LockoutThreshold:       5,
					LockoutDurationMinutes: 15,
				},
			},
			expectError: true,
		},
		{
			name: "Unknown provider type",
			cfg: &Config{
				Provider: ProviderConfig{
					Type: "unknown",
				},
				Privacy:    PrivacyConfig{DefaultToLocal: false},
				Logging:    LoggingConfig{Level: "info"},
				Guardrails: GuardrailsConfig{PIIDetection: "normal"},
				Server:     ServerConfig{Port: 8080},
				UserMode:   "single",
				Auth: AuthConfig{
					Provider:               "userpass",
					SessionExpiryDays:      7,
					LockoutThreshold:       5,
					LockoutDurationMinutes: 15,
				},
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
		Privacy: PrivacyConfig{DefaultToLocal: true},
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
		UserMode: "single",
		Auth: AuthConfig{
			Provider:               "userpass",
			SessionExpiryDays:      7,
			LockoutThreshold:       5,
			LockoutDurationMinutes: 15,
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

func TestBackwardCompatibility_MissingDebugEnabled(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create a config file without debug_enabled field (simulating old config)
	oldConfigJSON := `{
		"provider": {
			"type": "ollama",
			"ollama_endpoint": "http://localhost:11434",
			"ollama_embed_model": "nomic-embed-text",
			"ollama_chat_model": "llama3.2"
		},
		"privacy": {
			"enabled": true
		},
		"folders": [],
		"logging": {
			"level": "info"
		},
		"guardrails": {
			"max_file_size_mb": 10,
			"allowed_extensions": [".txt", ".md"],
			"max_concurrent": 3,
			"pii_detection": "normal",
			"auto_summarize": true
		},
		"server": {
			"port": 8080,
			"bind_address": "127.0.0.1"
		}
	}`

	if err := os.WriteFile(configPath, []byte(oldConfigJSON), 0600); err != nil {
		t.Fatalf("Failed to write old config: %v", err)
	}

	// Load config
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify that debug_enabled defaults to true for backward compatibility
	if cfg.Logging.DebugEnabled != true {
		t.Errorf("Expected debug_enabled to default to true for backward compatibility, got %v", cfg.Logging.DebugEnabled)
	}

	// Verify other defaults are applied
	if cfg.Logging.File != "debug.log" {
		t.Errorf("Expected file to default to 'debug.log', got '%s'", cfg.Logging.File)
	}
	if cfg.Logging.MaxSizeMB != 10 {
		t.Errorf("Expected max_size_mb to default to 10, got %d", cfg.Logging.MaxSizeMB)
	}
	if cfg.Logging.MaxBackups != 3 {
		t.Errorf("Expected max_backups to default to 3, got %d", cfg.Logging.MaxBackups)
	}
}

func TestValidate_LogLevel(t *testing.T) {
	tests := []struct {
		name        string
		level       string
		expectError bool
	}{
		{"Valid debug level", "debug", false},
		{"Valid info level", "info", false},
		{"Valid warn level", "warn", false},
		{"Valid error level", "error", false},
		{"Invalid level", "invalid", true},
		{"Empty level", "", true},
		{"Uppercase level", "INFO", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Provider: ProviderConfig{
					Type:           "ollama",
					OllamaEndpoint: "http://localhost:11434",
				},
				Privacy: PrivacyConfig{DefaultToLocal: true},
				Logging: LoggingConfig{
					Level:      tt.level,
					File:       "debug.log",
					MaxSizeMB:  10,
					MaxBackups: 3,
				},
				Guardrails: GuardrailsConfig{PIIDetection: "normal"},
				Server:     ServerConfig{Port: 8080},
				UserMode:   "single",
				Auth: AuthConfig{
					Provider:               "userpass",
					SessionExpiryDays:      7,
					LockoutThreshold:       5,
					LockoutDurationMinutes: 15,
				},
			}

			err := cfg.Validate()
			if tt.expectError && err == nil {
				t.Error("Expected validation error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestLoad_UserModeAndAuthDefaults(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Load config (should create default)
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify user_mode defaults
	if cfg.UserMode != "single" {
		t.Errorf("Expected user_mode 'single', got '%s'", cfg.UserMode)
	}

	// Verify auth defaults
	if cfg.Auth.Provider != "userpass" {
		t.Errorf("Expected auth provider 'userpass', got '%s'", cfg.Auth.Provider)
	}
	if cfg.Auth.SessionExpiryDays != 7 {
		t.Errorf("Expected session_expiry_days 7, got %d", cfg.Auth.SessionExpiryDays)
	}
	if cfg.Auth.LockoutThreshold != 5 {
		t.Errorf("Expected lockout_threshold 5, got %d", cfg.Auth.LockoutThreshold)
	}
	if cfg.Auth.LockoutDurationMinutes != 15 {
		t.Errorf("Expected lockout_duration_minutes 15, got %d", cfg.Auth.LockoutDurationMinutes)
	}
}

func TestEnvOverrides_UserModeAndAuth(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Set environment variables
	os.Setenv("NOODEXX_USER_MODE", "multi")
	os.Setenv("NOODEXX_AUTH_PROVIDER", "mfa")
	defer func() {
		os.Unsetenv("NOODEXX_USER_MODE")
		os.Unsetenv("NOODEXX_AUTH_PROVIDER")
	}()

	// Load config
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify environment overrides
	if cfg.UserMode != "multi" {
		t.Errorf("Expected user_mode 'multi', got '%s'", cfg.UserMode)
	}
	if cfg.Auth.Provider != "mfa" {
		t.Errorf("Expected auth provider 'mfa', got '%s'", cfg.Auth.Provider)
	}
}

func TestValidate_UserMode(t *testing.T) {
	tests := []struct {
		name        string
		userMode    string
		expectError bool
	}{
		{"Valid single mode", "single", false},
		{"Valid multi mode", "multi", false},
		{"Invalid mode", "invalid", true},
		{"Empty mode", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Provider: ProviderConfig{
					Type:           "ollama",
					OllamaEndpoint: "http://localhost:11434",
				},
				Privacy:    PrivacyConfig{DefaultToLocal: true},
				Logging:    LoggingConfig{Level: "info"},
				Guardrails: GuardrailsConfig{PIIDetection: "normal"},
				Server:     ServerConfig{Port: 8080},
				UserMode:   tt.userMode,
				Auth: AuthConfig{
					Provider:               "userpass",
					SessionExpiryDays:      7,
					LockoutThreshold:       5,
					LockoutDurationMinutes: 15,
				},
			}

			err := cfg.Validate()
			if tt.expectError && err == nil {
				t.Error("Expected validation error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestValidate_AuthProvider(t *testing.T) {
	tests := []struct {
		name        string
		provider    string
		expectError bool
	}{
		{"Valid userpass provider", "userpass", false},
		{"Valid mfa provider", "mfa", false},
		{"Valid sso provider", "sso", false},
		{"Invalid provider", "invalid", true},
		{"Empty provider", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Provider: ProviderConfig{
					Type:           "ollama",
					OllamaEndpoint: "http://localhost:11434",
				},
				Privacy:    PrivacyConfig{DefaultToLocal: true},
				Logging:    LoggingConfig{Level: "info"},
				Guardrails: GuardrailsConfig{PIIDetection: "normal"},
				Server:     ServerConfig{Port: 8080},
				UserMode:   "single",
				Auth: AuthConfig{
					Provider:               tt.provider,
					SessionExpiryDays:      7,
					LockoutThreshold:       5,
					LockoutDurationMinutes: 15,
				},
			}

			err := cfg.Validate()
			if tt.expectError && err == nil {
				t.Error("Expected validation error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestProviderConfig_ValidateLocal(t *testing.T) {
	tests := []struct {
		name        string
		cfg         ProviderConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid Ollama configuration",
			cfg: ProviderConfig{
				Type:             "ollama",
				OllamaEndpoint:   "http://localhost:11434",
				OllamaEmbedModel: "nomic-embed-text",
				OllamaChatModel:  "llama3.2",
			},
			expectError: false,
		},
		{
			name: "Valid Ollama with 127.0.0.1",
			cfg: ProviderConfig{
				Type:             "ollama",
				OllamaEndpoint:   "http://127.0.0.1:11434",
				OllamaEmbedModel: "nomic-embed-text",
				OllamaChatModel:  "llama3.2",
			},
			expectError: false,
		},
		{
			name: "Empty type is valid (not configured)",
			cfg: ProviderConfig{
				Type: "",
			},
			expectError: false,
		},
		{
			name: "Invalid provider type for local",
			cfg: ProviderConfig{
				Type:             "openai",
				OpenAIKey:        "test-key",
				OpenAIEmbedModel: "text-embedding-3-small",
				OpenAIChatModel:  "gpt-4",
			},
			expectError: true,
			errorMsg:    "local provider must be Ollama",
		},
		{
			name: "Missing Ollama endpoint",
			cfg: ProviderConfig{
				Type:             "ollama",
				OllamaEndpoint:   "",
				OllamaEmbedModel: "nomic-embed-text",
				OllamaChatModel:  "llama3.2",
			},
			expectError: true,
			errorMsg:    "Ollama endpoint is required",
		},
		{
			name: "Non-localhost endpoint",
			cfg: ProviderConfig{
				Type:             "ollama",
				OllamaEndpoint:   "http://192.168.1.100:11434",
				OllamaEmbedModel: "nomic-embed-text",
				OllamaChatModel:  "llama3.2",
			},
			expectError: true,
			errorMsg:    "local provider must use localhost endpoint",
		},
		{
			name: "Missing embed model",
			cfg: ProviderConfig{
				Type:             "ollama",
				OllamaEndpoint:   "http://localhost:11434",
				OllamaEmbedModel: "",
				OllamaChatModel:  "llama3.2",
			},
			expectError: true,
			errorMsg:    "Ollama models are required",
		},
		{
			name: "Missing chat model",
			cfg: ProviderConfig{
				Type:             "ollama",
				OllamaEndpoint:   "http://localhost:11434",
				OllamaEmbedModel: "nomic-embed-text",
				OllamaChatModel:  "",
			},
			expectError: true,
			errorMsg:    "Ollama models are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.ValidateLocal()
			if tt.expectError {
				if err == nil {
					t.Error("Expected validation error, got nil")
				} else if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestProviderConfig_ValidateCloud(t *testing.T) {
	tests := []struct {
		name        string
		cfg         ProviderConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid OpenAI configuration",
			cfg: ProviderConfig{
				Type:             "openai",
				OpenAIKey:        "sk-test-key",
				OpenAIEmbedModel: "text-embedding-3-small",
				OpenAIChatModel:  "gpt-4",
			},
			expectError: false,
		},
		{
			name: "Valid Anthropic configuration",
			cfg: ProviderConfig{
				Type:               "anthropic",
				AnthropicKey:       "sk-ant-test-key",
				AnthropicChatModel: "claude-3-opus-20240229",
			},
			expectError: false,
		},
		{
			name: "Empty type is valid (not configured)",
			cfg: ProviderConfig{
				Type: "",
			},
			expectError: false,
		},
		{
			name: "Invalid provider type for cloud",
			cfg: ProviderConfig{
				Type:             "ollama",
				OllamaEndpoint:   "http://localhost:11434",
				OllamaEmbedModel: "nomic-embed-text",
				OllamaChatModel:  "llama3.2",
			},
			expectError: true,
			errorMsg:    "invalid cloud provider type: ollama",
		},
		{
			name: "OpenAI missing API key",
			cfg: ProviderConfig{
				Type:             "openai",
				OpenAIKey:        "",
				OpenAIEmbedModel: "text-embedding-3-small",
				OpenAIChatModel:  "gpt-4",
			},
			expectError: true,
			errorMsg:    "OpenAI API key is required",
		},
		{
			name: "OpenAI missing embed model",
			cfg: ProviderConfig{
				Type:             "openai",
				OpenAIKey:        "sk-test-key",
				OpenAIEmbedModel: "",
				OpenAIChatModel:  "gpt-4",
			},
			expectError: true,
			errorMsg:    "OpenAI models are required",
		},
		{
			name: "OpenAI missing chat model",
			cfg: ProviderConfig{
				Type:             "openai",
				OpenAIKey:        "sk-test-key",
				OpenAIEmbedModel: "text-embedding-3-small",
				OpenAIChatModel:  "",
			},
			expectError: true,
			errorMsg:    "OpenAI models are required",
		},
		{
			name: "Anthropic missing API key",
			cfg: ProviderConfig{
				Type:               "anthropic",
				AnthropicKey:       "",
				AnthropicChatModel: "claude-3-opus-20240229",
			},
			expectError: true,
			errorMsg:    "Anthropic API key is required",
		},
		{
			name: "Anthropic missing chat model",
			cfg: ProviderConfig{
				Type:               "anthropic",
				AnthropicKey:       "sk-ant-test-key",
				AnthropicChatModel: "",
			},
			expectError: true,
			errorMsg:    "Anthropic chat model is required",
		},
		{
			name: "Unknown cloud provider type",
			cfg: ProviderConfig{
				Type: "unknown",
			},
			expectError: true,
			errorMsg:    "invalid cloud provider type: unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.ValidateCloud()
			if tt.expectError {
				if err == nil {
					t.Error("Expected validation error, got nil")
				} else if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestPrivacyConfig_ValidateRAGPolicy(t *testing.T) {
	tests := []struct {
		name        string
		cfg         PrivacyConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid no_rag policy",
			cfg: PrivacyConfig{
				DefaultToLocal: true,
				CloudRAGPolicy: "no_rag",
			},
			expectError: false,
		},
		{
			name: "Valid allow_rag policy",
			cfg: PrivacyConfig{
				DefaultToLocal: false,
				CloudRAGPolicy: "allow_rag",
			},
			expectError: false,
		},
		{
			name: "Invalid RAG policy",
			cfg: PrivacyConfig{
				DefaultToLocal: true,
				CloudRAGPolicy: "invalid",
			},
			expectError: true,
			errorMsg:    "invalid RAG policy: invalid (must be 'no_rag' or 'allow_rag')",
		},
		{
			name: "Empty RAG policy",
			cfg: PrivacyConfig{
				DefaultToLocal: true,
				CloudRAGPolicy: "",
			},
			expectError: false, // Empty is valid, will be defaulted
		},
		{
			name: "Case-sensitive validation",
			cfg: PrivacyConfig{
				DefaultToLocal: true,
				CloudRAGPolicy: "NO_RAG",
			},
			expectError: true,
			errorMsg:    "invalid RAG policy: NO_RAG (must be 'no_rag' or 'allow_rag')",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.ValidateRAGPolicy()
			if tt.expectError {
				if err == nil {
					t.Error("Expected validation error, got nil")
				} else if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}
