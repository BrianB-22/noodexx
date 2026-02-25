package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"noodexx/internal/logging"
	"strings"
	"time"
)

// OllamaProvider implements the Provider interface for Ollama
type OllamaProvider struct {
	endpoint   string
	embedModel string
	chatModel  string
	client     *http.Client
	logger     *logging.Logger
}

// NewOllamaProvider creates a new Ollama provider
func NewOllamaProvider(endpoint, embedModel, chatModel string, logger *logging.Logger) *OllamaProvider {
	return &OllamaProvider{
		endpoint:   endpoint,
		embedModel: embedModel,
		chatModel:  chatModel,
		client:     &http.Client{Timeout: 60 * time.Second},
		logger:     logger,
	}
}

// Embed generates an embedding vector for the given text
func (p *OllamaProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	logger := p.logger.WithFields(map[string]interface{}{
		"provider":  "ollama",
		"model":     p.embedModel,
		"operation": "embed",
	})
	logger.Debug("starting embedding request")

	start := time.Now()
	// Prepare request body
	reqBody := map[string]interface{}{
		"model":  p.embedModel,
		"prompt": text,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		logger.WithContext("error", err.Error()).Error("failed to marshal embed request")
		return nil, fmt.Errorf("ollama: failed to marshal embed request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", p.endpoint+"/api/embeddings", bytes.NewReader(body))
	if err != nil {
		logger.WithContext("error", err.Error()).Error("failed to create embed request")
		return nil, fmt.Errorf("ollama: failed to create embed request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := p.client.Do(req)
	if err != nil {
		latency := time.Since(start).Milliseconds()
		logger.WithFields(map[string]interface{}{
			"error":      err.Error(),
			"latency_ms": latency,
		}).Error("embed request failed")
		return nil, fmt.Errorf("ollama: embed request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		latency := time.Since(start).Milliseconds()
		logger.WithFields(map[string]interface{}{
			"status":     resp.StatusCode,
			"error":      string(bodyBytes),
			"latency_ms": latency,
		}).Error("embed returned non-OK status")
		return nil, fmt.Errorf("ollama: embed returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var result struct {
		Embedding []float32 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		latency := time.Since(start).Milliseconds()
		logger.WithFields(map[string]interface{}{
			"error":      err.Error(),
			"latency_ms": latency,
		}).Error("failed to decode embed response")
		return nil, fmt.Errorf("ollama: failed to decode embed response: %w", err)
	}

	if len(result.Embedding) == 0 {
		latency := time.Since(start).Milliseconds()
		logger.WithContext("latency_ms", latency).Error("received empty embedding vector")
		return nil, fmt.Errorf("ollama: received empty embedding vector")
	}

	latency := time.Since(start).Milliseconds()
	logger.WithFields(map[string]interface{}{
		"latency_ms":  latency,
		"vector_size": len(result.Embedding),
	}).Debug("embedding request completed")

	return result.Embedding, nil
}

// Stream generates a chat completion and streams it to the writer
func (p *OllamaProvider) Stream(ctx context.Context, messages []Message, w io.Writer) (string, error) {
	logger := p.logger.WithFields(map[string]interface{}{
		"provider":      "ollama",
		"model":         p.chatModel,
		"operation":     "stream",
		"message_count": len(messages),
	})
	logger.Debug("starting chat stream request")

	start := time.Now()
	// Prepare request body
	reqBody := map[string]interface{}{
		"model":    p.chatModel,
		"messages": messages,
		"stream":   true,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		logger.WithContext("error", err.Error()).Error("failed to marshal stream request")
		return "", fmt.Errorf("ollama: failed to marshal stream request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", p.endpoint+"/api/chat", bytes.NewReader(body))
	if err != nil {
		logger.WithContext("error", err.Error()).Error("failed to create stream request")
		return "", fmt.Errorf("ollama: failed to create stream request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := p.client.Do(req)
	if err != nil {
		latency := time.Since(start).Milliseconds()
		logger.WithFields(map[string]interface{}{
			"error":      err.Error(),
			"latency_ms": latency,
		}).Error("stream request failed")
		return "", fmt.Errorf("ollama: stream request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		latency := time.Since(start).Milliseconds()
		logger.WithFields(map[string]interface{}{
			"status":     resp.StatusCode,
			"error":      string(bodyBytes),
			"latency_ms": latency,
		}).Error("stream returned non-OK status")
		return "", fmt.Errorf("ollama: stream returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse streaming response using JSON decoder
	var fullResponse strings.Builder
	decoder := json.NewDecoder(resp.Body)
	tokenCount := 0

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
			latency := time.Since(start).Milliseconds()
			logger.WithFields(map[string]interface{}{
				"error":      err.Error(),
				"latency_ms": latency,
			}).Error("failed to decode stream chunk")
			return fullResponse.String(), fmt.Errorf("ollama: failed to decode stream chunk: %w", err)
		}

		// Write content to output writer
		if chunk.Message.Content != "" {
			fullResponse.WriteString(chunk.Message.Content)
			tokenCount++
			if _, err := w.Write([]byte(chunk.Message.Content)); err != nil {
				latency := time.Since(start).Milliseconds()
				logger.WithFields(map[string]interface{}{
					"error":      err.Error(),
					"latency_ms": latency,
				}).Error("failed to write stream content")
				return fullResponse.String(), fmt.Errorf("ollama: failed to write stream content: %w", err)
			}
		}

		// Check if streaming is complete
		if chunk.Done {
			break
		}
	}

	latency := time.Since(start).Milliseconds()
	logger.WithFields(map[string]interface{}{
		"latency_ms":      latency,
		"tokens":          tokenCount,
		"response_length": fullResponse.Len(),
	}).Debug("chat stream completed")

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
