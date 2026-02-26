package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// DataStore defines the interface for all database operations
// This abstraction allows swapping SQLite for PostgreSQL or other databases
type DataStore interface {
	// Lifecycle
	Close() error

	// User Management
	CreateUser(ctx context.Context, username, password, email string, isAdmin, mustChangePassword bool) (int64, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	GetUserByID(ctx context.Context, userID int64) (*User, error)
	ValidateCredentials(ctx context.Context, username, password string) (*User, error)
	UpdatePassword(ctx context.Context, userID int64, newPassword string) error
	UpdateLastLogin(ctx context.Context, userID int64) error
	ListUsers(ctx context.Context) ([]User, error)
	DeleteUser(ctx context.Context, userID int64) error

	// Session Token Management
	CreateSessionToken(ctx context.Context, token string, userID int64, expiresAt time.Time) error
	GetSessionToken(ctx context.Context, token string) (*SessionToken, error)
	DeleteSessionToken(ctx context.Context, token string) error
	CleanupExpiredTokens(ctx context.Context) error

	// Account Lockout
	RecordFailedLogin(ctx context.Context, username string) error
	ClearFailedLogins(ctx context.Context, username string) error
	IsAccountLocked(ctx context.Context, username string) (bool, time.Time)

	// User-Scoped Data Access
	SaveChunk(ctx context.Context, userID int64, source, text string, embedding []float32, tags []string, summary string) error
	SearchByUser(ctx context.Context, userID int64, queryVec []float32, topK int) ([]Chunk, error)
	LibraryByUser(ctx context.Context, userID int64) ([]LibraryEntry, error)
	DeleteChunksBySource(ctx context.Context, userID int64, source string) error

	// Session Management
	SaveChatMessage(ctx context.Context, userID int64, sessionID, role, content, providerMode string) error
	GetUserSessions(ctx context.Context, userID int64) ([]Session, error)
	GetSessionOwner(ctx context.Context, sessionID string) (int64, error)
	GetSessionMessages(ctx context.Context, userID int64, sessionID string) ([]ChatMessage, error)

	// Skills Management
	CreateSkill(ctx context.Context, userID int64, name, path string, enabled bool) (int64, error)
	GetUserSkills(ctx context.Context, userID int64) ([]Skill, error)
	UpdateSkillEnabled(ctx context.Context, userID int64, skillID int64, enabled bool) error
	DeleteSkill(ctx context.Context, userID int64, skillID int64) error

	// Watched Folders Management
	AddWatchedFolder(ctx context.Context, userID int64, path string) error
	GetWatchedFoldersByUser(ctx context.Context, userID int64) ([]WatchedFolder, error)
	RemoveWatchedFolder(ctx context.Context, userID int64, folderID int64) error

	// Audit Log
	LogAudit(ctx context.Context, userID int64, username, operation, details string) error
	GetAuditLogByUser(ctx context.Context, userID int64, limit int) ([]AuditEntry, error)
}

// NewDataStore creates a new DataStore instance based on configuration
// Currently only supports SQLite, but can be extended for PostgreSQL, MySQL, etc.
func NewDataStore(dbType, connectionString string) (DataStore, error) {
	switch dbType {
	case "sqlite", "": // Default to SQLite
		return NewSQLiteStore(connectionString)
	case "postgres":
		return nil, fmt.Errorf("PostgreSQL support coming in Phase 5")
	case "mysql":
		return nil, fmt.Errorf("MySQL support coming in Phase 5")
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
}

// NewSQLiteStore creates a new SQLite-backed DataStore
// This will be fully implemented in task 4.1 when Store is updated to implement DataStore
func NewSQLiteStore(path string) (DataStore, error) {
	// Enable WAL mode for concurrent access and busy timeout for write contention
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool for concurrent multi-user access
	db.SetMaxOpenConns(25)                 // Support up to 25 concurrent connections
	db.SetMaxIdleConns(5)                  // Keep 5 connections ready
	db.SetConnMaxLifetime(5 * time.Minute) // Recycle connections periodically

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	store := &Store{
		db:       db,
		userMode: "multi", // Default to multi-user mode for DataStore interface
	}

	// Run migrations
	if err := store.runMigrations(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return store, nil
}
