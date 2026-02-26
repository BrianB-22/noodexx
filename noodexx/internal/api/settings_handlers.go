package api

import (
	"encoding/json"
	"net/http"
	"noodexx/internal/config"
	"strconv"
)

// handleSaveSettings saves configuration changes to config.json
func (s *Server) handleSaveSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.logger.Debug("Received settings save request")

	// Parse form data
	if err := r.ParseForm(); err != nil {
		s.logger.Error("Failed to parse form: %v", err)
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	s.logger.Debug("Form data received: %v", r.Form)

	// Load current config
	cfg, err := config.Load(s.configPath)
	if err != nil {
		s.logger.Error("Failed to load config: %v", err)
		http.Error(w, "Failed to load config", http.StatusInternalServerError)
		return
	}

	s.logger.Debug("Current config loaded, privacy=%v, folders=%v", cfg.Privacy.Enabled, cfg.Folders)

	// Update privacy mode
	privacyValue := r.FormValue("privacy_enabled")
	s.logger.Debug("Privacy mode form value: '%s'", privacyValue)
	if privacyValue == "on" {
		cfg.Privacy.Enabled = true
	} else {
		cfg.Privacy.Enabled = false
	}
	s.logger.Debug("Privacy mode updated to: %v", cfg.Privacy.Enabled)

	// Update provider settings
	providerType := r.FormValue("provider_type")
	s.logger.Debug("Provider type from form: '%s'", providerType)
	if providerType != "" {
		cfg.Provider.Type = providerType
	}
	s.logger.Debug("Provider type set to: %s", cfg.Provider.Type)

	// Ollama settings
	if v := r.FormValue("ollama_endpoint"); v != "" {
		s.logger.Debug("Ollama endpoint: %s", v)
		cfg.Provider.OllamaEndpoint = v
	}
	if v := r.FormValue("ollama_embed_model"); v != "" {
		s.logger.Debug("Ollama embed model: %s", v)
		cfg.Provider.OllamaEmbedModel = v
	}
	if v := r.FormValue("ollama_chat_model"); v != "" {
		s.logger.Debug("Ollama chat model: %s", v)
		cfg.Provider.OllamaChatModel = v
	}

	// OpenAI settings
	if v := r.FormValue("openai_key"); v != "" {
		s.logger.Debug("OpenAI key provided: %d chars", len(v))
		cfg.Provider.OpenAIKey = v
	}
	if v := r.FormValue("openai_embed_model"); v != "" {
		s.logger.Debug("OpenAI embed model: %s", v)
		cfg.Provider.OpenAIEmbedModel = v
	}
	if v := r.FormValue("openai_chat_model"); v != "" {
		s.logger.Debug("OpenAI chat model: %s", v)
		cfg.Provider.OpenAIChatModel = v
	}

	// Anthropic settings
	if v := r.FormValue("anthropic_key"); v != "" {
		s.logger.Debug("Anthropic key provided: %d chars", len(v))
		cfg.Provider.AnthropicKey = v
	}
	if v := r.FormValue("anthropic_embed_model"); v != "" {
		s.logger.Debug("Anthropic embed model: %s", v)
		cfg.Provider.AnthropicEmbedModel = v
	}
	if v := r.FormValue("anthropic_chat_model"); v != "" {
		s.logger.Debug("Anthropic chat model: %s", v)
		cfg.Provider.AnthropicChatModel = v
	}

	// Watched folders
	folders := r.Form["folders"]
	s.logger.Debug("Watched folders from form: %v (count=%d)", folders, len(folders))
	if folders != nil {
		cfg.Folders = folders
	}
	s.logger.Debug("Config folders updated to: %v", cfg.Folders)

	// Guardrails settings
	if v := r.FormValue("pii_detection"); v != "" {
		s.logger.Debug("PII detection: %s", v)
		cfg.Guardrails.PIIDetection = v
	}
	autoSummarizeValue := r.FormValue("auto_summarize")
	s.logger.Debug("Auto summarize form value: '%s'", autoSummarizeValue)
	if autoSummarizeValue == "on" {
		cfg.Guardrails.AutoSummarize = true
	} else {
		cfg.Guardrails.AutoSummarize = false
	}
	s.logger.Debug("Auto summarize set to: %v", cfg.Guardrails.AutoSummarize)

	if v := r.FormValue("max_file_size_mb"); v != "" {
		if size, err := strconv.Atoi(v); err == nil {
			s.logger.Debug("Max file size: %d MB", size)
			cfg.Guardrails.MaxFileSizeMB = size
		} else {
			s.logger.Warn("Invalid max_file_size_mb value: %s", v)
		}
	}
	if v := r.FormValue("max_concurrent"); v != "" {
		if concurrent, err := strconv.Atoi(v); err == nil {
			s.logger.Debug("Max concurrent: %d", concurrent)
			cfg.Guardrails.MaxConcurrent = concurrent
		} else {
			s.logger.Warn("Invalid max_concurrent value: %s", v)
		}
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		s.logger.Error("Config validation failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid configuration: " + err.Error(),
		})
		return
	}

	s.logger.Debug("Config validated successfully")

	// Save configuration
	if err := cfg.Save(s.configPath); err != nil {
		s.logger.Error("Failed to save config: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to save configuration: " + err.Error(),
		})
		return
	}

	s.logger.Info("Settings saved successfully to %s", s.configPath)
	s.logger.Debug("Saved config: privacy=%v, folders=%v", cfg.Privacy.Enabled, cfg.Folders)

	// Return success with restart message
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Settings saved successfully. Please restart the application for changes to take effect.",
	})
}

// handlePrivacyMode toggles privacy mode on/off and switches LLM provider
func (s *Server) handlePrivacyMode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.logger.Debug("Received privacy mode toggle request")

	// Parse JSON body
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.Error("Failed to parse request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	s.logger.Debug("Privacy mode toggle: %v", req.Enabled)

	// Load current config
	cfg, err := config.Load(s.configPath)
	if err != nil {
		s.logger.Error("Failed to load config: %v", err)
		http.Error(w, "Failed to load config", http.StatusInternalServerError)
		return
	}

	// Update privacy mode
	cfg.Privacy.Enabled = req.Enabled

	// Determine which provider to use based on privacy mode
	var providerType string
	if req.Enabled {
		// Privacy mode ON: force Ollama
		providerType = "ollama"
		s.logger.Info("Privacy mode enabled - switching to Ollama")
	} else {
		// Privacy mode OFF: use configured provider
		providerType = cfg.Provider.Type
		s.logger.Info("Privacy mode disabled - using configured provider: %s", providerType)
	}

	// Save configuration
	if err := cfg.Save(s.configPath); err != nil {
		s.logger.Error("Failed to save config: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to save configuration: " + err.Error(),
		})
		return
	}

	// Update in-memory config
	s.config.PrivacyMode = req.Enabled

	// Update provider type in config to reflect the active provider
	if req.Enabled {
		s.config.Provider = "ollama"
	} else {
		s.config.Provider = cfg.Provider.Type
	}

	s.logger.Info("Privacy mode updated to: %v, active provider: %s", req.Enabled, providerType)

	// Return success with provider info
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"enabled":  req.Enabled,
		"provider": providerType,
	})
}
