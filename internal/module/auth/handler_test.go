package auth_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("response is not JSON: %v; body=%s", err, rec.Body.String())
	}
	return env
}

func expectError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if rec.Code != status {
		t.Errorf("status = %d, want %d; body=%s", rec.Code, status, rec.Body.String())
	}
	if env := decode(t, rec); env.Code != code {
		t.Errorf("code = %q, want %q", env.Code, code)
	}
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
	if c == nil {
		t.Fatal("refresh cookie was not sent to clear it")
	}
	if c.Value != "" || c.MaxAge >= 0 {
		t.Errorf("refresh cookie not cleared: value=%q maxAge=%d", c.Value, c.MaxAge)
	}
}

// -----------------------------------------------------------------------------
// Register
// -----------------------------------------------------------------------------

func TestHandler_Register_Created(t *testing.T) {
	svc := mock.NewMockService(gomock.NewController(t))
	svc.EXPECT().Register(gomock.Any(), &users.User{Name: "Ana", Email: testEmail}, "password123").
		Return(&users.User{ID: 1, Name: "Ana", Email: testEmail}, nil)

	h := auth.NewHandler(testConfig(), svc)
	rec := do(h.Register, `{"name":"Ana","email":"ana@example.com","password":"password123"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var got auth.UserResponse
	if err := json.Unmarshal(decode(t, rec).Data, &got); err != nil {
		t.Fatalf("decode data: %v", err)
	}
	if got.ID != 1 || got.Email != testEmail {
		t.Errorf("data = %+v, want the registered user", got)
	}
	if strings.Contains(rec.Body.String(), "password") {
		t.Error("response body mentions a password")
	}
}

func TestHandler_Register_InvalidJSON(t *testing.T) {
	// No expectations: a malformed body must never reach the service.
	svc := mock.NewMockService(gomock.NewController(t))
	h := auth.NewHandler(testConfig(), svc)

	rec := do(h.Register, `{"name":`)
	expectError(t, rec, http.StatusBadRequest, "INVALID_BODY")
}

func TestHandler_Register_ErrorMapping(t *testing.T) {
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
}

func TestHandler_Register_InternalErrorHidesDetails(t *testing.T) {
	// Storage details must not reach the client on a 500.
	svc := mock.NewMockService(gomock.NewController(t))
	svc.EXPECT().Register(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errors.New("pq: connection refused"))

	h := auth.NewHandler(testConfig(), svc)
	rec := do(h.Register, `{"name":"Ana","email":"ana@example.com","password":"password123"}`)
	if env := decode(t, rec); strings.Contains(env.Message, "connection refused") {
		t.Errorf("500 message exposes internal error: %q", env.Message)
	}
}

// -----------------------------------------------------------------------------
// Login
// -----------------------------------------------------------------------------

func TestHandler_Login_Success(t *testing.T) {
	svc := mock.NewMockService(gomock.NewController(t))
	svc.EXPECT().Login(gomock.Any(), testEmail, "password123").Return(
		&users.User{ID: 1, Name: "Ana", Email: testEmail},
		&auth.Tokens{AccessToken: "access-1", RefreshToken: "refresh-1"},
		nil,
	)

	h := auth.NewHandler(testConfig(), svc)
	rec := do(h.Login, `{"email":"ana@example.com","password":"password123"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var session auth.SessionResponse
	if err := json.Unmarshal(decode(t, rec).Data, &session); err != nil {
		t.Fatalf("decode data: %v", err)
	}
	if session.AccessToken != "access-1" {
		t.Errorf("access_token = %q, want %q", session.AccessToken, "access-1")
	}
	if session.User.Email != testEmail {
		t.Errorf("user.email = %q, want %q", session.User.Email, testEmail)
	}
	if strings.Contains(rec.Body.String(), "refresh-1") {
		t.Error("refresh token appears in the response body")
	}

	c := cookieNamed(rec, refreshCookieName)
	if c == nil {
		t.Fatal("refresh cookie was not set")
	}
	if c.Value != "refresh-1" {
		t.Errorf("cookie value = %q, want %q", c.Value, "refresh-1")
	}
	if !c.HttpOnly {
		t.Error("refresh cookie is not HttpOnly")
	}
	if c.Path != "/v1/auth" {
		t.Errorf("cookie path = %q, want %q", c.Path, "/v1/auth")
	}
	if c.SameSite != http.SameSiteStrictMode {
		t.Errorf("cookie SameSite = %v, want Strict", c.SameSite)
	}
	if c.Secure != testConfig().CookieSecure {
		t.Errorf("cookie Secure = %v, want %v", c.Secure, testConfig().CookieSecure)
	}
	if c.MaxAge != refreshMaxAge {
		t.Errorf("cookie MaxAge = %d, want %d", c.MaxAge, refreshMaxAge)
	}
}

func TestHandler_Login_InvalidCredentials(t *testing.T) {
	svc := mock.NewMockService(gomock.NewController(t))
	svc.EXPECT().Login(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil, auth.ErrInvalidCredentials)

	h := auth.NewHandler(testConfig(), svc)
	rec := do(h.Login, `{"email":"ana@example.com","password":"wrong"}`)
	expectError(t, rec, http.StatusUnauthorized, "INVALID_CREDENTIALS")
	if cookieNamed(rec, refreshCookieName) != nil {
		t.Error("refresh cookie was set on failed login")
	}
}

func TestHandler_Login_UnexpectedFailure(t *testing.T) {
	svc := mock.NewMockService(gomock.NewController(t))
	svc.EXPECT().Login(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil, errBoom)

	h := auth.NewHandler(testConfig(), svc)
	rec := do(h.Login, `{"email":"ana@example.com","password":"password123"}`)
	expectError(t, rec, http.StatusInternalServerError, "INTERNAL_ERROR")
}

// -----------------------------------------------------------------------------
// Refresh
// -----------------------------------------------------------------------------

func TestHandler_Refresh_MissingCookie(t *testing.T) {
	// No expectations: without a cookie the service must not be called.
	svc := mock.NewMockService(gomock.NewController(t))
	h := auth.NewHandler(testConfig(), svc)

	rec := do(h.Refresh, "")
	expectError(t, rec, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN")
}

func TestHandler_Refresh_Success(t *testing.T) {
	svc := mock.NewMockService(gomock.NewController(t))
	svc.EXPECT().Refresh(gomock.Any(), "refresh-1").Return(
		&users.User{ID: 1, Name: "Ana", Email: testEmail},
		&auth.Tokens{AccessToken: "access-2", RefreshToken: "refresh-2"},
		nil,
	)

	h := auth.NewHandler(testConfig(), svc)
	rec := do(h.Refresh, "", &http.Cookie{Name: refreshCookieName, Value: "refresh-1"})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var session auth.SessionResponse
	if err := json.Unmarshal(decode(t, rec).Data, &session); err != nil {
		t.Fatalf("decode data: %v", err)
	}
	if session.AccessToken != "access-2" {
		t.Errorf("access_token = %q, want %q", session.AccessToken, "access-2")
	}

	c := cookieNamed(rec, refreshCookieName)
	if c == nil || c.Value != "refresh-2" {
		t.Fatalf("replacement refresh cookie = %+v, want value %q", c, "refresh-2")
	}
	if c.MaxAge != refreshMaxAge {
		t.Errorf("cookie MaxAge = %d, want %d", c.MaxAge, refreshMaxAge)
	}
}

func TestHandler_Refresh_InvalidTokenClearsCookie(t *testing.T) {
	svc := mock.NewMockService(gomock.NewController(t))
	svc.EXPECT().Refresh(gomock.Any(), "stale").Return(nil, nil, auth.ErrInvalidRefreshToken)

	h := auth.NewHandler(testConfig(), svc)
	rec := do(h.Refresh, "", &http.Cookie{Name: refreshCookieName, Value: "stale"})
	expectError(t, rec, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN")
	expectCookieCleared(t, rec)
}

func TestHandler_Refresh_UnexpectedFailure(t *testing.T) {
	svc := mock.NewMockService(gomock.NewController(t))
	svc.EXPECT().Refresh(gomock.Any(), gomock.Any()).Return(nil, nil, errBoom)

	h := auth.NewHandler(testConfig(), svc)
	rec := do(h.Refresh, "", &http.Cookie{Name: refreshCookieName, Value: "refresh-1"})
	expectError(t, rec, http.StatusInternalServerError, "INTERNAL_ERROR")
}

// -----------------------------------------------------------------------------
// Logout
// -----------------------------------------------------------------------------

func TestHandler_Logout_WithoutCookie(t *testing.T) {
	// No expectations: without a cookie there is no session to revoke.
	svc := mock.NewMockService(gomock.NewController(t))
	h := auth.NewHandler(testConfig(), svc)

	rec := do(h.Logout, "")
	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	expectCookieCleared(t, rec)
}

func TestHandler_Logout_RevokesCookieToken(t *testing.T) {
	svc := mock.NewMockService(gomock.NewController(t))
	svc.EXPECT().Logout(gomock.Any(), "refresh-1").Return(nil)

	h := auth.NewHandler(testConfig(), svc)
	rec := do(h.Logout, "", &http.Cookie{Name: refreshCookieName, Value: "refresh-1"})
	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	expectCookieCleared(t, rec)
}

func TestHandler_Logout_FailureStillClearsCookie(t *testing.T) {
	svc := mock.NewMockService(gomock.NewController(t))
	svc.EXPECT().Logout(gomock.Any(), gomock.Any()).Return(errBoom)

	h := auth.NewHandler(testConfig(), svc)
	rec := do(h.Logout, "", &http.Cookie{Name: refreshCookieName, Value: "refresh-1"})
	expectError(t, rec, http.StatusInternalServerError, "INTERNAL_ERROR")
	expectCookieCleared(t, rec)
}
