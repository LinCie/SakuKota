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

	"sakukota-go/internal/config"
	"sakukota-go/internal/domain"
	"sakukota-go/internal/middleware"
	"sakukota-go/internal/module/reviews"
	"sakukota-go/internal/module/reviews/mock"
	"sakukota-go/internal/pkg/jwt"
	"sakukota-go/internal/storage"
)

// testConfig signs and checks the access tokens in these tests.
var testConfig = config.Config{JWTSecret: "test-secret", JWTExpiry: time.Hour}

// newRouter serves the review handlers on the paths the API uses for them.
// Creating a review requires an access token, as it does in production.
func newRouter(svc *mock.MockService) http.Handler {
	h := reviews.NewHandler(svc, mediaURL)
	r := chi.NewRouter()
	r.Route("/places", func(r chi.Router) {
		r.Route("/{id}/reviews", func(r chi.Router) {
			r.Get("/", h.List)
			r.With(middleware.WithAuth(testConfig)).Post("/", h.Create)
			r.Group(func(r chi.Router) {
				r.Use(middleware.WithAuth(testConfig))
				r.Post("/mine/medias/upload-url", h.CreateMediaUpload)
				r.Post("/mine/medias", h.AddMedia)
				r.Delete("/mine/medias/{mediaId}", h.DeleteMedia)
				r.Delete("/mine", h.DeleteMine)
			})
			// Admins only in production; the router test covers that check.
			r.Delete("/{reviewId}", h.Delete)
		})
	})
	return r
}

// mediaURL builds media links as the public storage URL would.
func mediaURL(key string) string { return "https://cdn.test/" + key }

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
func newService(t *testing.T) *mock.MockService {
	t.Helper()
	return mock.NewMockService(gomock.NewController(t))
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
		PlaceName: "Warung Bu Lin",
		UserID:    7,
		Rating:    4,
		Comment:   new("Murah, porsi besar"),
		CreatedAt: time.Date(2026, 10, 4, 12, 30, 0, 0, time.UTC),
		User:      domain.Author{ID: 7, Name: "Lin", AvatarKey: new("avatars/7/a.webp")},
		Medias: []domain.Media{
			{ID: 4, Key: "reviews/3/7/a.webp", ContentType: "image/webp", Size: 2048, Position: 1},
		},
	}
}

