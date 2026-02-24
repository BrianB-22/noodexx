# Design Document: Noodexx Phase 2 - Refactor and UI Enhancement

## Overview

This design document specifies the technical architecture for Noodexx Phase 2, which transforms the monolithic Phase 1 implementation into a modular, maintainable system with enhanced UI capabilities. The refactoring establishes a clean package structure following Go best practices, while the UI overhaul introduces modern web patterns including HTMX-based partial updates, WebSocket real-time notifications, and a comprehensive dashboard interface.

### Design Goals

1. **Modularity**: Break the monolithic main.go into focused packages with clear responsibilities
2. **Extensibility**: Enable plugin-based skill system for user customization
3. **Privacy**: Implement comprehensive privacy mode with local-only operation
4. **User Experience**: Modernize UI with responsive design, real-time updates, and keyboard navigation
5. **Maintainability**: Establish patterns for testing, logging, and error handling
6. **Backward Compatibility**: Preserve all Phase 1 data and functionality

### Key Technologies

- **Backend**: Go 1.21+ with standard library HTTP server
- **Database**: SQLite via modernc.org/sqlite (pure Go, no CGo)
- **Frontend**: HTMX for partial updates, vanilla JavaScript for interactions
- **WebSocket**: gorilla/websocket for real-time communication
- **Markdown**: goldmark for server-side rendering
- **File Watching**: fsnotify for folder monitoring
- **LLM Providers**: Ollama (local), OpenAI, Anthropic (cloud)

## Architecture

### High-Level System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Browser                              │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │Dashboard │  │   Chat   │  │ Library  │  │ Settings │   │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘   │
│       │             │              │              │          │
│       └─────────────┴──────────────┴──────────────┘          │
│                     │                                         │
│              HTMX + WebSocket                                │
└─────────────────────┼─────────────────────────────────────────┘
                      │
┌─────────────────────┼─────────────────────────────────────────┐
│                     ▼                                         │
│              HTTP Server (main.go)                           │
│                     │                                         │
│       ┌─────────────┴─────────────┐                         │
│       │                           │                         │
│       ▼                           ▼                         │
│  ┌─────────┐              ┌──────────────┐                 │
│  │   API   │              │  WebSocket   │                 │
│  │ Package │              │     Hub      │                 │
│  └────┬────┘              └──────────────┘                 │
│       │                                                     │
│       │                                                     │
│       ├──────┬──────┬──────┬──────┬──────┐              │
│       ▼      ▼      ▼      ▼      ▼      ▼              │
│   ┌──────┬──────┬──────┬──────┬──────┬──────┐          │
│   │Store │ LLM  │ RAG  │Ingest│Skills│Config│          │
│   │      │      │      │      │      │      │          │
│   └──┬───┴──────┴──────┴──────┴──────┴──────┘          │
│      │                                                   │
│      ▼                                                   │
│  ┌────────┐              ┌──────────────┐              │
│  │ SQLite │              │ Folder Watch │              │
│  │   DB   │              │   (fsnotify) │              │
│  └────────┘              └──────────────┘              │
│                                                         │
│                     ┌──────────────┐                   │
│                     │   Skills/    │                   │
│                     │  (external   │                   │
│                     │  processes)  │                   │
│                     └──────────────┘                   │
└─────────────────────────────────────────────────────────┘
```

### Package Structure

```
noodexx/
├── main.go                    # Entry point, initialization, wiring
├── config.json                # User configuration
├── go.mod                     # Dependencies
├── internal/                  # Private packages
│   ├── store/                 # Database abstraction
│   │   ├── store.go          # Store interface and implementation
│   │   ├── migrations.go     # Schema migrations
│   │   └── models.go         # Data models
│   ├── llm/                   # LLM provider abstraction
│   │   ├── provider.go       # Provider interface
│   │   ├── ollama.go         # Ollama implementation
│   │   ├── openai.go         # OpenAI implementation
│   │   └── anthropic.go      # Anthropic implementation
│   ├── rag/                   # RAG logic
│   │   ├── chunker.go        # Text chunking
│   │   ├── search.go         # Vector search
│   │   └── prompt.go         # Prompt building
│   ├── ingest/                # Document ingestion
│   │   ├── ingest.go         # Ingestion orchestration
│   │   ├── parsers.go        # File format parsers
│   │   ├── pii.go            # PII detection
│   │   └── guardrails.go     # Safety checks
│   ├── api/                   # HTTP handlers
│   │   ├── server.go         # Server struct and routes
│   │   ├── handlers.go       # HTTP handlers
│   │   ├── websocket.go      # WebSocket hub
│   │   └── middleware.go     # Logging, CORS, etc.
│   ├── skills/                # Skill system
│   │   ├── loader.go         # Skill discovery and loading
│   │   ├── executor.go       # Subprocess execution
│   │   ├── triggers.go       # Trigger handling
│   │   └── metadata.go       # skill.json parsing
│   ├── config/                # Configuration management
│   │   ├── config.go         # Config struct and loading
│   │   └── validation.go     # Config validation
│   ├── logging/               # Structured logging
│   │   └── logger.go         # Logger implementation
│   └── watcher/               # Folder watching
│       └── watcher.go        # fsnotify integration
├── web/                       # Frontend assets
│   ├── static/               # Static files
│   │   ├── style.css        # Main stylesheet
│   │   ├── htmx.min.js      # HTMX library
│   │   └── app.js           # Client-side JavaScript
│   └── templates/            # HTML templates
│       ├── base.html        # Base layout
│       ├── dashboard.html   # Dashboard page
│       ├── chat.html        # Chat interface
│       ├── library.html     # Library page
│       └── settings.html    # Settings page
└── skills/                    # User skills directory
    └── examples/             # Example skills
        ├── weather/
        ├── summarize-url/
        └── daily-digest/
```


## Components and Interfaces

### Store Package

The store package provides a clean abstraction over SQLite database operations.

#### Store Interface

```go
package store

import (
    "context"
    "time"
)

// Store provides database operations for Noodexx
type Store struct {
    db *sql.DB
}

// Chunk represents a text segment with its embedding
type Chunk struct {
    ID        int64
    Source    string
    Text      string
    Embedding []float32
    Tags      []string
    Summary   string
    CreatedAt time.Time
}

// LibraryEntry represents a document in the library
type LibraryEntry struct {
    Source     string
    ChunkCount int
    Summary    string
    Tags       []string
    CreatedAt  time.Time
}

// ChatMessage represents a chat message
type ChatMessage struct {
    ID        int64
    SessionID string
    Role      string // "user" or "assistant"
    Content   string
    CreatedAt time.Time
}

// Session represents a chat session
type Session struct {
    ID            string
    LastMessageAt time.Time
    MessageCount  int
}

// AuditEntry represents an audit log entry
type AuditEntry struct {
    ID            int64
    Timestamp     time.Time
    OperationType string // "ingest", "query", "delete", "config"
    Details       string
    UserContext   string
}

// WatchedFolder represents a monitored directory
type WatchedFolder struct {
    ID       int64
    Path     string
    Active   bool
    LastScan time.Time
}

// Core Methods
func NewStore(path string) (*Store, error)
func (s *Store) Close() error
func (s *Store) SaveChunk(ctx context.Context, source, text string, embedding []float32, tags []string, summary string) error
func (s *Store) Search(ctx context.Context, queryVec []float32, topK int) ([]Chunk, error)
func (s *Store) Library(ctx context.Context) ([]LibraryEntry, error)
func (s *Store) DeleteSource(ctx context.Context, source string) error

// Chat History Methods
func (s *Store) SaveMessage(ctx context.Context, sessionID, role, content string) error
func (s *Store) GetSessionHistory(ctx context.Context, sessionID string) ([]ChatMessage, error)
func (s *Store) ListSessions(ctx context.Context) ([]Session, error)

// Audit Log Methods
func (s *Store) AddAuditEntry(ctx context.Context, opType, details, userCtx string) error
func (s *Store) GetAuditLog(ctx context.Context, opType string, from, to time.Time) ([]AuditEntry, error)

// Folder Watching Methods
func (s *Store) AddWatchedFolder(ctx context.Context, path string) error
func (s *Store) GetWatchedFolders(ctx context.Context) ([]WatchedFolder, error)
func (s *Store) RemoveWatchedFolder(ctx context.Context, path string) error
```

#### Database Schema

```sql
-- Existing chunks table (Phase 1 compatible)
CREATE TABLE IF NOT EXISTS chunks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source TEXT NOT NULL,
    text TEXT NOT NULL,
    embedding BLOB NOT NULL,
    tags TEXT,              -- NEW: comma-separated tags
    summary TEXT,           -- NEW: document summary
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_chunks_source ON chunks(source);
CREATE INDEX IF NOT EXISTS idx_chunks_created ON chunks(created_at);

-- NEW: Chat messages table
CREATE TABLE IF NOT EXISTS chat_messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    role TEXT NOT NULL CHECK(role IN ('user', 'assistant')),
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_messages_session ON chat_messages(session_id);
CREATE INDEX IF NOT EXISTS idx_messages_created ON chat_messages(created_at);

-- NEW: Audit log table
CREATE TABLE IF NOT EXISTS audit_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    operation_type TEXT NOT NULL,
    details TEXT,
    user_context TEXT
);

CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_log(timestamp);
CREATE INDEX IF NOT EXISTS idx_audit_type ON audit_log(operation_type);

-- NEW: Watched folders table
CREATE TABLE IF NOT EXISTS watched_folders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    path TEXT NOT NULL UNIQUE,
    active BOOLEAN DEFAULT 1,
    last_scan TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```


### LLM Package

The llm package provides a unified interface for multiple LLM providers.

#### Provider Interface

```go
package llm

import (
    "context"
    "io"
)

// Provider defines the interface for LLM services
type Provider interface {
    // Embed generates an embedding vector for the given text
    Embed(ctx context.Context, text string) ([]float32, error)
    
    // Stream generates a chat completion and streams it to the writer
    Stream(ctx context.Context, messages []Message, w io.Writer) (string, error)
    
    // Name returns the provider name (e.g., "ollama", "openai")
    Name() string
    
    // IsLocal returns true if the provider runs locally
    IsLocal() bool
}

// Message represents a chat message
type Message struct {
    Role    string `json:"role"`    // "system", "user", "assistant"
    Content string `json:"content"`
}

// Config holds provider configuration
type Config struct {
    Type            string // "ollama", "openai", "anthropic"
    OllamaEndpoint  string
    OllamaEmbedModel string
    OllamaChatModel  string
    OpenAIKey       string
    OpenAIEmbedModel string
    OpenAIChatModel  string
    AnthropicKey    string
    AnthropicEmbedModel string
    AnthropicChatModel  string
}

// NewProvider creates a provider based on config
func NewProvider(cfg Config, privacyMode bool) (Provider, error)
```

#### Ollama Provider Implementation

```go
package llm

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
)

type OllamaProvider struct {
    endpoint   string
    embedModel string
    chatModel  string
    client     *http.Client
}

func NewOllamaProvider(endpoint, embedModel, chatModel string) *OllamaProvider {
    return &OllamaProvider{
        endpoint:   endpoint,
        embedModel: embedModel,
        chatModel:  chatModel,
        client:     &http.Client{Timeout: 60 * time.Second},
    }
}

func (p *OllamaProvider) Embed(ctx context.Context, text string) ([]float32, error) {
    reqBody := map[string]interface{}{
        "model":  p.embedModel,
        "prompt": text,
    }
    
    body, _ := json.Marshal(reqBody)
    req, _ := http.NewRequestWithContext(ctx, "POST", p.endpoint+"/api/embeddings", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := p.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("ollama embed request failed: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("ollama embed returned status %d", resp.StatusCode)
    }
    
    var result struct {
        Embedding []float32 `json:"embedding"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("failed to decode ollama response: %w", err)
    }
    
    return result.Embedding, nil
}

func (p *OllamaProvider) Stream(ctx context.Context, messages []Message, w io.Writer) (string, error) {
    reqBody := map[string]interface{}{
        "model":    p.chatModel,
        "messages": messages,
        "stream":   true,
    }
    
    body, _ := json.Marshal(reqBody)
    req, _ := http.NewRequestWithContext(ctx, "POST", p.endpoint+"/api/chat", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := p.client.Do(req)
    if err != nil {
        return "", fmt.Errorf("ollama stream request failed: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != 200 {
        return "", fmt.Errorf("ollama stream returned status %d", resp.StatusCode)
    }
    
    var fullResponse string
    decoder := json.NewDecoder(resp.Body)
    
    for {
        var chunk struct {
            Message struct {
                Content string `json:"content"`
            } `json:"message"`
            Done bool `json:"done"`
        }
        
        if err := decoder.Decode(&chunk); err != nil {
            if err == io.EOF {
                break
            }
            return fullResponse, fmt.Errorf("failed to decode stream: %w", err)
        }
        
        if chunk.Message.Content != "" {
            fullResponse += chunk.Message.Content
            w.Write([]byte(chunk.Message.Content))
        }
        
        if chunk.Done {
            break
        }
    }
    
    return fullResponse, nil
}

func (p *OllamaProvider) Name() string {
    return "ollama"
}

func (p *OllamaProvider) IsLocal() bool {
    return true
}
```


#### OpenAI Provider Implementation

```go
package llm

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
)

type OpenAIProvider struct {
    apiKey     string
    embedModel string
    chatModel  string
    client     *http.Client
}

func NewOpenAIProvider(apiKey, embedModel, chatModel string) *OpenAIProvider {
    return &OpenAIProvider{
        apiKey:     apiKey,
        embedModel: embedModel,
        chatModel:  chatModel,
        client:     &http.Client{Timeout: 60 * time.Second},
    }
}

func (p *OpenAIProvider) Embed(ctx context.Context, text string) ([]float32, error) {
    reqBody := map[string]interface{}{
        "model": p.embedModel,
        "input": text,
    }
    
    body, _ := json.Marshal(reqBody)
    req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/embeddings", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+p.apiKey)
    
    resp, err := p.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("openai embed request failed: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != 200 {
        bodyBytes, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("openai embed returned status %d: %s", resp.StatusCode, string(bodyBytes))
    }
    
    var result struct {
        Data []struct {
            Embedding []float32 `json:"embedding"`
        } `json:"data"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("failed to decode openai response: %w", err)
    }
    
    if len(result.Data) == 0 {
        return nil, fmt.Errorf("openai returned no embeddings")
    }
    
    return result.Data[0].Embedding, nil
}

func (p *OpenAIProvider) Stream(ctx context.Context, messages []Message, w io.Writer) (string, error) {
    reqBody := map[string]interface{}{
        "model":    p.chatModel,
        "messages": messages,
        "stream":   true,
    }
    
    body, _ := json.Marshal(reqBody)
    req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+p.apiKey)
    
    resp, err := p.client.Do(req)
    if err != nil {
        return "", fmt.Errorf("openai stream request failed: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != 200 {
        bodyBytes, _ := io.ReadAll(resp.Body)
        return "", fmt.Errorf("openai stream returned status %d: %s", resp.StatusCode, string(bodyBytes))
    }
    
    var fullResponse string
    scanner := bufio.NewScanner(resp.Body)
    
    for scanner.Scan() {
        line := scanner.Text()
        if !strings.HasPrefix(line, "data: ") {
            continue
        }
        
        data := strings.TrimPrefix(line, "data: ")
        if data == "[DONE]" {
            break
        }
        
        var chunk struct {
            Choices []struct {
                Delta struct {
                    Content string `json:"content"`
                } `json:"delta"`
            } `json:"choices"`
        }
        
        if err := json.Unmarshal([]byte(data), &chunk); err != nil {
            continue
        }
        
        if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
            content := chunk.Choices[0].Delta.Content
            fullResponse += content
            w.Write([]byte(content))
        }
    }
    
    return fullResponse, nil
}

