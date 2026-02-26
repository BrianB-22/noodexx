package skills

import (
	"io"
	"noodexx/internal/logging"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Helper function to create a test logger
func newTestLogger() *logging.Logger {
	return logging.NewLogger("test", logging.DEBUG, io.Discard)
}

func TestNewLoader(t *testing.T) {
	loader := NewLoader("/test/path", true, newTestLogger())
	if loader == nil {
		t.Fatal("NewLoader returned nil")
	}
	if loader.skillsDir != "/test/path" {
		t.Errorf("Expected skillsDir to be /test/path, got %s", loader.skillsDir)
	}
	if !loader.privacyMode {
		t.Error("Expected privacyMode to be true")
	}
}

func TestLoadAll_NonExistentDirectory(t *testing.T) {
	loader := NewLoader("/nonexistent/path", false, newTestLogger())
	skills, err := loader.LoadAll()
	if err != nil {
		t.Errorf("LoadAll should not error on nonexistent directory, got: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("Expected 0 skills, got %d", len(skills))
	}
}

func TestLoadAll_EmptyDirectory(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	loader := NewLoader(tmpDir, false, newTestLogger())
	skills, err := loader.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("Expected 0 skills in empty directory, got %d", len(skills))
	}
}

func TestLoadAll_ValidSkill(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "test-skill")
	if err := os.Mkdir(skillDir, 0755); err != nil {
		t.Fatalf("Failed to create skill directory: %v", err)
	}

	// Create skill.json
	skillJSON := `{
		"name": "test-skill",
		"version": "1.0.0",
		"description": "A test skill",
		"executable": "test.sh",
		"timeout": 10,
		"requires_network": false
	}`
	if err := os.WriteFile(filepath.Join(skillDir, "skill.json"), []byte(skillJSON), 0644); err != nil {
		t.Fatalf("Failed to write skill.json: %v", err)
	}

	// Create executable
	execPath := filepath.Join(skillDir, "test.sh")
	if err := os.WriteFile(execPath, []byte("#!/bin/bash\necho test"), 0755); err != nil {
		t.Fatalf("Failed to write executable: %v", err)
	}

	loader := NewLoader(tmpDir, false, newTestLogger())
	skills, err := loader.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("Expected 1 skill, got %d", len(skills))
	}

	skill := skills[0]
	if skill.Name != "test-skill" {
		t.Errorf("Expected name 'test-skill', got '%s'", skill.Name)
	}
	if skill.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", skill.Version)
	}
	if skill.Description != "A test skill" {
		t.Errorf("Expected description 'A test skill', got '%s'", skill.Description)
	}
	if skill.Timeout != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %v", skill.Timeout)
	}
	if skill.RequiresNet {
		t.Error("Expected RequiresNet to be false")
	}
}

func TestLoadAll_PrivacyModeFiltersNetworkSkills(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()

	// Create network-requiring skill
	skillDir := filepath.Join(tmpDir, "network-skill")
	if err := os.Mkdir(skillDir, 0755); err != nil {
		t.Fatalf("Failed to create skill directory: %v", err)
	}

	skillJSON := `{
		"name": "network-skill",
		"version": "1.0.0",
		"description": "A network skill",
		"executable": "test.sh",
		"requires_network": true
	}`
	if err := os.WriteFile(filepath.Join(skillDir, "skill.json"), []byte(skillJSON), 0644); err != nil {
		t.Fatalf("Failed to write skill.json: %v", err)
	}

	execPath := filepath.Join(skillDir, "test.sh")
	if err := os.WriteFile(execPath, []byte("#!/bin/bash\necho test"), 0755); err != nil {
		t.Fatalf("Failed to write executable: %v", err)
	}

	// Test with privacy mode enabled
	loader := NewLoader(tmpDir, true, newTestLogger())
	skills, err := loader.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("Expected 0 skills with privacy mode enabled, got %d", len(skills))
	}

	// Test with privacy mode disabled
	loader = NewLoader(tmpDir, false, newTestLogger())
	skills, err = loader.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	if len(skills) != 1 {
		t.Errorf("Expected 1 skill with privacy mode disabled, got %d", len(skills))
	}
}

func TestLoadSkill_MissingSkillJSON(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "bad-skill")
	if err := os.Mkdir(skillDir, 0755); err != nil {
		t.Fatalf("Failed to create skill directory: %v", err)
	}

	loader := NewLoader(tmpDir, false, newTestLogger())
	_, err := loader.loadSkill(skillDir)
	if err == nil {
		t.Error("Expected error for missing skill.json")
	}
}

