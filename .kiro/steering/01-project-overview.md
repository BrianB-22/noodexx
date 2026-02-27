# Noodexx Project Overview

## Purpose

This steering document provides a high-level understanding of the Noodexx project architecture, design philosophy, and key concepts for developers working on future enhancements or bug fixes.

## What is Noodexx?

Noodexx is a privacy-first, local-first AI knowledge assistant with Retrieval-Augmented Generation (RAG). It enables users to:
- Ingest and index documents (text, PDF, HTML, Markdown)
- Search through their knowledge base using vector similarity
- Chat with an AI that has access to their documents
- Extend functionality through a plugin system (skills)
- Choose between local AI (Ollama) and cloud AI (OpenAI/Anthropic)

## Core Design Principles

### 1. Privacy First
- Local provider (Ollama) is MANDATORY
- Cloud provider (OpenAI/Anthropic) is OPTIONAL
- Users can switch between providers instantly
- Privacy mode enforces local-only operation
- No data leaves the machine unless explicitly using cloud provider

### 2. Graceful Degradation
- Application should launch even if cloud provider is misconfigured
- Missing components should log warnings, not crash the application
- UI should adapt to available features (disable unavailable options)

### 3. Dual-Provider Architecture
- Local provider: Always required, runs on localhost (Ollama)
- Cloud provider: Optional, provides access to GPT-4, Claude, etc.
- Users toggle between providers via UI without restart
- RAG policy controls whether documents are sent to cloud

### 4. Modular Architecture
- Each package has a single, well-defined responsibility
- Packages communicate through interfaces (dependency injection)
- Adapters bridge between main.go and internal packages
- Easy to test, extend, and maintain

### 5. User Modes
- **Single-user mode**: No authentication, auto-login as "local-default" user
- **Multi-user mode**: Full authentication with username/password, sessions, lockout protection

## Technology Stack

### Backend
- **Language**: Go 1.21+
- **Database**: SQLite with vector storage
- **Web Framework**: Standard library net/http
- **Templates**: Go html/template
- **WebSocket**: gorilla/websocket for real-time updates

### Frontend
- **HTMX**: HTML-driven AJAX for partial page updates
- **Alpine.js**: Lightweight reactive JavaScript framework
- **Tailwind CSS**: Utility-first CSS framework (CDN + vendored fallback)
- **Server-Sent Events**: Streaming chat responses

### AI/ML
- **Local LLM**: Ollama (llama3.2, nomic-embed-text)
- **Cloud LLM**: OpenAI (GPT-4, text-embedding-3-small) or Anthropic (Claude)
- **Vector Search**: Cosine similarity on embeddings

## Project Structure

```
noodexx/
├── main.go                 # Application entry point, dependency wiring
├── adapters.go             # Adapter implementations for package interfaces
├── config.json             # Runtime configuration (gitignored)
├── config.json.example     # Template configuration for new setups
├── uistyle.json            # Centralized UI theme configuration
├── noodexx.db              # SQLite database (gitignored)
├── internal/               # Internal packages (not importable externally)
│   ├── api/                # HTTP handlers, WebSocket hub, server
│   ├── auth/               # Authentication, middleware, session management
│   ├── config/             # Configuration loading, validation, persistence
│   ├── ingest/             # Document parsing, PII detection, guardrails
│   ├── llm/                # LLM provider abstraction (Ollama, OpenAI, Anthropic)
│   ├── logging/            # Structured logging with rotation
│   ├── provider/           # Dual provider manager (local + cloud)
│   ├── rag/                # Chunking, vector search, prompt building, policy enforcement
│   ├── skills/             # Plugin system (loader, executor)
│   ├── store/              # Database operations, migrations, models
│   ├── uistyle/            # UI theme configuration management
│   └── watcher/            # Folder monitoring for auto-ingestion
├── web/                    # Frontend assets
│   ├── static/             # CSS, JS, images (vendored dependencies)
│   └── templates/          # Go HTML templates
│       ├── base.html       # Page shell with dependencies
│       ├── components/     # Reusable UI components
│       ├── chat.html       # Chat interface
│       ├── library.html    # Document library
│       ├── settings.html   # Configuration UI
│       └── dashboard.html  # System overview
├── skills/                 # User-installed skills (plugins)
│   └── examples/           # Example skills (weather, summarize-url, etc.)
└── .kiro/                  # Development metadata
    ├── specs/              # Feature and bugfix specifications
    └── steering/           # Development guidance documents (this file)
```

## Key Concepts

