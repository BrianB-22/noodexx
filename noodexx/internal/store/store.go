package store

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

// Store provides database operations for Noodexx
type Store struct {
	db       *sql.DB
	userMode string // "single" or "multi"
}

// NewStore creates a new Store instance and initializes the database
func NewStore(path string, userMode string) (*Store, error) {
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
		userMode: userMode,
	}

	// Run migrations
	if err := store.runMigrations(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return store, nil
}

// Close closes the database connection
func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// SaveChunk saves a text chunk with its embedding to the database
func (s *Store) SaveChunk(ctx context.Context, userID int64, source, text string, embedding []float32, tags []string, summary string) error {
	// Serialize embedding to bytes
	embeddingBytes := serializeEmbedding(embedding)

	// Join tags into comma-separated string
	var tagsStr string
	if len(tags) > 0 {
		tagsStr = joinTags(tags)
	}

	query := `INSERT INTO chunks (user_id, source, text, embedding, tags, summary, visibility) VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, query, userID, source, text, embeddingBytes, tagsStr, summary, "private")
	if err != nil {
		return fmt.Errorf("failed to save chunk: %w", err)
	}

	return nil
}

// Search performs vector similarity search and returns top K chunks
func (s *Store) Search(ctx context.Context, queryVec []float32, topK int) ([]Chunk, error) {
	// Get all chunks from database
	query := `SELECT id, source, text, embedding, tags, summary, created_at FROM chunks`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query chunks: %w", err)
	}
	defer rows.Close()

	// Calculate similarity scores for each chunk
	var scored []scoredChunk

	for rows.Next() {
		var c Chunk
		var embeddingBytes []byte
		var tagsStr sql.NullString
		var summary sql.NullString
		var createdAtStr string

		err := rows.Scan(&c.ID, &c.Source, &c.Text, &embeddingBytes, &tagsStr, &summary, &createdAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to scan chunk: %w", err)
		}

		// Deserialize embedding
		c.Embedding = deserializeEmbedding(embeddingBytes)

		// Parse tags
		if tagsStr.Valid && tagsStr.String != "" {
			c.Tags = splitTags(tagsStr.String)
		}

		// Set summary
		if summary.Valid {
			c.Summary = summary.String
		}

		// Parse timestamp
		if createdAtStr != "" {
			c.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr)
		}

		// Calculate cosine similarity
		score := cosineSimilarity(queryVec, c.Embedding)
		scored = append(scored, scoredChunk{chunk: c, score: score})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating chunks: %w", err)
	}

	// Sort by score descending
	sortByScore(scored)

	// Return top K
	var results []Chunk
	for i := 0; i < len(scored) && i < topK; i++ {
		results = append(results, scored[i].chunk)
	}

	return results, nil
}

// SearchByUser performs vector similarity search with user-scoped visibility filtering
// Returns chunks visible to the specified user: owned by user, public, or shared with user
func (s *Store) SearchByUser(ctx context.Context, userID int64, queryVec []float32, topK int) ([]Chunk, error) {
	// Query chunks with visibility filtering
	query := `
		SELECT id, source, text, embedding, tags, summary, created_at 
		FROM chunks
		WHERE user_id = ? 
			OR visibility = 'public'
			OR (',' || COALESCE(shared_with, '') || ',') LIKE '%,' || CAST(? AS TEXT) || ',%'
	`

	rows, err := s.db.QueryContext(ctx, query, userID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query chunks for user: %w", err)
	}
	defer rows.Close()

	// Calculate similarity scores for each chunk
	var scored []scoredChunk

	for rows.Next() {
		var c Chunk
		var embeddingBytes []byte
		var tagsStr sql.NullString
		var summary sql.NullString
		var createdAtStr string

		err := rows.Scan(&c.ID, &c.Source, &c.Text, &embeddingBytes, &tagsStr, &summary, &createdAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to scan chunk: %w", err)
		}

		// Deserialize embedding
		c.Embedding = deserializeEmbedding(embeddingBytes)

		// Parse tags
		if tagsStr.Valid && tagsStr.String != "" {
			c.Tags = splitTags(tagsStr.String)
		}

		// Set summary
		if summary.Valid {
			c.Summary = summary.String
		}

		// Parse timestamp
		if createdAtStr != "" {
			c.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr)
		}

		// Calculate cosine similarity
		score := cosineSimilarity(queryVec, c.Embedding)
		scored = append(scored, scoredChunk{chunk: c, score: score})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating chunks: %w", err)
	}

	// Sort by score descending
	sortByScore(scored)

	// Return top K
	var results []Chunk
	for i := 0; i < len(scored) && i < topK; i++ {
		results = append(results, scored[i].chunk)
	}

	return results, nil
}

// Library returns all unique sources with metadata
func (s *Store) Library(ctx context.Context) ([]LibraryEntry, error) {
	query := `
		SELECT 
			source,
			COUNT(*) as chunk_count,
			MAX(summary) as summary,
			MAX(tags) as tags,
			MIN(created_at) as created_at
		FROM chunks
		GROUP BY source
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query library: %w", err)
	}
	defer rows.Close()

	var entries []LibraryEntry
	for rows.Next() {
		var entry LibraryEntry
		var tagsStr sql.NullString
		var summary sql.NullString
		var createdAtStr string

		err := rows.Scan(&entry.Source, &entry.ChunkCount, &summary, &tagsStr, &createdAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to scan library entry: %w", err)
		}

		// Parse tags
		if tagsStr.Valid && tagsStr.String != "" {
			entry.Tags = splitTags(tagsStr.String)
		}

		// Set summary
		if summary.Valid {
			entry.Summary = summary.String
		}

		// Parse timestamp
		if createdAtStr != "" {
			entry.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr)
		}

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating library entries: %w", err)
	}

	return entries, nil
}

