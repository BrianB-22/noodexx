# Requirements Document

## Introduction

This document specifies requirements for Noodexx Phase 2, which consists of two major components: (1) refactoring the monolithic main.go into a modular package structure, and (2) implementing Phase 2 UI enhancements including sidebar navigation, dashboard, persistent chat history, and improved library features. The refactoring must preserve all existing Phase 1 functionality while establishing a clean architecture for future development. The UI overhaul will modernize the interface with HTMX-based interactions, WebSocket real-time updates, and enhanced user experience features.

## Glossary

- **Noodexx**: The personal AI-powered web application being developed
- **RAG**: Retrieval-Augmented Generation - the pattern of retrieving relevant documents before generating AI responses
- **Store**: The SQLite database abstraction layer for persisting documents, chunks, and chat history
- **Chunk**: A segment of text from an ingested document, stored with its embedding vector
- **Embedding**: A vector representation of text used for semantic search
- **LLM_Provider**: An abstraction for AI services (Ollama, OpenAI, Anthropic, etc.) used for embeddings and chat completions
- **Ollama**: A local LLM service provider option
- **Privacy_Mode**: A system-wide setting that restricts all operations to local-only (no cloud API calls, no external network requests)
- **Folder_Watcher**: A background service that monitors filesystem directories for changes and automatically ingests new/modified files
- **PII**: Personally Identifiable Information (social security numbers, credit cards, API keys, etc.)
- **Audit_Log**: A local record of all system operations (queries, ingestions, deletions) with timestamps
- **Skill**: A user-created executable (script, binary, or program in any language) that extends Noodexx functionality via subprocess execution
- **Skill_Trigger**: The condition that causes a skill to execute (manual, timer, keyword, event)
- **Session**: A conversation context identified by a session ID
- **Ingestion**: The process of parsing, chunking, embedding, and storing documents
- **API_Handler**: HTTP request handlers in the internal/api package
- **WebSocket_Hub**: The component managing WebSocket connections for real-time updates
- **Command_Palette**: A keyboard-accessible (⌘K) navigation interface
- **HTMX**: A library for partial page updates without full page reloads

## Requirements

### Requirement 1: Package Structure Refactoring

**User Story:** As a developer, I want the codebase organized into logical packages, so that I can maintain and extend the application more easily.

#### Acceptance Criteria

1. THE Refactored_Codebase SHALL contain an internal/store package with all SQLite database operations
2. THE Refactored_Codebase SHALL contain an internal/llm package with provider abstractions and implementations for multiple LLM services
3. THE Refactored_Codebase SHALL contain an internal/rag package with chunking, search, and prompt building logic
4. THE Refactored_Codebase SHALL contain an internal/ingest package with document parsing logic
5. THE Refactored_Codebase SHALL contain an internal/api package with all HTTP handlers
6. THE Refactored_Codebase SHALL contain an internal/skills package with skill loading, execution, and management
7. THE main.go file SHALL contain only initialization and package wiring logic
8. THE main.go file SHALL be less than 100 lines of code
9. FOR ALL existing Phase 1 functionality, the refactored code SHALL produce identical behavior to the original implementation

### Requirement 2: Store Package Interface

**User Story:** As a developer, I want a clean database abstraction, so that database operations are isolated and testable.

#### Acceptance Criteria

1. THE Store SHALL provide a SaveChunk method that accepts source, text, embedding vector, optional tags, and optional summary
2. THE Store SHALL provide a Search method that accepts a query vector and topK parameter and returns ranked chunks
3. THE Store SHALL provide a Library method that returns all unique sources with metadata (chunk count, summary, tags, created date)
4. THE Store SHALL provide a DeleteSource method that removes all chunks for a given source
5. THE Store SHALL provide a SaveMessage method that persists chat messages with session ID, role, and content
6. THE Store SHALL provide a GetSessionHistory method that retrieves all messages for a given session ID ordered by creation time
7. THE Store SHALL provide a ListSessions method that returns all unique session IDs with their most recent message timestamp
8. THE Store SHALL provide an AddAuditEntry method that records operations with timestamp, type, and details
9. THE Store SHALL provide a GetAuditLog method that retrieves audit entries with optional filtering by type and date range
10. THE Store SHALL handle database migrations automatically on initialization

### Requirement 3: LLM Provider Abstraction

