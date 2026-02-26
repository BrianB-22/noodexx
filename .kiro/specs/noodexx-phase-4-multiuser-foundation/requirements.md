# Requirements Document

## Introduction

This document specifies requirements for Noodexx Phase 4: Multi-User Support Foundation. This phase establishes the foundational architecture for multi-user support while maintaining backward compatibility with single-user deployments. The system will introduce user identity, data ownership, session isolation, and a configurable authentication framework. By default, the system operates in single-user mode with a local-default user, but can be configured for multi-user mode with username/password authentication. This phase creates the skeleton for full multi-user support while preserving all existing Phase 3 functionality.

## Glossary

- **User**: An authenticated identity with ownership of documents, sessions, and skills
- **User_Mode**: System configuration determining single-user or multi-user operation
- **Local_Default_User**: The automatic user identity used in single-user mode (username: "local-default")
- **User_Table**: Database table storing user accounts with credentials and metadata
- **User_ID**: Unique identifier for a user, used as foreign key in data tables
- **Session_Owner**: The user who created a chat session
- **Document_Owner**: The user who ingested a document
- **Skill_Owner**: The user who created or enabled a skill
- **Authentication_Middleware**: HTTP middleware that validates user identity before processing requests
- **Auth_Provider**: Abstraction for authentication methods (userpass, MFA, SSO)
- **Userpass_Auth**: Username and password authentication provider
- **MFA_Auth**: Multi-factor authentication provider (stub for Phase 5)
- **SSO_Auth**: Single sign-on authentication provider (stub for Phase 5)
- **Admin_User**: A user with elevated privileges for user management operations
- **User_Management_UI**: Administrative interface for creating, modifying, and deleting users
- **Privacy_Mode**: Existing system-wide setting that restricts operations to local-only
- **Audit_Log**: Existing record of system operations, now enhanced with user context
- **Watched_Folder**: Existing filesystem monitoring feature, now scoped per user
- **Skill**: Existing plugin system, now scoped per user
- **Data_Isolation**: Enforcement that users can only access their own data
- **Shared_Document**: A document explicitly shared with other users (Phase 5 feature)
- **Visibility_Scope**: Access control setting for documents (private, shared, public)

## Requirements

### Requirement 1: User Mode Configuration

**User Story:** As a system administrator, I want to configure whether Noodexx runs in single-user or multi-user mode, so that I can deploy it appropriately for different use cases.

#### Acceptance Criteria

1. THE Config_Manager SHALL load a user_mode field from the configuration file with values "single" or "multi"
2. WHEN user_mode is "single", THE System SHALL operate with the Local_Default_User without requiring authentication
3. WHEN user_mode is "multi", THE System SHALL require authentication for all operations
4. THE Config_Manager SHALL default user_mode to "single" if not specified
5. THE Config_Manager SHALL support NOODEXX_USER_MODE environment variable to override the configuration file
6. WHEN user_mode is changed from "single" to "multi", THE System SHALL migrate existing data to the Local_Default_User

### Requirement 2: User Table and Schema

**User Story:** As a developer, I want a users table to store account information, so that the system can support multiple authenticated users.

#### Acceptance Criteria

1. THE Store SHALL create a users table with columns: id, username, password_hash, email, is_admin, must_change_password, created_at, last_login
2. THE Store SHALL enforce unique constraint on username
3. THE Store SHALL enforce unique constraint on email
4. WHEN the System initializes in single-user mode, THE Store SHALL create the Local_Default_User with username "local-default" and is_admin set to true
5. WHEN the System initializes in multi-user mode AND no users exist, THE Store SHALL create a default admin account with username "admin"
6. WHEN creating the default admin account, THE Store SHALL generate a random temporary password using crypto/rand with at least 16 characters
7. WHEN creating the default admin account, THE Store SHALL log the temporary password to the console with a clear message
8. WHEN creating the default admin account, THE Store SHALL set must_change_password to true
9. THE Store SHALL provide a CreateUser method that accepts username, password, email, is_admin flag, and must_change_password flag
10. THE Store SHALL hash passwords using bcrypt before storing
11. THE Store SHALL provide a GetUserByUsername method that returns user details including must_change_password
12. THE Store SHALL provide a ValidateCredentials method that verifies username and password
13. THE Store SHALL provide an UpdateLastLogin method that records the timestamp of successful authentication
14. THE Store SHALL provide an UpdatePassword method that accepts user_id, new password, and sets must_change_password to false

