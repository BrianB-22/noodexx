# Implementation Plan: Noodexx Phase 4 - Multi-User Foundation

## Overview

This implementation plan establishes multi-user support in Noodexx while maintaining backward compatibility with single-user deployments. The system introduces user identity, authentication, data ownership, and session isolation through a configurable user mode that defaults to single-user operation.

Key implementation areas:
- Database layer with user tables and ownership columns
- Authentication package with pluggable providers
- Middleware for request authentication and context injection
- User-scoped data access across all operations
- UI for login, registration, and user management
- Migration from Phase 3 with data preservation
- Property-based tests for 30 correctness properties

Implementation language: Go

## Tasks

- [ ] 1. Set up configuration and data models
  - [x] 1.1 Add user_mode and auth configuration fields to Config struct
    - Add UserMode string field with "single" or "multi" values
    - Add AuthConfig struct with provider, session_expiry_days, lockout_threshold, lockout_duration_minutes
    - Add environment variable support for NOODEXX_USER_MODE and NOODEXX_AUTH_PROVIDER
    - _Requirements: 1.1, 1.4, 1.5, 15.1, 15.2, 15.3, 15.4, 15.5, 15.6, 15.7_
  
  - [ ]* 1.2 Write property test for configuration loading
    - **Property 1: Configuration Loading Preserves User Mode**
    - **Validates: Requirements 1.1**
  
  - [ ]* 1.3 Write property test for auth provider validation
    - **Property 29: Configuration Validation for Auth Provider**
    - **Validates: Requirements 15.8**
  
  - [x] 1.4 Create Go data models for User, SessionToken, Skill
    - Define User struct with ID, Username, PasswordHash, Email, IsAdmin, MustChangePassword, CreatedAt, LastLogin
    - Define SessionToken struct with Token, UserID, CreatedAt, ExpiresAt
    - Define Skill struct with ID, UserID, Name, Path, Enabled, CreatedAt
    - _Requirements: 2.1, 2.9, 11.1_

- [ ] 2. Implement DataStore interface abstraction
  - [x] 2.1 Define DataStore interface in internal/store package
    - Define interface with all database operation methods
    - Include user management, session tokens, account lockout, user-scoped data access methods
    - Add NewDataStore factory function that accepts dbType and connectionString
    - _Requirements: 2.9, 2.12, 7.2, 7.5, 7.7, 7.8_

- [ ] 3. Create database schema and migration system
  - [x] 3.1 Create users table schema
    - Create table with id, username, password_hash, email, is_admin, must_change_password, created_at, last_login columns
    - Add unique constraints on username and email
    - _Requirements: 2.1, 2.2, 2.3_
  
  - [x] 3.2 Create session_tokens table schema
    - Create table with token, user_id, created_at, expires_at columns
    - Add foreign key constraint to users.id with CASCADE delete
    - _Requirements: 7.5_
  
  - [x] 3.3 Create failed_logins table schema
    - Create table with id, username, attempted_at columns
    - Used for account lockout tracking
    - _Requirements: 7.9, 7.10_
  
  - [x] 3.4 Create sessions metadata table schema
    - Create table with id, user_id, title, created_at, last_message_at columns
    - Add foreign key constraint to users.id with CASCADE delete
    - _Requirements: 5.1_
  
  - [x] 3.5 Create skills metadata table schema
    - Create table with id, user_id, name, path, enabled, created_at columns
    - Add foreign key constraint to users.id with CASCADE delete
    - _Requirements: 11.1_
  
  - [x] 3.6 Add user_id columns to existing tables
    - Add user_id to chunks, chat_messages, audit_log, watched_folders tables
    - Add visibility and shared_with columns to chunks table
    - Add username column to audit_log table
    - Add foreign key constraints to users.id
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5, 3.6, 3.8, 4.1, 4.3, 6.1, 6.2_
  
  - [x] 3.7 Create indexes for performance
    - Create indexes on all user_id columns
    - Create index on session_tokens.expires_at
    - Create index on failed_logins.username and attempted_at
    - Create index on chunks.visibility
    - _Requirements: 3.9_

  - [x] 3.8 Implement Phase 3 to Phase 4 migration logic
    - Create migration function that runs in a transaction
    - Create new tables (users, session_tokens, failed_logins, sessions, skills)
    - Create default users based on user_mode (local-default always, admin in multi-user mode)
    - Generate secure random password for admin using crypto/rand
    - Log admin password to console with clear formatting
    - Add columns to existing tables
    - Migrate existing data to local-default user
    - Create foreign keys and indexes
    - Verify migration success (record counts, foreign keys)
    - _Requirements: 2.4, 2.5, 2.6, 2.7, 2.8, 14.1, 14.2, 14.3, 14.4, 14.5, 14.6, 14.8, 14.9_
  
  - [ ]* 3.9 Write property test for migration data preservation
    - **Property 30: Migration Data Preservation**
    - **Validates: Requirements 14.5, 14.6**

