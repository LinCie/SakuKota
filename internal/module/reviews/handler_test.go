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
	"rantau-go/internal/domain"
	"rantau-go/internal/middleware"
	"rantau-go/internal/module/reviews"
	"rantau-go/internal/module/reviews/mock"
	"rantau-go/internal/pkg/jwt"
)

// testConfig signs and checks the access tokens in these tests.
var testConfig = config.Config{JWTSecret: "test-secret", JWTExpiry: time.Hour}

// newRouter serves the review handlers on the paths the API uses for them.
// Creating a review requires an access token, as it does in production.
func newRouter(svc *mock.MockreviewService) http.Handler {
	h := reviews.NewHandler(svc)
	r := chi.NewRouter()
	r.Route("/places", func(r chi.Router) {
		r.Route("/{id}/reviews", func(r chi.Router) {
			r.Get("/", h.List)
			r.With(middleware.WithAuth(testConfig)).Post("/", h.Create)
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

// newService returns a mock service that fails the test on any call that has
// no expectation.
func newService(t *testing.T) *mock.MockreviewService {
	t.Helper()
	return mock.NewMockreviewService(gomock.NewController(t))
}

// errorResult summarizes an error response: its status, whether it is JSON,
// the code of its { code, message } envelope and whether it has a message.
type errorResult struct {
	Status     int
	JSON       bool
	Code       string
	AnyMessage bool
}

// readError summarizes the error response in rec.
func readError(t *testing.T, rec *httptest.ResponseRecorder) errorResult {
	t.Helper()
	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), "error body is not JSON: %s", rec.Body)
	return errorResult{
		Status:     rec.Code,
		JSON:       strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json"),
		Code:       body.Code,
		AnyMessage: body.Message != "",
	}
}

// wantError is the errorResult of an error response with the status and code.
func wantError(status int, code string) errorResult {
	return errorResult{Status: status, JSON: true, Code: code, AnyMessage: true}
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

// linReview is a stored review with every field set.
func linReview() domain.Review {
	return domain.Review{
		ID:        1,
		PlaceID:   3,
		UserID:    7,
		Rating:    4,
		Comment:   new("Murah, porsi besar"),
		CreatedAt: time.Date(2026, 10, 4, 12, 30, 0, 0, time.UTC),
		User:      domain.Author{ID: 7, Name: "Lin"},
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

func TestHandler_List(t *testing.T) {
	t.Run("rejects_invalid_place_id", func(t *testing.T) {
		tests := []struct {
			name   string
			target string
		}{
			{name: "place_id_is_not_a_number", target: "/places/abc/reviews"},
			{name: "place_id_is_zero", target: "/places/0/reviews"},
			{name: "place_id_is_negative", target: "/places/-1/reviews"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// An invalid id must never reach the service: the mock has no expectations.
				rec := serve(t, newRouter(newService(t)), jsonRequest(http.MethodGet, tc.target, ""))
				assert.Equal(t, wantError(http.StatusBadRequest, "INVALID_PLACE_ID"), readError(t, rec))
			})
		}
	})

	t.Run("maps_service_errors", func(t *testing.T) {
		tests := []struct {
			name       string
			err        error
			wantStatus int
			wantCode   string
		}{
			{name: "unknown_place", err: domain.ErrPlaceNotFound, wantStatus: http.StatusNotFound, wantCode: "PLACE_NOT_FOUND"},
			{name: "storage_failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				svc := newService(t)
				svc.EXPECT().ListByPlace(gomock.Any(), int64(3)).Return(nil, tc.err)

				rec := serve(t, newRouter(svc), jsonRequest(http.MethodGet, "/places/3/reviews", ""))
				assert.Equal(t, wantError(tc.wantStatus, tc.wantCode), readError(t, rec))
			})
		}
	})

	t.Run("success", func(t *testing.T) {
		sari := domain.Review{
			ID:        2,
			PlaceID:   3,
			UserID:    8,
			Rating:    5,
			CreatedAt: time.Date(2026, 10, 5, 8, 0, 0, 0, time.UTC),
			User:      domain.Author{ID: 8, Name: "Sari"},
		}

		tests := []struct {
			name    string
			reviews []domain.Review
			want    string
		}{
			{name: "no_reviews", reviews: []domain.Review{}, want: `[]`},
			{
				name:    "reviews_in_the_order_given",
				reviews: []domain.Review{sari, linReview()},
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
				svc := newService(t)
				svc.EXPECT().ListByPlace(gomock.Any(), int64(3)).Return(tc.reviews, nil)

				rec := serve(t, newRouter(svc), jsonRequest(http.MethodGet, "/places/3/reviews", ""))
				require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body)
				assert.JSONEq(t, tc.want, responseData(t, rec))
			})
		}
	})
}

func TestHandler_Create(t *testing.T) {
	t.Run("rejects_invalid_place_id", func(t *testing.T) {
		tests := []struct {
			name   string
			target string
		}{
			{name: "place_id_is_not_a_number", target: "/places/abc/reviews"},
			{name: "place_id_is_zero", target: "/places/0/reviews"},
			{name: "place_id_is_negative", target: "/places/-1/reviews"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// An invalid id must never reach the service: the mock has no expectations.
				rec := serve(t, newRouter(newService(t)), authRequest(t, http.MethodPost, tc.target, `{"rating":4}`, 7))
				assert.Equal(t, wantError(http.StatusBadRequest, "INVALID_PLACE_ID"), readError(t, rec))
			})
		}
	})

	t.Run("rejects_malformed_bodies", func(t *testing.T) {
		tests := []struct {
			name string
			body string
		}{
			{name: "empty_body", body: ``},
			{name: "body_is_null", body: `null`},
			{name: "malformed_json", body: `{"rating":`},
			{name: "not_json", body: `four stars`},
			{name: "array_instead_of_object", body: `[4]`},
			{name: "rating_as_a_string", body: `{"rating":"4"}`},
			{name: "fractional_rating", body: `{"rating":4.5}`},
			{name: "comment_as_a_number", body: `{"rating":4,"comment":5}`},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// A malformed body must never reach the service: the mock has no expectations.
				rec := serve(t, newRouter(newService(t)), authRequest(t, http.MethodPost, "/places/3/reviews", tc.body, 7))
				assert.Equal(t, wantError(http.StatusBadRequest, "INVALID_BODY"), readError(t, rec))
			})
		}
	})

	t.Run("without_user", func(t *testing.T) {
		// Without WithAuth in front there is no user to attribute the review
		// to, so it must never reach the service.
		r := chi.NewRouter()
		r.Post("/places/{id}/reviews", reviews.NewHandler(newService(t)).Create)

		rec := serve(t, r, jsonRequest(http.MethodPost, "/places/3/reviews", `{"rating":4}`))
		assert.Equal(t, wantError(http.StatusUnauthorized, "UNAUTHORIZED"), readError(t, rec))
	})

	t.Run("passes_review_to_service", func(t *testing.T) {
		tests := []struct {
			name string
			body string
			want *domain.Review
		}{
			{
				name: "rating_and_comment",
				body: `{"rating":4,"comment":"Murah"}`,
				want: &domain.Review{PlaceID: 3, UserID: 7, Rating: 4, Comment: new("Murah")},
			},
			{
				name: "comment_is_passed_on_untrimmed",
				body: `{"rating":4,"comment":"  Murah  "}`,
				want: &domain.Review{PlaceID: 3, UserID: 7, Rating: 4, Comment: new("  Murah  ")},
			},
			{
				name: "rating_only",
				body: `{"rating":5}`,
				want: &domain.Review{PlaceID: 3, UserID: 7, Rating: 5},
			},
			{
				name: "null_comment",
				body: `{"rating":5,"comment":null}`,
				want: &domain.Review{PlaceID: 3, UserID: 7, Rating: 5},
			},
			{
				name: "place_from_path_author_from_token",
				body: `{"rating":2,"id":99,"place_id":5,"user_id":1,"created_at":"yesterday"}`,
				want: &domain.Review{PlaceID: 3, UserID: 7, Rating: 2},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				svc := newService(t)
				svc.EXPECT().Create(gomock.Any(), tc.want).Return(new(linReview()), nil)

				rec := serve(t, newRouter(svc), authRequest(t, http.MethodPost, "/places/3/reviews", tc.body, 7))
				assert.Equal(t, http.StatusCreated, rec.Code, "POST /places/3/reviews %s; body: %s", tc.body, rec.Body)
			})
		}
	})

	t.Run("maps_service_errors", func(t *testing.T) {
		tests := []struct {
			name       string
			err        error
			wantStatus int
			wantCode   string
		}{
			{name: "rating_out_of_range", err: domain.ErrReviewRatingInvalid, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "comment_too_long", err: domain.ErrReviewCommentTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "unknown_place", err: domain.ErrPlaceNotFound, wantStatus: http.StatusNotFound, wantCode: "PLACE_NOT_FOUND"},
			{name: "already_reviewed", err: domain.ErrPlaceAlreadyReviewed, wantStatus: http.StatusConflict, wantCode: "ALREADY_REVIEWED"},
			{name: "storage_failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				svc := newService(t)
				svc.EXPECT().Create(gomock.Any(), &domain.Review{PlaceID: 3, UserID: 7, Rating: 4}).Return(nil, tc.err)

				rec := serve(t, newRouter(svc), authRequest(t, http.MethodPost, "/places/3/reviews", `{"rating":4}`, 7))
				assert.Equal(t, wantError(tc.wantStatus, tc.wantCode), readError(t, rec))
			})
		}
	})

	t.Run("success", func(t *testing.T) {
		svc := newService(t)
		svc.EXPECT().Create(gomock.Any(), &domain.Review{PlaceID: 3, UserID: 7, Rating: 4, Comment: new("Murah, porsi besar")}).Return(new(linReview()), nil)

		rec := serve(t, newRouter(svc), authRequest(t, http.MethodPost, "/places/3/reviews",
			`{"rating":4,"comment":"Murah, porsi besar"}`, 7))
		require.Equal(t, http.StatusCreated, rec.Code, "body: %s", rec.Body)
		assert.JSONEq(t, linReviewJSON, responseData(t, rec))
	})

	t.Run("context_reaches_service", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		svc := newService(t)
		svc.EXPECT().Create(gomock.Any(), &domain.Review{PlaceID: 3, UserID: 7, Rating: 4}).DoAndReturn(func(got context.Context, _ *domain.Review) (*domain.Review, error) {
			assert.Error(t, got.Err(), "service was not given the request's context")
			return nil, got.Err()
		})

		req := authRequest(t, http.MethodPost, "/places/3/reviews", `{"rating":4}`, 7).WithContext(ctx)
		rec := serve(t, newRouter(svc), req)
		assert.NotEqual(t, http.StatusCreated, rec.Code, "status for a request whose context had ended")
	})
}

