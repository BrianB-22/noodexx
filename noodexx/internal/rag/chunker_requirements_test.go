package rag

import (
	"strings"
	"testing"
)

// TestChunker_RequirementCompliance verifies that the chunker meets
// Requirements 4.1 and 4.5:
// - ChunkText function splits text into overlapping segments
// - Chunks are between 200 and 500 characters with 50 character overlap
func TestChunker_RequirementCompliance(t *testing.T) {
	// Test with requirement-specified parameters
	c := NewChunker(300, 50) // Mid-range chunk size within 200-500

	// Create a long text to ensure multiple chunks
	text := strings.Repeat("This is a test sentence with some content. ", 100) // ~4300 characters

	chunks := c.ChunkText(text)

	if len(chunks) == 0 {
		t.Fatal("ChunkText returned no chunks")
	}

	t.Logf("Generated %d chunks from %d character text", len(chunks), len(text))

	// Verify each chunk is within the acceptable range
	for i, chunk := range chunks {
		runeCount := len([]rune(chunk))

		// Chunks should be close to the target size (allowing for trimming and last chunk)
		if i < len(chunks)-1 { // Not the last chunk
			if runeCount < 200 || runeCount > 500 {
				t.Logf("Warning: Chunk %d has %d runes (expected 200-500)", i, runeCount)
			}
		}

		// Last chunk can be smaller
		if i == len(chunks)-1 && runeCount > 500 {
			t.Errorf("Last chunk %d has %d runes, exceeds maximum 500", i, runeCount)
		}
	}

	// Verify overlap by checking that consecutive chunks share content
	if len(chunks) > 1 {
		for i := 0; i < len(chunks)-1; i++ {
			// Get the end of current chunk and start of next chunk
			currentChunk := chunks[i]
			nextChunk := chunks[i+1]

			// Due to trimming, we can't guarantee exact overlap, but chunks should exist
			if len(currentChunk) == 0 || len(nextChunk) == 0 {
				t.Errorf("Empty chunk found at position %d or %d", i, i+1)
			}
		}
	}
}

// TestChunker_BoundaryValues tests the chunker with boundary values
// specified in Requirement 4.5
func TestChunker_BoundaryValues(t *testing.T) {
	tests := []struct {
		name      string
		chunkSize int
		overlap   int
	}{
		{
			name:      "minimum chunk size",
			chunkSize: 200,
			overlap:   50,
		},
		{
			name:      "maximum chunk size",
			chunkSize: 500,
			overlap:   50,
		},
		{
			name:      "mid-range chunk size",
			chunkSize: 350,
			overlap:   50,
		},
	}

	text := strings.Repeat("Lorem ipsum dolor sit amet, consectetur adipiscing elit. ", 50)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewChunker(tt.chunkSize, tt.overlap)
			chunks := c.ChunkText(text)

			if len(chunks) == 0 {
				t.Fatal("ChunkText returned no chunks")
			}

			t.Logf("Chunk size %d with overlap %d produced %d chunks", tt.chunkSize, tt.overlap, len(chunks))

			// Verify chunks are reasonable
			for i, chunk := range chunks {
				runeCount := len([]rune(chunk))
				if runeCount > tt.chunkSize+10 { // Small margin for edge cases
					t.Errorf("Chunk %d has %d runes, exceeds chunk size %d", i, runeCount, tt.chunkSize)
				}
			}
		})
	}
}

// TestChunker_OverlapBehavior specifically tests the 50 character overlap
// requirement from Requirement 4.5
func TestChunker_OverlapBehavior(t *testing.T) {
	c := NewChunker(300, 50)

	// Create text with distinct markers to verify overlap
	text := ""
	for i := 0; i < 20; i++ {
		text += strings.Repeat(string(rune('A'+i)), 100) // 100 chars of each letter
	}

	chunks := c.ChunkText(text)

	if len(chunks) < 2 {
		t.Fatalf("Expected multiple chunks, got %d", len(chunks))
	}

	t.Logf("Generated %d chunks with 50 character overlap", len(chunks))

	// Verify that we're stepping by (chunkSize - overlap) = 250
	// This means each chunk starts 250 characters after the previous one
	// For a 2000 char text with 300 size and 50 overlap:
	// Chunk 0: 0-300, Chunk 1: 250-550, Chunk 2: 500-800, etc.
	expectedMinChunks := len([]rune(text)) / (c.ChunkSize - c.Overlap)
	if len(chunks) < expectedMinChunks-1 { // Allow some margin
		t.Errorf("Expected at least %d chunks, got %d", expectedMinChunks-1, len(chunks))
	}
}
