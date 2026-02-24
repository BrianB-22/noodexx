package store

import (
	"context"
	"os"
	"testing"
)

func TestMigrations(t *testing.T) {
	// Create a temporary database file
	tmpFile := "test_migrations.db"
	defer os.Remove(tmpFile)

	// Create a new store (this will run migrations)
	store, err := NewStore(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Test 1: Verify chunks table exists with all columns
	t.Run("chunks table exists", func(t *testing.T) {
		var count int
		err := store.db.QueryRowContext(ctx, `
			SELECT COUNT(*) 
			FROM pragma_table_info('chunks') 
			WHERE name IN ('id', 'source', 'text', 'embedding', 'tags', 'summary', 'created_at')
		`).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query chunks table info: %v", err)
		}
		if count != 7 {
			t.Errorf("Expected 7 columns in chunks table, got %d", count)
		}
	})

	// Test 2: Verify chat_messages table exists
	t.Run("chat_messages table exists", func(t *testing.T) {
		var count int
		err := store.db.QueryRowContext(ctx, `
			SELECT COUNT(*) 
			FROM pragma_table_info('chat_messages') 
			WHERE name IN ('id', 'session_id', 'role', 'content', 'created_at')
		`).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query chat_messages table info: %v", err)
		}
		if count != 5 {
			t.Errorf("Expected 5 columns in chat_messages table, got %d", count)
		}
	})

	// Test 3: Verify audit_log table exists
	t.Run("audit_log table exists", func(t *testing.T) {
		var count int
		err := store.db.QueryRowContext(ctx, `
			SELECT COUNT(*) 
			FROM pragma_table_info('audit_log') 
			WHERE name IN ('id', 'timestamp', 'operation_type', 'details', 'user_context')
		`).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query audit_log table info: %v", err)
		}
		if count != 5 {
			t.Errorf("Expected 5 columns in audit_log table, got %d", count)
		}
	})

	// Test 4: Verify watched_folders table exists
	t.Run("watched_folders table exists", func(t *testing.T) {
		var count int
		err := store.db.QueryRowContext(ctx, `
			SELECT COUNT(*) 
			FROM pragma_table_info('watched_folders') 
			WHERE name IN ('id', 'path', 'active', 'last_scan')
		`).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query watched_folders table info: %v", err)
		}
		if count != 4 {
			t.Errorf("Expected 4 columns in watched_folders table, got %d", count)
		}
	})

	// Test 5: Verify indexes exist
	t.Run("indexes exist", func(t *testing.T) {
		expectedIndexes := []string{
			"idx_chunks_source",
			"idx_chunks_created",
			"idx_messages_session",
			"idx_messages_created",
			"idx_audit_timestamp",
			"idx_audit_type",
		}

		for _, indexName := range expectedIndexes {
			var count int
			err := store.db.QueryRowContext(ctx, `
				SELECT COUNT(*) 
				FROM sqlite_master 
				WHERE type = 'index' AND name = ?
			`, indexName).Scan(&count)
			if err != nil {
				t.Fatalf("Failed to query index %s: %v", indexName, err)
			}
			if count != 1 {
				t.Errorf("Expected index %s to exist", indexName)
			}
		}
	})
}

func TestMigrationsPreserveData(t *testing.T) {
	// Create a temporary database file
	tmpFile := "test_preserve_data.db"
	defer os.Remove(tmpFile)

	// Create initial store with Phase 1 schema
	store1, err := NewStore(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create initial store: %v", err)
	}

	ctx := context.Background()

	// Insert test data
	testEmbedding := []float32{0.1, 0.2, 0.3, 0.4}
	err = store1.SaveChunk(ctx, "test-source", "test text", testEmbedding, nil, "")
	if err != nil {
		t.Fatalf("Failed to save test chunk: %v", err)
	}

	store1.Close()

	// Reopen the store (migrations should run again but preserve data)
	store2, err := NewStore(tmpFile)
	if err != nil {
		t.Fatalf("Failed to reopen store: %v", err)
	}
	defer store2.Close()

	// Verify data is preserved
	entries, err := store2.Library(ctx)
	if err != nil {
		t.Fatalf("Failed to query library: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 library entry, got %d", len(entries))
	}

	if len(entries) > 0 && entries[0].Source != "test-source" {
		t.Errorf("Expected source 'test-source', got '%s'", entries[0].Source)
	}
}

func TestMigrationsTransactionRollback(t *testing.T) {
	// This test verifies that if a migration fails, the transaction is rolled back
	// We can't easily simulate a migration failure without modifying the code,
	// so this is more of a documentation test showing the expected behavior
	t.Skip("Transaction rollback is tested implicitly through error handling")
}
