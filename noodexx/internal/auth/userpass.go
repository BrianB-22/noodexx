package auth

import (
	"context"
	"fmt"
	"time"
)

// UserpassAuth implements username/password authentication
type UserpassAuth struct {
	store            Store
	sessionExpiry    time.Duration
	lockoutThreshold int
	lockoutDuration  time.Duration
}

// NewUserpassAuth creates a new username/password auth provider
func NewUserpassAuth(store Store, sessionExpiryDays, lockoutThreshold, lockoutDurationMinutes int) *UserpassAuth {
	return &UserpassAuth{
		store:            store,
		sessionExpiry:    time.Duration(sessionExpiryDays) * 24 * time.Hour,
		lockoutThreshold: lockoutThreshold,
		lockoutDuration:  time.Duration(lockoutDurationMinutes) * time.Minute,
	}
}

// Login authenticates credentials and returns a session token
func (u *UserpassAuth) Login(ctx context.Context, username, password string) (string, error) {
	// Check if account is locked
	locked, until := u.store.IsAccountLocked(ctx, username)
	if locked {
		// Convert until to time.Time if it's not already
		var untilTime time.Time
		switch v := until.(type) {
		case time.Time:
			untilTime = v
		default:
			untilTime = time.Now().Add(u.lockoutDuration)
		}
		return "", fmt.Errorf("account locked until %s", untilTime.Format(time.RFC3339))
	}

	// Get user by username
	user, err := u.store.GetUserByUsername(ctx, username)
	if err != nil {
		// Record failed attempt
		u.store.RecordFailedLogin(ctx, username)
		return "", fmt.Errorf("invalid credentials")
	}

	// Validate password
	if !checkPasswordHash(password, user.PasswordHash) {
		// Record failed attempt
		u.store.RecordFailedLogin(ctx, username)
		return "", fmt.Errorf("invalid credentials")
	}

	// Generate secure session token (32 bytes = 256 bits of entropy)
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

// Logout invalidates a session token
func (u *UserpassAuth) Logout(ctx context.Context, token string) error {
	return u.store.DeleteSessionToken(ctx, token)
}

// ValidateToken verifies a token and returns the user_id
func (u *UserpassAuth) ValidateToken(ctx context.Context, token string) (int64, error) {
	sessionToken, err := u.store.GetSessionToken(ctx, token)
	if err != nil {
		return 0, fmt.Errorf("invalid token: %w", err)
	}

	// Check if token is expired
	var expiresAt time.Time
	switch v := sessionToken.ExpiresAt.(type) {
	case time.Time:
		expiresAt = v
	default:
		return 0, fmt.Errorf("invalid expiration time format")
	}

	if time.Now().After(expiresAt) {
		// Token is expired, delete it
		u.store.DeleteSessionToken(ctx, token)
		return 0, fmt.Errorf("token expired")
	}

	return sessionToken.UserID, nil
}

// RefreshToken extends a session token's expiration (stub for Phase 5)
func (u *UserpassAuth) RefreshToken(ctx context.Context, token string) (string, error) {
	return "", fmt.Errorf("token refresh not implemented in Phase 4")
}
