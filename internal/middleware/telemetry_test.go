package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"sakukota-go/internal/middleware"
)

func TestWithTelemetry(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	previousMeters := otel.GetMeterProvider()
	otel.SetMeterProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))
	t.Cleanup(func() { otel.SetMeterProvider(previousMeters) })
	spans := tracetest.NewSpanRecorder()
	previousTracers := otel.GetTracerProvider()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spans)))
	t.Cleanup(func() { otel.SetTracerProvider(previousTracers) })

	r := chi.NewRouter()
	r.Use(middleware.WithTelemetry(8080))
	r.Route("/v1", func(r chi.Router) {
		r.Get("/places/{id}", func(w http.ResponseWriter, r *http.Request) {})
	})
	for _, path := range []string{"/v1/places/42", "/v1/places/43", "/nowhere/1"} {
		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, path, nil))
	}

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &rm))
	counts := map[string]uint64{} // request count by route label and status
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "http.server.request.duration" {
				continue
			}
			for _, dp := range m.Data.(metricdata.Histogram[float64]).DataPoints {
				route, _ := dp.Attributes.Value("http.route")
				status, _ := dp.Attributes.Value("http.response.status_code")
				host, _ := dp.Attributes.Value("server.address")
				assert.Equal(t, attribute.StringValue("sakukota-go"), host)
				counts[route.Emit()+" "+status.Emit()] += dp.Count
			}
		}
	}
	assert.Equal(t, map[string]uint64{"/v1/places/{id} 200": 2, " 404": 1}, counts)

	var names []string
	for _, span := range spans.Ended() {
		names = append(names, span.Name())
	}
	assert.Equal(t, []string{"GET /v1/places/{id}", "GET /v1/places/{id}", "GET"}, names)
}
