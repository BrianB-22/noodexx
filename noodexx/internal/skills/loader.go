package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"noodexx/internal/logging"
	"os"
	"path/filepath"
	"time"
)

// Skill represents a loaded skill with its metadata and configuration
type Skill struct {
	UserID      int64 // Owner of the skill (set when loaded via LoadForUser)
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

// Store interface for accessing user skills from database
type Store interface {
	GetUserSkills(ctx context.Context, userID int64) ([]SkillMetadata, error)
}

// SkillMetadata represents skill metadata from the database
type SkillMetadata struct {
	ID        int64
	UserID    int64
	Name      string
	Path      string
	Enabled   bool
	CreatedAt time.Time
}

// Loader discovers and loads skills
type Loader struct {
	skillsDir   string
	privacyMode bool
	logger      *logging.Logger
	store       Store
}

// NewLoader creates a skill loader
func NewLoader(skillsDir string, privacyMode bool, logger *logging.Logger) *Loader {
	return &Loader{
		skillsDir:   skillsDir,
		privacyMode: privacyMode,
		logger:      logger,
		store:       nil, // For backward compatibility
	}
}

// NewLoaderWithStore creates a skill loader with database store for user-scoped loading
func NewLoaderWithStore(skillsDir string, privacyMode bool, logger *logging.Logger, store Store) *Loader {
	return &Loader{
		skillsDir:   skillsDir,
		privacyMode: privacyMode,
		logger:      logger,
		store:       store,
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

// LoadForUser loads skills for a specific user by querying the database
// and loading only the enabled skills from the filesystem
func (l *Loader) LoadForUser(ctx context.Context, userID int64) ([]*Skill, error) {
	if l.store == nil {
		// Fallback to LoadAll for backward compatibility
		l.logger.Debug("store not configured, falling back to LoadAll")
		return l.LoadAll()
	}

	l.logger.WithFields(map[string]interface{}{
		"user_id":    userID,
		"skills_dir": l.skillsDir,
	}).Debug("loading skills for user")

	// Get user's skills from database
	userSkills, err := l.store.GetUserSkills(ctx, userID)
	if err != nil {
		l.logger.WithContext("error", err.Error()).Error("failed to get user skills from database")
		return nil, fmt.Errorf("failed to get user skills: %w", err)
	}

	var skills []*Skill
	for _, skillMeta := range userSkills {
		// Skip disabled skills
		if !skillMeta.Enabled {
			l.logger.WithFields(map[string]interface{}{
				"skill_name": skillMeta.Name,
				"user_id":    userID,
			}).Debug("skipping disabled skill")
			continue
		}

		// Load the skill from filesystem
		skillPath := filepath.Join(l.skillsDir, skillMeta.Path)

		// Check if skill directory exists
		if _, err := os.Stat(skillPath); os.IsNotExist(err) {
			l.logger.WithFields(map[string]interface{}{
				"skill_name": skillMeta.Name,
				"path":       skillPath,
			}).Warn("skill path does not exist")
			continue
		}

		skill, err := l.loadSkill(skillPath)
		if err != nil {
			l.logger.WithFields(map[string]interface{}{
				"skill_name": skillMeta.Name,
				"path":       skillPath,
				"error":      err.Error(),
			}).Warn("failed to load skill")
			continue
		}

		// Set the UserID from the metadata
		skill.UserID = skillMeta.UserID

		// Skip network-requiring skills in privacy mode
		if l.privacyMode && skill.RequiresNet {
			l.logger.WithFields(map[string]interface{}{
				"skill_name": skill.Name,
				"user_id":    userID,
			}).Debug("skipping skill (requires network)")
			continue
		}

		skills = append(skills, skill)
	}

	l.logger.WithFields(map[string]interface{}{
		"user_id": userID,
		"count":   len(skills),
	}).Debug("skills loaded for user")

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
