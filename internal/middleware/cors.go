package middleware

import (
	"net/http"

	"rantau-go/internal/config"
	"rantau-go/internal/pkg/httpx"
)

// WithCORS allows cross-origin requests from the configured origin and
// answers preflight requests.
func WithCORS(cfg config.Config) httpx.Constructor {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := w.Header()
			header.Set("Access-Control-Allow-Origin", cfg.CORSAllowedOrigin)
			header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			header.Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			header.Add("Vary", "Origin")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
