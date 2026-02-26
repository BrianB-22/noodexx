package provider

import (
	"fmt"
	"noodexx/internal/config"
	"noodexx/internal/llm"
	"noodexx/internal/logging"
)

// DualProviderManager manages two provider instances (local and cloud)
// and routes requests based on privacy toggle state
type DualProviderManager struct {
	localProvider llm.Provider
	cloudProvider llm.Provider
	config        *config.Config
	logger        *logging.Logger
	useLocalAI    bool // Internal state for provider selection
}

// NewDualProviderManager creates a manager with both providers
// Initializes both local and cloud providers if they are configured (Type is not empty)
// Returns error if neither provider is configured
func NewDualProviderManager(cfg *config.Config, logger *logging.Logger) (*DualProviderManager, error) {
	manager := &DualProviderManager{
		config:     cfg,
		logger:     logger,
		useLocalAI: cfg.Privacy.UseLocalAI, // Initialize from config
	}

	// Initialize local provider if configured
	if cfg.LocalProvider.Type != "" {
		localCfg := llm.Config{
			Type:                cfg.LocalProvider.Type,
			OllamaEndpoint:      cfg.LocalProvider.OllamaEndpoint,
			OllamaEmbedModel:    cfg.LocalProvider.OllamaEmbedModel,
			OllamaChatModel:     cfg.LocalProvider.OllamaChatModel,
			OpenAIKey:           cfg.LocalProvider.OpenAIKey,
			OpenAIEmbedModel:    cfg.LocalProvider.OpenAIEmbedModel,
			OpenAIChatModel:     cfg.LocalProvider.OpenAIChatModel,
			AnthropicKey:        cfg.LocalProvider.AnthropicKey,
			AnthropicEmbedModel: cfg.LocalProvider.AnthropicEmbedModel,
			AnthropicChatModel:  cfg.LocalProvider.AnthropicChatModel,
		}

		provider, err := llm.NewProvider(localCfg, false, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize local provider: %w", err)
		}
		manager.localProvider = provider
		logger.Info("Local provider initialized: %s", cfg.LocalProvider.Type)
	}

	// Initialize cloud provider if configured
	if cfg.CloudProvider.Type != "" {
		cloudCfg := llm.Config{
			Type:                cfg.CloudProvider.Type,
			OllamaEndpoint:      cfg.CloudProvider.OllamaEndpoint,
			OllamaEmbedModel:    cfg.CloudProvider.OllamaEmbedModel,
			OllamaChatModel:     cfg.CloudProvider.OllamaChatModel,
			OpenAIKey:           cfg.CloudProvider.OpenAIKey,
			OpenAIEmbedModel:    cfg.CloudProvider.OpenAIEmbedModel,
			OpenAIChatModel:     cfg.CloudProvider.OpenAIChatModel,
			AnthropicKey:        cfg.CloudProvider.AnthropicKey,
			AnthropicEmbedModel: cfg.CloudProvider.AnthropicEmbedModel,
			AnthropicChatModel:  cfg.CloudProvider.AnthropicChatModel,
		}

		provider, err := llm.NewProvider(cloudCfg, false, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize cloud provider: %w", err)
		}
		manager.cloudProvider = provider
		logger.Info("Cloud provider initialized: %s", cfg.CloudProvider.Type)
	}

	// At least one provider must be configured
	if manager.localProvider == nil && manager.cloudProvider == nil {
		return nil, fmt.Errorf("at least one provider (local or cloud) must be configured")
	}

	return manager, nil
}

// GetActiveProvider returns the currently active provider based on privacy toggle state
// Returns error if the active provider is not configured
func (m *DualProviderManager) GetActiveProvider() (llm.Provider, error) {
	m.logger.Debug("GetActiveProvider called: useLocalAI=%v", m.useLocalAI)

	if m.useLocalAI {
		// Local mode - return local provider
		if m.localProvider == nil {
			return nil, fmt.Errorf("local provider not configured")
		}
		m.logger.Debug("Returning local provider")
		return m.localProvider, nil
	}

	// Cloud mode - return cloud provider
	if m.cloudProvider == nil {
		return nil, fmt.Errorf("cloud provider not configured")
	}
	m.logger.Debug("Returning cloud provider")
	return m.cloudProvider, nil
}

// GetLocalProvider returns the local provider instance (may be nil if not configured)
func (m *DualProviderManager) GetLocalProvider() llm.Provider {
	return m.localProvider
}

// GetCloudProvider returns the cloud provider instance (may be nil if not configured)
func (m *DualProviderManager) GetCloudProvider() llm.Provider {
	return m.cloudProvider
}

// IsLocalMode returns true if privacy toggle is set to local AI
func (m *DualProviderManager) IsLocalMode() bool {
	return m.useLocalAI
}

