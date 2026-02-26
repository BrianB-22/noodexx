package store

import (
	"context"
	"database/sql"
	"os"
	"testing"
)

// TestSearchByUserStandalone is a minimal test that doesn't depend on other broken tests
func TestSearchByUserStandalone(t *testing.T) {
	tmpFile := "test_search_standalone.db"
	defer os.Remove(tmpFile)

	// Create store manually to avoid migration issues
	db, err := sql.Open("sqlite", tmpFile+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	store := &Store{db: db, userMode: "multi"}

	// Run migrations
	if err := store.runMigrations(context.Background()); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

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

	// Save chunks
	embedding1 := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
	embedding2 := []float32{0.9, 0.8, 0.7, 0.6, 0.5}

	// User1's private chunk
	err = store.SaveChunk(ctx, user1ID, "user1_doc.txt", "User1 content", embedding1, []string{"test"}, "Summary")
	if err != nil {
		t.Fatalf("Failed to save user1 chunk: %v", err)
	}

	// User2's private chunk
	err = store.SaveChunk(ctx, user2ID, "user2_doc.txt", "User2 content", embedding2, []string{"test"}, "Summary")
	if err != nil {
		t.Fatalf("Failed to save user2 chunk: %v", err)
	}

	// Public chunk
	err = store.SaveChunk(ctx, user1ID, "public_doc.txt", "Public content", embedding1, []string{"test"}, "Summary")
	if err != nil {
		t.Fatalf("Failed to save public chunk: %v", err)
	}
	_, err = store.db.ExecContext(ctx, "UPDATE chunks SET visibility = 'public' WHERE source = 'public_doc.txt'")
	if err != nil {
		t.Fatalf("Failed to set public visibility: %v", err)
	}

	// Test: User1 should see their own chunk and public chunk (2 total)
	queryVec := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
	results, err := store.SearchByUser(ctx, user1ID, queryVec, 10)
	if err != nil {
		t.Fatalf("SearchByUser failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results for user1, got %d", len(results))
		for i, r := range results {
			t.Logf("Result %d: %s", i, r.Source)
		}
	}

	// Verify user1 doesn't see user2's private chunk
	for _, result := range results {
		if result.Source == "user2_doc.txt" {
			t.Error("User1 should not see user2's private chunk")
		}
	}

	// Test: User2 should see their own chunk and public chunk (2 total)
	results, err = store.SearchByUser(ctx, user2ID, queryVec, 10)
	if err != nil {
		t.Fatalf("SearchByUser failed for user2: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results for user2, got %d", len(results))
	}

	// Verify user2 doesn't see user1's private chunk
	for _, result := range results {
		if result.Source == "user1_doc.txt" {
			t.Error("User2 should not see user1's private chunk")
		}
	}

	t.Log("SearchByUser implementation verified successfully!")
}
