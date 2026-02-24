# Implementation Plan: Noodexx Phase 2 - Refactor and UI Enhancement

## Overview

This implementation plan transforms the monolithic Noodexx Phase 1 codebase into a modular, maintainable system with enhanced UI capabilities. The refactoring establishes clean package boundaries following Go best practices, while the UI overhaul introduces HTMX-based interactions, WebSocket real-time updates, persistent chat history, and a comprehensive dashboard interface. All Phase 1 functionality is preserved while establishing patterns for future extensibility.

## Implementation Approach

- Incremental refactoring: Extract packages one at a time while maintaining working state
- Test-driven: Write tests alongside implementation to validate correctness
- UI-first for new features: Build templates and static assets before backend handlers
- Database migrations: Ensure backward compatibility with Phase 1 data
- Configuration-driven: All settings externalized to config.json

## Tasks

### Phase 1: Foundation and Package Structure

- [ ] 1. Set up package structure and configuration system
  - Create internal/ directory structure with subdirectories: store, llm, rag, ingest, api, skills, config, logging, watcher
  - Create web/ directory structure with subdirectories: static, templates
  - Create skills/examples/ directory for example skills
  - _Requirements: 1.1-1.7, 7.1-7.6_

  - [x] 1.1 Create configuration package
    - Implement internal/config/config.go with Config struct and all sub-structs (ProviderConfig, PrivacyConfig, LoggingConfig, GuardrailsConfig, ServerConfig)
    - Implement Load() function with JSON parsing and environment variable overrides
    - Implement Save() function for writing config to file
    - Implement Validate() function with privacy mode and provider validation
    - Implement applyEnvOverrides() for environment variable support
    - Create default config.json with sensible defaults (Ollama provider, privacy mode enabled, localhost binding)
    - _Requirements: 25.1-25.20_

  - [ ]* 1.2 Write unit tests for configuration package
    - Test default config creation
    - Test environment variable overrides (NOODEXX_PROVIDER, NOODEXX_OPENAI_KEY, NOODEXX_PRIVACY_MODE)
    - Test privacy mode validation (rejects cloud providers)
    - Test malformed JSON handling
    - Test provider-specific validation (API keys required for cloud providers)
    - _Requirements: 25.19, 25.20_

  - [x] 1.3 Create logging package
    - Implement internal/logging/logger.go with Logger struct
    - Implement Level enum (DEBUG, INFO, WARN, ERROR) with String() method
    - Implement NewLogger() constructor
    - Implement Debug(), Info(), Warn(), Error() methods
    - Implement log() method with timestamp and component formatting
    - Implement ParseLevel() function
    - _Requirements: 26.1-26.10_


  - [ ]* 1.4 Write unit tests for logging package
    - Test log level filtering (DEBUG logs not shown when level is INFO)
    - Test log message formatting (timestamp, level, component, message)
    - Test ParseLevel() with various inputs
    - _Requirements: 26.2-26.7_

### Phase 2: Store Package Implementation

