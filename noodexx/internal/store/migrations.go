package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"

	"golang.org/x/crypto/bcrypt"
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
	if err = createUsersTable(ctx, tx); err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	if err = createSessionTokensTable(ctx, tx); err != nil {
		return fmt.Errorf("failed to create session_tokens table: %w", err)
	}

	if err = createFailedLoginsTable(ctx, tx); err != nil {
		return fmt.Errorf("failed to create failed_logins table: %w", err)
	}

	if err = createSessionsTable(ctx, tx); err != nil {
		return fmt.Errorf("failed to create sessions table: %w", err)
	}

	if err = createSkillsTable(ctx, tx); err != nil {
		return fmt.Errorf("failed to create skills table: %w", err)
	}

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

	// Add user_id columns to existing tables (Phase 4)
	if err = addUserIDToChunks(ctx, tx); err != nil {
		return fmt.Errorf("failed to add user_id to chunks: %w", err)
	}

	if err = addUserIDToChatMessages(ctx, tx); err != nil {
		return fmt.Errorf("failed to add user_id to chat_messages: %w", err)
	}

	if err = addUserIDToAuditLog(ctx, tx); err != nil {
		return fmt.Errorf("failed to add user_id to audit_log: %w", err)
	}

	if err = addUserIDToWatchedFolders(ctx, tx); err != nil {
		return fmt.Errorf("failed to add user_id to watched_folders: %w", err)
	}

	// Run Phase 3 to Phase 4 data migration
	// This must happen after tables and columns are created but before indexes
	if err = migratePhase3ToPhase4(ctx, tx, s.userMode); err != nil {
		return fmt.Errorf("failed to migrate Phase 3 to Phase 4: %w", err)
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
		`CREATE INDEX IF NOT EXISTS idx_chunks_user ON chunks(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_chunks_visibility ON chunks(visibility)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_session ON chat_messages(session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_created ON chat_messages(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_user ON chat_messages(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_log(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_type ON audit_log(operation_type)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_log(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_skills_user ON skills(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_watched_folders_user ON watched_folders(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_session_tokens_user ON session_tokens(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_session_tokens_expires ON session_tokens(expires_at)`,
		`CREATE INDEX IF NOT EXISTS idx_failed_logins_username ON failed_logins(username)`,
		`CREATE INDEX IF NOT EXISTS idx_failed_logins_attempted ON failed_logins(attempted_at)`,
	}

	for _, indexQuery := range indexes {
		if _, err := tx.ExecContext(ctx, indexQuery); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// createUsersTable creates the users table if it doesn't exist (Phase 4)
func createUsersTable(ctx context.Context, tx *sql.Tx) error {
	query := `
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			email TEXT UNIQUE,
			is_admin BOOLEAN DEFAULT 0,
			must_change_password BOOLEAN DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_login TIMESTAMP
		)
	`
	_, err := tx.ExecContext(ctx, query)
	return err
}

// createSessionTokensTable creates the session_tokens table if it doesn't exist (Phase 4)
func createSessionTokensTable(ctx context.Context, tx *sql.Tx) error {
	query := `
		CREATE TABLE IF NOT EXISTS session_tokens (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`
	_, err := tx.ExecContext(ctx, query)
	return err
}

// createFailedLoginsTable creates the failed_logins table if it doesn't exist (Phase 4)
// This table tracks failed login attempts for account lockout mechanism
func createFailedLoginsTable(ctx context.Context, tx *sql.Tx) error {
	query := `
		CREATE TABLE IF NOT EXISTS failed_logins (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL,
			attempted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`
	_, err := tx.ExecContext(ctx, query)
	return err
}

// createSessionsTable creates the sessions metadata table if it doesn't exist (Phase 4)
// This table stores metadata about chat sessions (separate from chat_messages which stores the actual messages)
func createSessionsTable(ctx context.Context, tx *sql.Tx) error {
	query := `
		CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			title TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_message_at TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`
	_, err := tx.ExecContext(ctx, query)
	return err
}

// createSkillsTable creates the skills metadata table if it doesn't exist (Phase 4)
// This table stores metadata about user-owned skills/plugins
func createSkillsTable(ctx context.Context, tx *sql.Tx) error {
	query := `
		CREATE TABLE IF NOT EXISTS skills (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			path TEXT NOT NULL,
			enabled BOOLEAN DEFAULT 1,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`
	_, err := tx.ExecContext(ctx, query)
	return err
}

// addUserIDToChunks adds user_id, visibility, and shared_with columns to chunks table (Phase 4)
func addUserIDToChunks(ctx context.Context, tx *sql.Tx) error {
	// Check if user_id column exists
	var userIDExists bool
	err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*) > 0 
		FROM pragma_table_info('chunks') 
		WHERE name = 'user_id'
	`).Scan(&userIDExists)
	if err != nil {
		return fmt.Errorf("failed to check user_id column: %w", err)
	}

	// Add user_id column if it doesn't exist
	if !userIDExists {
		_, err = tx.ExecContext(ctx, `ALTER TABLE chunks ADD COLUMN user_id INTEGER REFERENCES users(id) ON DELETE CASCADE`)
		if err != nil {
			return fmt.Errorf("failed to add user_id column: %w", err)
		}
	}

	// Check if visibility column exists
	var visibilityExists bool
	err = tx.QueryRowContext(ctx, `
		SELECT COUNT(*) > 0 
		FROM pragma_table_info('chunks') 
		WHERE name = 'visibility'
	`).Scan(&visibilityExists)
	if err != nil {
		return fmt.Errorf("failed to check visibility column: %w", err)
	}

	// Add visibility column if it doesn't exist
	if !visibilityExists {
		_, err = tx.ExecContext(ctx, `ALTER TABLE chunks ADD COLUMN visibility TEXT DEFAULT 'private' CHECK(visibility IN ('private', 'shared', 'public'))`)
		if err != nil {
			return fmt.Errorf("failed to add visibility column: %w", err)
		}
	}

	// Check if shared_with column exists
	var sharedWithExists bool
	err = tx.QueryRowContext(ctx, `
		SELECT COUNT(*) > 0 
		FROM pragma_table_info('chunks') 
		WHERE name = 'shared_with'
	`).Scan(&sharedWithExists)
	if err != nil {
		return fmt.Errorf("failed to check shared_with column: %w", err)
	}

	// Add shared_with column if it doesn't exist
	if !sharedWithExists {
		_, err = tx.ExecContext(ctx, `ALTER TABLE chunks ADD COLUMN shared_with TEXT`)
		if err != nil {
			return fmt.Errorf("failed to add shared_with column: %w", err)
		}
	}

	return nil
}

// addUserIDToChatMessages adds user_id column to chat_messages table (Phase 4)
func addUserIDToChatMessages(ctx context.Context, tx *sql.Tx) error {
	// Check if user_id column exists
	var userIDExists bool
	err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*) > 0 
		FROM pragma_table_info('chat_messages') 
		WHERE name = 'user_id'
	`).Scan(&userIDExists)
	if err != nil {
		return fmt.Errorf("failed to check user_id column: %w", err)
	}

	// Add user_id column if it doesn't exist
	if !userIDExists {
		_, err = tx.ExecContext(ctx, `ALTER TABLE chat_messages ADD COLUMN user_id INTEGER REFERENCES users(id) ON DELETE CASCADE`)
		if err != nil {
			return fmt.Errorf("failed to add user_id column: %w", err)
		}
	}

	// Check if provider_mode column exists
	var providerModeExists bool
	err = tx.QueryRowContext(ctx, `
		SELECT COUNT(*) > 0 
		FROM pragma_table_info('chat_messages') 
		WHERE name = 'provider_mode'
	`).Scan(&providerModeExists)
	if err != nil {
		return fmt.Errorf("failed to check provider_mode column: %w", err)
	}

	// Add provider_mode column if it doesn't exist
	if !providerModeExists {
		_, err = tx.ExecContext(ctx, `ALTER TABLE chat_messages ADD COLUMN provider_mode TEXT DEFAULT 'local'`)
		if err != nil {
			return fmt.Errorf("failed to add provider_mode column: %w", err)
		}
	}

	return nil
}

