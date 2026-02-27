# Noodexx Architecture Deep Dive

## Purpose

This document provides detailed architectural information about each package in Noodexx, including responsibilities, interfaces, key functions, and interaction patterns.

## Package Architecture

### internal/api - HTTP Server and Handlers

**Responsibility**: HTTP request handling, WebSocket management, template rendering

**Key Files**:
- `server.go` - Server initialization, template loading, route registration
- `handlers.go` - HTTP handlers for pages and API endpoints
- `websocket.go` - WebSocket hub for real-time notifications
- `settings_handlers.go` - Configuration management endpoints

**Key Interfaces**:
```go
type Store interface {
    SaveChunk(ctx context.Context, userID int64, source, text string, embedding []float32, tags []string, summary string) error
    SearchChunks(ctx context.Context, userID int64, query []float32, limit int) ([]Chunk, error)
    SaveChatMessage(ctx context.Context, userID int64, sessionID, role, content, providerMode string) error
    GetUserByUsername(ctx context.Context, username string) (*User, error)
    UpdateUserDarkMode(ctx context.Context, userID int64, darkMode bool) error
    // ... more methods
}

type LLMProvider interface {
    Chat(ctx context.Context, messages []Message) (string, error)
    Embed(ctx context.Context, text string) ([]float32, error)
}

type ProviderManager interface {
    GetActiveProvider() (LLMProvider, error)
    GetLocalProvider() LLMProvider
    GetCloudProvider() LLMProvider
    IsLocalMode() bool
    Reload(cfg *config.Config) error
}
```

**Key Functions**:
- `NewServer()` - Creates server with all dependencies
- `handleChat()` - Renders chat interface with user's dark mode preference
- `handleAsk()` - Processes chat messages with streaming response
- `handleSettings()` - Renders settings page with cloud provider availability
- `handleIngestFile()` - Handles file uploads and ingestion
- `handleUpdatePreferences()` - Updates user preferences (dark mode)

**Template Data Pattern**:
```go
data := map[string]interface{}{
    "Title":                  "Page Title",
    "Page":                   "page-name",
    "CloudProviderAvailable": s.providerManager.GetCloudProvider() != nil,
    "UIStyle":                s.uiStyle,
    "DarkMode":               darkMode, // from user preferences
}
```

---

### internal/auth - Authentication and Authorization

**Responsibility**: User authentication, session management, middleware

**Key Files**:
- `middleware.go` - Authentication middleware for single/multi-user modes
- `userpass.go` - Username/password authentication provider
- `token.go` - Session token generation and validation
- `password.go` - Password hashing with bcrypt

**Key Interfaces**:
```go
type Store interface {
    GetUserByUsername(ctx context.Context, username string) (*User, error)
    CreateSessionToken(ctx context.Context, userID int64, token string, expiresAt time.Time) error
    GetSessionToken(ctx context.Context, token string) (*SessionToken, error)
    DeleteSessionToken(ctx context.Context, token string) error
}

type Provider interface {
    Login(ctx context.Context, username, password string) (string, error)
    ValidateToken(ctx context.Context, token string) (int64, error)
    Logout(ctx context.Context, token string) error
}
```

**Authentication Flow**:
1. **Single-user mode**: Middleware automatically injects "local-default" user
2. **Multi-user mode**: Middleware validates session token, injects user_id into context
3. Public endpoints (/login, /register, /static/) bypass authentication
4. Browser requests redirect to /login, API requests return 401

**Context Key Pattern**:
```go
const UserIDKey contextKey = "user_id"

// In middleware
ctx := context.WithValue(r.Context(), UserIDKey, userID)

// In handlers
userID, err := auth.GetUserID(r.Context())
```

---

### internal/config - Configuration Management

**Responsibility**: Load, validate, and persist configuration

**Key Files**:
- `config.go` - Configuration struct, loading, validation, environment overrides

**Configuration Structure**:
```go
type Config struct {
    LocalProvider ProviderConfig  // Mandatory Ollama configuration
    CloudProvider ProviderConfig  // Optional OpenAI/Anthropic configuration
    Privacy       PrivacyConfig   // Provider toggle and RAG policy
    Folders       []string        // Auto-ingest directories
    Logging       LoggingConfig   // Log level, file, rotation
    Guardrails    GuardrailsConfig // File size, extensions, PII detection
    Server        ServerConfig    // Port, bind address
    UserMode      string          // "single" or "multi"
    Auth          AuthConfig      // Session expiry, lockout settings
}
```

**Validation Rules**:
- Local provider type must be "ollama"
- Local provider endpoint must be localhost/127.0.0.1
- Cloud provider requires API key if type is "openai" or "anthropic"
- User mode must be "single" or "multi"
- Auth provider must be "userpass", "mfa", or "sso"

