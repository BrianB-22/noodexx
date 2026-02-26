package store

import (
	"context"
	"os"
	"testing"
)

// TestChunkOperations tests all four chunk operations: SaveChunk, Search, Library, DeleteSource
func TestChunkOperations(t *testing.T) {
	// Create a temporary database file
	tmpFile := "test_chunk_ops.db"
	defer os.Remove(tmpFile)

	// Create a new store
	store, err := NewStore(tmpFile, "single")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Test 1: SaveChunk with embedding serialization
	t.Run("SaveChunk", func(t *testing.T) {
		source := "document1.txt"
		text := "This is the first chunk"
		embedding := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
		tags := []string{"tag1", "tag2"}
		summary := "Summary of document1"

		err := store.SaveChunk(ctx, 1, source, text, embedding, tags, summary)
		if err != nil {
			t.Fatalf("SaveChunk failed: %v", err)
		}

		// Save another chunk for the same source
		err = store.SaveChunk(ctx, 1, source, "Second chunk", embedding, tags, summary)
		if err != nil {
			t.Fatalf("SaveChunk failed for second chunk: %v", err)
		}

		// Save a chunk for a different source
		err = store.SaveChunk(ctx, 1, "document2.txt", "Different document", []float32{0.9, 0.8, 0.7, 0.6, 0.5}, []string{"tag3"}, "Summary of document2")
		if err != nil {
			t.Fatalf("SaveChunk failed for different source: %v", err)
		}
	})

	// Test 2: Search with cosine similarity calculation
	t.Run("Search", func(t *testing.T) {
		// Query vector similar to the first document
		queryVec := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
		results, err := store.Search(ctx, queryVec, 5)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Should return results
		if len(results) == 0 {
			t.Fatal("Search returned no results")
		}

		// Should return at most topK results
		if len(results) > 5 {
			t.Errorf("Search returned %d results, expected at most 5", len(results))
		}

		// First result should be from document1 (highest similarity)
		if results[0].Source != "document1.txt" {
			t.Errorf("Expected first result from document1.txt, got %s", results[0].Source)
		}

		// Verify embedding was deserialized correctly
		if len(results[0].Embedding) != 5 {
			t.Errorf("Expected embedding length 5, got %d", len(results[0].Embedding))
		}

		// Verify tags were parsed correctly
		if len(results[0].Tags) != 2 {
			t.Errorf("Expected 2 tags, got %d", len(results[0].Tags))
		}

		// Verify summary was retrieved
		if results[0].Summary != "Summary of document1" {
			t.Errorf("Expected summary 'Summary of document1', got '%s'", results[0].Summary)
		}
	})

	// Test 3: Library with GROUP BY source aggregation
	t.Run("Library", func(t *testing.T) {
		entries, err := store.Library(ctx)
		if err != nil {
			t.Fatalf("Library failed: %v", err)
		}

		// Should return 2 unique sources
		if len(entries) != 2 {
			t.Fatalf("Expected 2 library entries, got %d", len(entries))
		}

		// Find document1 entry
		var doc1Entry *LibraryEntry
		for i := range entries {
			if entries[i].Source == "document1.txt" {
				doc1Entry = &entries[i]
				break
			}
		}

		if doc1Entry == nil {
			t.Fatal("document1.txt not found in library")
		}

		// Verify chunk count
		if doc1Entry.ChunkCount != 2 {
			t.Errorf("Expected chunk count 2 for document1, got %d", doc1Entry.ChunkCount)
		}

		// Verify summary
		if doc1Entry.Summary != "Summary of document1" {
			t.Errorf("Expected summary 'Summary of document1', got '%s'", doc1Entry.Summary)
		}

		// Verify tags
		if len(doc1Entry.Tags) != 2 {
			t.Errorf("Expected 2 tags for document1, got %d", len(doc1Entry.Tags))
		}

		// Verify created_at is set
		if doc1Entry.CreatedAt.IsZero() {
			t.Error("CreatedAt should not be zero")
		}
	})

	// Test 4: DeleteSource removes all chunks
	t.Run("DeleteSource", func(t *testing.T) {
		// Delete document1
		err := store.DeleteChunksBySource(ctx, 1, "document1.txt")
		if err != nil {
			t.Fatalf("DeleteSource failed: %v", err)
		}

		// Verify library now has only 1 entry
		entries, err := store.Library(ctx)
		if err != nil {
			t.Fatalf("Library failed after delete: %v", err)
		}

		if len(entries) != 1 {
			t.Errorf("Expected 1 library entry after delete, got %d", len(entries))
		}

		if entries[0].Source != "document2.txt" {
			t.Errorf("Expected remaining source to be document2.txt, got %s", entries[0].Source)
		}

		// Verify search doesn't return deleted chunks
		queryVec := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
		results, err := store.Search(ctx, queryVec, 10)
		if err != nil {
			t.Fatalf("Search failed after delete: %v", err)
		}

		// Should only return chunks from document2
		for _, result := range results {
			if result.Source == "document1.txt" {
				t.Error("Search returned deleted chunks from document1.txt")
			}
		}
	})
}

// TestCosineSimilarity tests the cosine similarity calculation
func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float32
		b        []float32
		expected float64
		delta    float64
	}{
		{
			name:     "identical vectors",
			a:        []float32{1.0, 0.0, 0.0},
			b:        []float32{1.0, 0.0, 0.0},
			expected: 1.0,
			delta:    0.0001,
		},
		{
			name:     "orthogonal vectors",
			a:        []float32{1.0, 0.0},
			b:        []float32{0.0, 1.0},
			expected: 0.0,
			delta:    0.0001,
		},
		{
			name:     "opposite vectors",
			a:        []float32{1.0, 0.0},
			b:        []float32{-1.0, 0.0},
			expected: -1.0,
			delta:    0.0001,
		},
		{
			name:     "different lengths",
			a:        []float32{1.0, 0.0},
			b:        []float32{1.0, 0.0, 0.0},
			expected: 0.0,
			delta:    0.0001,
		},
		{
			name:     "zero vector",
			a:        []float32{0.0, 0.0},
			b:        []float32{1.0, 1.0},
			expected: 0.0,
			delta:    0.0001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cosineSimilarity(tt.a, tt.b)
			if result < tt.expected-tt.delta || result > tt.expected+tt.delta {
				t.Errorf("cosineSimilarity(%v, %v) = %f, expected %f", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

// TestEmbeddingSerialization tests embedding serialization and deserialization
func TestEmbeddingSerialization(t *testing.T) {
	tests := []struct {
		name      string
		embedding []float32
	}{
		{
			name:      "small embedding",
			embedding: []float32{0.1, 0.2, 0.3},
		},
		{
			name:      "large embedding",
			embedding: []float32{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
		},
		{
			name:      "negative values",
			embedding: []float32{-0.5, -0.3, 0.2, 0.8},
		},
		{
			name:      "zero values",
			embedding: []float32{0.0, 0.0, 0.0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize
			bytes := serializeEmbedding(tt.embedding)

			// Deserialize
			result := deserializeEmbedding(bytes)

			// Verify length
			if len(result) != len(tt.embedding) {
				t.Fatalf("Expected length %d, got %d", len(tt.embedding), len(result))
			}

			// Verify values (with small delta for floating point comparison)
			for i := range tt.embedding {
				delta := 0.0001
				if result[i] < tt.embedding[i]-float32(delta) || result[i] > tt.embedding[i]+float32(delta) {
					t.Errorf("Index %d: expected %f, got %f", i, tt.embedding[i], result[i])
				}
			}
		})
	}
}
