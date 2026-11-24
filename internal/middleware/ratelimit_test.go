//go:build integration

package middleware_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"sakukota-go/internal/middleware"
	"sakukota-go/internal/testutil/redistest"
)

var testRedis *redis.Client

func TestMain(m *testing.M) {
	rdb, stop, err := redistest.Start(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	testRedis = rdb

	code := m.Run()
	stop()
	os.Exit(code)
}

// newLimited returns a handler limited to limit requests per window,
// answering 204 when it lets a request through.
func newLimited(t *testing.T, rdb *redis.Client, limit int64, window time.Duration) http.Handler {
	t.Helper()
	require.NoError(t, testRedis.FlushDB(t.Context()).Err(), "flush redis")

	return middleware.WithRateLimit(rdb, limit, window, 0)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
}

// send makes a request to h from the client at addr.
func send(h http.Handler, addr string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.RemoteAddr = addr + ":5000"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func TestWithRateLimit_RefusesOverLimitPerClient(t *testing.T) {
	h := newLimited(t, testRedis, 2, time.Minute)

	assert.Equal(t, http.StatusNoContent, send(h, "203.0.113.7").Code)
	assert.Equal(t, http.StatusNoContent, send(h, "203.0.113.7").Code)
	assert.Equal(t, http.StatusNoContent, send(h, "203.0.113.8").Code, "another client")
	rec := send(h, "203.0.113.7")

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.JSONEq(t, `{"code":"TOO_MANY_REQUESTS","message":"too many requests"}`, rec.Body.String())
}

func TestWithRateLimit_ResetsAfterWindow(t *testing.T) {
	h := newLimited(t, testRedis, 1, time.Second)

	assert.Equal(t, http.StatusNoContent, send(h, "203.0.113.7").Code)
	assert.Equal(t, http.StatusTooManyRequests, send(h, "203.0.113.7").Code)
	time.Sleep(1100 * time.Millisecond)
	assert.Equal(t, http.StatusNoContent, send(h, "203.0.113.7").Code)
}

func TestWithRateLimit_RedisFailureIsInternalError(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", MaxRetries: -1})
	t.Cleanup(func() { _ = rdb.Close() })
	h := newLimited(t, rdb, 1, time.Minute)

	assert.Equal(t, http.StatusInternalServerError, send(h, "203.0.113.7").Code)
}
