package logging

import (
	"bytes"
	"strings"
	"testing"
)

// TestMultiWriter_DebugDisabled tests that all messages go to console only when debug is disabled
func TestMultiWriter_DebugDisabled(t *testing.T) {
	consoleBuffer := &bytes.Buffer{}
	fileBuffer := &bytes.Buffer{}

	mw := NewMultiWriter(consoleBuffer, fileBuffer, false)

	testMessages := []string{
		"[2024-01-15 14:32:45] DEBUG [test] test message\n",
		"[2024-01-15 14:32:46] INFO [test] test message\n",
		"[2024-01-15 14:32:47] WARN [test] test message\n",
		"[2024-01-15 14:32:48] ERROR [test] test message\n",
	}

	for _, msg := range testMessages {
		n, err := mw.Write([]byte(msg))
		if err != nil {
			t.Errorf("Write() error = %v", err)
		}
		if n != len(msg) {
			t.Errorf("Write() wrote %d bytes, want %d", n, len(msg))
		}
	}

	// All messages should be in console
	consoleOutput := consoleBuffer.String()
	for _, msg := range testMessages {
		if !strings.Contains(consoleOutput, msg) {
			t.Errorf("Console output missing message: %s", msg)
		}
	}

	// File should be empty
	if fileBuffer.Len() != 0 {
		t.Errorf("File buffer should be empty when debug disabled, got: %s", fileBuffer.String())
	}
}

// TestMultiWriter_DebugEnabled_WarnError tests that WARN/ERROR go to both console and file
func TestMultiWriter_DebugEnabled_WarnError(t *testing.T) {
	consoleBuffer := &bytes.Buffer{}
	fileBuffer := &bytes.Buffer{}

	mw := NewMultiWriter(consoleBuffer, fileBuffer, true)

	warnMsg := "[2024-01-15 14:32:47] WARN [test] warning message\n"
	errorMsg := "[2024-01-15 14:32:48] ERROR [test] error message\n"

	// Write WARN message
	n, err := mw.Write([]byte(warnMsg))
	if err != nil {
		t.Errorf("Write(WARN) error = %v", err)
	}
	if n != len(warnMsg) {
		t.Errorf("Write(WARN) wrote %d bytes, want %d", n, len(warnMsg))
	}

	// Write ERROR message
	n, err = mw.Write([]byte(errorMsg))
	if err != nil {
		t.Errorf("Write(ERROR) error = %v", err)
	}
	if n != len(errorMsg) {
		t.Errorf("Write(ERROR) wrote %d bytes, want %d", n, len(errorMsg))
	}

	// Both messages should be in console
	consoleOutput := consoleBuffer.String()
	if !strings.Contains(consoleOutput, warnMsg) {
		t.Errorf("Console output missing WARN message")
	}
	if !strings.Contains(consoleOutput, errorMsg) {
		t.Errorf("Console output missing ERROR message")
	}

	// Both messages should be in file
	fileOutput := fileBuffer.String()
	if !strings.Contains(fileOutput, warnMsg) {
		t.Errorf("File output missing WARN message")
	}
	if !strings.Contains(fileOutput, errorMsg) {
		t.Errorf("File output missing ERROR message")
	}
}

// TestMultiWriter_DebugEnabled_DebugInfo tests that DEBUG/INFO go to file only
func TestMultiWriter_DebugEnabled_DebugInfo(t *testing.T) {
	consoleBuffer := &bytes.Buffer{}
	fileBuffer := &bytes.Buffer{}

	mw := NewMultiWriter(consoleBuffer, fileBuffer, true)

	debugMsg := "[2024-01-15 14:32:45] DEBUG [test] debug message\n"
	infoMsg := "[2024-01-15 14:32:46] INFO [test] info message\n"

	// Write DEBUG message
	n, err := mw.Write([]byte(debugMsg))
	if err != nil {
		t.Errorf("Write(DEBUG) error = %v", err)
	}
	if n != len(debugMsg) {
		t.Errorf("Write(DEBUG) wrote %d bytes, want %d", n, len(debugMsg))
	}

	// Write INFO message
	n, err = mw.Write([]byte(infoMsg))
	if err != nil {
		t.Errorf("Write(INFO) error = %v", err)
	}
	if n != len(infoMsg) {
		t.Errorf("Write(INFO) wrote %d bytes, want %d", n, len(infoMsg))
	}

	// Console should be empty
	if consoleBuffer.Len() != 0 {
		t.Errorf("Console buffer should be empty for DEBUG/INFO, got: %s", consoleBuffer.String())
	}

	// Both messages should be in file
	fileOutput := fileBuffer.String()
	if !strings.Contains(fileOutput, debugMsg) {
		t.Errorf("File output missing DEBUG message")
	}
	if !strings.Contains(fileOutput, infoMsg) {
		t.Errorf("File output missing INFO message")
	}
}

