package store

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestNewStoreWithWALMode verifies that the store is configured with WAL mode
func TestNewStoreWithWALMode(t *testing.T) {
	tmpFile := "test_wal_mode.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-wal")
	defer os.Remove(tmpFile + "-shm")

	store, err := NewStore(tmpFile, "single")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Query the journal mode to verify WAL is enabled
	var journalMode string
	err = store.db.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	if err != nil {
		t.Fatalf("Failed to query journal mode: %v", err)
	}

	if journalMode != "wal" {
		t.Errorf("Expected journal mode 'wal', got '%s'", journalMode)
	}
}

// TestNewStoreConnectionPooling verifies connection pool configuration
func TestNewStoreConnectionPooling(t *testing.T) {
	tmpFile := "test_pool.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-wal")
	defer os.Remove(tmpFile + "-shm")

	store, err := NewStore(tmpFile, "single")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Verify connection pool settings via stats
	stats := store.db.Stats()

	// The pool should be configured (we can't directly check the limits,
	// but we can verify the connection is working)
	if stats.MaxOpenConnections != 25 {
		t.Errorf("Expected MaxOpenConnections to be 25, got %d", stats.MaxOpenConnections)
	}
}

// TestNewStoreBusyTimeout verifies busy timeout is configured
func TestNewStoreBusyTimeout(t *testing.T) {
	tmpFile := "test_busy_timeout.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-wal")
	defer os.Remove(tmpFile + "-shm")

	store, err := NewStore(tmpFile, "single")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Query the busy timeout to verify it's set
	var busyTimeout int
	err = store.db.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout)
	if err != nil {
		t.Fatalf("Failed to query busy timeout: %v", err)
	}

	if busyTimeout != 5000 {
		t.Errorf("Expected busy timeout 5000ms, got %dms", busyTimeout)
	}
}

// TestNewStoreUserMode verifies userMode is stored correctly
func TestNewStoreUserMode(t *testing.T) {
	tmpFile := "test_user_mode.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-wal")
	defer os.Remove(tmpFile + "-shm")

	tests := []struct {
		name     string
		userMode string
	}{
		{"single user mode", "single"},
		{"multi user mode", "multi"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := NewStore(tmpFile, tt.userMode)
			if err != nil {
				t.Fatalf("Failed to create store: %v", err)
			}
			defer store.Close()

			if store.userMode != tt.userMode {
				t.Errorf("Expected userMode '%s', got '%s'", tt.userMode, store.userMode)
			}
		})
	}
}

// TestNewStoreMigrations verifies migrations run on initialization
func TestNewStoreMigrations(t *testing.T) {
	tmpFile := "test_migrations.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-wal")
	defer os.Remove(tmpFile + "-shm")

	store, err := NewStore(tmpFile, "single")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Verify that the chunks table exists (created by migrations)
	var tableName string
	err = store.db.QueryRowContext(context.Background(),
		"SELECT name FROM sqlite_master WHERE type='table' AND name='chunks'").Scan(&tableName)
	if err != nil {
		t.Fatalf("Failed to query for chunks table: %v", err)
	}

	if tableName != "chunks" {
		t.Errorf("Expected chunks table to exist, but it doesn't")
	}
}

// TestConnectionPoolConcurrency verifies concurrent access works
func TestConnectionPoolConcurrency(t *testing.T) {
	tmpFile := "test_concurrent.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-wal")
	defer os.Remove(tmpFile + "-shm")

	store, err := NewStore(tmpFile, "single")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Spawn multiple goroutines to test concurrent access
	done := make(chan bool)
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			ctx := context.Background()

			// Try to save a chunk
			err := store.SaveChunk(ctx, 1, "concurrent_test", "test data",
				[]float32{0.1, 0.2, 0.3}, []string{"test"}, "summary")
			if err != nil {
				errors <- err
				done <- false
				return
			}

			// Try to search
			_, err = store.Search(ctx, []float32{0.1, 0.2, 0.3}, 5)
			if err != nil {
				errors <- err
				done <- false
				return
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	successCount := 0
	for i := 0; i < 10; i++ {
		select {
		case success := <-done:
			if success {
				successCount++
			}
		case err := <-errors:
			t.Errorf("Concurrent operation failed: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent operations")
		}
	}

	if successCount != 10 {
		t.Errorf("Expected 10 successful operations, got %d", successCount)
	}
}

// TestConnectionRecycling verifies connection max lifetime is set
func TestConnectionRecycling(t *testing.T) {
	tmpFile := "test_recycle.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-wal")
	defer os.Remove(tmpFile + "-shm")

	store, err := NewStore(tmpFile, "single")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// We can't directly test the max lifetime, but we can verify
	// the store is functional and connections work
	ctx := context.Background()

	// Perform multiple operations to exercise the connection pool
	for i := 0; i < 5; i++ {
		err := store.SaveChunk(ctx, 1, "test", "data",
			[]float32{0.1, 0.2}, []string{}, "")
		if err != nil {
			t.Fatalf("Failed to save chunk: %v", err)
		}
	}

	// Verify stats show connections are being used
	stats := store.db.Stats()
	if stats.OpenConnections == 0 {
		t.Error("Expected at least one open connection")
	}
}
