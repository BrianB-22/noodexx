package store

import (
	"context"
	"database/sql"
	"fmt"
)

// runMigrations executes all database migrations in a transaction
func (s *Store) runMigrations(ctx context.Context) error {
	// Start a transaction for atomic migrations
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin migration transaction: %w", err)
	}

	// Ensure transaction is rolled back on error
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Run each migration
	if err = createChunksTable(ctx, tx); err != nil {
		return fmt.Errorf("failed to create chunks table: %w", err)
	}

	if err = addChunksColumns(ctx, tx); err != nil {
		return fmt.Errorf("failed to add chunks columns: %w", err)
	}

	if err = createChatMessagesTable(ctx, tx); err != nil {
		return fmt.Errorf("failed to create chat_messages table: %w", err)
	}

	if err = createAuditLogTable(ctx, tx); err != nil {
		return fmt.Errorf("failed to create audit_log table: %w", err)
	}

	if err = createWatchedFoldersTable(ctx, tx); err != nil {
		return fmt.Errorf("failed to create watched_folders table: %w", err)
	}

	if err = createIndexes(ctx, tx); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration transaction: %w", err)
	}

	return nil
}

// createChunksTable creates the chunks table if it doesn't exist (Phase 1 compatible)
func createChunksTable(ctx context.Context, tx *sql.Tx) error {
	query := `
		CREATE TABLE IF NOT EXISTS chunks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source TEXT NOT NULL,
			text TEXT NOT NULL,
			embedding BLOB NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`
	_, err := tx.ExecContext(ctx, query)
	return err
}

// addChunksColumns adds the tags and summary columns to chunks table if they don't exist
func addChunksColumns(ctx context.Context, tx *sql.Tx) error {
	// Check if tags column exists
	var tagsExists bool
	err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*) > 0 
		FROM pragma_table_info('chunks') 
		WHERE name = 'tags'
	`).Scan(&tagsExists)
	if err != nil {
		return fmt.Errorf("failed to check tags column: %w", err)
	}

	// Add tags column if it doesn't exist
	if !tagsExists {
		_, err = tx.ExecContext(ctx, `ALTER TABLE chunks ADD COLUMN tags TEXT`)
		if err != nil {
			return fmt.Errorf("failed to add tags column: %w", err)
		}
	}

	// Check if summary column exists
	var summaryExists bool
	err = tx.QueryRowContext(ctx, `
		SELECT COUNT(*) > 0 
		FROM pragma_table_info('chunks') 
		WHERE name = 'summary'
	`).Scan(&summaryExists)
	if err != nil {
		return fmt.Errorf("failed to check summary column: %w", err)
	}

	// Add summary column if it doesn't exist
	if !summaryExists {
		_, err = tx.ExecContext(ctx, `ALTER TABLE chunks ADD COLUMN summary TEXT`)
		if err != nil {
			return fmt.Errorf("failed to add summary column: %w", err)
		}
	}

	return nil
}

// createChatMessagesTable creates the chat_messages table if it doesn't exist
func createChatMessagesTable(ctx context.Context, tx *sql.Tx) error {
	query := `
		CREATE TABLE IF NOT EXISTS chat_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL,
			role TEXT NOT NULL CHECK(role IN ('user', 'assistant')),
			content TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`
	_, err := tx.ExecContext(ctx, query)
	return err
}

// createAuditLogTable creates the audit_log table if it doesn't exist
func createAuditLogTable(ctx context.Context, tx *sql.Tx) error {
	query := `
		CREATE TABLE IF NOT EXISTS audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			operation_type TEXT NOT NULL,
			details TEXT,
			user_context TEXT
		)
	`
	_, err := tx.ExecContext(ctx, query)
	return err
}

// createWatchedFoldersTable creates the watched_folders table if it doesn't exist
func createWatchedFoldersTable(ctx context.Context, tx *sql.Tx) error {
	query := `
		CREATE TABLE IF NOT EXISTS watched_folders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			path TEXT NOT NULL UNIQUE,
			active BOOLEAN DEFAULT 1,
			last_scan TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`
	_, err := tx.ExecContext(ctx, query)
	return err
}

// createIndexes creates performance indexes if they don't exist
func createIndexes(ctx context.Context, tx *sql.Tx) error {
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_chunks_source ON chunks(source)`,
		`CREATE INDEX IF NOT EXISTS idx_chunks_created ON chunks(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_session ON chat_messages(session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_created ON chat_messages(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_log(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_type ON audit_log(operation_type)`,
	}

	for _, indexQuery := range indexes {
		if _, err := tx.ExecContext(ctx, indexQuery); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}
