package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Level represents log severity
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger provides structured logging
type Logger struct {
	level     Level
	component string
	output    io.Writer
	context   map[string]interface{}
	formatter *LogFormatter
}

// NewLogger creates a logger for a component
func NewLogger(component string, level Level, output io.Writer) *Logger {
	if output == nil {
		output = os.Stdout
	}
	return &Logger{
		level:     level,
		component: component,
		output:    output,
		formatter: NewLogFormatter(),
	}
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

// WithContext returns a new Logger with an added context field
func (l *Logger) WithContext(key string, value interface{}) *Logger {
	// Create a copy of the logger with merged context
	newContext := make(map[string]interface{})
	for k, v := range l.context {
		newContext[k] = v
	}
	newContext[key] = value

	return &Logger{
		level:     l.level,
		component: l.component,
		output:    l.output,
		context:   newContext,
		formatter: l.formatter,
	}
}

// WithFields returns a new Logger with multiple context fields
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	// Create a copy of the logger with merged context
	newContext := make(map[string]interface{})
	for k, v := range l.context {
		newContext[k] = v
	}
	for k, v := range fields {
		newContext[k] = v
	}

	return &Logger{
		level:     l.level,
		component: l.component,
		output:    l.output,
		context:   newContext,
		formatter: l.formatter,
	}
}

// log writes a log entry
func (l *Logger) log(level Level, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	// Capture caller information
	// Skip 2 frames: log() and the calling method (Debug/Info/Warn/Error)
	_, file, line, ok := runtime.Caller(2)
	if ok {
		// Extract just the filename, not full path
		file = filepath.Base(file)
	} else {
		file = "unknown"
		line = 0
	}

	// Get function name
	pc, _, _, ok := runtime.Caller(2)
	funcName := "unknown"
	if ok {
		fn := runtime.FuncForPC(pc)
		if fn != nil {
			funcName = filepath.Base(fn.Name())
		}
	}

	// Create log entry
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Component: l.component,
		Source: SourceLocation{
			File:     file,
			Line:     line,
			Function: funcName,
		},
		Message: fmt.Sprintf(format, args...),
		Context: l.context,
	}

	// Format and write
	formatted := l.formatter.Format(entry)
	l.output.Write([]byte(formatted))
}

// ParseLevel converts a string to a Level
func ParseLevel(s string) Level {
	switch strings.ToLower(s) {
	case "debug":
		return DEBUG
	case "info":
		return INFO
	case "warn":
		return WARN
	case "error":
		return ERROR
	default:
		return INFO
	}
}