func (p *OpenAIProvider) Name() string {
    return "openai"
}

func (p *OpenAIProvider) IsLocal() bool {
    return false
}
```


#### Anthropic Provider Implementation

```go
package llm

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
)

type AnthropicProvider struct {
    apiKey     string
    embedModel string // Uses Voyage AI for embeddings
    chatModel  string
    client     *http.Client
}

func NewAnthropicProvider(apiKey, embedModel, chatModel string) *AnthropicProvider {
    return &AnthropicProvider{
        apiKey:     apiKey,
        embedModel: embedModel,
        chatModel:  chatModel,
        client:     &http.Client{Timeout: 60 * time.Second},
    }
}

func (p *AnthropicProvider) Embed(ctx context.Context, text string) ([]float32, error) {
    // Anthropic doesn't provide embeddings directly, use Voyage AI
    // This is a placeholder - actual implementation would use Voyage AI API
    return nil, fmt.Errorf("anthropic embeddings not yet implemented - use Voyage AI")
}

func (p *AnthropicProvider) Stream(ctx context.Context, messages []Message, w io.Writer) (string, error) {
    // Convert messages to Anthropic format (system message separate)
    var system string
    var anthropicMessages []map[string]string
    
    for _, msg := range messages {
        if msg.Role == "system" {
            system = msg.Content
        } else {
            anthropicMessages = append(anthropicMessages, map[string]string{
                "role":    msg.Role,
                "content": msg.Content,
            })
        }
    }
    
    reqBody := map[string]interface{}{
        "model":      p.chatModel,
        "messages":   anthropicMessages,
        "max_tokens": 4096,
        "stream":     true,
    }
    
    if system != "" {
        reqBody["system"] = system
    }
    
    body, _ := json.Marshal(reqBody)
    req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("x-api-key", p.apiKey)
    req.Header.Set("anthropic-version", "2023-06-01")
    
    resp, err := p.client.Do(req)
    if err != nil {
        return "", fmt.Errorf("anthropic stream request failed: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != 200 {
        bodyBytes, _ := io.ReadAll(resp.Body)
        return "", fmt.Errorf("anthropic stream returned status %d: %s", resp.StatusCode, string(bodyBytes))
    }
    
    var fullResponse string
    scanner := bufio.NewScanner(resp.Body)
    
    for scanner.Scan() {
        line := scanner.Text()
        if !strings.HasPrefix(line, "data: ") {
            continue
        }
        
        data := strings.TrimPrefix(line, "data: ")
        
        var event struct {
            Type  string `json:"type"`
            Delta struct {
                Type string `json:"type"`
                Text string `json:"text"`
            } `json:"delta"`
        }
        
        if err := json.Unmarshal([]byte(data), &event); err != nil {
            continue
        }
        
        if event.Type == "content_block_delta" && event.Delta.Text != "" {
            fullResponse += event.Delta.Text
            w.Write([]byte(event.Delta.Text))
        }
    }
    
    return fullResponse, nil
}

func (p *AnthropicProvider) Name() string {
    return "anthropic"
}

func (p *AnthropicProvider) IsLocal() bool {
    return false
}
```

#### Privacy Mode Enforcement

```go
package llm

import "fmt"

// NewProvider creates a provider with privacy mode enforcement
func NewProvider(cfg Config, privacyMode bool) (Provider, error) {
    if privacyMode && cfg.Type != "ollama" {
        return nil, fmt.Errorf("privacy mode is enabled - only Ollama provider is allowed")
    }
    
    switch cfg.Type {
    case "ollama":
        return NewOllamaProvider(cfg.OllamaEndpoint, cfg.OllamaEmbedModel, cfg.OllamaChatModel), nil
    case "openai":
        if cfg.OpenAIKey == "" {
            return nil, fmt.Errorf("openai API key is required")
        }
        return NewOpenAIProvider(cfg.OpenAIKey, cfg.OpenAIEmbedModel, cfg.OpenAIChatModel), nil
    case "anthropic":
        if cfg.AnthropicKey == "" {
            return nil, fmt.Errorf("anthropic API key is required")
        }
        return NewAnthropicProvider(cfg.AnthropicKey, cfg.AnthropicEmbedModel, cfg.AnthropicChatModel), nil
    default:
        return nil, fmt.Errorf("unknown provider type: %s", cfg.Type)
    }
}
```


### RAG Package

The RAG package handles text chunking, vector search, and prompt construction.

#### Interface

```go
package rag

import (
    "context"
    "math"
    "strings"
)

// Chunker splits text into overlapping segments
type Chunker struct {
    ChunkSize int // Target characters per chunk (200-500)
    Overlap   int // Overlap between chunks (50)
}

// ChunkText splits text into chunks with overlap
func (c *Chunker) ChunkText(text string) []string {
    var chunks []string
    runes := []rune(text)
    
    for i := 0; i < len(runes); i += c.ChunkSize - c.Overlap {
        end := i + c.ChunkSize
        if end > len(runes) {
            end = len(runes)
        }
        
        chunk := string(runes[i:end])
        chunks = append(chunks, strings.TrimSpace(chunk))
        
        if end == len(runes) {
            break
        }
    }
    
    return chunks
}

// Searcher performs vector similarity search
type Searcher struct {
    store Store // Interface to database
}

// Store interface for RAG operations
type Store interface {
    Search(ctx context.Context, queryVec []float32, topK int) ([]Chunk, error)
}

// Chunk represents a search result
type Chunk struct {
    Source string
    Text   string
    Score  float64
}

// Search finds relevant chunks using cosine similarity
func (s *Searcher) Search(ctx context.Context, queryVec []float32, topK int) ([]Chunk, error) {
    return s.store.Search(ctx, queryVec, topK)
}

// CosineSimilarity computes similarity between two vectors
func CosineSimilarity(a, b []float32) float64 {
    if len(a) != len(b) {
        return 0
    }
    
    var dotProduct, normA, normB float64
    for i := range a {
        dotProduct += float64(a[i] * b[i])
        normA += float64(a[i] * a[i])
        normB += float64(b[i] * b[i])
    }
    
    if normA == 0 || normB == 0 {
        return 0
    }
    
    return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// PromptBuilder constructs prompts with retrieved context
type PromptBuilder struct{}

// BuildPrompt combines query and chunks into a RAG prompt
func (pb *PromptBuilder) BuildPrompt(query string, chunks []Chunk) string {
    var sb strings.Builder
    
    sb.WriteString("You are a helpful assistant. Use the following context to answer the user's question.\n\n")
    sb.WriteString("Context:\n")
    
    for i, chunk := range chunks {
        sb.WriteString(fmt.Sprintf("\n[%d] Source: %s\n%s\n", i+1, chunk.Source, chunk.Text))
    }
    
    sb.WriteString("\n\nUser Question: ")
    sb.WriteString(query)
    sb.WriteString("\n\nAnswer based on the context above:")
    
    return sb.String()
}
```


### Ingest Package

The ingest package handles document parsing, PII detection, and safety guardrails.

#### Interface

```go
package ingest

import (
    "context"
    "fmt"
    "io"
    "mime/multipart"
    "net/http"
    "os"
    "path/filepath"
    "regexp"
    "strings"
)

// Ingester orchestrates document ingestion
type Ingester struct {
    provider      LLMProvider
    store         Store
    chunker       Chunker
    piiDetector   *PIIDetector
    guardrails    *Guardrails
    privacyMode   bool
    summarize     bool
}

// LLMProvider interface for embeddings and summarization
type LLMProvider interface {
    Embed(ctx context.Context, text string) ([]float32, error)
    Stream(ctx context.Context, messages []Message, w io.Writer) (string, error)
}

// Store interface for saving chunks
type Store interface {
    SaveChunk(ctx context.Context, source, text string, embedding []float32, tags []string, summary string) error
}

// IngestText processes plain text
func (ing *Ingester) IngestText(ctx context.Context, source, text string, tags []string) error {
    // Check guardrails
    if err := ing.guardrails.Check(source, text); err != nil {
        return fmt.Errorf("guardrails check failed: %w", err)
    }
    
    // Detect PII
    if piiTypes := ing.piiDetector.Detect(text); len(piiTypes) > 0 {
        return fmt.Errorf("PII detected: %v - ingestion blocked", piiTypes)
    }
    
    // Generate summary if enabled
    var summary string
    if ing.summarize {
        summary, _ = ing.generateSummary(ctx, text)
    }
    
    // Chunk text
    chunks := ing.chunker.ChunkText(text)
    
    // Embed and save each chunk
    for _, chunk := range chunks {
        embedding, err := ing.provider.Embed(ctx, chunk)
        if err != nil {
            return fmt.Errorf("embedding failed: %w", err)
        }
        
        if err := ing.store.SaveChunk(ctx, source, chunk, embedding, tags, summary); err != nil {
            return fmt.Errorf("save chunk failed: %w", err)
        }
    }
    
    return nil
}

// IngestURL fetches and processes a web page
func (ing *Ingester) IngestURL(ctx context.Context, url string, tags []string) error {
    if ing.privacyMode {
        return fmt.Errorf("URL ingestion is disabled in privacy mode")
    }
    
    // Fetch URL content
    resp, err := http.Get(url)
    if err != nil {
        return fmt.Errorf("failed to fetch URL: %w", err)
    }
    defer resp.Body.Close()
    
    // Parse HTML (using go-readability)
    article, err := readability.FromReader(resp.Body, url)
    if err != nil {
        return fmt.Errorf("failed to parse HTML: %w", err)
    }
    
    return ing.IngestText(ctx, url, article.TextContent, tags)
}

// IngestFile processes an uploaded file
func (ing *Ingester) IngestFile(ctx context.Context, file multipart.File, header *multipart.FileHeader, tags []string) error {
    // Check file size
    if header.Size > ing.guardrails.MaxFileSize {
        return fmt.Errorf("file size %d exceeds limit %d", header.Size, ing.guardrails.MaxFileSize)
    }
    
    // Check extension
    ext := strings.ToLower(filepath.Ext(header.Filename))
    if !ing.guardrails.IsAllowedExtension(ext) {
        return fmt.Errorf("file extension %s is not allowed", ext)
    }
    
    // Parse based on MIME type
    var text string
    var err error
    
    switch ext {
    case ".txt", ".md":
        text, err = ing.parseText(file)
    case ".pdf":
        text, err = ing.parsePDF(file)
    default:
        return fmt.Errorf("unsupported file type: %s", ext)
    }
    
    if err != nil {
        return fmt.Errorf("failed to parse file: %w", err)
    }
    
    return ing.IngestText(ctx, header.Filename, text, tags)
}

func (ing *Ingester) parseText(r io.Reader) (string, error) {
    bytes, err := io.ReadAll(r)
    if err != nil {
        return "", err
    }
    return string(bytes), nil
}

func (ing *Ingester) parsePDF(r io.Reader) (string, error) {
    // Use a PDF library like pdfcpu or unidoc
    // Placeholder implementation
    return "", fmt.Errorf("PDF parsing not yet implemented")
}

func (ing *Ingester) generateSummary(ctx context.Context, text string) (string, error) {
    // Take first 1000 chars
    input := text
    if len(input) > 1000 {
        input = input[:1000]
    }
    
    messages := []Message{
        {Role: "user", Content: "Summarize this document in 2-3 sentences:\n\n" + input},
    }
    
    var buf strings.Builder
    summary, err := ing.provider.Stream(ctx, messages, &buf)
    if err != nil {
        return "", err
    }
    
    return summary, nil
}
```


#### PII Detection

```go
package ingest

import (
    "regexp"
    "strings"
)

// PIIDetector scans text for personally identifiable information
type PIIDetector struct {
    patterns map[string]*regexp.Regexp
}

// NewPIIDetector creates a detector with common PII patterns
func NewPIIDetector() *PIIDetector {
    return &PIIDetector{
        patterns: map[string]*regexp.Regexp{
            "ssn":         regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
            "credit_card": regexp.MustCompile(`\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`),
            "api_key":     regexp.MustCompile(`\b(sk-[a-zA-Z0-9]{32,}|ghp_[a-zA-Z0-9]{36}|xox[baprs]-[a-zA-Z0-9-]+)\b`),
            "private_key": regexp.MustCompile(`-----BEGIN (RSA |EC |OPENSSH )?PRIVATE KEY-----`),
            "email":       regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`),
            "phone":       regexp.MustCompile(`\b\d{3}[-.]?\d{3}[-.]?\d{4}\b`),
        },
    }
}

// Detect returns a list of PII types found in the text
func (d *PIIDetector) Detect(text string) []string {
    var found []string
    
    for piiType, pattern := range d.patterns {
        if pattern.MatchString(text) {
            found = append(found, piiType)
        }
    }
    
    return found
}
```

#### Guardrails

```go
package ingest

import (
    "fmt"
    "path/filepath"
    "strings"
)

// Guardrails enforces safety checks on ingestion
type Guardrails struct {
    MaxFileSize        int64
    AllowedExtensions  []string
    BlockedExtensions  []string
    SensitiveFilenames []string
    MaxConcurrent      int
}

// NewGuardrails creates guardrails with safe defaults
func NewGuardrails() *Guardrails {
    return &Guardrails{
        MaxFileSize: 10 * 1024 * 1024, // 10MB
        AllowedExtensions: []string{".txt", ".md", ".pdf", ".html"},
        BlockedExtensions: []string{
            ".exe", ".dll", ".so", ".dylib", ".app",
            ".zip", ".tar", ".gz", ".rar",
            ".iso", ".dmg", ".img",
        },
        SensitiveFilenames: []string{
            ".env", "id_rsa", "id_ed25519", "credentials.json",
            ".aws/credentials", ".ssh/id_rsa",
        },
        MaxConcurrent: 3,
    }
}

// Check validates a file for ingestion
func (g *Guardrails) Check(filename, content string) error {
    // Check sensitive filenames
    for _, sensitive := range g.SensitiveFilenames {
        if strings.Contains(strings.ToLower(filename), strings.ToLower(sensitive)) {
            return fmt.Errorf("sensitive filename detected: %s", filename)
        }
    }
    
    // Check blocked extensions
    ext := strings.ToLower(filepath.Ext(filename))
    for _, blocked := range g.BlockedExtensions {
        if ext == blocked {
            return fmt.Errorf("blocked file extension: %s", ext)
        }
    }
    
    return nil
}

// IsAllowedExtension checks if a file extension is allowed
func (g *Guardrails) IsAllowedExtension(ext string) bool {
    ext = strings.ToLower(ext)
    for _, allowed := range g.AllowedExtensions {
        if ext == allowed {
            return true
        }
    }
    return false
}
```


### API Package

The API package provides HTTP handlers and WebSocket support.

#### Server Structure

```go
package api

import (
    "context"
    "html/template"
    "net/http"
    "sync"
)

// Server holds dependencies and provides HTTP handlers
type Server struct {
    store      Store
    provider   LLMProvider
    ingester   *Ingester
    searcher   *Searcher
    wsHub      *WebSocketHub
    templates  *template.Template
    config     *Config
}

// Store interface for API operations
type Store interface {
    SaveChunk(ctx context.Context, source, text string, embedding []float32, tags []string, summary string) error
    Search(ctx context.Context, queryVec []float32, topK int) ([]Chunk, error)
    Library(ctx context.Context) ([]LibraryEntry, error)
    DeleteSource(ctx context.Context, source string) error
    SaveMessage(ctx context.Context, sessionID, role, content string) error
    GetSessionHistory(ctx context.Context, sessionID string) ([]ChatMessage, error)
    ListSessions(ctx context.Context) ([]Session, error)
    AddAuditEntry(ctx context.Context, opType, details, userCtx string) error
}

// LLMProvider interface for chat and embeddings
type LLMProvider interface {
    Embed(ctx context.Context, text string) ([]float32, error)
    Stream(ctx context.Context, messages []Message, w io.Writer) (string, error)
    Name() string
    IsLocal() bool
}

// Config holds server configuration
type Config struct {
    PrivacyMode bool
    Provider    string
}

// NewServer creates a server with dependencies
func NewServer(store Store, provider LLMProvider, ingester *Ingester, searcher *Searcher, config *Config) (*Server, error) {
    // Load templates
    tmpl, err := template.ParseGlob("web/templates/*.html")
    if err != nil {
        return nil, fmt.Errorf("failed to load templates: %w", err)
    }
    
    srv := &Server{
        store:     store,
        provider:  provider,
        ingester:  ingester,
        searcher:  searcher,
        wsHub:     NewWebSocketHub(),
        templates: tmpl,
        config:    config,
    }
    
    // Start WebSocket hub
    go srv.wsHub.Run()
    
    return srv, nil
}

// RegisterRoutes sets up all HTTP routes
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
    // Static files
    mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
    
    // Page routes
    mux.HandleFunc("/", s.handleDashboard)
    mux.HandleFunc("/chat", s.handleChat)
    mux.HandleFunc("/library", s.handleLibrary)
    mux.HandleFunc("/settings", s.handleSettings)
    
    // API routes
    mux.HandleFunc("/api/ask", s.handleAsk)
    mux.HandleFunc("/api/ingest/text", s.handleIngestText)
    mux.HandleFunc("/api/ingest/url", s.handleIngestURL)
    mux.HandleFunc("/api/ingest/file", s.handleIngestFile)
    mux.HandleFunc("/api/delete", s.handleDelete)
    mux.HandleFunc("/api/sessions", s.handleSessions)
    mux.HandleFunc("/api/session/", s.handleSessionHistory)
    mux.HandleFunc("/api/config", s.handleConfig)
    
    // WebSocket
    mux.HandleFunc("/ws", s.handleWebSocket)
}
```

#### Key Handlers

```go
// handleAsk processes chat queries with RAG
func (s *Server) handleAsk(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    ctx := r.Context()
    
    // Parse request
    var req struct {
        Query     string `json:"query"`
        SessionID string `json:"session_id"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    // Save user message
    if err := s.store.SaveMessage(ctx, req.SessionID, "user", req.Query); err != nil {
        log.Printf("Failed to save user message: %v", err)
    }
    
    // Audit log
    s.store.AddAuditEntry(ctx, "query", req.Query, req.SessionID)
    
    // Embed query
    queryVec, err := s.provider.Embed(ctx, req.Query)
    if err != nil {
        http.Error(w, "Embedding failed", http.StatusInternalServerError)
        return
    }
    
    // Search for relevant chunks
    chunks, err := s.searcher.Search(ctx, queryVec, 5)
    if err != nil {
        http.Error(w, "Search failed", http.StatusInternalServerError)
        return
    }
    
    // Build prompt
    prompt := buildPrompt(req.Query, chunks)
    
    // Stream response
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    
    messages := []Message{
        {Role: "system", Content: "You are a helpful assistant."},
        {Role: "user", Content: prompt},
    }
    
    response, err := s.provider.Stream(ctx, messages, w)
    if err != nil {
        log.Printf("Stream failed: %v", err)
        return
    }
    
    // Save assistant message
    if err := s.store.SaveMessage(ctx, req.SessionID, "assistant", response); err != nil {
        log.Printf("Failed to save assistant message: %v", err)
    }
}
```


#### WebSocket Hub

```go
package api

import (
    "encoding/json"
    "log"
    "sync"
    
    "github.com/gorilla/websocket"
)

// WebSocketHub manages WebSocket connections
type WebSocketHub struct {
    clients    map[*websocket.Conn]bool
    broadcast  chan []byte
    register   chan *websocket.Conn
    unregister chan *websocket.Conn
    mu         sync.RWMutex
}

// NewWebSocketHub creates a hub
func NewWebSocketHub() *WebSocketHub {
    return &WebSocketHub{
        clients:    make(map[*websocket.Conn]bool),
        broadcast:  make(chan []byte, 256),
        register:   make(chan *websocket.Conn),
        unregister: make(chan *websocket.Conn),
    }
}

// Run starts the hub's event loop
func (h *WebSocketHub) Run() {
    for {
        select {
        case conn := <-h.register:
            h.mu.Lock()
            h.clients[conn] = true
            h.mu.Unlock()
            
        case conn := <-h.unregister:
            h.mu.Lock()
            if _, ok := h.clients[conn]; ok {
                delete(h.clients, conn)
                conn.Close()
            }
            h.mu.Unlock()
            
        case message := <-h.broadcast:
            h.mu.RLock()
            for conn := range h.clients {
                if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
                    log.Printf("WebSocket write error: %v", err)
                    conn.Close()
                    delete(h.clients, conn)
                }
            }
            h.mu.RUnlock()
        }
    }
}

// Broadcast sends a message to all connected clients
func (h *WebSocketHub) Broadcast(eventType, message string) {
    data := map[string]string{
        "type":    eventType,
        "message": message,
    }
    
    jsonData, _ := json.Marshal(data)
    h.broadcast <- jsonData
}

// handleWebSocket upgrades HTTP to WebSocket
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
    upgrader := websocket.Upgrader{
        CheckOrigin: func(r *http.Request) bool {
            return true // In production, validate origin
        },
    }
    
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Printf("WebSocket upgrade failed: %v", err)
        return
    }
    
    s.wsHub.register <- conn
    
    // Read loop (handle client messages if needed)
    go func() {
        defer func() {
            s.wsHub.unregister <- conn
        }()
        
        for {
            _, _, err := conn.ReadMessage()
            if err != nil {
                break
            }
        }
    }()
}
```


### Skills Package

The skills package implements a plugin system for user-defined extensions.

#### Architecture

```go
package skills

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "os"
    "os/exec"
    "path/filepath"
    "time"
)