- [ ] 2. Implement database abstraction layer
  - [x] 2.1 Create store package with core interfaces and models
    - Implement internal/store/models.go with Chunk, LibraryEntry, ChatMessage, Session, AuditEntry, WatchedFolder structs
    - Implement internal/store/store.go with Store struct and NewStore() constructor
    - Implement Close() method
    - Add database connection handling with modernc.org/sqlite
    - _Requirements: 2.1-2.10_

  - [x] 2.2 Implement database migrations
    - Create internal/store/migrations.go with migration functions
    - Implement chunks table creation (with tags and summary columns)
    - Implement chat_messages table creation
    - Implement audit_log table creation
    - Implement watched_folders table creation
    - Implement migration runner that executes in transaction
    - Add indexes for performance (source, created_at, session_id, timestamp)
    - _Requirements: 19.1-19.8, 23.1-23.4_

  - [ ]* 2.3 Write property test for database migrations
    - **Property 8: Database Migration Preserves Data**
    - **Validates: Requirements 2.10, 23.1-23.4**
    - Create database with Phase 1 schema and sample chunks
    - Run migrations
    - Verify all existing chunks preserved with same source, text, embeddings
    - _Requirements: 19.7, 23.1-23.4_

  - [x] 2.4 Implement chunk operations
    - Implement SaveChunk() method with embedding serialization
    - Implement Search() method with cosine similarity calculation
    - Implement Library() method with GROUP BY source aggregation
    - Implement DeleteSource() method
    - _Requirements: 2.1-2.4_

  - [ ]* 2.5 Write property tests for chunk operations
    - **Property 1: Chunk Save and Retrieve Round Trip**
    - **Validates: Requirements 2.1, 2.2**
    - **Property 2: Search Returns Correct Count**
    - **Validates: Requirements 2.2**
    - **Property 3: Library Returns Unique Sources**
    - **Validates: Requirements 2.3**
    - **Property 4: Delete Removes All Chunks**
    - **Validates: Requirements 2.4**
    - _Requirements: 2.1-2.4_


  - [x] 2.6 Implement chat history operations
    - Implement SaveMessage() method
    - Implement GetSessionHistory() method with ORDER BY created_at
    - Implement ListSessions() method with GROUP BY session_id
    - _Requirements: 2.5-2.7_

  - [ ]* 2.7 Write property tests for chat history
    - **Property 5: Chat Message Chronological Order**
    - **Validates: Requirements 2.6**
    - **Property 6: Session List Uniqueness**
    - **Validates: Requirements 2.7**
    - _Requirements: 2.6-2.7_

  - [x] 2.8 Implement audit log operations
    - Implement AddAuditEntry() method
    - Implement GetAuditLog() method with filtering by type and date range
    - _Requirements: 2.8-2.9, 30.1-30.10_

  - [ ]* 2.9 Write property test for audit log
    - **Property 7: Audit Log Filtering**
    - **Validates: Requirements 2.9**
    - _Requirements: 2.9_

  - [x] 2.10 Implement watched folder operations
    - Implement AddWatchedFolder() method
    - Implement GetWatchedFolders() method
    - Implement RemoveWatchedFolder() method
    - _Requirements: 27.1-27.12_

### Phase 3: LLM Package Implementation

- [ ] 3. Implement LLM provider abstraction
  - [x] 3.1 Create provider interface and factory
    - Implement internal/llm/provider.go with Provider interface (Embed, Stream, Name, IsLocal methods)
    - Implement Message struct
    - Implement Config struct with all provider settings
    - Implement NewProvider() factory function with privacy mode enforcement
    - _Requirements: 3.1-3.11_

  - [ ]* 3.2 Write unit test for privacy mode enforcement
    - **Property 13: Privacy Mode Blocks Cloud Providers**
    - **Validates: Requirements 3.10, 3.11**
    - Test NewProvider() with privacy mode enabled and OpenAI/Anthropic types returns error
    - Test NewProvider() with privacy mode enabled and Ollama type succeeds
    - _Requirements: 3.10-3.11_

  - [x] 3.3 Implement Ollama provider
    - Create internal/llm/ollama.go with OllamaProvider struct
    - Implement NewOllamaProvider() constructor
    - Implement Embed() method with HTTP POST to /api/embeddings
    - Implement Stream() method with HTTP POST to /api/chat and streaming response parsing
    - Implement Name() and IsLocal() methods
    - Add error handling with descriptive messages
    - _Requirements: 3.4, 3.8_


  - [ ]* 3.4 Write property tests for Ollama provider
    - **Property 9: Embedding Returns Fixed Dimension Vector**
    - **Validates: Requirements 3.2**
    - **Property 10: Stream Writes to Writer**
    - **Validates: Requirements 3.3**
    - **Property 11: Context Cancellation Stops Provider**
    - **Validates: Requirements 3.7**
    - **Property 12: Provider Errors Are Descriptive**
    - **Validates: Requirements 3.8**
    - _Requirements: 3.2, 3.3, 3.7, 3.8_

  - [x] 3.5 Implement OpenAI provider
    - Create internal/llm/openai.go with OpenAIProvider struct
    - Implement NewOpenAIProvider() constructor
    - Implement Embed() method with POST to /v1/embeddings
    - Implement Stream() method with POST to /v1/chat/completions and SSE parsing
    - Implement Name() and IsLocal() methods
    - Add Authorization header with Bearer token
    - _Requirements: 3.5, 3.8_

  - [ ]* 3.6 Write unit tests for OpenAI provider
    - Test API key requirement
    - Test error handling for API failures
    - Test streaming response parsing
    - _Requirements: 3.5, 3.8_

  - [x] 3.7 Implement Anthropic provider
    - Create internal/llm/anthropic.go with AnthropicProvider struct
    - Implement NewAnthropicProvider() constructor
    - Implement Embed() method (placeholder with error for Voyage AI)
    - Implement Stream() method with POST to /v1/messages and SSE parsing
    - Implement Name() and IsLocal() methods
    - Add x-api-key header and anthropic-version header
    - Handle system message separation in Anthropic format
    - _Requirements: 3.6, 3.8_

  - [ ]* 3.8 Write unit tests for Anthropic provider
    - Test API key requirement
    - Test system message handling
    - Test streaming response parsing
    - _Requirements: 3.6, 3.8_