**Environment Variable Overrides**:
- `NOODEXX_LOCAL_PROVIDER_TYPE`
- `NOODEXX_CLOUD_PROVIDER_OPENAI_KEY`
- `NOODEXX_PRIVACY_USE_LOCAL_AI`
- `NOODEXX_SERVER_PORT`
- `NOODEXX_USER_MODE`
- etc.

---

### internal/provider - Dual Provider Manager

**Responsibility**: Manage local and cloud LLM providers with graceful degradation

**Key Files**:
- `dual_manager.go` - Dual provider management with fallback logic

**Key Concepts**:
- **Local provider**: Always required, initialized first
- **Cloud provider**: Optional, graceful degradation if initialization fails
- **Active provider**: Determined by Privacy.DefaultToLocal setting
- **Reload**: Supports runtime configuration changes

**Graceful Degradation Logic**:
```go
// Cloud provider initialization
provider, err := llm.NewProvider(cloudCfg, false, logger)
if err != nil {
    // Log warning and continue with local provider only
    logger.Warn("Cloud provider initialization failed: %v. Application will run with local provider only.", err)
    manager.cloudProvider = nil
} else {
    manager.cloudProvider = provider
}

// Local provider is mandatory
if manager.localProvider == nil {
    return nil, fmt.Errorf("A local provider is required. Please refer to documentation on configuration.")
}
```

**Provider Switching**:
- User toggles privacy mode in UI
- API call updates config.Privacy.DefaultToLocal
- DualProviderManager.Reload() reinitializes providers
- GetActiveProvider() returns appropriate provider based on mode

---

### internal/llm - LLM Provider Abstraction

**Responsibility**: Abstract interface for different LLM providers

**Key Files**:
- `provider.go` - Provider interface and factory
- `ollama.go` - Ollama implementation
- `openai.go` - OpenAI implementation
- `anthropic.go` - Anthropic implementation

**Provider Interface**:
```go
type Provider interface {
    Chat(ctx context.Context, messages []Message) (string, error)
    Embed(ctx context.Context, text string) ([]float32, error)
    StreamChat(ctx context.Context, messages []Message) (<-chan string, <-chan error)
}
```

**Provider Selection**:
```go
func NewProvider(cfg Config, isLocal bool, logger *logging.Logger) (Provider, error) {
    switch cfg.Type {
    case "ollama":
        return NewOllamaProvider(cfg, logger)
    case "openai":
        return NewOpenAIProvider(cfg, logger)
    case "anthropic":
        return NewAnthropicProvider(cfg, logger)
    default:
        return nil, fmt.Errorf("unknown provider type: %s", cfg.Type)
    }
}
```

---

### internal/rag - Retrieval-Augmented Generation

**Responsibility**: Chunking, vector search, prompt building, policy enforcement

**Key Files**:
- `chunker.go` - Document chunking with overlap
- `search.go` - Vector similarity search
- `prompt.go` - Prompt building with retrieved context
- `policy_enforcer.go` - RAG policy enforcement for cloud providers

**RAG Pipeline**:
1. **Chunking**: Split document into ~500 token chunks with 50 token overlap
2. **Embedding**: Generate vector for each chunk
3. **Storage**: Save chunks with embeddings to database
4. **Retrieval**: Embed user query, find top-k similar chunks
5. **Policy Check**: Enforce RAG policy (no_rag vs allow_rag)
6. **Prompt Building**: Construct prompt with retrieved context
7. **Generation**: Send to LLM for response

**RAG Policy Enforcement**:
```go
func (e *RAGPolicyEnforcer) ShouldIncludeRAG(isLocalMode bool) bool {
    if isLocalMode {
        return true // Always allow RAG for local provider
    }
    // For cloud provider, check policy
    return e.config.Privacy.CloudRAGPolicy == "allow_rag"
}
```

---

### internal/store - Database Operations

**Responsibility**: SQLite database operations, migrations, models

**Key Files**:
- `store.go` - Main store implementation
- `datastore.go` - Core database operations
- `models.go` - Data models (User, Chunk, ChatMessage, etc.)
- `migrations.go` - Schema migrations

**Key Tables**:
- `users` - User accounts with dark_mode preference
- `chunks` - Document chunks with embeddings (user-scoped)
- `chat_messages` - Conversation history (user-scoped)
- `session_tokens` - Authentication sessions
- `audit_log` - Operation history
- `watched_folders` - Auto-ingest directories (user-scoped)

**Migration System**:
- Migrations run automatically on startup
- Version tracked in `schema_version` table
- Idempotent (safe to run multiple times)
- Creates default users based on user_mode

**User Scoping**:
All data operations include user_id for multi-tenancy:
```go
func (s *Store) SaveChunk(ctx context.Context, userID int64, source, text string, ...) error
func (s *Store) SearchChunks(ctx context.Context, userID int64, query []float32, limit int) ([]Chunk, error)
```

---

