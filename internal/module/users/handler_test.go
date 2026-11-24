package users_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"sakukota-go/internal/config"
	"sakukota-go/internal/domain"
	"sakukota-go/internal/middleware"
	"sakukota-go/internal/module/users"
	"sakukota-go/internal/module/users/mock"
	"sakukota-go/internal/pkg/jwt"
	"sakukota-go/internal/storage"
)

// testConfig signs and checks the access tokens in these tests.
var testConfig = config.Config{JWTSecret: "test-secret", JWTExpiry: time.Hour}

// newRouter serves the profile handlers on the paths the API uses for them,
// behind WithAuth as in production.
func newRouter(svc *mock.MockService) http.Handler {
	h := users.NewHandler(svc, mediaURL)
	r := chi.NewRouter()
	r.Route("/users/me", func(r chi.Router) {
		r.Use(middleware.WithAuth(testConfig))
		r.Get("/", h.Show)
		r.Put("/", h.Rename)
		r.Post("/avatar/upload-url", h.CreateAvatarUpload)
		r.Put("/avatar", h.SetAvatar)
	})
	return r
}

// mediaURL builds media links as the public storage URL would.
func mediaURL(key string) string { return "https://cdn.test/" + key }

// request sends a JSON request with an access token for user 7 through h.
func request(t *testing.T, h http.Handler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	token, err := jwt.GenerateToken(testConfig, 7, "user")
	require.NoError(t, err, "sign token")

	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func newService(t *testing.T) *mock.MockService {
	t.Helper()
	return mock.NewMockService(gomock.NewController(t))
}

// errorCode returns the code of the error envelope in rec.
func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), "error body is not JSON: %s", rec.Body)
	return body.Code
}

// responseData returns the JSON under the success envelope's "data" key.
func responseData(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env), "response is not JSON: %s", rec.Body)
	return string(env.Data)
}

// ana is user 7 with a profile photo.
func ana() *domain.User {
	return &domain.User{
		ID:           7,
		Name:         "Ana",
		Email:        "ana@example.com",
		PasswordHash: "secret-hash",
		Role:         "user",
		CreatedAt:    time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC),
		AvatarKey:    new("avatars/7/a.webp"),
	}
}

// anaJSON is ana as the API returns it, without password material.
const anaJSON = `{
	"id": 7,
	"name": "Ana",
	"email": "ana@example.com",
	"role": "user",
	"avatar_url": "https://cdn.test/avatars/7/a.webp",
	"created_at": "2026-10-04T12:00:00Z",
	"updated_at": "2026-10-05T12:00:00Z"
}`

func TestHandler_Show(t *testing.T) {
	t.Run("WithAvatar", func(t *testing.T) {
		svc := newService(t)
		svc.EXPECT().Fetch(gomock.Any(), int64(7)).Return(ana(), nil)

		rec := request(t, newRouter(svc), http.MethodGet, "/users/me", "")
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body)
		assert.JSONEq(t, anaJSON, responseData(t, rec))
	})

	t.Run("WithoutAvatar", func(t *testing.T) {
		u := ana()
		u.AvatarKey = nil
		svc := newService(t)
		svc.EXPECT().Fetch(gomock.Any(), int64(7)).Return(u, nil)

		rec := request(t, newRouter(svc), http.MethodGet, "/users/me", "")
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body)
		var got struct {
			AvatarURL *string `json:"avatar_url"`
		}
		require.NoError(t, json.Unmarshal([]byte(responseData(t, rec)), &got))
		assert.Nil(t, got.AvatarURL)
	})
}

func TestHandler_Rename(t *testing.T) {
	svc := newService(t)
	svc.EXPECT().Rename(gomock.Any(), int64(7), " Ana ").Return(ana(), nil)

	rec := request(t, newRouter(svc), http.MethodPut, "/users/me", `{"name":" Ana ","email":"x@example.com","role":"admin"}`)
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body)
	assert.JSONEq(t, anaJSON, responseData(t, rec))
}

func TestHandler_CreateAvatarUpload(t *testing.T) {
	svc := newService(t)
	svc.EXPECT().AvatarUploadURL(gomock.Any(), int64(7), "image/webp").Return(storage.Upload{
		Key:       "uploads/avatars/7/a.webp",
		URL:       "https://storage.test/signed",
		Method:    http.MethodPut,
		Headers:   map[string]string{"Content-Type": "image/webp"},
		ExpiresAt: time.Date(2026, 10, 4, 12, 35, 0, 0, time.UTC),
	}, nil)

	rec := request(t, newRouter(svc), http.MethodPost, "/users/me/avatar/upload-url", `{"content_type":"image/webp"}`)
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body)
	assert.JSONEq(t, `{
		"upload_key": "uploads/avatars/7/a.webp",
		"upload_url": "https://storage.test/signed",
		"method": "PUT",
		"headers": {"Content-Type": "image/webp"},
		"expires_at": "2026-10-04T12:35:00Z"
	}`, responseData(t, rec))
}

