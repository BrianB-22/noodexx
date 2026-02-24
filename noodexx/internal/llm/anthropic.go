package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// AnthropicProvider implements the Provider interface for Anthropic Claude
type AnthropicProvider struct {
	apiKey     string
	embedModel string // Uses Voyage AI for embeddings
	chatModel  string
	client     *http.Client
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider(apiKey, embedModel, chatModel string) *AnthropicProvider {
	return &AnthropicProvider{
		apiKey:     apiKey,
		embedModel: embedModel,
		chatModel:  chatModel,
		client:     &http.Client{Timeout: 60 * time.Second},
	}
}

// Embed generates an embedding vector for the given text
// Note: Anthropic doesn't provide embeddings directly, use Voyage AI
func (p *AnthropicProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	// Anthropic doesn't provide embeddings directly
	// This is a placeholder - actual implementation would use Voyage AI API
	return nil, fmt.Errorf("anthropic: embeddings not yet implemented - use Voyage AI")
}

// Stream generates a chat completion and streams it to the writer
func (p *AnthropicProvider) Stream(ctx context.Context, messages []Message, w io.Writer) (string, error) {
	// Convert messages to Anthropic format (system message separate)
	var system string
	var anthropicMessages []map[string]string

	for _, msg := range messages {
		if msg.Role == "system" {
			system = msg.Content
		} else {
			anthropicMessages = append(anthropicMessages, map[string]string{
				"role":    msg.Role,
				"content": msg.Content,
			})
		}
	}

	// Prepare request body
	reqBody := map[string]interface{}{
		"model":      p.chatModel,
		"messages":   anthropicMessages,
		"max_tokens": 4096,
		"stream":     true,
	}

	// Add system message if present
	if system != "" {
		reqBody["system"] = system
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("anthropic: failed to marshal stream request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("anthropic: failed to create stream request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// Execute request
	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("anthropic: stream request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("anthropic: stream returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse streaming response using SSE format
	var fullResponse strings.Builder
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()

		// Skip non-data lines
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// Parse the event
		var event struct {
			Type  string `json:"type"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
		}

		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		// Extract text from content_block_delta events
		if event.Type == "content_block_delta" && event.Delta.Text != "" {
			fullResponse.WriteString(event.Delta.Text)
			if _, err := w.Write([]byte(event.Delta.Text)); err != nil {
				return fullResponse.String(), fmt.Errorf("anthropic: failed to write stream content: %w", err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fullResponse.String(), fmt.Errorf("anthropic: failed to read stream: %w", err)
	}

	return fullResponse.String(), nil
}

// Name returns the provider name
func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

// IsLocal returns false since Anthropic is a cloud service
func (p *AnthropicProvider) IsLocal() bool {
	return false
}
