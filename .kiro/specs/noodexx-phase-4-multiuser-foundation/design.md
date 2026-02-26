# Design Document: Noodexx Phase 4 - Multi-User Foundation

## Overview

This design establishes the foundational architecture for multi-user support in Noodexx while maintaining complete backward compatibility with single-user deployments. The system introduces user identity, authentication, data ownership, and session isolation through a configurable user mode that defaults to single-user operation.

The design follows a layered approach:
- **Database Layer**: User tables, ownership columns, and migration strategy
- **Authentication Layer**: Pluggable auth providers with username/password implementation
- **Middleware Layer**: Request authentication and user context injection
- **Application Layer**: User-scoped data access and operations
- **UI Layer**: Login, registration, password management, and admin interfaces

Key design principles:
- **Backward Compatibility**: Phase 3 data migrates seamlessly to a "local-default" user
- **Zero-Config Single-User**: Default mode requires no authentication
- **Extensible Auth**: Provider interface supports future MFA and SSO
- **Data Isolation**: All user data is strictly scoped by user_id
- **Security First**: Bcrypt passwords, crypto/rand tokens, account lockout

## Architecture

### System Modes

The system operates in two distinct modes controlled by configuration:

**Single-User Mode (default)**:
- No authentication UI displayed
- All operations automatically use the "local-default" user
- Identical behavior to Phase 3 from user perspective
- Suitable for personal deployments

**Multi-User Mode**:
- Authentication required for all operations
- Login/registration UI displayed
- User-scoped data access enforced
- Admin capabilities for user management
- Suitable for shared/team deployments


### Architecture Diagram

```mermaid
graph TB
    subgraph "HTTP Layer"
        A[HTTP Request] --> B[AuthMiddleware]
        B --> C{User Mode?}
        C -->|Single| D[Inject local-default user]
        C -->|Multi| E[Validate Session Token]
        E -->|Valid| F[Inject user_id from token]
        E -->|Invalid| G[Return 401]
        D --> H[Request Context with user_id]
        F --> H
    end
    
    subgraph "Application Layer"
        H --> I[API Handlers]
        I --> J[Extract user_id from context]
        J --> K[Store Operations]
    end
    
    subgraph "Data Layer"
        K --> L[(Users Table)]
        K --> M[(Chunks + user_id)]
        K --> N[(Sessions + user_id)]
        K --> O[(Chat Messages + user_id)]
        K --> P[(Audit Log + user_id)]
    end
    
    subgraph "Auth Layer"
        Q[Auth Provider Interface] --> R[Userpass Auth]
        Q --> S[MFA Auth Stub]
        Q --> T[SSO Auth Stub]
        R --> U[(Session Tokens)]
        R --> L
    end
```

### Component Interactions

1. **Request Flow**: HTTP request → AuthMiddleware → Handler → Store → Database
2. **Auth Flow**: Login → Userpass Auth → Session Token → Cookie/Header
3. **Data Access**: Handler extracts user_id → Store filters by user_id → Returns user's data only
4. **Migration Flow**: Startup → Check schema version → Run migrations → Create default users

### Concurrent Multi-User Access

The design fully supports multiple users accessing the web server simultaneously:

**Concurrency Model**:
- **Stateless Request Handling**: Each HTTP request is independent with its own context
- **Per-Request User Context**: User identity is extracted from session token and stored in request context
- **Thread-Safe Authentication**: Session token validation is read-only and can be performed concurrently
- **Database Connection Pooling**: Go's `database/sql` package manages a pool of SQLite connections
- **User Data Isolation**: All queries filter by `user_id`, preventing cross-user data access

**SQLite Concurrency Configuration**:
```go
// Connection string with concurrency settings
db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
```

**WAL Mode Benefits**:
- **Write-Ahead Logging**: Enables concurrent reads while writes are in progress
- **Multiple Readers**: Unlimited concurrent read operations
- **Single Writer**: One write operation at a time (SQLite limitation)
- **Better Performance**: Reduced lock contention for read-heavy workloads

**Connection Pool Settings**:
```go
// Configure connection pool for concurrent access
db.SetMaxOpenConns(25)        // Maximum concurrent connections
db.SetMaxIdleConns(5)         // Keep connections ready
db.SetConnMaxLifetime(5 * time.Minute)
```

**Concurrency Patterns**:

1. **Read Operations** (Library, Search, Sessions):
   - Fully concurrent across all users
   - No locking required
   - Scales linearly with CPU cores

2. **Write Operations** (Ingest, Chat Messages):
   - Serialized by SQLite (one writer at a time)
   - Busy timeout (5 seconds) handles contention
   - Typical write latency: < 10ms

3. **Authentication Operations**:
   - Token validation: Read-only, fully concurrent
   - Login: Writes session token, serialized but fast
   - Logout: Deletes session token, serialized but fast

**Performance Characteristics**:
- **Concurrent Reads**: 100+ users can search/browse simultaneously
- **Mixed Workload**: 10-20 concurrent users with reads and writes
- **Write-Heavy**: 5-10 concurrent users ingesting documents
- **Bottleneck**: SQLite write serialization (single writer)

**Scaling Considerations**:
- **Phase 4**: SQLite with WAL mode suitable for 10-50 concurrent users
- **Phase 5+**: Consider PostgreSQL for 100+ concurrent users with heavy writes
- **Current Bottleneck**: LLM inference (Ollama) is slower than database operations

**Race Condition Prevention**:
- **Session Token Uniqueness**: crypto/rand ensures no token collisions
- **User ID Isolation**: Context-based user_id prevents cross-user access
- **Database Constraints**: Foreign keys and unique constraints enforced by SQLite
- **Transaction Safety**: Migrations run in transactions for atomicity


## Components and Interfaces

### 1. Configuration Package (`internal/config`)

**New Configuration Fields**:

```go
type Config struct {
    // ... existing fields ...
    UserMode string     `json:"user_mode"` // "single" or "multi"
    Auth     AuthConfig `json:"auth"`
}

type AuthConfig struct {
    Provider              string `json:"provider"`                 // "userpass", "mfa", "sso"
    SessionExpiryDays     int    `json:"session_expiry_days"`      // Default: 7
    LockoutThreshold      int    `json:"lockout_threshold"`        // Default: 5
    LockoutDurationMinutes int   `json:"lockout_duration_minutes"` // Default: 15
}
```

