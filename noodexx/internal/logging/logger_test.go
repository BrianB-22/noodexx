package logging

import (
	"bytes"
	"strings"
	"testing"
)

func TestLevelString(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
		{Level(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := tt.level.String(); got != tt.expected {
			t.Errorf("Level.String() = %v, want %v", got, tt.expected)
		}
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{"debug", DEBUG},
		{"DEBUG", DEBUG},
		{"info", INFO},
		{"INFO", INFO},
		{"warn", WARN},
		{"WARN", WARN},
		{"error", ERROR},
		{"ERROR", ERROR},
		{"invalid", INFO}, // defaults to INFO
		{"", INFO},        // defaults to INFO
	}

	for _, tt := range tests {
		if got := ParseLevel(tt.input); got != tt.expected {
			t.Errorf("ParseLevel(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestNewLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("test", INFO, &buf)

	if logger == nil {
		t.Fatal("NewLogger returned nil")
	}
	if logger.component != "test" {
		t.Errorf("component = %v, want %v", logger.component, "test")
	}
	if logger.level != INFO {
		t.Errorf("level = %v, want %v", logger.level, INFO)
	}
	if logger.output != &buf {
		t.Error("output not set correctly")
	}
}

func TestNewLoggerDefaultOutput(t *testing.T) {
	logger := NewLogger("test", INFO, nil)
	if logger.output == nil {
		t.Error("NewLogger with nil output should default to os.Stdout")
	}
}

func TestLogLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("test", INFO, &buf)

	// DEBUG should be filtered out when level is INFO
	logger.Debug("debug message")
	if buf.Len() > 0 {
		t.Error("DEBUG message should be filtered when level is INFO")
	}

	// INFO should be logged
	logger.Info("info message")
	if buf.Len() == 0 {
		t.Error("INFO message should be logged when level is INFO")
	}
}

func TestLogMessageFormatting(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("testcomponent", DEBUG, &buf)

	logger.Info("test message")

	output := buf.String()

	// Check that output contains expected components
	if !strings.Contains(output, "INFO") {
		t.Error("Log output should contain level INFO")
	}
	if !strings.Contains(output, "[testcomponent]") {
		t.Error("Log output should contain component name")
	}
	if !strings.Contains(output, "test message") {
		t.Error("Log output should contain the message")
	}
	// Check timestamp format (should contain date and time)
	if !strings.Contains(output, "[20") { // Year starts with 20
		t.Error("Log output should contain timestamp")
	}
}

func TestLogMethods(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("test", DEBUG, &buf)

	tests := []struct {
		name    string
		logFunc func(string, ...interface{})
		level   string
		message string
	}{
		{"Debug", logger.Debug, "DEBUG", "debug message"},
		{"Info", logger.Info, "INFO", "info message"},
		{"Warn", logger.Warn, "WARN", "warn message"},
		{"Error", logger.Error, "ERROR", "error message"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc(tt.message)

			output := buf.String()
			if !strings.Contains(output, tt.level) {
				t.Errorf("Log output should contain level %s", tt.level)
			}
			if !strings.Contains(output, tt.message) {
				t.Errorf("Log output should contain message %s", tt.message)
			}
		})
	}
}

func TestLogFormatting(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("test", INFO, &buf)

	logger.Info("formatted %s %d", "message", 42)

	output := buf.String()
	if !strings.Contains(output, "formatted message 42") {
		t.Error("Log should support format strings")
	}
}

func TestLogLevelHierarchy(t *testing.T) {
	tests := []struct {
		loggerLevel Level
		logLevel    Level
		shouldLog   bool
	}{
		{DEBUG, DEBUG, true},
		{DEBUG, INFO, true},
		{DEBUG, WARN, true},
		{DEBUG, ERROR, true},
		{INFO, DEBUG, false},
		{INFO, INFO, true},
		{INFO, WARN, true},
		{INFO, ERROR, true},
		{WARN, DEBUG, false},
		{WARN, INFO, false},
		{WARN, WARN, true},
		{WARN, ERROR, true},
		{ERROR, DEBUG, false},
		{ERROR, INFO, false},
		{ERROR, WARN, false},
		{ERROR, ERROR, true},
	}

	for _, tt := range tests {
		var buf bytes.Buffer
		logger := NewLogger("test", tt.loggerLevel, &buf)

		logger.log(tt.logLevel, "test message")

		logged := buf.Len() > 0
		if logged != tt.shouldLog {
			t.Errorf("Logger level %v, log level %v: logged=%v, want %v",
				tt.loggerLevel, tt.logLevel, logged, tt.shouldLog)
		}
	}
}