**User Story:** As a developer, I want a provider-agnostic LLM interface, so that I can use Ollama, OpenAI, Anthropic, or other services without changing application code.

#### Acceptance Criteria

1. THE LLM_Package SHALL define a Provider interface with Embed and Stream methods
2. THE Provider interface Embed method SHALL accept text and return a float32 vector
3. THE Provider interface Stream method SHALL accept messages and an io.Writer and stream the response
4. THE LLM_Package SHALL provide an OllamaProvider implementation of the Provider interface
5. THE LLM_Package SHALL provide an OpenAIProvider implementation of the Provider interface
6. THE LLM_Package SHALL provide an AnthropicProvider implementation of the Provider interface
7. THE LLM_Package SHALL use context.Context for cancellation support across all providers
8. WHEN a provider encounters an API error, THE Provider SHALL return a descriptive error message
9. THE System SHALL select the active provider based on configuration at startup
10. WHEN Privacy_Mode is enabled, THE System SHALL only allow instantiation of OllamaProvider
11. WHEN Privacy_Mode is enabled AND a cloud provider is requested, THE System SHALL return an error indicating privacy mode is active

### Requirement 4: RAG Package Interface

**User Story:** As a developer, I want RAG logic centralized, so that retrieval and prompt construction are reusable.

#### Acceptance Criteria

1. THE RAG_Package SHALL provide a ChunkText function that splits text into overlapping segments
2. THE RAG_Package SHALL provide a Search function that accepts a query string and returns relevant chunks with scores
3. THE RAG_Package SHALL provide a BuildPrompt function that combines user query and retrieved chunks into a prompt
4. THE RAG_Package SHALL use cosine similarity for vector comparison
5. THE ChunkText function SHALL produce chunks between 200 and 500 characters with 50 character overlap

### Requirement 5: Ingestion Package Interface

**User Story:** As a developer, I want document parsing isolated, so that I can add new file formats without modifying core logic.

#### Acceptance Criteria

1. THE Ingest_Package SHALL provide an IngestText function that processes plain text
2. THE Ingest_Package SHALL provide an IngestURL function that fetches and processes web pages
3. THE Ingest_Package SHALL provide an IngestFile function that processes uploaded files based on MIME type
4. THE Ingest_Package SHALL support PDF file parsing
5. THE Ingest_Package SHALL support plain text file parsing
6. WHEN ingestion fails, THE Ingest_Package SHALL return descriptive error messages indicating the failure reason
7. WHEN Privacy_Mode is enabled AND IngestURL is called, THE Ingest_Package SHALL return an error indicating URL ingestion is disabled in privacy mode
8. WHEN Privacy_Mode is disabled, THE Ingest_Package SHALL allow all ingestion methods

### Requirement 6: API Package Structure

**User Story:** As a developer, I want HTTP handlers organized, so that routing and request handling are maintainable.

#### Acceptance Criteria

1. THE API_Package SHALL provide a Server struct that holds dependencies (Store, LLM Provider, RAG engine)
2. THE API_Package SHALL provide handler methods for all existing endpoints (chat, ask, ingest, library, delete)
3. THE API_Package SHALL provide a RegisterRoutes function that sets up all HTTP routes
4. THE API_Package SHALL handle session management for chat history
5. THE API_Package SHALL render HTML templates from the web/templates directory

### Requirement 7: Template Extraction

**User Story:** As a developer, I want HTML templates in separate files, so that I can edit UI without modifying Go code.

#### Acceptance Criteria

1. THE Refactored_Codebase SHALL contain a web/templates/base.html file with the common layout structure
2. THE Refactored_Codebase SHALL contain a web/templates/chat.html file with chat interface markup
3. THE Refactored_Codebase SHALL contain a web/templates/library.html file with library interface markup
4. THE Refactored_Codebase SHALL contain a web/templates/dashboard.html file with dashboard interface markup
5. THE Refactored_Codebase SHALL contain a web/templates/settings.html file with settings interface markup
6. THE API_Handler SHALL load templates from the web/templates directory at startup

### Requirement 8: Sidebar Navigation UI

**User Story:** As a user, I want a fixed sidebar for navigation, so that I can quickly access different sections without scrolling.

#### Acceptance Criteria

