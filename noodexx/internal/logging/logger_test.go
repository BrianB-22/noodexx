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

func TestWithContext(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("test", INFO, &buf)

	// Create a logger with context
	contextLogger := logger.WithContext("request_id", "abc123")

	// Log a message
	contextLogger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "request_id=abc123") {
		t.Errorf("Log output should contain context field: got %q", output)
	}
	if !strings.Contains(output, "test message") {
		t.Errorf("Log output should contain message: got %q", output)
	}
}

func TestWithContextMultiple(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("test", INFO, &buf)

	// Chain multiple context calls
	contextLogger := logger.WithContext("request_id", "abc123").
		WithContext("user_id", "user456")

	contextLogger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "request_id=abc123") {
		t.Errorf("Log output should contain first context field: got %q", output)
	}
	if !strings.Contains(output, "user_id=user456") {
		t.Errorf("Log output should contain second context field: got %q", output)
	}
}

func TestWithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("test", INFO, &buf)

	// Create a logger with multiple fields
	fields := map[string]interface{}{
		"request_id": "abc123",
		"user_id":    "user456",
		"status":     200,
	}
	contextLogger := logger.WithFields(fields)

	contextLogger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "request_id=abc123") {
		t.Errorf("Log output should contain request_id: got %q", output)
	}
	if !strings.Contains(output, "user_id=user456") {
		t.Errorf("Log output should contain user_id: got %q", output)
	}
	if !strings.Contains(output, "status=200") {
		t.Errorf("Log output should contain status: got %q", output)
	}
}

func TestWithContextImmutability(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	logger := NewLogger("test", INFO, &buf1)

	// Create a logger with context
	contextLogger1 := logger.WithContext("key1", "value1")

	// Create another logger from the original
	contextLogger2 := logger.WithContext("key2", "value2")

	// Log with first context logger
	contextLogger1.Info("message1")
	output1 := buf1.String()

	// Reset buffer and update logger output for second test
	buf1.Reset()
	contextLogger2.output = &buf2

	// Log with second context logger
	contextLogger2.Info("message2")
	output2 := buf2.String()

	// Verify first logger only has key1
	if !strings.Contains(output1, "key1=value1") {
		t.Errorf("First logger should contain key1: got %q", output1)
	}
	if strings.Contains(output1, "key2=value2") {
		t.Errorf("First logger should not contain key2: got %q", output1)
	}

	// Verify second logger only has key2
	if !strings.Contains(output2, "key2=value2") {
		t.Errorf("Second logger should contain key2: got %q", output2)
	}
	if strings.Contains(output2, "key1=value1") {
		t.Errorf("Second logger should not contain key1: got %q", output2)
	}
}

func TestWithContextAndFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("test", INFO, &buf)

	// Mix WithContext and WithFields
	contextLogger := logger.WithContext("key1", "value1").
		WithFields(map[string]interface{}{
			"key2": "value2",
			"key3": "value3",
		})

	contextLogger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "key1=value1") {
		t.Errorf("Log output should contain key1: got %q", output)
	}
	if !strings.Contains(output, "key2=value2") {
		t.Errorf("Log output should contain key2: got %q", output)
	}
	if !strings.Contains(output, "key3=value3") {
		t.Errorf("Log output should contain key3: got %q", output)
	}
}

func TestLogWithoutContext(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("test", INFO, &buf)

	// Log without any context
	logger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Log output should contain message: got %q", output)
	}
	// Should not have any extra spaces or context markers
	if strings.Contains(output, "=") {
		t.Errorf("Log output should not contain context markers when no context: got %q", output)
	}
}

func TestContextWithDifferentTypes(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("test", INFO, &buf)

	// Test different value types
	contextLogger := logger.WithFields(map[string]interface{}{
		"string_val": "text",
		"int_val":    42,
		"float_val":  3.14,
		"bool_val":   true,
	})

	contextLogger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "string_val=text") {
		t.Errorf("Log output should contain string value: got %q", output)
	}
	if !strings.Contains(output, "int_val=42") {
		t.Errorf("Log output should contain int value: got %q", output)
	}
	if !strings.Contains(output, "float_val=3.14") {
		t.Errorf("Log output should contain float value: got %q", output)
	}
	if !strings.Contains(output, "bool_val=true") {
		t.Errorf("Log output should contain bool value: got %q", output)
	}
}
