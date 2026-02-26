package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Config holds all application configuration
type Config struct {
	Provider   ProviderConfig   `json:"provider"`
	Privacy    PrivacyConfig    `json:"privacy"`
	Folders    []string         `json:"folders"`
	Logging    LoggingConfig    `json:"logging"`
	Guardrails GuardrailsConfig `json:"guardrails"`
	Server     ServerConfig     `json:"server"`
	UserMode   string           `json:"user_mode"` // "single" or "multi"
	Auth       AuthConfig       `json:"auth"`
}

// ProviderConfig configures the LLM provider
type ProviderConfig struct {
	Type                string `json:"type"` // "ollama", "openai", "anthropic"
	OllamaEndpoint      string `json:"ollama_endpoint"`
	OllamaEmbedModel    string `json:"ollama_embed_model"`
	OllamaChatModel     string `json:"ollama_chat_model"`
	OpenAIKey           string `json:"openai_key"`
	OpenAIEmbedModel    string `json:"openai_embed_model"`
	OpenAIChatModel     string `json:"openai_chat_model"`
	AnthropicKey        string `json:"anthropic_key"`
	AnthropicEmbedModel string `json:"anthropic_embed_model"`
	AnthropicChatModel  string `json:"anthropic_chat_model"`
}

// PrivacyConfig controls privacy mode
type PrivacyConfig struct {
	Enabled bool `json:"enabled"`
}

// LoggingConfig controls logging behavior
type LoggingConfig struct {
	Level        string `json:"level"`         // "debug", "info", "warn", "error"
	DebugEnabled bool   `json:"debug_enabled"` // Enable debug file logging
	File         string `json:"file"`          // Debug log file path
	MaxSizeMB    int    `json:"max_size_mb"`   // Max file size before rotation
	MaxBackups   int    `json:"max_backups"`   // Number of backup files to keep
}

// GuardrailsConfig controls ingestion safety
type GuardrailsConfig struct {
	MaxFileSizeMB     int      `json:"max_file_size_mb"`
	AllowedExtensions []string `json:"allowed_extensions"`
	MaxConcurrent     int      `json:"max_concurrent"`
	PIIDetection      string   `json:"pii_detection"` // "strict", "normal", "off"
	AutoSummarize     bool     `json:"auto_summarize"`
}

// ServerConfig controls HTTP server
type ServerConfig struct {
	Port        int    `json:"port"`
	BindAddress string `json:"bind_address"`
}

// AuthConfig controls authentication behavior
type AuthConfig struct {
	Provider               string `json:"provider"`                 // "userpass", "mfa", "sso"
	SessionExpiryDays      int    `json:"session_expiry_days"`      // Default: 7
	LockoutThreshold       int    `json:"lockout_threshold"`        // Default: 5
	LockoutDurationMinutes int    `json:"lockout_duration_minutes"` // Default: 15
}

