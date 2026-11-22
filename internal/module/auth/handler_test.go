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

	"rantau-go/internal/domain"
	"rantau-go/internal/module/auth"
	"rantau-go/internal/module/auth/mock"
)

const refreshCookieName = "refresh_token"

// refreshMaxAge is RefreshTokenExpiry from testConfig, in seconds.
var refreshMaxAge = int((24 * time.Hour).Seconds())

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

// errorResult is the status and the code of an error response.
type errorResult struct {
	Status int
	Code   string
}

// readError returns the status and the error envelope's code of rec.
func readError(t *testing.T, rec *httptest.ResponseRecorder) errorResult {
	t.Helper()
	return errorResult{Status: rec.Code, Code: decode(t, rec).Code}
}

// newHandler returns a Handler over a mock service that fails the test on any
// call without an expectation.
func newHandler(t *testing.T) (*auth.Handler, *mock.MockService) {
	t.Helper()
	svc := mock.NewMockService(gomock.NewController(t))
	return auth.NewHandler(testConfig(), svc), svc
}

func cookieNamed(rec *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// refreshCookieCleared reports whether the response expires the refresh cookie:
// it is sent with an empty value and a negative MaxAge.
func refreshCookieCleared(rec *httptest.ResponseRecorder) bool {
	c := cookieNamed(rec, refreshCookieName)
	return c != nil && c.Value == "" && c.MaxAge < 0
}

const registerBody = `{"name":"Ana","email":"ana@example.com","password":"password123"}`

func TestHandler_Register(t *testing.T) {
	t.Run("created", func(t *testing.T) {
		h, svc := newHandler(t)
		svc.EXPECT().Register(gomock.Any(), &domain.User{Name: "Ana", Email: testEmail}, "password123").Return(&domain.User{ID: 1, Name: "Ana", Email: testEmail}, nil)

		rec := do(h.Register, registerBody)
		require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())

		var got auth.UserResponse
		require.NoError(t, json.Unmarshal(decode(t, rec).Data, &got), "decode data")
		assert.EqualValues(t, 1, got.ID)
		assert.Equal(t, testEmail, got.Email)
		assert.NotContains(t, rec.Body.String(), "password", "response body mentions a password")
	})

	t.Run("invalid_json", func(t *testing.T) {
		// A malformed body must never reach the service: the mock has no expectations.
		h, _ := newHandler(t)

		rec := do(h.Register, `{"name":`)
		assert.Equal(t, errorResult{http.StatusBadRequest, "INVALID_BODY"}, readError(t, rec))
	})

	t.Run("error_mapping", func(t *testing.T) {
		tests := []struct {
			name   string
			err    error
			status int
			code   string
		}{
			{name: "invalid_name", err: domain.ErrUserNameRequired, status: http.StatusBadRequest, code: "VALIDATION_ERROR"},
			{name: "invalid_email", err: domain.ErrEmailInvalid, status: http.StatusBadRequest, code: "VALIDATION_ERROR"},
			{name: "weak_password", err: domain.ErrPasswordTooShort, status: http.StatusBadRequest, code: "VALIDATION_ERROR"},
			{name: "email_taken", err: domain.ErrEmailTaken, status: http.StatusConflict, code: "EMAIL_TAKEN"},
			{name: "unexpected_failure", err: errBoom, status: http.StatusInternalServerError, code: "INTERNAL_ERROR"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				h, svc := newHandler(t)
				svc.EXPECT().Register(gomock.Any(), &domain.User{Name: "Ana", Email: testEmail}, "password123").Return(nil, tc.err)

				rec := do(h.Register, registerBody)
				assert.Equal(t, errorResult{tc.status, tc.code}, readError(t, rec))
			})
		}
	})

	t.Run("internal_error_hides_details", func(t *testing.T) {
		// Storage details must not reach the client on a 500.
		h, svc := newHandler(t)
		svc.EXPECT().Register(gomock.Any(), &domain.User{Name: "Ana", Email: testEmail}, "password123").Return(nil, errors.New("pq: connection refused"))

		rec := do(h.Register, registerBody)
		assert.NotContains(t, decode(t, rec).Message, "connection refused", "500 message exposes the internal error")
	})
}

