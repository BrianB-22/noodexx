package auth

import (
	"context"
	"testing"
	"time"
)

// MockStore implements the Store interface for testing
type MockStore struct {
	users        map[string]*User
	tokens       map[string]*SessionToken
	failedLogins map[string][]time.Time
	lockedUntil  map[string]time.Time
}

func NewMockStore() *MockStore {
	return &MockStore{
		users:        make(map[string]*User),
		tokens:       make(map[string]*SessionToken),
		failedLogins: make(map[string][]time.Time),
		lockedUntil:  make(map[string]time.Time),
	}
}

func (m *MockStore) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	user, ok := m.users[username]
	if !ok {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (m *MockStore) UpdateLastLogin(ctx context.Context, userID int64) error {
	return nil
}

func (m *MockStore) CreateSessionToken(ctx context.Context, token string, userID int64, expiresAt interface{}) error {
	m.tokens[token] = &SessionToken{
		Token:     token,
		UserID:    userID,
		ExpiresAt: expiresAt,
	}
	return nil
}

func (m *MockStore) GetSessionToken(ctx context.Context, token string) (*SessionToken, error) {
	sessionToken, ok := m.tokens[token]
	if !ok {
		return nil, ErrTokenNotFound
	}

	// Check if token has expired
	if expiresAt, ok := sessionToken.ExpiresAt.(time.Time); ok {
		if time.Now().After(expiresAt) {
			return nil, nil // Token expired
		}
	}

	return sessionToken, nil
}

func (m *MockStore) DeleteSessionToken(ctx context.Context, token string) error {
	delete(m.tokens, token)
	return nil
}

func (m *MockStore) IsAccountLocked(ctx context.Context, username string) (bool, interface{}) {
	until, ok := m.lockedUntil[username]
	if !ok {
		return false, time.Time{}
	}
	if time.Now().After(until) {
		delete(m.lockedUntil, username)
		return false, time.Time{}
	}
	return true, until
}

func (m *MockStore) RecordFailedLogin(ctx context.Context, username string) error {
	m.failedLogins[username] = append(m.failedLogins[username], time.Now())
	// Lock account after 5 failed attempts
	if len(m.failedLogins[username]) >= 5 {
		m.lockedUntil[username] = time.Now().Add(15 * time.Minute)
	}
	return nil
}

func (m *MockStore) ClearFailedLogins(ctx context.Context, username string) error {
	delete(m.failedLogins, username)
	delete(m.lockedUntil, username)
	return nil
}

var (
	ErrUserNotFound  = &mockError{"user not found"}
	ErrTokenNotFound = &mockError{"token not found"}
)

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}

func TestPasswordHashing(t *testing.T) {
	password := "testPassword123"

	// Hash the password
	hash, err := hashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	// Verify the hash is not the same as the password
	if hash == password {
		t.Error("Hash should not equal plaintext password")
	}

	// Verify correct password
	if !checkPasswordHash(password, hash) {
		t.Error("Correct password should validate")
	}

	// Verify incorrect password
	if checkPasswordHash("wrongPassword", hash) {
		t.Error("Incorrect password should not validate")
	}
}

func TestTokenGeneration(t *testing.T) {
	// Generate two tokens
	token1, err := generateSecureToken(32)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	token2, err := generateSecureToken(32)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Tokens should be different
	if token1 == token2 {
		t.Error("Generated tokens should be unique")
	}

	// Tokens should not be empty
	if token1 == "" || token2 == "" {
		t.Error("Generated tokens should not be empty")
	}
}

func TestUserpassAuth_Login_Success(t *testing.T) {
	store := NewMockStore()
	auth := NewUserpassAuth(store, 7, 5, 15)

	// Create a test user
	password := "testPassword123"
	hash, _ := hashPassword(password)
	store.users["testuser"] = &User{
		ID:           1,
		Username:     "testuser",
		PasswordHash: hash,
	}

	// Attempt login
	token, err := auth.Login(context.Background(), "testuser", password)
	if err != nil {
		t.Fatalf("Login should succeed: %v", err)
	}

	// Token should not be empty
	if token == "" {
		t.Error("Token should not be empty")
	}

	// Token should be stored
	if _, ok := store.tokens[token]; !ok {
		t.Error("Token should be stored in database")
	}
}

func TestUserpassAuth_Login_InvalidCredentials(t *testing.T) {
	store := NewMockStore()
	auth := NewUserpassAuth(store, 7, 5, 15)

	// Create a test user
	password := "testPassword123"
	hash, _ := hashPassword(password)
	store.users["testuser"] = &User{
		ID:           1,
		Username:     "testuser",
		PasswordHash: hash,
	}

	// Attempt login with wrong password
	_, err := auth.Login(context.Background(), "testuser", "wrongPassword")
	if err == nil {
		t.Error("Login should fail with wrong password")
	}

	// Failed login should be recorded
	if len(store.failedLogins["testuser"]) != 1 {
		t.Error("Failed login should be recorded")
	}
}