func TestHandler_InternalErrorsDoNotLeakDetails(t *testing.T) {
	dbErr := errors.New(`pq: relation "place_reviews" does not exist`)

	tests := []struct {
		name   string
		method string
		target string
		body   string
		expect func(svc *mock.MockreviewService)
	}{
		{
			name:   "list_reviews",
			method: http.MethodGet,
			target: "/places/3/reviews",
			expect: func(svc *mock.MockreviewService) {
				svc.EXPECT().ListByPlace(gomock.Any(), int64(3)).Return(nil, dbErr)
			},
		},
		{
			name:   "create_review",
			method: http.MethodPost,
			target: "/places/3/reviews",
			body:   `{"rating":4}`,
			expect: func(svc *mock.MockreviewService) {
				svc.EXPECT().Create(gomock.Any(), &domain.Review{PlaceID: 3, UserID: 7, Rating: 4}).Return(nil, dbErr)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := newService(t)
			tc.expect(svc)

			rec := serve(t, newRouter(svc), authRequest(t, tc.method, tc.target, tc.body, 7))
			require.Equal(t, http.StatusInternalServerError, rec.Code, "body: %s", rec.Body)
			for _, leak := range []string{"pq:", "relation"} {
				assert.NotContains(t, rec.Body.String(), leak, "500 body exposes storage detail")
			}
		})
	}
}
