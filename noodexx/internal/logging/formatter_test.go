package logging

import (
	"strings"
	"testing"
	"time"
)

func TestLogFormatter_Format(t *testing.T) {
	formatter := NewLogFormatter()

	tests := []struct {
		name     string
		entry    LogEntry
		contains []string
	}{
		{
			name: "basic log entry",
			entry: LogEntry{
				Timestamp: time.Date(2024, 1, 15, 14, 32, 45, 0, time.UTC),
				Level:     INFO,
				Component: "api",
				Source: SourceLocation{
					File:     "handlers.go",
					Line:     123,
					Function: "HandleChat",
				},
				Message: "processing request",
				Context: nil,
			},
			contains: []string{
				"[2024-01-15 14:32:45]",
				"INFO",
				"[api]",
				"handlers.go:123",
				"HandleChat",
				"processing request",
			},
		},
		{
			name: "log entry with context",
			entry: LogEntry{
				Timestamp: time.Date(2024, 1, 15, 14, 32, 45, 0, time.UTC),
				Level:     DEBUG,
				Component: "llm",
				Source: SourceLocation{
					File:     "openai.go",
					Line:     45,
					Function: "Chat",
				},
				Message: "sending request",
				Context: map[string]interface{}{
					"provider": "openai",
					"model":    "gpt-4",
					"tokens":   150,
				},
			},
			contains: []string{
				"[2024-01-15 14:32:45]",
				"DEBUG",
				"[llm]",
				"openai.go:45",
				"Chat",
				"sending request",
				"provider=openai",
				"model=gpt-4",
				"tokens=150",
			},
		},
		{
			name: "log entry with newlines",
			entry: LogEntry{
				Timestamp: time.Date(2024, 1, 15, 14, 32, 45, 0, time.UTC),
				Level:     ERROR,
				Component: "store",
				Source: SourceLocation{
					File:     "store.go",
					Line:     200,
					Function: "Query",
				},
				Message: "query failed\ndetails: connection timeout",
				Context: nil,
			},
			contains: []string{
				"[2024-01-15 14:32:45]",
				"ERROR",
				"[store]",
				"store.go:200",
				"Query",
				"query failed\ndetails: connection timeout",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.Format(tt.entry)

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Format() result missing expected substring:\nwant: %q\ngot: %q", expected, result)
				}
			}

			// Verify it ends with newline
			if !strings.HasSuffix(result, "\n") {
				t.Errorf("Format() result should end with newline")
			}
		})
	}
}

func TestSanitizeMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no control characters",
			input:    "normal message",
			expected: "normal message",
		},
		{
			name:     "preserve newline",
			input:    "line1\nline2",
			expected: "line1\nline2",
		},
		{
			name:     "preserve tab",
			input:    "col1\tcol2",
			expected: "col1\tcol2",
		},
		{
			name:     "remove null byte",
			input:    "text\x00more",
			expected: "text more",
		},
		{
			name:     "remove bell character",
			input:    "alert\x07here",
			expected: "alert here",
		},
		{
			name:     "remove escape sequence",
			input:    "color\x1b[31mred",
			expected: "color [31mred",
		},
		{
			name:     "mixed control characters",
			input:    "a\x00b\nc\td\x07e",
			expected: "a b\nc\td e",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeMessage(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeMessage() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestLogFormatter_FormatWithControlCharacters(t *testing.T) {
	formatter := NewLogFormatter()

	entry := LogEntry{
		Timestamp: time.Date(2024, 1, 15, 14, 32, 45, 0, time.UTC),
		Level:     WARN,
		Component: "test",
		Source: SourceLocation{
			File:     "test.go",
			Line:     1,
			Function: "TestFunc",
		},
		Message: "message with\x00null\x07bell\x1bescape",
		Context: nil,
	}

	result := formatter.Format(entry)

	// Verify control characters are sanitized
	if strings.Contains(result, "\x00") {
		t.Error("Format() should remove null bytes")
	}
	if strings.Contains(result, "\x07") {
		t.Error("Format() should remove bell characters")
	}
	if strings.Contains(result, "\x1b") {
		t.Error("Format() should remove escape characters")
	}

	// Verify message is still present (with spaces replacing control chars)
	if !strings.Contains(result, "message with null bell escape") {
		t.Errorf("Format() should preserve message text with sanitized control chars, got: %q", result)
	}
}
