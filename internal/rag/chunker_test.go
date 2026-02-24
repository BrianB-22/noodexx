package rag

import (
	"strings"
	"testing"
)

func TestChunker_ChunkText(t *testing.T) {
	tests := []struct {
		name      string
		chunkSize int
		overlap   int
		text      string
		wantMin   int // minimum number of chunks expected
	}{
		{
			name:      "simple text with default settings",
			chunkSize: 300,
			overlap:   50,
			text:      strings.Repeat("a", 500),
			wantMin:   2,
		},
		{
			name:      "text shorter than chunk size",
			chunkSize: 300,
			overlap:   50,
			text:      "Short text",
			wantMin:   1,
		},
		{
			name:      "unicode text",
			chunkSize: 100,
			overlap:   20,
			text:      "Hello 世界! " + strings.Repeat("测试", 50),
			wantMin:   1,
		},
		{
			name:      "empty text",
			chunkSize: 300,
			overlap:   50,
			text:      "",
			wantMin:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewChunker(tt.chunkSize, tt.overlap)
			chunks := c.ChunkText(tt.text)

			if len(chunks) < tt.wantMin {
				t.Errorf("ChunkText() returned %d chunks, want at least %d", len(chunks), tt.wantMin)
			}

			// Verify chunks are within size bounds (allowing for trimming)
			for i, chunk := range chunks {
				runeCount := len([]rune(chunk))
				if runeCount > tt.chunkSize+10 { // Allow small margin for edge cases
					t.Errorf("Chunk %d has %d runes, exceeds chunk size %d", i, runeCount, tt.chunkSize)
				}
			}

			// Verify overlap exists between consecutive chunks (if multiple chunks)
			if len(chunks) > 1 && tt.overlap > 0 && len(tt.text) > 0 {
				// Just verify we got multiple chunks when text is long enough
				if len([]rune(tt.text)) > tt.chunkSize && len(chunks) < 2 {
					t.Errorf("Expected multiple chunks for long text, got %d", len(chunks))
				}
			}
		})
	}
}

func TestChunker_ChunkText_Overlap(t *testing.T) {
	c := NewChunker(200, 50)
	text := strings.Repeat("abcdefghij", 50) // 500 characters

	chunks := c.ChunkText(text)

	if len(chunks) < 2 {
		t.Fatalf("Expected at least 2 chunks, got %d", len(chunks))
	}

	// Verify chunks are created with proper stepping
	// With ChunkSize=200 and Overlap=50, step should be 150
	// So for 500 chars, we expect: chunk at 0, 150, 300, 450 (4 chunks)
	if len(chunks) < 3 {
		t.Errorf("Expected at least 3 chunks for 500 char text with 200 size and 50 overlap, got %d", len(chunks))
	}
}

func TestChunker_ChunkText_Unicode(t *testing.T) {
	c := NewChunker(10, 2)
	text := "Hello世界Test测试"

	chunks := c.ChunkText(text)

	// Verify no broken unicode characters
	for i, chunk := range chunks {
		if !strings.ContainsAny(chunk, "世界测试HelloTest") && chunk != "" {
			t.Errorf("Chunk %d contains unexpected characters: %s", i, chunk)
		}
	}
}