### Requirement 3: User Identity on Data Tables

**User Story:** As a developer, I want all data tables to include user ownership, so that data can be isolated per user.

#### Acceptance Criteria

1. THE Store SHALL add a user_id column to the chunks table
2. THE Store SHALL add a user_id column to the chat_messages table
3. THE Store SHALL add a user_id column to the sessions table (new table for session metadata)
4. THE Store SHALL add a user_id column to the audit_log table
5. THE Store SHALL add a user_id column to the watched_folders table
6. THE Store SHALL add a user_id column to the skills table (new table for skill metadata)
7. WHEN migrating existing data, THE Store SHALL assign all existing records to the Local_Default_User
8. THE Store SHALL create foreign key constraints from user_id columns to users.id
9. THE Store SHALL create indexes on user_id columns for query performance

### Requirement 4: Document Visibility and Sharing

**User Story:** As a user, I want my documents to be private by default, so that other users cannot access my knowledge base.

#### Acceptance Criteria

1. THE Store SHALL add a visibility column to the chunks table with values "private", "shared", "public"
2. THE Store SHALL default visibility to "private" for all new documents
3. THE Store SHALL add a shared_with column to the chunks table storing comma-separated user IDs
4. WHEN a user queries the library, THE Store SHALL return only documents where user_id matches OR visibility is "public" OR the user's ID is in shared_with
5. WHEN a user performs RAG search, THE Store SHALL filter chunks to only those visible to the user
6. THE Library_Page SHALL display a visibility indicator on each document card
7. THE Library_Page SHALL provide a "Share" button on document cards (UI only, sharing logic in Phase 5)

### Requirement 5: Session Ownership and Isolation

**User Story:** As a user, I want my chat sessions to be private, so that other users cannot see my conversations.

#### Acceptance Criteria

1. THE Store SHALL create a sessions table with columns: id, user_id, title, created_at, last_message_at
2. WHEN a user starts a new chat, THE Store SHALL create a session record with the user's ID
3. WHEN a user loads the chat page, THE UI SHALL display only sessions owned by that user
4. WHEN a user loads a session, THE Store SHALL verify the session belongs to the user before returning messages
5. THE Store SHALL provide a GetUserSessions method that returns sessions for a specific user
6. THE Store SHALL provide a GetSessionOwner method that returns the user_id for a session

### Requirement 6: Audit Log User Context

**User Story:** As a system administrator, I want audit logs to include user information, so that I can track who performed each operation.

#### Acceptance Criteria

1. WHEN an operation is logged, THE Store SHALL include the user_id in the audit entry
2. WHEN an operation is logged, THE Store SHALL include the username in the audit entry
3. THE Audit_Log_Viewer SHALL display username alongside operation details
4. THE Store SHALL provide a GetAuditLogByUser method that filters audit entries by user_id
5. WHEN user_mode is "single", THE System SHALL log all operations with the Local_Default_User

### Requirement 7: Username/Password Authentication

**User Story:** As a user, I want to log in with a username and password, so that I can access my private knowledge base.

#### Acceptance Criteria

1. THE Auth_Package SHALL provide a Userpass_Auth provider that implements the Auth_Provider interface
2. THE Userpass_Auth SHALL provide a Login method that accepts username and password and returns a session token
3. THE Userpass_Auth SHALL validate credentials against the users table
4. THE Userpass_Auth SHALL generate a secure session token using crypto/rand
5. THE Userpass_Auth SHALL store session tokens in a sessions_tokens table with columns: token, user_id, created_at, expires_at
6. THE Userpass_Auth SHALL set session token expiration to 7 days by default
7. THE Userpass_Auth SHALL provide a ValidateToken method that verifies a session token and returns the user_id
8. THE Userpass_Auth SHALL provide a Logout method that invalidates a session token
9. WHEN a login attempt fails, THE Userpass_Auth SHALL log the failure with username and timestamp
10. WHEN a login attempt fails 5 times within 15 minutes, THE Userpass_Auth SHALL temporarily lock the account for 15 minutes