1. THE UI SHALL display a fixed left sidebar with navigation links
2. THE Sidebar SHALL contain icons and labels for Dashboard, Chat, Library, and Settings
3. THE Sidebar SHALL highlight the currently active page
4. THE Sidebar SHALL be collapsible via a toggle button
5. WHEN the sidebar is collapsed, THE UI SHALL display only icons without labels
6. THE Sidebar SHALL persist its collapsed state in browser localStorage

### Requirement 9: Dashboard Page

**User Story:** As a user, I want a dashboard overview, so that I can see system status and recent activity at a glance.

#### Acceptance Criteria

1. THE Dashboard SHALL display the total count of indexed documents
2. THE Dashboard SHALL display the active LLM provider name and model
3. THE Dashboard SHALL display the timestamp of the last document ingestion
4. WHEN Privacy_Mode is enabled, THE Dashboard SHALL display a "🔒 Privacy Mode: All data local" indicator
5. WHEN Privacy_Mode is disabled, THE Dashboard SHALL display the current provider (Ollama/OpenAI/Anthropic)
4. THE Dashboard SHALL display a feed of the 10 most recent activities (ingestions, chats, deletions)
5. THE Dashboard SHALL provide a quick access button to start a new chat
6. THE Dashboard SHALL refresh activity data without full page reload using HTMX

### Requirement 10: Persistent Chat History

**User Story:** As a user, I want my chat conversations saved, so that I can review past discussions after restarting the server.

#### Acceptance Criteria

1. WHEN a user sends a message, THE Store SHALL save the message to the chat_messages table
2. WHEN the assistant responds, THE Store SHALL save the response to the chat_messages table
3. WHEN a user loads the chat page, THE UI SHALL display a list of past sessions in the sidebar
4. WHEN a user clicks a past session, THE UI SHALL load that session's message history
5. THE UI SHALL display session timestamps in relative format (e.g., "2 hours ago")
6. THE UI SHALL allow users to start a new session via a "New Chat" button

### Requirement 11: Chat Message Rendering

**User Story:** As a user, I want formatted chat responses, so that code blocks and formatting are readable.

#### Acceptance Criteria

1. WHEN the assistant responds with markdown, THE UI SHALL render it as formatted HTML
2. THE Markdown_Renderer SHALL use the goldmark library for server-side rendering
3. THE UI SHALL apply syntax highlighting to code blocks
4. THE UI SHALL render inline code with monospace font and background color
5. THE UI SHALL render lists, headers, and emphasis correctly

### Requirement 12: Chat Interaction Features

**User Story:** As a user, I want to interact with chat responses, so that I can copy text and regenerate answers.

#### Acceptance Criteria

1. WHEN a user hovers over an assistant message, THE UI SHALL display a copy button
2. WHEN a user clicks the copy button, THE UI SHALL copy the message text to clipboard and show a confirmation toast
3. WHEN a user views the last assistant message, THE UI SHALL display a regenerate button
4. WHEN a user clicks regenerate, THE System SHALL re-submit the previous user query and stream a new response
5. THE UI SHALL replace the previous assistant message with the regenerated response

### Requirement 13: Library Card Grid View

**User Story:** As a user, I want to see my documents in a visual grid, so that I can browse my knowledge base more easily.

#### Acceptance Criteria

1. THE Library_Page SHALL display documents in a responsive card grid layout
2. THE Document_Card SHALL display the source name as a heading
3. THE Document_Card SHALL display a preview of the first chunk's text (truncated to 150 characters)
4. THE Document_Card SHALL display the chunk count for that source
5. THE Document_Card SHALL display any associated tags
6. WHEN a user clicks a document card, THE UI SHALL expand to show all chunks for that source

### Requirement 14: Drag and Drop File Upload

**User Story:** As a user, I want to drag files onto the library page, so that I can ingest documents quickly.

#### Acceptance Criteria

1. WHEN a user drags a file over the library page, THE UI SHALL display a drop zone overlay
2. WHEN a user drops a file, THE System SHALL upload and ingest the file
3. WHEN ingestion completes, THE UI SHALL update the library grid without full page reload
4. WHEN ingestion fails, THE UI SHALL display an error toast with the failure reason
5. THE UI SHALL support dropping multiple files simultaneously

### Requirement 15: Document Tagging System

**User Story:** As a user, I want to tag documents, so that I can organize my knowledge base by topic.

