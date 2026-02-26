package store

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestChatHistoryOperations tests all chat history methods
func TestChatHistoryOperations(t *testing.T) {
	// Create temporary database
	tmpFile, err := os.CreateTemp("", "test-chat-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Initialize store
	store, err := NewStore(tmpFile.Name(), "single")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	t.Run("SaveMessage persists messages", func(t *testing.T) {
		// Requirement 2.5: SaveMessage method persists chat messages with session ID, role, and content
		err := store.SaveMessage(ctx, "session-1", "user", "Hello, how are you?")
		if err != nil {
			t.Fatalf("Failed to save user message: %v", err)
		}

		err = store.SaveMessage(ctx, "session-1", "assistant", "I'm doing well, thank you!")
		if err != nil {
			t.Fatalf("Failed to save assistant message: %v", err)
		}

		// Verify messages were saved
		messages, err := store.GetSessionHistory(ctx, "session-1")
		if err != nil {
			t.Fatalf("Failed to get session history: %v", err)
		}

		if len(messages) != 2 {
			t.Errorf("Expected 2 messages, got %d", len(messages))
		}

		// Verify first message
		if messages[0].SessionID != "session-1" {
			t.Errorf("Expected session ID 'session-1', got '%s'", messages[0].SessionID)
		}
		if messages[0].Role != "user" {
			t.Errorf("Expected role 'user', got '%s'", messages[0].Role)
		}
		if messages[0].Content != "Hello, how are you?" {
			t.Errorf("Expected content 'Hello, how are you?', got '%s'", messages[0].Content)
		}

		// Verify second message
		if messages[1].Role != "assistant" {
			t.Errorf("Expected role 'assistant', got '%s'", messages[1].Role)
		}
		if messages[1].Content != "I'm doing well, thank you!" {
			t.Errorf("Expected content 'I'm doing well, thank you!', got '%s'", messages[1].Content)
		}
	})

	t.Run("GetSessionHistory returns messages ordered by creation time", func(t *testing.T) {
		// Requirement 2.6: GetSessionHistory retrieves all messages ordered by creation time
		sessionID := "session-2"

		// Add messages with slight delays to ensure different timestamps
		messages := []struct {
			role    string
			content string
		}{
			{"user", "First message"},
			{"assistant", "Second message"},
			{"user", "Third message"},
			{"assistant", "Fourth message"},
		}

		for _, msg := range messages {
			err := store.SaveMessage(ctx, sessionID, msg.role, msg.content)
			if err != nil {
				t.Fatalf("Failed to save message: %v", err)
			}
			time.Sleep(10 * time.Millisecond) // Small delay to ensure different timestamps
		}

		// Retrieve messages
		retrieved, err := store.GetSessionHistory(ctx, sessionID)
		if err != nil {
			t.Fatalf("Failed to get session history: %v", err)
		}

		if len(retrieved) != 4 {
			t.Fatalf("Expected 4 messages, got %d", len(retrieved))
		}

		// Verify messages are in chronological order
		for i := 0; i < len(retrieved); i++ {
			if retrieved[i].Content != messages[i].content {
				t.Errorf("Message %d: expected content '%s', got '%s'", i, messages[i].content, retrieved[i].Content)
			}
			if i > 0 {
				if retrieved[i].CreatedAt.Before(retrieved[i-1].CreatedAt) {
					t.Errorf("Message %d has earlier timestamp than message %d", i, i-1)
				}
			}
		}
	})

	t.Run("GetSessionHistory returns empty for non-existent session", func(t *testing.T) {
		messages, err := store.GetSessionHistory(ctx, "non-existent-session")
		if err != nil {
			t.Fatalf("Failed to get session history: %v", err)
		}

		if len(messages) != 0 {
			t.Errorf("Expected 0 messages for non-existent session, got %d", len(messages))
		}
	})

	t.Run("ListSessions returns all unique sessions with metadata", func(t *testing.T) {
		// Requirement 2.7: ListSessions returns all unique session IDs with most recent message timestamp

		// Create multiple sessions
		sessions := []string{"session-3", "session-4", "session-5"}
		for _, sessionID := range sessions {
			err := store.SaveMessage(ctx, sessionID, "user", "Test message in "+sessionID)
			if err != nil {
				t.Fatalf("Failed to save message: %v", err)
			}
			time.Sleep(10 * time.Millisecond)
		}

		// Add more messages to session-3 to test message count
		err := store.SaveMessage(ctx, "session-3", "assistant", "Response 1")
		if err != nil {
			t.Fatalf("Failed to save message: %v", err)
		}
		err = store.SaveMessage(ctx, "session-3", "user", "Follow-up")
		if err != nil {
			t.Fatalf("Failed to save message: %v", err)
		}

		// List all sessions
		sessionList, err := store.ListSessions(ctx)
		if err != nil {
			t.Fatalf("Failed to list sessions: %v", err)
		}

		// Should have at least the 3 new sessions plus any from previous tests
		if len(sessionList) < 3 {
			t.Errorf("Expected at least 3 sessions, got %d", len(sessionList))
		}

		// Find session-3 and verify message count
		var session3 *Session
		for i := range sessionList {
			if sessionList[i].ID == "session-3" {
				session3 = &sessionList[i]
				break
			}
		}

		if session3 == nil {
			t.Fatal("session-3 not found in session list")
		}

		if session3.MessageCount != 3 {
			t.Errorf("Expected session-3 to have 3 messages, got %d", session3.MessageCount)
		}

		// Verify sessions are ordered by most recent message (descending)
		for i := 1; i < len(sessionList); i++ {
			if sessionList[i].LastMessageAt.After(sessionList[i-1].LastMessageAt) {
				t.Errorf("Sessions not ordered by most recent message: session %d is newer than session %d", i, i-1)
			}
		}
	})

	t.Run("ListSessions groups by session_id correctly", func(t *testing.T) {
		// Requirement 2.7: ListSessions uses GROUP BY session_id
		sessionID := "session-6"

		// Add multiple messages to the same session
		for i := 0; i < 5; i++ {
			err := store.SaveMessage(ctx, sessionID, "user", "Message number "+string(rune('0'+i)))
			if err != nil {
				t.Fatalf("Failed to save message: %v", err)
			}
		}

		// List sessions
		sessionList, err := store.ListSessions(ctx)
		if err != nil {
			t.Fatalf("Failed to list sessions: %v", err)
		}

		// Count how many times session-6 appears (should be exactly once due to GROUP BY)
		count := 0
		var session6 *Session
		for i := range sessionList {
			if sessionList[i].ID == sessionID {
				count++
				session6 = &sessionList[i]
			}
		}

		if count != 1 {
			t.Errorf("Expected session-6 to appear exactly once in list, appeared %d times", count)
		}

		if session6 != nil && session6.MessageCount != 5 {
			t.Errorf("Expected session-6 to have 5 messages, got %d", session6.MessageCount)
		}
	})

	t.Run("Multiple sessions are isolated", func(t *testing.T) {
		// Verify that messages from different sessions don't interfere
		session7 := "session-7"
		session8 := "session-8"

		err := store.SaveMessage(ctx, session7, "user", "Message in session 7")
		if err != nil {
			t.Fatalf("Failed to save message: %v", err)
		}

		err = store.SaveMessage(ctx, session8, "user", "Message in session 8")
		if err != nil {
			t.Fatalf("Failed to save message: %v", err)
		}

		// Get history for session 7
		messages7, err := store.GetSessionHistory(ctx, session7)
		if err != nil {
			t.Fatalf("Failed to get session 7 history: %v", err)
		}

		// Get history for session 8
		messages8, err := store.GetSessionHistory(ctx, session8)
		if err != nil {
			t.Fatalf("Failed to get session 8 history: %v", err)
		}

		// Verify each session has only its own message
		if len(messages7) != 1 || messages7[0].Content != "Message in session 7" {
			t.Errorf("Session 7 has incorrect messages")
		}

		if len(messages8) != 1 || messages8[0].Content != "Message in session 8" {
			t.Errorf("Session 8 has incorrect messages")
		}
	})
}
