package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestMigration_OllamaToLocalProvider tests migration from old Ollama config to new dual-provider format
func TestMigration_OllamaToLocalProvider(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create an old-style config with Ollama provider
	oldConfig := `{
		"provider": {
			"type": "ollama",
			"ollama_endpoint": "http://localhost:11434",
			"ollama_embed_model": "custom-embed",
			"ollama_chat_model": "custom-chat"
		},
		"privacy": {
			"enabled": true
		},
		"folders": [],
		"logging": {"level": "info"},
		"guardrails": {"pii_detection": "normal"},
		"server": {"port": 8080, "bind_address": "127.0.0.1"},
		"user_mode": "single",
		"auth": {
			"provider": "userpass",
			"session_expiry_days": 7,
			"lockout_threshold": 5,
			"lockout_duration_minutes": 15
		}
	}`

	if err := os.WriteFile(configPath, []byte(oldConfig), 0600); err != nil {
		t.Fatalf("Failed to write old config: %v", err)
	}

	// Load config (should trigger migration)
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify migration results
	if cfg.LocalProvider.Type != "ollama" {
		t.Errorf("Expected LocalProvider.Type 'ollama', got '%s'", cfg.LocalProvider.Type)
	}
	if cfg.LocalProvider.OllamaEndpoint != "http://localhost:11434" {
		t.Errorf("Expected LocalProvider.OllamaEndpoint 'http://localhost:11434', got '%s'", cfg.LocalProvider.OllamaEndpoint)
	}
	if cfg.LocalProvider.OllamaEmbedModel != "custom-embed" {
		t.Errorf("Expected LocalProvider.OllamaEmbedModel 'custom-embed', got '%s'", cfg.LocalProvider.OllamaEmbedModel)
	}
	if cfg.LocalProvider.OllamaChatModel != "custom-chat" {
		t.Errorf("Expected LocalProvider.OllamaChatModel 'custom-chat', got '%s'", cfg.LocalProvider.OllamaChatModel)
	}

	// Verify CloudProvider is empty
	if cfg.CloudProvider.Type != "" {
		t.Errorf("Expected CloudProvider.Type to be empty, got '%s'", cfg.CloudProvider.Type)
	}

	// Verify privacy settings migrated
	if cfg.Privacy.UseLocalAI != true {
		t.Errorf("Expected Privacy.UseLocalAI true (from old enabled=true), got %v", cfg.Privacy.UseLocalAI)
	}
	if cfg.Privacy.CloudRAGPolicy != "no_rag" {
		t.Errorf("Expected Privacy.CloudRAGPolicy 'no_rag', got '%s'", cfg.Privacy.CloudRAGPolicy)
	}
}

// TestMigration_OpenAIToCloudProvider tests migration from old OpenAI config to new dual-provider format
func TestMigration_OpenAIToCloudProvider(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create an old-style config with OpenAI provider
	oldConfig := `{
		"provider": {
			"type": "openai",
			"openai_key": "sk-test-key",
			"openai_embed_model": "text-embedding-3-small",
			"openai_chat_model": "gpt-4"
		},
		"privacy": {
			"enabled": false
		},
		"folders": [],
		"logging": {"level": "info"},
		"guardrails": {"pii_detection": "normal"},
		"server": {"port": 8080, "bind_address": "127.0.0.1"},
		"user_mode": "single",
		"auth": {
			"provider": "userpass",
			"session_expiry_days": 7,
			"lockout_threshold": 5,
			"lockout_duration_minutes": 15
		}
	}`

	if err := os.WriteFile(configPath, []byte(oldConfig), 0600); err != nil {
		t.Fatalf("Failed to write old config: %v", err)
	}

	// Load config (should trigger migration)
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify migration results
	if cfg.CloudProvider.Type != "openai" {
		t.Errorf("Expected CloudProvider.Type 'openai', got '%s'", cfg.CloudProvider.Type)
	}
	if cfg.CloudProvider.OpenAIKey != "sk-test-key" {
		t.Errorf("Expected CloudProvider.OpenAIKey 'sk-test-key', got '%s'", cfg.CloudProvider.OpenAIKey)
	}
	if cfg.CloudProvider.OpenAIEmbedModel != "text-embedding-3-small" {
		t.Errorf("Expected CloudProvider.OpenAIEmbedModel 'text-embedding-3-small', got '%s'", cfg.CloudProvider.OpenAIEmbedModel)
	}
	if cfg.CloudProvider.OpenAIChatModel != "gpt-4" {
		t.Errorf("Expected CloudProvider.OpenAIChatModel 'gpt-4', got '%s'", cfg.CloudProvider.OpenAIChatModel)
	}

	// Verify LocalProvider has defaults
	if cfg.LocalProvider.Type != "ollama" {
		t.Errorf("Expected LocalProvider.Type 'ollama' (default), got '%s'", cfg.LocalProvider.Type)
	}

	// Verify privacy settings migrated
	if cfg.Privacy.UseLocalAI != false {
		t.Errorf("Expected Privacy.UseLocalAI false (from old enabled=false), got %v", cfg.Privacy.UseLocalAI)
	}
	if cfg.Privacy.CloudRAGPolicy != "no_rag" {
		t.Errorf("Expected Privacy.CloudRAGPolicy 'no_rag', got '%s'", cfg.Privacy.CloudRAGPolicy)
	}
}