// linReviewJSON is linReview as the API returns it.
const linReviewJSON = `{
	"id": 1,
	"place_id": 3,
	"place_name": "Warung Bu Lin",
	"rating": 4,
	"comment": "Murah, porsi besar",
	"created_at": "2026-10-04T12:30:00Z",
	"user": {"id": 7, "name": "Lin", "avatar_url": "https://cdn.test/avatars/7/a.webp"},
	"medias": [{"id": 4, "url": "https://cdn.test/reviews/3/7/a.webp", "content_type": "image/webp", "size": 2048, "position": 1}]
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
			PlaceName: "Warung Bu Lin",
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
					"place_name": "Warung Bu Lin",
					"rating": 5,
					"comment": null,
					"created_at": "2026-10-05T08:00:00Z",
					"user": {"id": 8, "name": "Sari", "avatar_url": null},
					"medias": []
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

func TestHandler_ListMine(t *testing.T) {
	newMineRouter := func(svc *mock.MockService) http.Handler {
		h := reviews.NewHandler(svc, mediaURL)
		r := chi.NewRouter()
		r.With(middleware.WithAuth(testConfig)).Get("/users/me/reviews", h.ListMine)
		return r
	}

	t.Run("requires_a_user", func(t *testing.T) {
		// Without WithAuth in front there is no user, so nothing reaches the service.
		rec := serve(t, http.HandlerFunc(reviews.NewHandler(newService(t), mediaURL).ListMine), jsonRequest(http.MethodGet, "/users/me/reviews", ""))
		assert.Equal(t, wantError(http.StatusUnauthorized, "UNAUTHORIZED"), readError(t, rec))
	})

	t.Run("storage_failure", func(t *testing.T) {
		svc := newService(t)
		svc.EXPECT().ListByUser(gomock.Any(), int64(7)).Return(nil, errors.New("pq: connection reset"))

		rec := serve(t, newMineRouter(svc), authRequest(t, http.MethodGet, "/users/me/reviews", "", 7))
		assert.Equal(t, wantError(http.StatusInternalServerError, "INTERNAL_ERROR"), readError(t, rec))
		assert.NotContains(t, rec.Body.String(), "pq:")
	})

	t.Run("success", func(t *testing.T) {
		tests := []struct {
			name    string
			reviews []domain.Review
			want    string
		}{
			{name: "no_reviews", reviews: []domain.Review{}, want: `[]`},
			{name: "reviews_of_the_token_user", reviews: []domain.Review{linReview()}, want: `[` + linReviewJSON + `]`},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				svc := newService(t)
				// The user comes from the token.
				svc.EXPECT().ListByUser(gomock.Any(), int64(7)).Return(tc.reviews, nil)

				rec := serve(t, newMineRouter(svc), authRequest(t, http.MethodGet, "/users/me/reviews", "", 7))
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
		r.Post("/places/{id}/reviews", reviews.NewHandler(newService(t), mediaURL).Create)

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
				body: `{"rating":2,"id":99,"place_id":5,"user_id":1,"created_at":"yesterday","medias":[{"id":1}]}`,
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
		expect func(svc *mock.MockService)
	}{
		{
			name:   "list_reviews",
			method: http.MethodGet,
			target: "/places/3/reviews",
			expect: func(svc *mock.MockService) {
				svc.EXPECT().ListByPlace(gomock.Any(), int64(3)).Return(nil, dbErr)
			},
		},
		{
			name:   "create_review",
			method: http.MethodPost,
			target: "/places/3/reviews",
			body:   `{"rating":4}`,
			expect: func(svc *mock.MockService) {
				svc.EXPECT().Create(gomock.Any(), &domain.Review{PlaceID: 3, UserID: 7, Rating: 4}).Return(nil, dbErr)
			},
		},
		{
			name:   "create_media_upload",
			method: http.MethodPost,
			target: "/places/3/reviews/mine/medias/upload-url",
			body:   `{"content_type":"image/png"}`,
			expect: func(svc *mock.MockService) {
				svc.EXPECT().MediaUploadURL(gomock.Any(), int64(3), int64(7), "image/png").Return(storage.Upload{}, dbErr)
			},
		},
		{
			name:   "add_media",
			method: http.MethodPost,
			target: "/places/3/reviews/mine/medias",
			body:   `{"upload_key":"k"}`,
			expect: func(svc *mock.MockService) {
				svc.EXPECT().AddMedia(gomock.Any(), int64(3), int64(7), "k").Return(nil, dbErr)
			},
		},
		{
			name:   "delete_media",
			method: http.MethodDelete,
			target: "/places/3/reviews/mine/medias/4",
			expect: func(svc *mock.MockService) {
				svc.EXPECT().DeleteMedia(gomock.Any(), int64(3), int64(7), int64(4)).Return(dbErr)
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

// mediaRequest is a signed-in request by user 7 to one of the review media routes.
type mediaRequest struct {
	method string
	target string
	body   string
}

var (
	uploadURLRequest = mediaRequest{http.MethodPost, "/places/3/reviews/mine/medias/upload-url", `{"content_type":"image/webp"}`}
	addMediaRequest  = mediaRequest{http.MethodPost, "/places/3/reviews/mine/medias", `{"upload_key":"uploads/reviews/3/7/a.webp"}`}
	deleteRequest    = mediaRequest{http.MethodDelete, "/places/3/reviews/mine/medias/4", ""}
)

func TestHandler_MediaRoutes_RejectInvalidRequests(t *testing.T) {
	tests := []struct {
		name     string
		req      mediaRequest
		wantCode string
	}{
		{name: "upload_url_bad_place_id", req: mediaRequest{http.MethodPost, "/places/abc/reviews/mine/medias/upload-url", `{"content_type":"image/png"}`}, wantCode: "INVALID_PLACE_ID"},
		{name: "upload_url_place_id_zero", req: mediaRequest{http.MethodPost, "/places/0/reviews/mine/medias/upload-url", `{"content_type":"image/png"}`}, wantCode: "INVALID_PLACE_ID"},
		{name: "upload_url_empty_body", req: mediaRequest{http.MethodPost, uploadURLRequest.target, ``}, wantCode: "INVALID_BODY"},
		{name: "upload_url_content_type_as_a_number", req: mediaRequest{http.MethodPost, uploadURLRequest.target, `{"content_type":5}`}, wantCode: "INVALID_BODY"},
		{name: "add_bad_place_id", req: mediaRequest{http.MethodPost, "/places/-1/reviews/mine/medias", `{"upload_key":"k"}`}, wantCode: "INVALID_PLACE_ID"},
		{name: "add_empty_body", req: mediaRequest{http.MethodPost, addMediaRequest.target, ``}, wantCode: "INVALID_BODY"},
		{name: "add_upload_key_as_a_number", req: mediaRequest{http.MethodPost, addMediaRequest.target, `{"upload_key":5}`}, wantCode: "INVALID_BODY"},
		{name: "delete_bad_place_id", req: mediaRequest{http.MethodDelete, "/places/abc/reviews/mine/medias/4", ""}, wantCode: "INVALID_PLACE_ID"},
		{name: "delete_bad_media_id", req: mediaRequest{http.MethodDelete, "/places/3/reviews/mine/medias/abc", ""}, wantCode: "INVALID_MEDIA_ID"},
		{name: "delete_media_id_zero", req: mediaRequest{http.MethodDelete, "/places/3/reviews/mine/medias/0", ""}, wantCode: "INVALID_MEDIA_ID"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// An invalid request must never reach the service: the mock has no expectations.
			rec := serve(t, newRouter(newService(t)), authRequest(t, tc.req.method, tc.req.target, tc.req.body, 7))
			assert.Equal(t, wantError(http.StatusBadRequest, tc.wantCode), readError(t, rec))
		})
	}
}

func TestHandler_MediaRoutes_RequireAUser(t *testing.T) {
	h := reviews.NewHandler(newService(t), mediaURL)
	// Without WithAuth in front there is no user, so nothing reaches the service.
	r := chi.NewRouter()
	r.Post("/places/{id}/reviews/mine/medias/upload-url", h.CreateMediaUpload)
	r.Post("/places/{id}/reviews/mine/medias", h.AddMedia)
	r.Delete("/places/{id}/reviews/mine/medias/{mediaId}", h.DeleteMedia)

	for _, req := range []mediaRequest{uploadURLRequest, addMediaRequest, deleteRequest} {
		rec := serve(t, r, jsonRequest(req.method, req.target, req.body))
		assert.Equal(t, wantError(http.StatusUnauthorized, "UNAUTHORIZED"), readError(t, rec), "%s %s", req.method, req.target)
	}
}

func TestHandler_MediaRoutes_MapServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "no_review", err: domain.ErrReviewNotFound, wantStatus: http.StatusNotFound, wantCode: "REVIEW_NOT_FOUND"},
		{name: "unknown_media", err: domain.ErrMediaNotFound, wantStatus: http.StatusNotFound, wantCode: "MEDIA_NOT_FOUND"},
		{name: "review_full", err: domain.ErrMediaLimitReached, wantStatus: http.StatusConflict, wantCode: "MEDIA_LIMIT_REACHED"},
		{name: "unsupported_type", err: storage.ErrInvalidMediaType, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "not_uploaded", err: storage.ErrMediaNotUploaded, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "too_large", err: storage.ErrImageTooLarge, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "invalid_upload_key", err: storage.ErrInvalidUploadKey, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "storage_failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
	}

	routes := []struct {
		req    mediaRequest
		expect func(svc *mock.MockService, err error)
	}{
		{req: uploadURLRequest, expect: func(svc *mock.MockService, err error) {
			svc.EXPECT().MediaUploadURL(gomock.Any(), int64(3), int64(7), "image/webp").Return(storage.Upload{}, err)
		}},
		{req: addMediaRequest, expect: func(svc *mock.MockService, err error) {
			svc.EXPECT().AddMedia(gomock.Any(), int64(3), int64(7), "uploads/reviews/3/7/a.webp").Return(nil, err)
		}},
		{req: deleteRequest, expect: func(svc *mock.MockService, err error) {
			svc.EXPECT().DeleteMedia(gomock.Any(), int64(3), int64(7), int64(4)).Return(err)
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, route := range routes {
				svc := newService(t)
				route.expect(svc, tc.err)

				rec := serve(t, newRouter(svc), authRequest(t, route.req.method, route.req.target, route.req.body, 7))
				assert.Equal(t, wantError(tc.wantStatus, tc.wantCode), readError(t, rec), "%s %s", route.req.method, route.req.target)
			}
		})
	}
}

func TestHandler_CreateMediaUpload(t *testing.T) {
	svc := newService(t)
	// The place comes from the path and the user from the token.
	svc.EXPECT().MediaUploadURL(gomock.Any(), int64(3), int64(7), "image/webp").Return(storage.Upload{
		Key:       "uploads/reviews/3/7/a.webp",
		URL:       "https://storage.test/signed",
		Method:    http.MethodPut,
		Headers:   map[string]string{"Content-Type": "image/webp"},
		ExpiresAt: time.Date(2026, 10, 4, 12, 35, 0, 0, time.UTC),
	}, nil)

	rec := serve(t, newRouter(svc), authRequest(t, uploadURLRequest.method, uploadURLRequest.target, uploadURLRequest.body, 7))
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body)
	assert.JSONEq(t, `{
		"upload_key": "uploads/reviews/3/7/a.webp",
		"upload_url": "https://storage.test/signed",
		"method": "PUT",
		"headers": {"Content-Type": "image/webp"},
		"expires_at": "2026-10-04T12:35:00Z"
	}`, responseData(t, rec))
}

func TestHandler_AddMedia(t *testing.T) {
	svc := newService(t)
	svc.EXPECT().AddMedia(gomock.Any(), int64(3), int64(7), "uploads/reviews/3/7/a.webp").Return(new(linReview()), nil)

	rec := serve(t, newRouter(svc), authRequest(t, addMediaRequest.method, addMediaRequest.target, addMediaRequest.body, 7))
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body)
	assert.JSONEq(t, linReviewJSON, responseData(t, rec))
}

func TestHandler_DeleteMedia(t *testing.T) {
	svc := newService(t)
	svc.EXPECT().DeleteMedia(gomock.Any(), int64(3), int64(7), int64(4)).Return(nil)

	rec := serve(t, newRouter(svc), authRequest(t, deleteRequest.method, deleteRequest.target, deleteRequest.body, 7))
	assert.Equal(t, http.StatusNoContent, rec.Code, "body: %s", rec.Body)
	assert.Empty(t, rec.Body.String())
}

func TestHandler_Delete(t *testing.T) {
	t.Run("deleted", func(t *testing.T) {
		svc := newService(t)
		svc.EXPECT().Delete(gomock.Any(), int64(3), int64(9)).Return(nil)

		rec := serve(t, newRouter(svc), httptest.NewRequest(http.MethodDelete, "/places/3/reviews/9", nil))
		assert.Equal(t, http.StatusNoContent, rec.Code, "body: %s", rec.Body)
		assert.Empty(t, rec.Body.String())
	})

	t.Run("context_reaches_service", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		svc := newService(t)
		svc.EXPECT().Delete(gomock.Any(), int64(3), int64(9)).DoAndReturn(func(got context.Context, _, _ int64) error {
			assert.Error(t, got.Err(), "service was not given the request's context")
			return got.Err()
		})

		req := httptest.NewRequestWithContext(ctx, http.MethodDelete, "/places/3/reviews/9", nil)
		rec := serve(t, newRouter(svc), req)
		assert.NotEqual(t, http.StatusNoContent, rec.Code, "status for a request whose context had ended")
	})
}

func TestHandler_Delete_InvalidID(t *testing.T) {
	tests := []struct {
		name   string
		target string
		want   string
	}{
		{name: "place", target: "/places/abc/reviews/9", want: "INVALID_PLACE_ID"},
		{name: "zero_place", target: "/places/0/reviews/9", want: "INVALID_PLACE_ID"},
		{name: "review", target: "/places/3/reviews/abc", want: "INVALID_REVIEW_ID"},
		{name: "negative_review", target: "/places/3/reviews/-1", want: "INVALID_REVIEW_ID"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := serve(t, newRouter(newService(t)), httptest.NewRequest(http.MethodDelete, tc.target, nil))
			assert.Equal(t, wantError(http.StatusBadRequest, tc.want), readError(t, rec), "DELETE %s", tc.target)
		})
	}
}

func TestHandler_Delete_Errors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want errorResult
	}{
		{name: "unknown_review", err: domain.ErrReviewNotFound, want: wantError(http.StatusNotFound, "REVIEW_NOT_FOUND")},
		{name: "storage_failure", err: errors.New("connection reset"), want: wantError(http.StatusInternalServerError, "INTERNAL_ERROR")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := newService(t)
			svc.EXPECT().Delete(gomock.Any(), int64(3), int64(9)).Return(tc.err)

			rec := serve(t, newRouter(svc), httptest.NewRequest(http.MethodDelete, "/places/3/reviews/9", nil))
			assert.Equal(t, tc.want, readError(t, rec))
		})
	}
}

func TestHandler_DeleteMine(t *testing.T) {
	t.Run("deleted", func(t *testing.T) {
		svc := newService(t)
		svc.EXPECT().DeleteMine(gomock.Any(), int64(3), int64(7)).Return(nil)

		rec := serve(t, newRouter(svc), authRequest(t, http.MethodDelete, "/places/3/reviews/mine", "", 7))
		assert.Equal(t, http.StatusNoContent, rec.Code, "body: %s", rec.Body)
		assert.Empty(t, rec.Body.String())
	})

	t.Run("context_reaches_service", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		svc := newService(t)
		svc.EXPECT().DeleteMine(gomock.Any(), int64(3), int64(7)).DoAndReturn(func(got context.Context, _, _ int64) error {
			assert.Error(t, got.Err(), "service was not given the request's context")
			return got.Err()
		})

		req := authRequest(t, http.MethodDelete, "/places/3/reviews/mine", "", 7).WithContext(ctx)
		rec := serve(t, newRouter(svc), req)
		assert.NotEqual(t, http.StatusNoContent, rec.Code, "status for a request whose context had ended")
	})

	t.Run("signed_out", func(t *testing.T) {
		rec := serve(t, newRouter(newService(t)), httptest.NewRequest(http.MethodDelete, "/places/3/reviews/mine", nil))
		assert.Equal(t, wantError(http.StatusUnauthorized, "UNAUTHORIZED"), readError(t, rec))
	})

	t.Run("invalid_place_id", func(t *testing.T) {
		rec := serve(t, newRouter(newService(t)), authRequest(t, http.MethodDelete, "/places/abc/reviews/mine", "", 7))
		assert.Equal(t, wantError(http.StatusBadRequest, "INVALID_PLACE_ID"), readError(t, rec))
	})
}

func TestHandler_DeleteMine_Errors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want errorResult
	}{
		{name: "no_review", err: domain.ErrReviewNotFound, want: wantError(http.StatusNotFound, "REVIEW_NOT_FOUND")},
		{name: "storage_failure", err: errors.New("connection reset"), want: wantError(http.StatusInternalServerError, "INTERNAL_ERROR")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := newService(t)
			svc.EXPECT().DeleteMine(gomock.Any(), int64(3), int64(7)).Return(tc.err)

			rec := serve(t, newRouter(svc), authRequest(t, http.MethodDelete, "/places/3/reviews/mine", "", 7))
			assert.Equal(t, tc.want, readError(t, rec))
		})
	}
}