**Environment Variable Overrides**:
- `NOODEXX_USER_MODE`: Override user_mode setting
- `NOODEXX_AUTH_PROVIDER`: Override auth provider

**Default Behavior**:
- If `user_mode` not specified: defaults to "single"
- If `auth.provider` not specified: defaults to "userpass"
- Creates default config.json with single-user mode if file doesn't exist

### 2. Authentication Package (`internal/auth`)

**Auth Provider Interface**:

```go
package auth

import (
    "context"
    "time"
)

// Provider defines the authentication interface
type Provider interface {
    // Login authenticates credentials and returns a session token
    Login(ctx context.Context, username, password string) (token string, err error)
    
    // Logout invalidates a session token
    Logout(ctx context.Context, token string) error
    
    // ValidateToken verifies a token and returns the user_id
    ValidateToken(ctx context.Context, token string) (userID int64, err error)
    
    // RefreshToken extends a session token's expiration
    RefreshToken(ctx context.Context, token string) (newToken string, err error)
}

// GetProvider returns the configured auth provider
func GetProvider(providerType string, store Store) (Provider, error)
```


**Userpass Auth Implementation**:

```go
package auth

import (
    "context"
    "crypto/rand"
    "encoding/base64"
    "fmt"
    "time"
    
    "golang.org/x/crypto/bcrypt"
)

type UserpassAuth struct {
    store           Store
    sessionExpiry   time.Duration
    lockoutThreshold int
    lockoutDuration time.Duration
}

func NewUserpassAuth(store Store, config AuthConfig) *UserpassAuth {
    return &UserpassAuth{
        store:           store,
        sessionExpiry:   time.Duration(config.SessionExpiryDays) * 24 * time.Hour,
        lockoutThreshold: config.LockoutThreshold,
        lockoutDuration: time.Duration(config.LockoutDurationMinutes) * time.Minute,
    }
}

func (u *UserpassAuth) Login(ctx context.Context, username, password string) (string, error) {
    // Check if account is locked
    if locked, until := u.store.IsAccountLocked(ctx, username); locked {
        return "", fmt.Errorf("account locked until %s", until.Format(time.RFC3339))
    }
    
    // Validate credentials
    user, err := u.store.ValidateCredentials(ctx, username, password)
    if err != nil {
        // Record failed attempt
        u.store.RecordFailedLogin(ctx, username)
        return "", fmt.Errorf("invalid credentials")
    }
    
    // Generate secure session token
    token, err := generateSecureToken(32)
    if err != nil {
        return "", fmt.Errorf("failed to generate token: %w", err)
    }
    
    // Store session token
    expiresAt := time.Now().Add(u.sessionExpiry)
    if err := u.store.CreateSessionToken(ctx, token, user.ID, expiresAt); err != nil {
        return "", fmt.Errorf("failed to create session: %w", err)
    }
    
    // Update last login timestamp
    u.store.UpdateLastLogin(ctx, user.ID)
    
    // Clear failed login attempts
    u.store.ClearFailedLogins(ctx, username)
    
    return token, nil
}

func generateSecureToken(length int) (string, error) {
    bytes := make([]byte, length)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return base64.URLEncoding.EncodeToString(bytes), nil
}
```

**MFA and SSO Stubs**:

```go
type MFAAuth struct{}

func (m *MFAAuth) Login(ctx context.Context, username, password string) (string, error) {
    return "", fmt.Errorf("MFA authentication not yet implemented")
}

type SSOAuth struct{}

func (s *SSOAuth) Login(ctx context.Context, username, password string) (string, error) {
    return "", fmt.Errorf("SSO authentication not yet implemented")
}
```


### 3. Store Package (`internal/store`)

**Database Abstraction Layer**:

To support future database migrations (e.g., PostgreSQL for larger deployments), we introduce a `DataStore` interface that abstracts all database operations. The SQLite implementation satisfies this interface, and future implementations (PostgreSQL, MySQL) can be added without changing application code.

```go
package store

import (
    "context"
    "time"
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
    SaveChatMessage(ctx context.Context, userID int64, sessionID, role, content string) error
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
```

**SQLite Implementation**:

```go
// SQLiteStore implements DataStore interface using SQLite
type SQLiteStore struct {
    db *sql.DB
}

// NewSQLiteStore creates a new SQLite-backed DataStore
func NewSQLiteStore(path string) (*SQLiteStore, error) {
    // Enable WAL mode for concurrent access and busy timeout for write contention
    db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }
    
    // Configure connection pool for concurrent multi-user access
    db.SetMaxOpenConns(25)                      // Support up to 25 concurrent connections
    db.SetMaxIdleConns(5)                       // Keep 5 connections ready
    db.SetConnMaxLifetime(5 * time.Minute)      // Recycle connections periodically
    
    // Test the connection
    if err := db.Ping(); err != nil {
        db.Close()
        return nil, fmt.Errorf("failed to ping database: %w", err)
    }
    
    store := &SQLiteStore{db: db}
    
    // Run migrations
    if err := store.runMigrations(context.Background()); err != nil {
        db.Close()
        return nil, fmt.Errorf("failed to run migrations: %w", err)
    }
    
    return store, nil
}

// Close closes the database connection
func (s *SQLiteStore) Close() error {
    if s.db != nil {
        return s.db.Close()
    }
    return nil
}
```

**Store Method Signatures** (implementing DataStore interface):

**Store Method Signatures** (implementing DataStore interface):