### Requirement 8: Authentication Middleware

**User Story:** As a developer, I want authentication enforced at the HTTP layer, so that unauthorized requests are blocked before reaching handlers.

#### Acceptance Criteria

1. THE API_Package SHALL provide an AuthMiddleware that wraps HTTP handlers
2. WHEN user_mode is "multi", THE AuthMiddleware SHALL extract the session token from the Authorization header or cookie
3. WHEN user_mode is "multi" AND no valid token is present, THE AuthMiddleware SHALL return HTTP 401 Unauthorized
4. WHEN user_mode is "multi" AND a valid token is present, THE AuthMiddleware SHALL add the user_id to the request context
5. WHEN user_mode is "single", THE AuthMiddleware SHALL automatically set the user_id to the Local_Default_User
6. THE API_Handler SHALL retrieve the user_id from request context for all operations
7. THE AuthMiddleware SHALL exclude the /login and /register endpoints from authentication requirements
8. THE AuthMiddleware SHALL exclude static assets (/static/*) from authentication requirements

### Requirement 9: Login and Registration UI

**User Story:** As a user, I want a login page, so that I can authenticate and access my knowledge base.

#### Acceptance Criteria

1. WHEN user_mode is "multi", THE System SHALL display a login page at the root URL (/)
2. WHEN user_mode is "single", THE System SHALL skip the login page entirely and automatically inject the Local_Default_User context without displaying any authentication UI
3. THE Login_Page SHALL provide username and password input fields
4. THE Login_Page SHALL provide a "Login" button that submits credentials
5. WHEN login succeeds AND must_change_password is false, THE System SHALL set a session cookie and redirect to the dashboard
6. WHEN login succeeds AND must_change_password is true, THE System SHALL redirect to a password change page before allowing access to the dashboard
7. WHEN login fails, THE Login_Page SHALL display an error message
8. THE Login_Page SHALL provide a "Register" link that navigates to the registration page
9. THE Registration_Page SHALL provide fields for username, email, and password
10. THE Registration_Page SHALL require password confirmation
11. WHEN registration succeeds, THE System SHALL create the user account and redirect to login
12. THE Password_Change_Page SHALL require the user to enter a new password twice for confirmation
13. WHEN password change succeeds, THE System SHALL update the password, set must_change_password to false, and redirect to the dashboard

### Requirement 10: Admin User Management UI

**User Story:** As an administrator, I want to manage user accounts, so that I can create, modify, and delete users.

#### Acceptance Criteria

1. THE Settings_Page SHALL display a "User Management" section when the current user is an admin
2. THE User_Management_UI SHALL display a table of all users with columns: username, email, is_admin, created_at, last_login
3. THE User_Management_UI SHALL provide a "Create User" button that opens a form
4. THE User_Management_UI SHALL provide an "Edit" button on each user row
5. THE User_Management_UI SHALL provide a "Delete" button on each user row
6. WHEN an admin creates a user, THE System SHALL validate that the username is unique
7. WHEN an admin deletes a user, THE System SHALL prompt for confirmation
8. WHEN an admin deletes a user, THE System SHALL optionally transfer or delete the user's data
9. THE User_Management_UI SHALL allow admins to reset user passwords
10. THE User_Management_UI SHALL allow admins to toggle the is_admin flag for users

### Requirement 11: Skill System User Scoping

**User Story:** As a user, I want my skills to be private, so that other users cannot execute or modify my custom plugins.

#### Acceptance Criteria

1. THE Store SHALL create a skills table with columns: id, user_id, name, path, enabled, created_at
2. WHEN a user loads skills, THE Skills_Package SHALL return only skills owned by that user
3. WHEN a user executes a skill, THE Skills_Package SHALL verify the skill belongs to the user
4. THE Settings_Page SHALL display only the current user's skills
5. THE Command_Palette SHALL display only the current user's manual-trigger skills
6. WHEN a skill is triggered by a keyword or event, THE Skills_Package SHALL execute it in the context of the owning user
7. THE Skills_Package SHALL provide a CreateSkill method that accepts user_id and skill metadata
8. THE Skills_Package SHALL provide a GetUserSkills method that returns skills for a specific user

### Requirement 12: Watched Folder User Scoping

**User Story:** As a user, I want my watched folders to be private, so that other users' files are not ingested into my knowledge base.

#### Acceptance Criteria

1. WHEN a user configures a watched folder, THE Store SHALL associate it with the user's ID
2. WHEN the Folder_Watcher ingests a file, THE Store SHALL assign the chunks to the folder's owner
3. WHEN a user views the Settings_Page, THE UI SHALL display only the user's watched folders
4. THE Folder_Watcher SHALL process files from all users' watched folders
5. THE Folder_Watcher SHALL tag ingested chunks with the owning user's ID
6. THE Settings_Page SHALL allow users to add and remove only their own watched folders

### Requirement 13: MFA and SSO Authentication Stubs

**User Story:** As a developer, I want authentication provider stubs, so that MFA and SSO can be implemented in Phase 5 without refactoring.

#### Acceptance Criteria

1. THE Auth_Package SHALL define an Auth_Provider interface with methods: Login, Logout, ValidateToken, RefreshToken
2. THE Auth_Package SHALL provide an MFA_Auth stub that returns "not implemented" errors
3. THE Auth_Package SHALL provide an SSO_Auth stub that returns "not implemented" errors
4. THE Config_Manager SHALL support auth_provider field with values "userpass", "mfa", "sso"
5. WHEN auth_provider is "mfa" or "sso", THE System SHALL return an error indicating the feature is not yet implemented
6. THE Auth_Package SHALL provide a GetAuthProvider factory function that returns the configured provider

### Requirement 14: Backward Compatibility with Phase 3

**User Story:** As a user, I want my existing Phase 3 data to work seamlessly, so that I don't need to re-ingest documents or reconfigure settings.

#### Acceptance Criteria

1. WHEN upgrading from Phase 3, THE Store SHALL migrate all existing chunks to the Local_Default_User
2. WHEN upgrading from Phase 3, THE Store SHALL migrate all existing chat_messages to the Local_Default_User
3. WHEN upgrading from Phase 3, THE Store SHALL migrate all existing audit_log entries to the Local_Default_User
4. WHEN upgrading from Phase 3, THE Store SHALL migrate all existing watched_folders to the Local_Default_User
5. THE Store SHALL preserve all existing embedding vectors during migration
6. THE Store SHALL preserve all existing chat history during migration
7. WHEN user_mode is "single", THE System SHALL behave identically to Phase 3 from the user's perspective
8. THE Migration SHALL execute in a transaction to ensure atomicity
9. WHEN migration fails, THE Store SHALL return a descriptive error and not start the application

### Requirement 15: Configuration Schema Updates

**User Story:** As a system administrator, I want clear configuration options for multi-user features, so that I can deploy Noodexx appropriately.

#### Acceptance Criteria

1. THE config.json file SHALL include a user_mode field in the root object
2. THE config.json file SHALL include an auth section with fields: provider, session_expiry_days, lockout_threshold, lockout_duration_minutes
3. THE Config_Manager SHALL default auth.provider to "userpass"
4. THE Config_Manager SHALL default auth.session_expiry_days to 7
5. THE Config_Manager SHALL default auth.lockout_threshold to 5
6. THE Config_Manager SHALL default auth.lockout_duration_minutes to 15
7. WHEN config.json is missing user_mode, THE Config_Manager SHALL default to "single"
8. THE Config_Manager SHALL validate that auth.provider is one of: "userpass", "mfa", "sso"
9. THE Config_Manager SHALL create a default config.json with user_mode set to "single" if the file does not exist

