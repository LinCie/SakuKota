package reviews_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"rantau-go/internal/config"
	"rantau-go/internal/middleware"
	"rantau-go/internal/module/reviews"
	"rantau-go/internal/module/reviews/mock"
	"rantau-go/internal/pkg/jwt"
)

// testConfig signs and checks the access tokens in these tests.
var testConfig = config.Config{JWTSecret: "test-secret", JWTExpiry: time.Hour}

// newRouter serves the review handlers on the paths the API uses for them.
// Creating a review requires an access token, as it does in production.
func newRouter(svc reviews.Service) http.Handler {
	h := reviews.NewHandler(svc)
	r := chi.NewRouter()
	r.Route("/places", func(r chi.Router) {
		r.Route("/{id}/reviews", func(r chi.Router) {
			r.Get("/", h.GetReviews)
			r.With(middleware.WithAuth(testConfig)).Post("/", h.CreateReview)
		})
	})
	return r
}

// serve runs req through h. A panic fails the test instead of the whole binary.
func serve(t *testing.T, h http.Handler, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("handler panicked: %v", r)
		}
	}()

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func jsonRequest(method, target, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// authRequest is a JSON request carrying an access token for the user.
func authRequest(t *testing.T, method, target, body string, userID int64) *http.Request {
	t.Helper()
	token, err := jwt.GenerateToken(testConfig, userID, "user")
	require.NoError(t, err, "sign token")

	req := jsonRequest(method, target, body)
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// assertError checks the status and the { code, message } error envelope.
func assertError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	require.Equal(t, status, rec.Code, "body: %s", rec.Body)
	ct := rec.Header().Get("Content-Type")
	assert.True(t, strings.HasPrefix(ct, "application/json"), "Content-Type = %q, want application/json", ct)

	var body errorBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), "error body is not JSON; body: %s", rec.Body)
	assert.Equal(t, code, body.Code)
	assert.NotEmpty(t, body.Message, "message is empty")
}

// responseData returns the value under the success envelope's "data" key.
func responseData(t *testing.T, rec *httptest.ResponseRecorder) json.RawMessage {
	t.Helper()
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env), "response is not JSON; body: %s", rec.Body)
	return env.Data
}

// assertData checks the status and that the response data is the JSON document want.
func assertData(t *testing.T, rec *httptest.ResponseRecorder, status int, want string) {
	t.Helper()
	require.Equal(t, status, rec.Code, "body: %s", rec.Body)
	assert.JSONEq(t, want, string(responseData(t, rec)))
}

// linReview is a stored review with every field set.
func linReview() reviews.Review {
	return reviews.Review{
		ID:        1,
		PlaceID:   3,
		UserID:    7,
		Rating:    4,
		Comment:   new("Murah, porsi besar"),
		CreatedAt: time.Date(2026, 10, 4, 12, 30, 0, 0, time.UTC),
		User:      reviews.Author{ID: 7, Name: "Lin"},
	}
}

// linReviewJSON is linReview as the API returns it.
const linReviewJSON = `{
	"id": 1,
	"place_id": 3,
	"rating": 4,
	"comment": "Murah, porsi besar",
	"created_at": "2026-10-04T12:30:00Z",
	"user": {"id": 7, "name": "Lin"}
}`

