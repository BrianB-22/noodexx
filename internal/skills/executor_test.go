package skills

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExecutor_Execute(t *testing.T) {
	// Create a temporary directory for test skill
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "test-skill")
	if err := os.Mkdir(skillDir, 0755); err != nil {
		t.Fatalf("Failed to create skill directory: %v", err)
	}

	// Create a simple test script that echoes input back
	scriptPath := filepath.Join(skillDir, "test.sh")
	scriptContent := `#!/bin/bash
# Read JSON from stdin and output valid JSON
cat <<EOF
{"result": "Success", "metadata": {"test": true}}
EOF
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	// Create test skill
	skill := &Skill{
		Name:       "test-skill",
		Version:    "1.0.0",
		Executable: scriptPath,
		Path:       skillDir,
		Timeout:    5 * time.Second,
		Settings:   map[string]interface{}{"key": "value"},
	}

	// Create executor
	executor := NewExecutor(false)

	// Execute skill
	input := Input{
		Query:    "test query",
		Context:  map[string]interface{}{"foo": "bar"},
		Settings: map[string]interface{}{"setting1": "value1"},
	}

	ctx := context.Background()
	output, err := executor.Execute(ctx, skill, input)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify output
	if output.Result == "" {
		t.Error("Expected non-empty result")
	}
	if output.Error != "" {
		t.Errorf("Expected no error, got: %s", output.Error)
	}
}

func TestExecutor_Execute_Timeout(t *testing.T) {
	// Skip on systems where bash might not be available or behave differently
	if testing.Short() {
		t.Skip("Skipping timeout test in short mode")
	}

	// Create a temporary directory for test skill
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "timeout-skill")
	if err := os.Mkdir(skillDir, 0755); err != nil {
		t.Fatalf("Failed to create skill directory: %v", err)
	}

	// Create a script that sleeps longer than timeout
	scriptPath := filepath.Join(skillDir, "timeout.sh")
	scriptContent := `#!/bin/bash
sleep 5
echo '{"result": "Should not reach here"}'
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	// Create test skill with short timeout
	skill := &Skill{
		Name:       "timeout-skill",
		Version:    "1.0.0",
		Executable: scriptPath,
		Path:       skillDir,
		Timeout:    500 * time.Millisecond, // Very short timeout
		Settings:   map[string]interface{}{},
	}

	// Create executor
	executor := NewExecutor(false)

	// Execute skill
	input := Input{
		Query:    "test query",
		Context:  map[string]interface{}{},
		Settings: map[string]interface{}{},
	}

	ctx := context.Background()
	_, err := executor.Execute(ctx, skill, input)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
	if err != nil && !contains(err.Error(), "timed out") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

func TestExecutor_buildEnv(t *testing.T) {
	skill := &Skill{
		Name:     "test-skill",
		Version:  "1.0.0",
		Settings: map[string]interface{}{"api_key": "secret", "timeout": 30},
	}

	tests := []struct {
		name        string
		privacyMode bool
		wantEnvVars []string
	}{
		{
			name:        "privacy mode enabled",
			privacyMode: true,
			wantEnvVars: []string{
				"NOODEXX_SKILL_NAME=test-skill",
				"NOODEXX_SKILL_VERSION=1.0.0",
				"NOODEXX_PRIVACY_MODE=true",
				"NOODEXX_SETTING_API_KEY=secret",
				"NOODEXX_SETTING_TIMEOUT=30",
			},
		},
		{
			name:        "privacy mode disabled",
			privacyMode: false,
			wantEnvVars: []string{
				"NOODEXX_SKILL_NAME=test-skill",
				"NOODEXX_SKILL_VERSION=1.0.0",
				"NOODEXX_SETTING_API_KEY=secret",
				"NOODEXX_SETTING_TIMEOUT=30",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewExecutor(tt.privacyMode)
			env := executor.buildEnv(skill)

			// Check that expected env vars are present
			for _, want := range tt.wantEnvVars {
				found := false
				for _, e := range env {
					if e == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected env var %q not found in: %v", want, env)
				}
			}

			// Check privacy mode env var presence
			hasPrivacyMode := false
			for _, e := range env {
				if e == "NOODEXX_PRIVACY_MODE=true" {
					hasPrivacyMode = true
					break
				}
			}
			if tt.privacyMode && !hasPrivacyMode {
				t.Error("Expected NOODEXX_PRIVACY_MODE=true in privacy mode")
			}
			if !tt.privacyMode && hasPrivacyMode {
				t.Error("Did not expect NOODEXX_PRIVACY_MODE=true when privacy mode disabled")
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
