package store

import (
	"context"
	"fmt"
	"os"
	"testing"
)

func TestLibraryByUser(t *testing.T) {
	// Create a temporary database file
	tmpFile := "test_library_by_user.db"
	defer os.Remove(tmpFile)

	// Create a new store
	store, err := NewStore(tmpFile, "multi")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Create test users
	user1ID, err := store.CreateUser(ctx, "user1", "password1", "user1@test.com", false, false)
	if err != nil {
		t.Fatalf("Failed to create user1: %v", err)
	}

	user2ID, err := store.CreateUser(ctx, "user2", "password2", "user2@test.com", false, false)
	if err != nil {
		t.Fatalf("Failed to create user2: %v", err)
	}

	user3ID, err := store.CreateUser(ctx, "user3", "password3", "user3@test.com", false, false)
	if err != nil {
		t.Fatalf("Failed to create user3: %v", err)
	}

	// Save chunks with different visibility settings
	embedding := []float32{0.1, 0.2, 0.3, 0.4}
	tags := []string{"test"}

	// User1's private document
	err = store.SaveChunk(ctx, user1ID, "user1_private.txt", "Private content", embedding, tags, "Private doc")
	if err != nil {
		t.Fatalf("Failed to save user1 private chunk: %v", err)
	}

	// User2's private document
	err = store.SaveChunk(ctx, user2ID, "user2_private.txt", "Private content", embedding, tags, "Private doc")
	if err != nil {
		t.Fatalf("Failed to save user2 private chunk: %v", err)
	}

	// Test: User1 should only see their own private document
	entries, err := store.LibraryByUser(ctx, user1ID)
	if err != nil {
		t.Fatalf("Failed to get library for user1: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry for user1, got %d", len(entries))
	}

	if len(entries) > 0 && entries[0].Source != "user1_private.txt" {
		t.Errorf("Expected source 'user1_private.txt', got '%s'", entries[0].Source)
	}

	// Test: User2 should only see their own private document
	entries, err = store.LibraryByUser(ctx, user2ID)
	if err != nil {
		t.Fatalf("Failed to get library for user2: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry for user2, got %d", len(entries))
	}

	if len(entries) > 0 && entries[0].Source != "user2_private.txt" {
		t.Errorf("Expected source 'user2_private.txt', got '%s'", entries[0].Source)
	}

	// Test: User3 should see no documents
	entries, err = store.LibraryByUser(ctx, user3ID)
	if err != nil {
		t.Fatalf("Failed to get library for user3: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("Expected 0 entries for user3, got %d", len(entries))
	}
}

func TestLibraryByUserWithPublicDocuments(t *testing.T) {
	// Create a temporary database file
	tmpFile := "test_library_public.db"
	defer os.Remove(tmpFile)

	// Create a new store
	store, err := NewStore(tmpFile, "multi")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Create test users
	user1ID, err := store.CreateUser(ctx, "user1", "password1", "user1@test.com", false, false)
	if err != nil {
		t.Fatalf("Failed to create user1: %v", err)
	}

	user2ID, err := store.CreateUser(ctx, "user2", "password2", "user2@test.com", false, false)
	if err != nil {
		t.Fatalf("Failed to create user2: %v", err)
	}

	embedding := []float32{0.1, 0.2, 0.3, 0.4}
	tags := []string{"test"}

	// Save user1's private document
	err = store.SaveChunk(ctx, user1ID, "user1_private.txt", "Private content", embedding, tags, "Private doc")
	if err != nil {
		t.Fatalf("Failed to save user1 private chunk: %v", err)
	}

	// Manually insert a public document owned by user1
	_, err = store.db.ExecContext(ctx, `
		INSERT INTO chunks (user_id, source, text, embedding, tags, summary, visibility)
		VALUES (?, ?, ?, ?, ?, ?, 'public')
	`, user1ID, "public_doc.txt", "Public content", serializeEmbedding(embedding), joinTags(tags), "Public doc")
	if err != nil {
		t.Fatalf("Failed to insert public chunk: %v", err)
	}

	// Test: User1 should see both their private and public documents
	entries, err := store.LibraryByUser(ctx, user1ID)
	if err != nil {
		t.Fatalf("Failed to get library for user1: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("Expected 2 entries for user1, got %d", len(entries))
	}

	// Test: User2 should only see the public document
	entries, err = store.LibraryByUser(ctx, user2ID)
	if err != nil {
		t.Fatalf("Failed to get library for user2: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry for user2 (public doc), got %d", len(entries))
	}

	if len(entries) > 0 && entries[0].Source != "public_doc.txt" {
		t.Errorf("Expected source 'public_doc.txt', got '%s'", entries[0].Source)
	}
}

func TestLibraryByUserWithSharedDocuments(t *testing.T) {
	// Create a temporary database file
	tmpFile := "test_library_shared.db"
	defer os.Remove(tmpFile)

	// Create a new store
	store, err := NewStore(tmpFile, "multi")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Create test users
	user1ID, err := store.CreateUser(ctx, "user1", "password1", "user1@test.com", false, false)
	if err != nil {
		t.Fatalf("Failed to create user1: %v", err)
	}

	user2ID, err := store.CreateUser(ctx, "user2", "password2", "user2@test.com", false, false)
	if err != nil {
		t.Fatalf("Failed to create user2: %v", err)
	}

	user3ID, err := store.CreateUser(ctx, "user3", "password3", "user3@test.com", false, false)
	if err != nil {
		t.Fatalf("Failed to create user3: %v", err)
	}

	embedding := []float32{0.1, 0.2, 0.3, 0.4}
	tags := []string{"test"}

	// Save user1's private document
	err = store.SaveChunk(ctx, user1ID, "user1_private.txt", "Private content", embedding, tags, "Private doc")
	if err != nil {
		t.Fatalf("Failed to save user1 private chunk: %v", err)
	}

	// Manually insert a shared document owned by user1, shared with user2
	_, err = store.db.ExecContext(ctx, `
		INSERT INTO chunks (user_id, source, text, embedding, tags, summary, visibility, shared_with)
		VALUES (?, ?, ?, ?, ?, ?, 'shared', ?)
	`, user1ID, "shared_doc.txt", "Shared content", serializeEmbedding(embedding), joinTags(tags), "Shared doc", fmt.Sprintf("%d", user2ID))
	if err != nil {
		t.Fatalf("Failed to insert shared chunk: %v", err)
	}

	// Test: User1 should see both their private and shared documents
	entries, err := store.LibraryByUser(ctx, user1ID)
	if err != nil {
		t.Fatalf("Failed to get library for user1: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("Expected 2 entries for user1, got %d", len(entries))
	}

	// Test: User2 should see the shared document
	entries, err = store.LibraryByUser(ctx, user2ID)
	if err != nil {
		t.Fatalf("Failed to get library for user2: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry for user2 (shared doc), got %d", len(entries))
	}

	if len(entries) > 0 && entries[0].Source != "shared_doc.txt" {
		t.Errorf("Expected source 'shared_doc.txt', got '%s'", entries[0].Source)
	}

	// Test: User3 should see no documents
	entries, err = store.LibraryByUser(ctx, user3ID)
	if err != nil {
		t.Fatalf("Failed to get library for user3: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("Expected 0 entries for user3, got %d", len(entries))
	}
}

func TestLibraryByUserWithMultipleSharedUsers(t *testing.T) {
	// Create a temporary database file
	tmpFile := "test_library_multi_shared.db"
	defer os.Remove(tmpFile)

	// Create a new store
	store, err := NewStore(tmpFile, "multi")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Create test users
	user1ID, err := store.CreateUser(ctx, "user1", "password1", "user1@test.com", false, false)
	if err != nil {
		t.Fatalf("Failed to create user1: %v", err)
	}

	user2ID, err := store.CreateUser(ctx, "user2", "password2", "user2@test.com", false, false)
	if err != nil {
		t.Fatalf("Failed to create user2: %v", err)
	}

	user3ID, err := store.CreateUser(ctx, "user3", "password3", "user3@test.com", false, false)
	if err != nil {
		t.Fatalf("Failed to create user3: %v", err)
	}

	user4ID, err := store.CreateUser(ctx, "user4", "password4", "user4@test.com", false, false)
	if err != nil {
		t.Fatalf("Failed to create user4: %v", err)
	}

	embedding := []float32{0.1, 0.2, 0.3, 0.4}
	tags := []string{"test"}

	// Manually insert a shared document owned by user1, shared with user2 and user3
	_, err = store.db.ExecContext(ctx, `
		INSERT INTO chunks (user_id, source, text, embedding, tags, summary, visibility, shared_with)
		VALUES (?, ?, ?, ?, ?, ?, 'shared', ?)
	`, user1ID, "multi_shared_doc.txt", "Multi-shared content", serializeEmbedding(embedding), joinTags(tags), "Multi-shared doc", fmt.Sprintf("%d,%d", user2ID, user3ID))
	if err != nil {
		t.Fatalf("Failed to insert multi-shared chunk: %v", err)
	}

	// Test: User1 (owner) should see the document
	entries, err := store.LibraryByUser(ctx, user1ID)
	if err != nil {
		t.Fatalf("Failed to get library for user1: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry for user1, got %d", len(entries))
	}

	// Test: User2 (shared with) should see the document
	entries, err = store.LibraryByUser(ctx, user2ID)
	if err != nil {
		t.Fatalf("Failed to get library for user2: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry for user2, got %d", len(entries))
	}

	// Test: User3 (shared with) should see the document
	entries, err = store.LibraryByUser(ctx, user3ID)
	if err != nil {
		t.Fatalf("Failed to get library for user3: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry for user3, got %d", len(entries))
	}

	// Test: User4 (not shared with) should NOT see the document
	entries, err = store.LibraryByUser(ctx, user4ID)
	if err != nil {
		t.Fatalf("Failed to get library for user4: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("Expected 0 entries for user4, got %d", len(entries))
	}
}