// Skill represents a loaded skill
type Skill struct {
    Name        string
    Version     string
    Description string
    Executable  string
    Triggers    []Trigger
    Settings    map[string]interface{}
    Timeout     time.Duration
    RequiresNet bool
    Path        string
}

// Trigger defines when a skill executes
type Trigger struct {
    Type       string                 // "manual", "timer", "keyword", "event"
    Parameters map[string]interface{} // Trigger-specific config
}

// Metadata is the skill.json structure
type Metadata struct {
    Name           string                 `json:"name"`
    Version        string                 `json:"version"`
    Description    string                 `json:"description"`
    Executable     string                 `json:"executable"`
    Triggers       []Trigger              `json:"triggers"`
    SettingsSchema map[string]interface{} `json:"settings_schema"`
    Timeout        int                    `json:"timeout"` // seconds
    RequiresNet    bool                   `json:"requires_network"`
}

// Loader discovers and loads skills
type Loader struct {
    skillsDir   string
    privacyMode bool
}

// NewLoader creates a skill loader
func NewLoader(skillsDir string, privacyMode bool) *Loader {
    return &Loader{
        skillsDir:   skillsDir,
        privacyMode: privacyMode,
    }
}

// LoadAll discovers and loads all skills
func (l *Loader) LoadAll() ([]*Skill, error) {
    var skills []*Skill
    
    entries, err := os.ReadDir(l.skillsDir)
    if err != nil {
        return nil, fmt.Errorf("failed to read skills directory: %w", err)
    }
    
    for _, entry := range entries {
        if !entry.IsDir() {
            continue
        }
        
        skillPath := filepath.Join(l.skillsDir, entry.Name())
        skill, err := l.loadSkill(skillPath)
        if err != nil {
            log.Printf("Failed to load skill %s: %v", entry.Name(), err)
            continue
        }
        
        // Skip network-requiring skills in privacy mode
        if l.privacyMode && skill.RequiresNet {
            log.Printf("Skipping skill %s: requires network but privacy mode is enabled", skill.Name)
            continue
        }
        
        skills = append(skills, skill)
    }
    
    return skills, nil
}

// loadSkill loads a single skill from a directory
func (l *Loader) loadSkill(path string) (*Skill, error) {
    metadataPath := filepath.Join(path, "skill.json")
    
    data, err := os.ReadFile(metadataPath)
    if err != nil {
        return nil, fmt.Errorf("failed to read skill.json: %w", err)
    }
    
    var meta Metadata
    if err := json.Unmarshal(data, &meta); err != nil {
        return nil, fmt.Errorf("failed to parse skill.json: %w", err)
    }
    
    // Validate required fields
    if meta.Name == "" || meta.Executable == "" {
        return nil, fmt.Errorf("skill.json missing required fields")
    }
    
    // Check executable exists
    execPath := filepath.Join(path, meta.Executable)
    if _, err := os.Stat(execPath); err != nil {
        return nil, fmt.Errorf("executable not found: %s", execPath)
    }
    
    timeout := time.Duration(meta.Timeout) * time.Second
    if timeout == 0 {
        timeout = 30 * time.Second
    }
    
    return &Skill{
        Name:        meta.Name,
        Version:     meta.Version,
        Description: meta.Description,
        Executable:  execPath,
        Triggers:    meta.Triggers,
        Timeout:     timeout,
        RequiresNet: meta.RequiresNet,
        Path:        path,
    }, nil
}
```


#### Skill Executor

```go
package skills

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "os/exec"
    "time"
)

// Executor runs skills as subprocesses
type Executor struct {
    privacyMode bool
}

// NewExecutor creates a skill executor
func NewExecutor(privacyMode bool) *Executor {
    return &Executor{
        privacyMode: privacyMode,
    }
}

// Input is the JSON sent to skill stdin
type Input struct {
    Query    string                 `json:"query"`
    Context  map[string]interface{} `json:"context"`
    Settings map[string]interface{} `json:"settings"`
}

// Output is the JSON received from skill stdout
type Output struct {
    Result   string                 `json:"result"`
    Error    string                 `json:"error"`
    Metadata map[string]interface{} `json:"metadata"`
}