### Phase 4: RAG Package Implementation

- [ ] 4. Implement RAG logic
  - [x] 4.1 Create chunking functionality
    - Implement internal/rag/chunker.go with Chunker struct
    - Implement ChunkText() method with overlap logic
    - Use rune-based slicing for proper Unicode handling
    - _Requirements: 4.1, 4.5_

  - [ ]* 4.2 Write property tests for chunking
    - **Property 14: Chunk Size Bounds**
    - **Validates: Requirements 4.5**
    - **Property 15: Chunk Overlap**
    - **Validates: Requirements 4.5**
    - _Requirements: 4.5_


  - [x] 4.3 Create search functionality
    - Implement internal/rag/search.go with Searcher struct
    - Implement Search() method that delegates to store
    - Implement CosineSimilarity() function
    - _Requirements: 4.2, 4.4_

  - [ ]* 4.4 Write property test for cosine similarity
    - **Property 16: Cosine Similarity Bounds**
    - **Validates: Requirements 4.4**
    - Test returns value between -1.0 and 1.0
    - Test identical vectors return 1.0
    - _Requirements: 4.4_

  - [x] 4.5 Create prompt building functionality
    - Implement internal/rag/prompt.go with PromptBuilder struct
    - Implement BuildPrompt() method that combines query and chunks
    - Format context with source attribution
    - _Requirements: 4.3_

  - [ ]* 4.6 Write property test for prompt building
    - **Property 17: Prompt Contains Query and Context**
    - **Validates: Requirements 4.3**
    - _Requirements: 4.3_

### Phase 5: Ingest Package Implementation

