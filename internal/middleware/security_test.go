package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"

	"sakukota-go/internal/middleware"
)

func TestWithSecurityHeaders(t *testing.T) {
	want := map[string]string{
		"Cache-Control":                "no-store",
		"Content-Security-Policy":      "default-src 'none'; frame-ancestors 'none'",
		"Cross-Origin-Resource-Policy": "same-origin",
		"Permissions-Policy":           "geolocation=(), camera=(), microphone=()",
		"Referrer-Policy":              "no-referrer",
		"Strict-Transport-Security":    "max-age=31536000",
		"X-Content-Type-Options":       "nosniff",
		"X-Frame-Options":              "DENY",
	}
	tests := []struct {
		name   string
		target string
	}{
		{name: "handled", target: "/ok"},
		{name: "not-found", target: "/nowhere"},
		{name: "error", target: "/error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Use(middleware.WithSecurityHeaders())
			r.Get("/ok", func(w http.ResponseWriter, r *http.Request) {})
			r.Get("/error", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			})
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.target, nil))

			for k, v := range want {
				assert.Equal(t, v, rec.Header().Get(k), k)
			}
		})
	}
}
