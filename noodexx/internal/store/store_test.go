package store

import (
	"context"
	"os"
	"testing"
)

func TestNewStore(t *testing.T) {
	// Create a temporary database file
	tmpFile := "test_store.db"
	defer os.Remove(tmpFile)

	// Create a new store
	store, err := NewStore(tmpFile, "single")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Verify the store is not nil
	if store == nil {
		t.Fatal("Store is nil")
	}

	// Verify the database connection is working
	if store.db == nil {
		t.Fatal("Database connection is nil")
	}
}

func TestStoreClose(t *testing.T) {
	// Create a temporary database file
	tmpFile := "test_store_close.db"
	defer os.Remove(tmpFile)

	// Create a new store
	store, err := NewStore(tmpFile, "single")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Close the store
	err = store.Close()
	if err != nil {
		t.Fatalf("Failed to close store: %v", err)
	}
}

func TestSaveAndSearchChunk(t *testing.T) {
	// Create a temporary database file
	tmpFile := "test_save_search.db"
	defer os.Remove(tmpFile)

	// Create a new store
	store, err := NewStore(tmpFile, "single")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Save a chunk
	source := "test_document.txt"
	text := "This is a test chunk"
	embedding := []float32{0.1, 0.2, 0.3, 0.4}
	tags := []string{"test", "example"}
	summary := "Test summary"

	err = store.SaveChunk(ctx, 1, source, text, embedding, tags, summary)
	if err != nil {
		t.Fatalf("Failed to save chunk: %v", err)
	}

	// Search for the chunk
	queryVec := []float32{0.1, 0.2, 0.3, 0.4}
	results, err := store.Search(ctx, queryVec, 10)
	if err != nil {
		t.Fatalf("Failed to search: %v", err)
	}

	// Verify we got results
	if len(results) == 0 {
		t.Fatal("No results returned from search")
	}

	// Verify the first result matches what we saved
	if results[0].Source != source {
		t.Errorf("Expected source %s, got %s", source, results[0].Source)
	}
	if results[0].Text != text {
		t.Errorf("Expected text %s, got %s", text, results[0].Text)
	}
}