- [ ] 5. Implement document ingestion
  - [x] 5.1 Create ingestion orchestrator
    - Implement internal/ingest/ingest.go with Ingester struct
    - Implement IngestText() method with chunking, embedding, and storage
    - Add summary generation when enabled
    - Integrate PII detection and guardrails checks
    - _Requirements: 5.1, 29.1-29.8_

  - [ ]* 5.2 Write property test for text ingestion
    - **Property 18: Text Ingestion Creates Chunks**
    - **Validates: Requirements 5.1**
    - _Requirements: 5.1_

  - [x] 5.3 Implement URL ingestion
    - Implement IngestURL() method with HTTP fetch
    - Use go-readability for HTML parsing
    - Add privacy mode check to block URL ingestion
    - _Requirements: 5.2, 5.7_

  - [ ]* 5.4 Write property test for URL ingestion privacy mode
    - **Property 19: URL Ingestion Blocked in Privacy Mode**
    - **Validates: Requirements 5.7**
    - _Requirements: 5.7_

  - [x] 5.5 Implement file ingestion
    - Implement IngestFile() method with MIME type detection
    - Implement parseText() for .txt and .md files
    - Implement parsePDF() for .pdf files (placeholder or use library)
    - Add file size validation
    - _Requirements: 5.3-5.5_


  - [ ]* 5.6 Write property test for ingestion errors
    - **Property 20: Ingestion Errors Are Descriptive**
    - **Validates: Requirements 5.6**
    - _Requirements: 5.6_

  - [x] 5.7 Implement PII detection
    - Create internal/ingest/pii.go with PIIDetector struct
    - Implement NewPIIDetector() with regex patterns for SSN, credit cards, API keys, private keys, emails, phones
    - Implement Detect() method that returns list of detected PII types
    - _Requirements: 28.1-28.11_

  - [ ]* 5.8 Write property tests for PII detection
    - **Property 31: PII Detection Finds Patterns**
    - **Validates: Requirements 28.1-28.7**
    - **Property 32: PII Logs Don't Contain Values**
    - **Validates: Requirements 28.11**
    - Test each PII pattern (SSN, credit card, API key, private key, email, phone)
    - _Requirements: 28.1-28.11_

  - [x] 5.9 Implement guardrails
    - Create internal/ingest/guardrails.go with Guardrails struct
    - Implement NewGuardrails() with safe defaults
    - Implement Check() method for filename and content validation
    - Implement IsAllowedExtension() method
    - Add blocked extensions list (executables, archives, disk images)
    - Add sensitive filename patterns
    - _Requirements: 31.1-31.11_

  - [ ]* 5.10 Write property tests for guardrails
    - **Property 33: Oversized Files Are Rejected**
    - **Validates: Requirements 31.1-31.2**
    - **Property 34: Disallowed Extensions Are Rejected**
    - **Validates: Requirements 31.3-31.4**
    - **Property 35: Executables Are Rejected**
    - **Validates: Requirements 31.5**
    - **Property 36: Archives Are Rejected**
    - **Validates: Requirements 31.6**
    - **Property 37: Sensitive Filenames Are Rejected**
    - **Validates: Requirements 31.8**
    - _Requirements: 31.1-31.11_

### Phase 6: Skills Package Implementation

- [ ] 6. Implement skill system
  - [x] 6.1 Create skill loader
    - Implement internal/skills/loader.go with Loader struct
    - Implement Skill and Metadata structs
    - Implement NewLoader() constructor
    - Implement LoadAll() method that discovers skills in directory
    - Implement loadSkill() method that parses skill.json and validates executable
    - Add privacy mode filtering for network-requiring skills
    - _Requirements: 33.1-33.22, 33.1.1-33.1.7_


  - [ ]* 6.2 Write property test for skill loading
    - **Property 38: Skills Load from Directory**
    - **Validates: Requirements 33.1-33.3**
    - _Requirements: 33.1-33.3_

  - [x] 6.3 Create skill executor
    - Implement internal/skills/executor.go with Executor struct
    - Implement Input and Output structs for JSON communication
    - Implement Execute() method with subprocess spawning
    - Add timeout enforcement with context.WithTimeout
    - Implement buildEnv() method for environment variable setup
    - Add privacy mode environment variable passing
    - _Requirements: 33.4-33.21_

  - [ ]* 6.4 Write property tests for skill execution
    - **Property 39: Skill Input/Output JSON Round Trip**
    - **Validates: Requirements 33.5-33.7**
    - **Property 40: Skill Timeout Enforced**
    - **Validates: Requirements 33.13-33.14**
    - **Property 41: Privacy Mode Env Var Passed to Skills**
    - **Validates: Requirements 33.15-33.16**
    - _Requirements: 33.5-33.7, 33.13-33.16_

  - [x] 6.5 Create example skills
    - Create skills/examples/weather/ directory with skill.json and weather.sh
    - Create skills/examples/summarize-url/ directory with skill.json and script
    - Create skills/examples/daily-digest/ directory with skill.json and script
    - Add comprehensive comments and documentation to example skills
    - _Requirements: 33.2.1-33.2.6_

