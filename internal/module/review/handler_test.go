package review_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/mock/gomock"

	"rantau-go/internal/config"
	"rantau-go/internal/jwt"
	"rantau-go/internal/middleware"
	"rantau-go/internal/module/review"
	"rantau-go/internal/module/review/mock"
)

// testConfig signs and checks the access tokens in these tests.
var testConfig = config.Config{JWTSecret: "test-secret", JWTExpiry: time.Hour}

// newRouter serves the review handlers on the paths the API uses for them.
// Creating a review requires an access token, as it does in production.
func newRouter(svc review.Service) http.Handler {
	h := review.NewHandler(svc)
	r := chi.NewRouter()
	r.Get("/places/{id}/reviews", h.GetReviews)
	r.With(middleware.WithAuth(testConfig)).Post("/places/{id}/reviews", h.CreateReview)
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
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

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
	if rec.Code != status {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, status, rec.Body)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var body errorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("error body is not JSON: %v; body: %s", err, rec.Body)
	}
	if body.Code != code {
		t.Errorf("code = %q, want %q", body.Code, code)
	}
	if body.Message == "" {
		t.Error("message is empty")
	}
}

// responseData returns the value under the success envelope's "data" key.
func responseData(t *testing.T, rec *httptest.ResponseRecorder) json.RawMessage {
	t.Helper()
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("response is not JSON: %v; body: %s", err, rec.Body)
	}
	return env.Data
}

// assertData checks the status and that the response data is the JSON document want.
func assertData(t *testing.T, rec *httptest.ResponseRecorder, status int, want string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, status, rec.Body)
	}

	var got, expected any
	if err := json.Unmarshal(responseData(t, rec), &got); err != nil {
		t.Fatalf("data is not JSON: %v", err)
	}
	if err := json.Unmarshal([]byte(want), &expected); err != nil {
		t.Fatalf("bad test fixture: %v", err)
	}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("data = %s, want %s", responseData(t, rec), want)
	}
}

// linReview is a stored review with every field set.
func linReview() review.Review {
	return review.Review{
		ID:        1,
		PlaceID:   3,
		UserID:    7,
		Rating:    4,
		Comment:   ptr("Murah, porsi besar"),
		CreatedAt: time.Date(2026, 10, 4, 12, 30, 0, 0, time.UTC),
		User:      review.Author{ID: 7, Name: "Lin"},
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

// -----------------------------------------------------------------------------
// GET /places/{id}/reviews
// -----------------------------------------------------------------------------

func TestHandler_GetReviews_RejectsInvalidPlaceID(t *testing.T) {
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
}

func TestHandler_GetReviews_MapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "unknown place", err: review.ErrPlaceNotFound, wantStatus: http.StatusNotFound, wantCode: "PLACE_NOT_FOUND"},
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
}

func TestHandler_GetReviews_Success(t *testing.T) {
	sari := review.Review{
		ID:        2,
		PlaceID:   3,
		UserID:    8,
		Rating:    5,
		CreatedAt: time.Date(2026, 10, 5, 8, 0, 0, 0, time.UTC),
		User:      review.Author{ID: 8, Name: "Sari"},
	}

	tests := []struct {
		name    string
		reviews []review.Review
		want    string
	}{
		{name: "no reviews", reviews: []review.Review{}, want: `[]`},
		{
			name:    "reviews in the order given",
			reviews: []review.Review{sari, linReview()},
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
}

// -----------------------------------------------------------------------------
// POST /places/{id}/reviews
// -----------------------------------------------------------------------------

func TestHandler_CreateReview_RejectsInvalidPlaceID(t *testing.T) {
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
}

func TestHandler_CreateReview_RejectsMalformedBodies(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "empty body", body: ``},
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
}

func TestHandler_CreateReview_WithoutUser(t *testing.T) {
	// No expectations: without WithAuth in front there is no user to
	// attribute the review to, so it must never reach the service.
	svc := mock.NewMockService(gomock.NewController(t))
	r := chi.NewRouter()
	r.Post("/places/{id}/reviews", review.NewHandler(svc).CreateReview)

	rec := serve(t, r, jsonRequest(http.MethodPost, "/places/3/reviews", `{"rating":4}`))
	assertError(t, rec, http.StatusUnauthorized, "UNAUTHORIZED")
}

func TestHandler_CreateReview_PassesReviewToService(t *testing.T) {
	tests := []struct {
		name string
		body string
		want *review.Review
	}{
		{
			name: "rating and comment",
			body: `{"rating":4,"comment":"Murah"}`,
			want: &review.Review{PlaceID: 3, UserID: 7, Rating: 4, Comment: ptr("Murah")},
		},
		{
			name: "comment is passed on untrimmed",
			body: `{"rating":4,"comment":"  Murah  "}`,
			want: &review.Review{PlaceID: 3, UserID: 7, Rating: 4, Comment: ptr("  Murah  ")},
		},
		{
			name: "rating only",
			body: `{"rating":5}`,
			want: &review.Review{PlaceID: 3, UserID: 7, Rating: 5},
		},
		{
			name: "null comment",
			body: `{"rating":5,"comment":null}`,
			want: &review.Review{PlaceID: 3, UserID: 7, Rating: 5},
		},
		{
			name: "place comes from the path and author from the token",
			body: `{"rating":2,"id":99,"place_id":5,"user_id":1,"created_at":"yesterday"}`,
			want: &review.Review{PlaceID: 3, UserID: 7, Rating: 2},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().Create(gomock.Any(), tc.want).Return(ptr(linReview()), nil)

			rec := serve(t, newRouter(svc), authRequest(t, http.MethodPost, "/places/3/reviews", tc.body, 7))
			if rec.Code != http.StatusCreated {
				t.Fatalf("status = %d, want 201; body: %s", rec.Code, rec.Body)
			}
		})
	}
}

func TestHandler_CreateReview_MapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "rating out of range", err: review.ErrInvalidRating, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "comment too long", err: review.ErrCommentTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "unknown place", err: review.ErrPlaceNotFound, wantStatus: http.StatusNotFound, wantCode: "PLACE_NOT_FOUND"},
		{name: "already reviewed", err: review.ErrAlreadyReviewed, wantStatus: http.StatusConflict, wantCode: "ALREADY_REVIEWED"},
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
}

func TestHandler_CreateReview_Success(t *testing.T) {
	svc := mock.NewMockService(gomock.NewController(t))
	svc.EXPECT().Create(gomock.Any(), gomock.Any()).Return(ptr(linReview()), nil)

	rec := serve(t, newRouter(svc), authRequest(t, http.MethodPost, "/places/3/reviews",
		`{"rating":4,"comment":"Murah, porsi besar"}`, 7))
	assertData(t, rec, http.StatusCreated, linReviewJSON)
}