// Load reads configuration from file and environment
func Load(path string) (*Config, error) {
	// Default configuration
	cfg := &Config{
		Provider: ProviderConfig{
			Type:             "ollama",
			OllamaEndpoint:   "http://localhost:11434",
			OllamaEmbedModel: "nomic-embed-text",
			OllamaChatModel:  "llama3.2",
		},
		Privacy: PrivacyConfig{
			Enabled: true,
		},
		Folders: []string{},
		Logging: LoggingConfig{
			Level:        "info",
			DebugEnabled: true,
			File:         "debug.log",
			MaxSizeMB:    10,
			MaxBackups:   3,
		},
		Guardrails: GuardrailsConfig{
			MaxFileSizeMB:     10,
			AllowedExtensions: []string{".txt", ".md", ".pdf", ".html"},
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

	// Load from file if exists
	if _, err := os.Stat(path); err == nil {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	} else {
		// Create default config file
		if err := cfg.Save(path); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
	}

	// Override with environment variables
	cfg.applyEnvOverrides()

	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// Save writes configuration to file
func (c *Config) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// applyEnvOverrides applies environment variable overrides
func (c *Config) applyEnvOverrides() {
	if v := os.Getenv("NOODEXX_PROVIDER"); v != "" {
		c.Provider.Type = v
	}
	if v := os.Getenv("NOODEXX_OLLAMA_ENDPOINT"); v != "" {
		c.Provider.OllamaEndpoint = v
	}
	if v := os.Getenv("NOODEXX_OLLAMA_EMBED_MODEL"); v != "" {
		c.Provider.OllamaEmbedModel = v
	}
	if v := os.Getenv("NOODEXX_OLLAMA_CHAT_MODEL"); v != "" {
		c.Provider.OllamaChatModel = v
	}
	if v := os.Getenv("NOODEXX_OPENAI_KEY"); v != "" {
		c.Provider.OpenAIKey = v
	}
	if v := os.Getenv("NOODEXX_OPENAI_EMBED_MODEL"); v != "" {
		c.Provider.OpenAIEmbedModel = v
	}
	if v := os.Getenv("NOODEXX_OPENAI_CHAT_MODEL"); v != "" {
		c.Provider.OpenAIChatModel = v
	}
	if v := os.Getenv("NOODEXX_ANTHROPIC_KEY"); v != "" {
		c.Provider.AnthropicKey = v
	}
	if v := os.Getenv("NOODEXX_ANTHROPIC_EMBED_MODEL"); v != "" {
		c.Provider.AnthropicEmbedModel = v
	}
	if v := os.Getenv("NOODEXX_ANTHROPIC_CHAT_MODEL"); v != "" {
		c.Provider.AnthropicChatModel = v
	}
	if v := os.Getenv("NOODEXX_PRIVACY_MODE"); v == "true" {
		c.Privacy.Enabled = true
	} else if v == "false" {
		c.Privacy.Enabled = false
	}
	if v := os.Getenv("NOODEXX_LOG_LEVEL"); v != "" {
		c.Logging.Level = v
	}
	if v := os.Getenv("NOODEXX_DEBUG_ENABLED"); v != "" {
		if v == "true" {
			c.Logging.DebugEnabled = true
		} else if v == "false" {
			c.Logging.DebugEnabled = false
		}
	}
	if v := os.Getenv("NOODEXX_LOG_FILE"); v != "" {
		c.Logging.File = v
	}
	if v := os.Getenv("NOODEXX_SERVER_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &c.Server.Port)
	}
	if v := os.Getenv("NOODEXX_SERVER_BIND_ADDRESS"); v != "" {
		c.Server.BindAddress = v
	}
	if v := os.Getenv("NOODEXX_USER_MODE"); v != "" {
		c.UserMode = v
	}
	if v := os.Getenv("NOODEXX_AUTH_PROVIDER"); v != "" {
		c.Auth.Provider = v
	}
}

// Validate checks configuration validity
func (c *Config) Validate() error {
	// Privacy mode validation
	if c.Privacy.Enabled && c.Provider.Type != "ollama" {
		return fmt.Errorf("privacy mode requires Ollama provider")
	}

	// Provider validation
	switch c.Provider.Type {
	case "ollama":
		if c.Privacy.Enabled {
			// Validate that Ollama endpoint is localhost
			if !strings.HasPrefix(c.Provider.OllamaEndpoint, "http://localhost") &&
				!strings.HasPrefix(c.Provider.OllamaEndpoint, "http://127.0.0.1") {
				return fmt.Errorf("privacy mode requires localhost Ollama endpoint")
			}
		}
	case "openai":
		if c.Provider.OpenAIKey == "" {
			return fmt.Errorf("OpenAI API key is required")
		}
	case "anthropic":
		if c.Provider.AnthropicKey == "" {
			return fmt.Errorf("Anthropic API key is required")
		}
	default:
		return fmt.Errorf("unknown provider type: %s", c.Provider.Type)
	}

	// Server validation
	if c.Server.Port < 1024 && os.Geteuid() != 0 {
		return fmt.Errorf("privileged port %d requires root", c.Server.Port)
	}

	// Logging level validation
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", c.Logging.Level)
	}

	// PII detection validation
	validPII := map[string]bool{"strict": true, "normal": true, "off": true}
	if !validPII[c.Guardrails.PIIDetection] {
		return fmt.Errorf("invalid PII detection level: %s (must be strict, normal, or off)", c.Guardrails.PIIDetection)
	}

	// User mode validation
	if c.UserMode != "single" && c.UserMode != "multi" {
		return fmt.Errorf("invalid user_mode: %s (must be single or multi)", c.UserMode)
	}

	// Auth provider validation
	validAuthProviders := map[string]bool{"userpass": true, "mfa": true, "sso": true}
	if !validAuthProviders[c.Auth.Provider] {
		return fmt.Errorf("invalid auth provider: %s (must be userpass, mfa, or sso)", c.Auth.Provider)
	}

	return nil
}