### Phase 7: Folder Watcher Implementation

- [ ] 7. Implement folder watching
  - [x] 7.1 Create watcher package
    - Implement internal/watcher/watcher.go with Watcher struct
    - Implement NewWatcher() constructor with fsnotify initialization
    - Implement Start() method that loads watched folders and starts event loop
    - Implement eventLoop() method for processing filesystem events
    - Implement handleEvent() method for create/modify/delete events
    - Implement shouldProcess() method for extension and size validation
    - Implement validatePath() method to block system directories
    - Implement AddFolder() method
    - _Requirements: 27.1-27.12_

  - [ ]* 7.2 Write property tests for folder watcher
    - **Property 25: File Creation Triggers Ingestion**
    - **Validates: Requirements 27.2**
    - **Property 26: File Modification Triggers Re-ingestion**
    - **Validates: Requirements 27.3**
    - **Property 27: File Deletion Removes Chunks**
    - **Validates: Requirements 27.4**
    - **Property 28: Watcher Skips Disallowed Extensions**
    - **Validates: Requirements 27.5**
    - **Property 29: Watcher Skips Oversized Files**
    - **Validates: Requirements 27.6**
    - **Property 30: Watcher Rejects System Directories**
    - **Validates: Requirements 27.8**
    - _Requirements: 27.2-27.8_


### Phase 8: API Package and HTTP Handlers

- [ ] 8. Implement API server and handlers
  - [x] 8.1 Create API server structure
    - Implement internal/api/server.go with Server struct
    - Implement NewServer() constructor with template loading
    - Implement RegisterRoutes() method for all HTTP routes
    - Add static file serving for /static/
    - _Requirements: 6.1-6.5_

  - [x] 8.2 Implement WebSocket hub
    - Create internal/api/websocket.go with WebSocketHub struct
    - Implement NewWebSocketHub() constructor
    - Implement Run() method for event loop (register, unregister, broadcast)
    - Implement Broadcast() method for sending messages to all clients
    - Implement handleWebSocket() handler for upgrading connections
    - _Requirements: 16.1-16.5_

  - [x] 8.3 Implement dashboard handler
    - Implement handleDashboard() in internal/api/handlers.go
    - Query store for document count, last ingestion timestamp
    - Get provider name and privacy mode status from config
    - Render dashboard.html template with stats
    - _Requirements: 9.1-9.6_

  - [x] 8.4 Implement chat handlers
    - Implement handleChat() for rendering chat page
    - Implement handleAsk() for processing queries with RAG
    - Add session management with session ID generation
    - Integrate embedding, search, prompt building, and streaming
    - Save user and assistant messages to store
    - Add audit logging for queries
    - _Requirements: 10.1-10.6, 11.1-11.5, 12.1-12.5_

  - [x] 8.5 Implement session management handlers
    - Implement handleSessions() for listing all sessions
    - Implement handleSessionHistory() for retrieving messages by session ID
    - Return HTML fragments for HTMX integration
    - _Requirements: 10.3-10.6_

  - [x] 8.6 Implement library handlers
    - Implement handleLibrary() for rendering library page
    - Add tag filtering support via query parameter
    - Return document cards as HTML fragments for HTMX
    - _Requirements: 13.1-13.5, 15.1-15.6_

  - [x] 8.7 Implement ingestion handlers
    - Implement handleIngestText() for plain text ingestion
    - Implement handleIngestURL() for URL ingestion
    - Implement handleIngestFile() for file upload ingestion
    - Add WebSocket broadcast on ingestion completion
    - Add audit logging for ingestions
    - _Requirements: 5.1-5.8, 14.1-14.5_

  - [x] 8.8 Implement delete handler
    - Implement handleDelete() for removing documents
    - Add WebSocket broadcast on deletion
    - Add audit logging for deletions
    - _Requirements: 2.4_


  - [ ] 8.9 Implement settings handlers
    - Implement handleSettings() for rendering settings page
    - Implement handleConfig() for saving configuration changes
    - Implement handleTestConnection() for testing provider connectivity
    - Add watched folder management endpoints
    - Add privacy mode toggle with validation
    - _Requirements: 18.1.1-18.1.9, 18.2.1-18.2.10_

  - [ ] 8.10 Implement activity feed handler
    - Implement handleActivity() for dashboard activity feed
    - Query audit log for recent 10 entries
    - Return HTML fragment with formatted activity items
    - _Requirements: 9.4, 30.1-30.10_

  - [ ] 8.11 Implement skills API handlers
    - Implement handleSkills() for listing available skills
    - Implement handleRunSkill() for executing manual-trigger skills
    - Return skill results as JSON
    - _Requirements: 33.9, 33.18-33.19_

  - [ ]* 8.12 Write integration tests for API handlers
    - Test handleAsk() with mock store and provider
    - Test handleIngestText() with validation
    - Test handleLibrary() with tag filtering
    - Test WebSocket broadcast on ingestion
    - _Requirements: 6.1-6.5_