// addUserIDToAuditLog adds user_id and username columns to audit_log table (Phase 4)
func addUserIDToAuditLog(ctx context.Context, tx *sql.Tx) error {
	// Check if user_id column exists
	var userIDExists bool
	err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*) > 0 
		FROM pragma_table_info('audit_log') 
		WHERE name = 'user_id'
	`).Scan(&userIDExists)
	if err != nil {
		return fmt.Errorf("failed to check user_id column: %w", err)
	}

	// Add user_id column if it doesn't exist
	if !userIDExists {
		_, err = tx.ExecContext(ctx, `ALTER TABLE audit_log ADD COLUMN user_id INTEGER REFERENCES users(id) ON DELETE SET NULL`)
		if err != nil {
			return fmt.Errorf("failed to add user_id column: %w", err)
		}
	}

	// Check if username column exists
	var usernameExists bool
	err = tx.QueryRowContext(ctx, `
		SELECT COUNT(*) > 0 
		FROM pragma_table_info('audit_log') 
		WHERE name = 'username'
	`).Scan(&usernameExists)
	if err != nil {
		return fmt.Errorf("failed to check username column: %w", err)
	}

	// Add username column if it doesn't exist
	if !usernameExists {
		_, err = tx.ExecContext(ctx, `ALTER TABLE audit_log ADD COLUMN username TEXT`)
		if err != nil {
			return fmt.Errorf("failed to add username column: %w", err)
		}
	}

	return nil
}

// addUserIDToWatchedFolders adds user_id column to watched_folders table (Phase 4)
func addUserIDToWatchedFolders(ctx context.Context, tx *sql.Tx) error {
	// Check if user_id column exists
	var userIDExists bool
	err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*) > 0 
		FROM pragma_table_info('watched_folders') 
		WHERE name = 'user_id'
	`).Scan(&userIDExists)
	if err != nil {
		return fmt.Errorf("failed to check user_id column: %w", err)
	}

	// Add user_id column if it doesn't exist
	if !userIDExists {
		_, err = tx.ExecContext(ctx, `ALTER TABLE watched_folders ADD COLUMN user_id INTEGER REFERENCES users(id) ON DELETE CASCADE`)
		if err != nil {
			return fmt.Errorf("failed to add user_id column: %w", err)
		}
	}

	return nil
}

