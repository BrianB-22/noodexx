package llm

import (
	"bufio"
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

// OpenAIProvider implements the Provider interface for OpenAI
type OpenAIProvider struct {
	apiKey     string
	embedModel string
	chatModel  string
	client     *http.Client
	logger     *logging.Logger
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(apiKey, embedModel, chatModel string, logger *logging.Logger) *OpenAIProvider {
	return &OpenAIProvider{
		apiKey:     apiKey,
		embedModel: embedModel,
		chatModel:  chatModel,
		client:     &http.Client{Timeout: 60 * time.Second},
		logger:     logger,
	}
}

// Embed generates an embedding vector for the given text
func (p *OpenAIProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	logger := p.logger.WithFields(map[string]interface{}{
		"provider":  "openai",
		"model":     p.embedModel,
		"operation": "embed",
	})
	logger.Debug("starting embedding request")

	start := time.Now()
	reqBody := map[string]interface{}{
		"model": p.embedModel,
		"input": text,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		logger.WithContext("error", err.Error()).Error("failed to marshal embed request")
		return nil, fmt.Errorf("openai: failed to marshal embed request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/embeddings", bytes.NewReader(body))
	if err != nil {
		logger.WithContext("error", err.Error()).Error("failed to create embed request")
		return nil, fmt.Errorf("openai: failed to create embed request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		latency := time.Since(start).Milliseconds()
		logger.WithFields(map[string]interface{}{
			"error":      err.Error(),
			"latency_ms": latency,
		}).Error("embed request failed")
		return nil, fmt.Errorf("openai: embed request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		latency := time.Since(start).Milliseconds()
		logger.WithFields(map[string]interface{}{
			"status":     resp.StatusCode,
			"error":      string(bodyBytes),
			"latency_ms": latency,
		}).Error("embed returned non-OK status")
		return nil, fmt.Errorf("openai: embed returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		latency := time.Since(start).Milliseconds()
		logger.WithFields(map[string]interface{}{
			"error":      err.Error(),
			"latency_ms": latency,
		}).Error("failed to decode embed response")
		return nil, fmt.Errorf("openai: failed to decode embed response: %w", err)
	}

	if len(result.Data) == 0 {
		latency := time.Since(start).Milliseconds()
		logger.WithContext("latency_ms", latency).Error("received empty embeddings")
		return nil, fmt.Errorf("openai: returned no embeddings")
	}

	latency := time.Since(start).Milliseconds()
	logger.WithFields(map[string]interface{}{
		"latency_ms":  latency,
		"vector_size": len(result.Data[0].Embedding),
	}).Debug("embedding request completed")

	return result.Data[0].Embedding, nil
}

// Stream generates a chat completion and streams it to the writer
func (p *OpenAIProvider) Stream(ctx context.Context, messages []Message, w io.Writer) (string, error) {
	logger := p.logger.WithFields(map[string]interface{}{
		"provider":      "openai",
		"model":         p.chatModel,
		"operation":     "stream",
		"message_count": len(messages),
	})
	logger.Debug("starting chat stream request")

	start := time.Now()
	reqBody := map[string]interface{}{
		"model":    p.chatModel,
		"messages": messages,
		"stream":   true,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		logger.WithContext("error", err.Error()).Error("failed to marshal stream request")
		return "", fmt.Errorf("openai: failed to marshal stream request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		logger.WithContext("error", err.Error()).Error("failed to create stream request")
		return "", fmt.Errorf("openai: failed to create stream request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		latency := time.Since(start).Milliseconds()
		logger.WithFields(map[string]interface{}{
			"error":      err.Error(),
			"latency_ms": latency,
		}).Error("stream request failed")
		return "", fmt.Errorf("openai: stream request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		latency := time.Since(start).Milliseconds()
		logger.WithFields(map[string]interface{}{
			"status":     resp.StatusCode,
			"error":      string(bodyBytes),
			"latency_ms": latency,
		}).Error("stream returned non-OK status")
		return "", fmt.Errorf("openai: stream returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var fullResponse strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	tokenCount := 0

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		if data == "[DONE]" {
			break
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}

		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			content := chunk.Choices[0].Delta.Content
			fullResponse.WriteString(content)
			tokenCount++
			if _, err := w.Write([]byte(content)); err != nil {
				latency := time.Since(start).Milliseconds()
				logger.WithFields(map[string]interface{}{
					"error":      err.Error(),
					"latency_ms": latency,
				}).Error("failed to write stream content")
				return fullResponse.String(), fmt.Errorf("openai: failed to write stream content: %w", err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		latency := time.Since(start).Milliseconds()
		logger.WithFields(map[string]interface{}{
			"error":      err.Error(),
			"latency_ms": latency,
		}).Error("failed to read stream")
		return fullResponse.String(), fmt.Errorf("openai: failed to read stream: %w", err)
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
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// IsLocal returns false since OpenAI is a cloud service
func (p *OpenAIProvider) IsLocal() bool {
	return false
}