### Phase 9: UI Templates

- [ ] 9. Create HTML templates
  - [ ] 9.1 Create base template
    - Create web/templates/base.html with common layout
    - Add sidebar with navigation links (Dashboard, Chat, Library, Settings)
    - Add sidebar toggle button with collapse functionality
    - Add privacy mode badge when enabled
    - Add toast container div
    - Add command palette div
    - Add WebSocket connection script
    - _Requirements: 7.1, 8.1-8.6_

  - [ ] 9.2 Create dashboard template
    - Create web/templates/dashboard.html
    - Add stats grid with document count, provider, last ingestion, privacy mode cards
    - Add quick actions section with buttons
    - Add activity feed section with HTMX auto-refresh
    - _Requirements: 9.1-9.6_

  - [ ] 9.3 Create chat template
    - Create web/templates/chat.html
    - Add session sidebar with "New Chat" button and session list
    - Add messages area with user/assistant message rendering
    - Add chat input form with textarea and send button
    - Add JavaScript for sendMessage(), addMessage(), copyMessage(), newChat()
    - Add markdown rendering for assistant messages
    - _Requirements: 10.1-10.6, 11.1-11.5, 12.1-12.5_


  - [ ] 9.4 Create library template
    - Create web/templates/library.html
    - Add library header with tag filter dropdown
    - Add drop zone overlay for drag-and-drop
    - Add document grid with HTMX refresh trigger
    - Add JavaScript for drag-and-drop file upload
    - Add JavaScript for uploadFile(), filterByTag(), deleteDocument()
    - _Requirements: 13.1-13.5, 14.1-14.5, 15.1-15.6_

  - [ ] 9.5 Create settings template
    - Create web/templates/settings.html
    - Add privacy mode toggle section with description
    - Add LLM provider selection with provider-specific settings sections
    - Add Ollama settings fields (endpoint, embed model, chat model)
    - Add OpenAI settings fields (API key, embed model, chat model)
    - Add Anthropic settings fields (API key, chat model)
    - Add watched folders section with add/remove functionality
    - Add guardrails section (PII detection, auto-summarize)
    - Add JavaScript for updateProviderFields(), togglePrivacyMode(), saveSettings(), testConnection()
    - _Requirements: 18.1.1-18.1.9, 18.2.1-18.2.10, 27.12_

### Phase 10: Frontend Assets

- [ ] 10. Create CSS and JavaScript
  - [ ] 10.1 Create main stylesheet
    - Create web/static/style.css with CSS variables for colors
    - Add sidebar styles with collapse transition
    - Add navigation item styles with hover and active states
    - Add privacy badge styles
    - Add main content area styles with responsive margin
    - Add dashboard stats grid and card styles
    - Add chat container, session sidebar, and message styles
    - Add library document grid and card styles with hover effects
    - Add drop zone styles
    - Add toast notification styles with animations
    - Add command palette styles
    - Add button styles (primary, secondary, icon)
    - Add toggle switch styles
    - Add form input and select styles
    - _Requirements: 21.1-21.5_

  - [ ] 10.2 Create client-side JavaScript
    - Create web/static/app.js
    - Implement showToast() function for notifications
    - Implement command palette toggle with ⌘K/Ctrl+K keyboard shortcut
    - Implement loadCommands() and renderCommands() for command palette
    - Implement sidebar toggle with localStorage persistence
    - Add markdown rendering helper (placeholder for server-side rendering)
    - _Requirements: 17.1-17.5, 18.1-18.5_

  - [ ] 10.3 Add HTMX library
    - Download htmx.min.js and place in web/static/
    - Verify HTMX version is 1.9.x or later
    - _Requirements: 20.1-20.5_


