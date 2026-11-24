package middleware

import (
	"context"
	"net/http"
	"time"
)

// WithTimeout cancels each request's context after d, so a slow handler stops
// its database, Redis, and storage calls instead of running on after the
// server has given up on the response. The handler still writes the response,
// which for a timed-out call is the usual 500.
func WithTimeout(d time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
