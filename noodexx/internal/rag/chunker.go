package rag

import "strings"

// Chunker splits text into overlapping segments
type Chunker struct {
	ChunkSize int // Target characters per chunk (200-500)
	Overlap   int // Overlap between chunks (50)
}

// NewChunker creates a new Chunker with default settings
func NewChunker(chunkSize, overlap int) *Chunker {
	return &Chunker{
		ChunkSize: chunkSize,
		Overlap:   overlap,
	}
}

// ChunkText splits text into chunks with overlap using rune-based slicing
// for proper Unicode handling
func (c *Chunker) ChunkText(text string) []string {
	var chunks []string
	runes := []rune(text)

	for i := 0; i < len(runes); i += c.ChunkSize - c.Overlap {
		end := i + c.ChunkSize
		if end > len(runes) {
			end = len(runes)
		}

		chunk := string(runes[i:end])
		chunks = append(chunks, strings.TrimSpace(chunk))

		if end == len(runes) {
			break
		}
	}

	return chunks
}