func TestUserpassAuth_ValidateToken(t *testing.T) {
	store := NewMockStore()
	auth := NewUserpassAuth(store, 7, 5, 15)

	// Create a test user and login
	password := "testPassword123"
	hash, _ := hashPassword(password)
	store.users["testuser"] = &User{
		ID:           1,
		Username:     "testuser",
		PasswordHash: hash,
	}

	token, _ := auth.Login(context.Background(), "testuser", password)

	// Validate the token
	userID, err := auth.ValidateToken(context.Background(), token)
	if err != nil {
		t.Fatalf("Token validation should succeed: %v", err)
	}

	if userID != 1 {
		t.Errorf("Expected userID 1, got %d", userID)
	}
}

func TestUserpassAuth_Logout(t *testing.T) {
	store := NewMockStore()
	auth := NewUserpassAuth(store, 7, 5, 15)

	// Create a test user and login
	password := "testPassword123"
	hash, _ := hashPassword(password)
	store.users["testuser"] = &User{
		ID:           1,
		Username:     "testuser",
		PasswordHash: hash,
	}

	token, _ := auth.Login(context.Background(), "testuser", password)

	// Logout
	err := auth.Logout(context.Background(), token)
	if err != nil {
		t.Fatalf("Logout should succeed: %v", err)
	}

	// Token should be deleted
	if _, ok := store.tokens[token]; ok {
		t.Error("Token should be deleted after logout")
	}

	// Validating the token should fail
	_, err = auth.ValidateToken(context.Background(), token)
	if err == nil {
		t.Error("Token validation should fail after logout")
	}
}

func TestGetProvider(t *testing.T) {
	store := NewMockStore()

	// Test userpass provider
	provider, err := GetProvider("userpass", store, 7, 5, 15)
	if err != nil {
		t.Fatalf("GetProvider should succeed for userpass: %v", err)
	}
	if _, ok := provider.(*UserpassAuth); !ok {
		t.Error("Provider should be UserpassAuth")
	}

	// Test MFA stub
	provider, err = GetProvider("mfa", store, 7, 5, 15)
	if err != nil {
		t.Fatalf("GetProvider should succeed for mfa: %v", err)
	}
	if _, ok := provider.(*MFAAuth); !ok {
		t.Error("Provider should be MFAAuth")
	}

	// Test SSO stub
	provider, err = GetProvider("sso", store, 7, 5, 15)
	if err != nil {
		t.Fatalf("GetProvider should succeed for sso: %v", err)
	}
	if _, ok := provider.(*SSOAuth); !ok {
		t.Error("Provider should be SSOAuth")
	}

	// Test invalid provider
	_, err = GetProvider("invalid", store, 7, 5, 15)
	if err == nil {
		t.Error("GetProvider should fail for invalid provider")
	}
}

func TestMFAAuth_Stubs(t *testing.T) {
	mfa := &MFAAuth{}

	_, err := mfa.Login(context.Background(), "user", "pass")
	if err == nil || err.Error() != "MFA authentication not yet implemented" {
		t.Error("MFA Login should return not implemented error")
	}

	err = mfa.Logout(context.Background(), "token")
	if err == nil || err.Error() != "MFA authentication not yet implemented" {
		t.Error("MFA Logout should return not implemented error")
	}

	_, err = mfa.ValidateToken(context.Background(), "token")
	if err == nil || err.Error() != "MFA authentication not yet implemented" {
		t.Error("MFA ValidateToken should return not implemented error")
	}

	_, err = mfa.RefreshToken(context.Background(), "token")
	if err == nil || err.Error() != "MFA authentication not yet implemented" {
		t.Error("MFA RefreshToken should return not implemented error")
	}
}

func TestSSOAuth_Stubs(t *testing.T) {
	sso := &SSOAuth{}

	_, err := sso.Login(context.Background(), "user", "pass")
	if err == nil || err.Error() != "SSO authentication not yet implemented" {
		t.Error("SSO Login should return not implemented error")
	}

	err = sso.Logout(context.Background(), "token")
	if err == nil || err.Error() != "SSO authentication not yet implemented" {
		t.Error("SSO Logout should return not implemented error")
	}

	_, err = sso.ValidateToken(context.Background(), "token")
	if err == nil || err.Error() != "SSO authentication not yet implemented" {
		t.Error("SSO ValidateToken should return not implemented error")
	}

	_, err = sso.RefreshToken(context.Background(), "token")
	if err == nil || err.Error() != "SSO authentication not yet implemented" {
		t.Error("SSO RefreshToken should return not implemented error")
	}
}
