package main

import (
	"context"
	"io"

	"noodexx/internal/ingest"
	"noodexx/internal/llm"
	"noodexx/internal/rag"
	"noodexx/internal/store"
)

// storeAdapter adapts store.Store to rag.Store interface
type storeAdapter struct {
	store *store.Store
}

func (sa *storeAdapter) Search(ctx context.Context, queryVec []float32, topK int) ([]rag.Chunk, error) {
	storeChunks, err := sa.store.Search(ctx, queryVec, topK)
	if err != nil {
		return nil, err
	}

	// Convert store.Chunk to rag.Chunk
	ragChunks := make([]rag.Chunk, len(storeChunks))
	for i, sc := range storeChunks {
		ragChunks[i] = rag.Chunk{
			Source: sc.Source,
			Text:   sc.Text,
			Score:  0, // Score calculated by store
		}
	}
	return ragChunks, nil
}

// providerAdapter adapts llm.Provider to ingest.LLMProvider interface
type providerAdapter struct {
	provider llm.Provider
}

func (pa *providerAdapter) Embed(ctx context.Context, text string) ([]float32, error) {
	return pa.provider.Embed(ctx, text)
}

func (pa *providerAdapter) Stream(ctx context.Context, messages []ingest.Message, w io.Writer) (string, error) {
	// Convert ingest.Message to llm.Message
	llmMessages := make([]llm.Message, len(messages))
	for i, msg := range messages {
		llmMessages[i] = llm.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return pa.provider.Stream(ctx, llmMessages, w)
}
