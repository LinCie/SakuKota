package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"sakukota-go/internal/middleware"
)

func TestWithTimeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		wantErr error
	}{
		{name: "expired", timeout: time.Millisecond, wantErr: context.DeadlineExceeded},
		{name: "in-time", timeout: time.Minute, wantErr: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got error
			h := middleware.WithTimeout(tt.timeout)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				select {
				case <-r.Context().Done():
				case <-time.After(50 * time.Millisecond):
				}
				got = r.Context().Err()
			}))

			h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

			assert.Equal(t, tt.wantErr, got)
		})
	}
}

func TestWithTimeout_SetsDeadline(t *testing.T) {
	var deadline time.Time
	var ok bool
	h := middleware.WithTimeout(time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deadline, ok = r.Context().Deadline()
	}))
	start := time.Now()

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	assert.True(t, ok, "context has no deadline")
	assert.WithinDuration(t, start.Add(time.Minute), deadline, time.Second)
}
