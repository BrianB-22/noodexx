package auth

import (
	"context"
	"net/http"
	"strings"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

// UserIDKey is the context key for storing user_id in request context
const UserIDKey contextKey = "user_id"

// AuthMiddleware creates HTTP middleware for authentication
// In single-user mode: automatically injects local-default user_id
// In multi-user mode: validates session token and injects user_id
func AuthMiddleware(store Store, userMode string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip authentication for public endpoints
			if isPublicEndpoint(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// Single-user mode: automatically inject local-default user
			if userMode == "single" {
				user, err := store.GetUserByUsername(r.Context(), "local-default")
				if err != nil {
					http.Error(w, "System error: local-default user not found", http.StatusInternalServerError)
					return
				}
				ctx := context.WithValue(r.Context(), UserIDKey, user.ID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Multi-user mode: validate token and inject user_id
			token := extractToken(r)
			if token == "" {
				// Check if this is a browser request (not API)
				if !strings.HasPrefix(r.URL.Path, "/api/") {
					http.Redirect(w, r, "/login", http.StatusSeeOther)
					return
				}
				http.Error(w, "Unauthorized: authentication required", http.StatusUnauthorized)
				return
			}

			// Validate token and get user_id
			sessionToken, err := store.GetSessionToken(r.Context(), token)
			if err != nil {
				// Check if this is a browser request (not API)
				if !strings.HasPrefix(r.URL.Path, "/api/") {
					http.Redirect(w, r, "/login", http.StatusSeeOther)
					return
				}
				http.Error(w, "Unauthorized: invalid session", http.StatusUnauthorized)
				return
			}
			if sessionToken == nil {
				// Check if this is a browser request (not API)
				if !strings.HasPrefix(r.URL.Path, "/api/") {
					http.Redirect(w, r, "/login", http.StatusSeeOther)
					return
				}
				http.Error(w, "Unauthorized: invalid or expired session", http.StatusUnauthorized)
				return
			}

			// Inject user_id into request context
			ctx := context.WithValue(r.Context(), UserIDKey, sessionToken.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractToken extracts the session token from the request
// First checks Authorization header with "Bearer " prefix
// Falls back to session_token cookie if header not present
// Returns empty string if neither is found
func extractToken(r *http.Request) string {
	// Try Authorization header first
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}

	// Fall back to cookie
	cookie, err := r.Cookie("session_token")
	if err == nil {
		return cookie.Value
	}

	return ""
}

// isPublicEndpoint checks if a path should bypass authentication
// Public endpoints: /login, /register, /static/, /api/login, /api/register
func isPublicEndpoint(path string) bool {
	publicPaths := []string{
		"/login",
		"/register",
		"/static/",
		"/api/login",
		"/api/register",
	}

	for _, p := range publicPaths {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// GetUserID extracts the user_id from request context
// Returns (userID int64, error)
// Returns error if user_id not found in context
func GetUserID(ctx context.Context) (int64, error) {
	userID, ok := ctx.Value(UserIDKey).(int64)
	if !ok {
		return 0, ErrUserIDNotFound
	}
	return userID, nil
}
