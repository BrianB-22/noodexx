package auth

import (
	"context"
	"fmt"
)

// MFAAuth is a stub for multi-factor authentication (Phase 5)
type MFAAuth struct{}

// Login returns a "not implemented" error
func (m *MFAAuth) Login(ctx context.Context, username, password string) (string, error) {
	return "", fmt.Errorf("MFA authentication not yet implemented")
}

// Logout returns a "not implemented" error
func (m *MFAAuth) Logout(ctx context.Context, token string) error {
	return fmt.Errorf("MFA authentication not yet implemented")
}

// ValidateToken returns a "not implemented" error
func (m *MFAAuth) ValidateToken(ctx context.Context, token string) (int64, error) {
	return 0, fmt.Errorf("MFA authentication not yet implemented")
}

// RefreshToken returns a "not implemented" error
func (m *MFAAuth) RefreshToken(ctx context.Context, token string) (string, error) {
	return "", fmt.Errorf("MFA authentication not yet implemented")
}

// SSOAuth is a stub for single sign-on authentication (Phase 5)
type SSOAuth struct{}

// Login returns a "not implemented" error
func (s *SSOAuth) Login(ctx context.Context, username, password string) (string, error) {
	return "", fmt.Errorf("SSO authentication not yet implemented")
}

// Logout returns a "not implemented" error
func (s *SSOAuth) Logout(ctx context.Context, token string) error {
	return fmt.Errorf("SSO authentication not yet implemented")
}

// ValidateToken returns a "not implemented" error
func (s *SSOAuth) ValidateToken(ctx context.Context, token string) (int64, error) {
	return 0, fmt.Errorf("SSO authentication not yet implemented")
}

// RefreshToken returns a "not implemented" error
func (s *SSOAuth) RefreshToken(ctx context.Context, token string) (string, error) {
	return "", fmt.Errorf("SSO authentication not yet implemented")
}
