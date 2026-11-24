package middleware

import "net/http"

// securityHeaders are set on every response. The API serves only JSON, so the
// policy forbids loading, framing, sniffing, caching, and cross-origin
// embedding. Browsers ignore Strict-Transport-Security over plain HTTP, so it
// is safe to send in local development.
var securityHeaders = map[string]string{
	"Cache-Control":                "no-store",
	"Content-Security-Policy":      "default-src 'none'; frame-ancestors 'none'",
	"Cross-Origin-Resource-Policy": "same-origin",
	"Permissions-Policy":           "geolocation=(), camera=(), microphone=()",
	"Referrer-Policy":              "no-referrer",
	"Strict-Transport-Security":    "max-age=31536000",
	"X-Content-Type-Options":       "nosniff",
	"X-Frame-Options":              "DENY",
}

// WithSecurityHeaders sets securityHeaders before calling the next handler,
// which may override them (the HTML docs page relaxes the content policy).
func WithSecurityHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			for k, v := range securityHeaders {
				h.Set(k, v)
			}
			next.ServeHTTP(w, r)
		})
	}
}