#### Acceptance Criteria

1. THE Store SHALL support storing comma-separated tags for each chunk
2. THE Library_Page SHALL display an "Add Tag" button on each document card
3. WHEN a user clicks "Add Tag", THE UI SHALL display a tag input field
4. WHEN a user submits a tag, THE System SHALL update all chunks for that source with the new tag
5. THE Library_Page SHALL provide a tag filter dropdown that shows all unique tags
6. WHEN a user selects a tag filter, THE UI SHALL display only documents with that tag

### Requirement 16: WebSocket Real-Time Updates

**User Story:** As a user, I want real-time notifications, so that I know when background operations complete.

#### Acceptance Criteria

1. THE System SHALL establish a WebSocket connection when the user loads any page
2. WHEN a document ingestion completes, THE WebSocket_Hub SHALL broadcast an update message
3. WHEN a user receives an update message, THE UI SHALL display a toast notification
4. WHEN a user receives an update message on the library page, THE UI SHALL refresh the document grid
5. THE WebSocket_Hub SHALL handle client disconnections gracefully

### Requirement 17: Toast Notification System

**User Story:** As a user, I want non-intrusive notifications, so that I'm informed of operations without disrupting my workflow.

#### Acceptance Criteria

1. THE UI SHALL display toast notifications in the top-right corner
2. THE Toast SHALL automatically dismiss after 5 seconds
3. THE Toast SHALL support success, error, and info variants with distinct styling
4. THE Toast SHALL be dismissible via a close button
5. WHEN multiple toasts are active, THE UI SHALL stack them vertically

### Requirement 18: Command Palette Navigation

**User Story:** As a user, I want keyboard-driven navigation, so that I can quickly jump to any section without using the mouse.

#### Acceptance Criteria

1. WHEN a user presses ⌘K (or Ctrl+K on Windows/Linux), THE UI SHALL display the command palette overlay
2. THE Command_Palette SHALL list all available navigation commands (Go to Dashboard, Go to Chat, Go to Library, Go to Settings, New Chat) and manual-trigger skills
3. WHEN a user types in the command palette, THE UI SHALL filter commands and skills by fuzzy match
4. WHEN a user selects a command, THE System SHALL navigate to that page and close the palette
5. WHEN a user presses Escape, THE UI SHALL close the command palette

### Requirement 18.1: LLM Provider Settings UI

**User Story:** As a user, I want to configure my LLM provider through the UI, so that I can switch between local and cloud services without editing config files.

#### Acceptance Criteria

1. THE Settings_Page SHALL display a provider selection dropdown with options (Ollama, OpenAI, Anthropic)
2. WHEN a user selects Ollama, THE Settings_Page SHALL display fields for endpoint URL, embedding model, and chat model
3. WHEN a user selects OpenAI, THE Settings_Page SHALL display fields for API key, embedding model (text-embedding-3-small, text-embedding-3-large), and chat model (gpt-4, gpt-3.5-turbo)
4. WHEN a user selects Anthropic, THE Settings_Page SHALL display fields for API key, embedding model (via Voyage AI), and chat model (claude-3-opus, claude-3-sonnet, claude-3-haiku)
5. WHEN a user saves settings, THE System SHALL validate the configuration by making a test API call
6. WHEN validation succeeds, THE System SHALL save the configuration and display a success toast
7. WHEN validation fails, THE System SHALL display an error toast with the failure reason
8. THE Settings_Page SHALL mask API key fields with password input type
9. THE Settings_Page SHALL provide a "Test Connection" button to verify provider settings without saving

### Requirement 18.2: Privacy Mode

**User Story:** As a privacy-conscious user, I want to enable "Privacy Mode" to ensure all my data stays local, so that nothing ever leaves my machine.

#### Acceptance Criteria