// TestMultiWriter_ExtractLevel tests the level extraction logic
func TestMultiWriter_ExtractLevel(t *testing.T) {
	mw := NewMultiWriter(nil, nil, true)

	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "DEBUG level",
			message:  "[2024-01-15 14:32:45] DEBUG [test] message",
			expected: "DEBUG",
		},
		{
			name:     "INFO level",
			message:  "[2024-01-15 14:32:45] INFO [test] message",
			expected: "INFO",
		},
		{
			name:     "WARN level",
			message:  "[2024-01-15 14:32:45] WARN [test] message",
			expected: "WARN",
		},
		{
			name:     "ERROR level",
			message:  "[2024-01-15 14:32:45] ERROR [test] message",
			expected: "ERROR",
		},
		{
			name:     "Malformed message - no bracket",
			message:  "invalid message",
			expected: "",
		},
		{
			name:     "Malformed message - no space after level",
			message:  "[2024-01-15 14:32:45] DEBUG",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level := mw.extractLevel([]byte(tt.message))
			if level != tt.expected {
				t.Errorf("extractLevel() = %q, want %q", level, tt.expected)
			}
		})
	}
}

// TestMultiWriter_MixedLevels tests routing with mixed log levels
func TestMultiWriter_MixedLevels(t *testing.T) {
	consoleBuffer := &bytes.Buffer{}
	fileBuffer := &bytes.Buffer{}

	mw := NewMultiWriter(consoleBuffer, fileBuffer, true)

	messages := []struct {
		msg           string
		shouldConsole bool
		shouldFile    bool
	}{
		{"[2024-01-15 14:32:45] DEBUG [test] debug\n", false, true},
		{"[2024-01-15 14:32:46] INFO [test] info\n", false, true},
		{"[2024-01-15 14:32:47] WARN [test] warn\n", true, true},
		{"[2024-01-15 14:32:48] ERROR [test] error\n", true, true},
		{"[2024-01-15 14:32:49] DEBUG [test] debug2\n", false, true},
		{"[2024-01-15 14:32:50] WARN [test] warn2\n", true, true},
	}

	for _, m := range messages {
		_, err := mw.Write([]byte(m.msg))
		if err != nil {
			t.Errorf("Write() error = %v for message: %s", err, m.msg)
		}
	}

	consoleOutput := consoleBuffer.String()
	fileOutput := fileBuffer.String()

	for _, m := range messages {
		consoleHas := strings.Contains(consoleOutput, m.msg)
		fileHas := strings.Contains(fileOutput, m.msg)

		if m.shouldConsole && !consoleHas {
			t.Errorf("Console missing message: %s", m.msg)
		}
		if !m.shouldConsole && consoleHas {
			t.Errorf("Console should not have message: %s", m.msg)
		}
		if m.shouldFile && !fileHas {
			t.Errorf("File missing message: %s", m.msg)
		}
		if !m.shouldFile && fileHas {
			t.Errorf("File should not have message: %s", m.msg)
		}
	}
}

// TestMultiWriter_EmptyMessage tests handling of empty messages
func TestMultiWriter_EmptyMessage(t *testing.T) {
	consoleBuffer := &bytes.Buffer{}
	fileBuffer := &bytes.Buffer{}

	mw := NewMultiWriter(consoleBuffer, fileBuffer, true)

	n, err := mw.Write([]byte(""))
	if err != nil {
		t.Errorf("Write() error = %v", err)
	}
	if n != 0 {
		t.Errorf("Write() wrote %d bytes, want 0", n)
	}
}

// TestMultiWriter_LargeMessage tests handling of large messages
func TestMultiWriter_LargeMessage(t *testing.T) {
	consoleBuffer := &bytes.Buffer{}
	fileBuffer := &bytes.Buffer{}

	mw := NewMultiWriter(consoleBuffer, fileBuffer, true)

	// Create a large message (10KB)
	largeContent := strings.Repeat("x", 10000)
	largeMsg := "[2024-01-15 14:32:45] ERROR [test] " + largeContent + "\n"

	n, err := mw.Write([]byte(largeMsg))
	if err != nil {
		t.Errorf("Write() error = %v", err)
	}
	if n != len(largeMsg) {
		t.Errorf("Write() wrote %d bytes, want %d", n, len(largeMsg))
	}

	// Should be in both console and file (ERROR level)
	if !strings.Contains(consoleBuffer.String(), largeContent) {
		t.Errorf("Console missing large message content")
	}
	if !strings.Contains(fileBuffer.String(), largeContent) {
		t.Errorf("File missing large message content")
	}
}
