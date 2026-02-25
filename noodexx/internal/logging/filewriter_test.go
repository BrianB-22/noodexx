package logging

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestNewFileWriter tests basic file writer creation
func TestNewFileWriter(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	fw, err := NewFileWriter(logPath, 10, 3)
	if err != nil {
		t.Fatalf("NewFileWriter failed: %v", err)
	}
	defer fw.Close()

	// Verify file was created
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Errorf("Log file was not created")
	}

	// Verify file writer is not closed
	if fw.closed {
		t.Errorf("File writer should not be closed after creation")
	}
}

// TestFileWriterWrite tests writing to the buffer
func TestFileWriterWrite(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	fw, err := NewFileWriter(logPath, 10, 3)
	if err != nil {
		t.Fatalf("NewFileWriter failed: %v", err)
	}
	defer fw.Close()

	// Write some data
	data := []byte("test log message\n")
	n, err := fw.Write(data)
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Write returned %d bytes, expected %d", n, len(data))
	}

	// Flush to ensure data is written
	if err := fw.Flush(); err != nil {
		t.Errorf("Flush failed: %v", err)
	}

	// Read file and verify content
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	if string(content) != string(data) {
		t.Errorf("File content = %q, expected %q", string(content), string(data))
	}
}

// TestFileWriterFlush tests manual flushing
func TestFileWriterFlush(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	fw, err := NewFileWriter(logPath, 10, 3)
	if err != nil {
		t.Fatalf("NewFileWriter failed: %v", err)
	}
	defer fw.Close()

	// Write data
	data := []byte("test message\n")
	fw.Write(data)

	// Flush
	if err := fw.Flush(); err != nil {
		t.Errorf("Flush failed: %v", err)
	}

	// Verify data is on disk
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	if string(content) != string(data) {
		t.Errorf("File content = %q, expected %q", string(content), string(data))
	}
}

// TestFileWriterAutoFlush tests automatic flushing after 5 seconds
func TestFileWriterAutoFlush(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	fw, err := NewFileWriter(logPath, 10, 3)
	if err != nil {
		t.Fatalf("NewFileWriter failed: %v", err)
	}
	defer fw.Close()

	// Write data
	data := []byte("auto flush test\n")
	fw.Write(data)

	// Wait for auto flush (5 seconds + buffer)
	time.Sleep(6 * time.Second)

	// Verify data is on disk
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	if string(content) != string(data) {
		t.Errorf("File content = %q, expected %q", string(content), string(data))
	}
}

// TestFileWriterRotation tests log rotation when size threshold is exceeded
func TestFileWriterRotation(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	// Create file writer with very small max size (1 byte = 1/1024/1024 MB)
	// This means threshold is 1 byte, so any write will trigger rotation
	fw, err := NewFileWriter(logPath, 1, 2)
	if err != nil {
		t.Fatalf("NewFileWriter failed: %v", err)
	}
	defer fw.Close()

	// Write enough data to exceed 1MB threshold
	// Write 1.5MB of data to ensure rotation
	data := make([]byte, 1024*1024+512*1024) // 1.5MB
	for i := range data {
		data[i] = 'A'
	}

	fw.Write(data)
	fw.Flush()

	// Verify rotation occurred - backup file should exist
	backupPath := logPath + ".1"
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Errorf("Backup file %s was not created after rotation", backupPath)
	}

	// Verify new log file exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Errorf("New log file was not created after rotation")
	}
}

// TestFileWriterClose tests closing the file writer
func TestFileWriterClose(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	fw, err := NewFileWriter(logPath, 10, 3)
	if err != nil {
		t.Fatalf("NewFileWriter failed: %v", err)
	}

	// Write data
	data := []byte("close test\n")
	fw.Write(data)

	// Close
	if err := fw.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Verify closed flag is set
	if !fw.closed {
		t.Errorf("File writer should be closed")
	}

	// Verify data was flushed
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	if string(content) != string(data) {
		t.Errorf("File content = %q, expected %q", string(content), string(data))
	}

	// Verify writing after close returns error
	_, err = fw.Write([]byte("should fail"))
	if err == nil {
		t.Errorf("Write after close should return error")
	}
}

// TestFileWriterGracefulDegradation tests error handling when file operations fail
func TestFileWriterGracefulDegradation(t *testing.T) {
	// Test with invalid path
	invalidPath := "/invalid/path/that/does/not/exist/test.log"
	_, err := NewFileWriter(invalidPath, 10, 3)
	if err == nil {
		t.Errorf("NewFileWriter with invalid path should return error")
	}
}

// TestFileWriterConcurrentWrites tests thread-safe concurrent writes
func TestFileWriterConcurrentWrites(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	fw, err := NewFileWriter(logPath, 10, 3)
	if err != nil {
		t.Fatalf("NewFileWriter failed: %v", err)
	}
	defer fw.Close()

	// Write from multiple goroutines
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				fw.Write([]byte("concurrent write\n"))
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Flush and verify no errors
	if err := fw.Flush(); err != nil {
		t.Errorf("Flush after concurrent writes failed: %v", err)
	}
}

// TestFileWriterSharedReadAccess tests that the file can be read while being written
func TestFileWriterSharedReadAccess(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	fw, err := NewFileWriter(logPath, 10, 3)
	if err != nil {
		t.Fatalf("NewFileWriter failed: %v", err)
	}
	defer fw.Close()

	// Write some data
	data := []byte("shared read test\n")
	fw.Write(data)
	fw.Flush()

	// Try to open file for reading while writer is active
	file, err := os.Open(logPath)
	if err != nil {
		t.Errorf("Failed to open log file for reading: %v", err)
	}
	defer file.Close()

	// Read content
	content := make([]byte, len(data))
	n, err := file.Read(content)
	if err != nil {
		t.Errorf("Failed to read log file: %v", err)
	}
	if n != len(data) {
		t.Errorf("Read %d bytes, expected %d", n, len(data))
	}
	if string(content) != string(data) {
		t.Errorf("Read content = %q, expected %q", string(content), string(data))
	}
}
