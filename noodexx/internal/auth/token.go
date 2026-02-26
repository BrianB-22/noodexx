package auth

import (
	"crypto/rand"
	"encoding/base64"
)

// generateSecureToken generates a cryptographically secure random token
// using crypto/rand with the specified number of bytes (32 bytes = 256 bits of entropy)
func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}