// Execute runs a skill with the given input
func (e *Executor) Execute(ctx context.Context, skill *Skill, input Input) (*Output, error) {
    // Create context with timeout
    ctx, cancel := context.WithTimeout(ctx, skill.Timeout)
    defer cancel()
    
    // Prepare command
    cmd := exec.CommandContext(ctx, skill.Executable)
    cmd.Dir = skill.Path
    
    // Set environment variables
    cmd.Env = e.buildEnv(skill)
    
    // Prepare input JSON
    inputJSON, err := json.Marshal(input)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal input: %w", err)
    }
    
    cmd.Stdin = bytes.NewReader(inputJSON)
    
    // Capture output
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr
    
    // Run command
    err = cmd.Run()
    
    // Check for timeout
    if ctx.Err() == context.DeadlineExceeded {
        return nil, fmt.Errorf("skill execution timed out after %v", skill.Timeout)
    }
    
    // Parse output
    var output Output
    if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
        return nil, fmt.Errorf("failed to parse skill output: %w (stderr: %s)", err, stderr.String())
    }
    
    if output.Error != "" {
        return &output, fmt.Errorf("skill error: %s", output.Error)
    }
    
    return &output, nil
}

// buildEnv creates environment variables for the skill
func (e *Executor) buildEnv(skill *Skill) []string {
    env := []string{
        "PATH=" + os.Getenv("PATH"),
        "HOME=" + os.Getenv("HOME"),
        "USER=" + os.Getenv("USER"),
        "NOODEXX_SKILL_NAME=" + skill.Name,
        "NOODEXX_SKILL_VERSION=" + skill.Version,
    }
    
    if e.privacyMode {
        env = append(env, "NOODEXX_PRIVACY_MODE=true")
    }
    
    // Add skill-specific settings as env vars
    for key, value := range skill.Settings {
        env = append(env, fmt.Sprintf("NOODEXX_SETTING_%s=%v", strings.ToUpper(key), value))
    }
    
    return env
}
```

#### Example skill.json

```json
{
  "name": "weather",
  "version": "1.0.0",
  "description": "Fetches current weather for a location",
  "executable": "weather.sh",
  "triggers": [
    {
      "type": "manual"
    },
    {
      "type": "keyword",
      "parameters": {
        "keywords": ["weather", "forecast", "temperature"]
      }
    }
  ],
  "settings_schema": {
    "default_location": {
      "type": "string",
      "default": "San Francisco"
    }
  },
  "timeout": 10,
  "requires_network": true
}
```

#### Example Skill Script (weather.sh)

```bash
#!/bin/bash

# Read JSON input from stdin
INPUT=$(cat)

# Parse query from JSON
QUERY=$(echo "$INPUT" | jq -r '.query')
LOCATION=$(echo "$INPUT" | jq -r '.settings.default_location // "San Francisco"')

# Check privacy mode
if [ "$NOODEXX_PRIVACY_MODE" = "true" ]; then
    echo '{"error": "Weather skill requires network access"}'
    exit 1
fi

# Fetch weather
WEATHER=$(curl -s "https://wttr.in/${LOCATION}?format=3")

# Return JSON output
echo "{\"result\": \"$WEATHER\", \"metadata\": {\"location\": \"$LOCATION\"}}"
```


### Configuration System

The config package manages application configuration with JSON and environment variable support.

#### Configuration Structure

```go
package config

import (
    "encoding/json"
    "fmt"
    "os"
    "strings"
)

// Config holds all application configuration
type Config struct {
    Provider   ProviderConfig   `json:"provider"`
    Privacy    PrivacyConfig    `json:"privacy"`
    Folders    []string         `json:"folders"`
    Logging    LoggingConfig    `json:"logging"`
    Guardrails GuardrailsConfig `json:"guardrails"`
    Server     ServerConfig     `json:"server"`
}

// ProviderConfig configures the LLM provider
type ProviderConfig struct {
    Type            string `json:"type"` // "ollama", "openai", "anthropic"
    OllamaEndpoint  string `json:"ollama_endpoint"`
    OllamaEmbedModel string `json:"ollama_embed_model"`
    OllamaChatModel  string `json:"ollama_chat_model"`
    OpenAIKey       string `json:"openai_key"`
    OpenAIEmbedModel string `json:"openai_embed_model"`
    OpenAIChatModel  string `json:"openai_chat_model"`
    AnthropicKey    string `json:"anthropic_key"`
    AnthropicEmbedModel string `json:"anthropic_embed_model"`
    AnthropicChatModel  string `json:"anthropic_chat_model"`
}

// PrivacyConfig controls privacy mode
type PrivacyConfig struct {
    Enabled bool `json:"enabled"`
}

// LoggingConfig controls logging behavior
type LoggingConfig struct {
    Level      string `json:"level"` // "debug", "info", "warn", "error"
    File       string `json:"file"`  // Optional log file path
    MaxSizeMB  int    `json:"max_size_mb"`
    MaxBackups int    `json:"max_backups"`
}

// GuardrailsConfig controls ingestion safety
type GuardrailsConfig struct {
    MaxFileSizeMB    int      `json:"max_file_size_mb"`
    AllowedExtensions []string `json:"allowed_extensions"`
    MaxConcurrent    int      `json:"max_concurrent"`
    PIIDetection     string   `json:"pii_detection"` // "strict", "normal", "off"
    AutoSummarize    bool     `json:"auto_summarize"`
}

// ServerConfig controls HTTP server
type ServerConfig struct {
    Port        int    `json:"port"`
    BindAddress string `json:"bind_address"`
}

// Load reads configuration from file and environment
func Load(path string) (*Config, error) {
    // Default configuration
    cfg := &Config{
        Provider: ProviderConfig{
            Type:             "ollama",
            OllamaEndpoint:   "http://localhost:11434",
            OllamaEmbedModel: "nomic-embed-text",
            OllamaChatModel:  "llama3.2",
        },
        Privacy: PrivacyConfig{
            Enabled: true,
        },
        Logging: LoggingConfig{
            Level: "info",
        },
        Guardrails: GuardrailsConfig{
            MaxFileSizeMB:     10,
            AllowedExtensions: []string{".txt", ".md", ".pdf", ".html"},
            MaxConcurrent:     3,
            PIIDetection:      "normal",
            AutoSummarize:     true,
        },
        Server: ServerConfig{
            Port:        8080,
            BindAddress: "127.0.0.1",
        },
    }
    
    // Load from file if exists
    if _, err := os.Stat(path); err == nil {
        data, err := os.ReadFile(path)
        if err != nil {
            return nil, fmt.Errorf("failed to read config file: %w", err)
        }
        
        if err := json.Unmarshal(data, cfg); err != nil {
            return nil, fmt.Errorf("failed to parse config file: %w", err)
        }
    } else {
        // Create default config file
        if err := cfg.Save(path); err != nil {
            return nil, fmt.Errorf("failed to create default config: %w", err)
        }
    }
    
    // Override with environment variables
    cfg.applyEnvOverrides()
    
    // Validate
    if err := cfg.Validate(); err != nil {
        return nil, fmt.Errorf("invalid configuration: %w", err)
    }
    
    return cfg, nil
}

// Save writes configuration to file
func (c *Config) Save(path string) error {
    data, err := json.MarshalIndent(c, "", "  ")
    if err != nil {
        return err
    }
    
    return os.WriteFile(path, data, 0600)
}

// applyEnvOverrides applies environment variable overrides
func (c *Config) applyEnvOverrides() {
    if v := os.Getenv("NOODEXX_PROVIDER"); v != "" {
        c.Provider.Type = v
    }
    if v := os.Getenv("NOODEXX_OPENAI_KEY"); v != "" {
        c.Provider.OpenAIKey = v
    }
    if v := os.Getenv("NOODEXX_ANTHROPIC_KEY"); v != "" {
        c.Provider.AnthropicKey = v
    }
    if v := os.Getenv("NOODEXX_PRIVACY_MODE"); v == "true" {
        c.Privacy.Enabled = true
    }
    if v := os.Getenv("NOODEXX_LOG_LEVEL"); v != "" {
        c.Logging.Level = v
    }
}

// Validate checks configuration validity
func (c *Config) Validate() error {
    // Privacy mode validation
    if c.Privacy.Enabled && c.Provider.Type != "ollama" {
        return fmt.Errorf("privacy mode requires Ollama provider")
    }
    
    // Provider validation
    switch c.Provider.Type {
    case "ollama":
        if !strings.HasPrefix(c.Provider.OllamaEndpoint, "http://localhost") &&
           !strings.HasPrefix(c.Provider.OllamaEndpoint, "http://127.0.0.1") {
            if c.Privacy.Enabled {
                return fmt.Errorf("privacy mode requires localhost Ollama endpoint")
            }
        }
    case "openai":
        if c.Provider.OpenAIKey == "" {
            return fmt.Errorf("OpenAI API key is required")
        }
    case "anthropic":
        if c.Provider.AnthropicKey == "" {
            return fmt.Errorf("Anthropic API key is required")
        }
    default:
        return fmt.Errorf("unknown provider type: %s", c.Provider.Type)
    }
    
    // Server validation
    if c.Server.Port < 1024 && os.Geteuid() != 0 {
        return fmt.Errorf("privileged port %d requires root", c.Server.Port)
    }
    
    return nil
}
```


### Logging System

The logging package provides structured logging with levels and components.

```go
package logging

import (
    "fmt"
    "io"
    "log"
    "os"
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
}