### Phase 11: Main Entry Point Refactoring

- [ ] 11. Refactor main.go
  - [x] 11.1 Create minimal main.go entry point
    - Load configuration from config.json
    - Initialize logger with configured level
    - Initialize store with database migrations
    - Initialize LLM provider based on config and privacy mode
    - Initialize RAG components (chunker, searcher, prompt builder)
    - Initialize ingester with PII detector and guardrails
    - Initialize skill loader and load all skills
    - Initialize folder watcher and start watching
    - Initialize API server with all dependencies
    - Register routes and start HTTP server
    - Add graceful shutdown handling
    - Ensure main.go is under 100 lines
    - _Requirements: 1.7-1.9_

  - [ ]* 11.2 Write integration test for main initialization
    - Test full application startup with in-memory database
    - Test configuration loading
    - Test all components initialize successfully
    - _Requirements: 1.9_

### Phase 12: Markdown Rendering

- [ ] 12. Implement markdown rendering
  - [ ] 12.1 Add goldmark dependency
    - Add github.com/yuin/goldmark to go.mod
    - Add syntax highlighting extension
    - _Requirements: 11.1-11.5, 22.1_

  - [ ] 12.2 Implement markdown rendering in API handlers
    - Create renderMarkdown() helper function using goldmark
    - Apply markdown rendering to assistant messages in handleAsk()
    - Configure goldmark with syntax highlighting and safe HTML
    - _Requirements: 11.1-11.5_

  - [ ]* 12.3 Write unit tests for markdown rendering
    - Test code block rendering with syntax highlighting
    - Test inline code rendering
    - Test list and header rendering
    - Test XSS protection (script tags escaped)
    - _Requirements: 11.1-11.5_

### Phase 13: Dependency Management

- [ ] 13. Update dependencies
  - [ ] 13.1 Update go.mod with all dependencies
    - Add github.com/yuin/goldmark for markdown rendering
    - Add github.com/gorilla/websocket for WebSocket support
    - Add github.com/fsnotify/fsnotify for folder watching
    - Verify modernc.org/sqlite is present
    - Verify github.com/go-shiori/go-readability is present
    - Run go mod tidy to clean up
    - _Requirements: 22.1-22.5_

  - [ ] 13.2 Verify build
    - Run go build to ensure no compilation errors
    - Run go test ./... to ensure all tests pass
    - _Requirements: 22.5_


### Phase 14: Testing and Validation

- [ ] 14. Comprehensive testing
  - [ ] 14.1 Run all unit tests
    - Execute go test ./internal/config -v
    - Execute go test ./internal/logging -v
    - Execute go test ./internal/store -v
    - Execute go test ./internal/llm -v
    - Execute go test ./internal/rag -v
    - Execute go test ./internal/ingest -v
    - Execute go test ./internal/skills -v
    - Execute go test ./internal/watcher -v
    - Execute go test ./internal/api -v
    - _Requirements: All package requirements_

  - [ ] 14.2 Run all property-based tests
    - Execute property tests with gopter
    - Verify all 41 correctness properties pass
    - Check test coverage meets goals (store 90%+, llm 80%+, rag 85%+, ingest 85%+, api 75%+, skills 80%+)
    - _Requirements: All property requirements_

  - [ ] 14.3 Manual testing checklist
    - Test dashboard loads with correct stats
    - Test chat with new session creation
    - Test chat with session history loading
    - Test library with document cards
    - Test drag-and-drop file upload
    - Test tag filtering in library
    - Test document deletion
    - Test settings page with provider switching
    - Test privacy mode toggle
    - Test command palette (⌘K)
    - Test sidebar collapse/expand
    - Test WebSocket real-time updates
    - Test toast notifications
    - Test folder watcher with file creation
    - Test skill execution (manual trigger)
    - _Requirements: All UI requirements_

  - [ ] 14.4 Backward compatibility testing
    - Create Phase 1 database with sample chunks
    - Start Phase 2 application
    - Verify all Phase 1 chunks are readable
    - Verify search works with Phase 1 embeddings
    - Verify no data loss during migration
    - _Requirements: 23.1-23.4_