```go
// User Management Methods
func (s *SQLiteStore) CreateUser(ctx context.Context, username, password, email string, isAdmin, mustChangePassword bool) (int64, error)
func (s *SQLiteStore) GetUserByUsername(ctx context.Context, username string) (*User, error)
func (s *SQLiteStore) GetUserByID(ctx context.Context, userID int64) (*User, error)
func (s *SQLiteStore) ValidateCredentials(ctx context.Context, username, password string) (*User, error)
func (s *SQLiteStore) UpdatePassword(ctx context.Context, userID int64, newPassword string) error
func (s *SQLiteStore) UpdateLastLogin(ctx context.Context, userID int64) error
func (s *SQLiteStore) ListUsers(ctx context.Context) ([]User, error)
func (s *SQLiteStore) DeleteUser(ctx context.Context, userID int64) error

// Session Token Management Methods
func (s *SQLiteStore) CreateSessionToken(ctx context.Context, token string, userID int64, expiresAt time.Time) error
func (s *SQLiteStore) GetSessionToken(ctx context.Context, token string) (*SessionToken, error)
func (s *SQLiteStore) DeleteSessionToken(ctx context.Context, token string) error
func (s *SQLiteStore) CleanupExpiredTokens(ctx context.Context) error

// Account Lockout Methods
func (s *SQLiteStore) RecordFailedLogin(ctx context.Context, username string) error
func (s *SQLiteStore) ClearFailedLogins(ctx context.Context, username string) error
func (s *SQLiteStore) IsAccountLocked(ctx context.Context, username string) (bool, time.Time)

// User-Scoped Data Access Methods
func (s *SQLiteStore) SaveChunk(ctx context.Context, userID int64, source, text string, embedding []float32, tags []string, summary string) error
func (s *SQLiteStore) SearchByUser(ctx context.Context, userID int64, queryVec []float32, topK int) ([]Chunk, error)
func (s *SQLiteStore) LibraryByUser(ctx context.Context, userID int64) ([]LibraryEntry, error)
func (s *SQLiteStore) GetUserSessions(ctx context.Context, userID int64) ([]Session, error)
func (s *SQLiteStore) GetSessionOwner(ctx context.Context, sessionID string) (int64, error)
func (s *SQLiteStore) GetUserSkills(ctx context.Context, userID int64) ([]Skill, error)
func (s *SQLiteStore) GetWatchedFoldersByUser(ctx context.Context, userID int64) ([]WatchedFolder, error)
```

**Usage in Application Code**:

```go
// In main.go or initialization code
store, err := store.NewDataStore("sqlite", "noodexx.db")
if err != nil {
    log.Fatal(err)
}
defer store.Close()

// All application code uses the DataStore interface
// This allows swapping SQLite for PostgreSQL without changing application logic
server := api.NewServer(store, config, authProvider)
```

**Benefits of Abstraction Layer**:
1. **Future-Proof**: Easy migration to PostgreSQL, MySQL, or other databases
2. **Testability**: Mock implementations for unit testing
3. **Flexibility**: Different databases for different deployment scenarios
4. **No Vendor Lock-In**: Not tied to SQLite forever
5. **Clean Architecture**: Application logic doesn't depend on database implementation

**Password Hashing**:

```go
func hashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return "", err
    }
    return string(bytes), nil
}

func checkPasswordHash(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}
```


### 4. API Package (`internal/api`)

**Authentication Middleware**:

```go
package api

import (
    "context"
    "net/http"
    "strings"
)

type contextKey string

const UserIDKey contextKey = "user_id"

func (s *Server) AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Skip auth for public endpoints
        if isPublicEndpoint(r.URL.Path) {
            next.ServeHTTP(w, r)
            return
        }
        
        // Single-user mode: inject local-default user
        if s.config.UserMode == "single" {
            user, err := s.store.GetUserByUsername(r.Context(), "local-default")
            if err != nil {
                http.Error(w, "System error", http.StatusInternalServerError)
                return
            }
            ctx := context.WithValue(r.Context(), UserIDKey, user.ID)
            next.ServeHTTP(w, r.WithContext(ctx))
            return
        }
        
        // Multi-user mode: validate token
        token := extractToken(r)
        if token == "" {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        
        userID, err := s.authProvider.ValidateToken(r.Context(), token)
        if err != nil {
            http.Error(w, "Invalid session", http.StatusUnauthorized)
            return
        }
        
        ctx := context.WithValue(r.Context(), UserIDKey, userID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func extractToken(r *http.Request) string {
    // Try Authorization header first
    auth := r.Header.Get("Authorization")
    if strings.HasPrefix(auth, "Bearer ") {
        return strings.TrimPrefix(auth, "Bearer ")
    }
    
    // Fall back to cookie
    cookie, err := r.Cookie("session_token")
    if err == nil {
        return cookie.Value
    }
    
    return ""
}

func isPublicEndpoint(path string) bool {
    publicPaths := []string{"/login", "/register", "/static/"}
    for _, p := range publicPaths {
        if strings.HasPrefix(path, p) {
            return true
        }
    }
    return false
}

func GetUserID(ctx context.Context) (int64, error) {
    userID, ok := ctx.Value(UserIDKey).(int64)
    if !ok {
        return 0, fmt.Errorf("user_id not found in context")
    }
    return userID, nil
}
```


**New API Endpoints**:

```go
// Authentication endpoints
POST   /api/login          - Authenticate user and return session token
POST   /api/logout         - Invalidate session token
POST   /api/register       - Create new user account (if enabled)
POST   /api/change-password - Change user password

// User management endpoints (admin only)
GET    /api/users          - List all users
POST   /api/users          - Create new user
PUT    /api/users/:id      - Update user details
DELETE /api/users/:id      - Delete user
POST   /api/users/:id/reset-password - Reset user password

// Modified existing endpoints (now user-scoped)
GET    /api/library        - Returns only user's documents
POST   /api/search         - Searches only user's documents
GET    /api/sessions       - Returns only user's chat sessions
GET    /api/skills         - Returns only user's skills
GET    /api/watched-folders - Returns only user's watched folders
```

**Handler Example with User Context**:

```go
func (s *Server) handleLibrary(w http.ResponseWriter, r *http.Request) {
    userID, err := GetUserID(r.Context())
    if err != nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    entries, err := s.store.LibraryByUser(r.Context(), userID)
    if err != nil {
        http.Error(w, "Failed to fetch library", http.StatusInternalServerError)
        return
    }
    
    json.NewEncoder(w).Encode(entries)
}
```