### internal/uistyle - UI Theme Configuration

**Responsibility**: Load and validate centralized UI theme configuration

**Key Files**:
- `config.go` - UIStyle configuration struct, loading, validation

**Theme Configuration**:
```go
type UIStyleConfig struct {
    Colors       ColorScheme      // Primary, secondary, success, warning, error, info, surface
    Typography   TypographyConfig // Font families and sizes
    Spacing      SpacingConfig    // Unit and scale
    BorderRadius RadiusConfig     // none, sm, base, md, lg, xl, full
    Shadows      ShadowConfig     // sm, base, md, lg, xl
}
```

**Validation**:
- All color palettes must have shades 50-900
- All colors must be valid hex codes (#RRGGBB)
- Application fails to start if uistyle.json is invalid

**Usage in Templates**:
```html
<script>
tailwind.config = {
    theme: {
        extend: {
            colors: {
                primary: {{.UIStyle.Colors.Primary | toJSON}},
                // ... more colors
            }
        }
    }
}
</script>
```

---

### internal/watcher - Folder Monitoring

**Responsibility**: Monitor directories for automatic document ingestion

**Key Files**:
- `watcher.go` - File system watcher using fsnotify

**Watch Events**:
- **Create**: Auto-ingest new files
- **Write**: Re-ingest modified files
- **Remove**: Delete chunks from database

**Guardrails**:
- File size limits
- Extension allowlist
- Concurrent processing limits
- PII detection

---

### internal/ingest - Document Ingestion

**Responsibility**: Parse documents, detect PII, apply guardrails

**Key Files**:
- `ingest.go` - Main ingestion logic
- `pii.go` - PII detection (SSN, credit cards, API keys, private keys)
- `guardrails.go` - File validation (size, extension, sensitive filenames)

**Ingestion Pipeline**:
1. Validate file (size, extension)
2. Parse content (text, PDF, HTML, Markdown)
3. Detect PII (warn user if found)
4. Chunk content
5. Generate embeddings
6. Save to database

---

### internal/skills - Plugin System

**Responsibility**: Load and execute external skills (plugins)

**Key Files**:
- `loader.go` - Skill discovery and loading
- `executor.go` - Skill execution with timeout

**Skill Communication**:
- **Input**: JSON on stdin (query, context, settings)
- **Output**: JSON on stdout (result, error, metadata)
- **Timeout**: Configurable per skill
- **Privacy Mode**: Blocks skills with requires_network=true

---

### internal/logging - Structured Logging

**Responsibility**: Dual-output logging with rotation

**Key Files**:
- `logger.go` - Logger implementation
- `filewriter.go` - File writer with rotation
- `formatter.go` - Log formatting

**Log Outputs**:
- **Console**: WARN and ERROR only, minimal format
- **File**: All levels (DEBUG, INFO, WARN, ERROR), structured format with source location

**Log Rotation**:
- Rotates when file reaches max_size_mb
- Keeps max_backups old files
- Automatic cleanup of old backups

---

## Interaction Patterns

### Request Flow (Chat)
1. User sends message via HTMX POST to /api/ask
2. handleAsk() extracts user_id from context
3. Embed user query using active provider
4. Search for relevant chunks (user-scoped)
5. Check RAG policy (allow context for cloud?)
6. Build prompt with context
7. Stream response from LLM
8. Save message to database (user-scoped)
9. Return SSE stream to client

### Provider Switching Flow
1. User toggles privacy mode in UI
2. HTMX POST to /api/privacy-toggle
3. Update config.Privacy.DefaultToLocal
4. Save config to disk
5. Reload DualProviderManager
6. Return success response
7. UI updates toggle state

### Graceful Degradation Flow
1. Application starts
2. Load config.json
3. Initialize local provider (required)
4. Attempt to initialize cloud provider
5. If cloud provider fails:
   - Log warning
   - Set cloudProvider = nil
   - Continue with local only
6. UI adapts (disable cloud toggle)
7. Settings page shows configuration message

---

## Testing Patterns

### Mock Interfaces
```go
type mockStore struct {
    chunks []Chunk
    users  map[string]*User
}

func (m *mockStore) SaveChunk(...) error {
    // Mock implementation
}
```

### Bug Condition Tests
```go
func TestBugCondition_ApplicationLaunchWithMissingCloudProviderCredentials(t *testing.T) {
    // Test that application launches successfully with local provider only
    // when cloud provider credentials are missing
}
```

### Preservation Tests
```go
func TestPreservation_DualProviderFunctionalityWithValidCredentials(t *testing.T) {
    // Test that existing functionality works exactly as before
    // when both providers have valid credentials
}
```

---

## Next Steps

For specific implementation guidance, see:
- `03-common-tasks.md` - How to implement common features
- `04-testing-guide.md` - Testing strategies and patterns
- `05-ui-development.md` - Frontend development guide
