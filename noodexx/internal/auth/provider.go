package auth

import (
	"context"
	"errors"
	"fmt"
)

// Common errors
var (
	ErrUserIDNotFound = errors.New("user_id not found in context")
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

// Store defines the interface for database operations needed by auth providers
type Store interface {
	// User operations
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	UpdateLastLogin(ctx context.Context, userID int64) error

	// Session token operations
	CreateSessionToken(ctx context.Context, token string, userID int64, expiresAt interface{}) error
	GetSessionToken(ctx context.Context, token string) (*SessionToken, error)
	DeleteSessionToken(ctx context.Context, token string) error

	// Account lockout operations
	IsAccountLocked(ctx context.Context, username string) (bool, interface{})
	RecordFailedLogin(ctx context.Context, username string) error
	ClearFailedLogins(ctx context.Context, username string) error
}

// User represents a user account
type User struct {
	ID                 int64
	Username           string
	PasswordHash       string
	Email              string
	IsAdmin            bool
	MustChangePassword bool
}

// SessionToken represents a session token
type SessionToken struct {
	Token     string
	UserID    int64
	ExpiresAt interface{}
}

// GetProvider returns the configured auth provider
func GetProvider(providerType string, store Store, sessionExpiryDays, lockoutThreshold, lockoutDurationMinutes int) (Provider, error) {
	switch providerType {
	case "userpass", "":
		return NewUserpassAuth(store, sessionExpiryDays, lockoutThreshold, lockoutDurationMinutes), nil
	case "mfa":
		return &MFAAuth{}, nil
	case "sso":
		return &SSOAuth{}, nil
	default:
		return nil, fmt.Errorf("unsupported auth provider: %s", providerType)
	}
}
