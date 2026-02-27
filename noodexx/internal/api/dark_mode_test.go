package api

import (
	"context"
	"noodexx/internal/store"
	"os"
	"testing"
)

// TestDarkModeStorage tests that dark mode preference can be stored and retrieved
func TestDarkModeStorage(t *testing.T) {
	// Create a test database
	tmpFile := t.TempDir() + "/test_dark_mode.db"
	defer os.Remove(tmpFile)

	testStore, err := store.NewStore(tmpFile, "multi")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer testStore.Close()

	ctx := context.Background()

	// Create a test user
	userID, err := testStore.CreateUser(ctx, "testuser", "password123", "test@example.com", false, false)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Verify default dark mode is false
	user, err := testStore.GetUserByID(ctx, userID)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	if user.DarkMode {
		t.Errorf("Expected default dark mode to be false, got true")
	}

	// Enable dark mode
	err = testStore.UpdateUserDarkMode(ctx, userID, true)
	if err != nil {
		t.Fatalf("Failed to update dark mode: %v", err)
	}

	// Verify dark mode was set
	user, err = testStore.GetUserByID(ctx, userID)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	if !user.DarkMode {
		t.Errorf("Expected dark mode to be true, got false")
	}

	// Disable dark mode
	err = testStore.UpdateUserDarkMode(ctx, userID, false)
	if err != nil {
		t.Fatalf("Failed to disable dark mode: %v", err)
	}

	// Verify dark mode was disabled
	user, err = testStore.GetUserByID(ctx, userID)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	if user.DarkMode {
		t.Errorf("Expected dark mode to be false, got true")
	}
}
