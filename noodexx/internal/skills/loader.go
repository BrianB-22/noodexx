package skills

import (
	"encoding/json"
	"fmt"
	"noodexx/internal/logging"
	"os"
	"path/filepath"
	"time"
)

// Skill represents a loaded skill with its metadata and configuration
type Skill struct {
	Name        string
	Version     string
	Description string
	Executable  string
	Triggers    []Trigger
	Settings    map[string]interface{}
	Timeout     time.Duration
	RequiresNet bool
	Path        string
}

// Trigger defines when a skill executes
type Trigger struct {
	Type       string                 // "manual", "timer", "keyword", "event"
	Parameters map[string]interface{} // Trigger-specific config
}

// Metadata is the skill.json structure
type Metadata struct {
	Name           string                 `json:"name"`
	Version        string                 `json:"version"`
	Description    string                 `json:"description"`
	Executable     string                 `json:"executable"`
	Triggers       []Trigger              `json:"triggers"`
	SettingsSchema map[string]interface{} `json:"settings_schema"`
	Timeout        int                    `json:"timeout"` // seconds
	RequiresNet    bool                   `json:"requires_network"`
}

// Loader discovers and loads skills
type Loader struct {
	skillsDir   string
	privacyMode bool
	logger      *logging.Logger
}

// NewLoader creates a skill loader
func NewLoader(skillsDir string, privacyMode bool, logger *logging.Logger) *Loader {
	return &Loader{
		skillsDir:   skillsDir,
		privacyMode: privacyMode,
		logger:      logger,
	}
}

// LoadAll discovers and loads all skills from the skills directory
func (l *Loader) LoadAll() ([]*Skill, error) {
	l.logger.WithContext("skills_dir", l.skillsDir).Debug("loading skills")
	var skills []*Skill

	// Check if skills directory exists
	if _, err := os.Stat(l.skillsDir); os.IsNotExist(err) {
		// Skills directory doesn't exist, return empty list (not an error)
		l.logger.Debug("skills directory does not exist")
		return skills, nil
	}

	entries, err := os.ReadDir(l.skillsDir)
	if err != nil {
		l.logger.WithContext("error", err.Error()).Error("failed to read skills directory")
		return nil, fmt.Errorf("failed to read skills directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillPath := filepath.Join(l.skillsDir, entry.Name())

		// Check if skill.json exists before trying to load
		skillJSONPath := filepath.Join(skillPath, "skill.json")
		if _, err := os.Stat(skillJSONPath); os.IsNotExist(err) {
			// No skill.json, skip this directory silently
			continue
		}

		skill, err := l.loadSkill(skillPath)
		if err != nil {
			// Log error but continue loading other skills
			l.logger.WithFields(map[string]interface{}{
				"skill_name": entry.Name(),
				"error":      err.Error(),
			}).Warn("failed to load skill")
			continue
		}

		// Skip network-requiring skills in privacy mode
		if l.privacyMode && skill.RequiresNet {
			l.logger.WithContext("skill_name", skill.Name).Debug("skipping skill (requires network)")
			continue
		}

		skills = append(skills, skill)
	}

	l.logger.WithContext("count", len(skills)).Debug("skills loaded")
	return skills, nil
}

// loadSkill loads a single skill from a directory
func (l *Loader) loadSkill(path string) (*Skill, error) {
	metadataPath := filepath.Join(path, "skill.json")

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read skill.json: %w", err)
	}

	var meta Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("failed to parse skill.json: %w", err)
	}

	// Validate required fields
	if meta.Name == "" {
		return nil, fmt.Errorf("skill.json missing required field: name")
	}
	if meta.Executable == "" {
		return nil, fmt.Errorf("skill.json missing required field: executable")
	}

	// Check executable exists and is within the skill directory
	execPath := filepath.Join(path, meta.Executable)
	info, err := os.Stat(execPath)
	if err != nil {
		return nil, fmt.Errorf("executable not found: %s", execPath)
	}

	// Check if executable has execute permissions (on Unix-like systems)
	if info.Mode()&0111 == 0 {
		return nil, fmt.Errorf("executable %s does not have execute permissions", execPath)
	}

	// Set default timeout if not specified
	timeout := time.Duration(meta.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Skill{
		Name:        meta.Name,
		Version:     meta.Version,
		Description: meta.Description,
		Executable:  execPath,
		Triggers:    meta.Triggers,
		Timeout:     timeout,
		RequiresNet: meta.RequiresNet,
		Path:        path,
	}, nil
}