// migratePhase3ToPhase4 creates default users for Phase 4
// Note: Existing data is dropped - no migration needed per user request
func migratePhase3ToPhase4(ctx context.Context, tx *sql.Tx, userMode string) error {
	// Check if users already exist (migration already ran)
	var userCount int
	err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&userCount)
	if err != nil {
		return fmt.Errorf("failed to check existing users: %w", err)
	}

	// If users already exist, skip migration
	if userCount > 0 {
		return nil
	}

	// Create default users
	_, err = createDefaultUsers(ctx, tx, userMode)
	if err != nil {
		return fmt.Errorf("failed to create default users: %w", err)
	}

	return nil
}

// createDefaultUsers creates the default users based on user_mode
// Returns the local-default user ID
func createDefaultUsers(ctx context.Context, tx *sql.Tx, userMode string) (int64, error) {
	// Always create local-default user
	localDefaultPassword, err := generateSecurePassword(16)
	if err != nil {
		return 0, fmt.Errorf("failed to generate local-default password: %w", err)
	}

	localDefaultHash, err := bcrypt.GenerateFromPassword([]byte(localDefaultPassword), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("failed to hash local-default password: %w", err)
	}

	result, err := tx.ExecContext(ctx, `
		INSERT INTO users (username, password_hash, email, is_admin, must_change_password)
		VALUES (?, ?, ?, ?, ?)
	`, "local-default", string(localDefaultHash), "", true, false)
	if err != nil {
		return 0, fmt.Errorf("failed to create local-default user: %w", err)
	}

	localDefaultID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get local-default user ID: %w", err)
	}

	// In multi-user mode, also create admin user
	if userMode == "multi" {
		adminPassword, err := generateSecurePassword(16)
		if err != nil {
			return 0, fmt.Errorf("failed to generate admin password: %w", err)
		}

		adminHash, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
		if err != nil {
			return 0, fmt.Errorf("failed to hash admin password: %w", err)
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO users (username, password_hash, email, is_admin, must_change_password)
			VALUES (?, ?, NULL, ?, ?)
		`, "admin", string(adminHash), true, true)
		if err != nil {
			return 0, fmt.Errorf("failed to create admin user: %w", err)
		}

		// Log admin password to console with clear formatting
		fmt.Printf("\n")
		fmt.Printf("===========================================\n")
		fmt.Printf("DEFAULT ADMIN ACCOUNT CREATED\n")
		fmt.Printf("===========================================\n")
		fmt.Printf("Username: admin\n")
		fmt.Printf("Temporary Password: %s\n", adminPassword)
		fmt.Printf("===========================================\n")
		fmt.Printf("You MUST change this password on first login\n")
		fmt.Printf("===========================================\n")
		fmt.Printf("\n")
	}

	return localDefaultID, nil
}

// generateSecurePassword generates a cryptographically secure random password
func generateSecurePassword(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	password := make([]byte, length)

	for i := range password {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", fmt.Errorf("failed to generate random number: %w", err)
		}
		password[i] = charset[num.Int64()]
	}

	return string(password), nil
}

// migrateExistingData migrates all existing Phase 3 data to the local-default user
func migrateExistingData(ctx context.Context, tx *sql.Tx, localDefaultID int64) error {
	// Migrate chunks to local-default user
	_, err := tx.ExecContext(ctx, `
		UPDATE chunks 
		SET user_id = ?, visibility = 'private' 
		WHERE user_id IS NULL
	`, localDefaultID)
	if err != nil {
		return fmt.Errorf("failed to migrate chunks: %w", err)
	}

	// Migrate chat_messages to local-default user
	_, err = tx.ExecContext(ctx, `
		UPDATE chat_messages 
		SET user_id = ? 
		WHERE user_id IS NULL
	`, localDefaultID)
	if err != nil {
		return fmt.Errorf("failed to migrate chat_messages: %w", err)
	}

	// Migrate audit_log to local-default user
	_, err = tx.ExecContext(ctx, `
		UPDATE audit_log 
		SET user_id = ?, username = 'local-default' 
		WHERE user_id IS NULL
	`, localDefaultID)
	if err != nil {
		return fmt.Errorf("failed to migrate audit_log: %w", err)
	}

	// Migrate watched_folders to local-default user
	_, err = tx.ExecContext(ctx, `
		UPDATE watched_folders 
		SET user_id = ? 
		WHERE user_id IS NULL
	`, localDefaultID)
	if err != nil {
		return fmt.Errorf("failed to migrate watched_folders: %w", err)
	}

	return nil
}

// verifyMigration verifies that the migration was successful
func verifyMigration(ctx context.Context, tx *sql.Tx, localDefaultID int64, expectedChunks, expectedMessages, expectedAudit, expectedFolders int) error {
	// Verify chunks count
	var actualChunks int
	err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM chunks WHERE user_id = ?`, localDefaultID).Scan(&actualChunks)
	if err != nil {
		return fmt.Errorf("failed to verify chunks count: %w", err)
	}
	if actualChunks != expectedChunks {
		return fmt.Errorf("chunks count mismatch: expected %d, got %d", expectedChunks, actualChunks)
	}

	// Verify chat_messages count
	var actualMessages int
	err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM chat_messages WHERE user_id = ?`, localDefaultID).Scan(&actualMessages)
	if err != nil {
		return fmt.Errorf("failed to verify chat_messages count: %w", err)
	}
	if actualMessages != expectedMessages {
		return fmt.Errorf("chat_messages count mismatch: expected %d, got %d", expectedMessages, actualMessages)
	}

	// Verify audit_log count
	var actualAudit int
	err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_log WHERE user_id = ?`, localDefaultID).Scan(&actualAudit)
	if err != nil {
		return fmt.Errorf("failed to verify audit_log count: %w", err)
	}
	if actualAudit != expectedAudit {
		return fmt.Errorf("audit_log count mismatch: expected %d, got %d", expectedAudit, actualAudit)
	}

	// Verify watched_folders count
	var actualFolders int
	err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM watched_folders WHERE user_id = ?`, localDefaultID).Scan(&actualFolders)
	if err != nil {
		return fmt.Errorf("failed to verify watched_folders count: %w", err)
	}
	if actualFolders != expectedFolders {
		return fmt.Errorf("watched_folders count mismatch: expected %d, got %d", expectedFolders, actualFolders)
	}

	// Verify no NULL user_id values remain in chunks
	var nullChunks int
	err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM chunks WHERE user_id IS NULL`).Scan(&nullChunks)
	if err != nil {
		return fmt.Errorf("failed to check NULL user_id in chunks: %w", err)
	}
	if nullChunks > 0 {
		return fmt.Errorf("found %d chunks with NULL user_id", nullChunks)
	}

	// Verify no NULL user_id values remain in chat_messages
	var nullMessages int
	err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM chat_messages WHERE user_id IS NULL`).Scan(&nullMessages)
	if err != nil {
		return fmt.Errorf("failed to check NULL user_id in chat_messages: %w", err)
	}
	if nullMessages > 0 {
		return fmt.Errorf("found %d chat_messages with NULL user_id", nullMessages)
	}

	// Verify no NULL user_id values remain in watched_folders
	var nullFolders int
	err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM watched_folders WHERE user_id IS NULL`).Scan(&nullFolders)
	if err != nil {
		return fmt.Errorf("failed to check NULL user_id in watched_folders: %w", err)
	}
	if nullFolders > 0 {
		return fmt.Errorf("found %d watched_folders with NULL user_id", nullFolders)
	}

	return nil
}