## Data Models

### Database Schema Changes

**New Tables**:

```sql
-- Users table
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    email TEXT UNIQUE,
    is_admin BOOLEAN DEFAULT 0,
    must_change_password BOOLEAN DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login TIMESTAMP
);

-- Session tokens table
CREATE TABLE session_tokens (
    token TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Failed login attempts table (for account lockout)
CREATE TABLE failed_logins (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL,
    attempted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Sessions metadata table
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL,
    title TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_message_at TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Skills metadata table
CREATE TABLE skills (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    path TEXT NOT NULL,
    enabled BOOLEAN DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
```


**Modified Tables** (adding user_id and visibility):

```sql
-- Add user_id to chunks table
ALTER TABLE chunks ADD COLUMN user_id INTEGER REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE chunks ADD COLUMN visibility TEXT DEFAULT 'private' CHECK(visibility IN ('private', 'shared', 'public'));
ALTER TABLE chunks ADD COLUMN shared_with TEXT; -- Comma-separated user IDs

-- Add user_id to chat_messages table
ALTER TABLE chat_messages ADD COLUMN user_id INTEGER REFERENCES users(id) ON DELETE CASCADE;

-- Add user_id to audit_log table
ALTER TABLE audit_log ADD COLUMN user_id INTEGER REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE audit_log ADD COLUMN username TEXT;

-- Add user_id to watched_folders table
ALTER TABLE watched_folders ADD COLUMN user_id INTEGER REFERENCES users(id) ON DELETE CASCADE;
```

**Indexes for Performance**:

```sql
CREATE INDEX idx_chunks_user ON chunks(user_id);
CREATE INDEX idx_chunks_visibility ON chunks(visibility);
CREATE INDEX idx_messages_user ON chat_messages(user_id);
CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_skills_user ON skills(user_id);
CREATE INDEX idx_watched_folders_user ON watched_folders(user_id);
CREATE INDEX idx_audit_user ON audit_log(user_id);
CREATE INDEX idx_session_tokens_user ON session_tokens(user_id);
CREATE INDEX idx_session_tokens_expires ON session_tokens(expires_at);
CREATE INDEX idx_failed_logins_username ON failed_logins(username);
CREATE INDEX idx_failed_logins_attempted ON failed_logins(attempted_at);
```

### Go Data Models

```go
type User struct {
    ID                 int64
    Username           string
    PasswordHash       string
    Email              string
    IsAdmin            bool
    MustChangePassword bool
    CreatedAt          time.Time
    LastLogin          time.Time
}

type SessionToken struct {
    Token     string
    UserID    int64
    CreatedAt time.Time
    ExpiresAt time.Time
}

type Skill struct {
    ID        int64
    UserID    int64
    Name      string
    Path      string
    Enabled   bool
    CreatedAt time.Time
}
```


## Migration Strategy

### Phase 3 to Phase 4 Migration

The migration must be atomic, preserve all existing data, and create the necessary user infrastructure. The migration executes in a single transaction to ensure consistency.

**Migration Steps**:

1. **Create New Tables**:
   - users
   - session_tokens
   - failed_logins
   - sessions (metadata)
   - skills (metadata)

2. **Create Default Users**:
   - In single-user mode: Create "local-default" user with is_admin=true
   - In multi-user mode: Create "admin" user with random password and must_change_password=true

3. **Add Columns to Existing Tables**:
   - Add user_id to: chunks, chat_messages, audit_log, watched_folders
   - Add visibility and shared_with to chunks
   - Add username to audit_log

4. **Migrate Existing Data**:
   - Set user_id = local-default user's ID for all existing records
   - Set visibility = 'private' for all existing chunks

5. **Create Foreign Keys and Indexes**:
   - Foreign key constraints from user_id columns to users.id
   - Indexes on user_id columns for performance

6. **Verify Migration**:
   - Count records before and after
   - Verify all user_id values are set
   - Verify foreign key constraints

**Migration Code Structure**:

```go
func (s *Store) migrateToPhase4(ctx context.Context, tx *sql.Tx, userMode string) error {
    // Step 1: Create new tables
    if err := createUsersTable(ctx, tx); err != nil {
        return err
    }
    
    // Step 2: Create default user(s)
    localDefaultID, err := createDefaultUsers(ctx, tx, userMode)
    if err != nil {
        return err
    }
    
    // Step 3: Add columns to existing tables
    if err := addUserIDColumns(ctx, tx); err != nil {
        return err
    }
    
    // Step 4: Migrate existing data
    if err := migrateExistingData(ctx, tx, localDefaultID); err != nil {
        return err
    }
    
    // Step 5: Create constraints and indexes
    if err := createPhase4Indexes(ctx, tx); err != nil {
        return err
    }
    
    return nil
}
```


**Default Admin Password Generation**:

```go
func generateSecurePassword(length int) (string, error) {
    const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
    password := make([]byte, length)
    
    for i := range password {
        num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
        if err != nil {
            return "", err
        }
        password[i] = charset[num.Int64()]
    }
    
    return string(password), nil
}

func createDefaultUsers(ctx context.Context, tx *sql.Tx, userMode string) (int64, error) {
    // Always create local-default user
    localDefaultPassword, _ := generateSecurePassword(16)
    localDefaultHash, _ := bcrypt.GenerateFromPassword([]byte(localDefaultPassword), bcrypt.DefaultCost)
    
    result, err := tx.ExecContext(ctx, `
        INSERT INTO users (username, password_hash, email, is_admin, must_change_password)
        VALUES (?, ?, ?, ?, ?)
    `, "local-default", string(localDefaultHash), "", true, false)
    if err != nil {
        return 0, err
    }
    
    localDefaultID, _ := result.LastInsertId()
    
    // In multi-user mode, also create admin user
    if userMode == "multi" {
        adminPassword, _ := generateSecurePassword(16)
        adminHash, _ := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
        
        _, err := tx.ExecContext(ctx, `
            INSERT INTO users (username, password_hash, email, is_admin, must_change_password)
            VALUES (?, ?, ?, ?, ?)
        `, "admin", string(adminHash), "", true, true)
        if err != nil {
            return 0, err
        }
        
        // Log admin password to console
        fmt.Printf("\n===========================================\n")
        fmt.Printf("DEFAULT ADMIN ACCOUNT CREATED\n")
        fmt.Printf("Username: admin\n")
        fmt.Printf("Temporary Password: %s\n", adminPassword)
        fmt.Printf("You MUST change this password on first login\n")
        fmt.Printf("===========================================\n\n")
    }
    
    return localDefaultID, nil
}
```

