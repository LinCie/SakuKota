package auth_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"rantau-go/internal/module/auth"
	"rantau-go/internal/module/auth/mock"
	"rantau-go/internal/module/users"
)

const refreshCookieName = "refresh_token"

// refreshMaxAge is RefreshTokenExpiry from testConfig, in seconds.
var refreshMaxAge = int((24 * time.Hour).Seconds())

// -----------------------------------------------------------------------------
// helpers
// -----------------------------------------------------------------------------

func do(handler http.HandlerFunc, body string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/v1/auth", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	handler(rec, req)
	return rec
}

// envelope covers both the { "data": ... } success shape and the
// { "code": ..., "message": ... } error shape.
type envelope struct {
	Data    json.RawMessage `json:"data"`
	Code    string          `json:"code"`
	Message string          `json:"message"`
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) envelope {
	t.Helper()
	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env), "response is not JSON; body=%s", rec.Body.String())
	return env
}

func expectError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	assert.Equal(t, status, rec.Code, "body=%s", rec.Body.String())
	assert.Equal(t, code, decode(t, rec).Code)
}

func cookieNamed(rec *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func expectCookieCleared(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	c := cookieNamed(rec, refreshCookieName)
	require.NotNil(t, c, "refresh cookie was not sent to clear it")
	assert.Empty(t, c.Value, "refresh cookie not cleared")
	assert.Negative(t, c.MaxAge, "refresh cookie not cleared")
}

func TestHandler_Register(t *testing.T) {
	t.Run("created", func(t *testing.T) {
		svc := mock.NewMockService(gomock.NewController(t))
		svc.EXPECT().Register(gomock.Any(), &users.User{Name: "Ana", Email: testEmail}, "password123").
			Return(&users.User{ID: 1, Name: "Ana", Email: testEmail}, nil)

		h := auth.NewHandler(testConfig(), svc)
		rec := do(h.Register, `{"name":"Ana","email":"ana@example.com","password":"password123"}`)
		require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())

		var got auth.UserResponse
		require.NoError(t, json.Unmarshal(decode(t, rec).Data, &got), "decode data")
		assert.Equal(t, int64(1), got.ID)
		assert.Equal(t, testEmail, got.Email)
		assert.NotContains(t, rec.Body.String(), "password", "response body mentions a password")
	})

	t.Run("invalid json", func(t *testing.T) {
		// No expectations: a malformed body must never reach the service.
		svc := mock.NewMockService(gomock.NewController(t))
		h := auth.NewHandler(testConfig(), svc)

		rec := do(h.Register, `{"name":`)
		expectError(t, rec, http.StatusBadRequest, "INVALID_BODY")
	})

	t.Run("error mapping", func(t *testing.T) {
		tests := []struct {
			name   string
			err    error
			status int
			code   string
		}{
			{name: "invalid name", err: auth.ErrInvalidName, status: http.StatusBadRequest, code: "VALIDATION_ERROR"},
			{name: "invalid email", err: auth.ErrInvalidEmail, status: http.StatusBadRequest, code: "VALIDATION_ERROR"},
			{name: "weak password", err: auth.ErrWeakPassword, status: http.StatusBadRequest, code: "VALIDATION_ERROR"},
			{name: "email taken", err: users.ErrEmailTaken, status: http.StatusConflict, code: "EMAIL_TAKEN"},
			{name: "unexpected failure", err: errBoom, status: http.StatusInternalServerError, code: "INTERNAL_ERROR"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				svc := mock.NewMockService(gomock.NewController(t))
				svc.EXPECT().Register(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, tc.err)

				h := auth.NewHandler(testConfig(), svc)
				rec := do(h.Register, `{"name":"Ana","email":"ana@example.com","password":"password123"}`)
				expectError(t, rec, tc.status, tc.code)
			})
		}
	})

	t.Run("internal error hides details", func(t *testing.T) {
		// Storage details must not reach the client on a 500.
		svc := mock.NewMockService(gomock.NewController(t))
		svc.EXPECT().Register(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, errors.New("pq: connection refused"))

		h := auth.NewHandler(testConfig(), svc)
		rec := do(h.Register, `{"name":"Ana","email":"ana@example.com","password":"password123"}`)
		assert.NotContains(t, decode(t, rec).Message, "connection refused", "500 message exposes internal error")
	})
}

