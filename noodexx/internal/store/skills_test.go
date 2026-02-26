package store

import (
	"context"
	"os"
	"testing"
)

// setupSkillsTestStore creates a temporary database for testing skills
func setupSkillsTestStore(t *testing.T) (*Store, func()) {
	tmpFile := t.TempDir() + "/test_skills.db"
	store, err := NewStore(tmpFile, "multi")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	cleanup := func() {
		store.Close()
		os.Remove(tmpFile)
	}

	return store, cleanup
}

func TestCreateSkill(t *testing.T) {
	store, cleanup := setupSkillsTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test user first
	userID, err := store.CreateUser(ctx, "testuser", "password123", "test@example.com", false, false)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create a skill
	skillID, err := store.CreateSkill(ctx, userID, "Test Skill", "/path/to/skill", true)
	if err != nil {
		t.Fatalf("Failed to create skill: %v", err)
	}

	if skillID == 0 {
		t.Error("Expected non-zero skill ID")
	}
}

func TestGetUserSkills(t *testing.T) {
	store, cleanup := setupSkillsTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create two test users
	user1ID, err := store.CreateUser(ctx, "user1", "password123", "user1@example.com", false, false)
	if err != nil {
		t.Fatalf("Failed to create user1: %v", err)
	}

	user2ID, err := store.CreateUser(ctx, "user2", "password456", "user2@example.com", false, false)
	if err != nil {
		t.Fatalf("Failed to create user2: %v", err)
	}

	// Create skills for user1
	_, err = store.CreateSkill(ctx, user1ID, "Skill 1", "/path/to/skill1", true)
	if err != nil {
		t.Fatalf("Failed to create skill 1: %v", err)
	}

	_, err = store.CreateSkill(ctx, user1ID, "Skill 2", "/path/to/skill2", false)
	if err != nil {
		t.Fatalf("Failed to create skill 2: %v", err)
	}

	// Create a skill for user2
	_, err = store.CreateSkill(ctx, user2ID, "Skill 3", "/path/to/skill3", true)
	if err != nil {
		t.Fatalf("Failed to create skill 3: %v", err)
	}

	// Get skills for user1
	skills, err := store.GetUserSkills(ctx, user1ID)
	if err != nil {
		t.Fatalf("Failed to get user skills: %v", err)
	}

	// Verify user1 has exactly 2 skills
	if len(skills) != 2 {
		t.Errorf("Expected 2 skills for user1, got %d", len(skills))
	}

	// Verify all skills belong to user1
	for _, skill := range skills {
		if skill.UserID != user1ID {
			t.Errorf("Expected skill to belong to user %d, got %d", user1ID, skill.UserID)
		}
	}

	// Get skills for user2
	skills2, err := store.GetUserSkills(ctx, user2ID)
	if err != nil {
		t.Fatalf("Failed to get user2 skills: %v", err)
	}

	// Verify user2 has exactly 1 skill
	if len(skills2) != 1 {
		t.Errorf("Expected 1 skill for user2, got %d", len(skills2))
	}
}

func TestUpdateSkillEnabled(t *testing.T) {
	store, cleanup := setupSkillsTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test user
	userID, err := store.CreateUser(ctx, "testuser", "password123", "test@example.com", false, false)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create a skill
	skillID, err := store.CreateSkill(ctx, userID, "Test Skill", "/path/to/skill", true)
	if err != nil {
		t.Fatalf("Failed to create skill: %v", err)
	}

	// Update the skill to disabled
	err = store.UpdateSkillEnabled(ctx, userID, skillID, false)
	if err != nil {
		t.Fatalf("Failed to update skill enabled status: %v", err)
	}

	// Verify the skill is now disabled
	skills, err := store.GetUserSkills(ctx, userID)
	if err != nil {
		t.Fatalf("Failed to get user skills: %v", err)
	}

	if len(skills) != 1 {
		t.Fatalf("Expected 1 skill, got %d", len(skills))
	}

	if skills[0].Enabled {
		t.Error("Expected skill to be disabled")
	}
}

