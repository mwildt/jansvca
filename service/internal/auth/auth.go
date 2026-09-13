// Package auth provides OAuth2 authentication and a minimal role model
// (single Administrator role with all permissions).
package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

// Role is an authorization role.
type Role string

const (
	// RoleAdmin has all permissions. For now every authenticated user is an
	// admin.
	RoleAdmin Role = "administrator"
)

// User is the authenticated principal.
type User struct {
	Subject string
	Name    string
	Roles   []Role
}

// IsAdmin reports whether the user has the Administrator role.
func (u User) IsAdmin() bool {
	for _, r := range u.Roles {
		if r == RoleAdmin {
			return true
		}
	}
	return false
}

// TokenVerifier verifies an OAuth2 bearer token and returns the principal.
// Implementations call the configured OAuth2 provider's token/introspection
// endpoint; the provider is intentionally pluggable.
type TokenVerifier interface {
	Verify(ctx context.Context, token string) (User, error)
}

// ErrUnauthorized is returned when no valid token is present.
var ErrUnauthorized = errors.New("auth: unauthorized")

// Middleware returns an http middleware that requires a valid OAuth2 bearer
// token. Authenticated users are stored in the request context.
func Middleware(verifier TokenVerifier, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		user, err := verifier.Verify(r.Context(), token)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), user)))
	})
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) {
		return ""
	}
	return strings.TrimSpace(h[len(prefix):])
}

type ctxKey struct{}

// WithUser returns a copy of ctx carrying the authenticated user.
func WithUser(ctx context.Context, u User) context.Context {
	return context.WithValue(ctx, ctxKey{}, u)
}

// FromUser returns the authenticated user from the context, if any.
func FromUser(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(ctxKey{}).(User)
	return u, ok
}