func TestHandler_SetAvatar(t *testing.T) {
	svc := newService(t)
	svc.EXPECT().SetAvatar(gomock.Any(), int64(7), "uploads/avatars/7/a.webp").Return(ana(), nil)

	rec := request(t, newRouter(svc), http.MethodPut, "/users/me/avatar", `{"upload_key":"uploads/avatars/7/a.webp"}`)
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body)
	assert.JSONEq(t, anaJSON, responseData(t, rec))
}

func TestHandler_RejectsInvalidBodies(t *testing.T) {
	tests := []struct {
		name   string
		method string
		target string
		body   string
	}{
		{name: "RenameEmptyBody", method: http.MethodPut, target: "/users/me", body: ``},
		{name: "RenameNameAsANumber", method: http.MethodPut, target: "/users/me", body: `{"name":5}`},
		{name: "UploadURLEmptyBody", method: http.MethodPost, target: "/users/me/avatar/upload-url", body: ``},
		{name: "UploadURLContentTypeAsANumber", method: http.MethodPost, target: "/users/me/avatar/upload-url", body: `{"content_type":5}`},
		{name: "SetAvatarEmptyBody", method: http.MethodPut, target: "/users/me/avatar", body: ``},
		{name: "SetAvatarUploadKeyAsANumber", method: http.MethodPut, target: "/users/me/avatar", body: `{"upload_key":5}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// An invalid body must never reach the service: the mock has no expectations.
			rec := request(t, newRouter(newService(t)), tc.method, tc.target, tc.body)
			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Equal(t, "INVALID_BODY", errorCode(t, rec))
		})
	}
}

func TestHandler_RequiresAUser(t *testing.T) {
	h := users.NewHandler(newService(t), mediaURL)
	// Without WithAuth in front there is no user, so nothing reaches the service.
	r := chi.NewRouter()
	r.Get("/users/me", h.Show)
	r.Put("/users/me", h.Rename)
	r.Post("/users/me/avatar/upload-url", h.CreateAvatarUpload)
	r.Put("/users/me/avatar", h.SetAvatar)

	for _, req := range []*http.Request{
		httptest.NewRequest(http.MethodGet, "/users/me", nil),
		httptest.NewRequest(http.MethodPut, "/users/me", strings.NewReader(`{"name":"Ana"}`)),
		httptest.NewRequest(http.MethodPost, "/users/me/avatar/upload-url", strings.NewReader(`{"content_type":"image/png"}`)),
		httptest.NewRequest(http.MethodPut, "/users/me/avatar", strings.NewReader(`{"upload_key":"k"}`)),
	} {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code, "%s %s", req.Method, req.URL)
		assert.Equal(t, "UNAUTHORIZED", errorCode(t, rec), "%s %s", req.Method, req.URL)
	}
}

func TestHandler_MapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "UserDeleted", err: domain.ErrUserNotFound, wantStatus: http.StatusNotFound, wantCode: "USER_NOT_FOUND"},
		{name: "NameRequired", err: domain.ErrUserNameRequired, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "NameTooLong", err: domain.ErrUserNameTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "NotAnImage", err: domain.ErrAvatarNotImage, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "UnsupportedType", err: storage.ErrInvalidMediaType, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "NotUploaded", err: storage.ErrMediaNotUploaded, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "TooLarge", err: storage.ErrImageTooLarge, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "InvalidUploadKey", err: storage.ErrInvalidUploadKey, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "StorageFailure", err: errBoom, wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
	}

	routes := []struct {
		method, target, body string
		expect               func(svc *mock.MockService, err error)
	}{
		{http.MethodGet, "/users/me", "", func(svc *mock.MockService, err error) {
			svc.EXPECT().Fetch(gomock.Any(), int64(7)).Return(nil, err)
		}},
		{http.MethodPut, "/users/me", `{"name":"Ana"}`, func(svc *mock.MockService, err error) {
			svc.EXPECT().Rename(gomock.Any(), int64(7), "Ana").Return(nil, err)
		}},
		{http.MethodPost, "/users/me/avatar/upload-url", `{"content_type":"image/png"}`, func(svc *mock.MockService, err error) {
			svc.EXPECT().AvatarUploadURL(gomock.Any(), int64(7), "image/png").Return(storage.Upload{}, err)
		}},
		{http.MethodPut, "/users/me/avatar", `{"upload_key":"k"}`, func(svc *mock.MockService, err error) {
			svc.EXPECT().SetAvatar(gomock.Any(), int64(7), "k").Return(nil, err)
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, route := range routes {
				svc := newService(t)
				route.expect(svc, tc.err)

				rec := request(t, newRouter(svc), route.method, route.target, route.body)
				assert.Equal(t, tc.wantStatus, rec.Code, "%s %s", route.method, route.target)
				assert.Equal(t, tc.wantCode, errorCode(t, rec), "%s %s", route.method, route.target)
				assert.NotContains(t, rec.Body.String(), "boom", "%s %s leaks the internal error", route.method, route.target)
			}
		})
	}
}