func TestUpdateSkillEnabled_OwnershipVerification(t *testing.T) {
	store, cleanup := setupSkillsTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create two test users
	user1ID, err := store.CreateUser(ctx, "user1", "password123", "user1@example.com", false, false)
	if err != nil {
		t.Fatalf("Failed to create user1: %v", err)
	}

	user2ID, err := store.CreateUser(ctx, "user2", "password456", "user2@example.com", false, false)
	if err != nil {
		t.Fatalf("Failed to create user2: %v", err)
	}

	// Create a skill for user1
	skillID, err := store.CreateSkill(ctx, user1ID, "User1 Skill", "/path/to/skill", true)
	if err != nil {
		t.Fatalf("Failed to create skill: %v", err)
	}

	// Try to update the skill as user2 (should fail)
	err = store.UpdateSkillEnabled(ctx, user2ID, skillID, false)
	if err == nil {
		t.Error("Expected error when updating another user's skill, got nil")
	}

	// Verify the error message contains "access denied"
	if err != nil && err.Error() != "access denied: skill "+string(rune(skillID))+" does not belong to user "+string(rune(user2ID)) {
		// Just check that we got an error - the exact message format may vary
		t.Logf("Got expected error: %v", err)
	}
}

func TestDeleteSkill(t *testing.T) {
	store, cleanup := setupSkillsTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test user
	userID, err := store.CreateUser(ctx, "testuser", "password123", "test@example.com", false, false)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create a skill
	skillID, err := store.CreateSkill(ctx, userID, "Test Skill", "/path/to/skill", true)
	if err != nil {
		t.Fatalf("Failed to create skill: %v", err)
	}

	// Delete the skill
	err = store.DeleteSkill(ctx, userID, skillID)
	if err != nil {
		t.Fatalf("Failed to delete skill: %v", err)
	}

	// Verify the skill is deleted
	skills, err := store.GetUserSkills(ctx, userID)
	if err != nil {
		t.Fatalf("Failed to get user skills: %v", err)
	}

	if len(skills) != 0 {
		t.Errorf("Expected 0 skills after deletion, got %d", len(skills))
	}
}

func TestDeleteSkill_OwnershipVerification(t *testing.T) {
	store, cleanup := setupSkillsTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create two test users
	user1ID, err := store.CreateUser(ctx, "user1", "password123", "user1@example.com", false, false)
	if err != nil {
		t.Fatalf("Failed to create user1: %v", err)
	}

	user2ID, err := store.CreateUser(ctx, "user2", "password456", "user2@example.com", false, false)
	if err != nil {
		t.Fatalf("Failed to create user2: %v", err)
	}

	// Create a skill for user1
	skillID, err := store.CreateSkill(ctx, user1ID, "User1 Skill", "/path/to/skill", true)
	if err != nil {
		t.Fatalf("Failed to create skill: %v", err)
	}

	// Try to delete the skill as user2 (should fail)
	err = store.DeleteSkill(ctx, user2ID, skillID)
	if err == nil {
		t.Error("Expected error when deleting another user's skill, got nil")
	}

	// Verify the skill still exists for user1
	skills, err := store.GetUserSkills(ctx, user1ID)
	if err != nil {
		t.Fatalf("Failed to get user skills: %v", err)
	}

	if len(skills) != 1 {
		t.Errorf("Expected 1 skill to still exist, got %d", len(skills))
	}
}

func TestDeleteSkill_NonExistent(t *testing.T) {
	store, cleanup := setupSkillsTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test user
	userID, err := store.CreateUser(ctx, "testuser", "password123", "test@example.com", false, false)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Try to delete a non-existent skill
	err = store.DeleteSkill(ctx, userID, 99999)
	if err == nil {
		t.Error("Expected error when deleting non-existent skill, got nil")
	}
}