1. THE Settings_Page SHALL display a prominent "Privacy Mode" toggle switch at the top
2. WHEN Privacy_Mode is enabled, THE System SHALL restrict the provider selection to Ollama only
3. WHEN Privacy_Mode is enabled, THE System SHALL disable and gray out cloud provider options (OpenAI, Anthropic)
4. WHEN Privacy_Mode is enabled, THE System SHALL block all external network requests except to localhost
5. WHEN Privacy_Mode is enabled AND a user attempts to ingest a URL, THE System SHALL display a warning that URL ingestion requires external network access
6. WHEN Privacy_Mode is enabled, THE Settings_Page SHALL display a badge or indicator showing "🔒 Privacy Mode Active"
7. WHEN Privacy_Mode is disabled, THE System SHALL allow selection of any provider (Ollama, OpenAI, Anthropic)
8. WHEN a user enables Privacy_Mode while using a cloud provider, THE System SHALL prompt to confirm switching to Ollama
9. THE System SHALL persist the Privacy_Mode setting across restarts
10. WHEN Privacy_Mode is enabled, THE Dashboard SHALL display a privacy indicator showing "All data local"

### Requirement 19: Database Schema Migration

**User Story:** As a developer, I want automatic schema updates, so that existing databases are upgraded without manual intervention.

#### Acceptance Criteria

1. THE Store SHALL create a chat_messages table if it does not exist
2. THE Store SHALL add a tags column to the chunks table if it does not exist
3. THE Store SHALL add a summary column to the chunks table if it does not exist
4. THE Store SHALL create an audit_log table if it does not exist with columns: id, timestamp, operation_type, details, user_context
5. THE Store SHALL create a watched_folders table if it does not exist with columns: id, path, active, last_scan
6. THE Store SHALL preserve all existing data during migration
7. THE Store SHALL execute migrations in a transaction to ensure atomicity
8. WHEN migration fails, THE Store SHALL return a descriptive error and not start the application

### Requirement 20: HTMX Integration

**User Story:** As a user, I want smooth page updates, so that interactions feel responsive without full page reloads.

#### Acceptance Criteria

1. THE UI SHALL include the HTMX library in web/static/htmx.min.js
2. THE Library_Page SHALL use HTMX for document deletion without page reload
3. THE Dashboard SHALL use HTMX for activity feed updates
4. THE Chat_Page SHALL use HTMX for loading past sessions
5. THE API_Handler SHALL return HTML fragments for HTMX requests instead of full pages

### Requirement 21: CSS Transitions and Styling

**User Story:** As a user, I want smooth visual transitions, so that the interface feels polished and professional.

#### Acceptance Criteria

1. THE UI SHALL apply CSS transitions to sidebar collapse/expand with 200ms duration
2. THE UI SHALL apply CSS transitions to toast notifications with fade-in and fade-out effects
3. THE UI SHALL apply hover effects to interactive elements (buttons, cards, links)
4. THE UI SHALL use a consistent color scheme across all pages
5. THE UI SHALL be responsive and usable on tablet and desktop screen sizes

### Requirement 22: Dependency Management

**User Story:** As a developer, I want new dependencies properly managed, so that the project builds reliably.

#### Acceptance Criteria

1. THE go.mod file SHALL include github.com/yuin/goldmark for markdown rendering
2. THE go.mod file SHALL include github.com/gorilla/websocket for WebSocket support
3. THE go.mod file SHALL include github.com/fsnotify/fsnotify for folder watching
4. THE go.mod file SHALL maintain existing dependencies (modernc.org/sqlite, github.com/go-shiori/go-readability)
5. THE Project SHALL build successfully with `go build` after adding new dependencies

### Requirement 23: Backward Compatibility

**User Story:** As a user, I want my existing data preserved, so that I don't lose my Phase 1 documents and embeddings.

#### Acceptance Criteria

1. THE Refactored_System SHALL read existing chunks from the Phase 1 database schema
2. THE Refactored_System SHALL preserve all existing embedding vectors
3. THE Refactored_System SHALL maintain compatibility with the existing chunks table structure
4. THE Refactored_System SHALL not require users to re-ingest documents after upgrade

### Requirement 24: Error Handling and Logging

**User Story:** As a developer, I want consistent error handling, so that I can diagnose issues quickly.

#### Acceptance Criteria

1. WHEN a package function encounters an error, THE Function SHALL return a descriptive error message
2. THE API_Handler SHALL log all HTTP requests with method, path, and status code
3. THE Store SHALL log all database errors with query context
4. THE LLM_Provider SHALL log all API errors with provider name and request details
5. THE System SHALL not expose internal error details to users in HTTP responses

### Requirement 25: Configuration Management

**User Story:** As a user, I want configurable settings in a simple JSON file, so that I can easily customize Noodexx without editing code.

