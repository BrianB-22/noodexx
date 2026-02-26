package store

import (
	"context"
	"fmt"
	"os"
	"testing"
)

// TestSearchByUser tests the SearchByUser method with visibility filtering
func TestSearchByUser(t *testing.T) {
	// Create a temporary database file
	tmpFile := "test_search_by_user.db"
	defer os.Remove(tmpFile)

	// Create a new store in multi-user mode
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

	// Test embedding vectors
	embedding1 := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
	embedding2 := []float32{0.9, 0.8, 0.7, 0.6, 0.5}
	embedding3 := []float32{0.5, 0.5, 0.5, 0.5, 0.5}

	// Save chunks with different ownership and visibility
	// User1's private chunk
	err = store.SaveChunk(ctx, user1ID, "user1_private.txt", "User1 private content", embedding1, []string{"private"}, "Private doc")
	if err != nil {
		t.Fatalf("Failed to save user1 private chunk: %v", err)
	}

	// User2's private chunk
	err = store.SaveChunk(ctx, user2ID, "user2_private.txt", "User2 private content", embedding2, []string{"private"}, "Private doc")
	if err != nil {
		t.Fatalf("Failed to save user2 private chunk: %v", err)
	}

	// Public chunk owned by user1
	err = store.SaveChunk(ctx, user1ID, "public_doc.txt", "Public content", embedding3, []string{"public"}, "Public doc")
	if err != nil {
		t.Fatalf("Failed to save public chunk: %v", err)
	}

	// Manually set visibility to public for the public chunk
	_, err = store.db.ExecContext(ctx, "UPDATE chunks SET visibility = 'public' WHERE source = 'public_doc.txt'")
	if err != nil {
		t.Fatalf("Failed to set public visibility: %v", err)
	}

	// Shared chunk owned by user2, shared with user1
	err = store.SaveChunk(ctx, user2ID, "shared_doc.txt", "Shared content", embedding1, []string{"shared"}, "Shared doc")
	if err != nil {
		t.Fatalf("Failed to save shared chunk: %v", err)
	}

	// Manually set shared_with for the shared chunk
	// Convert user1ID to string for the shared_with column
	sharedWith := fmt.Sprintf("%d", user1ID)
	_, err = store.db.ExecContext(ctx, "UPDATE chunks SET visibility = 'shared', shared_with = ? WHERE source = 'shared_doc.txt'", sharedWith)
	if err != nil {
		t.Fatalf("Failed to set shared_with: %v", err)
	}

	// Test 1: User1 should see their own private chunk, public chunk, and shared chunk
	t.Run("User1 visibility", func(t *testing.T) {
		queryVec := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
		results, err := store.SearchByUser(ctx, user1ID, queryVec, 10)
		if err != nil {
			t.Fatalf("SearchByUser failed for user1: %v", err)
		}

		// User1 should see 3 chunks: their private, public, and shared
		if len(results) != 3 {
			t.Errorf("Expected 3 results for user1, got %d", len(results))
		}

		// Verify user1 can see their own private chunk
		foundPrivate := false
		foundPublic := false
		foundShared := false
		for _, result := range results {
			if result.Source == "user1_private.txt" {
				foundPrivate = true
			}
			if result.Source == "public_doc.txt" {
				foundPublic = true
			}
			if result.Source == "shared_doc.txt" {
				foundShared = true
			}
			// Should NOT see user2's private chunk
			if result.Source == "user2_private.txt" {
				t.Error("User1 should not see user2's private chunk")
			}
		}

		if !foundPrivate {
			t.Error("User1 should see their own private chunk")
		}
		if !foundPublic {
			t.Error("User1 should see public chunk")
		}
		if !foundShared {
			t.Error("User1 should see shared chunk")
		}
	})

	// Test 2: User2 should see their own private chunk, public chunk, but NOT the shared chunk (not shared with them)
	t.Run("User2 visibility", func(t *testing.T) {
		queryVec := []float32{0.9, 0.8, 0.7, 0.6, 0.5}
		results, err := store.SearchByUser(ctx, user2ID, queryVec, 10)
		if err != nil {
			t.Fatalf("SearchByUser failed for user2: %v", err)
		}

		// User2 should see 3 chunks: their private, their shared, and public
		if len(results) != 3 {
			t.Errorf("Expected 3 results for user2, got %d", len(results))
		}

		// Verify visibility
		foundPrivate := false
		foundPublic := false
		foundShared := false
		for _, result := range results {
			if result.Source == "user2_private.txt" {
				foundPrivate = true
			}
			if result.Source == "public_doc.txt" {
				foundPublic = true
			}
			if result.Source == "shared_doc.txt" {
				foundShared = true
			}
			// Should NOT see user1's private chunk
			if result.Source == "user1_private.txt" {
				t.Error("User2 should not see user1's private chunk")
			}
		}

		if !foundPrivate {
			t.Error("User2 should see their own private chunk")
		}
		if !foundPublic {
			t.Error("User2 should see public chunk")
		}
		if !foundShared {
			t.Error("User2 should see their own shared chunk")
		}
	})

	// Test 3: User3 should only see the public chunk
	t.Run("User3 visibility", func(t *testing.T) {
		queryVec := []float32{0.5, 0.5, 0.5, 0.5, 0.5}
		results, err := store.SearchByUser(ctx, user3ID, queryVec, 10)
		if err != nil {
			t.Fatalf("SearchByUser failed for user3: %v", err)
		}

		// User3 should only see 1 chunk: the public one
		if len(results) != 1 {
			t.Errorf("Expected 1 result for user3, got %d", len(results))
		}

		if len(results) > 0 && results[0].Source != "public_doc.txt" {
			t.Errorf("User3 should only see public chunk, got %s", results[0].Source)
		}
	})

	// Test 4: Verify results are sorted by similarity score
	t.Run("Results sorted by score", func(t *testing.T) {
		// Query vector identical to embedding1
		queryVec := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
		results, err := store.SearchByUser(ctx, user1ID, queryVec, 10)
		if err != nil {
			t.Fatalf("SearchByUser failed: %v", err)
		}

		if len(results) < 2 {
			t.Fatal("Need at least 2 results to test sorting")
		}

		// First result should be user1_private.txt or shared_doc.txt (both have embedding1)
		// which are most similar to the query
		if results[0].Source != "user1_private.txt" && results[0].Source != "shared_doc.txt" {
			t.Errorf("Expected first result to be most similar, got %s", results[0].Source)
		}
	})

	// Test 5: Verify topK limit is respected
	t.Run("TopK limit", func(t *testing.T) {
		queryVec := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
		results, err := store.SearchByUser(ctx, user1ID, queryVec, 2)
		if err != nil {
			t.Fatalf("SearchByUser failed: %v", err)
		}

		// Should return at most 2 results
		if len(results) > 2 {
			t.Errorf("Expected at most 2 results with topK=2, got %d", len(results))
		}
	})
}

// TestSearchByUserEmptyResults tests SearchByUser when no chunks are visible
func TestSearchByUserEmptyResults(t *testing.T) {
	tmpFile := "test_search_empty.db"
	defer os.Remove(tmpFile)

	store, err := NewStore(tmpFile, "multi")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Create a user
	userID, err := store.CreateUser(ctx, "testuser", "password", "test@test.com", false, false)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Search with no chunks in database
	queryVec := []float32{0.1, 0.2, 0.3}
	results, err := store.SearchByUser(ctx, userID, queryVec, 10)
	if err != nil {
		t.Fatalf("SearchByUser failed: %v", err)
	}

	// Should return empty results, not error
	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}