// GetProviderName returns the name of the active provider for UI display
// Returns a human-readable name like "Local AI (Ollama)" or "Cloud AI (GPT-4)"
func (m *DualProviderManager) GetProviderName() string {
	if m.useLocalAI {
		// Local mode
		if m.localProvider == nil {
			return "Local AI (Not Configured)"
		}
		return fmt.Sprintf("Local AI (%s)", m.config.LocalProvider.Type)
	}

	// Cloud mode
	if m.cloudProvider == nil {
		return "Cloud AI (Not Configured)"
	}

	// For cloud providers, include the model name for more specificity
	providerType := m.config.CloudProvider.Type
	switch providerType {
	case "openai":
		if m.config.CloudProvider.OpenAIChatModel != "" {
			return fmt.Sprintf("Cloud AI (%s)", m.config.CloudProvider.OpenAIChatModel)
		}
		return "Cloud AI (OpenAI)"
	case "anthropic":
		if m.config.CloudProvider.AnthropicChatModel != "" {
			return fmt.Sprintf("Cloud AI (%s)", m.config.CloudProvider.AnthropicChatModel)
		}
		return "Cloud AI (Anthropic)"
	default:
		return fmt.Sprintf("Cloud AI (%s)", providerType)
	}
}

// Reload reinitializes providers after configuration changes
// This method updates the manager's config reference and reinitializes both providers
// based on the new configuration. It handles provider initialization errors gracefully
// by logging them and continuing with the providers that can be initialized.
func (m *DualProviderManager) Reload(cfg *config.Config) error {
	m.logger.Info("Reloading provider configuration: UseLocalAI=%v", cfg.Privacy.UseLocalAI)
	m.config = cfg
	m.useLocalAI = cfg.Privacy.UseLocalAI // Update internal state

	// Reinitialize local provider if configured
	if cfg.LocalProvider.Type != "" {
		localCfg := llm.Config{
			Type:                cfg.LocalProvider.Type,
			OllamaEndpoint:      cfg.LocalProvider.OllamaEndpoint,
			OllamaEmbedModel:    cfg.LocalProvider.OllamaEmbedModel,
			OllamaChatModel:     cfg.LocalProvider.OllamaChatModel,
			OpenAIKey:           cfg.LocalProvider.OpenAIKey,
			OpenAIEmbedModel:    cfg.LocalProvider.OpenAIEmbedModel,
			OpenAIChatModel:     cfg.LocalProvider.OpenAIChatModel,
			AnthropicKey:        cfg.LocalProvider.AnthropicKey,
			AnthropicEmbedModel: cfg.LocalProvider.AnthropicEmbedModel,
			AnthropicChatModel:  cfg.LocalProvider.AnthropicChatModel,
		}

		provider, err := llm.NewProvider(localCfg, false, m.logger)
		if err != nil {
			m.logger.Error("Failed to reinitialize local provider: %v", err)
			m.localProvider = nil
		} else {
			m.localProvider = provider
			m.logger.Info("Local provider reinitialized: %s", cfg.LocalProvider.Type)
		}
	} else {
		// Local provider was removed from config
		m.localProvider = nil
		m.logger.Info("Local provider removed from configuration")
	}

	// Reinitialize cloud provider if configured
	if cfg.CloudProvider.Type != "" {
		cloudCfg := llm.Config{
			Type:                cfg.CloudProvider.Type,
			OllamaEndpoint:      cfg.CloudProvider.OllamaEndpoint,
			OllamaEmbedModel:    cfg.CloudProvider.OllamaEmbedModel,
			OllamaChatModel:     cfg.CloudProvider.OllamaChatModel,
			OpenAIKey:           cfg.CloudProvider.OpenAIKey,
			OpenAIEmbedModel:    cfg.CloudProvider.OpenAIEmbedModel,
			OpenAIChatModel:     cfg.CloudProvider.OpenAIChatModel,
			AnthropicKey:        cfg.CloudProvider.AnthropicKey,
			AnthropicEmbedModel: cfg.CloudProvider.AnthropicEmbedModel,
			AnthropicChatModel:  cfg.CloudProvider.AnthropicChatModel,
		}

		provider, err := llm.NewProvider(cloudCfg, false, m.logger)
		if err != nil {
			m.logger.Error("Failed to reinitialize cloud provider: %v", err)
			m.cloudProvider = nil
		} else {
			m.cloudProvider = provider
			m.logger.Info("Cloud provider reinitialized: %s", cfg.CloudProvider.Type)
		}
	} else {
		// Cloud provider was removed from config
		m.cloudProvider = nil
		m.logger.Info("Cloud provider removed from configuration")
	}

	// At least one provider must be configured
	if m.localProvider == nil && m.cloudProvider == nil {
		return fmt.Errorf("at least one provider (local or cloud) must be configured after reload")
	}

	return nil
}
