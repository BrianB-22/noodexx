package api

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"noodexx/internal/auth"
	"path/filepath"
	"strings"
	"time"
)

// Server holds dependencies and provides HTTP handlers
type Server struct {
	store           Store
	provider        LLMProvider
	ingester        Ingester
	searcher        Searcher
	wsHub           *WebSocketHub
	templates       *template.Template
	config          *ServerConfig
	skillsLoader    SkillsLoader
	skillsExecutor  SkillsExecutor
	logger          Logger
	authProvider    AuthProvider
	configPath      string // Path to config file for saving
	providerManager ProviderManager
	ragEnforcer     RAGEnforcer
	uiStyle         interface{} // UIStyle configuration for theming
}

// Logger interface for structured logging
type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
	WithContext(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger
}

// Store interface for API operations
type Store interface {
	SaveChunk(ctx context.Context, source, text string, embedding []float32, tags []string, summary string) error
	Search(ctx context.Context, queryVec []float32, topK int) ([]Chunk, error)
	SearchByUser(ctx context.Context, userID int64, queryVec []float32, topK int) ([]Chunk, error)
	Library(ctx context.Context) ([]LibraryEntry, error)
	LibraryByUser(ctx context.Context, userID int64) ([]LibraryEntry, error)
	DeleteSource(ctx context.Context, source string) error
	SaveMessage(ctx context.Context, sessionID, role, content string) error
	SaveChatMessage(ctx context.Context, userID int64, sessionID, role, content, providerMode string) error
	GetSessionHistory(ctx context.Context, sessionID string) ([]ChatMessage, error)
	GetSessionMessages(ctx context.Context, userID int64, sessionID string) ([]ChatMessage, error)
	ListSessions(ctx context.Context) ([]Session, error)
	GetUserSessions(ctx context.Context, userID int64) ([]Session, error)
	GetSessionOwner(ctx context.Context, sessionID string) (int64, error)
	AddAuditEntry(ctx context.Context, opType, details, userCtx string) error
	GetAuditLog(ctx context.Context, opType string, from, to time.Time) ([]AuditEntry, error)
	// User management methods
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	GetUserByID(ctx context.Context, userID int64) (*User, error)
	CreateUser(ctx context.Context, username, password, email string, isAdmin, mustChangePassword bool) (int64, error)
	UpdatePassword(ctx context.Context, userID int64, newPassword string) error
	UpdateUserDarkMode(ctx context.Context, userID int64, darkMode bool) error
	ListUsers(ctx context.Context) ([]User, error)
	DeleteUser(ctx context.Context, userID int64) error
	// Skills management methods
	GetUserSkills(ctx context.Context, userID int64) ([]Skill, error)
	// Watched folders management methods
	GetWatchedFoldersByUser(ctx context.Context, userID int64) ([]WatchedFolder, error)
}

// AuthProvider interface for authentication operations
type AuthProvider interface {
	Login(ctx context.Context, username, password string) (token string, err error)
	Logout(ctx context.Context, token string) error
	ValidateToken(ctx context.Context, token string) (userID int64, err error)
	RefreshToken(ctx context.Context, token string) (newToken string, err error)
}

// User represents a user account
type User struct {
	ID                 int64
	Username           string
	PasswordHash       string
	Email              string
	IsAdmin            bool
	MustChangePassword bool
	CreatedAt          time.Time
	LastLogin          time.Time
	DarkMode           bool
}

// LLMProvider interface for chat and embeddings
type LLMProvider interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	Stream(ctx context.Context, messages []Message, w io.Writer) (string, error)
	Name() string
	IsLocal() bool
}

// ProviderManager interface for managing dual providers
type ProviderManager interface {
	GetActiveProvider() (LLMProvider, error)
	GetLocalProvider() LLMProvider
	GetCloudProvider() LLMProvider
	IsLocalMode() bool
	GetProviderName() string
	Reload(cfg interface{}) error
}

// RAGEnforcer interface for RAG policy enforcement
type RAGEnforcer interface {
	ShouldPerformRAG() bool
	GetRAGStatus() string
	Reload(cfg interface{})
}