func TestHandler_Login(t *testing.T) {
	const loginBody = `{"email":"ana@example.com","password":"password123"}`

	t.Run("success", func(t *testing.T) {
		h, svc := newHandler(t)
		svc.EXPECT().Login(gomock.Any(), testEmail, "password123").Return(
			&domain.User{ID: 1, Name: "Ana", Email: testEmail, Role: "admin"},
			&auth.Tokens{AccessToken: "access-1", RefreshToken: "refresh-1"},
			nil)

		rec := do(h.Login, loginBody)
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

	t.Run("invalid_credentials", func(t *testing.T) {
		h, svc := newHandler(t)
		svc.EXPECT().Login(gomock.Any(), testEmail, "wrong").Return(nil, nil, domain.ErrInvalidCredentials)

		rec := do(h.Login, `{"email":"ana@example.com","password":"wrong"}`)
		assert.Equal(t, errorResult{http.StatusUnauthorized, "INVALID_CREDENTIALS"}, readError(t, rec))
		assert.Nil(t, cookieNamed(rec, refreshCookieName), "refresh cookie was set on failed login")
	})

	t.Run("unexpected_failure", func(t *testing.T) {
		h, svc := newHandler(t)
		svc.EXPECT().Login(gomock.Any(), testEmail, "password123").Return(nil, nil, errBoom)

		rec := do(h.Login, loginBody)
		assert.Equal(t, errorResult{http.StatusInternalServerError, "INTERNAL_ERROR"}, readError(t, rec))
	})

	t.Run("context_reaches_service", func(t *testing.T) {
		// The service must see the request's context: it is cancelled here, so a
		// service that ignores it would succeed.
		h, svc := newHandler(t)
		svc.EXPECT().Login(gomock.Any(), testEmail, "password123").DoAndReturn(
			func(got context.Context, _, _ string) (*domain.User, *auth.Tokens, error) {
				assert.Error(t, got.Err(), "service was not given the request's context")
				return nil, nil, got.Err()
			})

		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		req := httptest.NewRequest(http.MethodPost, "/v1/auth/login",
			strings.NewReader(`{"email":"`+testEmail+`","password":"password123"}`)).WithContext(ctx)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.Login(rec, req)

		assert.NotEqual(t, http.StatusOK, rec.Code, "status for a request whose context had ended")
	})
}

func TestHandler_Refresh(t *testing.T) {
	t.Run("missing_cookie", func(t *testing.T) {
		// Without a cookie the service must not be called: the mock has no expectations.
		h, _ := newHandler(t)

		rec := do(h.Refresh, "")
		assert.Equal(t, errorResult{http.StatusUnauthorized, "INVALID_REFRESH_TOKEN"}, readError(t, rec))
		assert.True(t, refreshCookieCleared(rec), "refresh cookie was not cleared")
	})

	t.Run("success", func(t *testing.T) {
		h, svc := newHandler(t)
		svc.EXPECT().Refresh(gomock.Any(), "refresh-1").Return(
			&domain.User{ID: 1, Name: "Ana", Email: testEmail},
			&auth.Tokens{AccessToken: "access-2", RefreshToken: "refresh-2"},
			nil)

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

	t.Run("invalid_token_clears_cookie", func(t *testing.T) {
		h, svc := newHandler(t)
		svc.EXPECT().Refresh(gomock.Any(), "stale").Return(nil, nil, domain.ErrInvalidRefreshToken)

		rec := do(h.Refresh, "", &http.Cookie{Name: refreshCookieName, Value: "stale"})
		assert.Equal(t, errorResult{http.StatusUnauthorized, "INVALID_REFRESH_TOKEN"}, readError(t, rec))
		assert.True(t, refreshCookieCleared(rec), "refresh cookie was not cleared")
	})

	t.Run("unexpected_failure", func(t *testing.T) {
		h, svc := newHandler(t)
		svc.EXPECT().Refresh(gomock.Any(), "refresh-1").Return(nil, nil, errBoom)

		rec := do(h.Refresh, "", &http.Cookie{Name: refreshCookieName, Value: "refresh-1"})
		assert.Equal(t, errorResult{http.StatusInternalServerError, "INTERNAL_ERROR"}, readError(t, rec))
	})
}

func TestHandler_Logout(t *testing.T) {
	t.Run("without_cookie", func(t *testing.T) {
		// Without a cookie there is no session to revoke: the mock has no expectations.
		h, _ := newHandler(t)

		rec := do(h.Logout, "")
		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.True(t, refreshCookieCleared(rec), "refresh cookie was not cleared")
	})

	t.Run("revokes_cookie_token", func(t *testing.T) {
		h, svc := newHandler(t)
		svc.EXPECT().Logout(gomock.Any(), "refresh-1").Return(nil)

		rec := do(h.Logout, "", &http.Cookie{Name: refreshCookieName, Value: "refresh-1"})
		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.True(t, refreshCookieCleared(rec), "refresh cookie was not cleared")
	})

	t.Run("failure_still_clears_cookie", func(t *testing.T) {
		h, svc := newHandler(t)
		svc.EXPECT().Logout(gomock.Any(), "refresh-1").Return(errBoom)

		rec := do(h.Logout, "", &http.Cookie{Name: refreshCookieName, Value: "refresh-1"})
		assert.Equal(t, errorResult{http.StatusInternalServerError, "INTERNAL_ERROR"}, readError(t, rec))
		assert.True(t, refreshCookieCleared(rec), "refresh cookie was not cleared")
	})
}

func TestHandler_NullBody_IsInvalidBody(t *testing.T) {
	// A body that is not an object must never reach the service: the mock has no expectations.
	h, _ := newHandler(t)

	assert.Equal(t, errorResult{http.StatusBadRequest, "INVALID_BODY"}, readError(t, do(h.Register, `null`)))
	assert.Equal(t, errorResult{http.StatusBadRequest, "INVALID_BODY"}, readError(t, do(h.Login, `null`)))
}

func TestHandler_InternalErrorsDoNotLeakDetails(t *testing.T) {
	dbErr := errors.New(`pq: relation "users" does not exist`)
	cookie := &http.Cookie{Name: refreshCookieName, Value: "refresh-1"}

	tests := []struct {
		name    string
		handler func(*auth.Handler) http.HandlerFunc
		body    string
		cookies []*http.Cookie
		expect  func(svc *mock.MockService)
	}{
		{
			name:    "register",
			handler: func(h *auth.Handler) http.HandlerFunc { return h.Register },
			body:    `{"name":"Ana","email":"` + testEmail + `","password":"password123"}`,
			expect: func(svc *mock.MockService) {
				svc.EXPECT().Register(gomock.Any(), &domain.User{Name: "Ana", Email: testEmail}, "password123").Return(nil, dbErr)
			},
		},
		{
			name:    "login",
			handler: func(h *auth.Handler) http.HandlerFunc { return h.Login },
			body:    `{"email":"` + testEmail + `","password":"password123"}`,
			expect: func(svc *mock.MockService) {
				svc.EXPECT().Login(gomock.Any(), testEmail, "password123").Return(nil, nil, dbErr)
			},
		},
		{
			name:    "refresh",
			handler: func(h *auth.Handler) http.HandlerFunc { return h.Refresh },
			cookies: []*http.Cookie{cookie},
			expect: func(svc *mock.MockService) {
				svc.EXPECT().Refresh(gomock.Any(), "refresh-1").Return(nil, nil, dbErr)
			},
		},
		{
			name:    "logout",
			handler: func(h *auth.Handler) http.HandlerFunc { return h.Logout },
			cookies: []*http.Cookie{cookie},
			expect: func(svc *mock.MockService) {
				svc.EXPECT().Logout(gomock.Any(), "refresh-1").Return(dbErr)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, svc := newHandler(t)
			tc.expect(svc)

			rec := do(tc.handler(h), tc.body, tc.cookies...)
			assert.Equal(t, errorResult{http.StatusInternalServerError, "INTERNAL_ERROR"}, readError(t, rec))
			for _, leak := range []string{"pq:", "relation"} {
				assert.NotContains(t, rec.Body.String(), leak, "500 body exposes storage detail")
			}
		})
	}
}