// NewLogger creates a logger for a component
func NewLogger(component string, level Level, output io.Writer) *Logger {
    return &Logger{
        level:     level,
        component: component,
        output:    output,
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

// log writes a log entry
func (l *Logger) log(level Level, format string, args ...interface{}) {
    if level < l.level {
        return
    }
    
    timestamp := time.Now().Format("2006-01-02 15:04:05")
    message := fmt.Sprintf(format, args...)
    
    logLine := fmt.Sprintf("[%s] %s [%s] %s\n", timestamp, level, l.component, message)
    l.output.Write([]byte(logLine))
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
```


### Folder Watching System

The watcher package monitors directories for file changes using fsnotify.

```go
package watcher

import (
    "context"
    "log"
    "path/filepath"
    "strings"
    "time"
    
    "github.com/fsnotify/fsnotify"
)

// Watcher monitors folders for file changes
type Watcher struct {
    fsWatcher *fsnotify.Watcher
    ingester  Ingester
    store     Store
    folders   []string
    privacyMode bool
    allowedExts []string
    maxSize     int64
}

// Ingester interface for processing files
type Ingester interface {
    IngestFile(ctx context.Context, path string, tags []string) error
}

// Store interface for folder management
type Store interface {
    AddWatchedFolder(ctx context.Context, path string) error
    GetWatchedFolders(ctx context.Context) ([]WatchedFolder, error)
}

// WatchedFolder represents a monitored directory
type WatchedFolder struct {
    Path     string
    Active   bool
    LastScan time.Time
}

// NewWatcher creates a folder watcher
func NewWatcher(ingester Ingester, store Store, privacyMode bool) (*Watcher, error) {
    fsw, err := fsnotify.NewWatcher()
    if err != nil {
        return nil, err
    }
    
    return &Watcher{
        fsWatcher:   fsw,
        ingester:    ingester,
        store:       store,
        privacyMode: privacyMode,
        allowedExts: []string{".txt", ".md", ".pdf"},
        maxSize:     10 * 1024 * 1024, // 10MB
    }, nil
}

// Start begins watching configured folders
func (w *Watcher) Start(ctx context.Context) error {
    // Load watched folders from database
    folders, err := w.store.GetWatchedFolders(ctx)
    if err != nil {
        return err
    }
    
    // Add each folder to fsnotify
    for _, folder := range folders {
        if !folder.Active {
            continue
        }
        
        if err := w.validatePath(folder.Path); err != nil {
            log.Printf("Skipping invalid folder %s: %v", folder.Path, err)
            continue
        }
        
        if err := w.fsWatcher.Add(folder.Path); err != nil {
            log.Printf("Failed to watch folder %s: %v", folder.Path, err)
            continue
        }
        
        log.Printf("Watching folder: %s", folder.Path)
    }
    
    // Start event loop
    go w.eventLoop(ctx)
    
    return nil
}

// eventLoop processes filesystem events
func (w *Watcher) eventLoop(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            w.fsWatcher.Close()
            return
            
        case event, ok := <-w.fsWatcher.Events:
            if !ok {
                return
            }
            
            w.handleEvent(ctx, event)
            
        case err, ok := <-w.fsWatcher.Errors:
            if !ok {
                return
            }
            log.Printf("Watcher error: %v", err)
        }
    }
}

// handleEvent processes a single filesystem event
func (w *Watcher) handleEvent(ctx context.Context, event fsnotify.Event) {
    // Check if it's a file we care about
    if !w.shouldProcess(event.Name) {
        return
    }
    
    switch {
    case event.Op&fsnotify.Create == fsnotify.Create:
        log.Printf("File created: %s", event.Name)
        w.ingestFile(ctx, event.Name)
        
    case event.Op&fsnotify.Write == fsnotify.Write:
        log.Printf("File modified: %s", event.Name)
        w.ingestFile(ctx, event.Name)
        
    case event.Op&fsnotify.Remove == fsnotify.Remove:
        log.Printf("File deleted: %s", event.Name)
        // TODO: Remove from database
    }
}

// shouldProcess checks if a file should be processed
func (w *Watcher) shouldProcess(path string) bool {
    // Check extension
    ext := strings.ToLower(filepath.Ext(path))
    allowed := false
    for _, allowedExt := range w.allowedExts {
        if ext == allowedExt {
            allowed = true
            break
        }
    }
    
    if !allowed {
        return false
    }
    
    // Check file size
    info, err := os.Stat(path)
    if err != nil {
        return false
    }
    
    if info.Size() > w.maxSize {
        log.Printf("File %s exceeds size limit", path)
        return false
    }
    
    return true
}

// ingestFile processes a file
func (w *Watcher) ingestFile(ctx context.Context, path string) {
    tags := []string{"auto-ingested"}
    
    if err := w.ingester.IngestFile(ctx, path, tags); err != nil {
        log.Printf("Failed to ingest %s: %v", path, err)
    } else {
        log.Printf("Successfully ingested %s", path)
    }
}

// validatePath ensures a path is safe to watch
func (w *Watcher) validatePath(path string) error {
    // Block system directories
    systemDirs := []string{"/etc", "/System", "/Windows", "/sys", "/proc"}
    for _, sysDir := range systemDirs {
        if strings.HasPrefix(path, sysDir) {
            return fmt.Errorf("cannot watch system directory: %s", path)
        }
    }
    
    // Ensure path exists
    if _, err := os.Stat(path); err != nil {
        return fmt.Errorf("path does not exist: %s", path)
    }
    
    return nil
}

// AddFolder adds a new folder to watch
func (w *Watcher) AddFolder(ctx context.Context, path string) error {
    if err := w.validatePath(path); err != nil {
        return err
    }
    
    if err := w.fsWatcher.Add(path); err != nil {
        return err
    }
    
    return w.store.AddWatchedFolder(ctx, path)
}
```


## Data Models

### Core Data Structures

```go
// Chunk represents a text segment with embedding
type Chunk struct {
    ID        int64
    Source    string
    Text      string
    Embedding []float32
    Tags      []string
    Summary   string
    CreatedAt time.Time
}

// LibraryEntry represents a document in the library
type LibraryEntry struct {
    Source     string
    ChunkCount int
    Summary    string
    Tags       []string
    CreatedAt  time.Time
}

// ChatMessage represents a chat message
type ChatMessage struct {
    ID        int64
    SessionID string
    Role      string // "user" or "assistant"
    Content   string
    CreatedAt time.Time
}

// Session represents a chat session
type Session struct {
    ID            string
    LastMessageAt time.Time
    MessageCount  int
}

// AuditEntry represents an audit log entry
type AuditEntry struct {
    ID            int64
    Timestamp     time.Time
    OperationType string // "ingest", "query", "delete", "config"
    Details       string
    UserContext   string
}
```

## UI Architecture

### Template Structure

The UI uses Go's html/template with a base layout and page-specific templates.

#### Base Template (base.html)

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}} - Noodexx</title>
    <link rel="stylesheet" href="/static/style.css">
    <script src="/static/htmx.min.js"></script>
</head>
<body>
    <!-- Sidebar -->
    <aside id="sidebar" class="sidebar">
        <div class="sidebar-header">
            <h1>Noodexx</h1>
            <button id="sidebar-toggle" class="sidebar-toggle">☰</button>
        </div>
        
        <nav class="sidebar-nav">
            <a href="/" class="nav-item {{if eq .Page "dashboard"}}active{{end}}">
                <span class="icon">📊</span>
                <span class="label">Dashboard</span>
            </a>
            <a href="/chat" class="nav-item {{if eq .Page "chat"}}active{{end}}">
                <span class="icon">💬</span>
                <span class="label">Chat</span>
            </a>
            <a href="/library" class="nav-item {{if eq .Page "library"}}active{{end}}">
                <span class="icon">📚</span>
                <span class="label">Library</span>
            </a>
            <a href="/settings" class="nav-item {{if eq .Page "settings"}}active{{end}}">
                <span class="icon">⚙️</span>
                <span class="label">Settings</span>
            </a>
        </nav>
        
        {{if .PrivacyMode}}
        <div class="privacy-badge">
            🔒 Privacy Mode
        </div>
        {{end}}
    </aside>
    
    <!-- Main Content -->
    <main id="main-content" class="main-content">
        {{template "content" .}}
    </main>
    
    <!-- Toast Container -->
    <div id="toast-container" class="toast-container"></div>
    
    <!-- Command Palette -->
    <div id="command-palette" class="command-palette hidden">
        <input type="text" id="command-input" placeholder="Type a command...">
        <div id="command-results"></div>
    </div>
    
    <script src="/static/app.js"></script>
    <script>
        // WebSocket connection
        const ws = new WebSocket('ws://' + window.location.host + '/ws');
        
        ws.onmessage = function(event) {
            const data = JSON.parse(event.data);
            showToast(data.type, data.message);
            
            // Refresh library if on library page
            if (data.type === 'ingest_complete' && window.location.pathname === '/library') {
                htmx.trigger('#library-grid', 'refresh');
            }
        };
    </script>
</body>
</html>
```


#### Dashboard Template (dashboard.html)

```html
{{define "content"}}
<div class="dashboard">
    <h1>Dashboard</h1>
    
    <!-- Stats Cards -->
    <div class="stats-grid">
        <div class="stat-card">
            <div class="stat-icon">📄</div>
            <div class="stat-value">{{.DocumentCount}}</div>
            <div class="stat-label">Documents Indexed</div>
        </div>
        
        <div class="stat-card">
            <div class="stat-icon">🤖</div>
            <div class="stat-value">{{.Provider}}</div>
            <div class="stat-label">LLM Provider</div>
        </div>
        
        <div class="stat-card">
            <div class="stat-icon">⏰</div>
            <div class="stat-value">{{.LastIngestion}}</div>
            <div class="stat-label">Last Ingestion</div>
        </div>
        
        {{if .PrivacyMode}}
        <div class="stat-card privacy">
            <div class="stat-icon">🔒</div>
            <div class="stat-value">Active</div>
            <div class="stat-label">Privacy Mode</div>
        </div>
        {{end}}
    </div>
    
    <!-- Quick Actions -->
    <div class="quick-actions">
        <a href="/chat" class="btn btn-primary">Start New Chat</a>
        <a href="/library" class="btn btn-secondary">Browse Library</a>
    </div>
    
    <!-- Recent Activity -->
    <div class="activity-feed" 
         hx-get="/api/activity" 
         hx-trigger="load, every 30s"
         hx-swap="innerHTML">
        <h2>Recent Activity</h2>
        <div class="loading">Loading...</div>
    </div>
</div>
{{end}}
```

#### Chat Template (chat.html)

```html
{{define "content"}}
<div class="chat-container">
    <!-- Session Sidebar -->
    <aside class="session-sidebar">
        <button class="btn btn-primary new-chat-btn" onclick="newChat()">
            + New Chat
        </button>
        
        <div class="session-list" 
             hx-get="/api/sessions" 
             hx-trigger="load"
             hx-swap="innerHTML">
            <!-- Sessions loaded via HTMX -->
        </div>
    </aside>
    
    <!-- Chat Area -->
    <div class="chat-area">
        <div id="messages" class="messages">
            {{range .Messages}}
            <div class="message {{.Role}}">
                <div class="message-content">
                    {{if eq .Role "assistant"}}
                        {{.ContentHTML}}
                    {{else}}
                        {{.Content}}
                    {{end}}
                </div>
                {{if eq .Role "assistant"}}
                <div class="message-actions">
                    <button class="btn-icon" onclick="copyMessage(this)">📋</button>
                </div>
                {{end}}
            </div>
            {{end}}
        </div>
        
        <form id="chat-form" class="chat-input-form" onsubmit="sendMessage(event)">
            <textarea 
                id="query-input" 
                placeholder="Ask a question..." 
                rows="3"
                autofocus></textarea>
            <button type="submit" class="btn btn-primary">Send</button>
        </form>
    </div>
</div>

<script>
let currentSessionId = '{{.SessionID}}';

function sendMessage(event) {
    event.preventDefault();
    const input = document.getElementById('query-input');
    const query = input.value.trim();
    
    if (!query) return;
    
    // Add user message to UI
    addMessage('user', query);
    input.value = '';
    
    // Stream response
    fetch('/api/ask', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({query, session_id: currentSessionId})
    }).then(response => {
        const reader = response.body.getReader();
        const decoder = new TextDecoder();
        let assistantDiv = addMessage('assistant', '');
        
        function read() {
            reader.read().then(({done, value}) => {
                if (done) return;
                const text = decoder.decode(value);
                assistantDiv.querySelector('.message-content').innerHTML += text;
                read();
            });
        }
        read();
    });
}

function addMessage(role, content) {
    const messagesDiv = document.getElementById('messages');
    const messageDiv = document.createElement('div');
    messageDiv.className = `message ${role}`;
    messageDiv.innerHTML = `<div class="message-content">${content}</div>`;
    messagesDiv.appendChild(messageDiv);
    messagesDiv.scrollTop = messagesDiv.scrollHeight;
    return messageDiv;
}

function copyMessage(btn) {
    const content = btn.closest('.message').querySelector('.message-content').textContent;
    navigator.clipboard.writeText(content);
    showToast('success', 'Copied to clipboard');
}

function newChat() {
    currentSessionId = Date.now().toString();
    document.getElementById('messages').innerHTML = '';
}
</script>
{{end}}
```


#### Library Template (library.html)

```html
{{define "content"}}
<div class="library">
    <div class="library-header">
        <h1>Library</h1>
        
        <!-- Tag Filter -->
        <select id="tag-filter" onchange="filterByTag(this.value)">
            <option value="">All Tags</option>
            {{range .Tags}}
            <option value="{{.}}">{{.}}</option>
            {{end}}
        </select>
    </div>
    
    <!-- Drop Zone -->
    <div id="drop-zone" class="drop-zone hidden">
        <div class="drop-zone-content">
            <p>Drop files here to ingest</p>
        </div>
    </div>
    
    <!-- Document Grid -->
    <div id="library-grid" 
         class="document-grid"
         hx-get="/api/library"
         hx-trigger="load, refresh from:body"
         hx-swap="innerHTML">
        <!-- Documents loaded via HTMX -->
    </div>
</div>

<script>
// Drag and drop
const dropZone = document.getElementById('drop-zone');
const libraryDiv = document.querySelector('.library');

libraryDiv.addEventListener('dragover', (e) => {
    e.preventDefault();
    dropZone.classList.remove('hidden');
});

dropZone.addEventListener('dragleave', (e) => {
    if (e.target === dropZone) {
        dropZone.classList.add('hidden');
    }
});

dropZone.addEventListener('drop', (e) => {
    e.preventDefault();
    dropZone.classList.add('hidden');
    
    const files = Array.from(e.dataTransfer.files);
    files.forEach(file => uploadFile(file));
});

function uploadFile(file) {
    const formData = new FormData();
    formData.append('file', file);
    
    fetch('/api/ingest/file', {
        method: 'POST',
        body: formData
    })
    .then(response => response.json())
    .then(data => {
        if (data.error) {
            showToast('error', data.error);
        } else {
            showToast('success', 'File ingested successfully');
            htmx.trigger('#library-grid', 'refresh');
        }
    })
    .catch(err => {
        showToast('error', 'Upload failed: ' + err.message);
    });
}

function filterByTag(tag) {
    const url = tag ? `/api/library?tag=${encodeURIComponent(tag)}` : '/api/library';
    htmx.ajax('GET', url, {target: '#library-grid', swap: 'innerHTML'});
}

function deleteDocument(source) {
    if (!confirm('Delete this document?')) return;
    
    fetch('/api/delete', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({source})
    })
    .then(() => {
        showToast('success', 'Document deleted');
        htmx.trigger('#library-grid', 'refresh');
    })
    .catch(err => {
        showToast('error', 'Delete failed: ' + err.message);
    });
}
</script>
{{end}}
```

#### Settings Template (settings.html)

```html
{{define "content"}}
<div class="settings">
    <h1>Settings</h1>
    
    <!-- Privacy Mode -->
    <section class="settings-section">
        <h2>Privacy Mode</h2>
        <div class="setting-item">
            <label class="toggle">
                <input type="checkbox" id="privacy-mode" {{if .PrivacyMode}}checked{{end}} onchange="togglePrivacyMode(this.checked)">
                <span class="toggle-slider"></span>
            </label>
            <div class="setting-description">
                <strong>Enable Privacy Mode</strong>
                <p>All data stays local. Only Ollama provider allowed. No external network requests.</p>
            </div>
        </div>
    </section>
    
    <!-- LLM Provider -->
    <section class="settings-section">
        <h2>LLM Provider</h2>
        
        <div class="setting-item">
            <label>Provider</label>
            <select id="provider-type" onchange="updateProviderFields(this.value)" {{if .PrivacyMode}}disabled{{end}}>
                <option value="ollama" {{if eq .Provider "ollama"}}selected{{end}}>Ollama (Local)</option>
                <option value="openai" {{if eq .Provider "openai"}}selected{{end}}>OpenAI</option>
                <option value="anthropic" {{if eq .Provider "anthropic"}}selected{{end}}>Anthropic</option>
            </select>
        </div>
        
        <!-- Ollama Settings -->
        <div id="ollama-settings" class="provider-settings">
            <div class="setting-item">
                <label>Endpoint</label>
                <input type="text" id="ollama-endpoint" value="{{.OllamaEndpoint}}" placeholder="http://localhost:11434">
            </div>
            <div class="setting-item">
                <label>Embedding Model</label>
                <input type="text" id="ollama-embed-model" value="{{.OllamaEmbedModel}}" placeholder="nomic-embed-text">
            </div>
            <div class="setting-item">
                <label>Chat Model</label>
                <input type="text" id="ollama-chat-model" value="{{.OllamaChatModel}}" placeholder="llama3.2">
            </div>
        </div>
        
        <!-- OpenAI Settings -->
        <div id="openai-settings" class="provider-settings hidden">
            <div class="setting-item">
                <label>API Key</label>
                <input type="password" id="openai-key" placeholder="sk-...">
            </div>
            <div class="setting-item">
                <label>Embedding Model</label>
                <select id="openai-embed-model">
                    <option value="text-embedding-3-small">text-embedding-3-small</option>
                    <option value="text-embedding-3-large">text-embedding-3-large</option>
                </select>
            </div>
            <div class="setting-item">
                <label>Chat Model</label>
                <select id="openai-chat-model">
                    <option value="gpt-4">GPT-4</option>
                    <option value="gpt-3.5-turbo">GPT-3.5 Turbo</option>
                </select>
            </div>
        </div>
        
        <!-- Anthropic Settings -->
        <div id="anthropic-settings" class="provider-settings hidden">
            <div class="setting-item">
                <label>API Key</label>
                <input type="password" id="anthropic-key" placeholder="sk-ant-...">
            </div>
            <div class="setting-item">
                <label>Chat Model</label>
                <select id="anthropic-chat-model">
                    <option value="claude-3-opus-20240229">Claude 3 Opus</option>
                    <option value="claude-3-sonnet-20240229">Claude 3 Sonnet</option>
                    <option value="claude-3-haiku-20240307">Claude 3 Haiku</option>
                </select>
            </div>
        </div>
        
        <div class="setting-actions">
            <button class="btn btn-secondary" onclick="testConnection()">Test Connection</button>
            <button class="btn btn-primary" onclick="saveSettings()">Save Settings</button>
        </div>
    </section>
    
    <!-- Watched Folders -->
    <section class="settings-section">
        <h2>Watched Folders</h2>
        <div id="watched-folders-list">
            {{range .WatchedFolders}}
            <div class="watched-folder-item">
                <span>{{.Path}}</span>
                <button class="btn-icon" onclick="removeFolder('{{.Path}}')">🗑️</button>
            </div>
            {{end}}
        </div>
        <button class="btn btn-secondary" onclick="addFolder()">Add Folder</button>
    </section>
    
    <!-- Guardrails -->
    <section class="settings-section">
        <h2>Ingestion Guardrails</h2>
        
        <div class="setting-item">
            <label>PII Detection</label>
            <select id="pii-detection">
                <option value="strict">Strict</option>
                <option value="normal" selected>Normal</option>
                <option value="off">Off</option>
            </select>
        </div>
        
        <div class="setting-item">
            <label class="toggle">
                <input type="checkbox" id="auto-summarize" checked>
                <span class="toggle-slider"></span>
            </label>
            <div class="setting-description">
                <strong>Auto-Summarize Documents</strong>
                <p>Generate summaries for ingested documents using LLM</p>
            </div>
        </div>
    </section>
</div>

<script>
function updateProviderFields(provider) {
    document.querySelectorAll('.provider-settings').forEach(el => el.classList.add('hidden'));
    document.getElementById(provider + '-settings').classList.remove('hidden');
}

function togglePrivacyMode(enabled) {
    fetch('/api/config', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({privacy_mode: enabled})
    })
    .then(() => {
        showToast('success', 'Privacy mode ' + (enabled ? 'enabled' : 'disabled'));
        if (enabled) {
            document.getElementById('provider-type').value = 'ollama';
            document.getElementById('provider-type').disabled = true;
            updateProviderFields('ollama');
        } else {
            document.getElementById('provider-type').disabled = false;
        }
    });
}

function saveSettings() {
    const provider = document.getElementById('provider-type').value;
    const config = {provider};
    
    // Collect provider-specific settings
    if (provider === 'ollama') {
        config.ollama_endpoint = document.getElementById('ollama-endpoint').value;
        config.ollama_embed_model = document.getElementById('ollama-embed-model').value;
        config.ollama_chat_model = document.getElementById('ollama-chat-model').value;
    }
    // ... similar for other providers
    
    fetch('/api/config', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify(config)
    })
    .then(() => showToast('success', 'Settings saved'))
    .catch(err => showToast('error', 'Save failed: ' + err.message));
}

function testConnection() {
    fetch('/api/test-connection', {method: 'POST'})
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                showToast('success', 'Connection successful');
            } else {
                showToast('error', 'Connection failed: ' + data.error);
            }
        });
}
</script>
{{end}}
```


### CSS Styling (style.css)

```css
:root {
    --primary-color: #3b82f6;
    --secondary-color: #6b7280;
    --success-color: #10b981;
    --error-color: #ef4444;
    --bg-color: #f9fafb;
    --sidebar-width: 250px;
    --sidebar-collapsed-width: 60px;
}

* {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
}

body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    background: var(--bg-color);
    color: #1f2937;
}

/* Sidebar */
.sidebar {
    position: fixed;
    left: 0;
    top: 0;
    bottom: 0;
    width: var(--sidebar-width);
    background: white;
    border-right: 1px solid #e5e7eb;
    transition: width 0.2s ease;
    z-index: 100;
}

.sidebar.collapsed {
    width: var(--sidebar-collapsed-width);
}

.sidebar-header {
    padding: 1.5rem;
    border-bottom: 1px solid #e5e7eb;
    display: flex;
    justify-content: space-between;
    align-items: center;
}

.sidebar-nav {
    padding: 1rem 0;
}

.nav-item {
    display: flex;
    align-items: center;
    padding: 0.75rem 1.5rem;
    color: #6b7280;
    text-decoration: none;
    transition: all 0.2s;
}

.nav-item:hover {
    background: #f3f4f6;
    color: var(--primary-color);
}

.nav-item.active {
    background: #eff6ff;
    color: var(--primary-color);
    border-left: 3px solid var(--primary-color);
}

.nav-item .icon {
    margin-right: 0.75rem;
    font-size: 1.25rem;
}

.sidebar.collapsed .nav-item .label {
    display: none;
}

.privacy-badge {
    position: absolute;
    bottom: 1rem;
    left: 1rem;
    right: 1rem;
    padding: 0.5rem;
    background: #fef3c7;
    border-radius: 0.5rem;
    font-size: 0.875rem;
    text-align: center;
}

/* Main Content */
.main-content {
    margin-left: var(--sidebar-width);
    padding: 2rem;
    transition: margin-left 0.2s ease;
}

.sidebar.collapsed ~ .main-content {
    margin-left: var(--sidebar-collapsed-width);
}

/* Dashboard */
.stats-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 1.5rem;
    margin: 2rem 0;
}

.stat-card {
    background: white;
    padding: 1.5rem;
    border-radius: 0.5rem;
    box-shadow: 0 1px 3px rgba(0,0,0,0.1);
}

.stat-icon {
    font-size: 2rem;
    margin-bottom: 0.5rem;
}

.stat-value {
    font-size: 1.5rem;
    font-weight: bold;
    margin-bottom: 0.25rem;
}

.stat-label {
    color: #6b7280;
    font-size: 0.875rem;
}

/* Chat */
.chat-container {
    display: flex;
    height: calc(100vh - 4rem);
    gap: 1rem;
}

.session-sidebar {
    width: 250px;
    background: white;
    border-radius: 0.5rem;
    padding: 1rem;
    overflow-y: auto;
}

.chat-area {
    flex: 1;
    display: flex;
    flex-direction: column;
    background: white;
    border-radius: 0.5rem;
    padding: 1rem;
}

.messages {
    flex: 1;
    overflow-y: auto;
    padding: 1rem;
}

.message {
    margin-bottom: 1.5rem;
    padding: 1rem;
    border-radius: 0.5rem;
}

.message.user {
    background: #eff6ff;
    margin-left: 20%;
}

.message.assistant {
    background: #f9fafb;
    margin-right: 20%;
}

.message-content {
    line-height: 1.6;
}

.message-content code {
    background: #1f2937;
    color: #f9fafb;
    padding: 0.125rem 0.25rem;
    border-radius: 0.25rem;
    font-family: 'Courier New', monospace;
}

.message-content pre {
    background: #1f2937;
    color: #f9fafb;
    padding: 1rem;
    border-radius: 0.5rem;
    overflow-x: auto;
    margin: 0.5rem 0;
}

.chat-input-form {
    display: flex;
    gap: 0.5rem;
    padding: 1rem;
    border-top: 1px solid #e5e7eb;
}

.chat-input-form textarea {
    flex: 1;
    padding: 0.75rem;
    border: 1px solid #d1d5db;
    border-radius: 0.5rem;
    resize: none;
    font-family: inherit;
}

/* Library */
.document-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
    gap: 1.5rem;
    margin-top: 2rem;
}

.document-card {
    background: white;
    padding: 1.5rem;
    border-radius: 0.5rem;
    box-shadow: 0 1px 3px rgba(0,0,0,0.1);
    transition: transform 0.2s, box-shadow 0.2s;
}

.document-card:hover {
    transform: translateY(-2px);
    box-shadow: 0 4px 6px rgba(0,0,0,0.1);
}

.drop-zone {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(59, 130, 246, 0.1);
    border: 3px dashed var(--primary-color);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
}

.drop-zone.hidden {
    display: none;
}

/* Toast Notifications */
.toast-container {
    position: fixed;
    top: 1rem;
    right: 1rem;
    z-index: 1000;
}

.toast {
    background: white;
    padding: 1rem 1.5rem;
    border-radius: 0.5rem;
    box-shadow: 0 4px 6px rgba(0,0,0,0.1);
    margin-bottom: 0.5rem;
    display: flex;
    align-items: center;
    gap: 0.75rem;
    animation: slideIn 0.3s ease;
}

@keyframes slideIn {
    from {
        transform: translateX(100%);
        opacity: 0;
    }
    to {
        transform: translateX(0);
        opacity: 1;
    }
}

.toast.success {
    border-left: 4px solid var(--success-color);
}

.toast.error {
    border-left: 4px solid var(--error-color);
}

/* Command Palette */
.command-palette {
    position: fixed;
    top: 20%;
    left: 50%;
    transform: translateX(-50%);
    width: 600px;
    max-width: 90vw;
    background: white;
    border-radius: 0.5rem;
    box-shadow: 0 20px 25px rgba(0,0,0,0.15);
    z-index: 1000;
}

.command-palette.hidden {
    display: none;
}

.command-palette input {
    width: 100%;
    padding: 1rem;
    border: none;
    border-bottom: 1px solid #e5e7eb;
    font-size: 1rem;
}

/* Buttons */
.btn {
    padding: 0.5rem 1rem;
    border: none;
    border-radius: 0.375rem;
    font-size: 0.875rem;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s;
}

.btn-primary {
    background: var(--primary-color);
    color: white;
}

.btn-primary:hover {
    background: #2563eb;
}

.btn-secondary {
    background: #e5e7eb;
    color: #374151;
}

.btn-secondary:hover {
    background: #d1d5db;
}

/* Toggle Switch */
.toggle {
    position: relative;
    display: inline-block;
    width: 50px;
    height: 24px;
}

.toggle input {
    opacity: 0;
    width: 0;
    height: 0;
}

.toggle-slider {
    position: absolute;
    cursor: pointer;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background-color: #ccc;
    transition: 0.4s;
    border-radius: 24px;
}

.toggle-slider:before {
    position: absolute;
    content: "";
    height: 18px;
    width: 18px;
    left: 3px;
    bottom: 3px;
    background-color: white;
    transition: 0.4s;
    border-radius: 50%;
}

.toggle input:checked + .toggle-slider {
    background-color: var(--primary-color);
}

.toggle input:checked + .toggle-slider:before {
    transform: translateX(26px);
}
```


### Client-Side JavaScript (app.js)

```javascript
// Toast notifications
function showToast(type, message) {
    const container = document.getElementById('toast-container');
    const toast = document.createElement('div');
    toast.className = `toast ${type}`;
    
    const icon = type === 'success' ? '✓' : type === 'error' ? '✗' : 'ℹ';
    toast.innerHTML = `
        <span class="toast-icon">${icon}</span>
        <span class="toast-message">${message}</span>
        <button class="toast-close" onclick="this.parentElement.remove()">×</button>
    `;
    
    container.appendChild(toast);
    
    // Auto-dismiss after 5 seconds
    setTimeout(() => toast.remove(), 5000);
}

// Command palette
let commandPaletteOpen = false;

document.addEventListener('keydown', (e) => {
    // ⌘K or Ctrl+K
    if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        toggleCommandPalette();
    }
    
    // Escape
    if (e.key === 'Escape' && commandPaletteOpen) {
        toggleCommandPalette();
    }
});

function toggleCommandPalette() {
    const palette = document.getElementById('command-palette');
    commandPaletteOpen = !commandPaletteOpen;
    
    if (commandPaletteOpen) {
        palette.classList.remove('hidden');
        document.getElementById('command-input').focus();
        loadCommands();
    } else {
        palette.classList.add('hidden');
        document.getElementById('command-input').value = '';
    }
}

function loadCommands() {
    const commands = [
        {name: 'Go to Dashboard', action: () => window.location.href = '/'},
        {name: 'Go to Chat', action: () => window.location.href = '/chat'},
        {name: 'Go to Library', action: () => window.location.href = '/library'},
        {name: 'Go to Settings', action: () => window.location.href = '/settings'},
        {name: 'New Chat', action: () => window.location.href = '/chat?new=true'},
    ];
    
    // Add skills
    fetch('/api/skills')
        .then(r => r.json())
        .then(skills => {
            skills.forEach(skill => {
                commands.push({
                    name: `Run: ${skill.name}`,
                    action: () => runSkill(skill.name)
                });
            });
            
            renderCommands(commands);
        });
}

function renderCommands(commands) {
    const results = document.getElementById('command-results');
    results.innerHTML = commands.map((cmd, i) => `
        <div class="command-item" onclick="executeCommand(${i})">
            ${cmd.name}
        </div>
    `).join('');
    
    window.commandList = commands;
}

function executeCommand(index) {
    window.commandList[index].action();
    toggleCommandPalette();
}

// Sidebar toggle
document.getElementById('sidebar-toggle')?.addEventListener('click', () => {
    const sidebar = document.getElementById('sidebar');
    sidebar.classList.toggle('collapsed');
    localStorage.setItem('sidebar-collapsed', sidebar.classList.contains('collapsed'));
});

// Restore sidebar state
if (localStorage.getItem('sidebar-collapsed') === 'true') {
    document.getElementById('sidebar')?.classList.add('collapsed');
}

// Markdown rendering helper
function renderMarkdown(text) {
    // This is a placeholder - actual rendering happens server-side with goldmark
    return text;
}
```


## Data Flow Diagrams

### Document Ingestion Flow

```
┌──────────┐
│  User    │
│ Uploads  │
│  File    │
└────┬─────┘
     │
     ▼
┌─────────────────┐
│  API Handler    │
│ (handleIngest)  │
└────┬────────────┘
     │
     ▼
┌─────────────────┐
│   Guardrails    │
│  - Size check   │
│  - Extension    │
│  - PII detect   │
└────┬────────────┘
     │
     ▼
┌─────────────────┐
│  File Parser    │
│  (PDF/TXT/MD)   │
└────┬────────────┘
     │
     ▼
┌─────────────────┐
│    Chunker      │
│ (200-500 chars) │
└────┬────────────┘
     │
     ▼
┌─────────────────┐
│  LLM Provider   │
│   (Embed)       │
└────┬────────────┘
     │
     ▼
┌─────────────────┐
│     Store       │
│  (SaveChunk)    │
└────┬────────────┘
     │
     ▼
┌─────────────────┐
│  Audit Log      │
└────┬────────────┘
     │
     ▼
┌─────────────────┐
│  WebSocket Hub  │
│  (Broadcast)    │
└────┬────────────┘
     │
     ▼
┌─────────────────┐
│  UI Update      │
│ (Toast + Grid)  │
└─────────────────┘
```

### Chat Query Flow

```
┌──────────┐
│  User    │
│  Query   │
└────┬─────┘
     │
     ▼
┌─────────────────┐
│  API Handler    │
│  (handleAsk)    │
└────┬────────────┘
     │
     ├──────────────────┐
     │                  │
     ▼                  ▼
┌─────────────┐   ┌─────────────┐
│    Store    │   │ LLM Provider│
│ SaveMessage │   │   (Embed)   │
│   (user)    │   │             │
└─────────────┘   └────┬────────┘
                       │
                       ▼
                  ┌─────────────┐
                  │    Store    │
                  │  (Search)   │
                  │  Top 5      │
                  └────┬────────┘
                       │
                       ▼
                  ┌─────────────┐
                  │   Prompt    │
                  │  Builder    │
                  └────┬────────┘
                       │
                       ▼
                  ┌─────────────┐
                  │ LLM Provider│
                  │  (Stream)   │
                  └────┬────────┘
                       │
                       ▼
                  ┌─────────────┐
                  │  Response   │
                  │   Stream    │
                  │  to Client  │
                  └────┬────────┘
                       │
                       ▼
                  ┌─────────────┐
                  │    Store    │
                  │ SaveMessage │
                  │ (assistant) │
                  └─────────────┘
```

### Skill Execution Flow

```
┌──────────┐
│  Trigger │
│ (manual/ │
│ keyword/ │
│  timer)  │
└────┬─────┘
     │
     ▼
┌─────────────────┐
│ Skill Executor  │
└────┬────────────┘
     │
     ├─────────────────┐
     │                 │
     ▼                 ▼
┌──────────┐     ┌──────────┐
│ Prepare  │     │  Check   │
│  Input   │     │ Privacy  │
│  JSON    │     │   Mode   │
└────┬─────┘     └────┬─────┘
     │                │
     └────────┬───────┘
              │
              ▼
        ┌──────────┐
        │  Spawn   │
        │Subprocess│
        └────┬─────┘
             │
             ├──────────┐
             │          │
             ▼          ▼
        ┌────────┐ ┌────────┐
        │ stdin  │ │  env   │
        │  JSON  │ │  vars  │
        └────┬───┘ └────┬───┘
             │          │
             └────┬─────┘
                  │
                  ▼
            ┌──────────┐
            │  Skill   │
            │ Executes │
            └────┬─────┘
                 │
                 ▼
            ┌──────────┐
            │ stdout   │
            │  JSON    │
            └────┬─────┘
                 │
                 ▼
            ┌──────────┐
            │  Parse   │
            │ Output   │
            └────┬─────┘
                 │
                 ▼
            ┌──────────┐
            │  Audit   │
            │   Log    │
            └────┬─────┘
                 │
                 ▼
            ┌──────────┐
            │  Return  │
            │  Result  │
            └──────────┘
```

### Folder Watching Flow

```
┌──────────────┐
│  fsnotify    │
│   Watcher    │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ File Event   │
│ (Create/     │
│  Modify/     │
│  Delete)     │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  Validate    │
│ - Extension  │
│ - Size       │
│ - Not system │
└──────┬───────┘
       │
       ├─────────────┐
       │             │
       ▼             ▼
  ┌────────┐   ┌────────┐
  │ Create │   │ Delete │
  │Modify  │   │        │
  └───┬────┘   └───┬────┘
      │            │
      ▼            ▼
  ┌────────┐   ┌────────┐
  │Ingester│   │ Store  │
  │        │   │ Delete │
  └───┬────┘   └────────┘
      │
      ▼
  ┌────────┐
  │  Log   │
  │ Event  │
  └────────┘
```


## Security & Guardrails

### Privacy Mode Enforcement

Privacy mode is enforced at multiple layers:

1. **Configuration Layer**: Config validation rejects cloud providers when privacy mode is enabled
2. **LLM Layer**: Provider factory refuses to instantiate cloud providers in privacy mode
3. **Ingestion Layer**: URL ingestion is blocked when privacy mode is enabled
4. **Skill Layer**: Skills requiring network access are not loaded in privacy mode
5. **Watcher Layer**: Only local filesystem paths are monitored

### PII Detection Patterns

The system detects the following PII patterns:

- Social Security Numbers: `\d{3}-\d{2}-\d{4}`
- Credit Cards: `\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}`
- API Keys: `sk-[a-zA-Z0-9]{32,}`, `ghp_[a-zA-Z0-9]{36}`, `xox[baprs]-[a-zA-Z0-9-]+`
- Private Keys: `-----BEGIN (RSA |EC |OPENSSH )?PRIVATE KEY-----`
- Email Addresses: Standard email regex
- Phone Numbers: `\d{3}[-.]?\d{3}[-.]?\d{4}`

### File Ingestion Guardrails

**Allowed Extensions**: `.txt`, `.md`, `.pdf`, `.html`

**Blocked Extensions**:
- Executables: `.exe`, `.dll`, `.so`, `.dylib`, `.app`
- Archives: `.zip`, `.tar`, `.gz`, `.rar`
- Disk Images: `.iso`, `.dmg`, `.img`

**Sensitive Filenames**: `.env`, `id_rsa`, `id_ed25519`, `credentials.json`, `.aws/credentials`

**Size Limits**: Default 10MB per file

**System Directory Protection**: Blocks `/etc`, `/System`, `/Windows`, `/sys`, `/proc`


## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Chunk Save and Retrieve Round Trip

*For any* source, text, embedding vector, tags, and summary, saving a chunk and then searching with the same embedding vector should return a chunk with the same text and metadata.

**Validates: Requirements 2.1, 2.2**

### Property 2: Search Returns Correct Count

*For any* query vector and topK parameter, the search method should return at most topK results, and all results should have similarity scores.

**Validates: Requirements 2.2**

### Property 3: Library Returns Unique Sources

*For any* database state, the Library method should return each source exactly once, with accurate chunk counts.

**Validates: Requirements 2.3**

### Property 4: Delete Removes All Chunks

*For any* source that exists in the database, after calling DeleteSource, searching for chunks from that source should return zero results.

**Validates: Requirements 2.4**

### Property 5: Chat Message Chronological Order

*For any* session ID with multiple messages, GetSessionHistory should return messages in chronological order (oldest to newest).

**Validates: Requirements 2.6**

### Property 6: Session List Uniqueness

*For any* database state with chat messages, ListSessions should return each session ID exactly once.

**Validates: Requirements 2.7**

### Property 7: Audit Log Filtering

*For any* audit log with multiple operation types, filtering by a specific type should return only entries of that type.

**Validates: Requirements 2.9**

### Property 8: Database Migration Preserves Data

*For any* existing database with chunks, running migrations should preserve all existing chunk data (source, text, embeddings).

**Validates: Requirements 2.10, 23.1, 23.2, 23.3**

### Property 9: Embedding Returns Fixed Dimension Vector

*For any* text input, the Embed method should return a vector of consistent dimensionality (e.g., 768 or 1536 depending on model).

**Validates: Requirements 3.2**

### Property 10: Stream Writes to Writer

*For any* valid message list, the Stream method should write non-empty content to the provided io.Writer.

**Validates: Requirements 3.3**

### Property 11: Context Cancellation Stops Provider

*For any* provider operation (Embed or Stream), canceling the context should cause the operation to return an error within a reasonable timeout.

**Validates: Requirements 3.7**

### Property 12: Provider Errors Are Descriptive

*For any* provider API error, the returned error message should contain information about the failure (not just "error occurred").

**Validates: Requirements 3.8**

### Property 13: Privacy Mode Blocks Cloud Providers

*For any* cloud provider type (OpenAI, Anthropic), attempting to instantiate it with privacy mode enabled should return an error.

**Validates: Requirements 3.10, 3.11**

### Property 14: Chunk Size Bounds

*For any* text input, ChunkText should produce chunks where each chunk's length is between 200 and 500 characters (except possibly the last chunk).

**Validates: Requirements 4.5**

### Property 15: Chunk Overlap

*For any* text input longer than one chunk, consecutive chunks should share approximately 50 characters of overlap.

**Validates: Requirements 4.5**

### Property 16: Cosine Similarity Bounds

*For any* two vectors, CosineSimilarity should return a value between -1.0 and 1.0, with identical vectors returning 1.0.

**Validates: Requirements 4.4**

### Property 17: Prompt Contains Query and Context

*For any* query string and list of chunks, BuildPrompt should return a string containing both the query and all chunk texts.

**Validates: Requirements 4.3**

### Property 18: Text Ingestion Creates Chunks

*For any* valid text input, IngestText should create at least one chunk in the database.

**Validates: Requirements 5.1**

### Property 19: URL Ingestion Blocked in Privacy Mode

*For any* URL, calling IngestURL with privacy mode enabled should return an error indicating privacy mode restriction.

**Validates: Requirements 5.7**

### Property 20: Ingestion Errors Are Descriptive

*For any* ingestion failure (size limit, PII detected, parse error), the error message should indicate the specific failure reason.

**Validates: Requirements 5.6**

### Property 21: Environment Variables Override Config

*For any* config value that has an environment variable set, the loaded config should use the environment variable value instead of the file value.

**Validates: Requirements 19.2**

### Property 22: Privacy Mode Forces Ollama

*For any* configuration with privacy mode enabled and a cloud provider specified, config validation should either force Ollama or return an error.

**Validates: Requirements 19.16**

### Property 23: Privacy Mode Validates Localhost

*For any* Ollama endpoint that is not localhost or 127.0.0.1, config validation with privacy mode enabled should return an error.

**Validates: Requirements 19.17**

### Property 24: Malformed Config Produces Clear Error

*For any* invalid JSON in config.json, the config loader should return an error message indicating JSON parsing failure.

**Validates: Requirements 19.20**

### Property 25: File Creation Triggers Ingestion

*For any* file created in a watched folder with an allowed extension, the watcher should trigger ingestion of that file.

**Validates: Requirements 27.2**

### Property 26: File Modification Triggers Re-ingestion

*For any* file modified in a watched folder, the watcher should trigger re-ingestion and update existing chunks.

**Validates: Requirements 27.3**

### Property 27: File Deletion Removes Chunks

*For any* file deleted from a watched folder, the watcher should remove all chunks associated with that file from the database.

**Validates: Requirements 27.4**

### Property 28: Watcher Skips Disallowed Extensions

*For any* file with an extension not in the allowed list, the watcher should skip processing that file.

**Validates: Requirements 27.5**

### Property 29: Watcher Skips Oversized Files

*For any* file larger than the configured size limit, the watcher should skip processing that file.

**Validates: Requirements 27.6**

### Property 30: Watcher Rejects System Directories

*For any* path that starts with a system directory prefix (/etc, /System, /Windows), the watcher should reject adding that path.

**Validates: Requirements 27.8**

### Property 31: PII Detection Finds Patterns

*For any* text containing PII patterns (SSN, credit card, API key, private key, email, phone), the PII detector should identify at least one PII type.

**Validates: Requirements 28.1, 28.7**

### Property 32: PII Logs Don't Contain Values

*For any* text with detected PII, the warning log should contain the PII type but not the actual PII value.

**Validates: Requirements 28.11**

### Property 33: Oversized Files Are Rejected

*For any* file larger than the configured maximum size, ingestion should fail with an error indicating size limit exceeded.

**Validates: Requirements 31.1, 31.2**

### Property 34: Disallowed Extensions Are Rejected

*For any* file with an extension not in the allowed list, ingestion should fail with an error indicating disallowed extension.

**Validates: Requirements 31.3, 31.4**

### Property 35: Executables Are Rejected

*For any* file with an executable extension (.exe, .dll, .so, .dylib, .app), ingestion should fail with an error.

**Validates: Requirements 31.5**

### Property 36: Archives Are Rejected

*For any* file with an archive extension (.zip, .tar, .gz, .rar), ingestion should fail with an error.

**Validates: Requirements 31.6**

### Property 37: Sensitive Filenames Are Rejected

*For any* filename containing sensitive patterns (.env, id_rsa, credentials.json), ingestion should fail with an error.

**Validates: Requirements 31.8**

### Property 38: Skills Load from Directory

*For any* valid skill directory with skill.json and executable, the skill loader should successfully load that skill.

**Validates: Requirements 33.1**

### Property 39: Skill Input/Output JSON Round Trip

*For any* skill input JSON, executing a skill should produce output JSON that can be parsed into the expected structure.

**Validates: Requirements 33.5, 33.6, 33.7**

### Property 40: Skill Timeout Enforced

*For any* skill that runs longer than its configured timeout, the executor should terminate the subprocess and return a timeout error.

**Validates: Requirements 33.13, 33.14**

### Property 41: Privacy Mode Env Var Passed to Skills

*For any* skill executed with privacy mode enabled, the skill subprocess should receive the NOODEXX_PRIVACY_MODE=true environment variable.

**Validates: Requirements 33.15, 33.16**


## Error Handling

### Error Handling Strategy

The system follows these error handling principles:

1. **Descriptive Errors**: All errors include context about what failed and why
2. **Error Wrapping**: Use `fmt.Errorf("context: %w", err)` to preserve error chains
3. **No Panic**: Avoid panics in production code; return errors instead
4. **Logging**: Log errors at appropriate levels (ERROR for failures, WARN for recoverable issues)
5. **User-Facing Errors**: Don't expose internal details in HTTP responses

### Error Categories

**Database Errors**:
- Connection failures: "failed to connect to database: %w"
- Query failures: "failed to execute query: %w"
- Migration failures: "database migration failed: %w"

**LLM Provider Errors**:
- API failures: "ollama embed request failed: %w"
- Timeout errors: "provider request timed out after 60s"
- Invalid responses: "failed to decode provider response: %w"

**Ingestion Errors**:
- Size limit: "file size %d exceeds limit %d"
- PII detected: "PII detected: %v - ingestion blocked"
- Parse failures: "failed to parse PDF: %w"
- Guardrail violations: "blocked file extension: %s"

**Configuration Errors**:
- Missing file: "config file not found, creating default"
- Parse errors: "failed to parse config.json: %w"
- Validation errors: "privacy mode requires Ollama provider"

**Skill Errors**:
- Load failures: "failed to load skill %s: %w"
- Execution failures: "skill execution failed: %w"
- Timeout: "skill execution timed out after %v"
- Invalid output: "failed to parse skill output: %w"

### HTTP Error Responses

```go
// Error response structure
type ErrorResponse struct {
    Error   string `json:"error"`
    Code    string `json:"code,omitempty"`
    Details string `json:"details,omitempty"`
}

// Example usage
func (s *Server) handleError(w http.ResponseWriter, err error, status int) {
    log.Printf("HTTP error: %v", err)
    
    resp := ErrorResponse{
        Error: "An error occurred",
    }
    
    // Don't expose internal errors to users
    if status >= 500 {
        resp.Error = "Internal server error"
    } else {
        resp.Error = err.Error()
    }
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(resp)
}
```


## Testing Strategy

### Dual Testing Approach

The testing strategy combines unit tests and property-based tests for comprehensive coverage:

**Unit Tests**:
- Specific examples demonstrating correct behavior
- Edge cases (empty inputs, boundary conditions)
- Error conditions and failure modes
- Integration points between components
- HTTP handler responses

**Property-Based Tests**:
- Universal properties that hold for all inputs
- Comprehensive input coverage through randomization
- Invariant validation across operations
- Round-trip properties (serialize/deserialize, save/load)

### Property-Based Testing Configuration

**Library Selection**: Use [gopter](https://github.com/leanovate/gopter) for Go property-based testing

**Test Configuration**:
- Minimum 100 iterations per property test
- Each test tagged with design document property reference
- Tag format: `// Feature: noodexx-phase-2-refactor-and-ui, Property N: [property text]`

**Example Property Test**:

```go
package store_test

import (
    "testing"
    "github.com/leanovate/gopter"
    "github.com/leanovate/gopter/gen"
    "github.com/leanovate/gopter/prop"
)

// Feature: noodexx-phase-2-refactor-and-ui, Property 1: Chunk Save and Retrieve Round Trip
func TestChunkSaveRetrieveRoundTrip(t *testing.T) {
    properties := gopter.NewProperties(nil)
    
    properties.Property("saving and retrieving chunk preserves data", prop.ForAll(
        func(source string, text string, embedding []float32) bool {
            store, _ := NewStore(":memory:")
            defer store.Close()
            
            // Save chunk
            err := store.SaveChunk(ctx, source, text, embedding, nil, "")
            if err != nil {
                return false
            }
            
            // Search with same embedding
            chunks, err := store.Search(ctx, embedding, 1)
            if err != nil || len(chunks) == 0 {
                return false
            }
            
            // Verify data preserved
            return chunks[0].Source == source && chunks[0].Text == text
        },
        gen.AlphaString(),
        gen.AlphaString(),
        gen.SliceOf(gen.Float32Range(-1, 1)),
    ))
    
    properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: noodexx-phase-2-refactor-and-ui, Property 14: Chunk Size Bounds
func TestChunkSizeBounds(t *testing.T) {
    properties := gopter.NewProperties(nil)
    
    properties.Property("chunks are between 200-500 chars", prop.ForAll(
        func(text string) bool {
            if len(text) < 200 {
                return true // Skip short texts
            }
            
            chunker := &Chunker{ChunkSize: 400, Overlap: 50}
            chunks := chunker.ChunkText(text)
            
            for i, chunk := range chunks {
                // Last chunk can be shorter
                if i == len(chunks)-1 {
                    continue
                }
                
                if len(chunk) < 200 || len(chunk) > 500 {
                    return false
                }
            }
            
            return true
        },
        gen.AlphaString().SuchThat(func(s string) bool {
            return len(s) > 500
        }),
    ))
    
    properties.TestingRun(t, gopter.ConsoleReporter(false))
}
```

### Unit Test Examples

```go
package ingest_test

import (
    "testing"
)

// Test specific PII patterns
func TestPIIDetection_SSN(t *testing.T) {
    detector := NewPIIDetector()
    
    tests := []struct {
        text     string
        expected []string
    }{
        {"My SSN is 123-45-6789", []string{"ssn"}},
        {"No PII here", []string{}},
        {"Card: 1234-5678-9012-3456", []string{"credit_card"}},
    }
    
    for _, tt := range tests {
        result := detector.Detect(tt.text)
        if !equal(result, tt.expected) {
            t.Errorf("Detect(%q) = %v, want %v", tt.text, result, tt.expected)
        }
    }
}

// Test edge case: empty config file
func TestConfigLoad_EmptyFile(t *testing.T) {
    tmpfile := createTempFile(t, "")
    defer os.Remove(tmpfile)
    
    _, err := config.Load(tmpfile)
    if err == nil {
        t.Error("Expected error for empty config file")
    }
}

// Test integration: HTTP handler
func TestHandleAsk_Success(t *testing.T) {
    store := setupTestStore(t)
    provider := &MockProvider{}
    server := NewServer(store, provider, nil, nil, &Config{})
    
    req := httptest.NewRequest("POST", "/api/ask", strings.NewReader(`{"query":"test"}`))
    w := httptest.NewRecorder()
    
    server.handleAsk(w, req)
    
    if w.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", w.Code)
    }
}
```

### Test Coverage Goals

- **Store Package**: 90%+ coverage (critical data layer)
- **LLM Package**: 80%+ coverage (provider implementations)
- **RAG Package**: 85%+ coverage (core logic)
- **Ingest Package**: 85%+ coverage (safety-critical)
- **API Package**: 75%+ coverage (handlers)
- **Skills Package**: 80%+ coverage (subprocess management)
- **Config Package**: 90%+ coverage (validation logic)

### Integration Testing

Integration tests verify end-to-end flows:

1. **Ingestion Flow**: Upload file → Parse → Chunk → Embed → Store → Verify in library
2. **Chat Flow**: Query → Embed → Search → Build prompt → Stream → Save message
3. **Skill Flow**: Trigger → Load → Execute → Parse output → Audit log
4. **Watcher Flow**: Create file → Detect → Ingest → Verify chunks

### Performance Testing

Key performance benchmarks:

- **Embedding**: < 500ms for 1000 character text
- **Search**: < 100ms for top-5 search in 10,000 chunks
- **Chunking**: < 50ms for 10,000 character document
- **Database**: < 10ms for single chunk save/retrieve


## Implementation Notes

### Migration from Phase 1

The refactoring preserves Phase 1 functionality while establishing new architecture:

1. **Database Compatibility**: New columns (tags, summary) are added with ALTER TABLE, preserving existing chunks
2. **Embedding Preservation**: Existing embedding vectors remain valid and searchable
3. **Gradual Migration**: Phase 1 code can coexist during transition
4. **No Re-ingestion**: Users don't need to re-ingest documents

### Main.go Structure

The refactored main.go should be minimal (~80 lines):

```go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    
    "noodexx/internal/api"
    "noodexx/internal/config"
    "noodexx/internal/ingest"
    "noodexx/internal/llm"
    "noodexx/internal/logging"
    "noodexx/internal/rag"
    "noodexx/internal/skills"
    "noodexx/internal/store"
    "noodexx/internal/watcher"
)

func main() {
    // Load configuration
    cfg, err := config.Load("config.json")
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }
    
    // Initialize logger
    logger := logging.NewLogger("main", logging.ParseLevel(cfg.Logging.Level), os.Stdout)
    logger.Info("Starting Noodexx Phase 2")
    
    // Initialize store
    store, err := store.NewStore("noodexx.db")
    if err != nil {
        log.Fatalf("Failed to initialize store: %v", err)
    }
    defer store.Close()
    
    // Initialize LLM provider
    provider, err := llm.NewProvider(cfg.Provider, cfg.Privacy.Enabled)
    if err != nil {
        log.Fatalf("Failed to initialize LLM provider: %v", err)
    }
    
    // Initialize components
    chunker := &rag.Chunker{ChunkSize: 400, Overlap: 50}
    searcher := &rag.Searcher{Store: store}
    ingester := ingest.NewIngester(provider, store, chunker, cfg)
    
    // Initialize skill system
    skillLoader := skills.NewLoader("skills", cfg.Privacy.Enabled)
    loadedSkills, _ := skillLoader.LoadAll()
    logger.Info("Loaded %d skills", len(loadedSkills))
    
    // Initialize API server
    server, err := api.NewServer(store, provider, ingester, searcher, cfg)
    if err != nil {
        log.Fatalf("Failed to initialize server: %v", err)
    }
    
    // Initialize folder watcher
    watcher, err := watcher.NewWatcher(ingester, store, cfg.Privacy.Enabled)
    if err != nil {
        log.Fatalf("Failed to initialize watcher: %v", err)
    }
    watcher.Start(context.Background())
    
    // Register routes
    mux := http.NewServeMux()
    server.RegisterRoutes(mux)
    
    // Start HTTP server
    addr := fmt.Sprintf("%s:%d", cfg.Server.BindAddress, cfg.Server.Port)
    logger.Info("Server listening on %s", addr)
    
    httpServer := &http.Server{
        Addr:    addr,
        Handler: mux,
    }
    
    // Graceful shutdown
    go func() {
        sigChan := make(chan os.Signal, 1)
        signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
        <-sigChan
        
        logger.Info("Shutting down...")
        httpServer.Shutdown(context.Background())
    }()
    
    if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
        log.Fatalf("Server error: %v", err)
    }
}
```

### Dependency Management

Add to go.mod:

```
require (
    github.com/yuin/goldmark v1.6.0
    github.com/gorilla/websocket v1.5.1
    github.com/fsnotify/fsnotify v1.7.0
    github.com/go-shiori/go-readability v0.0.0-20231029095239-6b97d5aba789
    modernc.org/sqlite v1.28.0
)
```

### Development Workflow

1. **Phase 1**: Create package structure and interfaces
2. **Phase 2**: Migrate store package with tests
3. **Phase 3**: Migrate LLM package with provider implementations
4. **Phase 4**: Migrate RAG and ingest packages
5. **Phase 5**: Migrate API package and templates
6. **Phase 6**: Implement skills system
7. **Phase 7**: Implement folder watcher
8. **Phase 8**: UI enhancements (sidebar, dashboard, etc.)
9. **Phase 9**: WebSocket integration
10. **Phase 10**: Final testing and documentation

### Deployment Considerations

**Binary Size**: Expect ~15-20MB compiled binary (pure Go, no CGo)

**Memory Usage**: ~50-100MB baseline, scales with document count

**Database Size**: ~1KB per chunk (text + embedding)

**Port Configuration**: Default 127.0.0.1:8080 (localhost only)

**File Permissions**: Requires read/write access to:
- Application directory (for config.json)
- Database file (noodexx.db)
- Watched folders (if configured)
- Skills directory

**Security Recommendations**:
- Keep default bind address (127.0.0.1) unless network access needed
- Enable privacy mode for sensitive data
- Review skills before enabling
- Use strong API keys for cloud providers
- Regularly review audit logs


## Summary

This design document specifies a comprehensive refactoring and enhancement of Noodexx, transforming it from a monolithic Phase 1 implementation into a modular, extensible system with modern UI capabilities.

### Key Architectural Decisions

1. **Package Structure**: Clean separation of concerns with internal packages for store, llm, rag, ingest, api, skills, config, logging, and watcher
2. **Provider Abstraction**: Unified interface supporting Ollama (local), OpenAI, and Anthropic with privacy mode enforcement
3. **Plugin System**: JSON-based skill metadata with subprocess execution and controlled environment
4. **Privacy-First**: Multi-layer privacy mode enforcement across configuration, providers, ingestion, and skills
5. **Modern UI**: HTMX-based partial updates, WebSocket real-time notifications, responsive design
6. **Safety Guardrails**: PII detection, file type validation, size limits, system directory protection

### Technology Stack

- **Backend**: Go 1.21+ with standard library HTTP server
- **Database**: SQLite (modernc.org/sqlite, pure Go)
- **Frontend**: HTMX + vanilla JavaScript
- **WebSocket**: gorilla/websocket
- **Markdown**: goldmark
- **File Watching**: fsnotify
- **Testing**: gopter for property-based tests

### Implementation Priorities

**Critical Path**:
1. Store package (database foundation)
2. LLM package (provider abstraction)
3. RAG package (core functionality)
4. API package (HTTP handlers)
5. UI templates (user interface)

**Secondary Features**:
6. Skills system (extensibility)
7. Folder watcher (automation)
8. WebSocket hub (real-time updates)
9. Configuration system (flexibility)
10. Logging system (observability)

### Success Criteria

- All Phase 1 functionality preserved
- main.go under 100 lines
- 80%+ test coverage on critical packages
- All 41 correctness properties validated
- Privacy mode fully enforced
- UI responsive on desktop and tablet
- Documentation complete

### Next Steps

After design approval:
1. Create package structure
2. Implement store package with migrations
3. Implement LLM provider abstraction
4. Migrate existing functionality to new packages
5. Implement UI enhancements
6. Add skills system
7. Add folder watcher
8. Write comprehensive tests
9. Update documentation
10. Deploy and validate

---

**Design Version**: 1.0  
**Last Updated**: 2024  
**Status**: Ready for Review
