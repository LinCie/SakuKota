package middleware

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

// WithTelemetry traces each request and records OpenTelemetry HTTP server
// metrics (request count, duration, and sizes) through the global providers.
// It must run inside a chi router: spans are named and metrics labelled by the
// matched route pattern, such as /v1/places/{id}, never the raw path.
//
// port fixes the server address label, which otelhttp would otherwise take from
// the client's Host header and so let clients create unbounded label values.
func WithTelemetry(port int) func(http.Handler) http.Handler {
	instrument := otelhttp.NewMiddleware("api", otelhttp.WithServerName(fmt.Sprintf("sakukota-go:%d", port)))
	return func(next http.Handler) http.Handler {
		return instrument(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
			// otelhttp cannot see the route itself: chi sets Request.Pattern on
			// a copy of the request. The route context is shared, so read it here.
			route := chi.RouteContext(r.Context()).RoutePattern()
			if route == "" {
				return
			}
			labeler, _ := otelhttp.LabelerFromContext(r.Context()) // always set by instrument
			labeler.Add(semconv.HTTPRoute(route))
			span := trace.SpanFromContext(r.Context())
			span.SetName(r.Method + " " + route)
			span.SetAttributes(semconv.HTTPRoute(route))
		}))
	}
}
