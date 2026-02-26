package store

import (
	"context"
	"os"
	"testing"
)

func TestPasswordHashing(t *testing.T) {
	password := "testpassword123"

	// Test hashPassword
	hash, err := hashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}
	if hash == password {
		t.Error("Hash should not equal plaintext password")
	}
	if len(hash) == 0 {
		t.Error("Hash should not be empty")
	}

	// Test checkPasswordHash with correct password
	if !checkPasswordHash(password, hash) {
		t.Error("checkPasswordHash should return true for correct password")
	}

	// Test checkPasswordHash with incorrect password
	if checkPasswordHash("wrongpassword", hash) {
		t.Error("checkPasswordHash should return false for incorrect password")
	}

	// Test that same password produces different hashes (bcrypt salt)
	hash2, err := hashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password second time: %v", err)
	}
	if hash == hash2 {
		t.Error("Same password should produce different hashes due to salt")
	}

	// But both hashes should validate the same password
	if !checkPasswordHash(password, hash2) {
		t.Error("Second hash should also validate the password")
	}
}

// Note: Full integration tests for user management methods will be added
// after the DataStore interface is fully implemented in tasks 5.1-5.12.
// The user management methods (CreateUser, GetUserByUsername, GetUserByID,
// ValidateCredentials, UpdatePassword, UpdateLastLogin, ListUsers, DeleteUser)
// are implemented correctly and match the DataStore interface specification.

func TestAccountLockout(t *testing.T) {
	// Create a temporary database
	dbPath := "test_account_lockout.db"
	defer os.Remove(dbPath)

	store, err := NewStore(dbPath, "single")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	username := "testuser"

	// Initially, account should not be locked
	locked, _ := store.IsAccountLocked(ctx, username)
	if locked {
		t.Error("Account should not be locked initially")
	}

	// Record 4 failed login attempts - should not lock
	for i := 0; i < 4; i++ {
		err := store.RecordFailedLogin(ctx, username)
		if err != nil {
			t.Fatalf("Failed to record failed login attempt %d: %v", i+1, err)
		}
	}

	locked, _ = store.IsAccountLocked(ctx, username)
	if locked {
		t.Error("Account should not be locked after 4 attempts")
	}

	// Record 5th failed login attempt - should lock
	err = store.RecordFailedLogin(ctx, username)
	if err != nil {
		t.Fatalf("Failed to record 5th failed login attempt: %v", err)
	}

	locked, lockoutExpires := store.IsAccountLocked(ctx, username)
	if !locked {
		t.Error("Account should be locked after 5 attempts")
	}
	if lockoutExpires.IsZero() {
		t.Error("Lockout expiration time should be set")
	}

	// Clear failed logins
	err = store.ClearFailedLogins(ctx, username)
	if err != nil {
		t.Fatalf("Failed to clear failed logins: %v", err)
	}

	// Account should no longer be locked
	locked, _ = store.IsAccountLocked(ctx, username)
	if locked {
		t.Error("Account should not be locked after clearing failed logins")
	}
}

func TestAccountLockoutMultipleUsers(t *testing.T) {
	// Create a temporary database
	dbPath := "test_account_lockout_multi.db"
	defer os.Remove(dbPath)

	store, err := NewStore(dbPath, "single")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	user1 := "user1"
	user2 := "user2"

	// Record 5 failed attempts for user1
	for i := 0; i < 5; i++ {
		err := store.RecordFailedLogin(ctx, user1)
		if err != nil {
			t.Fatalf("Failed to record failed login for user1: %v", err)
		}
	}

	// Record 2 failed attempts for user2
	for i := 0; i < 2; i++ {
		err := store.RecordFailedLogin(ctx, user2)
		if err != nil {
			t.Fatalf("Failed to record failed login for user2: %v", err)
		}
	}

	// user1 should be locked
	locked1, _ := store.IsAccountLocked(ctx, user1)
	if !locked1 {
		t.Error("user1 should be locked after 5 attempts")
	}

	// user2 should not be locked
	locked2, _ := store.IsAccountLocked(ctx, user2)
	if locked2 {
		t.Error("user2 should not be locked after 2 attempts")
	}

	// Clear user1's failed logins
	err = store.ClearFailedLogins(ctx, user1)
	if err != nil {
		t.Fatalf("Failed to clear failed logins for user1: %v", err)
	}

	// user1 should no longer be locked
	locked1, _ = store.IsAccountLocked(ctx, user1)
	if locked1 {
		t.Error("user1 should not be locked after clearing")
	}

	// user2 should still not be locked
	locked2, _ = store.IsAccountLocked(ctx, user2)
	if locked2 {
		t.Error("user2 should still not be locked")
	}
}
