package skills

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"noodexx/internal/logging"
	"os"
	"os/exec"
	"strings"
)

// Executor runs skills as subprocesses
type Executor struct {
	privacyMode bool
	logger      *logging.Logger
}

// NewExecutor creates a skill executor
func NewExecutor(privacyMode bool, logger *logging.Logger) *Executor {
	return &Executor{
		privacyMode: privacyMode,
		logger:      logger,
	}
}

// Input is the JSON sent to skill stdin
type Input struct {
	Query    string                 `json:"query"`
	Context  map[string]interface{} `json:"context"`
	Settings map[string]interface{} `json:"settings"`
}

// Output is the JSON received from skill stdout
type Output struct {
	Result   string                 `json:"result"`
	Error    string                 `json:"error"`
	Metadata map[string]interface{} `json:"metadata"`
}

// Execute runs a skill with the given input
func (e *Executor) Execute(ctx context.Context, skill *Skill, input Input) (*Output, error) {
	logger := e.logger.WithFields(map[string]interface{}{
		"skill_name": skill.Name,
		"skill_path": skill.Path,
	})
	logger.Debug("starting skill execution")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, skill.Timeout)
	defer cancel()

	// Prepare command
	cmd := exec.CommandContext(ctx, skill.Executable)
	cmd.Dir = skill.Path

	// Set environment variables
	cmd.Env = e.buildEnv(skill)

	// Prepare input JSON
	inputJSON, err := json.Marshal(input)
	if err != nil {
		logger.WithContext("error", err.Error()).Error("failed to marshal input")
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}

	cmd.Stdin = bytes.NewReader(inputJSON)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run command
	err = cmd.Run()

	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		logger.WithContext("timeout", skill.Timeout).Error("skill execution timed out")
		return nil, fmt.Errorf("skill execution timed out after %v", skill.Timeout)
	}

	// Parse output
	var output Output
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		logger.WithFields(map[string]interface{}{
			"error":  err.Error(),
			"stderr": stderr.String(),
		}).Error("failed to parse skill output")
		return nil, fmt.Errorf("failed to parse skill output: %w (stderr: %s)", err, stderr.String())
	}

	if output.Error != "" {
		logger.WithContext("skill_error", output.Error).Error("skill returned error")
		return &output, fmt.Errorf("skill error: %s", output.Error)
	}

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	logger.WithContext("exit_code", exitCode).Debug("skill execution completed")
	return &output, nil
}

// buildEnv creates environment variables for the skill
func (e *Executor) buildEnv(skill *Skill) []string {
	env := []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
		"USER=" + os.Getenv("USER"),
		"NOODEXX_SKILL_NAME=" + skill.Name,
		"NOODEXX_SKILL_VERSION=" + skill.Version,
	}

	if e.privacyMode {
		env = append(env, "NOODEXX_PRIVACY_MODE=true")
	}

	// Add skill-specific settings as env vars
	for key, value := range skill.Settings {
		env = append(env, fmt.Sprintf("NOODEXX_SETTING_%s=%v", strings.ToUpper(key), value))
	}

	return env
}
