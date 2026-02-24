package api

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"time"
)

// Server holds dependencies and provides HTTP handlers
type Server struct {
	store          Store
	provider       LLMProvider
	ingester       Ingester
	searcher       Searcher
	wsHub          *WebSocketHub
	templates      *template.Template
	config         *ServerConfig
	skillsLoader   SkillsLoader
	skillsExecutor SkillsExecutor
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
	GetAuditLog(ctx context.Context, opType string, from, to time.Time) ([]AuditEntry, error)
}

// LLMProvider interface for chat and embeddings
type LLMProvider interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	Stream(ctx context.Context, messages []Message, w io.Writer) (string, error)
	Name() string
	IsLocal() bool
}

// Ingester interface for document ingestion
type Ingester interface {
	IngestText(ctx context.Context, source, text string, tags []string) error
	IngestURL(ctx context.Context, url string, tags []string) error
}

// Searcher interface for RAG search
type Searcher interface {
	Search(ctx context.Context, queryVec []float32, topK int) ([]Chunk, error)
}

// SkillsLoader interface for loading skills
type SkillsLoader interface {
	LoadAll() ([]*Skill, error)
}

// SkillsExecutor interface for executing skills
type SkillsExecutor interface {
	Execute(ctx context.Context, skill *Skill, input SkillInput) (*SkillOutput, error)
}

// Skill represents a loaded skill
type Skill struct {
	Name        string
	Version     string
	Description string
	Executable  string
	Triggers    []SkillTrigger
	Timeout     time.Duration
	RequiresNet bool
	Path        string
}

// SkillTrigger defines when a skill executes
type SkillTrigger struct {
	Type       string
	Parameters map[string]interface{}
}

// SkillInput is the input to a skill
type SkillInput struct {
	Query    string                 `json:"query"`
	Context  map[string]interface{} `json:"context"`
	Settings map[string]interface{} `json:"settings"`
}

// SkillOutput is the output from a skill
type SkillOutput struct {
	Result   string                 `json:"result"`
	Error    string                 `json:"error"`
	Metadata map[string]interface{} `json:"metadata"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Chunk represents a search result
type Chunk struct {
	Source string
	Text   string
	Score  float64
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
	Role      string
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
	OperationType string
	Details       string
	UserContext   string
}

// ServerConfig holds server configuration
type ServerConfig struct {
	PrivacyMode bool
	Provider    string
}

// NewServer creates a server with dependencies and loads templates
func NewServer(store Store, provider LLMProvider, ingester Ingester, searcher Searcher, config *ServerConfig, skillsLoader SkillsLoader, skillsExecutor SkillsExecutor) (*Server, error) {
	return NewServerWithTemplatePath(store, provider, ingester, searcher, config, skillsLoader, skillsExecutor, "web/templates/*.html")
}

// NewServerWithTemplatePath creates a server with a custom template path (useful for testing)
func NewServerWithTemplatePath(store Store, provider LLMProvider, ingester Ingester, searcher Searcher, config *ServerConfig, skillsLoader SkillsLoader, skillsExecutor SkillsExecutor, templatePath string) (*Server, error) {
	// Load templates from the specified path
	tmpl, err := template.ParseGlob(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	srv := &Server{
		store:          store,
		provider:       provider,
		ingester:       ingester,
		searcher:       searcher,
		wsHub:          NewWebSocketHub(),
		templates:      tmpl,
		config:         config,
		skillsLoader:   skillsLoader,
		skillsExecutor: skillsExecutor,
	}

	// Start WebSocket hub
	go srv.wsHub.Run()

	return srv, nil
}

// RegisterRoutes sets up all HTTP routes
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	// Static files - serve from web/static/
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
	mux.HandleFunc("/api/test-connection", s.handleTestConnection)
	mux.HandleFunc("/api/activity", s.handleActivity)
	mux.HandleFunc("/api/skills", s.handleSkills)
	mux.HandleFunc("/api/skills/run", s.handleRunSkill)

	// WebSocket
	mux.HandleFunc("/ws", s.handleWebSocket)
}
