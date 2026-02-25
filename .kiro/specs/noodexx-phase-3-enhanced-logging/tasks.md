# Implementation Plan: Enhanced Logging System

## Overview

This plan implements a dual-output logging system for Noodexx that routes WARN/ERROR messages to console while writing all log levels to a debug file. The implementation adds source location tracking, structured context fields, buffered file I/O, automatic log rotation, and comprehensive debug output across all services.

The implementation builds on the existing logger in `noodexx/internal/logging/logger.go` and extends the configuration in `noodexx/internal/config/config.go`.

## Tasks

- [x] 1. Extend configuration for enhanced logging
  - Add `debug_enabled` boolean field to LoggingConfig struct
  - Set default values: debug_enabled=true, file="debug.log", max_size_mb=10, max_backups=3
  - Add NOODEXX_DEBUG_ENABLED environment variable override in applyEnvOverrides()
  - Update config validation to handle new fields
  - _Requirements: 2.1, 2.4, 2.5, 3.2, 3.3, 5.6, 10.1, 10.2, 10.3, 10.5_

- [ ]* 1.1 Write property test for configuration round-trip
  - **Property 2: Configuration round-trip for debug_enabled**
  - **Validates: Requirements 2.1**

- [x] 2. Implement log formatter with source location capture
  - Create LogEntry struct with timestamp, level, component, source location, message, and context fields
  - Create SourceLocation struct with file, line, and function name
  - Implement LogFormatter with Format() method
  - Use runtime.Caller() to capture file, line, and function name (skip 2 frames)
  - Format output as: `[YYYY-MM-DD HH:MM:SS] LEVEL [component] file.go:line function message key=value`
  - Sanitize control characters (except \n and \t) to prevent log injection
  - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5_

- [ ]* 2.1 Write property test for log format completeness
  - **Property 10: Log message format completeness**
  - **Validates: Requirements 7.1, 7.2, 7.3**

- [ ]* 2.2 Write property test for multi-line preservation
  - **Property 11: Multi-line message preservation**
  - **Validates: Requirements 7.4**

- [ ]* 2.3 Write property test for control character sanitization
  - **Property 12: Control character sanitization**
  - **Validates: Requirements 7.5**

- [x] 3. Implement log rotator for automatic file rotation
  - Create LogRotator struct with basePath, maxSizeMB, and maxBackups fields
  - Implement ShouldRotate() to check if file size exceeds threshold
  - Implement Rotate() to rename current file to .1, increment existing backups, delete oldest
  - Handle rotation errors gracefully (log to console, continue with current file)
  - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5_

- [ ]* 3.1 Write property test for log rotation on size threshold
  - **Property 7: Log rotation on size threshold**
  - **Validates: Requirements 5.1, 5.2, 5.5**

- [ ]* 3.2 Write property test for backup file incremental naming
  - **Property 8: Backup file incremental naming**
  - **Validates: Requirements 5.3**

- [ ]* 3.3 Write property test for backup file cleanup
  - **Property 9: Backup file cleanup**
  - **Validates: Requirements 5.4**

- [x] 4. Implement file writer with buffering and shared access
  - Create FileWriter struct with path, file handle, bufio.Writer, rotator, mutex, and flush timer
  - Open file with os.O_CREATE|os.O_WRONLY|os.O_APPEND and 0644 permissions
  - Use 64KB buffer with bufio.Writer
  - Implement Write() to append to buffer
  - Implement Flush() to write buffer to disk and check for rotation
  - Set up 5-second flush timer
  - Implement Close() to flush and close file
  - Handle file creation/write errors gracefully (log to console, fall back to console-only)
  - _Requirements: 3.4, 3.5, 6.1, 6.2, 6.3, 6.4, 9.1, 9.2, 9.3, 9.4, 9.5_

- [ ]* 4.1 Write property test for file creation with graceful degradation
  - **Property 4: File creation with graceful degradation**
  - **Validates: Requirements 3.4, 3.5**

- [ ]* 4.2 Write property test for custom log file path usage
  - **Property 3: Custom log file path usage**
  - **Validates: Requirements 3.2**

- [x] 5. Implement multi-writer for dual-output routing
  - Create MultiWriter struct with consoleWriter, fileWriter, and debugEnabled fields
  - Implement Write() method that routes based on log level
  - Route WARN/ERROR to console (always)
  - Route all levels to file (when debugEnabled=true)
  - Route all levels to console only (when debugEnabled=false)
  - _Requirements: 1.1, 1.2, 1.3, 1.4_

- [ ]* 5.1 Write property test for level-based message routing
  - **Property 1: Level-based message routing**
  - **Validates: Requirements 1.1, 1.2, 1.4**

