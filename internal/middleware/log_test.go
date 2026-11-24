package middleware_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"sakukota-go/internal/middleware"
)

func TestWithRequestLog(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		wantRoute  string
		wantStatus float64
		wantBytes  float64
	}{
		{name: "written", target: "/places/42?q=secret", wantRoute: "/places/{id}", wantStatus: http.StatusCreated, wantBytes: 2},
		{name: "nothing-written", target: "/empty", wantRoute: "/empty", wantStatus: http.StatusOK, wantBytes: 0},
		{name: "not-found", target: "/nowhere", wantRoute: "", wantStatus: http.StatusNotFound, wantBytes: 19},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			r := chi.NewRouter()
			r.Use(middleware.WithRequestLog(slog.New(slog.NewJSONHandler(&buf, nil))))
			r.Get("/places/{id}", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte("{}")) // the recorder cannot fail
			})
			r.Get("/empty", func(w http.ResponseWriter, r *http.Request) {})

			r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, tt.target, nil))

			var record map[string]any
			require.NoError(t, json.Unmarshal(buf.Bytes(), &record))
			assert.Equal(t, "request", record["msg"])
			assert.Equal(t, "GET", record["method"])
			assert.Equal(t, tt.wantRoute, record["route"])
			assert.Equal(t, tt.wantStatus, record["status"])
			assert.Equal(t, tt.wantBytes, record["bytes"])
			assert.NotContains(t, buf.String(), "/places/42", "raw path must not be logged")
			assert.NotContains(t, buf.String(), "/nowhere", "raw path must not be logged")
			assert.NotContains(t, buf.String(), "secret", "query string must not be logged")
			assert.Contains(t, record, "duration")
		})
	}
}
