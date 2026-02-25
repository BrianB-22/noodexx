package logging

import (
	"fmt"
	"os"
)

// LogRotator handles automatic log file rotation based on size thresholds.
// It manages backup files by renaming and deleting old backups when limits are exceeded.
type LogRotator struct {
	basePath   string // Base path of the log file (e.g., "debug.log")
	maxSizeMB  int    // Maximum file size in megabytes before rotation
	maxBackups int    // Maximum number of backup files to keep
}

// NewLogRotator creates a new LogRotator with the specified configuration.
func NewLogRotator(basePath string, maxSizeMB, maxBackups int) *LogRotator {
	return &LogRotator{
		basePath:   basePath,
		maxSizeMB:  maxSizeMB,
		maxBackups: maxBackups,
	}
}

// ShouldRotate checks if the current file size exceeds the rotation threshold.
// Returns true if currentSize exceeds maxSizeMB * 1024 * 1024 bytes.
func (r *LogRotator) ShouldRotate(currentSize int64) bool {
	threshold := int64(r.maxSizeMB) * 1024 * 1024
	return currentSize >= threshold
}

// Rotate performs log file rotation by:
// 1. Renaming existing backups in reverse order: debug.log.N → debug.log.N+1
// 2. Deleting the oldest backup if count exceeds maxBackups
// 3. Renaming current file: debug.log → debug.log.1
//
// Returns an error if rotation fails. The caller should handle the error gracefully
// (e.g., log to console and continue with the current file).
func (r *LogRotator) Rotate() error {
	// Handle edge case: maxBackups = 0 means no backups, just delete current file
	if r.maxBackups == 0 {
		if err := os.Remove(r.basePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove current log file: %w", err)
		}
		return nil
	}

	// Step 1 & 2: Rename existing backups in reverse order and delete oldest
	// First, delete the oldest backup if it exists (the one that would exceed maxBackups)
	oldestPath := fmt.Sprintf("%s.%d", r.basePath, r.maxBackups)
	if _, err := os.Stat(oldestPath); err == nil {
		if err := os.Remove(oldestPath); err != nil {
			return fmt.Errorf("failed to delete oldest backup %s: %w", oldestPath, err)
		}
	}

	// Now rename existing backups in reverse order: N → N+1
	// Start from maxBackups-1 and work backwards to 1
	for i := r.maxBackups - 1; i >= 1; i-- {
		oldPath := fmt.Sprintf("%s.%d", r.basePath, i)
		newPath := fmt.Sprintf("%s.%d", r.basePath, i+1)

		// Check if the old backup exists
		if _, err := os.Stat(oldPath); err == nil {
			// Rename it to increment the number
			if err := os.Rename(oldPath, newPath); err != nil {
				return fmt.Errorf("failed to rename backup %s to %s: %w", oldPath, newPath, err)
			}
		}
		// If the file doesn't exist, that's fine - just continue
	}

	// Step 3: Rename current file to .1
	// Check if current file exists before trying to rename
	if _, err := os.Stat(r.basePath); err == nil {
		backupPath := fmt.Sprintf("%s.1", r.basePath)
		if err := os.Rename(r.basePath, backupPath); err != nil {
			return fmt.Errorf("failed to rename current log %s to %s: %w", r.basePath, backupPath, err)
		}
	} else if !os.IsNotExist(err) {
		// If there's an error other than "file doesn't exist", return it
		return fmt.Errorf("failed to stat current log file: %w", err)
	}
	// If file doesn't exist, that's fine - rotation is complete

	return nil
}
