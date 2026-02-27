package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Config holds all application configuration
type Config struct {
	Provider      ProviderConfig   `json:"provider,omitempty"` // Legacy single provider (for backward compatibility, omit if empty)
	LocalProvider ProviderConfig   `json:"local_provider"`     // Local AI provider configuration
	CloudProvider ProviderConfig   `json:"cloud_provider"`     // Cloud AI provider configuration
	Privacy       PrivacyConfig    `json:"privacy"`
	Folders       []string         `json:"folders"`
	Logging       LoggingConfig    `json:"logging"`
	Guardrails    GuardrailsConfig `json:"guardrails"`
	Server        ServerConfig     `json:"server"`
	UserMode      string           `json:"user_mode"` // "single" or "multi"
	Auth          AuthConfig       `json:"auth"`
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
	DefaultToLocal bool   `json:"default_to_local"` // Privacy toggle state (true = local, false = cloud)
	CloudRAGPolicy string `json:"cloud_rag_policy"` // "no_rag" or "allow_rag"
}

// UnmarshalJSON implements custom JSON unmarshaling for backward compatibility
func (p *PrivacyConfig) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as new format first
	type PrivacyConfigAlias PrivacyConfig
	var newFormat PrivacyConfigAlias
	if err := json.Unmarshal(data, &newFormat); err == nil {
		*p = PrivacyConfig(newFormat)

		// Check if we need to migrate from old "enabled" field
		var oldFormat struct {
			Enabled        *bool  `json:"enabled"`
			DefaultToLocal *bool  `json:"default_to_local"`
			CloudRAGPolicy string `json:"cloud_rag_policy"`
		}
		if err := json.Unmarshal(data, &oldFormat); err == nil {
			// If "enabled" field exists but "default_to_local" doesn't, migrate
			if oldFormat.Enabled != nil && oldFormat.DefaultToLocal == nil {
				p.DefaultToLocal = *oldFormat.Enabled
			}
		}
		return nil
	}
	return nil
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
		// Set legacy Provider field for backward compatibility (points to local provider by default)
		Provider: ProviderConfig{
			Type:             "ollama",
			OllamaEndpoint:   "http://localhost:11434",
			OllamaEmbedModel: "nomic-embed-text",
			OllamaChatModel:  "llama3.2",
		},
		// LocalProvider defaults to Ollama
		LocalProvider: ProviderConfig{
			Type:             "ollama",
			OllamaEndpoint:   "http://localhost:11434",
			OllamaEmbedModel: "nomic-embed-text",
			OllamaChatModel:  "llama3.2",
		},
		// CloudProvider is empty by default (user must configure)
		CloudProvider: ProviderConfig{},
		Privacy: PrivacyConfig{
			DefaultToLocal: true,
			CloudRAGPolicy: "no_rag",
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

		// Unmarshal into a temporary struct to check what's in the file
		var fileCfg Config
		if err := json.Unmarshal(data, &fileCfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}

		// Check if debug_enabled was explicitly set in the file
		var rawConfig map[string]interface{}
		json.Unmarshal(data, &rawConfig)
		if logging, ok := rawConfig["logging"].(map[string]interface{}); ok {
			if _, hasDebugEnabled := logging["debug_enabled"]; !hasDebugEnabled {
				// debug_enabled not in file, default to true for backward compatibility
				fileCfg.Logging.DebugEnabled = true
			}
		} else {
			// No logging section, default to true
			fileCfg.Logging.DebugEnabled = true
		}

		// Copy file config over defaults
		cfg = &fileCfg

		// Apply defaults for any missing fields
		if cfg.Logging.Level == "" {
			cfg.Logging.Level = "info"
		}
		if cfg.Logging.File == "" {
			cfg.Logging.File = "debug.log"
		}
		if cfg.Logging.MaxSizeMB == 0 {
			cfg.Logging.MaxSizeMB = 10
		}
		if cfg.Logging.MaxBackups == 0 {
			cfg.Logging.MaxBackups = 3
		}
		if cfg.Server.Port == 0 {
			cfg.Server.Port = 8080
		}
		if cfg.Server.BindAddress == "" {
			cfg.Server.BindAddress = "127.0.0.1"
		}
		if cfg.UserMode == "" {
			cfg.UserMode = "single"
		}
		if cfg.Auth.Provider == "" {
			cfg.Auth.Provider = "userpass"
		}
		if cfg.Auth.SessionExpiryDays == 0 {
			cfg.Auth.SessionExpiryDays = 7
		}
		if cfg.Auth.LockoutThreshold == 0 {
			cfg.Auth.LockoutThreshold = 5
		}
		if cfg.Auth.LockoutDurationMinutes == 0 {
			cfg.Auth.LockoutDurationMinutes = 15
		}
		if cfg.Guardrails.MaxFileSizeMB == 0 {
			cfg.Guardrails.MaxFileSizeMB = 10
		}
		if cfg.Guardrails.MaxConcurrent == 0 {
			cfg.Guardrails.MaxConcurrent = 3
		}
		if cfg.Guardrails.PIIDetection == "" {
			cfg.Guardrails.PIIDetection = "normal"
		}
		if len(cfg.Guardrails.AllowedExtensions) == 0 {
			cfg.Guardrails.AllowedExtensions = []string{".txt", ".md", ".pdf", ".html"}
		}
		if cfg.Privacy.CloudRAGPolicy == "" {
			cfg.Privacy.CloudRAGPolicy = "no_rag"
		}

		// Migrate old single-provider config to dual-provider format if needed
		cfg.migrateFromLegacyConfig()
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

// MarshalJSON implements custom JSON marshaling
func (c *Config) MarshalJSON() ([]byte, error) {
	// Create a type alias to avoid infinite recursion
	type ConfigAlias Config

	// Marshal normally - include Provider field for backward compatibility
	return json.Marshal((*ConfigAlias)(c))
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
	// Skip validation of legacy Provider field if it's empty (dual-provider mode)
	if c.Provider.Type != "" {
		// Legacy provider validation (for backward compatibility)
		// Provider validation
		switch c.Provider.Type {
		case "ollama":
			// No additional validation needed for ollama
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

	// Privacy mode validation
	if c.Privacy.DefaultToLocal {
		// When privacy mode is enabled (default to local), check that provider is compatible
		// Check both LocalProvider (new) and Provider (legacy) for backward compatibility
		providerToCheck := c.LocalProvider
		if providerToCheck.Type == "" && c.Provider.Type != "" {
			// Use legacy Provider if LocalProvider is not set
			providerToCheck = c.Provider
		}

		// Only validate that the provider type is compatible with local mode
		// Don't require full configuration (models, etc.) as that's checked elsewhere
		if providerToCheck.Type != "" && providerToCheck.Type != "ollama" {
			return fmt.Errorf("privacy mode requires local provider (Ollama), got %s", providerToCheck.Type)
		}

		// If Ollama is configured, check that endpoint is localhost
		if providerToCheck.Type == "ollama" && providerToCheck.OllamaEndpoint != "" {
			if !strings.HasPrefix(providerToCheck.OllamaEndpoint, "http://localhost") &&
				!strings.HasPrefix(providerToCheck.OllamaEndpoint, "http://127.0.0.1") {
				return fmt.Errorf("privacy mode requires localhost endpoint, got %s", providerToCheck.OllamaEndpoint)
			}
		}
	}

	// RAG policy validation
	if err := c.Privacy.ValidateRAGPolicy(); err != nil {
		return err
	}

	return nil
}

// ValidateLocal validates local provider (Ollama) configuration
func (p *ProviderConfig) ValidateLocal() error {
	if p.Type == "" {
		return nil // Not configured is valid
	}
	if p.Type != "ollama" {
		return fmt.Errorf("local provider must be Ollama")
	}
	if p.OllamaEndpoint == "" {
		return fmt.Errorf("Ollama endpoint is required")
	}
	if !strings.HasPrefix(p.OllamaEndpoint, "http://localhost") &&
		!strings.HasPrefix(p.OllamaEndpoint, "http://127.0.0.1") {
		return fmt.Errorf("local provider must use localhost endpoint")
	}
	if p.OllamaEmbedModel == "" || p.OllamaChatModel == "" {
		return fmt.Errorf("Ollama models are required")
	}
	return nil
}

// ValidateCloud validates cloud provider (OpenAI/Anthropic) configuration
func (p *ProviderConfig) ValidateCloud() error {
	if p.Type == "" {
		return nil // Not configured is valid
	}
	switch p.Type {
	case "openai":
		if p.OpenAIKey == "" {
			return fmt.Errorf("OpenAI API key is required")
		}
		if p.OpenAIEmbedModel == "" || p.OpenAIChatModel == "" {
			return fmt.Errorf("OpenAI models are required")
		}
	case "anthropic":
		if p.AnthropicKey == "" {
			return fmt.Errorf("Anthropic API key is required")
		}
		if p.AnthropicChatModel == "" {
			return fmt.Errorf("Anthropic chat model is required")
		}
	default:
		return fmt.Errorf("invalid cloud provider type: %s", p.Type)
	}
	return nil
}

// ValidateRAGPolicy validates RAG policy configuration
func (p *PrivacyConfig) ValidateRAGPolicy() error {
	// Empty is valid (will be defaulted)
	if p.CloudRAGPolicy == "" {
		return nil
	}
	if p.CloudRAGPolicy != "no_rag" && p.CloudRAGPolicy != "allow_rag" {
		return fmt.Errorf("invalid RAG policy: %s (must be 'no_rag' or 'allow_rag')", p.CloudRAGPolicy)
	}
	return nil
}

// migrateFromLegacyConfig migrates old single-provider configuration to dual-provider format
func (c *Config) migrateFromLegacyConfig() {
	// Check if migration is needed (both new fields are empty but old Provider field has data)
	if c.LocalProvider.Type == "" && c.CloudProvider.Type == "" && c.Provider.Type != "" {
		// Migrate based on provider type
		if c.Provider.Type == "ollama" {
			// Ollama goes to local provider
			c.LocalProvider = c.Provider
		} else {
			// OpenAI/Anthropic go to cloud provider
			c.CloudProvider = c.Provider
		}

		// Set safe default for RAG policy
		if c.Privacy.CloudRAGPolicy == "" {
			c.Privacy.CloudRAGPolicy = "no_rag"
		}
	}

	// If LocalProvider is still empty after migration, set defaults
	if c.LocalProvider.Type == "" {
		c.LocalProvider = ProviderConfig{
			Type:             "ollama",
			OllamaEndpoint:   "http://localhost:11434",
			OllamaEmbedModel: "nomic-embed-text",
			OllamaChatModel:  "llama3.2",
		}
	}

	// Ensure CloudRAGPolicy has a valid default if not set
	if c.Privacy.CloudRAGPolicy == "" {
		c.Privacy.CloudRAGPolicy = "no_rag"
	}
}
