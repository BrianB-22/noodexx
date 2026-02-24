package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OllamaProvider implements the Provider interface for Ollama
type OllamaProvider struct {
	endpoint   string
	embedModel string
	chatModel  string
	client     *http.Client
}

// NewOllamaProvider creates a new Ollama provider
func NewOllamaProvider(endpoint, embedModel, chatModel string) *OllamaProvider {
	return &OllamaProvider{
		endpoint:   endpoint,
		embedModel: embedModel,
		chatModel:  chatModel,
		client:     &http.Client{Timeout: 60 * time.Second},
	}
}

// Embed generates an embedding vector for the given text
func (p *OllamaProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	// Prepare request body
	reqBody := map[string]interface{}{
		"model":  p.embedModel,
		"prompt": text,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("ollama: failed to marshal embed request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", p.endpoint+"/api/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("ollama: failed to create embed request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama: embed request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama: embed returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var result struct {
		Embedding []float32 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("ollama: failed to decode embed response: %w", err)
	}

	if len(result.Embedding) == 0 {
		return nil, fmt.Errorf("ollama: received empty embedding vector")
	}

	return result.Embedding, nil
}

// Stream generates a chat completion and streams it to the writer
func (p *OllamaProvider) Stream(ctx context.Context, messages []Message, w io.Writer) (string, error) {
	// Prepare request body
	reqBody := map[string]interface{}{
		"model":    p.chatModel,
		"messages": messages,
		"stream":   true,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("ollama: failed to marshal stream request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", p.endpoint+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("ollama: failed to create stream request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama: stream request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama: stream returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse streaming response using JSON decoder
	var fullResponse strings.Builder
	decoder := json.NewDecoder(resp.Body)

	for {
		var chunk struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			Done bool `json:"done"`
		}

		if err := decoder.Decode(&chunk); err != nil {
			if err == io.EOF {
				break
			}
			return fullResponse.String(), fmt.Errorf("ollama: failed to decode stream chunk: %w", err)
		}

		// Write content to output writer
		if chunk.Message.Content != "" {
			fullResponse.WriteString(chunk.Message.Content)
			if _, err := w.Write([]byte(chunk.Message.Content)); err != nil {
				return fullResponse.String(), fmt.Errorf("ollama: failed to write stream content: %w", err)
			}
		}

		// Check if streaming is complete
		if chunk.Done {
			break
		}
	}

	return fullResponse.String(), nil
}

// Name returns the provider name
func (p *OllamaProvider) Name() string {
	return "ollama"
}

// IsLocal returns true since Ollama runs locally
func (p *OllamaProvider) IsLocal() bool {
	return true
}