#### Acceptance Criteria

1. THE System SHALL read configuration from a config.json file in the application directory
2. THE System SHALL support environment variable overrides for all config values
3. THE config.json file SHALL include sections for: provider, privacy, folders, logging, guardrails, and server
4. THE System SHALL support configuring the LLM provider type (ollama, openai, anthropic)
5. THE System SHALL support configuring provider-specific settings (API keys, endpoints, model names)
6. THE System SHALL support configuring the embedding model name per provider
7. THE System SHALL support configuring the chat model name per provider
8. THE System SHALL support configuring Privacy_Mode as a boolean flag
9. THE System SHALL support configuring watched folder paths as an array
10. THE System SHALL support configuring logging level (debug, info, warn, error)
11. THE System SHALL support configuring file size limits for ingestion
12. THE System SHALL support configuring the HTTP server port and bind address
13. WHEN using OpenAI provider, THE System SHALL require an API key via config or environment variable
14. WHEN using Anthropic provider, THE System SHALL require an API key via config or environment variable
15. WHEN using Ollama provider, THE System SHALL use localhost:11434 as the default endpoint
16. WHEN Privacy_Mode is enabled, THE System SHALL ignore any cloud provider configuration and force Ollama
17. WHEN Privacy_Mode is enabled, THE System SHALL validate that the Ollama endpoint is localhost or 127.0.0.1
18. WHEN configuration is missing, THE System SHALL create a default config.json with sensible defaults (Ollama provider, Privacy_Mode enabled, port 8080, bind to 127.0.0.1)
19. THE System SHALL encrypt API keys in the config file using a local encryption key
20. WHEN config.json is malformed, THE System SHALL fail with a descriptive error message indicating the parsing issue

### Requirement 26: Structured Logging System

**User Story:** As a developer, I want configurable logging with levels, so that I can troubleshoot issues without being overwhelmed by debug output.

#### Acceptance Criteria

1. THE System SHALL implement a structured logging system with levels: DEBUG, INFO, WARN, ERROR
2. THE Logging_System SHALL write logs to stdout in a structured format (timestamp, level, component, message)
3. THE Logging_System SHALL respect the configured log level from config.json
4. WHEN log level is DEBUG, THE System SHALL log all messages including detailed API calls and database queries
5. WHEN log level is INFO, THE System SHALL log normal operations (ingestions, queries, server start)
6. WHEN log level is WARN, THE System SHALL log warnings (PII detected, file size exceeded, rate limits)
7. WHEN log level is ERROR, THE System SHALL log errors (API failures, database errors, ingestion failures)
8. THE Logging_System SHALL include the component name in each log entry (store, llm, ingest, api, watcher)
9. THE Logging_System SHALL optionally write logs to a file when configured
10. THE Logging_System SHALL rotate log files when they exceed a configured size

### Requirement 27: Folder Watching System

**User Story:** As a user, I want Noodexx to automatically ingest files I add to watched folders, so that my knowledge base stays in sync without manual uploads.

#### Acceptance Criteria

1. THE System SHALL use fsnotify to watch configured folder paths for filesystem events
2. WHEN a new file is created in a watched folder, THE System SHALL automatically ingest it
3. WHEN a file is modified in a watched folder, THE System SHALL re-ingest it and update existing chunks
4. WHEN a file is deleted in a watched folder, THE System SHALL remove its chunks from the database
5. THE Folder_Watcher SHALL only process files with allowed extensions (.txt, .md, .pdf)
6. THE Folder_Watcher SHALL skip files larger than the configured size limit (default 10MB)
7. THE Folder_Watcher SHALL not follow symlinks outside the watched directory
8. THE Folder_Watcher SHALL validate that watched paths do not include system directories (/etc, /System, C:\Windows)
9. THE Folder_Watcher SHALL process files in a queue with configurable concurrency (default 3 concurrent ingestions)
10. WHEN Privacy_Mode is enabled, THE Folder_Watcher SHALL only watch local filesystem paths
11. THE Folder_Watcher SHALL log all file events (created, modified, deleted) at INFO level
12. THE Settings_Page SHALL allow users to add and remove watched folders via the UI

### Requirement 28: PII Detection and Warnings

**User Story:** As a privacy-conscious user, I want warnings before ingesting sensitive data, so that I don't accidentally embed credentials or personal information.

