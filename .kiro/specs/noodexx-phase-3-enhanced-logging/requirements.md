# Requirements Document

## Introduction

This document specifies requirements for the Enhanced Logging System in Noodexx Phase 3. The system will provide dual-output logging with console output limited to warnings and errors, while comprehensive debug information is written to a file. This enables production-ready console output while maintaining detailed debugging capabilities for troubleshooting.

## Glossary

- **Logger**: The logging component that formats and writes log messages
- **Console_Output**: Log messages written to standard output (stdout) or standard error (stderr)
- **Debug_Log_File**: The file-based log destination for verbose debug information (debug.log)
- **Log_Level**: The severity classification of a log message (DEBUG, INFO, WARN, ERROR)
- **Log_Writer**: The component responsible for writing log messages to a destination
- **Multi_Writer**: A log writer that outputs to multiple destinations simultaneously
- **Config_Manager**: The component that loads and validates application configuration
- **Service**: Any application component that generates log messages (API, LLM providers, ingestion, RAG, skills, database, file watcher)
- **Log_Rotation**: The process of archiving old log files when size or count limits are reached
- **Runtime_Viewer**: A mechanism to view log file contents while the application is running

## Requirements

### Requirement 1: Dual-Output Logging Architecture

**User Story:** As a user, I want the console to show only important messages, so that I can focus on warnings and errors without noise.

#### Acceptance Criteria

1. THE Logger SHALL write WARN and ERROR level messages to Console_Output
2. THE Logger SHALL write DEBUG, INFO, WARN, and ERROR level messages to Debug_Log_File
3. WHEN Debug_Log_File is disabled in configuration, THE Logger SHALL write all messages to Console_Output only
4. THE Multi_Writer SHALL route messages to appropriate destinations based on Log_Level

### Requirement 2: Debug Logging Configuration

**User Story:** As a developer, I want to enable or disable debug file logging, so that I can control logging overhead in different environments.

#### Acceptance Criteria

1. THE Config_Manager SHALL load a debug_enabled boolean field from the logging configuration section
2. WHEN debug_enabled is true, THE Logger SHALL write verbose messages to Debug_Log_File
3. WHEN debug_enabled is false, THE Logger SHALL write only to Console_Output
4. THE Config_Manager SHALL default debug_enabled to true if not specified
5. THE Config_Manager SHALL support NOODEXX_DEBUG_ENABLED environment variable to override the configuration file

### Requirement 3: Debug Log File Location

**User Story:** As a user, I want debug logs in a predictable location, so that I can easily find them when troubleshooting.

#### Acceptance Criteria

1. WHEN Debug_Log_File is enabled, THE Logger SHALL write to a file named debug.log in the application directory
2. THE Config_Manager SHALL support configuring the debug log file path via the logging.file field
3. WHEN logging.file is empty or not specified, THE Logger SHALL default to debug.log in the current directory
4. THE Logger SHALL create the Debug_Log_File if it does not exist
5. IF the Debug_Log_File cannot be created or written, THEN THE Logger SHALL write an error to Console_Output and continue with console-only logging

### Requirement 4: Service Debug Output

**User Story:** As a developer, I want all services to output meaningful debug information, so that I can trace errors and understand system behavior.

#### Acceptance Criteria

1. THE API_Service SHALL log HTTP request details including method, path, and response status at DEBUG level
2. THE LLM_Provider_Service SHALL log provider selection, model names, and request/response metadata at DEBUG level
3. THE Ingestion_Service SHALL log file processing start, progress, and completion at DEBUG level
4. THE RAG_Service SHALL log search queries, result counts, and relevance scores at DEBUG level
5. THE Skills_Service SHALL log skill loading, execution start, and execution completion at DEBUG level
6. THE Database_Service SHALL log query execution and transaction boundaries at DEBUG level
7. THE File_Watcher_Service SHALL log file system events and processing triggers at DEBUG level
8. WHEN an error occurs in any Service, THE Service SHALL log the error with context including operation, input parameters, and stack trace at ERROR level

### Requirement 5: Log Rotation Support

**User Story:** As a user, I want old log files to be archived automatically, so that disk space is managed efficiently.

#### Acceptance Criteria

1. WHEN Debug_Log_File size exceeds logging.max_size_mb megabytes, THE Logger SHALL rotate the log file
2. THE Logger SHALL rename the current Debug_Log_File to debug.log.1 during rotation
3. THE Logger SHALL rename existing backup files incrementally (debug.log.1 becomes debug.log.2)
4. THE Logger SHALL delete the oldest backup file when the count exceeds logging.max_backups
5. THE Logger SHALL create a new empty Debug_Log_File after rotation
6. THE Config_Manager SHALL default max_size_mb to 10 and max_backups to 3 if not specified

### Requirement 6: Runtime Log Viewing

**User Story:** As a developer, I want to view debug logs while the application is running, so that I can monitor system behavior in real-time.

#### Acceptance Criteria

1. THE Logger SHALL open Debug_Log_File with shared read access to allow concurrent readers
2. WHEN a user opens Debug_Log_File with a text editor or tail command, THE Logger SHALL continue writing without errors
3. THE Logger SHALL flush log writes immediately to ensure real-time visibility
4. THE Logger SHALL handle file locking gracefully on Windows and Unix systems

### Requirement 7: Log Message Format

**User Story:** As a developer, I want consistent log formatting, so that I can parse and analyze logs effectively.

#### Acceptance Criteria

1. THE Logger SHALL format each log message with timestamp, level, component name, and message text
2. THE Logger SHALL use ISO 8601 timestamp format (YYYY-MM-DD HH:MM:SS)
3. THE Logger SHALL include the component name in square brackets for each message
4. WHEN a log message contains multiple lines, THE Logger SHALL preserve line breaks in the output
5. THE Logger SHALL escape or sanitize control characters that could corrupt log output

### Requirement 8: Structured Context Logging

**User Story:** As a developer, I want to include structured context in log messages, so that I can correlate related events.

#### Acceptance Criteria

1. THE Logger SHALL support adding key-value context fields to log messages
2. WHEN a Service logs with context fields, THE Logger SHALL append them to the message in a structured format
3. THE Logger SHALL support common context fields including request_id, user_id, file_path, and operation_name
4. THE Logger SHALL format context fields as key=value pairs separated by spaces

### Requirement 9: Performance and Buffering

**User Story:** As a developer, I want logging to have minimal performance impact, so that it doesn't slow down the application.

#### Acceptance Criteria

1. THE Logger SHALL use buffered I/O for Debug_Log_File writes
2. THE Logger SHALL flush the buffer every 5 seconds to balance performance and real-time visibility
3. WHEN the application shuts down, THE Logger SHALL flush all buffered log messages before closing files
4. THE Logger SHALL not block Service operations while writing to Debug_Log_File
5. IF Debug_Log_File writes fail, THEN THE Logger SHALL continue operation and log the failure to Console_Output

### Requirement 10: Backward Compatibility

**User Story:** As a user, I want existing configuration to continue working, so that upgrades don't break my setup.

#### Acceptance Criteria

1. WHEN an existing config.json does not have debug_enabled field, THE Config_Manager SHALL default to true
2. THE Config_Manager SHALL continue supporting the existing logging.level field for Console_Output filtering
3. WHEN logging.file is specified in existing configuration, THE Logger SHALL use it as the Debug_Log_File path
4. THE Logger SHALL maintain the existing log message format for Console_Output
5. THE Config_Manager SHALL validate that logging.level is one of: debug, info, warn, error
