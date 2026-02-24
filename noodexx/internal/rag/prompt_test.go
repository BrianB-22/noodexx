package rag

import (
	"strings"
	"testing"
)

func TestBuildPrompt(t *testing.T) {
	pb := NewPromptBuilder()

	t.Run("builds prompt with query and chunks", func(t *testing.T) {
		query := "What is the capital of France?"
		chunks := []Chunk{
			{Source: "geography.txt", Text: "Paris is the capital and largest city of France.", Score: 0.95},
			{Source: "cities.txt", Text: "France is a country in Western Europe with Paris as its capital.", Score: 0.88},
		}

		result := pb.BuildPrompt(query, chunks)

		// Verify the prompt contains the query
		if !strings.Contains(result, query) {
			t.Errorf("Expected prompt to contain query %q", query)
		}

		// Verify the prompt contains all chunk texts
		for _, chunk := range chunks {
			if !strings.Contains(result, chunk.Text) {
				t.Errorf("Expected prompt to contain chunk text %q", chunk.Text)
			}
		}

		// Verify the prompt contains source attribution
		for _, chunk := range chunks {
			if !strings.Contains(result, chunk.Source) {
				t.Errorf("Expected prompt to contain source %q", chunk.Source)
			}
		}

		// Verify the prompt has the expected structure
		if !strings.Contains(result, "Context:") {
			t.Error("Expected prompt to contain 'Context:' section")
		}

		if !strings.Contains(result, "User Question:") {
			t.Error("Expected prompt to contain 'User Question:' section")
		}
	})

	t.Run("handles empty chunks", func(t *testing.T) {
		query := "What is AI?"
		chunks := []Chunk{}

		result := pb.BuildPrompt(query, chunks)

		// Should still contain the query
		if !strings.Contains(result, query) {
			t.Errorf("Expected prompt to contain query %q", query)
		}

		// Should have the basic structure
		if !strings.Contains(result, "Context:") {
			t.Error("Expected prompt to contain 'Context:' section")
		}
	})

	t.Run("formats chunks with numbered source attribution", func(t *testing.T) {
		query := "Test query"
		chunks := []Chunk{
			{Source: "doc1.txt", Text: "First chunk", Score: 0.9},
			{Source: "doc2.txt", Text: "Second chunk", Score: 0.8},
			{Source: "doc3.txt", Text: "Third chunk", Score: 0.7},
		}

		result := pb.BuildPrompt(query, chunks)

		// Verify numbered formatting
		if !strings.Contains(result, "[1] Source: doc1.txt") {
			t.Error("Expected prompt to contain '[1] Source: doc1.txt'")
		}
		if !strings.Contains(result, "[2] Source: doc2.txt") {
			t.Error("Expected prompt to contain '[2] Source: doc2.txt'")
		}
		if !strings.Contains(result, "[3] Source: doc3.txt") {
			t.Error("Expected prompt to contain '[3] Source: doc3.txt'")
		}
	})
}