// TestMigration_AnthropicToCloudProvider tests migration from old Anthropic config to new dual-provider format
func TestMigration_AnthropicToCloudProvider(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create an old-style config with Anthropic provider
	oldConfig := `{
		"provider": {
			"type": "anthropic",
			"anthropic_key": "sk-ant-test-key",
			"anthropic_chat_model": "claude-3-opus-20240229"
		},
		"privacy": {
			"enabled": false
		},
		"folders": [],
		"logging": {"level": "info"},
		"guardrails": {"pii_detection": "normal"},
		"server": {"port": 8080, "bind_address": "127.0.0.1"},
		"user_mode": "single",
		"auth": {
			"provider": "userpass",
			"session_expiry_days": 7,
			"lockout_threshold": 5,
			"lockout_duration_minutes": 15
		}
	}`

	if err := os.WriteFile(configPath, []byte(oldConfig), 0600); err != nil {
		t.Fatalf("Failed to write old config: %v", err)
	}

	// Load config (should trigger migration)
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify migration results
	if cfg.CloudProvider.Type != "anthropic" {
		t.Errorf("Expected CloudProvider.Type 'anthropic', got '%s'", cfg.CloudProvider.Type)
	}
	if cfg.CloudProvider.AnthropicKey != "sk-ant-test-key" {
		t.Errorf("Expected CloudProvider.AnthropicKey 'sk-ant-test-key', got '%s'", cfg.CloudProvider.AnthropicKey)
	}
	if cfg.CloudProvider.AnthropicChatModel != "claude-3-opus-20240229" {
		t.Errorf("Expected CloudProvider.AnthropicChatModel 'claude-3-opus-20240229', got '%s'", cfg.CloudProvider.AnthropicChatModel)
	}

	// Verify LocalProvider has defaults
	if cfg.LocalProvider.Type != "ollama" {
		t.Errorf("Expected LocalProvider.Type 'ollama' (default), got '%s'", cfg.LocalProvider.Type)
	}

	// Verify privacy settings migrated
	if cfg.Privacy.UseLocalAI != false {
		t.Errorf("Expected Privacy.UseLocalAI false (from old enabled=false), got %v", cfg.Privacy.UseLocalAI)
	}
	if cfg.Privacy.CloudRAGPolicy != "no_rag" {
		t.Errorf("Expected Privacy.CloudRAGPolicy 'no_rag', got '%s'", cfg.Privacy.CloudRAGPolicy)
	}
}

// TestMigration_NewConfigNotMigrated tests that new configs are not migrated
func TestMigration_NewConfigNotMigrated(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create a new-style config with both providers
	newConfig := `{
		"provider": {
			"type": "ollama",
			"ollama_endpoint": "http://localhost:11434",
			"ollama_embed_model": "old-embed",
			"ollama_chat_model": "old-chat"
		},
		"local_provider": {
			"type": "ollama",
			"ollama_endpoint": "http://localhost:11434",
			"ollama_embed_model": "new-local-embed",
			"ollama_chat_model": "new-local-chat"
		},
		"cloud_provider": {
			"type": "openai",
			"openai_key": "sk-new-key",
			"openai_embed_model": "text-embedding-3-small",
			"openai_chat_model": "gpt-4"
		},
		"privacy": {
			"enabled": true,
			"use_local_ai": false,
			"cloud_rag_policy": "allow_rag"
		},
		"folders": [],
		"logging": {"level": "info"},
		"guardrails": {"pii_detection": "normal"},
		"server": {"port": 8080, "bind_address": "127.0.0.1"},
		"user_mode": "single",
		"auth": {
			"provider": "userpass",
			"session_expiry_days": 7,
			"lockout_threshold": 5,
			"lockout_duration_minutes": 15
		}
	}`

	if err := os.WriteFile(configPath, []byte(newConfig), 0600); err != nil {
		t.Fatalf("Failed to write new config: %v", err)
	}

	// Load config (should NOT trigger migration)
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify new config values are preserved (not overwritten by migration)
	if cfg.LocalProvider.OllamaEmbedModel != "new-local-embed" {
		t.Errorf("Expected LocalProvider.OllamaEmbedModel 'new-local-embed', got '%s'", cfg.LocalProvider.OllamaEmbedModel)
	}
	if cfg.CloudProvider.Type != "openai" {
		t.Errorf("Expected CloudProvider.Type 'openai', got '%s'", cfg.CloudProvider.Type)
	}
	if cfg.Privacy.UseLocalAI != false {
		t.Errorf("Expected Privacy.UseLocalAI false, got %v", cfg.Privacy.UseLocalAI)
	}
	if cfg.Privacy.CloudRAGPolicy != "allow_rag" {
		t.Errorf("Expected Privacy.CloudRAGPolicy 'allow_rag', got '%s'", cfg.Privacy.CloudRAGPolicy)
	}
}