- [ ] 4. Implement SQLiteStore with user management
  - [x] 4.1 Create SQLiteStore struct and constructor
    - Implement NewSQLiteStore with WAL mode and connection pooling
    - Configure connection pool: SetMaxOpenConns(25), SetMaxIdleConns(5), SetConnMaxLifetime(5min)
    - Add busy_timeout(5000) for write contention handling
    - Call runMigrations on initialization
    - _Requirements: 2.9_
  
  - [x] 4.2 Implement user management methods
    - Implement CreateUser with bcrypt password hashing
    - Implement GetUserByUsername
    - Implement GetUserByID
    - Implement ValidateCredentials with bcrypt comparison
    - Implement UpdatePassword with must_change_password reset
    - Implement UpdateLastLogin
    - Implement ListUsers
    - Implement DeleteUser
    - _Requirements: 2.9, 2.10, 2.11, 2.12, 2.13, 2.14_

  - [ ]* 4.3 Write property tests for user management
    - **Property 2: Username Uniqueness Enforcement**
    - **Property 3: Email Uniqueness Enforcement**
    - **Property 4: Password Hashing Round Trip**
    - **Property 5: Credential Validation Correctness**
    - **Property 6: Last Login Timestamp Update**
    - **Property 7: Password Update Clears Must-Change Flag**
    - **Validates: Requirements 2.2, 2.3, 2.10, 2.12, 2.13, 2.14, 7.3**
  
  - [x] 4.4 Implement session token management methods
    - Implement CreateSessionToken
    - Implement GetSessionToken
    - Implement DeleteSessionToken
    - Implement CleanupExpiredTokens
    - _Requirements: 7.5, 7.8_
  
  - [ ]* 4.5 Write property tests for session tokens
    - **Property 17: Session Token Storage and Retrieval**
    - **Property 18: Token Validation Correctness**
    - **Property 19: Logout Invalidates Token**
    - **Validates: Requirements 7.5, 7.7, 7.8**
  
  - [x] 4.6 Implement account lockout methods
    - Implement RecordFailedLogin
    - Implement ClearFailedLogins
    - Implement IsAccountLocked (check for 5 attempts in 15 minutes)
    - _Requirements: 7.9, 7.10_

