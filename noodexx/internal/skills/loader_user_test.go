package skills

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// mockStore implements the Store interface for testing
type mockStore struct {
	skills []SkillMetadata
}

func (m *mockStore) GetUserSkills(ctx context.Context, userID int64) ([]SkillMetadata, error) {
	var userSkills []SkillMetadata
	for _, skill := range m.skills {
		if skill.UserID == userID {
			userSkills = append(userSkills, skill)
		}
	}
	return userSkills, nil
}

func TestLoadForUser_WithStore(t *testing.T) {
	// Create temporary directory with test skills
	tmpDir := t.TempDir()

	// Create skill1 directory
	skill1Dir := filepath.Join(tmpDir, "skill1")
	if err := os.Mkdir(skill1Dir, 0755); err != nil {
		t.Fatalf("Failed to create skill1 directory: %v", err)
	}

	// Create skill.json for skill1
	skill1JSON := `{
		"name": "skill1",
		"version": "1.0.0",
		"description": "Test skill 1",
		"executable": "run.sh",
		"triggers": [{"type": "manual"}]
	}`
	if err := os.WriteFile(filepath.Join(skill1Dir, "skill.json"), []byte(skill1JSON), 0644); err != nil {
		t.Fatalf("Failed to write skill.json: %v", err)
	}

	// Create executable
	execPath := filepath.Join(skill1Dir, "run.sh")
	if err := os.WriteFile(execPath, []byte("#!/bin/bash\necho test"), 0755); err != nil {
		t.Fatalf("Failed to write executable: %v", err)
	}

	// Create skill2 directory
	skill2Dir := filepath.Join(tmpDir, "skill2")
	if err := os.Mkdir(skill2Dir, 0755); err != nil {
		t.Fatalf("Failed to create skill2 directory: %v", err)
	}

	// Create skill.json for skill2
	skill2JSON := `{
		"name": "skill2",
		"version": "1.0.0",
		"description": "Test skill 2",
		"executable": "run.sh",
		"triggers": [{"type": "manual"}]
	}`
	if err := os.WriteFile(filepath.Join(skill2Dir, "skill.json"), []byte(skill2JSON), 0644); err != nil {
		t.Fatalf("Failed to write skill.json: %v", err)
	}

	// Create executable
	execPath2 := filepath.Join(skill2Dir, "run.sh")
	if err := os.WriteFile(execPath2, []byte("#!/bin/bash\necho test"), 0755); err != nil {
		t.Fatalf("Failed to write executable: %v", err)
	}

	// Create mock store with skills for different users
	store := &mockStore{
		skills: []SkillMetadata{
			{
				ID:        1,
				UserID:    1,
				Name:      "skill1",
				Path:      "skill1",
				Enabled:   true, // Enable the skill
				CreatedAt: time.Now(),
			},
			{
				ID:        2,
				UserID:    2,
				Name:      "skill2",
				Path:      "skill2",
				Enabled:   true, // Enable the skill
				CreatedAt: time.Now(),
			},
		},
	}

	// Create loader with store
	loader := NewLoaderWithStore(tmpDir, false, newTestLogger(), store)

	// Load skills for user 1
	ctx := context.Background()
	skills, err := loader.LoadForUser(ctx, 1)
	if err != nil {
		t.Fatalf("LoadForUser failed: %v", err)
	}

	// Should only get skill1
	if len(skills) != 1 {
		t.Errorf("Expected 1 skill for user 1, got %d", len(skills))
	}

	if len(skills) > 0 && skills[0].Name != "skill1" {
		t.Errorf("Expected skill1, got %s", skills[0].Name)
	}

	// Load skills for user 2
	skills2, err := loader.LoadForUser(ctx, 2)
	if err != nil {
		t.Fatalf("LoadForUser failed: %v", err)
	}

	// Should only get skill2
	if len(skills2) != 1 {
		t.Errorf("Expected 1 skill for user 2, got %d", len(skills2))
	}

	if len(skills2) > 0 && skills2[0].Name != "skill2" {
		t.Errorf("Expected skill2, got %s", skills2[0].Name)
	}
}

func TestLoadForUser_DisabledSkills(t *testing.T) {
	// Create temporary directory with test skill
	tmpDir := t.TempDir()

	// Create skill directory
	skillDir := filepath.Join(tmpDir, "skill1")
	if err := os.Mkdir(skillDir, 0755); err != nil {
		t.Fatalf("Failed to create skill directory: %v", err)
	}

	// Create skill.json
	skillJSON := `{
		"name": "skill1",
		"version": "1.0.0",
		"description": "Test skill",
		"executable": "run.sh",
		"triggers": [{"type": "manual"}]
	}`
	if err := os.WriteFile(filepath.Join(skillDir, "skill.json"), []byte(skillJSON), 0644); err != nil {
		t.Fatalf("Failed to write skill.json: %v", err)
	}

	// Create executable
	execPath := filepath.Join(skillDir, "run.sh")
	if err := os.WriteFile(execPath, []byte("#!/bin/bash\necho test"), 0755); err != nil {
		t.Fatalf("Failed to write executable: %v", err)
	}

	// Create mock store with disabled skill
	store := &mockStore{
		skills: []SkillMetadata{
			{
				ID:        1,
				UserID:    1,
				Name:      "skill1",
				Path:      "skill1",
				Enabled:   false, // Disabled
				CreatedAt: time.Now(),
			},
		},
	}

	// Create loader with store
	loader := NewLoaderWithStore(tmpDir, false, newTestLogger(), store)

	// Load skills for user 1
	ctx := context.Background()
	skills, err := loader.LoadForUser(ctx, 1)
	if err != nil {
		t.Fatalf("LoadForUser failed: %v", err)
	}

	// Should get no skills (disabled)
	if len(skills) != 0 {
		t.Errorf("Expected 0 skills (disabled), got %d", len(skills))
	}
}

func TestLoadForUser_FallbackToLoadAll(t *testing.T) {
	// Create temporary directory with test skill
	tmpDir := t.TempDir()

	// Create skill directory
	skillDir := filepath.Join(tmpDir, "skill1")
	if err := os.Mkdir(skillDir, 0755); err != nil {
		t.Fatalf("Failed to create skill directory: %v", err)
	}

	// Create skill.json
	skillJSON := `{
		"name": "skill1",
		"version": "1.0.0",
		"description": "Test skill",
		"executable": "run.sh",
		"triggers": [{"type": "manual"}]
	}`
	if err := os.WriteFile(filepath.Join(skillDir, "skill.json"), []byte(skillJSON), 0644); err != nil {
		t.Fatalf("Failed to write skill.json: %v", err)
	}

	// Create executable
	execPath := filepath.Join(skillDir, "run.sh")
	if err := os.WriteFile(execPath, []byte("#!/bin/bash\necho test"), 0755); err != nil {
		t.Fatalf("Failed to write executable: %v", err)
	}

	// Create loader WITHOUT store (backward compatibility)
	loader := NewLoader(tmpDir, false, newTestLogger())

	// Load skills for user 1 - should fallback to LoadAll
	ctx := context.Background()
	skills, err := loader.LoadForUser(ctx, 1)
	if err != nil {
		t.Fatalf("LoadForUser failed: %v", err)
	}

	// Should get all skills (fallback to LoadAll)
	if len(skills) != 1 {
		t.Errorf("Expected 1 skill (fallback), got %d", len(skills))
	}
}
