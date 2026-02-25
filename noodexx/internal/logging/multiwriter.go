package logging

import (
	"io"
	"strings"
)

// MultiWriter routes log messages to multiple destinations based on log level
type MultiWriter struct {
	consoleWriter io.Writer
	fileWriter    io.Writer
	debugEnabled  bool
}

// NewMultiWriter creates a new MultiWriter with the specified writers
func NewMultiWriter(consoleWriter, fileWriter io.Writer, debugEnabled bool) *MultiWriter {
	return &MultiWriter{
		consoleWriter: consoleWriter,
		fileWriter:    fileWriter,
		debugEnabled:  debugEnabled,
	}
}

// Write implements io.Writer interface and routes messages based on log level
func (m *MultiWriter) Write(p []byte) (n int, err error) {
	// If debug is disabled, write everything to console only
	if !m.debugEnabled {
		return m.consoleWriter.Write(p)
	}

	// Parse log level from message
	// Format: [YYYY-MM-DD HH:MM:SS] LEVEL [component] ...
	level := m.extractLevel(p)

	// Routing logic when debug is enabled:
	// - WARN/ERROR: write to both console and file
	// - DEBUG/INFO: write to file only
	var consoleErr, fileErr error
	var consoleN, fileN int

	if level == "WARN" || level == "ERROR" {
		// Write to console
		consoleN, consoleErr = m.consoleWriter.Write(p)
		// Write to file
		fileN, fileErr = m.fileWriter.Write(p)
	} else {
		// DEBUG or INFO: write to file only
		fileN, fileErr = m.fileWriter.Write(p)
		consoleN = len(p) // Pretend we wrote to console for return value
	}

	// Return the maximum bytes written and prioritize file errors
	n = fileN
	if consoleN > n {
		n = consoleN
	}

	// If file write failed, return that error
	if fileErr != nil {
		return n, fileErr
	}

	// Otherwise return console error if any
	return n, consoleErr
}

// extractLevel parses the log level from a formatted log message
// Expected format: [YYYY-MM-DD HH:MM:SS] LEVEL [component] ...
func (m *MultiWriter) extractLevel(p []byte) string {
	msg := string(p)

	// Find the first "] " which ends the timestamp
	firstBracket := strings.Index(msg, "] ")
	if firstBracket == -1 {
		return ""
	}

	// Skip past "] " to get to the level
	levelStart := firstBracket + 2

	// Find the next space which ends the level
	spaceAfterLevel := strings.Index(msg[levelStart:], " ")
	if spaceAfterLevel == -1 {
		return ""
	}

	// Extract the level
	level := msg[levelStart : levelStart+spaceAfterLevel]
	return level
}