#### Acceptance Criteria

1. THE Ingest_Package SHALL scan text for PII patterns before embedding
2. THE PII_Detector SHALL detect social security numbers (XXX-XX-XXXX pattern)
3. THE PII_Detector SHALL detect credit card numbers (16-digit patterns)
4. THE PII_Detector SHALL detect API keys and tokens (common patterns like sk-*, ghp_*, etc.)
5. THE PII_Detector SHALL detect private key files (BEGIN PRIVATE KEY, BEGIN RSA PRIVATE KEY)
6. THE PII_Detector SHALL detect email addresses and phone numbers
7. WHEN PII is detected, THE System SHALL log a warning with the PII type and file name
8. WHEN PII is detected during manual ingestion, THE UI SHALL display a confirmation dialog listing detected PII types
9. WHEN PII is detected during folder watching, THE System SHALL skip the file and log a warning
10. THE Settings_Page SHALL allow users to configure PII detection sensitivity (strict, normal, off)
11. THE PII_Detector SHALL not log the actual PII values, only the types detected

### Requirement 29: Auto-Summarization on Ingest

**User Story:** As a user, I want automatic summaries of ingested documents, so that I can quickly scan my library without reading full texts.

#### Acceptance Criteria

1. WHEN a document is ingested, THE System SHALL generate a 2-3 sentence summary using the LLM
2. THE Store SHALL save the summary as metadata associated with the source
3. THE Library_Page SHALL display the summary on each document card instead of raw chunk preview
4. THE Summary_Generator SHALL use the first 1000 characters of the document as input to the LLM
5. THE Summary_Generator SHALL use a prompt template: "Summarize this document in 2-3 sentences"
6. WHEN summarization fails, THE System SHALL fall back to displaying the first 150 characters of text
7. THE Summary_Generator SHALL respect the configured LLM provider (Ollama or cloud)
8. THE Settings_Page SHALL allow users to enable/disable auto-summarization

### Requirement 30: Audit Logging

**User Story:** As a user, I want a local record of all Noodexx operations, so that I can review what data has been accessed and when.

#### Acceptance Criteria

1. THE System SHALL maintain an audit log table in the database
2. THE Audit_Log SHALL record all document ingestions with timestamp, source, and file size
3. THE Audit_Log SHALL record all chat queries with timestamp, session ID, and query text
4. THE Audit_Log SHALL record all document deletions with timestamp and source name
5. THE Audit_Log SHALL record all email accesses (when Phase 4 is implemented) with timestamp and email subject
6. THE Audit_Log SHALL record all configuration changes with timestamp and changed settings
7. THE Settings_Page SHALL provide an "Audit Log" view showing recent operations
8. THE Audit_Log_View SHALL allow filtering by operation type (ingest, query, delete, config)
9. THE Audit_Log_View SHALL allow filtering by date range
10. THE System SHALL optionally auto-purge audit log entries older than a configured retention period (default 90 days)

### Requirement 31: Ingestion Guardrails

**User Story:** As a user, I want protection against accidentally ingesting problematic files, so that Noodexx remains stable and secure.

#### Acceptance Criteria

1. THE Ingest_Package SHALL enforce a maximum file size limit (configurable, default 10MB)
2. WHEN a file exceeds the size limit, THE System SHALL skip it and log a warning
3. THE Ingest_Package SHALL maintain an allowlist of safe file extensions (.txt, .md, .pdf, .html)
4. WHEN a file has a disallowed extension, THE System SHALL skip it and log a warning
5. THE Ingest_Package SHALL reject executable files (.exe, .dll, .so, .dylib, .app)
6. THE Ingest_Package SHALL reject archive files (.zip, .tar, .gz, .rar)
7. THE Ingest_Package SHALL reject disk images (.iso, .dmg, .img)
8. THE Ingest_Package SHALL detect sensitive filenames (.env, id_rsa, credentials.json, .aws/credentials)
9. WHEN a sensitive filename is detected, THE System SHALL skip it and log a warning
10. THE Ingest_Package SHALL rate limit concurrent ingestions (configurable, default 3 concurrent)
11. WHEN the ingestion queue exceeds 100 files, THE System SHALL log a warning and process them in batches

### Requirement 32: Server Security Defaults

