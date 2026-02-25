package logging

import (
	"bufio"
	"fmt"
	"os"
	"sync"
	"time"
)

// FileWriter manages buffered file writing with automatic rotation and shared access.
// It provides thread-safe writes, periodic flushing, and graceful error handling.
type FileWriter struct {
	path       string        // Path to the log file
	file       *os.File      // File handle
	buffer     *bufio.Writer // Buffered writer (64KB)
	rotator    *LogRotator   // Log rotator for size-based rotation
	mu         sync.Mutex    // Mutex for thread-safe operations
	flushTimer *time.Timer   // Timer for periodic flushing (5 seconds)
	closed     bool          // Flag to track if writer is closed
}

// NewFileWriter creates a new FileWriter with the specified configuration.
// It opens the file with append mode and shared read access, creates a 64KB buffer,
// and sets up a 5-second flush timer.
//
// Parameters:
//   - path: Path to the log file
//   - maxSizeMB: Maximum file size in MB before rotation
//   - maxBackups: Number of backup files to keep
//
// Returns an error if file creation fails. The caller should handle this gracefully
// by logging to console and falling back to console-only logging.
func NewFileWriter(path string, maxSizeMB int, maxBackups int) (*FileWriter, error) {
	// Open file with create, write-only, append flags and 0644 permissions
	// This allows shared read access on Unix systems
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file %s: %w", path, err)
	}

	// Create 64KB buffer
	buffer := bufio.NewWriterSize(file, 64*1024)

	// Create rotator
	rotator := NewLogRotator(path, maxSizeMB, maxBackups)

	fw := &FileWriter{
		path:    path,
		file:    file,
		buffer:  buffer,
		rotator: rotator,
		closed:  false,
	}

	// Set up 5-second flush timer
	fw.flushTimer = time.AfterFunc(5*time.Second, func() {
		fw.mu.Lock()
		defer fw.mu.Unlock()
		if !fw.closed {
			fw.flushInternal()
			// Reset timer for next flush
			fw.flushTimer.Reset(5 * time.Second)
		}
	})

	return fw, nil
}

// Write appends data to the buffer.
// This method is thread-safe and implements the io.Writer interface.
//
// Returns the number of bytes written and any error encountered.
// If an error occurs, it is logged to console and the writer continues operation.
func (fw *FileWriter) Write(p []byte) (n int, err error) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.closed {
		return 0, fmt.Errorf("file writer is closed")
	}

	// Write to buffer
	n, err = fw.buffer.Write(p)
	if err != nil {
		// Log error to console and continue
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to write to log buffer: %v\n", err)
		return n, err
	}

	return n, nil
}

// Flush writes the buffer to disk and checks if rotation is needed.
// This method is thread-safe and should be called periodically or before closing.
//
// Returns an error if flushing or rotation fails. Errors are logged to console
// and the writer continues operation.
func (fw *FileWriter) Flush() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.closed {
		return fmt.Errorf("file writer is closed")
	}

	return fw.flushInternal()
}

// flushInternal performs the actual flush operation without locking.
// This is called by Flush() and the flush timer.
// Caller must hold the mutex.
func (fw *FileWriter) flushInternal() error {
	// Flush buffer to disk
	if err := fw.buffer.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to flush log buffer: %v\n", err)
		return err
	}

	// Check if rotation is needed
	fileInfo, err := fw.file.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to stat log file: %v\n", err)
		return err
	}

	if fw.rotator.ShouldRotate(fileInfo.Size()) {
		if err := fw.rotate(); err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to rotate log file: %v\n", err)
			// Continue operation with current file even if rotation fails
			return err
		}
	}

	return nil
}

// rotate performs log file rotation.
// Caller must hold the mutex.
func (fw *FileWriter) rotate() error {
	// Close current file
	if err := fw.file.Close(); err != nil {
		return fmt.Errorf("failed to close file before rotation: %w", err)
	}

	// Perform rotation (rename files)
	if err := fw.rotator.Rotate(); err != nil {
		// Try to reopen the file even if rotation failed
		file, reopenErr := os.OpenFile(fw.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if reopenErr != nil {
			return fmt.Errorf("rotation failed and could not reopen file: %v (original error: %w)", reopenErr, err)
		}
		fw.file = file
		fw.buffer = bufio.NewWriterSize(file, 64*1024)
		return err
	}

	// Reopen file after rotation
	file, err := os.OpenFile(fw.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to reopen file after rotation: %w", err)
	}

	fw.file = file
	fw.buffer = bufio.NewWriterSize(file, 64*1024)

	return nil
}

// Close stops the flush timer, flushes the buffer, and closes the file.
// This method should be called during application shutdown to ensure all
// buffered messages are written to disk.
//
// Returns an error if flushing or closing fails.
func (fw *FileWriter) Close() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.closed {
		return nil
	}

	fw.closed = true

	// Stop flush timer
	if fw.flushTimer != nil {
		fw.flushTimer.Stop()
	}

	// Flush buffer
	if err := fw.buffer.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to flush buffer during close: %v\n", err)
		// Continue to close file even if flush fails
	}

	// Close file
	if err := fw.file.Close(); err != nil {
		return fmt.Errorf("failed to close log file: %w", err)
	}

	return nil
}