func TestLoadSkill_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "bad-skill")
	if err := os.Mkdir(skillDir, 0755); err != nil {
		t.Fatalf("Failed to create skill directory: %v", err)
	}

	// Write invalid JSON
	if err := os.WriteFile(filepath.Join(skillDir, "skill.json"), []byte("{invalid json"), 0644); err != nil {
		t.Fatalf("Failed to write skill.json: %v", err)
	}

	loader := NewLoader(tmpDir, false, newTestLogger())
	_, err := loader.loadSkill(skillDir)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestLoadSkill_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{
			name: "missing name",
			json: `{"executable": "test.sh"}`,
		},
		{
			name: "missing executable",
			json: `{"name": "test"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			skillDir := filepath.Join(tmpDir, "bad-skill")
			if err := os.Mkdir(skillDir, 0755); err != nil {
				t.Fatalf("Failed to create skill directory: %v", err)
			}

			if err := os.WriteFile(filepath.Join(skillDir, "skill.json"), []byte(tt.json), 0644); err != nil {
				t.Fatalf("Failed to write skill.json: %v", err)
			}

			loader := NewLoader(tmpDir, false, newTestLogger())
			_, err := loader.loadSkill(skillDir)
			if err == nil {
				t.Errorf("Expected error for %s", tt.name)
			}
		})
	}
}

func TestLoadSkill_MissingExecutable(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "bad-skill")
	if err := os.Mkdir(skillDir, 0755); err != nil {
		t.Fatalf("Failed to create skill directory: %v", err)
	}

	skillJSON := `{
		"name": "test-skill",
		"executable": "nonexistent.sh"
	}`
	if err := os.WriteFile(filepath.Join(skillDir, "skill.json"), []byte(skillJSON), 0644); err != nil {
		t.Fatalf("Failed to write skill.json: %v", err)
	}

	loader := NewLoader(tmpDir, false, newTestLogger())
	_, err := loader.loadSkill(skillDir)
	if err == nil {
		t.Error("Expected error for missing executable")
	}
}

func TestLoadSkill_DefaultTimeout(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "test-skill")
	if err := os.Mkdir(skillDir, 0755); err != nil {
		t.Fatalf("Failed to create skill directory: %v", err)
	}

	// Create skill.json without timeout
	skillJSON := `{
		"name": "test-skill",
		"executable": "test.sh"
	}`
	if err := os.WriteFile(filepath.Join(skillDir, "skill.json"), []byte(skillJSON), 0644); err != nil {
		t.Fatalf("Failed to write skill.json: %v", err)
	}

	execPath := filepath.Join(skillDir, "test.sh")
	if err := os.WriteFile(execPath, []byte("#!/bin/bash\necho test"), 0755); err != nil {
		t.Fatalf("Failed to write executable: %v", err)
	}

	loader := NewLoader(tmpDir, false, newTestLogger())
	skill, err := loader.loadSkill(skillDir)
	if err != nil {
		t.Fatalf("loadSkill failed: %v", err)
	}

	if skill.Timeout != 30*time.Second {
		t.Errorf("Expected default timeout 30s, got %v", skill.Timeout)
	}
}

func TestLoadSkill_WithTriggers(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "test-skill")
	if err := os.Mkdir(skillDir, 0755); err != nil {
		t.Fatalf("Failed to create skill directory: %v", err)
	}

	skillJSON := `{
		"name": "test-skill",
		"executable": "test.sh",
		"triggers": [
			{
				"type": "manual"
			},
			{
				"type": "keyword",
				"parameters": {
					"keywords": ["test", "demo"]
				}
			}
		]
	}`
	if err := os.WriteFile(filepath.Join(skillDir, "skill.json"), []byte(skillJSON), 0644); err != nil {
		t.Fatalf("Failed to write skill.json: %v", err)
	}

	execPath := filepath.Join(skillDir, "test.sh")
	if err := os.WriteFile(execPath, []byte("#!/bin/bash\necho test"), 0755); err != nil {
		t.Fatalf("Failed to write executable: %v", err)
	}

	loader := NewLoader(tmpDir, false, newTestLogger())
	skill, err := loader.loadSkill(skillDir)
	if err != nil {
		t.Fatalf("loadSkill failed: %v", err)
	}

	if len(skill.Triggers) != 2 {
		t.Errorf("Expected 2 triggers, got %d", len(skill.Triggers))
	}
	if skill.Triggers[0].Type != "manual" {
		t.Errorf("Expected first trigger type 'manual', got '%s'", skill.Triggers[0].Type)
	}
	if skill.Triggers[1].Type != "keyword" {
		t.Errorf("Expected second trigger type 'keyword', got '%s'", skill.Triggers[1].Type)
	}
}
