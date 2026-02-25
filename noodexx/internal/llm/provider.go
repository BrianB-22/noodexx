package llm

import (
	"context"
	"fmt"
	"io"
	"noodexx/internal/logging"
)

// Provider defines the interface for LLM services
type Provider interface {
	// Embed generates an embedding vector for the given text
	Embed(ctx context.Context, text string) ([]float32, error)

	// Stream generates a chat completion and streams it to the writer
	Stream(ctx context.Context, messages []Message, w io.Writer) (string, error)

	// Name returns the provider name (e.g., "ollama", "openai", "anthropic")
	Name() string

	// IsLocal returns true if the provider runs locally
	IsLocal() bool
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"` // "system", "user", "assistant"
	Content string `json:"content"`
}

// Config holds provider configuration
type Config struct {
	Type                string // "ollama", "openai", "anthropic"
	OllamaEndpoint      string
	OllamaEmbedModel    string
	OllamaChatModel     string
	OpenAIKey           string
	OpenAIEmbedModel    string
	OpenAIChatModel     string
	AnthropicKey        string
	AnthropicEmbedModel string
	AnthropicChatModel  string
}

// NewProvider creates a provider based on config with privacy mode enforcement
func NewProvider(cfg Config, privacyMode bool, logger *logging.Logger) (Provider, error) {
	// Privacy mode enforcement: only allow Ollama when privacy mode is enabled
	if privacyMode && cfg.Type != "ollama" {
		return nil, fmt.Errorf("privacy mode is enabled - only Ollama provider is allowed")
	}

	switch cfg.Type {
	case "ollama":
		return NewOllamaProvider(cfg.OllamaEndpoint, cfg.OllamaEmbedModel, cfg.OllamaChatModel, logger), nil
	case "openai":
		if cfg.OpenAIKey == "" {
			return nil, fmt.Errorf("openai API key is required")
		}
		return NewOpenAIProvider(cfg.OpenAIKey, cfg.OpenAIEmbedModel, cfg.OpenAIChatModel, logger), nil
	case "anthropic":
		if cfg.AnthropicKey == "" {
			return nil, fmt.Errorf("anthropic API key is required")
		}
		return NewAnthropicProvider(cfg.AnthropicKey, cfg.AnthropicEmbedModel, cfg.AnthropicChatModel, logger), nil
	default:
		return nil, fmt.Errorf("unknown provider type: %s", cfg.Type)
	}
}
