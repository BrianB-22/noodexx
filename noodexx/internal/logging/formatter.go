package logging

import (
	"fmt"
	"strings"
	"time"
)

// SourceLocation captures the source code location of a log call
type SourceLocation struct {
	File     string
	Line     int
	Function string
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp time.Time
	Level     Level
	Component string
	Source    SourceLocation
	Message   string
	Context   map[string]interface{}
}

// LogFormatter formats log entries into strings
type LogFormatter struct{}

// NewLogFormatter creates a new log formatter
func NewLogFormatter() *LogFormatter {
	return &LogFormatter{}
}

// Format formats a log entry into a string
// Output format: [YYYY-MM-DD HH:MM:SS] LEVEL [component] file.go:line function message key=value
func (f *LogFormatter) Format(entry LogEntry) string {
	var sb strings.Builder

	// Timestamp in ISO 8601 format
	timestamp := entry.Timestamp.Format("2006-01-02 15:04:05")
	sb.WriteString("[")
	sb.WriteString(timestamp)
	sb.WriteString("] ")

	// Level
	sb.WriteString(entry.Level.String())
	sb.WriteString(" ")

	// Component
	sb.WriteString("[")
	sb.WriteString(entry.Component)
	sb.WriteString("] ")

	// Source location
	sb.WriteString(entry.Source.File)
	sb.WriteString(":")
	sb.WriteString(fmt.Sprintf("%d", entry.Source.Line))
	sb.WriteString(" ")
	sb.WriteString(entry.Source.Function)
	sb.WriteString(" ")

	// Message (sanitized)
	sanitized := sanitizeMessage(entry.Message)
	sb.WriteString(sanitized)

	// Context fields as key=value pairs
	if len(entry.Context) > 0 {
		for key, value := range entry.Context {
			sb.WriteString(" ")
			sb.WriteString(key)
			sb.WriteString("=")
			sb.WriteString(fmt.Sprintf("%v", value))
		}
	}

	sb.WriteString("\n")
	return sb.String()
}

// sanitizeMessage removes control characters except \n and \t to prevent log injection
func sanitizeMessage(msg string) string {
	var sb strings.Builder
	for _, r := range msg {
		// Allow newline (0x0A) and tab (0x09)
		if r == '\n' || r == '\t' {
			sb.WriteRune(r)
		} else if r < 0x20 {
			// Replace other control characters (0x00-0x1F) with space
			sb.WriteRune(' ')
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}
