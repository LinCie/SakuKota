package router_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"sakukota-go/internal/config"
	"sakukota-go/internal/domain"
	"sakukota-go/internal/pkg/httpx"
	"sakukota-go/internal/pkg/jwt"
	"sakukota-go/internal/router"
)

// access is who may call a route.
type access int

const (
	// public routes need no token.
	public access = iota
	// authenticated routes need any valid token.
	authenticated
	// adminOnly routes need a valid token with the admin role.
	adminOnly
)

// routeAccess lists every route of the API and who may call it. The test
// fails when the router has a route that is missing here, or the reverse, so
// a new route cannot ship without deciding its access. Keys are the method and
// the route pattern as chi reports it.
var routeAccess = map[string]access{
	"GET /docs":         public,
	"GET /openapi.json": public,

	"POST /v1/auth/register": public,
	"POST /v1/auth/login":    public,
	"POST /v1/auth/refresh":  public,
	"POST /v1/auth/logout":   public,

	"GET /v1/places/":                                      public,
	"GET /v1/places/{id}":                                  public,
	"GET /v1/places/categories/":                           public,
	"GET /v1/places/categories/{id}":                       public,
	"GET /v1/places/{id}/reviews/":                         public,
	"POST /v1/places/{id}/reviews/":                        authenticated,
	"DELETE /v1/places/{id}/reviews/mine":                  authenticated,
	"POST /v1/places/{id}/reviews/mine/medias/upload-url":  authenticated,
	"POST /v1/places/{id}/reviews/mine/medias":             authenticated,
	"DELETE /v1/places/{id}/reviews/mine/medias/{mediaId}": authenticated,
	"DELETE /v1/places/{id}/reviews/{reviewId}":            adminOnly,
	"POST /v1/places/{id}/reviews/{reviewId}/reports":      authenticated,
	"POST /v1/places/{id}/reports":                         authenticated,
	"POST /v1/places/":                                     adminOnly,
	"PUT /v1/places/{id}":                                  adminOnly,
	"DELETE /v1/places/{id}":                               adminOnly,
	"POST /v1/places/{id}/medias/upload-url":               adminOnly,
	"POST /v1/places/{id}/medias":                          adminOnly,
	"DELETE /v1/places/{id}/medias/{mediaId}":              adminOnly,

	"POST /v1/suggestions/":             authenticated,
	"GET /v1/suggestions/mine":          authenticated,
	"GET /v1/suggestions/":              adminOnly,
	"GET /v1/suggestions/{id}":          adminOnly,
	"POST /v1/suggestions/{id}/approve": adminOnly,
	"POST /v1/suggestions/{id}/reject":  adminOnly,

	"GET /v1/reports/":              adminOnly,
	"GET /v1/reports/{id}":          adminOnly,
	"POST /v1/reports/{id}/remove":  adminOnly,
	"POST /v1/reports/{id}/dismiss": adminOnly,

	"GET /v1/subdistricts/":                         public,
	"GET /v1/subdistricts/{id}":                     public,
	"POST /v1/subdistricts/":                        adminOnly,
	"PUT /v1/subdistricts/{id}":                     adminOnly,
	"DELETE /v1/subdistricts/{id}":                  adminOnly,
	"POST /v1/subdistricts/{id}/medias/upload-url":  adminOnly,
	"POST /v1/subdistricts/{id}/medias":             adminOnly,
	"DELETE /v1/subdistricts/{id}/medias/{mediaId}": adminOnly,
	"PUT /v1/subdistricts/{id}/boundary":            adminOnly,
}

func testConfig() config.Config {
	const endpoint = "http://127.0.0.1:1"

	return config.Config{
		JWTSecret:          "0123456789abcdef0123456789abcdef",
		JWTExpiry:          5 * time.Minute,
		RefreshTokenExpiry: time.Hour,
		Storage: config.StorageConfig{
			Endpoint:        endpoint,
			PublicEndpoint:  endpoint,
			PublicURL:       endpoint + "/images",
			Region:          "garage",
			Bucket:          "test",
			AccessKeyID:     "key",
			SecretAccessKey: "secret",
		},
	}
}

