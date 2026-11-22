package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// WithRequestLog logs one line per request to logger once it is served. It
// logs the request context, so a handler that adds the trace to each record
// links the line to its trace. It must run inside a chi router: it logs the
// matched route pattern, such as /v1/places/{id}, and empty when none matched.
// It never logs the raw path, query string, or client address, which may hold
// IDs or personal data.
func WithRequestLog(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			status := ww.Status()
			if status == 0 { // nothing written, so net/http sends 200
				status = http.StatusOK
			}
			logger.InfoContext(r.Context(), "request",
				"method", r.Method,
				"route", chi.RouteContext(r.Context()).RoutePattern(),
				"status", status,
				"bytes", ww.BytesWritten(),
				"duration", time.Since(start),
			)
		})
	}
}