### Rollback Strategy

If migration fails at any step:
1. Transaction is automatically rolled back
2. Database remains in Phase 3 state
3. Descriptive error is logged
4. Application refuses to start
5. User must fix the issue and retry

### Backward Compatibility

**Single-User Mode Compatibility**:
- No UI changes visible to user
- No authentication required
- All operations use local-default user automatically
- Existing workflows unchanged
- Configuration defaults to single-user mode

**Data Compatibility**:
- All Phase 3 data is preserved
- Embeddings remain unchanged
- Chat history remains unchanged
- Audit logs remain unchanged
- Watched folders remain unchanged


## Correctness Properties

A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.

### Property 1: Configuration Loading Preserves User Mode

For any valid configuration file containing a user_mode field with value "single" or "multi", loading the configuration should result in the Config struct having that exact user_mode value.

**Validates: Requirements 1.1**

### Property 2: Username Uniqueness Enforcement

For any two user creation attempts with the same username, the second attempt should fail with a unique constraint violation error.

**Validates: Requirements 2.2**

### Property 3: Email Uniqueness Enforcement

For any two user creation attempts with the same email address, the second attempt should fail with a unique constraint violation error.

**Validates: Requirements 2.3**

### Property 4: Password Hashing Round Trip

For any valid password string, after hashing with bcrypt and then validating with bcrypt.CompareHashAndPassword, the validation should succeed. Additionally, the stored hash should never equal the plaintext password.

**Validates: Requirements 2.10**

### Property 5: Credential Validation Correctness

For any user account, ValidateCredentials should return success when given the correct password and should fail when given any incorrect password.

**Validates: Requirements 2.12, 7.3**

### Property 6: Last Login Timestamp Update

For any user, after calling UpdateLastLogin, the user's last_login timestamp should be greater than or equal to the timestamp before the update.

**Validates: Requirements 2.13**

### Property 7: Password Update Clears Must-Change Flag

For any user with must_change_password=true, after calling UpdatePassword with a new password, the user's must_change_password field should be false and the new password should validate successfully.

**Validates: Requirements 2.14**


### Property 8: Document Visibility Defaults to Private

For any newly created chunk where visibility is not explicitly specified, the chunk's visibility field should be set to "private".

**Validates: Requirements 4.2**

### Property 9: Library Query Respects Visibility Rules

For any user and any chunk in the database, the chunk should appear in the user's library results if and only if: (1) the chunk's user_id matches the querying user's ID, OR (2) the chunk's visibility is "public", OR (3) the querying user's ID appears in the chunk's shared_with list.

**Validates: Requirements 4.4**

### Property 10: RAG Search Respects Visibility Rules

For any user and any search query, all returned chunks should satisfy the visibility rules: the chunk's user_id matches the user's ID, OR visibility is "public", OR the user's ID is in shared_with.

**Validates: Requirements 4.5**

### Property 11: Session Ownership Association

For any user creating a new chat session, the created session's user_id should match the creating user's ID.

**Validates: Requirements 5.2**

### Property 12: Session List Filtering

For any user, GetUserSessions should return only sessions where the session's user_id matches the querying user's ID.

**Validates: Requirements 5.3**

### Property 13: Session Access Authorization

For any user attempting to load a session, if the session's user_id does not match the user's ID, the operation should fail with an authorization error.

**Validates: Requirements 5.4**

### Property 14: Audit Log User Context

For any operation that creates an audit log entry, the entry should contain both the user_id and username of the user who performed the operation.

**Validates: Requirements 6.1, 6.2**

### Property 15: Audit Log User Filtering

For any user, GetAuditLogByUser should return only audit entries where the entry's user_id matches the specified user's ID.

**Validates: Requirements 6.4**


### Property 16: Session Token Generation Uniqueness

For any two calls to generateSecureToken using crypto/rand, the generated tokens should be different with overwhelming probability (collision rate < 1 in 2^128).

**Validates: Requirements 7.4**

### Property 17: Session Token Storage and Retrieval

For any successful login, a session token should be created in the session_tokens table with the correct user_id and an expires_at timestamp set to the configured session expiry duration from the creation time.

**Validates: Requirements 7.5**

### Property 18: Token Validation Correctness

For any valid, non-expired session token, ValidateToken should return the correct user_id. For any invalid or expired token, ValidateToken should return an error.

**Validates: Requirements 7.7**

### Property 19: Logout Invalidates Token

For any valid session token, after calling Logout with that token, subsequent calls to ValidateToken with the same token should fail.

**Validates: Requirements 7.8**

### Property 20: Token Extraction from Multiple Sources

For any HTTP request containing a valid session token in either the Authorization header (as "Bearer <token>") or in a session_token cookie, the AuthMiddleware should successfully extract the token.

**Validates: Requirements 8.2**

### Property 21: Context Injection in Multi-User Mode

For any authenticated request in multi-user mode with a valid session token, after passing through AuthMiddleware, the request context should contain the user_id associated with that token.

**Validates: Requirements 8.4**

### Property 22: Username Uniqueness in User Creation

For any admin attempting to create a new user, if a user with that username already exists, the creation should fail with a validation error.

**Validates: Requirements 10.6**


### Property 23: Skills User Scoping

For any user, GetUserSkills should return only skills where the skill's user_id matches the querying user's ID.

**Validates: Requirements 11.2**

### Property 24: Skill Execution Authorization

For any user attempting to execute a skill, if the skill's user_id does not match the user's ID, the execution should fail with an authorization error.

**Validates: Requirements 11.3**

### Property 25: Skill Execution Context