// TestMigration_RoundTrip tests that migrated config can be saved and loaded again
func TestMigration_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create an old-style config
	oldConfig := `{
		"provider": {
			"type": "ollama",
			"ollama_endpoint": "http://localhost:11434",
			"ollama_embed_model": "test-embed",
			"ollama_chat_model": "test-chat"
		},
		"privacy": {
			"enabled": true
		},
		"folders": ["/test"],
		"logging": {"level": "info"},
		"guardrails": {"pii_detection": "normal"},
		"server": {"port": 8080, "bind_address": "127.0.0.1"},
		"user_mode": "single",
		"auth": {
			"provider": "userpass",
			"session_expiry_days": 7,
			"lockout_threshold": 5,
			"lockout_duration_minutes": 15
		}
	}`

	if err := os.WriteFile(configPath, []byte(oldConfig), 0600); err != nil {
		t.Fatalf("Failed to write old config: %v", err)
	}

	// Load config (should trigger migration)
	cfg1, err := Load(configPath)
	if err != nil {
		t.Fatalf("First Load() failed: %v", err)
	}

	// Save the migrated config
	if err := cfg1.Save(configPath); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Load again (should NOT trigger migration this time)
	cfg2, err := Load(configPath)
	if err != nil {
		t.Fatalf("Second Load() failed: %v", err)
	}

	// Verify both configs are equivalent
	if cfg1.LocalProvider.OllamaEmbedModel != cfg2.LocalProvider.OllamaEmbedModel {
		t.Errorf("LocalProvider.OllamaEmbedModel mismatch: '%s' vs '%s'",
			cfg1.LocalProvider.OllamaEmbedModel, cfg2.LocalProvider.OllamaEmbedModel)
	}
	if cfg1.Privacy.UseLocalAI != cfg2.Privacy.UseLocalAI {
		t.Errorf("Privacy.UseLocalAI mismatch: %v vs %v",
			cfg1.Privacy.UseLocalAI, cfg2.Privacy.UseLocalAI)
	}
	if cfg1.Privacy.CloudRAGPolicy != cfg2.Privacy.CloudRAGPolicy {
		t.Errorf("Privacy.CloudRAGPolicy mismatch: '%s' vs '%s'",
			cfg1.Privacy.CloudRAGPolicy, cfg2.Privacy.CloudRAGPolicy)
	}

	// Verify the saved config has the new format
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read saved config: %v", err)
	}

	var savedConfig map[string]interface{}
	if err := json.Unmarshal(data, &savedConfig); err != nil {
		t.Fatalf("Failed to parse saved config: %v", err)
	}

	// Check that new fields exist in saved config
	if _, ok := savedConfig["local_provider"]; !ok {
		t.Error("Saved config missing 'local_provider' field")
	}
	if _, ok := savedConfig["cloud_provider"]; !ok {
		t.Error("Saved config missing 'cloud_provider' field")
	}
	if privacy, ok := savedConfig["privacy"].(map[string]interface{}); ok {
		if _, ok := privacy["use_local_ai"]; !ok {
			t.Error("Saved config missing 'privacy.use_local_ai' field")
		}
		if _, ok := privacy["cloud_rag_policy"]; !ok {
			t.Error("Saved config missing 'privacy.cloud_rag_policy' field")
		}
	} else {
		t.Error("Saved config has invalid 'privacy' structure")
	}
}