func TestHandler_GetReviews(t *testing.T) {
	t.Run("rejects invalid place id", func(t *testing.T) {
		tests := []struct {
			name   string
			target string
		}{
			{name: "place id is not a number", target: "/places/abc/reviews"},
			{name: "place id is zero", target: "/places/0/reviews"},
			{name: "place id is negative", target: "/places/-1/reviews"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// No expectations: an invalid id must never reach the service.
				svc := mock.NewMockService(gomock.NewController(t))
				rec := serve(t, newRouter(svc), jsonRequest(http.MethodGet, tc.target, ""))
				assertError(t, rec, http.StatusBadRequest, "INVALID_PLACE_ID")
			})
		}
	})

	t.Run("maps service errors", func(t *testing.T) {
		tests := []struct {
			name       string
			err        error
			wantStatus int
			wantCode   string
		}{
			{name: "unknown place", err: reviews.ErrPlaceNotFound, wantStatus: http.StatusNotFound, wantCode: "PLACE_NOT_FOUND"},
			{name: "storage failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				svc := mock.NewMockService(gomock.NewController(t))
				svc.EXPECT().GetByPlace(gomock.Any(), gomock.Any()).Return(nil, tc.err)

				rec := serve(t, newRouter(svc), jsonRequest(http.MethodGet, "/places/3/reviews", ""))
				assertError(t, rec, tc.wantStatus, tc.wantCode)
			})
		}
	})

	t.Run("success", func(t *testing.T) {
		sari := reviews.Review{
			ID:        2,
			PlaceID:   3,
			UserID:    8,
			Rating:    5,
			CreatedAt: time.Date(2026, 10, 5, 8, 0, 0, 0, time.UTC),
			User:      reviews.Author{ID: 8, Name: "Sari"},
		}

		tests := []struct {
			name    string
			reviews []reviews.Review
			want    string
		}{
			{name: "no reviews", reviews: []reviews.Review{}, want: `[]`},
			{
				name:    "reviews in the order given",
				reviews: []reviews.Review{sari, linReview()},
				want: `[{
					"id": 2,
					"place_id": 3,
					"rating": 5,
					"comment": null,
					"created_at": "2026-10-05T08:00:00Z",
					"user": {"id": 8, "name": "Sari"}
				}, ` + linReviewJSON + `]`,
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				svc := mock.NewMockService(gomock.NewController(t))
				svc.EXPECT().GetByPlace(gomock.Any(), int64(3)).Return(tc.reviews, nil)

				rec := serve(t, newRouter(svc), jsonRequest(http.MethodGet, "/places/3/reviews", ""))
				assertData(t, rec, http.StatusOK, tc.want)
			})
		}
	})
}

