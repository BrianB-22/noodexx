package main

import (
	"os"
	"testing"

	"noodexx/internal/config"
)

func TestInitializeLogging(t *testing.T) {
	tests := []struct {
		name          string
		cfg           *config.Config
		expectError   bool
		checkFileMode bool
	}{
		{
			name: "debug enabled with valid file path",
			cfg: &config.Config{
				Logging: config.LoggingConfig{
					Level:        "info",
					DebugEnabled: true,
					File:         "test_debug.log",
					MaxSizeMB:    10,
					MaxBackups:   3,
				},
			},
			expectError:   false,
			checkFileMode: true,
		},
		{
			name: "debug disabled",
			cfg: &config.Config{
				Logging: config.LoggingConfig{
					Level:        "info",
					DebugEnabled: false,
					File:         "",
					MaxSizeMB:    10,
					MaxBackups:   3,
				},
			},
			expectError:   false,
			checkFileMode: false,
		},
		{
			name: "debug enabled with invalid path (should fallback)",
			cfg: &config.Config{
				Logging: config.LoggingConfig{
					Level:        "info",
					DebugEnabled: true,
					File:         "/invalid/path/debug.log",
					MaxSizeMB:    10,
					MaxBackups:   3,
				},
			},
			expectError:   false, // Should not error, just fallback to console
			checkFileMode: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _, err := initializeLogging(tt.cfg)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if logger == nil {
				t.Errorf("expected logger to be non-nil")
			}

			// Clean up test file if created
			if tt.checkFileMode && tt.cfg.Logging.File != "" {
				os.Remove(tt.cfg.Logging.File)
			}
		})
	}
}
