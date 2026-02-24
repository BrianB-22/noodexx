package store

import (
	"context"
	"path/filepath"
	"testing"
)

func TestWatchedFolderOperations(t *testing.T) {
	// Create a temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create store
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Test AddWatchedFolder
	t.Run("AddWatchedFolder", func(t *testing.T) {
		testPath := "/test/path/documents"
		err := store.AddWatchedFolder(ctx, testPath)
		if err != nil {
			t.Errorf("AddWatchedFolder failed: %v", err)
		}

		// Verify it was added
		folders, err := store.GetWatchedFolders(ctx)
		if err != nil {
			t.Errorf("GetWatchedFolders failed: %v", err)
		}

		if len(folders) != 1 {
			t.Errorf("Expected 1 folder, got %d", len(folders))
		}

		if folders[0].Path != testPath {
			t.Errorf("Expected path %s, got %s", testPath, folders[0].Path)
		}

		if !folders[0].Active {
			t.Errorf("Expected folder to be active")
		}
	})

	// Test GetWatchedFolders with multiple folders
	t.Run("GetWatchedFolders_Multiple", func(t *testing.T) {
		// Add more folders
		paths := []string{"/test/path/notes", "/test/path/projects"}
		for _, path := range paths {
			err := store.AddWatchedFolder(ctx, path)
			if err != nil {
				t.Errorf("AddWatchedFolder failed for %s: %v", path, err)
			}
		}

		// Get all folders
		folders, err := store.GetWatchedFolders(ctx)
		if err != nil {
			t.Errorf("GetWatchedFolders failed: %v", err)
		}

		// Should have 3 folders total (1 from previous test + 2 new)
		if len(folders) != 3 {
			t.Errorf("Expected 3 folders, got %d", len(folders))
		}

		// Verify folders are sorted by path
		expectedPaths := []string{"/test/path/documents", "/test/path/notes", "/test/path/projects"}
		for i, folder := range folders {
			if folder.Path != expectedPaths[i] {
				t.Errorf("Expected path %s at index %d, got %s", expectedPaths[i], i, folder.Path)
			}
		}
	})

	// Test RemoveWatchedFolder
	t.Run("RemoveWatchedFolder", func(t *testing.T) {
		pathToRemove := "/test/path/notes"
		err := store.RemoveWatchedFolder(ctx, pathToRemove)
		if err != nil {
			t.Errorf("RemoveWatchedFolder failed: %v", err)
		}

		// Verify it was removed
		folders, err := store.GetWatchedFolders(ctx)
		if err != nil {
			t.Errorf("GetWatchedFolders failed: %v", err)
		}

		if len(folders) != 2 {
			t.Errorf("Expected 2 folders after removal, got %d", len(folders))
		}

		// Verify the removed path is not in the list
		for _, folder := range folders {
			if folder.Path == pathToRemove {
				t.Errorf("Path %s should have been removed", pathToRemove)
			}
		}
	})

	// Test duplicate path handling
	t.Run("AddWatchedFolder_Duplicate", func(t *testing.T) {
		duplicatePath := "/test/path/documents"
		err := store.AddWatchedFolder(ctx, duplicatePath)
		if err == nil {
			t.Errorf("Expected error when adding duplicate path, got nil")
		}
	})

	// Test RemoveWatchedFolder with non-existent path
	t.Run("RemoveWatchedFolder_NonExistent", func(t *testing.T) {
		nonExistentPath := "/test/path/nonexistent"
		err := store.RemoveWatchedFolder(ctx, nonExistentPath)
		// Should not error even if path doesn't exist
		if err != nil {
			t.Errorf("RemoveWatchedFolder failed for non-existent path: %v", err)
		}
	})
}

func TestWatchedFolderTimestamps(t *testing.T) {
	// Create a temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create store
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Add a watched folder
	testPath := "/test/path/timestamps"
	err = store.AddWatchedFolder(ctx, testPath)
	if err != nil {
		t.Fatalf("AddWatchedFolder failed: %v", err)
	}

	// Get the folder and check timestamp
	folders, err := store.GetWatchedFolders(ctx)
	if err != nil {
		t.Fatalf("GetWatchedFolders failed: %v", err)
	}

	if len(folders) != 1 {
		t.Fatalf("Expected 1 folder, got %d", len(folders))
	}

	// Verify LastScan timestamp is set
	if folders[0].LastScan.IsZero() {
		t.Errorf("Expected LastScan timestamp to be set, got zero value")
	}
}

func TestWatchedFolderEmptyList(t *testing.T) {
	// Create a temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create store
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Get folders from empty database
	folders, err := store.GetWatchedFolders(ctx)
	if err != nil {
		t.Errorf("GetWatchedFolders failed on empty database: %v", err)
	}

	if len(folders) != 0 {
		t.Errorf("Expected 0 folders in empty database, got %d", len(folders))
	}
}

// TestWatchedFolderIntegration tests the complete workflow
func TestWatchedFolderIntegration(t *testing.T) {
	// Create a temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create store
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Simulate a complete workflow
	folders := []string{
		"/home/user/documents",
		"/home/user/notes",
		"/home/user/projects",
	}

	// Add all folders
	for _, folder := range folders {
		if err := store.AddWatchedFolder(ctx, folder); err != nil {
			t.Errorf("Failed to add folder %s: %v", folder, err)
		}
	}

	// Verify all were added
	watchedFolders, err := store.GetWatchedFolders(ctx)
	if err != nil {
		t.Fatalf("GetWatchedFolders failed: %v", err)
	}

	if len(watchedFolders) != len(folders) {
		t.Errorf("Expected %d folders, got %d", len(folders), len(watchedFolders))
	}

	// Remove one folder
	if err := store.RemoveWatchedFolder(ctx, folders[1]); err != nil {
		t.Errorf("Failed to remove folder: %v", err)
	}

	// Verify removal
	watchedFolders, err = store.GetWatchedFolders(ctx)
	if err != nil {
		t.Fatalf("GetWatchedFolders failed: %v", err)
	}

	if len(watchedFolders) != len(folders)-1 {
		t.Errorf("Expected %d folders after removal, got %d", len(folders)-1, len(watchedFolders))
	}

	// Verify the correct folder was removed
	for _, wf := range watchedFolders {
		if wf.Path == folders[1] {
			t.Errorf("Folder %s should have been removed", folders[1])
		}
	}
}