// Ingester interface for document ingestion
type Ingester interface {
	IngestText(ctx context.Context, userID int64, source, text string, tags []string) error
	IngestURL(ctx context.Context, userID int64, url string, tags []string) error
}

// Searcher interface for RAG search
type Searcher interface {
	Search(ctx context.Context, queryVec []float32, topK int) ([]Chunk, error)
}

// SkillsLoader interface for loading skills
type SkillsLoader interface {
	LoadAll() ([]*Skill, error)
	LoadForUser(ctx context.Context, userID int64) ([]*Skill, error)
}

// SkillsExecutor interface for executing skills
type SkillsExecutor interface {
	Execute(ctx context.Context, skill *Skill, input SkillInput) (*SkillOutput, error)
}

// Skill represents a loaded skill
type Skill struct {
	UserID      int64 // Owner of the skill
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
	ID           int64
	SessionID    string
	Role         string
	Content      string
	ProviderMode string
	CreatedAt    time.Time
}

// Session represents a chat session
type Session struct {
	ID            string
	LastMessageAt time.Time
	MessageCount  int
}

// WatchedFolder represents a monitored directory
type WatchedFolder struct {
	ID     int64
	Path   string
	UserID int64
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
	PrivacyMode        bool
	UserMode           string // "single" or "multi"
	Provider           string
	OllamaEndpoint     string
	OllamaEmbedModel   string
	OllamaChatModel    string
	OpenAIKey          string
	OpenAIEmbedModel   string
	OpenAIChatModel    string
	AnthropicKey       string
	AnthropicChatModel string
}

// NewServer creates a server with dependencies and loads templates
func NewServer(store Store, provider LLMProvider, ingester Ingester, searcher Searcher, config *ServerConfig, skillsLoader SkillsLoader, skillsExecutor SkillsExecutor, logger Logger, authProvider AuthProvider, configPath string, providerManager ProviderManager, ragEnforcer RAGEnforcer, uiStyle interface{}) (*Server, error) {
	return NewServerWithTemplatePath(store, provider, ingester, searcher, config, skillsLoader, skillsExecutor, logger, authProvider, configPath, "web/templates/*.html", providerManager, ragEnforcer, uiStyle)
}

// NewServerWithTemplatePath creates a server with a custom template path (useful for testing)
func NewServerWithTemplatePath(store Store, provider LLMProvider, ingester Ingester, searcher Searcher, config *ServerConfig, skillsLoader SkillsLoader, skillsExecutor SkillsExecutor, logger Logger, authProvider AuthProvider, configPath string, templatePath string, providerManager ProviderManager, ragEnforcer RAGEnforcer, uiStyle interface{}) (*Server, error) {
	// Create template with custom functions
	funcMap := template.FuncMap{
		"toJSON": func(v interface{}) (template.JS, error) {
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				return "", err
			}
			return template.JS(jsonBytes), nil
		},
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, fmt.Errorf("dict requires an even number of arguments")
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
		"default": func(defaultValue interface{}, value interface{}) interface{} {
			// Return defaultValue if value is nil, empty string, or zero value
			if value == nil {
				return defaultValue
			}
			if str, ok := value.(string); ok && str == "" {
				return defaultValue
			}
			return value
		},
		"html": func(s string) template.HTML {
			// Convert string to template.HTML to prevent escaping
			return template.HTML(s)
		},
	}

	// Load templates from the specified path with custom functions
	tmpl, err := template.New("").Funcs(funcMap).ParseGlob(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	// Also load component templates if they exist
	componentPath := "web/templates/components/*.html"
	matches, _ := filepath.Glob(componentPath)
	if len(matches) > 0 {
		tmpl, err = tmpl.ParseGlob(componentPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load component templates: %w", err)
		}
	}

	srv := &Server{
		store:           store,
		provider:        provider,
		ingester:        ingester,
		searcher:        searcher,
		wsHub:           NewWebSocketHub(),
		templates:       tmpl,
		config:          config,
		skillsLoader:    skillsLoader,
		skillsExecutor:  skillsExecutor,
		logger:          logger,
		authProvider:    authProvider,
		configPath:      configPath,
		providerManager: providerManager,
		ragEnforcer:     ragEnforcer,
		uiStyle:         uiStyle,
	}

	// Start WebSocket hub
	go srv.wsHub.Run()

	return srv, nil
}

