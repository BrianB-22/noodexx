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

		// Should NOT have context instructions when chunks is empty
		if strings.Contains(result, "Context:") {
			t.Error("Expected prompt to NOT contain 'Context:' section when chunks is empty")
		}
		if strings.Contains(result, "Use the following context") {
			t.Error("Expected prompt to NOT contain 'Use the following context' when chunks is empty")
		}

		// Should have simple format
		if !strings.Contains(result, "You are a helpful assistant") {
			t.Error("Expected prompt to contain 'You are a helpful assistant'")
		}
	})

	// Bug Condition Exploration Test - Property 1: Fault Condition
	// CRITICAL: This test MUST FAIL on unfixed code to prove the bug exists
	// When chunks is empty (RAG disabled), the prompt should NOT include context instructions
	t.Run("empty chunks should not include context instructions (bug exploration)", func(t *testing.T) {
		testCases := []struct {
			name  string
			query string
		}{
			{"simple query", "What is Go?"},
			{"configuration query", "How do I configure the server?"},
			{"explanation query", "Explain authentication"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				chunks := []Chunk{} // Empty chunks - RAG disabled

				result := pb.BuildPrompt(tc.query, chunks)

				// Bug condition: These should NOT be present when chunks is empty
				// EXPECTED TO FAIL on unfixed code
				if strings.Contains(result, "Use the following context") {
					t.Errorf("Prompt should NOT contain 'Use the following context' when chunks is empty, but got: %s", result)
				}
				if strings.Contains(result, "Context:") {
					t.Errorf("Prompt should NOT contain 'Context:' section when chunks is empty, but got: %s", result)
				}
				if strings.Contains(result, "Answer based on the context above") {
					t.Errorf("Prompt should NOT contain 'Answer based on the context above' when chunks is empty, but got: %s", result)
				}

				// Expected behavior: Simple prompt with just system message and query
				if !strings.Contains(result, "You are a helpful assistant") {
					t.Errorf("Prompt should contain 'You are a helpful assistant', but got: %s", result)
				}
				if !strings.Contains(result, tc.query) {
					t.Errorf("Prompt should contain query %q, but got: %s", tc.query, result)
				}
			})
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

	// Preservation Property Tests - Property 2: Preservation
	// These tests capture the CURRENT behavior with non-empty chunks
	// They should PASS on unfixed code and continue to PASS after the fix
	t.Run("non-empty chunks preserve RAG format (preservation)", func(t *testing.T) {
		t.Run("single chunk includes all context instructions", func(t *testing.T) {
			query := "What is the capital?"
			chunks := []Chunk{
				{Source: "geography.txt", Text: "Paris is the capital of France.", Score: 0.95},
			}

			result := pb.BuildPrompt(query, chunks)

			// Verify RAG instructions are present
			if !strings.Contains(result, "Use the following context to answer the user's question") {
				t.Error("Expected prompt to contain 'Use the following context to answer the user's question'")
			}
			if !strings.Contains(result, "Context:") {
				t.Error("Expected prompt to contain 'Context:' section")
			}
			if !strings.Contains(result, "Answer based on the context above") {
				t.Error("Expected prompt to contain 'Answer based on the context above'")
			}

			// Verify source attribution
			if !strings.Contains(result, "[1] Source: geography.txt") {
				t.Error("Expected prompt to contain '[1] Source: geography.txt'")
			}
			if !strings.Contains(result, chunks[0].Text) {
				t.Error("Expected prompt to contain chunk text")
			}

			// Verify query is present
			if !strings.Contains(result, query) {
				t.Error("Expected prompt to contain query")
			}
		})

		t.Run("multiple chunks preserve sequential numbering", func(t *testing.T) {
			query := "Tell me about cities"
			chunks := []Chunk{
				{Source: "cities1.txt", Text: "London is in England.", Score: 0.9},
				{Source: "cities2.txt", Text: "Tokyo is in Japan.", Score: 0.85},
				{Source: "cities3.txt", Text: "New York is in USA.", Score: 0.8},
			}

			result := pb.BuildPrompt(query, chunks)

			// Verify all chunks are numbered sequentially
			if !strings.Contains(result, "[1] Source: cities1.txt") {
				t.Error("Expected '[1] Source: cities1.txt'")
			}
			if !strings.Contains(result, "[2] Source: cities2.txt") {
				t.Error("Expected '[2] Source: cities2.txt'")
			}
			if !strings.Contains(result, "[3] Source: cities3.txt") {
				t.Error("Expected '[3] Source: cities3.txt'")
			}

			// Verify all chunk texts are present
			for _, chunk := range chunks {
				if !strings.Contains(result, chunk.Text) {
					t.Errorf("Expected prompt to contain chunk text: %s", chunk.Text)
				}
			}
		})

		t.Run("special characters in chunks are preserved", func(t *testing.T) {
			query := "What about special chars?"
			chunks := []Chunk{
				{Source: "special.txt", Text: "Text with \"quotes\" and 'apostrophes'", Score: 0.9},
				{Source: "symbols.txt", Text: "Symbols: @#$%^&*()", Score: 0.85},
			}

			result := pb.BuildPrompt(query, chunks)

			// Verify special characters are preserved
			if !strings.Contains(result, chunks[0].Text) {
				t.Error("Expected prompt to preserve quotes and apostrophes")
			}
			if !strings.Contains(result, chunks[1].Text) {
				t.Error("Expected prompt to preserve special symbols")
			}
		})
	})
}