### Phase 15: Documentation

- [ ] 15. Update documentation
  - [ ] 15.1 Update README.md
    - Add Phase 2 features overview
    - Add installation instructions
    - Add configuration guide with config.json examples
    - Add privacy mode explanation
    - Add skill development guide
    - Add API documentation for HTTP endpoints
    - Add troubleshooting section
    - _Requirements: All requirements_

  - [ ] 15.2 Create SKILLS.md
    - Document skill.json format
    - Document skill input/output JSON format
    - Document environment variables available to skills
    - Document trigger types (manual, timer, keyword, event)
    - Provide example skills with explanations
    - _Requirements: 33.1-33.22, 33.1.1-33.1.7, 33.2.1-33.2.6_


  - [ ] 15.3 Create CONFIG.md
    - Document all configuration options
    - Document environment variable overrides
    - Document provider-specific settings
    - Document privacy mode implications
    - Document guardrails settings
    - Document logging configuration
    - Provide example configurations for different use cases
    - _Requirements: 25.1-25.20_

### Phase 16: Final Integration and Polish

- [ ] 16. Final integration
  - [ ] 16.1 End-to-end testing
    - Start fresh application with default config
    - Ingest sample documents via UI
    - Perform chat queries with RAG
    - Test all UI interactions
    - Test WebSocket updates
    - Test folder watcher
    - Test skill execution
    - Verify audit log entries
    - _Requirements: All requirements_

  - [ ] 16.2 Performance validation
    - Test with 1000+ documents in library
    - Test search performance with large vector database
    - Test concurrent ingestion (folder watcher)
    - Test WebSocket with multiple clients
    - Verify no memory leaks during long-running operation
    - _Requirements: Performance-related requirements_

  - [ ] 16.3 Security validation
    - Test privacy mode enforcement (cloud providers blocked)
    - Test PII detection with sample data
    - Test guardrails with malicious filenames
    - Test system directory blocking in folder watcher
    - Test API key masking in settings UI
    - Test localhost binding by default
    - _Requirements: 28.1-28.11, 31.1-31.11, 32.1-32.5_

  - [ ] 16.4 UI polish
    - Verify all CSS transitions work smoothly
    - Verify responsive layout on different screen sizes
    - Verify toast notifications auto-dismiss
    - Verify command palette fuzzy search
    - Verify sidebar state persistence
    - Verify all icons and badges display correctly
    - _Requirements: 21.1-21.5_

  - [ ] 16.5 Error handling validation
    - Test with invalid config.json
    - Test with unreachable Ollama endpoint
    - Test with invalid API keys
    - Test with corrupted database
    - Test with missing templates
    - Verify all errors are descriptive and logged
    - _Requirements: 24.1-24.5_

## Notes

- Tasks marked with `*` are optional testing tasks and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Property tests validate universal correctness properties from the design document
- Unit tests validate specific examples and edge cases
- All Phase 1 functionality must remain working throughout refactoring
- Database migrations must be backward compatible
- Privacy mode enforcement is critical for security

## Completion Criteria

The implementation is complete when:
1. All non-optional tasks are completed
2. All unit tests pass
3. Manual testing checklist is verified
4. Backward compatibility with Phase 1 data is confirmed
5. Documentation is updated
6. Application builds and runs without errors