- [ ] 5. Implement user-scoped data access methods
  - [x] 5.1 Update SaveChunk to accept user_id parameter
    - Add user_id to INSERT statement
    - Set visibility to "private" by default
    - _Requirements: 3.1, 4.1, 4.2_

  - [x] 5.2 Implement LibraryByUser with visibility filtering
    - Filter by user_id OR visibility="public" OR user_id in shared_with
    - Return only chunks visible to the specified user
    - _Requirements: 4.4_
  
  - [ ]* 5.3 Write property tests for document visibility
    - **Property 8: Document Visibility Defaults to Private**
    - **Property 9: Library Query Respects Visibility Rules**
    - **Property 10: RAG Search Respects Visibility Rules**
    - **Validates: Requirements 4.2, 4.4, 4.5**
  
  - [x] 5.4 Implement SearchByUser with visibility filtering
    - Filter vector search results by user_id and visibility rules
    - _Requirements: 4.5_
  
  - [x] 5.5 Implement DeleteChunksBySource with user_id parameter
    - Only delete chunks owned by the specified user
    - _Requirements: 3.1_
  
  - [x] 5.6 Implement session management methods
    - Implement SaveChatMessage with user_id parameter
    - Implement GetUserSessions filtering by user_id
    - Implement GetSessionOwner
    - Implement GetSessionMessages with ownership verification
    - _Requirements: 3.2, 5.1, 5.2, 5.3, 5.4, 5.5, 5.6_
  
  - [ ]* 5.7 Write property tests for session ownership
    - **Property 11: Session Ownership Association**
    - **Property 12: Session List Filtering**
    - **Property 13: Session Access Authorization**
    - **Validates: Requirements 5.2, 5.3, 5.4**
  
  - [x] 5.8 Implement skills management methods
    - Implement CreateSkill with user_id parameter
    - Implement GetUserSkills filtering by user_id
    - Implement UpdateSkillEnabled with ownership verification
    - Implement DeleteSkill with ownership verification
    - _Requirements: 11.1, 11.2, 11.7, 11.8_

  - [ ]* 5.9 Write property tests for skills user scoping
    - **Property 23: Skills User Scoping**
    - **Property 24: Skill Execution Authorization**
    - **Property 25: Skill Execution Context**
    - **Validates: Requirements 11.2, 11.3, 11.6**
  
  - [x] 5.10 Implement watched folders management methods
    - Implement AddWatchedFolder with user_id parameter
    - Implement GetWatchedFoldersByUser filtering by user_id
    - Implement RemoveWatchedFolder with ownership verification
    - _Requirements: 3.5, 12.1, 12.3_
  
  - [ ]* 5.11 Write property tests for watched folder ownership
    - **Property 26: Watched Folder Ownership**
    - **Property 27: Watched Folder Chunk Ownership**
    - **Property 28: Watched Folder Access Control**
    - **Validates: Requirements 12.1, 12.2, 12.5, 12.6**
  
  - [x] 5.12 Update audit logging methods
    - Update LogAudit to accept user_id and username parameters
    - Implement GetAuditLogByUser filtering by user_id
    - _Requirements: 3.4, 6.1, 6.2, 6.4_
  
  - [ ]* 5.13 Write property tests for audit log user context
    - **Property 14: Audit Log User Context**
    - **Property 15: Audit Log User Filtering**
    - **Validates: Requirements 6.1, 6.2, 6.4**

- [x] 6. Checkpoint - Verify database layer
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 7. Implement authentication package
  - [x] 7.1 Define Auth Provider interface
    - Define interface with Login, Logout, ValidateToken, RefreshToken methods
    - Create GetProvider factory function
    - _Requirements: 13.1, 13.6_
  
  - [x] 7.2 Implement password hashing utilities
    - Implement hashPassword using bcrypt.GenerateFromPassword
    - Implement checkPasswordHash using bcrypt.CompareHashAndPassword
    - _Requirements: 2.10_
  
  - [x] 7.3 Implement secure token generation
    - Implement generateSecureToken using crypto/rand with 32 bytes
    - Use base64.URLEncoding for token encoding
    - _Requirements: 7.4_
  
  - [ ]* 7.4 Write property test for token generation uniqueness
    - **Property 16: Session Token Generation Uniqueness**
    - **Validates: Requirements 7.4**
  
  - [x] 7.5 Implement Userpass Auth provider
    - Implement NewUserpassAuth constructor with config
    - Implement Login method with credential validation, account lockout check, token generation
    - Implement Logout method with token deletion
    - Implement ValidateToken method with expiration check
    - Implement RefreshToken method (stub for Phase 5)
    - Record failed login attempts and clear on success
    - Update last login timestamp on successful login
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5, 7.6, 7.7, 7.8, 7.9, 7.10_
  
  - [x] 7.6 Implement MFA and SSO auth stubs
    - Create MFAAuth struct with Login returning "not implemented" error
    - Create SSOAuth struct with Login returning "not implemented" error
    - _Requirements: 13.2, 13.3, 13.5_