### RAG (Retrieval-Augmented Generation)
1. **Ingestion**: Documents are chunked into ~500 token pieces
2. **Embedding**: Each chunk is converted to a vector using an embedding model
3. **Storage**: Vectors stored in SQLite with metadata (source, tags, summary)
4. **Retrieval**: User query is embedded, similar chunks found via cosine similarity
5. **Augmentation**: Retrieved chunks added to LLM prompt as context
6. **Generation**: LLM generates response using both its training and retrieved context

### Dual Provider Manager
- Manages two provider instances: local (Ollama) and cloud (OpenAI/Anthropic)
- Local provider is mandatory, cloud provider is optional
- Graceful degradation: if cloud provider fails to initialize, app continues with local only
- User can switch between providers via UI toggle
- RAG policy controls whether documents are sent to cloud provider

### Skills System
- Extensible plugin architecture
- Skills are external executables (shell scripts, Python, binaries)
- Communicate via JSON stdin/stdout
- Triggers: manual, keyword, timer, event
- Privacy mode enforcement (blocks network-dependent skills)

### Authentication Modes
- **Single-user mode**: No login required, auto-authenticated as "local-default"
- **Multi-user mode**: Username/password authentication with sessions
- Middleware automatically injects user context based on mode
- All data operations are user-scoped (chunks, messages, sessions)

## Development Workflow

### Spec-Driven Development
1. Create spec in `.kiro/specs/{feature-name}/`
2. Write requirements.md (user stories, acceptance criteria)
3. Write design.md (architecture, implementation details)
4. Write tasks.md (implementation checklist)
5. Implement tasks sequentially
6. Write tests (bug condition, preservation, unit, integration)
7. Verify all tests pass before marking complete

### Bug Fix Workflow
1. Create bugfix spec in `.kiro/specs/{bugfix-name}/`
2. Document bug condition (when does it occur?)
3. Document expected behavior (what should happen?)
4. Document preservation requirements (what must not break?)
5. Write bug condition exploration test (should fail on unfixed code)
6. Write preservation tests (should pass on unfixed code)
7. Implement fix
8. Verify bug condition test now passes
9. Verify preservation tests still pass

### Testing Philosophy
- **Bug condition tests**: Prove the bug exists, then prove it's fixed
- **Preservation tests**: Ensure existing functionality is not broken
- **Property-based testing**: Generate many test cases for stronger guarantees
- **Integration tests**: Test full request/response cycles
- **Unit tests**: Test individual functions and components

## Common Patterns

### Adapter Pattern
main.go creates adapters that implement package interfaces:
```go
type apiStoreAdapter struct {
    store *store.Store
}

func (a *apiStoreAdapter) SaveChunk(...) error {
    return a.store.SaveChunk(...)
}
```

### Dependency Injection
Packages receive dependencies through constructors:
```go
func NewServer(
    store Store,
    provider LLMProvider,
    ingester Ingester,
    // ... more dependencies
) (*Server, error)
```

### Interface-Based Design
Packages define interfaces for their dependencies:
```go
type Store interface {
    SaveChunk(ctx context.Context, ...) error
    SearchChunks(ctx context.Context, ...) ([]Chunk, error)
}
```

### Error Handling
- Return errors, don't panic
- Wrap errors with context: `fmt.Errorf("failed to X: %w", err)`
- Log errors before returning
- Graceful degradation where possible

## Configuration Management

### config.json
- Runtime configuration (providers, privacy, guardrails, server)
- Gitignored (contains API keys)
- Validated on load
- Can be updated via UI (settings page)
- Environment variables override config values

### uistyle.json
- UI theme configuration (colors, typography, spacing, shadows)
- Validated on load (strict schema)
- Injected into Tailwind CSS config
- Applies to all users consistently

## Database Schema

### Key Tables
- **users**: User accounts (username, password_hash, dark_mode preference)
- **chunks**: Document chunks with embeddings (user_id, source, text, embedding, tags)
- **chat_messages**: Conversation history (user_id, session_id, role, content)
- **session_tokens**: Authentication sessions (user_id, token, expires_at)
- **audit_log**: Operation history (user_id, operation_type, details)
- **watched_folders**: Auto-ingest directories (user_id, path)

### Migrations
- Managed by internal/store/migrations.go
- Applied automatically on startup
- Version tracked in schema_version table
- Idempotent (safe to run multiple times)

## Next Steps

For specific development guidance, see:
- `02-architecture-deep-dive.md` - Detailed package architecture
- `03-common-tasks.md` - How to implement common features
- `04-testing-guide.md` - Testing strategies and patterns
- `05-ui-development.md` - Frontend development guide
- `06-troubleshooting.md` - Common issues and solutions