// LibraryByUser returns library entries visible to the specified user
// Filters by: user_id OR visibility="public" OR user_id in shared_with
func (s *Store) LibraryByUser(ctx context.Context, userID int64) ([]LibraryEntry, error) {
	query := `
		SELECT 
			source,
			COUNT(*) as chunk_count,
			MAX(summary) as summary,
			MAX(tags) as tags,
			MIN(created_at) as created_at
		FROM chunks
		WHERE user_id = ? 
			OR visibility = 'public'
			OR (',' || COALESCE(shared_with, '') || ',') LIKE '%,' || CAST(? AS TEXT) || ',%'
		GROUP BY source
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query library by user: %w", err)
	}
	defer rows.Close()

	var entries []LibraryEntry
	for rows.Next() {
		var entry LibraryEntry
		var tagsStr sql.NullString
		var summary sql.NullString
		var createdAtStr string

		err := rows.Scan(&entry.Source, &entry.ChunkCount, &summary, &tagsStr, &createdAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to scan library entry: %w", err)
		}

		// Parse tags
		if tagsStr.Valid && tagsStr.String != "" {
			entry.Tags = splitTags(tagsStr.String)
		}

		// Set summary
		if summary.Valid {
			entry.Summary = summary.String
		}

		// Parse timestamp
		if createdAtStr != "" {
			entry.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr)
		}

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating library entries: %w", err)
	}

	return entries, nil
}

// DeleteChunksBySource removes all chunks for a given source owned by the specified user
func (s *Store) DeleteChunksBySource(ctx context.Context, userID int64, source string) error {
	query := `DELETE FROM chunks WHERE source = ? AND user_id = ?`
	_, err := s.db.ExecContext(ctx, query, source, userID)
	if err != nil {
		return fmt.Errorf("failed to delete chunks by source: %w", err)
	}
	return nil
}

// SaveMessage persists a chat message to the database
// SaveChatMessage saves a chat message with user ownership and provider mode
func (s *Store) SaveChatMessage(ctx context.Context, userID int64, sessionID, role, content, providerMode string) error {
	// Start a transaction to update both chat_messages and sessions tables
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert the chat message with provider_mode
	query := `INSERT INTO chat_messages (session_id, role, content, user_id, provider_mode) VALUES (?, ?, ?, ?, ?)`
	_, err = tx.ExecContext(ctx, query, sessionID, role, content, userID, providerMode)
	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	// Update or create session metadata
	// Use INSERT OR REPLACE to handle both new sessions and updates
	sessionQuery := `
		INSERT INTO sessions (id, user_id, last_message_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET last_message_at = CURRENT_TIMESTAMP
	`
	_, err = tx.ExecContext(ctx, sessionQuery, sessionID, userID)
	if err != nil {
		return fmt.Errorf("failed to update session metadata: %w", err)
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// SaveMessage is deprecated, use SaveChatMessage instead
// Kept for backward compatibility
func (s *Store) SaveMessage(ctx context.Context, sessionID, role, content string) error {
	// Get local-default user for backward compatibility
	user, err := s.GetUserByUsername(ctx, "local-default")
	if err != nil {
		return fmt.Errorf("failed to get local-default user: %w", err)
	}
	return s.SaveChatMessage(ctx, user.ID, sessionID, role, content, "local")
}

// GetSessionHistory retrieves all messages for a given session ID ordered by creation time
func (s *Store) GetSessionHistory(ctx context.Context, sessionID string) ([]ChatMessage, error) {
	query := `SELECT id, session_id, role, content, COALESCE(provider_mode, 'local') as provider_mode, created_at FROM chat_messages WHERE session_id = ? ORDER BY created_at ASC`
	rows, err := s.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query session history: %w", err)
	}
	defer rows.Close()

	var messages []ChatMessage
	for rows.Next() {
		var msg ChatMessage
		var createdAtStr string
		err := rows.Scan(&msg.ID, &msg.SessionID, &msg.Role, &msg.Content, &msg.ProviderMode, &createdAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		// Parse timestamp
		if createdAtStr != "" {
			msg.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr)
		}
		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}

	return messages, nil
}

// ListSessions returns all unique session IDs with their most recent message timestamp
func (s *Store) ListSessions(ctx context.Context) ([]Session, error) {
	query := `
		SELECT 
			session_id,
			MAX(created_at) as last_message_at,
			COUNT(*) as message_count
		FROM chat_messages
		GROUP BY session_id
		ORDER BY last_message_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var session Session
		var lastMessageAtStr string
		err := rows.Scan(&session.ID, &lastMessageAtStr, &session.MessageCount)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		// Parse timestamp
		if lastMessageAtStr != "" {
			session.LastMessageAt, _ = time.Parse("2006-01-02 15:04:05", lastMessageAtStr)
		}
		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sessions: %w", err)
	}

	return sessions, nil
}

