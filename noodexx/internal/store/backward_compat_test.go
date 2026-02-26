package store

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "modernc.org/sqlite"
)

// TestBackwardCompatibilityWithPhase1 verifies that Phase 2 migrations work with Phase 1 databases
func TestBackwardCompatibilityWithPhase1(t *testing.T) {
	tmpFile := "test_phase1_compat.db"
	defer os.Remove(tmpFile)

	ctx := context.Background()

	// Step 1: Create a Phase 1 database (without tags and summary columns)
	db, err := sql.Open("sqlite", tmpFile+"?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Create Phase 1 schema (chunks table without tags and summary)
	_, err = db.Exec(`
		CREATE TABLE chunks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source TEXT NOT NULL,
			text TEXT NOT NULL,
			embedding BLOB NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create Phase 1 chunks table: %v", err)
	}

	// Insert Phase 1 test data
	testEmbedding := serializeEmbedding([]float32{0.1, 0.2, 0.3, 0.4, 0.5})
	_, err = db.Exec(`INSERT INTO chunks (source, text, embedding) VALUES (?, ?, ?)`,
		"phase1-doc.txt", "This is Phase 1 test data", testEmbedding)
	if err != nil {
		t.Fatalf("Failed to insert Phase 1 data: %v", err)
	}

	// Insert another chunk
	_, err = db.Exec(`INSERT INTO chunks (source, text, embedding) VALUES (?, ?, ?)`,
		"phase1-doc.txt", "More Phase 1 content", testEmbedding)
	if err != nil {
		t.Fatalf("Failed to insert second Phase 1 chunk: %v", err)
	}

	db.Close()

	// Step 2: Open with Phase 2 Store (this should run migrations)
	store, err := NewStore(tmpFile, "single")
	if err != nil {
		t.Fatalf("Failed to open Phase 2 store: %v", err)
	}
	defer store.Close()

	// Step 3: Verify Phase 1 data is preserved
	t.Run("Phase 1 data preserved", func(t *testing.T) {
		entries, err := store.Library(ctx)
		if err != nil {
			t.Fatalf("Failed to query library: %v", err)
		}

		if len(entries) != 1 {
			t.Errorf("Expected 1 library entry, got %d", len(entries))
		}

		if len(entries) > 0 {
			if entries[0].Source != "phase1-doc.txt" {
				t.Errorf("Expected source 'phase1-doc.txt', got '%s'", entries[0].Source)
			}
			if entries[0].ChunkCount != 2 {
				t.Errorf("Expected 2 chunks, got %d", entries[0].ChunkCount)
			}
		}
	})

	// Step 4: Verify new columns exist and can be used
	t.Run("New columns exist", func(t *testing.T) {
		// Try to save a chunk with tags and summary
		err := store.SaveChunk(ctx, 1, "phase2-doc.txt", "Phase 2 content",
			[]float32{0.6, 0.7, 0.8, 0.9, 1.0},
			[]string{"test", "phase2"},
			"This is a Phase 2 document with tags and summary")
		if err != nil {
			t.Fatalf("Failed to save Phase 2 chunk: %v", err)
		}

		// Verify it was saved correctly
		entries, err := store.Library(ctx)
		if err != nil {
			t.Fatalf("Failed to query library: %v", err)
		}

		if len(entries) != 2 {
			t.Errorf("Expected 2 library entries, got %d", len(entries))
		}

		// Find the Phase 2 entry
		var phase2Entry *LibraryEntry
		for i := range entries {
			if entries[i].Source == "phase2-doc.txt" {
				phase2Entry = &entries[i]
				break
			}
		}

		if phase2Entry == nil {
			t.Fatal("Phase 2 entry not found")
		}

		if len(phase2Entry.Tags) != 2 {
			t.Errorf("Expected 2 tags, got %d", len(phase2Entry.Tags))
		}

		if phase2Entry.Summary != "This is a Phase 2 document with tags and summary" {
			t.Errorf("Summary mismatch: got '%s'", phase2Entry.Summary)
		}
	})

	// Step 5: Verify new tables exist
	t.Run("New tables exist", func(t *testing.T) {
		// Test chat_messages table
		err := store.SaveMessage(ctx, "test-session", "user", "Hello")
		if err != nil {
			t.Fatalf("Failed to save message: %v", err)
		}

		messages, err := store.GetSessionHistory(ctx, "test-session")
		if err != nil {
			t.Fatalf("Failed to get session history: %v", err)
		}
		if len(messages) != 1 {
			t.Errorf("Expected 1 message, got %d", len(messages))
		}

		// Test audit_log table
		err = store.AddAuditEntry(ctx, "test", "test details", "test context")
		if err != nil {
			t.Fatalf("Failed to add audit entry: %v", err)
		}

		// Test watched_folders table
		err = store.AddWatchedFolder(ctx, 1, "/test/path")
		if err != nil {
			t.Fatalf("Failed to add watched folder: %v", err)
		}

		folders, err := store.GetWatchedFolders(ctx)
		if err != nil {
			t.Fatalf("Failed to get watched folders: %v", err)
		}
		if len(folders) != 1 {
			t.Errorf("Expected 1 watched folder, got %d", len(folders))
		}
	})

	// Step 6: Verify search still works with Phase 1 embeddings
	t.Run("Search works with Phase 1 embeddings", func(t *testing.T) {
		queryVec := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
		results, err := store.Search(ctx, queryVec, 5)
		if err != nil {
			t.Fatalf("Failed to search: %v", err)
		}

		if len(results) < 2 {
			t.Errorf("Expected at least 2 search results, got %d", len(results))
		}

		// Verify Phase 1 chunks are in results
		foundPhase1 := false
		for _, result := range results {
			if result.Source == "phase1-doc.txt" {
				foundPhase1 = true
				break
			}
		}

		if !foundPhase1 {
			t.Error("Phase 1 chunks not found in search results")
		}
	})
}

// TestMigrationIdempotency verifies that running migrations multiple times is safe
func TestMigrationIdempotency(t *testing.T) {
	tmpFile := "test_idempotency.db"
	defer os.Remove(tmpFile)

	ctx := context.Background()

	// Create and close store multiple times
	for i := 0; i < 3; i++ {
		store, err := NewStore(tmpFile, "single")
		if err != nil {
			t.Fatalf("Failed to create store on iteration %d: %v", i, err)
		}

		// Add some data
		err = store.SaveChunk(ctx, 1, "test-source", "test text",
			[]float32{0.1, 0.2, 0.3}, nil, "")
		if err != nil {
			t.Fatalf("Failed to save chunk on iteration %d: %v", i, err)
		}

		store.Close()
	}

	// Verify data is correct
	store, err := NewStore(tmpFile, "single")
	if err != nil {
		t.Fatalf("Failed to open final store: %v", err)
	}
	defer store.Close()

	entries, err := store.Library(ctx)
	if err != nil {
		t.Fatalf("Failed to query library: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 library entry, got %d", len(entries))
	}

	if len(entries) > 0 && entries[0].ChunkCount != 3 {
		t.Errorf("Expected 3 chunks, got %d", entries[0].ChunkCount)
	}
}