func TestHandler_Login(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := mock.NewMockService(gomock.NewController(t))
		svc.EXPECT().Login(gomock.Any(), testEmail, "password123").Return(
			&users.User{ID: 1, Name: "Ana", Email: testEmail, Role: "admin"},
			&auth.Tokens{AccessToken: "access-1", RefreshToken: "refresh-1"},
			nil,
		)

		h := auth.NewHandler(testConfig(), svc)
		rec := do(h.Login, `{"email":"ana@example.com","password":"password123"}`)
		require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

		var session auth.SessionResponse
		require.NoError(t, json.Unmarshal(decode(t, rec).Data, &session), "decode data")
		assert.Equal(t, "access-1", session.AccessToken)
		assert.Equal(t, testEmail, session.User.Email)
		assert.Equal(t, "admin", session.User.Role)
		assert.NotContains(t, rec.Body.String(), "refresh-1", "refresh token appears in the response body")

		c := cookieNamed(rec, refreshCookieName)
		require.NotNil(t, c, "refresh cookie was not set")
		assert.Equal(t, "refresh-1", c.Value)
		assert.True(t, c.HttpOnly, "refresh cookie is not HttpOnly")
		assert.Equal(t, "/v1/auth", c.Path)
		assert.Equal(t, http.SameSiteStrictMode, c.SameSite)
		assert.Equal(t, testConfig().CookieSecure, c.Secure)
		assert.Equal(t, refreshMaxAge, c.MaxAge)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		svc := mock.NewMockService(gomock.NewController(t))
		svc.EXPECT().Login(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil, auth.ErrInvalidCredentials)

		h := auth.NewHandler(testConfig(), svc)
		rec := do(h.Login, `{"email":"ana@example.com","password":"wrong"}`)
		expectError(t, rec, http.StatusUnauthorized, "INVALID_CREDENTIALS")
		assert.Nil(t, cookieNamed(rec, refreshCookieName), "refresh cookie was set on failed login")
	})

	t.Run("unexpected failure", func(t *testing.T) {
		svc := mock.NewMockService(gomock.NewController(t))
		svc.EXPECT().Login(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil, errBoom)

		h := auth.NewHandler(testConfig(), svc)
		rec := do(h.Login, `{"email":"ana@example.com","password":"password123"}`)
		expectError(t, rec, http.StatusInternalServerError, "INTERNAL_ERROR")
	})

	t.Run("context reaches service", func(t *testing.T) {
		cancelled := func() (context.Context, context.CancelFunc) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			return ctx, cancel
		}

		// The service must see the request's context: it is cancelled here, so a
		// service that ignores it would succeed.
		svc := mock.NewMockService(gomock.NewController(t))
		svc.EXPECT().Login(gomock.Any(), testEmail, "password123").DoAndReturn(
			func(got context.Context, _, _ string) (*users.User, *auth.Tokens, error) {
				assert.Error(t, got.Err(), "service was not given the request's context")
				return nil, nil, got.Err()
			})

		ctx, cancel := cancelled()
		defer cancel()
		req := httptest.NewRequest(http.MethodPost, "/v1/auth/login",
			strings.NewReader(`{"email":"`+testEmail+`","password":"password123"}`)).WithContext(ctx)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h := auth.NewHandler(testConfig(), svc)
		h.Login(rec, req)

		assert.NotEqual(t, http.StatusOK, rec.Code, "status 200 for a request whose context had ended")
	})
}

