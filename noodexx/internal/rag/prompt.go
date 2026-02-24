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
	var sb strings.Builder

	sb.WriteString("You are a helpful assistant. Use the following context to answer the user's question.\n\n")
	sb.WriteString("Context:\n")

	for i, chunk := range chunks {
		sb.WriteString(fmt.Sprintf("\n[%d] Source: %s\n%s\n", i+1, chunk.Source, chunk.Text))
	}

	sb.WriteString("\n\nUser Question: ")
	sb.WriteString(query)
	sb.WriteString("\n\nAnswer based on the context above:")

	return sb.String()
}