- [ ] 8. Implement authentication middleware
  - [x] 8.1 Create AuthMiddleware for HTTP request authentication
    - Define contextKey type and UserIDKey constant
    - Implement middleware that checks user_mode
    - In single-user mode: inject local-default user_id automatically
    - In multi-user mode: extract token, validate, inject user_id
    - Skip authentication for public endpoints (/login, /register, /static/*)
    - Return 401 Unauthorized for invalid/missing tokens in multi-user mode
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5, 8.7, 8.8, 1.2, 1.3_
  
  - [x] 8.2 Implement token extraction from headers and cookies
    - Extract from Authorization header with "Bearer " prefix
    - Fall back to session_token cookie
    - _Requirements: 8.2_
  
  - [ ]* 8.3 Write property test for token extraction
    - **Property 20: Token Extraction from Multiple Sources**
    - **Validates: Requirements 8.2**
  
  - [x] 8.3 Implement GetUserID context helper function
    - Extract user_id from request context
    - Return error if not found
    - _Requirements: 8.6_
  
  - [ ]* 8.4 Write property test for context injection
    - **Property 21: Context Injection in Multi-User Mode**
    - **Validates: Requirements 8.4**

- [ ] 9. Implement API endpoints for authentication
  - [x] 9.1 Implement POST /api/login endpoint
    - Accept username and password in request body
    - Call authProvider.Login
    - Check must_change_password flag
    - Set session_token cookie (HttpOnly, Secure in production)
    - Return user info and redirect URL
    - _Requirements: 7.2, 9.5, 9.6_

  - [x] 9.2 Implement POST /api/logout endpoint
    - Extract token from request
    - Call authProvider.Logout
    - Clear session_token cookie
    - Return success response
    - _Requirements: 7.8_
  
  - [x] 9.3 Implement POST /api/register endpoint
    - Accept username, email, password, confirm_password in request body
    - Validate password confirmation
    - Call store.CreateUser with is_admin=false, must_change_password=false
    - Return success or validation error
    - _Requirements: 9.8, 9.9, 9.10, 9.11_
  
  - [x] 9.4 Implement POST /api/change-password endpoint
    - Extract user_id from context
    - Accept new_password and confirm_password in request body
    - Validate password confirmation
    - Call store.UpdatePassword
    - Return success response
    - _Requirements: 9.12, 9.13_

- [ ] 10. Implement admin user management endpoints
  - [x] 10.1 Implement GET /api/users endpoint (admin only)
    - Verify current user is admin
    - Call store.ListUsers
    - Return user list
    - _Requirements: 10.2_
  
  - [x] 10.2 Implement POST /api/users endpoint (admin only)
    - Verify current user is admin
    - Accept username, email, password, is_admin in request body
    - Call store.CreateUser
    - Return created user or validation error
    - _Requirements: 10.3, 10.6_

  - [x] 10.3 Implement DELETE /api/users/:id endpoint (admin only)
    - Verify current user is admin
    - Prompt for confirmation (client-side)
    - Call store.DeleteUser
    - Return success response
    - _Requirements: 10.5, 10.7, 10.8_
  
  - [x] 10.4 Implement POST /api/users/:id/reset-password endpoint (admin only)
    - Verify current user is admin
    - Generate random password using crypto/rand
    - Call store.UpdatePassword with must_change_password=true
    - Return temporary password
    - _Requirements: 10.9_
  
  - [ ]* 10.5 Write property test for username uniqueness in user creation
    - **Property 22: Username Uniqueness in User Creation**
    - **Validates: Requirements 10.6**

- [ ] 11. Update existing API handlers for user scoping
  - [x] 11.1 Update POST /api/ingest handler
    - Extract user_id from context using GetUserID
    - Pass user_id to store.SaveChunk
    - _Requirements: 8.6, 3.1_
  
  - [x] 11.2 Update GET /api/library handler
    - Extract user_id from context
    - Call store.LibraryByUser with user_id
    - _Requirements: 8.6, 4.4_
  
  - [x] 11.3 Update POST /api/search handler
    - Extract user_id from context
    - Call store.SearchByUser with user_id
    - _Requirements: 8.6, 4.5_

  - [x] 11.4 Update GET /api/sessions handler
    - Extract user_id from context
    - Call store.GetUserSessions with user_id
    - _Requirements: 8.6, 5.3_
  
  - [x] 11.5 Update POST /api/chat handler
    - Extract user_id from context
    - Verify session ownership before loading messages
    - Pass user_id to store.SaveChatMessage
    - _Requirements: 8.6, 5.4_
  
  - [x] 11.6 Update GET /api/skills handler
    - Extract user_id from context
    - Call store.GetUserSkills with user_id
    - _Requirements: 8.6, 11.2_
  
  - [x] 11.7 Update GET /api/watched-folders handler
    - Extract user_id from context
    - Call store.GetWatchedFoldersByUser with user_id
    - _Requirements: 8.6, 12.3_

- [x] 12. Checkpoint - Verify API layer
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 13. Update folder watcher for user scoping
  - [x] 13.1 Update folder watcher to process all users' folders
    - Load all watched folders from database
    - Process files from each folder with the folder's user_id
    - Pass user_id to store.SaveChunk when ingesting
    - _Requirements: 12.4, 12.5_

- [ ] 14. Update skills system for user scoping
  - [x] 14.1 Update skill loading to filter by user context
    - Load skills for the current user only
    - Pass user_id when creating or updating skills
    - _Requirements: 11.2, 11.7_

  - [x] 14.2 Update skill execution to verify ownership
    - Check skill's user_id matches current user before execution
    - Execute skill in the context of the owning user
    - _Requirements: 11.3, 11.6_

- [x] 15. Implement login page UI
  - [x] 15.1 Create login page HTML template
    - Add username and password input fields
    - Add "Login" button
    - Add error message display area
    - Add "Register" link
    - _Requirements: 9.1, 9.3, 9.4_
  
  - [x] 15.2 Implement login page JavaScript
    - Handle form submission
    - Call POST /api/login
    - Handle must_change_password redirect
    - Display error messages
    - Set session cookie
    - Redirect to dashboard on success
    - _Requirements: 9.5, 9.6, 9.7_
  
  - [x] 15.3 Implement routing logic for user_mode
    - In multi-user mode: show login page at root URL
    - In single-user mode: skip login page entirely, go directly to dashboard
    - _Requirements: 9.1, 9.2_

- [x] 16. Implement registration page UI
  - [x] 16.1 Create registration page HTML template
    - Add username, email, password, confirm_password input fields
    - Add "Create Account" button
    - Add "Back to Login" link
    - _Requirements: 9.8, 9.9_

  - [x] 16.2 Implement registration page JavaScript
    - Validate password confirmation matches
    - Validate username format (3-32 chars, alphanumeric)
    - Validate email format
    - Call POST /api/register
    - Redirect to login on success
    - Display validation errors
    - _Requirements: 9.10, 9.11_

- [x] 17. Implement password change page UI
  - [x] 17.1 Create password change page HTML template
    - Add new_password and confirm_password input fields
    - Add "Change Password" button
    - Display message about required password change
    - _Requirements: 9.12_
  
  - [x] 17.2 Implement password change page JavaScript
    - Validate password confirmation matches
    - Call POST /api/change-password
    - Redirect to dashboard on success
    - Display error messages
    - _Requirements: 9.13_
  
  - [x] 17.3 Implement middleware redirect for must_change_password
    - Check user's must_change_password flag
    - Redirect to /change-password if true
    - Allow access to dashboard only after password change
    - _Requirements: 9.6_

- [x] 18. Implement user management UI
  - [x] 18.1 Add user management section to settings page
    - Display section only if current user is admin
    - Add user list table with columns: username, email, is_admin, created_at, last_login
    - Add "Create User" button
    - Add "Edit", "Delete", "Reset Password" buttons per row
    - _Requirements: 10.1, 10.2, 10.4, 10.5_

  - [x] 18.2 Implement create/edit user modal
    - Add form fields for username, email, password, is_admin checkbox
    - Call POST /api/users for creation
    - Display validation errors
    - Refresh user list on success
    - _Requirements: 10.3, 10.6, 10.10_
  
  - [x] 18.3 Implement delete user confirmation
    - Show confirmation dialog before deletion
    - Call DELETE /api/users/:id
    - Refresh user list on success
    - _Requirements: 10.7, 10.8_
  
  - [x] 18.4 Implement reset password functionality
    - Call POST /api/users/:id/reset-password
    - Display temporary password to admin
    - Show message that user must change password on next login
    - _Requirements: 10.9_

- [x] 19. Add user profile section to settings
  - [x] 19.1 Add user profile display
    - Show current username and email
    - Add "Change Password" button
    - Add "Logout" button
    - _Requirements: 9.12_

- [x] 20. Update library page with visibility indicators
  - [x] 20.1 Add visibility icons to document cards
    - Lock icon for private documents
    - People icon with count for shared documents
    - Globe icon for public documents
    - _Requirements: 4.6_
  
  - [x] 20.2 Add share button to document cards (UI only)
    - Display "Share" button on each card
    - Disable button with tooltip: "Sharing coming in Phase 5"
    - _Requirements: 4.7_

- [x] 21. Add navigation bar user menu
  - [x] 21.1 Add user menu in multi-user mode
    - Display current username in navigation bar
    - Add dropdown menu with "Settings", "User Management" (admin only), "Logout"
    - Hide user menu in single-user mode
    - _Requirements: 9.1, 9.2_

- [x] 22. Checkpoint - Verify UI layer
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 23. Write integration tests
  - [ ]* 23.1 Write end-to-end login flow test
    - Test successful login with valid credentials
    - Test failed login with invalid credentials
    - Test account lockout after 5 failed attempts
    - Test must_change_password redirect
    - _Requirements: 7.2, 7.9, 7.10, 9.5, 9.6_
  
  - [ ]* 23.2 Write end-to-end registration flow test
    - Test successful registration
    - Test duplicate username rejection
    - Test password confirmation validation
    - _Requirements: 9.8, 9.9, 9.10, 9.11_
  
  - [ ]* 23.3 Write data isolation integration test
    - Create two users
    - Ingest documents for each user
    - Verify each user sees only their own documents
    - Verify search results are user-scoped
    - _Requirements: 4.4, 4.5_
  
  - [ ]* 23.4 Write admin user management integration test
    - Test admin can create users
    - Test admin can delete users
    - Test admin can reset passwords
    - Test non-admin cannot access user management
    - _Requirements: 10.3, 10.5, 10.6, 10.9_

  - [ ]* 23.5 Write backward compatibility integration test
    - Start with Phase 3 database
    - Run migration
    - Verify all data migrated to local-default user
    - Verify single-user mode behavior matches Phase 3
    - _Requirements: 14.1, 14.2, 14.3, 14.4, 14.7_
  
  - [ ]* 23.6 Write session ownership integration test
    - Create user and start chat session
    - Verify session is owned by user
    - Verify other users cannot access session
    - _Requirements: 5.2, 5.3, 5.4_
  
  - [ ]* 23.7 Write skills user scoping integration test
    - Create user and add skill
    - Verify skill is owned by user
    - Verify other users cannot see or execute skill
    - _Requirements: 11.2, 11.3_
  
  - [ ]* 23.8 Write watched folder integration test
    - Create user and add watched folder
    - Ingest file from watched folder
    - Verify chunks are owned by user
    - Verify other users cannot see chunks
    - _Requirements: 12.1, 12.2, 12.5_

- [ ] 24. Write unit tests for edge cases
  - [ ]* 24.1 Write unit tests for authentication edge cases
    - Test expired session token
    - Test invalid session token format
    - Test missing Authorization header and cookie
    - Test account lockout expiration
    - _Requirements: 7.7, 7.8, 8.2, 8.3_

  - [ ]* 24.2 Write unit tests for configuration edge cases
    - Test missing user_mode defaults to "single"
    - Test invalid auth provider value
    - Test environment variable overrides
    - _Requirements: 1.4, 1.5, 15.7, 15.8_
  
  - [ ]* 24.3 Write unit tests for migration edge cases
    - Test migration with empty database
    - Test migration rollback on error
    - Test migration with existing users table
    - _Requirements: 14.8, 14.9_
  
  - [ ]* 24.4 Write unit tests for visibility filtering edge cases
    - Test visibility with null shared_with
    - Test visibility with empty shared_with
    - Test visibility with multiple users in shared_with
    - _Requirements: 4.3, 4.4, 4.5_
  
  - [ ]* 24.5 Write unit tests for MFA and SSO stubs
    - Test MFA auth returns "not implemented" error
    - Test SSO auth returns "not implemented" error
    - _Requirements: 13.2, 13.3, 13.5_

- [ ] 25. Performance testing and optimization
  - [ ]* 25.1 Test concurrent user access
    - Simulate 10-20 concurrent users
    - Verify no race conditions
    - Verify connection pool handles load
    - _Requirements: Database concurrency design_
  
  - [ ]* 25.2 Test migration performance
    - Create database with 100,000 chunks
    - Run migration and measure time
    - Verify migration completes in < 5 minutes
    - _Requirements: 14.5_

  - [ ]* 25.3 Test user-scoped query performance
    - Create 10 users with 1000 documents each
    - Measure library query time
    - Verify queries complete in < 100ms
    - _Requirements: Performance targets in design_

- [ ] 26. Security audit and hardening
  - [ ]* 26.1 Audit password security
    - Verify bcrypt is used for all passwords
    - Verify passwords are never logged
    - Verify constant-time comparison
    - _Requirements: Password security in design_
  
  - [ ]* 26.2 Audit session security
    - Verify crypto/rand is used for tokens
    - Verify tokens have sufficient entropy (256 bits)
    - Verify HttpOnly and Secure cookie flags
    - _Requirements: Session security in design_
  
  - [ ]* 26.3 Audit SQL injection prevention
    - Verify all queries use parameterized statements
    - Verify no string concatenation in SQL
    - _Requirements: SQL injection prevention in design_
  
  - [ ]* 26.4 Audit authorization checks
    - Verify all endpoints extract user_id from context
    - Verify ownership checks before data access
    - Verify admin checks for user management
    - _Requirements: Authorization in design_

- [ ] 27. Documentation updates
  - [ ]* 27.1 Update README with multi-user setup instructions
    - Document user_mode configuration
    - Document default admin account creation
    - Document migration from Phase 3
    - _Requirements: Deployment considerations in design_

  - [ ]* 27.2 Create migration guide
    - Document backup procedure
    - Document migration steps
    - Document rollback procedure
    - Document verification steps
    - _Requirements: Migration from Phase 3 in design_
  
  - [ ]* 27.3 Update API documentation
    - Document new authentication endpoints
    - Document user management endpoints
    - Document user_id context requirement
    - _Requirements: API endpoints in design_

- [ ] 28. Final integration and wiring
  - [x] 28.1 Wire authentication package into main application
    - Initialize auth provider based on config
    - Pass auth provider to API server
    - _Requirements: 13.4, 13.6_
  
  - [x] 28.2 Wire middleware into HTTP router
    - Apply AuthMiddleware to all routes except public endpoints
    - Configure public endpoint list
    - _Requirements: 8.1, 8.7, 8.8_
  
  - [x] 28.3 Update main.go initialization
    - Load config with user_mode and auth settings
    - Initialize DataStore with migration
    - Initialize auth provider
    - Create API server with dependencies
    - Start background job for expired token cleanup
    - _Requirements: 1.1, 2.4, 7.5_
  
  - [x] 28.4 Add background job for token cleanup
    - Run CleanupExpiredTokens every hour
    - Run in separate goroutine
    - _Requirements: Session expiration in design_

- [x] 29. Final checkpoint - End-to-end verification
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation at key milestones
- Property tests validate universal correctness properties from the design document
- Unit tests validate specific examples and edge cases
- Integration tests verify end-to-end workflows
- The implementation uses Go with bcrypt for password hashing and crypto/rand for secure token generation
- SQLite is configured with WAL mode and connection pooling for concurrent multi-user access
- All data access is user-scoped with visibility filtering
- Single-user mode maintains complete backward compatibility with Phase 3
- Multi-user mode requires authentication and enforces data isolation
