package rag

import (
	"fmt"
	"strings"
)

// PromptBuilder constructs prompts with retrieved context
type PromptBuilder struct{}

// NewPromptBuilder creates a new PromptBuilder
func NewPromptBuilder() *PromptBuilder {
	return &PromptBuilder{}
}

// BuildPrompt combines query and chunks into a RAG prompt
// It formats the context with source attribution and combines it with the user query
func (pb *PromptBuilder) BuildPrompt(query string, chunks []Chunk) string {
	// Handle empty chunks case (RAG disabled)
	if len(chunks) == 0 {
		return fmt.Sprintf("You are a helpful assistant.\n\nUser Question: %s", query)
	}

	// Existing logic for non-empty chunks (RAG enabled)
	var sb strings.Builder

	sb.WriteString("You are a helpful assistant. Use the following context to answer the user's question if it's relevant, or use your general knowledge if the context doesn't contain the answer.\n\n")
	sb.WriteString("Context:\n")

	for i, chunk := range chunks {
		sb.WriteString(fmt.Sprintf("\n[%d] Source: %s\n%s\n", i+1, chunk.Source, chunk.Text))
	}

	sb.WriteString("\n\nUser Question: ")
	sb.WriteString(query)
	sb.WriteString("\n\nAnswer based on the context above if relevant, otherwise answer from your general knowledge.")

	return sb.String()
}