// GetUserSessions returns all sessions owned by a specific user
func (s *Store) GetUserSessions(ctx context.Context, userID int64) ([]Session, error) {
	query := `
		SELECT 
			s.id,
			s.title,
			s.created_at,
			s.last_message_at,
			COUNT(cm.id) as message_count
		FROM sessions s
		LEFT JOIN chat_messages cm ON s.id = cm.session_id
		WHERE s.user_id = ?
		GROUP BY s.id, s.title, s.created_at, s.last_message_at
		ORDER BY s.last_message_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user sessions: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var session Session
		var title sql.NullString
		var createdAtStr string
		var lastMessageAtStr sql.NullString
		err := rows.Scan(&session.ID, &title, &createdAtStr, &lastMessageAtStr, &session.MessageCount)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}

		// Parse timestamps
		if createdAtStr != "" {
			session.LastMessageAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr)
		}
		if lastMessageAtStr.Valid && lastMessageAtStr.String != "" {
			session.LastMessageAt, _ = time.Parse("2006-01-02 15:04:05", lastMessageAtStr.String)
		}

		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sessions: %w", err)
	}

	return sessions, nil
}

// GetSessionOwner returns the user_id of the session owner
func (s *Store) GetSessionOwner(ctx context.Context, sessionID string) (int64, error) {
	var userID int64
	query := `SELECT user_id FROM sessions WHERE id = ?`
	err := s.db.QueryRowContext(ctx, query, sessionID).Scan(&userID)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("session not found: %s", sessionID)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get session owner: %w", err)
	}
	return userID, nil
}

// GetSessionMessages retrieves all messages for a session with ownership verification
func (s *Store) GetSessionMessages(ctx context.Context, userID int64, sessionID string) ([]ChatMessage, error) {
	// First verify the session belongs to the user
	ownerID, err := s.GetSessionOwner(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if ownerID != userID {
		return nil, fmt.Errorf("access denied: session %s does not belong to user %d", sessionID, userID)
	}

	// Retrieve messages
	query := `
		SELECT id, session_id, role, content, COALESCE(provider_mode, 'local') as provider_mode, created_at 
		FROM chat_messages 
		WHERE session_id = ? AND user_id = ?
		ORDER BY created_at ASC
	`
	rows, err := s.db.QueryContext(ctx, query, sessionID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query session messages: %w", err)
	}
	defer rows.Close()

	var messages []ChatMessage
	for rows.Next() {
		var msg ChatMessage
		var createdAtStr string
		err := rows.Scan(&msg.ID, &msg.SessionID, &msg.Role, &msg.Content, &msg.ProviderMode, &createdAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		// Parse timestamp
		if createdAtStr != "" {
			msg.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr)
		}
		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}

	return messages, nil
}

// AddAuditEntry records an operation in the audit log
func (s *Store) AddAuditEntry(ctx context.Context, opType, details, userCtx string) error {
	query := `INSERT INTO audit_log (operation_type, details, user_context) VALUES (?, ?, ?)`
	_, err := s.db.ExecContext(ctx, query, opType, details, userCtx)
	if err != nil {
		return fmt.Errorf("failed to add audit entry: %w", err)
	}
	return nil
}

// GetAuditLog retrieves audit entries with optional filtering by type and date range
func (s *Store) GetAuditLog(ctx context.Context, opType string, from, to time.Time) ([]AuditEntry, error) {
	query := `SELECT id, timestamp, operation_type, details, user_context FROM audit_log WHERE 1=1`
	args := []interface{}{}

	// Add optional filters
	if opType != "" {
		query += ` AND operation_type = ?`
		args = append(args, opType)
	}

	if !from.IsZero() {
		query += ` AND timestamp >= ?`
		args = append(args, from)
	}

	if !to.IsZero() {
		query += ` AND timestamp <= ?`
		args = append(args, to)
	}

	query += ` ORDER BY timestamp DESC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit log: %w", err)
	}
	defer rows.Close()

	var entries []AuditEntry
	for rows.Next() {
		var entry AuditEntry
		var details, userCtx sql.NullString

		err := rows.Scan(&entry.ID, &entry.Timestamp, &entry.OperationType, &details, &userCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit entry: %w", err)
		}

		if details.Valid {
			entry.Details = details.String
		}
		if userCtx.Valid {
			entry.UserContext = userCtx.String
		}

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating audit entries: %w", err)
	}

	return entries, nil
}