**User Story:** As a security-conscious user, I want Noodexx to bind to localhost by default, so that my data isn't exposed to my network.

#### Acceptance Criteria

1. THE HTTP_Server SHALL bind to 127.0.0.1 by default, not 0.0.0.0
2. THE Settings_Page SHALL display a warning when changing the bind address to 0.0.0.0
3. THE config.json SHALL include a bind_address field with 127.0.0.1 as the default
4. WHEN bind_address is set to 0.0.0.0, THE System SHALL log a warning on startup
5. THE System SHALL validate that the configured port is not a privileged port (<1024) unless running as root

### Requirement 33: Skill System (Plugin Architecture)

**User Story:** As a power user, I want to extend Noodexx with custom scripts and programs, so that I can add functionality without modifying the core codebase.

#### Acceptance Criteria

1. THE System SHALL support loading skills from a skills/ directory in the application folder
2. THE Skill SHALL be any executable file (shell script, Python script, compiled binary, etc.)
3. THE Skill SHALL include a skill.json metadata file in its directory defining name, description, triggers, and settings schema
4. THE Skill_System SHALL execute skills as subprocesses with controlled environment variables
5. THE Skill_System SHALL communicate with skills via stdin/stdout using JSON messages
6. THE Skill SHALL receive input as a JSON object on stdin with fields: query, context, settings
7. THE Skill SHALL return output as a JSON object on stdout with fields: result, error, metadata
8. THE Skill_System SHALL support trigger types: manual, timer, keyword, event
9. WHEN trigger type is "manual", THE Skill SHALL appear in the command palette and be callable by name
10. WHEN trigger type is "timer", THE Skill SHALL execute on a cron-like schedule defined in skill.json
11. WHEN trigger type is "keyword", THE Skill SHALL execute when a chat message contains specified keywords
12. WHEN trigger type is "event", THE Skill SHALL execute when system events occur (ingest_complete, alert_created)
13. THE Skill_System SHALL enforce a timeout (configurable per skill, default 30 seconds)
14. WHEN a skill exceeds its timeout, THE System SHALL terminate the subprocess and log an error
15. THE Skill_System SHALL respect Privacy_Mode by controlling network access via environment variables
16. WHEN Privacy_Mode is enabled, THE Skill SHALL receive NOODEXX_PRIVACY_MODE=true environment variable
17. THE Skill_System SHALL pass skill-specific settings from config.json as environment variables
18. THE Settings_Page SHALL display installed skills with enable/disable toggles
19. THE Settings_Page SHALL allow configuring skill-specific settings via a dynamic form based on skill.json schema
20. THE Skill_System SHALL log all skill executions to the audit log with skill name, trigger, duration, and exit code
21. THE Skill_System SHALL sandbox skills by limiting environment variables to: NOODEXX_*, PATH, HOME, USER
22. THE Skill_System SHALL provide a skill template generator via CLI: `noodexx skill create my-skill`

### Requirement 33.1: Skill Metadata Format

**User Story:** As a skill developer, I want a simple metadata format, so that I can define my skill's behavior declaratively.

#### Acceptance Criteria

1. THE skill.json file SHALL include required fields: name, version, description, executable
2. THE skill.json file SHALL include optional fields: triggers, settings_schema, timeout, requires_network
3. THE triggers field SHALL be an array of trigger objects with type and parameters
4. THE settings_schema field SHALL define configurable settings using JSON Schema format
5. THE executable field SHALL specify the relative path to the executable file within the skill directory
6. WHEN requires_network is true AND Privacy_Mode is enabled, THE System SHALL refuse to load the skill
7. THE Skill_System SHALL validate skill.json against a schema on load and reject invalid skills

### Requirement 33.2: Built-in Example Skills

**User Story:** As a new user, I want example skills included, so that I can learn how to create my own.

#### Acceptance Criteria

1. THE System SHALL include an example skill "weather" that fetches weather via wttr.in
2. THE System SHALL include an example skill "summarize-url" that fetches and summarizes a URL
3. THE System SHALL include an example skill "daily-digest" that runs on a timer and generates a summary
4. THE Example_Skills SHALL be located in skills/examples/ directory
5. THE Example_Skills SHALL include well-commented code and comprehensive skill.json files
6. THE Settings_Page SHALL display example skills with a badge indicating they are examples