func TestHandler_Refresh(t *testing.T) {
	t.Run("missing cookie", func(t *testing.T) {
		// No expectations: without a cookie the service must not be called.
		svc := mock.NewMockService(gomock.NewController(t))
		h := auth.NewHandler(testConfig(), svc)

		rec := do(h.Refresh, "")
		expectError(t, rec, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN")
		expectCookieCleared(t, rec)
	})

	t.Run("success", func(t *testing.T) {
		svc := mock.NewMockService(gomock.NewController(t))
		svc.EXPECT().Refresh(gomock.Any(), "refresh-1").Return(
			&users.User{ID: 1, Name: "Ana", Email: testEmail},
			&auth.Tokens{AccessToken: "access-2", RefreshToken: "refresh-2"},
			nil,
		)

		h := auth.NewHandler(testConfig(), svc)
		rec := do(h.Refresh, "", &http.Cookie{Name: refreshCookieName, Value: "refresh-1"})
		require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

		var session auth.SessionResponse
		require.NoError(t, json.Unmarshal(decode(t, rec).Data, &session), "decode data")
		assert.Equal(t, "access-2", session.AccessToken)

		c := cookieNamed(rec, refreshCookieName)
		require.NotNil(t, c, "replacement refresh cookie was not set")
		assert.Equal(t, "refresh-2", c.Value)
		assert.Equal(t, refreshMaxAge, c.MaxAge)
	})

	t.Run("invalid token clears cookie", func(t *testing.T) {
		svc := mock.NewMockService(gomock.NewController(t))
		svc.EXPECT().Refresh(gomock.Any(), "stale").Return(nil, nil, auth.ErrInvalidRefreshToken)

		h := auth.NewHandler(testConfig(), svc)
		rec := do(h.Refresh, "", &http.Cookie{Name: refreshCookieName, Value: "stale"})
		expectError(t, rec, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN")
		expectCookieCleared(t, rec)
	})

	t.Run("unexpected failure", func(t *testing.T) {
		svc := mock.NewMockService(gomock.NewController(t))
		svc.EXPECT().Refresh(gomock.Any(), gomock.Any()).Return(nil, nil, errBoom)

		h := auth.NewHandler(testConfig(), svc)
		rec := do(h.Refresh, "", &http.Cookie{Name: refreshCookieName, Value: "refresh-1"})
		expectError(t, rec, http.StatusInternalServerError, "INTERNAL_ERROR")
	})
}

func TestHandler_Logout(t *testing.T) {
	t.Run("without cookie", func(t *testing.T) {
		// No expectations: without a cookie there is no session to revoke.
		svc := mock.NewMockService(gomock.NewController(t))
		h := auth.NewHandler(testConfig(), svc)

		rec := do(h.Logout, "")
		assert.Equal(t, http.StatusNoContent, rec.Code)
		expectCookieCleared(t, rec)
	})

	t.Run("revokes cookie token", func(t *testing.T) {
		svc := mock.NewMockService(gomock.NewController(t))
		svc.EXPECT().Logout(gomock.Any(), "refresh-1").Return(nil)

		h := auth.NewHandler(testConfig(), svc)
		rec := do(h.Logout, "", &http.Cookie{Name: refreshCookieName, Value: "refresh-1"})
		assert.Equal(t, http.StatusNoContent, rec.Code)
		expectCookieCleared(t, rec)
	})

	t.Run("failure still clears cookie", func(t *testing.T) {
		svc := mock.NewMockService(gomock.NewController(t))
		svc.EXPECT().Logout(gomock.Any(), gomock.Any()).Return(errBoom)

		h := auth.NewHandler(testConfig(), svc)
		rec := do(h.Logout, "", &http.Cookie{Name: refreshCookieName, Value: "refresh-1"})
		expectError(t, rec, http.StatusInternalServerError, "INTERNAL_ERROR")
		expectCookieCleared(t, rec)
	})
}

// -----------------------------------------------------------------------------
// Request bodies and failures shared by the endpoints
// -----------------------------------------------------------------------------

func TestHandler_NullBody_IsInvalidBody(t *testing.T) {
	// No expectations: a body that is not an object must never reach the service.
	svc := mock.NewMockService(gomock.NewController(t))
	h := auth.NewHandler(testConfig(), svc)

	expectError(t, do(h.Register, `null`), http.StatusBadRequest, "INVALID_BODY")
	expectError(t, do(h.Login, `null`), http.StatusBadRequest, "INVALID_BODY")
}

func TestHandler_InternalErrorsDoNotLeakDetails(t *testing.T) {
	dbErr := errors.New(`pq: relation "users" does not exist`)
	cookie := &http.Cookie{Name: refreshCookieName, Value: "refresh-1"}

	tests := []struct {
		name    string
		handler func(*auth.Handler) http.HandlerFunc
		body    string
		cookies []*http.Cookie
		fail    func(svc *mock.MockService)
	}{
		{
			name:    "register",
			handler: func(h *auth.Handler) http.HandlerFunc { return h.Register },
			body:    `{"name":"Ana","email":"` + testEmail + `","password":"password123"}`,
			fail: func(s *mock.MockService) {
				s.EXPECT().Register(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, dbErr)
			},
		},
		{
			name:    "login",
			handler: func(h *auth.Handler) http.HandlerFunc { return h.Login },
			body:    `{"email":"` + testEmail + `","password":"password123"}`,
			fail: func(s *mock.MockService) {
				s.EXPECT().Login(gomock.Any(), testEmail, "password123").Return(nil, nil, dbErr)
			},
		},
		{
			name:    "refresh",
			handler: func(h *auth.Handler) http.HandlerFunc { return h.Refresh },
			cookies: []*http.Cookie{cookie},
			fail: func(s *mock.MockService) {
				s.EXPECT().Refresh(gomock.Any(), "refresh-1").Return(nil, nil, dbErr)
			},
		},
		{
			name:    "logout",
			handler: func(h *auth.Handler) http.HandlerFunc { return h.Logout },
			cookies: []*http.Cookie{cookie},
			fail: func(s *mock.MockService) {
				s.EXPECT().Logout(gomock.Any(), "refresh-1").Return(dbErr)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			tc.fail(svc)

			rec := do(tc.handler(auth.NewHandler(testConfig(), svc)), tc.body, tc.cookies...)
			expectError(t, rec, http.StatusInternalServerError, "INTERNAL_ERROR")
			assert.NotContains(t, rec.Body.String(), "pq:", "500 body exposes storage detail")
			assert.NotContains(t, rec.Body.String(), "relation", "500 body exposes storage detail")
		})
	}
}
