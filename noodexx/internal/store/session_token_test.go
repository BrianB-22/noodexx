package store

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestSessionTokenManagement(t *testing.T) {
	// Create a temporary database
	dbPath := "test_session_tokens.db"
	defer os.Remove(dbPath)

	store, err := NewStore(dbPath, "single")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Create a test user
	userID, err := store.CreateUser(ctx, "testuser", "password123", "test@example.com", false, false)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Test CreateSessionToken
	t.Run("CreateSessionToken", func(t *testing.T) {
		token := "test-token-123"
		expiresAt := time.Now().Add(24 * time.Hour)

		err := store.CreateSessionToken(ctx, token, userID, expiresAt)
		if err != nil {
			t.Errorf("CreateSessionToken failed: %v", err)
		}
	})

	// Test GetSessionToken - valid token
	t.Run("GetSessionToken_Valid", func(t *testing.T) {
		token := "test-token-456"
		expiresAt := time.Now().Add(24 * time.Hour)

		err := store.CreateSessionToken(ctx, token, userID, expiresAt)
		if err != nil {
			t.Fatalf("Failed to create token: %v", err)
		}

		st, err := store.GetSessionToken(ctx, token)
		if err != nil {
			t.Errorf("GetSessionToken failed: %v", err)
		}
		if st == nil {
			t.Error("Expected token to be found, got nil")
		}
		if st != nil && st.UserID != userID {
			t.Errorf("Expected userID %d, got %d", userID, st.UserID)
		}
	})

	// Test GetSessionToken - expired token
	t.Run("GetSessionToken_Expired", func(t *testing.T) {
		token := "test-token-expired"
		expiresAt := time.Now().Add(-1 * time.Hour) // Expired 1 hour ago

		err := store.CreateSessionToken(ctx, token, userID, expiresAt)
		if err != nil {
			t.Fatalf("Failed to create token: %v", err)
		}

		st, err := store.GetSessionToken(ctx, token)
		if err != nil {
			t.Errorf("GetSessionToken failed: %v", err)
		}
		if st != nil {
			t.Error("Expected expired token to return nil, got token")
		}
	})

	// Test GetSessionToken - non-existent token
	t.Run("GetSessionToken_NotFound", func(t *testing.T) {
		st, err := store.GetSessionToken(ctx, "non-existent-token")
		if err != nil {
			t.Errorf("GetSessionToken failed: %v", err)
		}
		if st != nil {
			t.Error("Expected non-existent token to return nil, got token")
		}
	})

	// Test DeleteSessionToken
	t.Run("DeleteSessionToken", func(t *testing.T) {
		token := "test-token-delete"
		expiresAt := time.Now().Add(24 * time.Hour)

		err := store.CreateSessionToken(ctx, token, userID, expiresAt)
		if err != nil {
			t.Fatalf("Failed to create token: %v", err)
		}

		// Verify token exists
		st, err := store.GetSessionToken(ctx, token)
		if err != nil {
			t.Fatalf("GetSessionToken failed: %v", err)
		}
		if st == nil {
			t.Fatal("Expected token to exist before deletion")
		}

		// Delete the token
		err = store.DeleteSessionToken(ctx, token)
		if err != nil {
			t.Errorf("DeleteSessionToken failed: %v", err)
		}

		// Verify token is deleted
		st, err = store.GetSessionToken(ctx, token)
		if err != nil {
			t.Errorf("GetSessionToken failed: %v", err)
		}
		if st != nil {
			t.Error("Expected token to be deleted, but it still exists")
		}
	})

	// Test CleanupExpiredTokens
	t.Run("CleanupExpiredTokens", func(t *testing.T) {
		// Create some expired tokens
		for i := 0; i < 3; i++ {
			token := "expired-token-" + string(rune('a'+i))
			expiresAt := time.Now().Add(-1 * time.Hour)
			err := store.CreateSessionToken(ctx, token, userID, expiresAt)
			if err != nil {
				t.Fatalf("Failed to create expired token: %v", err)
			}
		}

		// Create a valid token
		validToken := "valid-token"
		expiresAt := time.Now().Add(24 * time.Hour)
		err := store.CreateSessionToken(ctx, validToken, userID, expiresAt)
		if err != nil {
			t.Fatalf("Failed to create valid token: %v", err)
		}

		// Cleanup expired tokens
		err = store.CleanupExpiredTokens(ctx)
		if err != nil {
			t.Errorf("CleanupExpiredTokens failed: %v", err)
		}

		// Verify valid token still exists
		st, err := store.GetSessionToken(ctx, validToken)
		if err != nil {
			t.Errorf("GetSessionToken failed: %v", err)
		}
		if st == nil {
			t.Error("Expected valid token to still exist after cleanup")
		}

		// Verify expired tokens are gone
		for i := 0; i < 3; i++ {
			token := "expired-token-" + string(rune('a'+i))
			st, err := store.GetSessionToken(ctx, token)
			if err != nil {
				t.Errorf("GetSessionToken failed: %v", err)
			}
			if st != nil {
				t.Errorf("Expected expired token %s to be cleaned up, but it still exists", token)
			}
		}
	})
}