For any skill triggered by a keyword or event, the skill should execute with the user context of the skill's owning user (the user_id associated with the skill).

**Validates: Requirements 11.6**

### Property 26: Watched Folder Ownership

For any user creating a watched folder, the created folder's user_id should match the creating user's ID.

**Validates: Requirements 12.1**

### Property 27: Watched Folder Chunk Ownership

For any file ingested from a watched folder, all created chunks should have a user_id matching the watched folder's user_id.

**Validates: Requirements 12.2, 12.5**

### Property 28: Watched Folder Access Control

For any user attempting to modify or remove a watched folder, if the folder's user_id does not match the user's ID, the operation should fail with an authorization error.

**Validates: Requirements 12.6**

### Property 29: Configuration Validation for Auth Provider

For any configuration with an auth.provider field, if the value is not one of "userpass", "mfa", or "sso", the configuration validation should fail with an error.

**Validates: Requirements 15.8**

### Property 30: Migration Data Preservation

For any Phase 3 database with existing chunks, chat messages, audit log entries, and watched folders, after migration to Phase 4, the count of records in each table should be unchanged and all embedding vectors should be byte-identical.

**Validates: Requirements 14.5, 14.6**


## Error Handling

### Authentication Errors

**Invalid Credentials**:
- HTTP 401 Unauthorized
- Generic error message: "Invalid username or password"
- Log failed attempt with username and timestamp
- Increment failed login counter

**Account Locked**:
- HTTP 403 Forbidden
- Error message: "Account temporarily locked. Try again at [timestamp]"
- Log lockout event

**Expired Session**:
- HTTP 401 Unauthorized
- Error message: "Session expired. Please log in again."
- Clear expired token from database

**Missing Token**:
- HTTP 401 Unauthorized
- Error message: "Authentication required"

### Authorization Errors

**Insufficient Permissions**:
- HTTP 403 Forbidden
- Error message: "You do not have permission to perform this action"
- Log unauthorized access attempt

**Resource Not Found or Unauthorized**:
- HTTP 404 Not Found (to avoid leaking existence)
- Error message: "Resource not found"
- Log access attempt with user context

### Data Validation Errors

**Duplicate Username**:
- HTTP 400 Bad Request
- Error message: "Username already exists"

**Duplicate Email**:
- HTTP 400 Bad Request
- Error message: "Email already registered"

**Invalid Password**:
- HTTP 400 Bad Request
- Error message: "Password must be at least 8 characters"

**Password Mismatch**:
- HTTP 400 Bad Request
- Error message: "Passwords do not match"

### Migration Errors

**Migration Failure**:
- Transaction rollback
- Log detailed error with stack trace
- Exit application with error code 1
- Error message: "Database migration failed: [details]"

**Schema Version Mismatch**:
- Refuse to start
- Error message: "Database schema version incompatible. Manual intervention required."

### Configuration Errors

**Invalid User Mode**:
- Error message: "user_mode must be 'single' or 'multi'"
- Use default value "single"

**Invalid Auth Provider**:
- Error message: "auth.provider must be 'userpass', 'mfa', or 'sso'"
- Exit application

**Unimplemented Auth Provider**:
- Error message: "MFA/SSO authentication not yet implemented"
- Exit application


## Testing Strategy

### Dual Testing Approach

This feature requires both unit tests and property-based tests for comprehensive coverage:

**Unit Tests**: Focus on specific examples, edge cases, and integration points
- Single-user mode initialization
- Multi-user mode initialization with admin account creation
- Login flow with must_change_password=true
- Account lockout after 5 failed attempts
- Migration from Phase 3 database
- Public endpoint access without authentication
- Admin user management operations
- MFA/SSO stub error responses

**Property-Based Tests**: Verify universal properties across all inputs
- Configuration loading with random valid values
- Password hashing and validation with random passwords
- Session token generation uniqueness
- User data isolation across random user sets
- Visibility rules with random ownership and sharing configurations
- Authorization checks with random user/resource combinations

### Property-Based Testing Configuration

