package store

import (
	"context"
	"os"
	"testing"
)

// setupTestStore creates a temporary test database and returns the store and cleanup function
func setupTestStore(t *testing.T) (*Store, func()) {
	dbPath := "test_session_management.db"
	store, err := NewStore(dbPath, "single")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	cleanup := func() {
		store.Close()
		os.Remove(dbPath)
	}

	return store, cleanup
}

// TestSaveChatMessage tests the SaveChatMessage method
func TestSaveChatMessage(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test user
	userID, err := store.CreateUser(ctx, "testuser", "password123", "test@example.com", false, false)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Save a chat message
	sessionID := "test-session-1"
	err = store.SaveChatMessage(ctx, userID, sessionID, "user", "Hello, world!")
	if err != nil {
		t.Fatalf("Failed to save chat message: %v", err)
	}

	// Verify the message was saved
	messages, err := store.GetSessionMessages(ctx, userID, sessionID)
	if err != nil {
		t.Fatalf("Failed to get session messages: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	if messages[0].Content != "Hello, world!" {
		t.Errorf("Expected content 'Hello, world!', got '%s'", messages[0].Content)
	}

	if messages[0].Role != "user" {
		t.Errorf("Expected role 'user', got '%s'", messages[0].Role)
	}
}

// TestGetUserSessions tests the GetUserSessions method
func TestGetUserSessions(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create two test users
	user1ID, err := store.CreateUser(ctx, "user1", "password123", "user1@example.com", false, false)
	if err != nil {
		t.Fatalf("Failed to create user1: %v", err)
	}

	user2ID, err := store.CreateUser(ctx, "user2", "password123", "user2@example.com", false, false)
	if err != nil {
		t.Fatalf("Failed to create user2: %v", err)
	}

	// Create sessions for user1
	err = store.SaveChatMessage(ctx, user1ID, "session-1", "user", "Message 1")
	if err != nil {
		t.Fatalf("Failed to save message for user1 session-1: %v", err)
	}

	err = store.SaveChatMessage(ctx, user1ID, "session-2", "user", "Message 2")
	if err != nil {
		t.Fatalf("Failed to save message for user1 session-2: %v", err)
	}

	// Create session for user2
	err = store.SaveChatMessage(ctx, user2ID, "session-3", "user", "Message 3")
	if err != nil {
		t.Fatalf("Failed to save message for user2 session-3: %v", err)
	}

	// Get sessions for user1
	user1Sessions, err := store.GetUserSessions(ctx, user1ID)
	if err != nil {
		t.Fatalf("Failed to get user1 sessions: %v", err)
	}

	if len(user1Sessions) != 2 {
		t.Fatalf("Expected 2 sessions for user1, got %d", len(user1Sessions))
	}

	// Get sessions for user2
	user2Sessions, err := store.GetUserSessions(ctx, user2ID)
	if err != nil {
		t.Fatalf("Failed to get user2 sessions: %v", err)
	}

	if len(user2Sessions) != 1 {
		t.Fatalf("Expected 1 session for user2, got %d", len(user2Sessions))
	}
}

// TestGetSessionOwner tests the GetSessionOwner method
func TestGetSessionOwner(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test user
	userID, err := store.CreateUser(ctx, "testuser", "password123", "test@example.com", false, false)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create a session
	sessionID := "test-session-1"
	err = store.SaveChatMessage(ctx, userID, sessionID, "user", "Hello!")
	if err != nil {
		t.Fatalf("Failed to save chat message: %v", err)
	}

	// Get the session owner
	ownerID, err := store.GetSessionOwner(ctx, sessionID)
	if err != nil {
		t.Fatalf("Failed to get session owner: %v", err)
	}

	if ownerID != userID {
		t.Errorf("Expected owner ID %d, got %d", userID, ownerID)
	}

	// Test non-existent session
	_, err = store.GetSessionOwner(ctx, "non-existent-session")
	if err == nil {
		t.Error("Expected error for non-existent session, got nil")
	}
}

// TestGetSessionMessagesOwnershipVerification tests ownership verification in GetSessionMessages
func TestGetSessionMessagesOwnershipVerification(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create two test users
	user1ID, err := store.CreateUser(ctx, "user1", "password123", "user1@example.com", false, false)
	if err != nil {
		t.Fatalf("Failed to create user1: %v", err)
	}

	user2ID, err := store.CreateUser(ctx, "user2", "password123", "user2@example.com", false, false)
	if err != nil {
		t.Fatalf("Failed to create user2: %v", err)
	}

	// Create a session for user1
	sessionID := "user1-session"
	err = store.SaveChatMessage(ctx, user1ID, sessionID, "user", "User1's message")
	if err != nil {
		t.Fatalf("Failed to save message for user1: %v", err)
	}

	// User1 should be able to access their own session
	messages, err := store.GetSessionMessages(ctx, user1ID, sessionID)
	if err != nil {
		t.Fatalf("User1 should be able to access their session: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	// User2 should NOT be able to access user1's session
	_, err = store.GetSessionMessages(ctx, user2ID, sessionID)
	if err == nil {
		t.Error("User2 should not be able to access user1's session")
	}
}

// TestSaveChatMessageUpdatesSessionMetadata tests that SaveChatMessage updates session metadata
func TestSaveChatMessageUpdatesSessionMetadata(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test user
	userID, err := store.CreateUser(ctx, "testuser", "password123", "test@example.com", false, false)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Save first message
	sessionID := "test-session-1"
	err = store.SaveChatMessage(ctx, userID, sessionID, "user", "First message")
	if err != nil {
		t.Fatalf("Failed to save first message: %v", err)
	}

	// Get sessions to verify metadata was created
	sessions, err := store.GetUserSessions(ctx, userID)
	if err != nil {
		t.Fatalf("Failed to get user sessions: %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("Expected 1 session, got %d", len(sessions))
	}

	if sessions[0].ID != sessionID {
		t.Errorf("Expected session ID '%s', got '%s'", sessionID, sessions[0].ID)
	}

	// Save second message to same session
	err = store.SaveChatMessage(ctx, userID, sessionID, "assistant", "Second message")
	if err != nil {
		t.Fatalf("Failed to save second message: %v", err)
	}

	// Verify session metadata was updated (last_message_at should be updated)
	sessions, err = store.GetUserSessions(ctx, userID)
	if err != nil {
		t.Fatalf("Failed to get user sessions after second message: %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("Expected 1 session after second message, got %d", len(sessions))
	}

	// Verify we have 2 messages
	messages, err := store.GetSessionMessages(ctx, userID, sessionID)
	if err != nil {
		t.Fatalf("Failed to get session messages: %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(messages))
	}
}
