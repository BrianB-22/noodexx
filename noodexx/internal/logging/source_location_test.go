package logging

import (
	"bytes"
	"strings"
	"testing"
)

func TestSourceLocationCapture(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("test", DEBUG, &buf)

	logger.Info("test message")

	output := buf.String()

	// Check that output contains source location information
	if !strings.Contains(output, "source_location_test.go:") {
		t.Errorf("Log output should contain source file name: got %q", output)
	}

	// Check that output contains function name
	if !strings.Contains(output, "TestSourceLocationCapture") {
		t.Errorf("Log output should contain function name: got %q", output)
	}

	// Check that output still contains the message
	if !strings.Contains(output, "test message") {
		t.Errorf("Log output should contain message: got %q", output)
	}

	// Check that output contains level
	if !strings.Contains(output, "INFO") {
		t.Errorf("Log output should contain level: got %q", output)
	}

	// Check that output contains component
	if !strings.Contains(output, "[test]") {
		t.Errorf("Log output should contain component: got %q", output)
	}
}

func TestSourceLocationWithContext(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("test", DEBUG, &buf)

	logger.WithContext("key", "value").Warn("warning with context")

	output := buf.String()

	// Check that output contains source location
	if !strings.Contains(output, "source_location_test.go:") {
		t.Errorf("Log output should contain source file name: got %q", output)
	}

	// Check that output contains context
	if !strings.Contains(output, "key=value") {
		t.Errorf("Log output should contain context: got %q", output)
	}

	// Check that output contains message
	if !strings.Contains(output, "warning with context") {
		t.Errorf("Log output should contain message: got %q", output)
	}
}

func TestSourceLocationDifferentLevels(t *testing.T) {
	tests := []struct {
		name    string
		logFunc func(*Logger)
		level   string
	}{
		{"Debug", func(l *Logger) { l.Debug("debug message") }, "DEBUG"},
		{"Info", func(l *Logger) { l.Info("info message") }, "INFO"},
		{"Warn", func(l *Logger) { l.Warn("warn message") }, "WARN"},
		{"Error", func(l *Logger) { l.Error("error message") }, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := NewLogger("test", DEBUG, &buf)

			tt.logFunc(logger)

			output := buf.String()

			// Check that output contains source location
			if !strings.Contains(output, "source_location_test.go:") {
				t.Errorf("Log output should contain source file name: got %q", output)
			}

			// Check that output contains level
			if !strings.Contains(output, tt.level) {
				t.Errorf("Log output should contain level %s: got %q", tt.level, output)
			}
		})
	}
}