func TestHandler_CreateReview(t *testing.T) {
	t.Run("rejects invalid place id", func(t *testing.T) {
		tests := []struct {
			name   string
			target string
		}{
			{name: "place id is not a number", target: "/places/abc/reviews"},
			{name: "place id is zero", target: "/places/0/reviews"},
			{name: "place id is negative", target: "/places/-1/reviews"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// No expectations: an invalid id must never reach the service.
				svc := mock.NewMockService(gomock.NewController(t))
				rec := serve(t, newRouter(svc), authRequest(t, http.MethodPost, tc.target, `{"rating":4}`, 7))
				assertError(t, rec, http.StatusBadRequest, "INVALID_PLACE_ID")
			})
		}
	})

	t.Run("rejects malformed bodies", func(t *testing.T) {
		tests := []struct {
			name string
			body string
		}{
			{name: "empty body", body: ``},
			{name: "body is null", body: `null`},
			{name: "malformed JSON", body: `{"rating":`},
			{name: "not JSON", body: `four stars`},
			{name: "array instead of object", body: `[4]`},
			{name: "rating as a string", body: `{"rating":"4"}`},
			{name: "fractional rating", body: `{"rating":4.5}`},
			{name: "comment as a number", body: `{"rating":4,"comment":5}`},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// No expectations: a malformed body must never reach the service.
				svc := mock.NewMockService(gomock.NewController(t))
				rec := serve(t, newRouter(svc), authRequest(t, http.MethodPost, "/places/3/reviews", tc.body, 7))
				assertError(t, rec, http.StatusBadRequest, "INVALID_BODY")
			})
		}
	})

	t.Run("without user", func(t *testing.T) {
		// No expectations: without WithAuth in front there is no user to
		// attribute the review to, so it must never reach the service.
		svc := mock.NewMockService(gomock.NewController(t))
		r := chi.NewRouter()
		r.Post("/places/{id}/reviews", reviews.NewHandler(svc).CreateReview)

		rec := serve(t, r, jsonRequest(http.MethodPost, "/places/3/reviews", `{"rating":4}`))
		assertError(t, rec, http.StatusUnauthorized, "UNAUTHORIZED")
	})

	t.Run("passes review to service", func(t *testing.T) {
		tests := []struct {
			name string
			body string
			want *reviews.Review
		}{
			{
				name: "rating and comment",
				body: `{"rating":4,"comment":"Murah"}`,
				want: &reviews.Review{PlaceID: 3, UserID: 7, Rating: 4, Comment: new("Murah")},
			},
			{
				name: "comment is passed on untrimmed",
				body: `{"rating":4,"comment":"  Murah  "}`,
				want: &reviews.Review{PlaceID: 3, UserID: 7, Rating: 4, Comment: new("  Murah  ")},
			},
			{
				name: "rating only",
				body: `{"rating":5}`,
				want: &reviews.Review{PlaceID: 3, UserID: 7, Rating: 5},
			},
			{
				name: "null comment",
				body: `{"rating":5,"comment":null}`,
				want: &reviews.Review{PlaceID: 3, UserID: 7, Rating: 5},
			},
			{
				name: "place comes from the path and author from the token",
				body: `{"rating":2,"id":99,"place_id":5,"user_id":1,"created_at":"yesterday"}`,
				want: &reviews.Review{PlaceID: 3, UserID: 7, Rating: 2},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				svc := mock.NewMockService(gomock.NewController(t))
				svc.EXPECT().Create(gomock.Any(), tc.want).Return(new(linReview()), nil)

				rec := serve(t, newRouter(svc), authRequest(t, http.MethodPost, "/places/3/reviews", tc.body, 7))
				require.Equal(t, http.StatusCreated, rec.Code, "body: %s", rec.Body)
			})
		}
	})

	t.Run("maps service errors", func(t *testing.T) {
		tests := []struct {
			name       string
			err        error
			wantStatus int
			wantCode   string
		}{
			{name: "rating out of range", err: reviews.ErrInvalidRating, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "comment too long", err: reviews.ErrCommentTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "unknown place", err: reviews.ErrPlaceNotFound, wantStatus: http.StatusNotFound, wantCode: "PLACE_NOT_FOUND"},
			{name: "already reviewed", err: reviews.ErrAlreadyReviewed, wantStatus: http.StatusConflict, wantCode: "ALREADY_REVIEWED"},
			{name: "storage failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				svc := mock.NewMockService(gomock.NewController(t))
				svc.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, tc.err)

				rec := serve(t, newRouter(svc), authRequest(t, http.MethodPost, "/places/3/reviews", `{"rating":4}`, 7))
				assertError(t, rec, tc.wantStatus, tc.wantCode)
			})
		}
	})

	t.Run("success", func(t *testing.T) {
		svc := mock.NewMockService(gomock.NewController(t))
		svc.EXPECT().Create(gomock.Any(), gomock.Any()).Return(new(linReview()), nil)

		rec := serve(t, newRouter(svc), authRequest(t, http.MethodPost, "/places/3/reviews",
			`{"rating":4,"comment":"Murah, porsi besar"}`, 7))
		assertData(t, rec, http.StatusCreated, linReviewJSON)
	})

	t.Run("context reaches service", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		svc := mock.NewMockService(gomock.NewController(t))
		svc.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
			func(got context.Context, _ *reviews.Review) (*reviews.Review, error) {
				assert.Error(t, got.Err(), "service was not given the request's context")
				return nil, got.Err()
			})

		req := authRequest(t, http.MethodPost, "/places/3/reviews", `{"rating":4}`, 7).WithContext(ctx)
		rec := serve(t, newRouter(svc), req)
		assert.NotEqual(t, http.StatusCreated, rec.Code, "status 201 for a request whose context had ended")
	})
}

func TestHandler_InternalErrorsDoNotLeakDetails(t *testing.T) {
	dbErr := errors.New(`pq: relation "place_reviews" does not exist`)

	tests := []struct {
		name   string
		method string
		target string
		body   string
		fail   func(svc *mock.MockService)
	}{
		{
			name:   "list reviews",
			method: http.MethodGet,
			target: "/places/3/reviews",
			fail: func(s *mock.MockService) {
				s.EXPECT().GetByPlace(gomock.Any(), int64(3)).Return(nil, dbErr)
			},
		},
		{
			name:   "create review",
			method: http.MethodPost,
			target: "/places/3/reviews",
			body:   `{"rating":4}`,
			fail: func(s *mock.MockService) {
				s.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, dbErr)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			tc.fail(svc)

			rec := serve(t, newRouter(svc), authRequest(t, tc.method, tc.target, tc.body, 7))
			require.Equal(t, http.StatusInternalServerError, rec.Code, "body: %s", rec.Body)
			assert.NotContains(t, rec.Body.String(), "pq:", "500 body exposes storage detail")
			assert.NotContains(t, rec.Body.String(), "relation", "500 body exposes storage detail")
		})
	}
}
