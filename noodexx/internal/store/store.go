package store

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"
	"unsafe"

	_ "modernc.org/sqlite"
)

// Store provides database operations for Noodexx
type Store struct {
	db *sql.DB
}

// NewStore creates a new Store instance and initializes the database
func NewStore(path string) (*Store, error) {
	// Add _pragma for better timestamp handling
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	store := &Store{db: db}

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
func (s *Store) SaveChunk(ctx context.Context, source, text string, embedding []float32, tags []string, summary string) error {
	// Serialize embedding to bytes
	embeddingBytes := serializeEmbedding(embedding)

	// Join tags into comma-separated string
	var tagsStr string
	if len(tags) > 0 {
		tagsStr = joinTags(tags)
	}

	query := `INSERT INTO chunks (source, text, embedding, tags, summary) VALUES (?, ?, ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, query, source, text, embeddingBytes, tagsStr, summary)
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

// DeleteSource removes all chunks for a given source
func (s *Store) DeleteSource(ctx context.Context, source string) error {
	query := `DELETE FROM chunks WHERE source = ?`
	_, err := s.db.ExecContext(ctx, query, source)
	if err != nil {
		return fmt.Errorf("failed to delete source: %w", err)
	}
	return nil
}

// SaveMessage persists a chat message to the database
func (s *Store) SaveMessage(ctx context.Context, sessionID, role, content string) error {
	query := `INSERT INTO chat_messages (session_id, role, content) VALUES (?, ?, ?)`
	_, err := s.db.ExecContext(ctx, query, sessionID, role, content)
	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}
	return nil
}

// GetSessionHistory retrieves all messages for a given session ID ordered by creation time
func (s *Store) GetSessionHistory(ctx context.Context, sessionID string) ([]ChatMessage, error) {
	query := `SELECT id, session_id, role, content, created_at FROM chat_messages WHERE session_id = ? ORDER BY created_at ASC`
	rows, err := s.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query session history: %w", err)
	}
	defer rows.Close()

	var messages []ChatMessage
	for rows.Next() {
		var msg ChatMessage
		var createdAtStr string
		err := rows.Scan(&msg.ID, &msg.SessionID, &msg.Role, &msg.Content, &createdAtStr)
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

// AddWatchedFolder adds a folder to the watched folders list
func (s *Store) AddWatchedFolder(ctx context.Context, path string) error {
	query := `INSERT INTO watched_folders (path) VALUES (?)`
	_, err := s.db.ExecContext(ctx, query, path)
	if err != nil {
		return fmt.Errorf("failed to add watched folder: %w", err)
	}
	return nil
}

// GetWatchedFolders returns all watched folders
func (s *Store) GetWatchedFolders(ctx context.Context) ([]WatchedFolder, error) {
	query := `SELECT id, path, active, last_scan FROM watched_folders ORDER BY path`
	rows, err := s.db.QueryContext(ctx, query)
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

// RemoveWatchedFolder removes a folder from the watched folders list
func (s *Store) RemoveWatchedFolder(ctx context.Context, path string) error {
	query := `DELETE FROM watched_folders WHERE path = ?`
	_, err := s.db.ExecContext(ctx, query, path)
	if err != nil {
		return fmt.Errorf("failed to remove watched folder: %w", err)
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