// AddWatchedFolder adds a folder to the watched folders list for a specific user
func (s *Store) AddWatchedFolder(ctx context.Context, userID int64, path string) error {
	query := `INSERT INTO watched_folders (user_id, path) VALUES (?, ?)`
	_, err := s.db.ExecContext(ctx, query, userID, path)
	if err != nil {
		return fmt.Errorf("failed to add watched folder: %w", err)
	}
	return nil
}

// GetWatchedFolders returns all watched folders
func (s *Store) GetWatchedFolders(ctx context.Context) ([]WatchedFolder, error) {
	query := `SELECT id, user_id, path, active, last_scan FROM watched_folders ORDER BY path`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query watched folders: %w", err)
	}
	defer rows.Close()

	var folders []WatchedFolder
	for rows.Next() {
		var folder WatchedFolder
		var lastScanStr sql.NullString
		err := rows.Scan(&folder.ID, &folder.UserID, &folder.Path, &folder.Active, &lastScanStr)
		if err != nil {
			return nil, fmt.Errorf("failed to scan watched folder: %w", err)
		}
		// Parse timestamp - try multiple formats
		if lastScanStr.Valid && lastScanStr.String != "" {
			// Try ISO 8601 format first (what SQLite returns)
			folder.LastScan, err = time.Parse(time.RFC3339, lastScanStr.String)
			if err != nil {
				// Fall back to SQLite datetime format
				folder.LastScan, _ = time.Parse("2006-01-02 15:04:05", lastScanStr.String)
			}
		}
		folders = append(folders, folder)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating watched folders: %w", err)
	}

	return folders, nil
}

// RemoveWatchedFolder removes a folder from the watched folders list with ownership verification
func (s *Store) RemoveWatchedFolder(ctx context.Context, userID int64, folderID int64) error {
	query := `DELETE FROM watched_folders WHERE id = ? AND user_id = ?`
	result, err := s.db.ExecContext(ctx, query, folderID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove watched folder: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("watched folder not found or access denied: %d", folderID)
	}

	return nil
}

// Helper functions