// newRouter builds the real router. Its database refuses every connection, so
// a request that gets past authentication ends in a 500 from the handler,
// which is all these tests need: they check who is turned away, not what the
// handlers do.
func newRouter(t *testing.T) http.Handler {
	t.Helper()

	raw, err := sql.Open("pgx", "postgres://nobody@127.0.0.1:1/none?sslmode=disable&connect_timeout=1")
	require.NoError(t, err, "open database")
	db := sqlx.NewDb(raw, "pgx")
	t.Cleanup(func() { _ = db.Close() })

	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	t.Cleanup(func() { _ = rdb.Close() })

	return router.NewRouter(db, rdb, testConfig())
}

func token(t *testing.T, role string) string {
	t.Helper()

	signed, err := jwt.GenerateToken(testConfig(), 1, role)
	require.NoError(t, err, "generate token")
	return signed
}

// call sends the route's request, with the bearer token when it is not empty.
func call(h http.Handler, key, bearer string) *httptest.ResponseRecorder {
	method, pattern, _ := strings.Cut(key, " ")
	path := strings.ReplaceAll(pattern, "{id}", "1")
	if path != "/" {
		path = strings.TrimSuffix(path, "/")
	}

	req := httptest.NewRequest(method, path, strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func errorCode(rec *httptest.ResponseRecorder) string {
	var body struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return body.Code
}

// refusal is the status and error code of a response.
type refusal struct {
	Status int
	Code   string
}

// refusalOf returns the response's status and error code. A middleware
// refusal is told from a handler answering 401 (a bad refresh token, say) by
// the code.
func refusalOf(rec *httptest.ResponseRecorder) refusal {
	return refusal{Status: rec.Code, Code: errorCode(rec)}
}

// isTurnedAway reports whether the response is an authentication or
// authorization refusal.
func isTurnedAway(rec *httptest.ResponseRecorder) bool {
	return rec.Code == http.StatusForbidden ||
		(rec.Code == http.StatusUnauthorized && errorCode(rec) == httpx.CodeUnauthorized)
}

func TestRouter_EveryRouteHasADeclaredAccess(t *testing.T) {
	var registered []string
	err := chi.Walk(newRouter(t).(chi.Routes), func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		registered = append(registered, method+" "+route)
		return nil
	})
	require.NoError(t, err, "walk routes")

	declared := make([]string, 0, len(routeAccess))
	for key := range routeAccess {
		declared = append(declared, key)
	}
	slices.Sort(registered)
	slices.Sort(declared)

	assert.Equal(t, declared, registered, "routes registered in the router differ from routeAccess")
}

func TestRouter_Access(t *testing.T) {
	h := newRouter(t)
	user, admin := token(t, "user"), token(t, domain.RoleAdmin)

	for key, level := range routeAccess {
		t.Run(key, func(t *testing.T) {
			unauthorized := refusal{Status: http.StatusUnauthorized, Code: httpx.CodeUnauthorized}
			forbidden := refusal{Status: http.StatusForbidden, Code: httpx.CodeForbidden}
			letThrough := func(rec *httptest.ResponseRecorder, who string) {
				assert.False(t, isTurnedAway(rec), "%s was turned away: %d %s", who, rec.Code, rec.Body)
			}

			switch level {
			case public:
				letThrough(call(h, key, ""), "anonymous caller")

			case authenticated:
				assert.Equal(t, unauthorized, refusalOf(call(h, key, "")))
				assert.Equal(t, unauthorized, refusalOf(call(h, key, "garbage")))
				letThrough(call(h, key, user), "user")
				letThrough(call(h, key, admin), "admin")

			case adminOnly:
				assert.Equal(t, unauthorized, refusalOf(call(h, key, "")))
				assert.Equal(t, unauthorized, refusalOf(call(h, key, "garbage")))
				assert.Equal(t, forbidden, refusalOf(call(h, key, user)))
				letThrough(call(h, key, admin), "admin")
			}
		})
	}
}

func TestRouter_SecurityHeaders(t *testing.T) {
	h := newRouter(t)
	tests := []struct {
		name    string
		target  string
		wantCSP string
	}{
		{name: "api", target: "/v1/nowhere", wantCSP: "default-src 'none'; frame-ancestors 'none'"},
		{name: "spec", target: "/openapi.json", wantCSP: "default-src 'none'; frame-ancestors 'none'"},
		{name: "docs-page", target: "/docs", wantCSP: "frame-ancestors 'none'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()

			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.target, nil))

			assert.Equal(t, tt.wantCSP, rec.Header().Get("Content-Security-Policy"))
			assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
			assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
		})
	}
}