**Framework**: Use [gopter](https://github.com/leanovate/gopter) for Go property-based testing

**Test Configuration**:
- Minimum 100 iterations per property test
- Each test tagged with feature name and property number
- Tag format: `// Feature: noodexx-phase-4-multiuser-foundation, Property N: [property text]`

**Example Property Test**:

```go
// Feature: noodexx-phase-4-multiuser-foundation, Property 2: Username Uniqueness Enforcement
func TestUsernameUniquenessProperty(t *testing.T) {
    properties := gopter.NewProperties(nil)
    
    properties.Property("duplicate usernames should fail", prop.ForAll(
        func(username string, password1 string, password2 string) bool {
            store := setupTestStore(t)
            defer store.Close()
            
            // Create first user
            _, err1 := store.CreateUser(context.Background(), username, password1, "", false, false)
            if err1 != nil {
                return false
            }
            
            // Attempt to create second user with same username
            _, err2 := store.CreateUser(context.Background(), username, password2, "", false, false)
            
            // Should fail with unique constraint error
            return err2 != nil && strings.Contains(err2.Error(), "unique")
        },
        gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
        gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 8 }),
        gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 8 }),
    ))
    
    properties.TestingRun(t, gopter.ConsoleReporter(false))
}
```


### Test Coverage Areas

**Authentication Package Tests**:
- Userpass authentication flow
- Token generation and validation
- Session expiration handling
- Account lockout mechanism
- Password hashing and verification
- MFA/SSO stub error responses

**Store Package Tests**:
- User CRUD operations
- Session token management
- User-scoped data queries
- Migration from Phase 3
- Foreign key constraint enforcement
- Index creation and performance

**API Package Tests**:
- AuthMiddleware in single-user mode
- AuthMiddleware in multi-user mode
- Token extraction from headers and cookies
- Context injection
- Public endpoint access
- Protected endpoint authorization
- User management endpoints (admin only)

**Config Package Tests**:
- User mode configuration loading
- Auth configuration loading
- Environment variable overrides
- Default value handling
- Validation of auth provider values

**Integration Tests**:
- End-to-end login flow
- End-to-end registration flow
- Password change enforcement
- Admin user management workflow
- Data isolation between users
- Backward compatibility with Phase 3

### Test Data Generators

For property-based tests, implement generators for:
- Random valid usernames (alphanumeric, 3-32 chars)
- Random valid passwords (8-128 chars, mixed case, numbers, symbols)
- Random email addresses
- Random session tokens
- Random user IDs
- Random visibility settings ("private", "shared", "public")
- Random user sets for sharing tests

### Performance Testing

**Load Testing Scenarios**:
- 100 concurrent users performing searches
- 1000 session token validations per second
- Migration of database with 100,000 chunks
- Library queries with 10,000 documents per user

**Performance Targets**:
- Token validation: < 10ms
- User-scoped library query: < 100ms for 1000 documents
- Migration: < 5 minutes for 100,000 records
- Login: < 200ms including bcrypt verification


## UI Changes

### Login Page (Multi-User Mode Only)

**Route**: `/` (root) when user_mode is "multi" and not authenticated

**Components**:
- Username input field
- Password input field (type="password")
- "Login" button
- "Register" link (if registration enabled)
- Error message display area

**Behavior**:
- On successful login with must_change_password=false: Set session cookie, redirect to `/dashboard`
- On successful login with must_change_password=true: Redirect to `/change-password`
- On failed login: Display error message, clear password field
- On account locked: Display lockout message with unlock time

**Implementation**:
```html
<form id="login-form">
    <input type="text" name="username" placeholder="Username" required>
    <input type="password" name="password" placeholder="Password" required>
    <button type="submit">Login</button>
    <div id="error-message"></div>
    <a href="/register">Create Account</a>
</form>
```

### Registration Page

**Route**: `/register`

**Components**:
- Username input field
- Email input field
- Password input field
- Confirm password input field
- "Create Account" button
- "Back to Login" link

**Validation**:
- Username: 3-32 characters, alphanumeric and underscore only
- Email: Valid email format
- Password: Minimum 8 characters
- Confirm password: Must match password

### Password Change Page

**Route**: `/change-password`

**Components**:
- New password input field
- Confirm new password input field
- "Change Password" button

**Behavior**:
- Required when must_change_password=true
- Cannot be bypassed (middleware redirects)
- On success: Set must_change_password=false, redirect to dashboard

### Settings Page Additions

**User Management Section** (Admin Only):

**Components**:
- User list table with columns: Username, Email, Admin, Created, Last Login
- "Create User" button
- "Edit" button per user row
- "Delete" button per user row
- "Reset Password" button per user row

**Create/Edit User Modal**:
- Username field
- Email field
- Password field (create only)
- Is Admin checkbox
- Must Change Password checkbox
- "Save" and "Cancel" buttons

**User Profile Section** (All Users):
- Display current username and email
- "Change Password" button
- "Logout" button


### Library Page Modifications

**Visibility Indicators**:
- Private documents: Lock icon
- Shared documents: People icon with count
- Public documents: Globe icon

**Share Button** (UI only, functionality in Phase 5):
- Displayed on each document card
- Disabled with tooltip: "Sharing coming in Phase 5"

### Navigation Bar Additions

**User Menu** (Multi-User Mode):
- Display current username
- Dropdown menu with:
  - "Settings"
  - "User Management" (admin only)
  - "Logout"

**Single-User Mode**:
- No user menu displayed
- No authentication UI visible

## Implementation Phases

### Phase 4.1: Database and Core Auth (Week 1)

- Implement database schema changes
- Create migration logic
- Implement User and SessionToken models
- Implement password hashing utilities
- Create users table and related tables
- Test migration with Phase 3 database

### Phase 4.2: Authentication Package (Week 1-2)

- Implement Auth Provider interface
- Implement Userpass Auth provider
- Implement session token management
- Implement account lockout logic
- Create MFA and SSO stubs
- Unit tests for auth package

### Phase 4.3: Middleware and API (Week 2)

- Implement AuthMiddleware
- Add user context to request handling
- Implement login/logout endpoints
- Implement registration endpoint
- Implement password change endpoint
- Update existing endpoints for user scoping

### Phase 4.4: Store User Scoping (Week 2-3)

- Implement user-scoped query methods
- Update SaveChunk to include user_id
- Update Search to filter by user_id
- Update Library to filter by user_id
- Implement visibility filtering
- Update audit logging with user context

### Phase 4.5: UI Implementation (Week 3)

- Create login page
- Create registration page
- Create password change page
- Add user management to settings
- Add visibility indicators to library
- Add user menu to navigation

### Phase 4.6: Testing and Documentation (Week 4)

- Write property-based tests
- Write integration tests
- Performance testing
- Update user documentation
- Create migration guide
- Security audit


## Security Considerations

### Password Security

**Hashing**:
- Use bcrypt with default cost (currently 10)
- Never store plaintext passwords
- Never log passwords
- Use constant-time comparison for password validation

**Password Requirements**:
- Minimum 8 characters
- Recommend mixed case, numbers, and symbols
- No maximum length (bcrypt handles truncation at 72 bytes)

**Password Reset**:
- Admin-initiated reset generates random password
- User must change password on next login
- Old password is immediately invalidated

### Session Security

**Token Generation**:
- Use crypto/rand for cryptographically secure randomness
- 32-byte tokens (256 bits of entropy)
- Base64 URL-safe encoding

**Token Storage**:
- Store tokens in database with expiration
- Index on expires_at for efficient cleanup
- Cascade delete on user deletion

**Token Transmission**:
- Support both Authorization header and cookie
- Cookie should be HttpOnly and Secure (in production)
- Cookie should have SameSite=Strict

**Session Expiration**:
- Default 7 days
- Configurable via auth.session_expiry_days
- Background job to clean up expired tokens

### Account Lockout

**Lockout Policy**:
- 5 failed attempts within 15 minutes triggers lockout
- Lockout duration: 15 minutes
- Failed attempts are cleared on successful login
- Lockout applies to username, not IP (to prevent account enumeration via timing)

**Lockout Storage**:
- Store failed attempts in failed_logins table
- Clean up old attempts (> 15 minutes) periodically
- Index on username and attempted_at for performance

### Authorization

**Principle of Least Privilege**:
- Users can only access their own data by default
- Admin flag required for user management operations
- Session ownership verified on every access
- Skill ownership verified on execution

**Data Isolation**:
- All queries filtered by user_id
- Foreign key constraints enforce referential integrity
- Cascade delete removes user data on user deletion

### SQL Injection Prevention

**Parameterized Queries**:
- All database queries use parameterized statements
- Never concatenate user input into SQL
- Use database/sql package's built-in protection

### CSRF Protection

**Token-Based Protection**:
- Session tokens serve as CSRF tokens
- Require token in Authorization header or cookie
- Validate token on all state-changing operations

### Privacy Mode Compatibility

**Single-User Mode**:
- Privacy mode still enforced
- Local-default user is the only user
- No external authentication
- All data remains local

**Multi-User Mode**:
- Privacy mode can still be enabled
- Restricts LLM provider to Ollama
- Enforces localhost-only Ollama endpoint
- All user data remains local


## Configuration Example

### Default config.json (Single-User Mode)

```json
{
  "user_mode": "single",
  "auth": {
    "provider": "userpass",
    "session_expiry_days": 7,
    "lockout_threshold": 5,
    "lockout_duration_minutes": 15
  },
  "provider": {
    "type": "ollama",
    "ollama_endpoint": "http://localhost:11434",
    "ollama_embed_model": "nomic-embed-text:latest",
    "ollama_chat_model": "noodexx-chat:latest"
  },
  "privacy": {
    "enabled": true
  },
  "folders": [],
  "logging": {
    "level": "info",
    "debug_enabled": true,
    "file": "debug.log",
    "max_size_mb": 10,
    "max_backups": 3
  },
  "guardrails": {
    "max_file_size_mb": 10,
    "allowed_extensions": [".txt", ".md", ".pdf", ".html"],
    "max_concurrent": 3,
    "pii_detection": "normal",
    "auto_summarize": true
  },
  "server": {
    "port": 8080,
    "bind_address": "127.0.0.1"
  }
}
```

### Multi-User Mode Configuration

```json
{
  "user_mode": "multi",
  "auth": {
    "provider": "userpass",
    "session_expiry_days": 7,
    "lockout_threshold": 5,
    "lockout_duration_minutes": 15
  },
  "server": {
    "port": 8080,
    "bind_address": "0.0.0.0"
  }
}
```

### Environment Variable Overrides

```bash
# Override user mode
export NOODEXX_USER_MODE=multi

# Override auth provider
export NOODEXX_AUTH_PROVIDER=userpass

# Override session expiry
export NOODEXX_SESSION_EXPIRY_DAYS=14
```

## Deployment Considerations

### Single-User Deployment

**Use Cases**:
- Personal knowledge base
- Local development
- Privacy-focused individual use

**Configuration**:
- user_mode: "single"
- bind_address: "127.0.0.1" (localhost only)
- privacy.enabled: true

**Characteristics**:
- No authentication UI
- No user management
- Identical to Phase 3 experience
- Maximum privacy

### Multi-User Deployment

**Use Cases**:
- Team knowledge base
- Shared research environment
- Multi-tenant deployments

**Configuration**:
- user_mode: "multi"
- bind_address: "0.0.0.0" (network accessible)
- Consider HTTPS reverse proxy (nginx, Caddy)

**Characteristics**:
- Authentication required
- User management available
- Data isolation enforced
- Admin capabilities

### Migration from Phase 3

**Steps**:
1. Backup existing noodexx.db
2. Update to Phase 4 binary
3. Update config.json (or leave as default single-user)
4. Start application (migration runs automatically)
5. Verify migration success in logs
6. Test existing functionality

**Rollback**:
1. Stop Phase 4 application
2. Restore Phase 3 binary
3. Restore backed-up database
4. Start Phase 3 application

## Future Extensibility

### Phase 5 Preparation

This design includes stubs and interfaces for Phase 5 features:

**Document Sharing**:
- shared_with column already exists
- Visibility filtering already implemented
- UI placeholders for share functionality

**MFA Authentication**:
- Auth Provider interface supports MFA
- MFA stub returns "not implemented"
- Configuration supports auth.provider = "mfa"

**SSO Authentication**:
- Auth Provider interface supports SSO
- SSO stub returns "not implemented"
- Configuration supports auth.provider = "sso"

### Extension Points

**Database Migration to PostgreSQL**:

The DataStore abstraction layer makes migrating to PostgreSQL straightforward:

1. **Implement PostgreSQLStore**:
```go
type PostgreSQLStore struct {
    db *sql.DB
}

func NewPostgreSQLStore(connectionString string) (*PostgreSQLStore, error) {
    db, err := sql.Open("postgres", connectionString)
    if err != nil {
        return nil, err
    }
    
    // Configure connection pool for high concurrency
    db.SetMaxOpenConns(100)
    db.SetMaxIdleConns(10)
    
    return &PostgreSQLStore{db: db}, nil
}

// Implement all DataStore interface methods...
```

2. **Update Configuration**:
```json
{
  "database": {
    "type": "postgres",
    "connection_string": "postgres://user:pass@localhost/noodexx"
  }
}
```

3. **Migration Strategy**:
   - Export SQLite data to SQL dump
   - Convert SQLite-specific syntax to PostgreSQL
   - Import into PostgreSQL
   - Update config and restart

**When to Migrate to PostgreSQL**:
- 50+ concurrent users
- Heavy write workload (multiple users ingesting simultaneously)
- Need for advanced features (full-text search, JSON queries)
- Enterprise deployment requirements

**Custom Auth Providers**:
- Implement Auth Provider interface
- Add to GetAuthProvider factory
- Add configuration validation

**Additional User Metadata**:
- Add columns to users table
- Update User model
- Update CreateUser and related methods

**Role-Based Access Control**:
- Add roles table
- Add user_roles junction table
- Extend authorization checks

**Audit Log Enhancements**:
- Add IP address tracking
- Add user agent tracking
- Add request duration tracking