// serializeEmbedding converts a float32 slice to bytes
func serializeEmbedding(embedding []float32) []byte {
	bytes := make([]byte, len(embedding)*4)
	for i, v := range embedding {
		bits := uint32(0)
		if v != 0 {
			// Convert float32 to uint32 bits
			bits = *(*uint32)(unsafe.Pointer(&v))
		}
		bytes[i*4] = byte(bits)
		bytes[i*4+1] = byte(bits >> 8)
		bytes[i*4+2] = byte(bits >> 16)
		bytes[i*4+3] = byte(bits >> 24)
	}
	return bytes
}

// deserializeEmbedding converts bytes back to float32 slice
func deserializeEmbedding(bytes []byte) []float32 {
	if len(bytes)%4 != 0 {
		return nil
	}
	embedding := make([]float32, len(bytes)/4)
	for i := 0; i < len(embedding); i++ {
		bits := uint32(bytes[i*4]) |
			uint32(bytes[i*4+1])<<8 |
			uint32(bytes[i*4+2])<<16 |
			uint32(bytes[i*4+3])<<24
		embedding[i] = *(*float32)(unsafe.Pointer(&bits))
	}
	return embedding
}

// joinTags converts a string slice to comma-separated string
func joinTags(tags []string) string {
	return strings.Join(tags, ",")
}

// splitTags converts a comma-separated string to string slice
func splitTags(tagsStr string) []string {
	if tagsStr == "" {
		return nil
	}
	tags := strings.Split(tagsStr, ",")
	// Trim whitespace from each tag
	for i := range tags {
		tags[i] = strings.TrimSpace(tags[i])
	}
	return tags
}

// cosineSimilarity computes the cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
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

// scoredChunk is a helper type for sorting chunks by similarity score
type scoredChunk struct {
	chunk Chunk
	score float64
}

// sortByScore sorts scored chunks by score in descending order
func sortByScore(scored []scoredChunk) {
	// Simple bubble sort for small datasets
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}
}

// User Management Methods

// CreateUser creates a new user with bcrypt password hashing
func (s *Store) CreateUser(ctx context.Context, username, password, email string, isAdmin, mustChangePassword bool) (int64, error) {
	// Hash the password using bcrypt
	passwordHash, err := hashPassword(password)
	if err != nil {
		return 0, fmt.Errorf("failed to hash password: %w", err)
	}

	query := `
		INSERT INTO users (username, password_hash, email, is_admin, must_change_password)
		VALUES (?, ?, ?, ?, ?)
	`
	result, err := s.db.ExecContext(ctx, query, username, passwordHash, email, isAdmin, mustChangePassword)
	if err != nil {
		return 0, fmt.Errorf("failed to create user: %w", err)
	}

	userID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get user ID: %w", err)
	}

	return userID, nil
}

// GetUserByUsername retrieves a user by username
func (s *Store) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	query := `
		SELECT id, username, password_hash, email, is_admin, must_change_password, created_at, last_login, COALESCE(dark_mode, 0) as dark_mode
		FROM users
		WHERE username = ?
	`

	var user User
	var lastLogin sql.NullTime

	err := s.db.QueryRowContext(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Email,
		&user.IsAdmin,
		&user.MustChangePassword,
		&user.CreatedAt,
		&lastLogin,
		&user.DarkMode,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found: %s", username)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if lastLogin.Valid {
		user.LastLogin = lastLogin.Time
	}

	return &user, nil
}

// GetUserByID retrieves a user by ID
func (s *Store) GetUserByID(ctx context.Context, userID int64) (*User, error) {
	query := `
		SELECT id, username, password_hash, email, is_admin, must_change_password, created_at, last_login, COALESCE(dark_mode, 0) as dark_mode
		FROM users
		WHERE id = ?
	`

	var user User
	var lastLogin sql.NullTime

	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Email,
		&user.IsAdmin,
		&user.MustChangePassword,
		&user.CreatedAt,
		&lastLogin,
		&user.DarkMode,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found: %d", userID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if lastLogin.Valid {
		user.LastLogin = lastLogin.Time
	}

	return &user, nil
}

// ValidateCredentials verifies username and password, returns user if valid
func (s *Store) ValidateCredentials(ctx context.Context, username, password string) (*User, error) {
	user, err := s.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Verify password using bcrypt
	if !checkPasswordHash(password, user.PasswordHash) {
		return nil, fmt.Errorf("invalid credentials")
	}

	return user, nil
}