- [x] 6. Checkpoint - Ensure core logging components work
  - Ensure all tests pass, ask the user if questions arise.

- [x] 7. Add structured context support to Logger
  - Add context map[string]interface{} field to Logger struct
  - Implement WithContext(key, value) method that returns new Logger with added context
  - Implement WithFields(fields) method that returns new Logger with multiple context fields
  - Update log() method to include context fields in formatted output
  - Format context as space-separated key=value pairs
  - _Requirements: 8.1, 8.2, 8.3, 8.4_

- [ ]* 7.1 Write property test for context field formatting
  - **Property 13: Context field formatting**
  - **Validates: Requirements 8.1, 8.2, 8.4**

- [x] 8. Update Logger to use enhanced components
  - Update Logger struct to use new formatter and support source location
  - Modify log() method to capture caller info with runtime.Caller(2)
  - Update log() method to create LogEntry and format with LogFormatter
  - Update NewLogger() to accept io.Writer for flexibility
  - Maintain backward compatibility with existing log format for console output
  - _Requirements: 7.1, 7.2, 7.3, 10.4_

- [x] 9. Update main.go to initialize enhanced logging
  - Create initializeLogging() function
  - Check if cfg.Logging.DebugEnabled is true
  - If true, create FileWriter with cfg.Logging.File, MaxSizeMB, MaxBackups
  - If FileWriter creation fails, log error and fall back to console-only
  - Create MultiWriter with console and file writers
  - If false, use console writer only
  - Parse log level and create Logger with appropriate writer
  - _Requirements: 1.3, 2.2, 2.3, 3.1, 3.2, 3.5_

- [x] 10. Add debug logging to API service
  - Update handlers.go to add request logging with context fields
  - Log HTTP request details: method, path, request_id at DEBUG level
  - Log response status and latency at DEBUG level
  - Log errors with context: operation, parameters, error message at ERROR level
  - _Requirements: 4.1, 4.8_

- [ ]* 10.1 Write property test for service debug output completeness
  - **Property 5: Service debug output completeness**
  - **Validates: Requirements 4.1**

- [ ]* 10.2 Write property test for error logging with context
  - **Property 6: Error logging with context**
  - **Validates: Requirements 4.8**

- [x] 11. Add debug logging to LLM provider services
  - Update openai.go, ollama.go, anthropic.go to add provider logging
  - Log provider selection, model names at DEBUG level
  - Log request/response metadata: tokens, latency_ms at DEBUG level
  - Log errors with context: provider, model, error message at ERROR level
  - _Requirements: 4.2, 4.8_

- [x] 12. Add debug logging to ingestion service
  - Update ingestion code to add file processing logging
  - Log file processing start with file_path, file_size at DEBUG level
  - Log progress with chunk_index, total_chunks at DEBUG level
  - Log completion with total chunks processed at DEBUG level
  - Log errors with context: file_path, operation, error message at ERROR level
  - _Requirements: 4.3, 4.8_

- [x] 13. Add debug logging to RAG service
  - Update RAG search code to add query logging
  - Log search queries with query, limit at DEBUG level
  - Log result counts and relevance scores at DEBUG level
  - Log errors with context: query, operation, error message at ERROR level
  - _Requirements: 4.4, 4.8_

- [x] 14. Add debug logging to skills service
  - Update skills execution code to add skill logging
  - Log skill loading with skill_name, skill_path at DEBUG level
  - Log execution start and completion with exit_code at DEBUG level
  - Log errors with context: skill_name, operation, error message at ERROR level
  - _Requirements: 4.5, 4.8_

- [x] 15. Add debug logging to database service
  - Update store.go to add database operation logging
  - Log query execution with operation, table at DEBUG level
  - Log transaction boundaries at DEBUG level
  - Log errors with context: operation, table, error message at ERROR level
  - _Requirements: 4.6, 4.8_

- [x] 16. Add debug logging to file watcher service
  - Update file watcher code to add event logging
  - Log file system events with file_path, event type at DEBUG level
  - Log processing triggers at DEBUG level
  - Log errors with context: file_path, operation, error message at ERROR level
  - _Requirements: 4.7, 4.8_

- [x] 17. Final checkpoint - Verify all services produce debug output
  - Ensure all tests pass, ask the user if questions arise.

- [ ]* 18. Write property test for configuration validation
  - **Property 14: Configuration validation for log level**
  - **Validates: Requirements 10.5**

## Notes

- Tasks marked with `*` are optional property-based tests that can be skipped
- Each task references specific requirements for traceability
- The implementation maintains backward compatibility with existing configuration
- File logging degrades gracefully to console-only on errors
- All services will produce structured debug output with context fields
- Log rotation prevents unbounded disk usage (default: 40MB max)
