package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"rantau-go/internal/config"
	"rantau-go/internal/pkg/httpx"
	"rantau-go/internal/pkg/jwt"
)

type contextKey string

const (
	userIDKey contextKey = "user_id"
	roleKey   contextKey = "role"
)

// WithAuth validates the bearer token and stores the user ID and role in the
// request context.
func WithAuth(cfg config.Config) httpx.Constructor {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			tokenStr, found := strings.CutPrefix(authHeader, "Bearer ")
			if !found {
				httpx.RespondError(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "missing bearer token")
				return
			}

			claims, err := jwt.ValidateToken(cfg, tokenStr)
			if err != nil {
				httpx.RespondError(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, claims.Subject)
			ctx = context.WithValue(ctx, roleKey, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext returns the authenticated user ID set by WithAuth.
func UserIDFromContext(ctx context.Context) (int64, bool) {
	// WithAuth stores the token's subject, which holds the ID as a string.
	subject, ok := ctx.Value(userIDKey).(string)
	if !ok {
		return 0, false
	}

	userID, err := strconv.ParseInt(subject, 10, 64)
	if err != nil {
		return 0, false
	}

	return userID, true
}

// RoleFromContext returns the authenticated user role set by WithAuth.
func RoleFromContext(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(roleKey).(string)
	return role, ok
}
