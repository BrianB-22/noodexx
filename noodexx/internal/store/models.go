package store

import (
	"database/sql"
	"time"
)

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
	ID           int64
	SessionID    string
	Role         string // "user" or "assistant"
	Content      string
	ProviderMode string // "local" or "cloud"
	CreatedAt    time.Time
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
	UserID   int64
	Path     string
	Active   bool
	LastScan time.Time
}

// User represents a user account
type User struct {
	ID                 int64
	Username           string
	PasswordHash       string
	Email              sql.NullString
	IsAdmin            bool
	MustChangePassword bool
	CreatedAt          time.Time
	LastLogin          time.Time
	DarkMode           bool
}

// SessionToken represents an authentication session token
type SessionToken struct {
	Token     string
	UserID    int64
	CreatedAt time.Time
	ExpiresAt time.Time
}

// Skill represents a user-owned skill/plugin
type Skill struct {
	ID        int64
	UserID    int64
	Name      string
	Path      string
	Enabled   bool
	CreatedAt time.Time
}
