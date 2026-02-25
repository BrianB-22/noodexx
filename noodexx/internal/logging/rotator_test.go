package logging

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLogRotator_ShouldRotate(t *testing.T) {
	tests := []struct {
		name        string
		maxSizeMB   int
		currentSize int64
		want        bool
	}{
		{
			name:        "size below threshold",
			maxSizeMB:   10,
			currentSize: 5 * 1024 * 1024, // 5MB
			want:        false,
		},
		{
			name:        "size exactly at threshold",
			maxSizeMB:   10,
			currentSize: 10 * 1024 * 1024, // 10MB
			want:        true,
		},
		{
			name:        "size above threshold",
			maxSizeMB:   10,
			currentSize: 15 * 1024 * 1024, // 15MB
			want:        true,
		},
		{
			name:        "zero size",
			maxSizeMB:   10,
			currentSize: 0,
			want:        false,
		},
		{
			name:        "small threshold",
			maxSizeMB:   1,
			currentSize: 1024 * 1024, // 1MB
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewLogRotator("test.log", tt.maxSizeMB, 3)
			got := r.ShouldRotate(tt.currentSize)
			if got != tt.want {
				t.Errorf("ShouldRotate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLogRotator_Rotate(t *testing.T) {
	tests := []struct {
		name           string
		maxBackups     int
		existingFiles  []string // Files to create before rotation
		expectedFiles  []string // Files that should exist after rotation
		expectedErrors bool
	}{
		{
			name:       "no existing backups",
			maxBackups: 3,
			existingFiles: []string{
				"test.log",
			},
			expectedFiles: []string{
				"test.log.1",
			},
		},
		{
			name:       "one existing backup",
			maxBackups: 3,
			existingFiles: []string{
				"test.log",
				"test.log.1",
			},
			expectedFiles: []string{
				"test.log.1",
				"test.log.2",
			},
		},
		{
			name:       "multiple existing backups",
			maxBackups: 3,
			existingFiles: []string{
				"test.log",
				"test.log.1",
				"test.log.2",
			},
			expectedFiles: []string{
				"test.log.1",
				"test.log.2",
				"test.log.3",
			},
		},
		{
			name:       "delete oldest backup when at limit",
			maxBackups: 3,
			existingFiles: []string{
				"test.log",
				"test.log.1",
				"test.log.2",
				"test.log.3",
			},
			expectedFiles: []string{
				"test.log.1",
				"test.log.2",
				"test.log.3",
			},
		},
		{
			name:       "maxBackups is 0",
			maxBackups: 0,
			existingFiles: []string{
				"test.log",
			},
			expectedFiles: []string{},
		},
		{
			name:       "maxBackups is 1",
			maxBackups: 1,
			existingFiles: []string{
				"test.log",
			},
			expectedFiles: []string{
				"test.log.1",
			},
		},
		{
			name:       "maxBackups is 1 with existing backup",
			maxBackups: 1,
			existingFiles: []string{
				"test.log",
				"test.log.1",
			},
			expectedFiles: []string{
				"test.log.1",
			},
		},
		{
			name:          "no existing file",
			maxBackups:    3,
			existingFiles: []string{},
			expectedFiles: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test
			tmpDir := t.TempDir()
			basePath := filepath.Join(tmpDir, "test.log")

			// Create existing files
			for _, file := range tt.existingFiles {
				path := filepath.Join(tmpDir, file)
				if err := os.WriteFile(path, []byte("test content"), 0644); err != nil {
					t.Fatalf("failed to create test file %s: %v", file, err)
				}
			}

			// Create rotator and perform rotation
			r := NewLogRotator(basePath, 10, tt.maxBackups)
			err := r.Rotate()

			if tt.expectedErrors && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectedErrors && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Verify expected files exist
			for _, expectedFile := range tt.expectedFiles {
				path := filepath.Join(tmpDir, expectedFile)
				if _, err := os.Stat(path); os.IsNotExist(err) {
					t.Errorf("expected file %s does not exist", expectedFile)
				}
			}

			// Verify original file doesn't exist (unless maxBackups is 0 and no file was created)
			if len(tt.existingFiles) > 0 {
				if _, err := os.Stat(basePath); !os.IsNotExist(err) {
					// Original file should not exist after rotation
					// (unless it was recreated, which is not part of Rotate's responsibility)
					if err == nil {
						// Check if this is expected (e.g., when no rotation happened)
						found := false
						for _, ef := range tt.expectedFiles {
							if ef == "test.log" {
								found = true
								break
							}
						}
						if !found {
							t.Errorf("original file %s still exists after rotation", basePath)
						}
					}
				}
			}

			// Verify no unexpected files exist
			entries, err := os.ReadDir(tmpDir)
			if err != nil {
				t.Fatalf("failed to read temp directory: %v", err)
			}

			for _, entry := range entries {
				found := false
				for _, expectedFile := range tt.expectedFiles {
					if entry.Name() == filepath.Base(expectedFile) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("unexpected file exists: %s", entry.Name())
				}
			}
		})
	}
}

func TestLogRotator_Rotate_ErrorHandling(t *testing.T) {
	t.Run("handles missing current file gracefully", func(t *testing.T) {
		tmpDir := t.TempDir()
		basePath := filepath.Join(tmpDir, "nonexistent.log")

		r := NewLogRotator(basePath, 10, 3)
		err := r.Rotate()

		// Should not return an error when file doesn't exist
		if err != nil {
			t.Errorf("unexpected error when rotating non-existent file: %v", err)
		}
	})

	t.Run("handles missing backup files gracefully", func(t *testing.T) {
		tmpDir := t.TempDir()
		basePath := filepath.Join(tmpDir, "test.log")

		// Create only the current file, no backups
		if err := os.WriteFile(basePath, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		r := NewLogRotator(basePath, 10, 5)
		err := r.Rotate()

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Verify current file was renamed to .1
		backupPath := basePath + ".1"
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			t.Error("backup file .1 was not created")
		}
	})
}

func TestLogRotator_Integration(t *testing.T) {
	t.Run("full rotation cycle", func(t *testing.T) {
		tmpDir := t.TempDir()
		basePath := filepath.Join(tmpDir, "test.log")
		maxBackups := 3

		r := NewLogRotator(basePath, 1, maxBackups) // 1MB threshold

		// Simulate multiple rotations
		for i := 0; i < 5; i++ {
			// Create a file
			content := []byte("log content for rotation " + string(rune('0'+i)))
			if err := os.WriteFile(basePath, content, 0644); err != nil {
				t.Fatalf("failed to create log file: %v", err)
			}

			// Perform rotation
			if err := r.Rotate(); err != nil {
				t.Fatalf("rotation %d failed: %v", i, err)
			}
		}

		// After 5 rotations with maxBackups=3, we should have exactly 3 backup files
		// (test.log.1, test.log.2, test.log.3)
		for i := 1; i <= maxBackups; i++ {
			backupPath := filepath.Join(tmpDir, "test.log."+string(rune('0'+i)))
			if _, err := os.Stat(backupPath); os.IsNotExist(err) {
				t.Errorf("expected backup file test.log.%d does not exist", i)
			}
		}

		// Verify no extra backups exist
		backupPath := filepath.Join(tmpDir, "test.log.4")
		if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
			t.Error("unexpected backup file test.log.4 exists")
		}
	})
}