// RegisterRoutes sets up all HTTP routes
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	log.Printf("=== Registering HTTP routes ===")

	// Static files - serve from web/static/ with cache control
	staticHandler := http.StripPrefix("/static/", http.FileServer(http.Dir("web/static")))
	mux.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
		// Set cache control headers for static assets
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		staticHandler.ServeHTTP(w, r)
	})
	log.Printf("Registered: /static/")

	// API routes (register before page routes to avoid conflicts)
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
	mux.HandleFunc("/api/library", s.handleLibrary) // API endpoint for HTMX library loading
	mux.HandleFunc("/api/skills", s.handleSkills)
	mux.HandleFunc("/api/skills/run", s.handleRunSkill)
	mux.HandleFunc("/api/watched-folders", s.handleWatchedFolders)
	mux.HandleFunc("/api/settings", s.handleSaveSettings)              // Save settings endpoint
	mux.HandleFunc("/api/privacy-mode", s.handlePrivacyMode)           // Toggle privacy mode
	mux.HandleFunc("/api/privacy-toggle", s.handlePrivacyToggle)       // Toggle between local and cloud AI
	mux.HandleFunc("/api/user/preferences", s.handleUpdatePreferences) // Update user preferences (dark mode, etc.)
	// Authentication routes
	mux.HandleFunc("/api/login", s.handleLogin)
	mux.HandleFunc("/api/logout", s.handleLogout)
	mux.HandleFunc("/api/register", s.handleRegister)
	mux.HandleFunc("/api/change-password", s.handleChangePassword)
	// Admin user management routes
	mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			s.handleGetUsers(w, r)
		} else if r.Method == http.MethodPost {
			s.handleCreateUser(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/users/", func(w http.ResponseWriter, r *http.Request) {
		// Handle /api/users/:id and /api/users/:id/reset-password
		if strings.HasSuffix(r.URL.Path, "/reset-password") {
			if r.Method == http.MethodPost {
				s.handleResetUserPassword(w, r)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		} else {
			if r.Method == http.MethodDelete {
				s.handleDeleteUser(w, r)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		}
	})
	log.Printf("Registered: API routes")

	// WebSocket
	mux.HandleFunc("/ws", s.handleWebSocket)
	log.Printf("Registered: /ws")

	// Authentication page routes (multi-user mode only, but registered always)
	mux.HandleFunc("/login", s.handleLoginPage)
	log.Printf("Registered: /login -> handleLoginPage")

	mux.HandleFunc("/register", s.handleRegisterPage)
	log.Printf("Registered: /register -> handleRegisterPage")

	mux.HandleFunc("/change-password", s.handleChangePasswordPage)
	log.Printf("Registered: /change-password -> handleChangePasswordPage")

	// Page routes (register last, with exact path matching)
	mux.HandleFunc("/settings", s.handleSettings)
	log.Printf("Registered: /settings -> handleSettings")

	mux.HandleFunc("/library", s.handleLibrary)
	log.Printf("Registered: /library -> handleLibrary")

	mux.HandleFunc("/chat", s.handleChat)
	log.Printf("Registered: /chat -> handleChat")

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Only handle exact "/" path
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		// In multi-user mode, redirect to login if not authenticated
		if s.config.UserMode == "multi" {
			// Check if user is authenticated by looking for user_id in context
			ctx := r.Context()
			userID, err := auth.GetUserID(ctx)
			log.Printf("Root handler: user_mode=multi, userID=%d, err=%v", userID, err)
			if err != nil || userID == 0 {
				// Not authenticated, redirect to login
				log.Printf("Redirecting to /login")
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
		}

		// Authenticated or single-user mode, show dashboard
		s.handleDashboard(w, r)
	})
	log.Printf("Registered: / -> handleDashboard (with user_mode routing)")
	log.Printf("=== Route registration complete ===")
}