// UpdatePassword updates a user's password and resets must_change_password flag
func (s *Store) UpdatePassword(ctx context.Context, userID int64, newPassword string) error {
	// Hash the new password using bcrypt
	passwordHash, err := hashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	query := `
		UPDATE users
		SET password_hash = ?, must_change_password = 0
		WHERE id = ?
	`

	_, err = s.db.ExecContext(ctx, query, passwordHash, userID)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// UpdateLastLogin updates the last_login timestamp for a user
func (s *Store) UpdateLastLogin(ctx context.Context, userID int64) error {
	query := `
		UPDATE users
		SET last_login = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	_, err := s.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	return nil
}

// UpdateUserDarkMode updates a user's dark mode preference
func (s *Store) UpdateUserDarkMode(ctx context.Context, userID int64, darkMode bool) error {
	query := `
		UPDATE users
		SET dark_mode = ?
		WHERE id = ?
	`

	_, err := s.db.ExecContext(ctx, query, darkMode, userID)
	if err != nil {
		return fmt.Errorf("failed to update dark mode: %w", err)
	}

	return nil
}

// GetUserDarkMode retrieves a user's dark mode preference
func (s *Store) GetUserDarkMode(ctx context.Context, userID int64) (bool, error) {
	query := `
		SELECT COALESCE(dark_mode, 0)
		FROM users
		WHERE id = ?
	`

	var darkMode bool
	err := s.db.QueryRowContext(ctx, query, userID).Scan(&darkMode)
	if err == sql.ErrNoRows {
		return false, fmt.Errorf("user not found: %d", userID)
	}
	if err != nil {
		return false, fmt.Errorf("failed to get dark mode: %w", err)
	}

	return darkMode, nil
}

// ListUsers returns all users in the system
func (s *Store) ListUsers(ctx context.Context) ([]User, error) {
	query := `
		SELECT id, username, password_hash, email, is_admin, must_change_password, created_at, last_login
		FROM users
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		var lastLogin sql.NullTime

		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.PasswordHash,
			&user.Email,
			&user.IsAdmin,
			&user.MustChangePassword,
			&user.CreatedAt,
			&lastLogin,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		if lastLogin.Valid {
			user.LastLogin = lastLogin.Time
		}

		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

// DeleteUser deletes a user from the system
// Note: Foreign key constraints will cascade delete user's data
func (s *Store) DeleteUser(ctx context.Context, userID int64) error {
	query := `DELETE FROM users WHERE id = ?`

	result, err := s.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found: %d", userID)
	}

	return nil
}

// Password hashing helper functions

// hashPassword hashes a password using bcrypt
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// checkPasswordHash verifies a password against a bcrypt hash
func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// Session Token Management Methods

// CreateSessionToken stores a new session token in the database
// The token is associated with a user and has an expiration time
func (s *Store) CreateSessionToken(ctx context.Context, token string, userID int64, expiresAt time.Time) error {
	query := `INSERT INTO session_tokens (token, user_id, expires_at) VALUES (?, ?, ?)`

	_, err := s.db.ExecContext(ctx, query, token, userID, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to create session token: %w", err)
	}

	return nil
}

// GetSessionToken retrieves a session token from the database
// Returns nil if the token doesn't exist or has expired
func (s *Store) GetSessionToken(ctx context.Context, token string) (*SessionToken, error) {
	query := `
		SELECT token, user_id, created_at, expires_at 
		FROM session_tokens 
		WHERE token = ?
	`

	var st SessionToken
	err := s.db.QueryRowContext(ctx, query, token).Scan(
		&st.Token,
		&st.UserID,
		&st.CreatedAt,
		&st.ExpiresAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Token not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session token: %w", err)
	}

	// Check if token has expired
	if time.Now().After(st.ExpiresAt) {
		return nil, nil // Token expired
	}

	return &st, nil
}

// DeleteSessionToken removes a session token from the database
// Used for logout functionality
func (s *Store) DeleteSessionToken(ctx context.Context, token string) error {
	query := `DELETE FROM session_tokens WHERE token = ?`

	_, err := s.db.ExecContext(ctx, query, token)
	if err != nil {
		return fmt.Errorf("failed to delete session token: %w", err)
	}

	return nil
}

// CleanupExpiredTokens removes all expired session tokens from the database
// This should be called periodically as a background job
func (s *Store) CleanupExpiredTokens(ctx context.Context) error {
	query := `DELETE FROM session_tokens WHERE expires_at < ?`

	result, err := s.db.ExecContext(ctx, query, time.Now())
	if err != nil {
		return fmt.Errorf("failed to cleanup expired tokens: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	// Log the number of tokens cleaned up (optional, for monitoring)
	if rowsAffected > 0 {
		fmt.Printf("Cleaned up %d expired session tokens\n", rowsAffected)
	}

	return nil
}

// RecordFailedLogin records a failed login attempt for the given username
// This is used for account lockout tracking
func (s *Store) RecordFailedLogin(ctx context.Context, username string) error {
	query := `INSERT INTO failed_logins (username, attempted_at) VALUES (?, ?)`

	_, err := s.db.ExecContext(ctx, query, username, time.Now())
	if err != nil {
		return fmt.Errorf("failed to record failed login: %w", err)
	}

	return nil
}

// ClearFailedLogins removes all failed login attempts for the given username
// This should be called after a successful login
func (s *Store) ClearFailedLogins(ctx context.Context, username string) error {
	query := `DELETE FROM failed_logins WHERE username = ?`

	_, err := s.db.ExecContext(ctx, query, username)
	if err != nil {
		return fmt.Errorf("failed to clear failed logins: %w", err)
	}

	return nil
}

// IsAccountLocked checks if an account is locked due to too many failed login attempts
// Returns true and the lockout expiration time if the account is locked
// An account is locked if there are 5 or more failed attempts within the last 15 minutes
func (s *Store) IsAccountLocked(ctx context.Context, username string) (bool, time.Time) {
	// Calculate the time threshold (15 minutes ago)
	threshold := time.Now().Add(-15 * time.Minute)

	// Count failed login attempts within the last 15 minutes
	query := `SELECT COUNT(*) FROM failed_logins WHERE username = ? AND attempted_at > ?`

	var count int
	err := s.db.QueryRowContext(ctx, query, username, threshold).Scan(&count)
	if err != nil {
		// If there's an error, assume not locked (fail open for availability)
		return false, time.Time{}
	}

	// If 5 or more attempts, account is locked
	if count >= 5 {
		// Find the timestamp of the 5th most recent attempt
		// The lockout expires 15 minutes after that attempt
		query := `SELECT attempted_at FROM failed_logins 
		          WHERE username = ? AND attempted_at > ?
		          ORDER BY attempted_at DESC
		          LIMIT 1 OFFSET 4`

		var fifthAttempt time.Time
		err := s.db.QueryRowContext(ctx, query, username, threshold).Scan(&fifthAttempt)
		if err != nil {
			// If we can't find the 5th attempt, use the threshold as a fallback
			return true, threshold.Add(15 * time.Minute)
		}

		// Lockout expires 15 minutes after the 5th attempt
		lockoutExpires := fifthAttempt.Add(15 * time.Minute)
		return true, lockoutExpires
	}

	return false, time.Time{}
}

// Skills Management Methods

// CreateSkill creates a new skill for a user
// Returns the skill ID on success
func (s *Store) CreateSkill(ctx context.Context, userID int64, name, path string, enabled bool) (int64, error) {
	query := `INSERT INTO skills (user_id, name, path, enabled) VALUES (?, ?, ?, ?)`

	result, err := s.db.ExecContext(ctx, query, userID, name, path, enabled)
	if err != nil {
		return 0, fmt.Errorf("failed to create skill: %w", err)
	}

	skillID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get skill ID: %w", err)
	}

	return skillID, nil
}

// GetUserSkills retrieves all skills owned by a specific user
func (s *Store) GetUserSkills(ctx context.Context, userID int64) ([]Skill, error) {
	query := `
		SELECT id, user_id, name, path, enabled, created_at
		FROM skills
		WHERE user_id = ?
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user skills: %w", err)
	}
	defer rows.Close()

	var skills []Skill
	for rows.Next() {
		var skill Skill
		err := rows.Scan(
			&skill.ID,
			&skill.UserID,
			&skill.Name,
			&skill.Path,
			&skill.Enabled,
			&skill.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan skill: %w", err)
		}
		skills = append(skills, skill)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating skills: %w", err)
	}

	return skills, nil
}

// UpdateSkillEnabled updates the enabled status of a skill with ownership verification
func (s *Store) UpdateSkillEnabled(ctx context.Context, userID int64, skillID int64, enabled bool) error {
	// First verify the skill belongs to the user
	var ownerID int64
	checkQuery := `SELECT user_id FROM skills WHERE id = ?`
	err := s.db.QueryRowContext(ctx, checkQuery, skillID).Scan(&ownerID)
	if err == sql.ErrNoRows {
		return fmt.Errorf("skill not found: %d", skillID)
	}
	if err != nil {
		return fmt.Errorf("failed to verify skill ownership: %w", err)
	}

	if ownerID != userID {
		return fmt.Errorf("access denied: skill %d does not belong to user %d", skillID, userID)
	}

	// Update the enabled status
	updateQuery := `UPDATE skills SET enabled = ? WHERE id = ? AND user_id = ?`
	result, err := s.db.ExecContext(ctx, updateQuery, enabled, skillID, userID)
	if err != nil {
		return fmt.Errorf("failed to update skill enabled status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("skill not found or access denied: %d", skillID)
	}

	return nil
}

// DeleteSkill deletes a skill with ownership verification
func (s *Store) DeleteSkill(ctx context.Context, userID int64, skillID int64) error {
	// First verify the skill belongs to the user
	var ownerID int64
	checkQuery := `SELECT user_id FROM skills WHERE id = ?`
	err := s.db.QueryRowContext(ctx, checkQuery, skillID).Scan(&ownerID)
	if err == sql.ErrNoRows {
		return fmt.Errorf("skill not found: %d", skillID)
	}
	if err != nil {
		return fmt.Errorf("failed to verify skill ownership: %w", err)
	}

	if ownerID != userID {
		return fmt.Errorf("access denied: skill %d does not belong to user %d", skillID, userID)
	}

	// Delete the skill
	deleteQuery := `DELETE FROM skills WHERE id = ? AND user_id = ?`
	result, err := s.db.ExecContext(ctx, deleteQuery, skillID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete skill: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("skill not found or access denied: %d", skillID)
	}

	return nil
}

// Watched Folders Management Methods

// GetWatchedFoldersByUser returns all watched folders for a specific user
func (s *Store) GetWatchedFoldersByUser(ctx context.Context, userID int64) ([]WatchedFolder, error) {
	query := `SELECT id, path, active, last_scan FROM watched_folders WHERE user_id = ? ORDER BY path`
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query watched folders: %w", err)
	}
	defer rows.Close()

	var folders []WatchedFolder
	for rows.Next() {
		var folder WatchedFolder
		var lastScanStr sql.NullString
		err := rows.Scan(&folder.ID, &folder.Path, &folder.Active, &lastScanStr)
		if err != nil {
			return nil, fmt.Errorf("failed to scan watched folder: %w", err)
		}
		// Parse timestamp - try multiple formats
		if lastScanStr.Valid && lastScanStr.String != "" {
			// Try ISO 8601 format first (what SQLite returns)
			folder.LastScan, err = time.Parse(time.RFC3339, lastScanStr.String)
			if err != nil {
				// Fall back to SQLite datetime format
				folder.LastScan, _ = time.Parse("2006-01-02 15:04:05", lastScanStr.String)
			}
		}
		folders = append(folders, folder)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating watched folders: %w", err)
	}

	return folders, nil
}

// Audit Log Methods

// LogAudit records an operation in the audit log with user context
func (s *Store) LogAudit(ctx context.Context, userID int64, username, operation, details string) error {
	query := `INSERT INTO audit_log (user_id, username, operation_type, details) VALUES (?, ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, query, userID, username, operation, details)
	if err != nil {
		return fmt.Errorf("failed to log audit entry: %w", err)
	}
	return nil
}

// GetAuditLogByUser retrieves audit entries for a specific user with optional limit
func (s *Store) GetAuditLogByUser(ctx context.Context, userID int64, limit int) ([]AuditEntry, error) {
	query := `
		SELECT id, timestamp, operation_type, details, user_context
		FROM audit_log
		WHERE user_id = ?
		ORDER BY timestamp DESC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit log: %w", err)
	}
	defer rows.Close()

	var entries []AuditEntry
	for rows.Next() {
		var entry AuditEntry
		var details, userCtx sql.NullString

		err := rows.Scan(&entry.ID, &entry.Timestamp, &entry.OperationType, &details, &userCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit entry: %w", err)
		}

		if details.Valid {
			entry.Details = details.String
		}
		if userCtx.Valid {
			entry.UserContext = userCtx.String
		}

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating audit entries: %w", err)
	}

	return entries, nil
}
