package suggestion_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	"rantau-go/internal/module/place"
	"rantau-go/internal/module/suggestion"
	"rantau-go/internal/module/suggestion/mock"
	"rantau-go/internal/postgis"
)

// testConfig signs and checks the access tokens in these tests.
var testConfig = config.Config{JWTSecret: "test-secret", JWTExpiry: time.Hour}

// newRouter serves the suggestion handlers on the paths the API uses for them.
// Every route requires an access token, as it does in production. The admin
// role is checked by the API router in front of the admin routes, not by the
// handlers.
func newRouter(svc suggestion.Service) http.Handler {
	h := suggestion.NewHandler(svc)
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(middleware.WithAuth(testConfig))

		// Any signed-in user.
		r.Post("/suggestions", h.CreateSuggestion)
		r.Get("/suggestions/mine", h.GetMySuggestions)

		// Admins only.
		r.Get("/suggestions", h.GetSuggestions)
		r.Get("/suggestions/{id}", h.GetSuggestion)
		r.Post("/suggestions/{id}/approve", h.ApproveSuggestion)
		r.Post("/suggestions/{id}/reject", h.RejectSuggestion)
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

// tokenRequest is a JSON request carrying an access token for the user with
// the role.
func tokenRequest(t *testing.T, method, target, body string, userID int64, role string) *http.Request {
	t.Helper()
	token, err := jwt.GenerateToken(testConfig, userID, role)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	req := jsonRequest(method, target, body)
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

// authRequest is a JSON request carrying an access token for a regular user.
func authRequest(t *testing.T, method, target, body string, userID int64) *http.Request {
	t.Helper()
	return tokenRequest(t, method, target, body, userID, "user")
}

// adminRequest is a JSON request carrying an access token for an admin.
func adminRequest(t *testing.T, method, target, body string, adminID int64) *http.Request {
	t.Helper()
	return tokenRequest(t, method, target, body, adminID, "admin")
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

// linSuggestion is a stored, pending suggestion by Lin with most fields set.
func linSuggestion() suggestion.Suggestion {
	return suggestion.Suggestion{
		ID:            3,
		UserID:        7,
		SubDistrictID: 1,
		CategoryID:    2,
		Name:          "Warung Bu Ani",
		Address:       ptr("Jl. Sabang 12"),
		Location:      postgis.MustPoint(106.8, -6.2),
		Attributes:    json.RawMessage(`{"halal":true}`),
		Note:          ptr("Murah, porsi besar"),
		Status:        suggestion.StatusPending,
		CreatedAt:     time.Date(2026, 10, 4, 12, 30, 0, 0, time.UTC),
		User:          suggestion.Author{ID: 7, Name: "Lin"},
	}
}

// linSuggestionJSON is linSuggestion as the API returns it. Who reviewed a
// suggestion is not exposed.
const linSuggestionJSON = `{
	"id": 3,
	"sub_district_id": 1,
	"category_id": 2,
	"name": "Warung Bu Ani",
	"address": "Jl. Sabang 12",
	"location": {"type": "Point", "coordinates": [106.8, -6.2]},
	"operating_hours": null,
	"estimated_price_range": null,
	"short_description": null,
	"attributes": {"halal": true},
	"note": "Murah, porsi besar",
	"status": "pending",
	"rejection_reason": null,
	"place_id": null,
	"reviewed_at": null,
	"created_at": "2026-10-04T12:30:00Z",
	"user": {"id": 7, "name": "Lin"}
}`

// linApproved is linSuggestion after admin 99 approved it as place 42.
func linApproved() suggestion.Suggestion {
	s := linSuggestion()
	s.Status = suggestion.StatusApproved
	s.PlaceID = ptr(int64(42))
	s.ReviewedBy = ptr(int64(99))
	s.ReviewedAt = ptr(time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC))
	return s
}

const linApprovedJSON = `{
	"id": 3,
	"sub_district_id": 1,
	"category_id": 2,
	"name": "Warung Bu Ani",
	"address": "Jl. Sabang 12",
	"location": {"type": "Point", "coordinates": [106.8, -6.2]},
	"operating_hours": null,
	"estimated_price_range": null,
	"short_description": null,
	"attributes": {"halal": true},
	"note": "Murah, porsi besar",
	"status": "approved",
	"rejection_reason": null,
	"place_id": 42,
	"reviewed_at": "2026-10-05T09:00:00Z",
	"created_at": "2026-10-04T12:30:00Z",
	"user": {"id": 7, "name": "Lin"}
}`

// linRejected is linSuggestion after admin 99 rejected it.
func linRejected() suggestion.Suggestion {
	s := linSuggestion()
	s.Status = suggestion.StatusRejected
	s.RejectionReason = ptr("Sudah tutup")
	s.ReviewedBy = ptr(int64(99))
	s.ReviewedAt = ptr(time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC))
	return s
}

const linRejectedJSON = `{
	"id": 3,
	"sub_district_id": 1,
	"category_id": 2,
	"name": "Warung Bu Ani",
	"address": "Jl. Sabang 12",
	"location": {"type": "Point", "coordinates": [106.8, -6.2]},
	"operating_hours": null,
	"estimated_price_range": null,
	"short_description": null,
	"attributes": {"halal": true},
	"note": "Murah, porsi besar",
	"status": "rejected",
	"rejection_reason": "Sudah tutup",
	"place_id": null,
	"reviewed_at": "2026-10-05T09:00:00Z",
	"created_at": "2026-10-04T12:30:00Z",
	"user": {"id": 7, "name": "Lin"}
}`

// minimalBody is a suggestion body with only the required fields.
const minimalBody = `{"name":"Warung Bu Ani","sub_district_id":1,"category_id":2,` +
	`"location":{"type":"Point","coordinates":[106.8,-6.2]}}`

// -----------------------------------------------------------------------------
// POST /suggestions
// -----------------------------------------------------------------------------

func TestHandler_CreateSuggestion_RejectsMalformedBodies(t *testing.T) {
	const prefix = `{"name":"Warung","sub_district_id":1,"category_id":2,"location":`

	tests := []struct {
		name string
		body string
	}{
		{name: "empty body", body: ``},
		{name: "body is null", body: `null`},
		{name: "truncated JSON", body: `{"name":"Warung"`},
		{name: "not JSON", body: `warung enak`},
		{name: "array instead of object", body: `[]`},
		{name: "name is a number", body: `{"name":5,"sub_district_id":1,"category_id":2}`},
		{name: "sub_district_id is a string", body: `{"name":"Warung","sub_district_id":"one","category_id":2}`},
		{name: "category_id is fractional", body: `{"name":"Warung","sub_district_id":1,"category_id":2.5}`},
		{name: "note is a number", body: `{"name":"Warung","sub_district_id":1,"category_id":2,"note":5}`},
		{name: "location is a line", body: prefix + `{"type":"LineString","coordinates":[[106.8,-6.2],[106.9,-6.3]]}}`},
		{name: "coordinates are swapped", body: prefix + `{"type":"Point","coordinates":[-6.2,106.8]}}`},
		{name: "latitude above 90", body: prefix + `{"type":"Point","coordinates":[106.8,91]}}`},
		{name: "longitude above 180", body: prefix + `{"type":"Point","coordinates":[181,-6.2]}}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// No expectations: a malformed body must never reach the service.
			svc := mock.NewMockService(gomock.NewController(t))
			rec := serve(t, newRouter(svc), authRequest(t, http.MethodPost, "/suggestions", tc.body, 7))
			assertError(t, rec, http.StatusBadRequest, "INVALID_BODY")
		})
	}
}

func TestHandler_CreateSuggestion_WithoutUser(t *testing.T) {
	// No expectations: without WithAuth in front there is no user to
	// attribute the suggestion to, so it must never reach the service.
	svc := mock.NewMockService(gomock.NewController(t))
	r := chi.NewRouter()
	r.Post("/suggestions", suggestion.NewHandler(svc).CreateSuggestion)

	rec := serve(t, r, jsonRequest(http.MethodPost, "/suggestions", minimalBody))
	assertError(t, rec, http.StatusUnauthorized, "UNAUTHORIZED")
}

func TestHandler_CreateSuggestion_WithoutToken(t *testing.T) {
	// No expectations: an anonymous request must never reach the service.
	svc := mock.NewMockService(gomock.NewController(t))
	rec := serve(t, newRouter(svc), jsonRequest(http.MethodPost, "/suggestions", minimalBody))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body: %s", rec.Code, rec.Body)
	}
}

func TestHandler_CreateSuggestion_PassesSuggestionToService(t *testing.T) {
	tests := []struct {
		name string
		body string
		want *suggestion.Suggestion
	}{
		{
			name: "required fields only",
			body: minimalBody,
			want: &suggestion.Suggestion{
				UserID: 7, SubDistrictID: 1, CategoryID: 2, Name: "Warung Bu Ani",
				Location: postgis.MustPoint(106.8, -6.2),
			},
		},
		{
			name: "every field",
			body: `{"name":"Warung Bu Ani","sub_district_id":1,"category_id":2,` +
				`"location":{"type":"Point","coordinates":[106.8,-6.2]},` +
				`"address":"Jl. Sabang 12","operating_hours":"07:00-21:00",` +
				`"estimated_price_range":"Rp15.000-30.000","short_description":"Masakan Padang",` +
				`"attributes":{"halal":true},"note":"Murah, porsi besar"}`,
			want: &suggestion.Suggestion{
				UserID: 7, SubDistrictID: 1, CategoryID: 2, Name: "Warung Bu Ani",
				Location:            postgis.MustPoint(106.8, -6.2),
				Address:             ptr("Jl. Sabang 12"),
				OperatingHours:      ptr("07:00-21:00"),
				EstimatedPriceRange: ptr("Rp15.000-30.000"),
				ShortDescription:    ptr("Masakan Padang"),
				Attributes:          json.RawMessage(`{"halal":true}`),
				Note:                ptr("Murah, porsi besar"),
			},
		},
		{
			name: "text is passed on untrimmed",
			body: `{"name":"  Warung Bu Ani  ","sub_district_id":1,"category_id":2,` +
				`"location":{"type":"Point","coordinates":[106.8,-6.2]},"note":"  Murah  "}`,
			want: &suggestion.Suggestion{
				UserID: 7, SubDistrictID: 1, CategoryID: 2, Name: "  Warung Bu Ani  ",
				Location: postgis.MustPoint(106.8, -6.2), Note: ptr("  Murah  "),
			},
		},
		{
			name: "null optional fields are absent",
			body: `{"name":"Warung Bu Ani","sub_district_id":1,"category_id":2,` +
				`"location":{"type":"Point","coordinates":[106.8,-6.2]},"address":null,"note":null}`,
			want: &suggestion.Suggestion{
				UserID: 7, SubDistrictID: 1, CategoryID: 2, Name: "Warung Bu Ani",
				Location: postgis.MustPoint(106.8, -6.2),
			},
		},
		{
			name: "attributes are passed on as sent",
			body: `{"name":"Warung Bu Ani","sub_district_id":1,"category_id":2,` +
				`"location":{"type":"Point","coordinates":[106.8,-6.2]},"attributes":[1,2]}`,
			want: &suggestion.Suggestion{
				UserID: 7, SubDistrictID: 1, CategoryID: 2, Name: "Warung Bu Ani",
				Location: postgis.MustPoint(106.8, -6.2), Attributes: json.RawMessage(`[1,2]`),
			},
		},
		{
			name: "missing location is left for the service to reject",
			body: `{"name":"Warung Bu Ani","sub_district_id":1,"category_id":2}`,
			want: &suggestion.Suggestion{UserID: 7, SubDistrictID: 1, CategoryID: 2, Name: "Warung Bu Ani"},
		},
		{
			name: "author comes from the token and review fields are ignored",
			body: `{"name":"Warung Bu Ani","sub_district_id":1,"category_id":2,` +
				`"location":{"type":"Point","coordinates":[106.8,-6.2]},` +
				`"id":99,"user_id":1,"status":"approved","place_id":5,"reviewed_by":1,` +
				`"rejection_reason":"x","reviewed_at":"yesterday","created_at":"yesterday"}`,
			want: &suggestion.Suggestion{
				UserID: 7, SubDistrictID: 1, CategoryID: 2, Name: "Warung Bu Ani",
				Location: postgis.MustPoint(106.8, -6.2),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().Create(gomock.Any(), tc.want).Return(ptr(linSuggestion()), nil)

			rec := serve(t, newRouter(svc), authRequest(t, http.MethodPost, "/suggestions", tc.body, 7))
			if rec.Code != http.StatusCreated {
				t.Fatalf("status = %d, want 201; body: %s", rec.Code, rec.Body)
			}
		})
	}
}

func TestHandler_CreateSuggestion_MapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "blank name", err: suggestion.ErrInvalidName, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "name too long", err: suggestion.ErrNameTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "missing or outside location", err: suggestion.ErrInvalidLocation, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "note too long", err: suggestion.ErrNoteTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "address too long", err: suggestion.ErrAddressTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "operating hours too long", err: suggestion.ErrOperatingHoursTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "estimated price range too long", err: suggestion.ErrEstimatedPriceRangeTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "attributes not an object", err: suggestion.ErrInvalidAttributes, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "unknown sub-district", err: suggestion.ErrSubDistrictNotFound, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "unknown category", err: suggestion.ErrCategoryNotFound, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "too many pending", err: suggestion.ErrTooManyPending, wantStatus: http.StatusTooManyRequests, wantCode: "TOO_MANY_PENDING_SUGGESTIONS"},
		{name: "wrapped too many pending", err: fmt.Errorf("create: %w", suggestion.ErrTooManyPending), wantStatus: http.StatusTooManyRequests, wantCode: "TOO_MANY_PENDING_SUGGESTIONS"},
		{name: "storage failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, tc.err)

			rec := serve(t, newRouter(svc), authRequest(t, http.MethodPost, "/suggestions", minimalBody, 7))
			assertError(t, rec, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestHandler_CreateSuggestion_Success(t *testing.T) {
	svc := mock.NewMockService(gomock.NewController(t))
	svc.EXPECT().Create(gomock.Any(), gomock.Any()).Return(ptr(linSuggestion()), nil)

	rec := serve(t, newRouter(svc), authRequest(t, http.MethodPost, "/suggestions", minimalBody, 7))
	assertData(t, rec, http.StatusCreated, linSuggestionJSON)
}

// -----------------------------------------------------------------------------
// GET /suggestions/mine
// -----------------------------------------------------------------------------

func TestHandler_GetMySuggestions_WithoutUser(t *testing.T) {
	// No expectations: without a user there is nobody to list for.
	svc := mock.NewMockService(gomock.NewController(t))
	r := chi.NewRouter()
	r.Get("/suggestions/mine", suggestion.NewHandler(svc).GetMySuggestions)

	rec := serve(t, r, jsonRequest(http.MethodGet, "/suggestions/mine", ""))
	assertError(t, rec, http.StatusUnauthorized, "UNAUTHORIZED")
}

func TestHandler_GetMySuggestions_MapsServiceErrors(t *testing.T) {
	svc := mock.NewMockService(gomock.NewController(t))
	svc.EXPECT().GetByUser(gomock.Any(), int64(7)).Return(nil, errors.New("storage failure"))

	rec := serve(t, newRouter(svc), authRequest(t, http.MethodGet, "/suggestions/mine", "", 7))
	assertError(t, rec, http.StatusInternalServerError, "INTERNAL_ERROR")
}

func TestHandler_GetMySuggestions_Success(t *testing.T) {
	tests := []struct {
		name        string
		suggestions []suggestion.Suggestion
		want        string
	}{
		{name: "no suggestions", suggestions: []suggestion.Suggestion{}, want: `[]`},
		{
			name:        "suggestions in the order given",
			suggestions: []suggestion.Suggestion{linRejected(), linSuggestion()},
			want:        `[` + linRejectedJSON + `, ` + linSuggestionJSON + `]`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The user to list for comes from the token.
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().GetByUser(gomock.Any(), int64(7)).Return(tc.suggestions, nil)

			rec := serve(t, newRouter(svc), authRequest(t, http.MethodGet, "/suggestions/mine", "", 7))
			assertData(t, rec, http.StatusOK, tc.want)
		})
	}
}

// -----------------------------------------------------------------------------
// GET /suggestions
// -----------------------------------------------------------------------------

func TestHandler_GetSuggestions_PassesFilterToService(t *testing.T) {
	tests := []struct {
		name   string
		target string
		want   suggestion.SuggestionFilter
	}{
		{name: "no filter", target: "/suggestions", want: suggestion.SuggestionFilter{}},
		{name: "empty status is absent", target: "/suggestions?status=", want: suggestion.SuggestionFilter{}},
		{name: "pending", target: "/suggestions?status=pending", want: suggestion.SuggestionFilter{Status: suggestion.StatusPending}},
		{name: "approved", target: "/suggestions?status=approved", want: suggestion.SuggestionFilter{Status: suggestion.StatusApproved}},
		{name: "rejected", target: "/suggestions?status=rejected", want: suggestion.SuggestionFilter{Status: suggestion.StatusRejected}},
		{name: "unknown status is left for the service to reject", target: "/suggestions?status=deleted", want: suggestion.SuggestionFilter{Status: "deleted"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().GetMany(gomock.Any(), tc.want).Return([]suggestion.Suggestion{}, nil)

			rec := serve(t, newRouter(svc), adminRequest(t, http.MethodGet, tc.target, "", 99))
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body)
			}
		})
	}
}

func TestHandler_GetSuggestions_MapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "unknown status", err: suggestion.ErrInvalidStatus, wantStatus: http.StatusBadRequest, wantCode: "INVALID_STATUS"},
		{name: "storage failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().GetMany(gomock.Any(), gomock.Any()).Return(nil, tc.err)

			rec := serve(t, newRouter(svc), adminRequest(t, http.MethodGet, "/suggestions?status=pending", "", 99))
			assertError(t, rec, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestHandler_GetSuggestions_Success(t *testing.T) {
	tests := []struct {
		name        string
		suggestions []suggestion.Suggestion
		want        string
	}{
		{name: "no suggestions", suggestions: []suggestion.Suggestion{}, want: `[]`},
		{
			name:        "suggestions in the order given",
			suggestions: []suggestion.Suggestion{linSuggestion(), linApproved()},
			want:        `[` + linSuggestionJSON + `, ` + linApprovedJSON + `]`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().GetMany(gomock.Any(), gomock.Any()).Return(tc.suggestions, nil)

			rec := serve(t, newRouter(svc), adminRequest(t, http.MethodGet, "/suggestions", "", 99))
			assertData(t, rec, http.StatusOK, tc.want)
		})
	}
}

// -----------------------------------------------------------------------------
// GET /suggestions/{id}
// -----------------------------------------------------------------------------

// invalidIDs are suggestion paths whose id is not a positive 64-bit integer.
var invalidIDs = []struct {
	name string
	id   string
}{
	{name: "id is not a number", id: "abc"},
	{name: "id is fractional", id: "1.5"},
	{name: "id is zero", id: "0"},
	{name: "id is negative", id: "-1"},
	{name: "id overflows 64 bits", id: "99999999999999999999"},
}

func TestHandler_GetSuggestion_RejectsInvalidID(t *testing.T) {
	for _, tc := range invalidIDs {
		t.Run(tc.name, func(t *testing.T) {
			// No expectations: an invalid id must never reach the service.
			svc := mock.NewMockService(gomock.NewController(t))
			rec := serve(t, newRouter(svc), adminRequest(t, http.MethodGet, "/suggestions/"+tc.id, "", 99))
			assertError(t, rec, http.StatusBadRequest, "INVALID_SUGGESTION_ID")
		})
	}
}

func TestHandler_GetSuggestion_MapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "unknown suggestion", err: suggestion.ErrSuggestionNotFound, wantStatus: http.StatusNotFound, wantCode: "SUGGESTION_NOT_FOUND"},
		{name: "wrapped unknown suggestion", err: fmt.Errorf("get: %w", suggestion.ErrSuggestionNotFound), wantStatus: http.StatusNotFound, wantCode: "SUGGESTION_NOT_FOUND"},
		{name: "storage failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().GetSuggestion(gomock.Any(), int64(3)).Return(nil, tc.err)

			rec := serve(t, newRouter(svc), adminRequest(t, http.MethodGet, "/suggestions/3", "", 99))
			assertError(t, rec, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestHandler_GetSuggestion_Success(t *testing.T) {
	svc := mock.NewMockService(gomock.NewController(t))
	svc.EXPECT().GetSuggestion(gomock.Any(), int64(3)).Return(ptr(linSuggestion()), nil)

	rec := serve(t, newRouter(svc), adminRequest(t, http.MethodGet, "/suggestions/3", "", 99))
	assertData(t, rec, http.StatusOK, linSuggestionJSON)
}

// -----------------------------------------------------------------------------
// POST /suggestions/{id}/approve
// -----------------------------------------------------------------------------

// approveBody is the final place an admin approves a suggestion as. It has the
// same shape as the body of POST /places.
const approveBody = `{"name":"Warung Bu Ani","sub_district_id":1,"category_id":2,` +
	`"location":{"type":"Point","coordinates":[106.8,-6.2]}}`

func TestHandler_ApproveSuggestion_RejectsInvalidID(t *testing.T) {
	for _, tc := range invalidIDs {
		t.Run(tc.name, func(t *testing.T) {
			// No expectations: an invalid id must never reach the service.
			svc := mock.NewMockService(gomock.NewController(t))
			rec := serve(t, newRouter(svc), adminRequest(t, http.MethodPost, "/suggestions/"+tc.id+"/approve", approveBody, 99))
			assertError(t, rec, http.StatusBadRequest, "INVALID_SUGGESTION_ID")
		})
	}
}

func TestHandler_ApproveSuggestion_RejectsMalformedBodies(t *testing.T) {
	const prefix = `{"name":"Warung","sub_district_id":1,"category_id":2,"location":`

	tests := []struct {
		name string
		body string
	}{
		{name: "empty body", body: ``},
		{name: "body is null", body: `null`},
		{name: "truncated JSON", body: `{"name":"Warung"`},
		{name: "array instead of object", body: `[]`},
		{name: "sub_district_id is a string", body: `{"name":"Warung","sub_district_id":"one","category_id":2}`},
		{name: "location is a line", body: prefix + `{"type":"LineString","coordinates":[[106.8,-6.2],[106.9,-6.3]]}}`},
		{name: "coordinates are swapped", body: prefix + `{"type":"Point","coordinates":[-6.2,106.8]}}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// No expectations: a malformed body must never reach the service.
			svc := mock.NewMockService(gomock.NewController(t))
			rec := serve(t, newRouter(svc), adminRequest(t, http.MethodPost, "/suggestions/3/approve", tc.body, 99))
			assertError(t, rec, http.StatusBadRequest, "INVALID_BODY")
		})
	}
}

func TestHandler_ApproveSuggestion_WithoutUser(t *testing.T) {
	// No expectations: without a user there is no reviewer to record.
	svc := mock.NewMockService(gomock.NewController(t))
	r := chi.NewRouter()
	r.Post("/suggestions/{id}/approve", suggestion.NewHandler(svc).ApproveSuggestion)

	rec := serve(t, r, jsonRequest(http.MethodPost, "/suggestions/3/approve", approveBody))
	assertError(t, rec, http.StatusUnauthorized, "UNAUTHORIZED")
}

func TestHandler_ApproveSuggestion_PassesPlaceToService(t *testing.T) {
	tests := []struct {
		name string
		body string
		want *place.Place
	}{
		{
			name: "required fields only",
			body: approveBody,
			want: &place.Place{
				Name: "Warung Bu Ani", SubDistrictID: 1, CategoryID: 2,
				Location: postgis.MustPoint(106.8, -6.2),
			},
		},
		{
			name: "every field",
			body: `{"name":"Warung Bu Ani","sub_district_id":1,"category_id":2,` +
				`"location":{"type":"Point","coordinates":[106.8,-6.2]},` +
				`"address":"Jl. Sabang 12","operating_hours":"07:00-21:00",` +
				`"estimated_price_range":"Rp15.000-30.000","short_description":"Masakan Padang",` +
				`"attributes":{"halal":true}}`,
			want: &place.Place{
				Name: "Warung Bu Ani", SubDistrictID: 1, CategoryID: 2,
				Location:            postgis.MustPoint(106.8, -6.2),
				Address:             ptr("Jl. Sabang 12"),
				OperatingHours:      ptr("07:00-21:00"),
				EstimatedPriceRange: ptr("Rp15.000-30.000"),
				ShortDescription:    ptr("Masakan Padang"),
				Attributes:          json.RawMessage(`{"halal":true}`),
			},
		},
		{
			name: "missing location is left for the service to reject",
			body: `{"name":"Warung Bu Ani","sub_district_id":1,"category_id":2}`,
			want: &place.Place{Name: "Warung Bu Ani", SubDistrictID: 1, CategoryID: 2},
		},
		{
			name: "client id and unknown keys are ignored",
			body: `{"name":"Warung Bu Ani","sub_district_id":1,"category_id":2,` +
				`"location":{"type":"Point","coordinates":[106.8,-6.2]},"id":77,"created_at":"yesterday"}`,
			want: &place.Place{
				Name: "Warung Bu Ani", SubDistrictID: 1, CategoryID: 2,
				Location: postgis.MustPoint(106.8, -6.2),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The suggestion comes from the path and the reviewer from the token.
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().Approve(gomock.Any(), int64(3), int64(99), tc.want).Return(ptr(linApproved()), nil)

			rec := serve(t, newRouter(svc), adminRequest(t, http.MethodPost, "/suggestions/3/approve", tc.body, 99))
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body)
			}
		})
	}
}

func TestHandler_ApproveSuggestion_MapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "unknown suggestion", err: suggestion.ErrSuggestionNotFound, wantStatus: http.StatusNotFound, wantCode: "SUGGESTION_NOT_FOUND"},
		{name: "already reviewed", err: suggestion.ErrAlreadyReviewed, wantStatus: http.StatusConflict, wantCode: "ALREADY_REVIEWED"},
		{name: "wrapped already reviewed", err: fmt.Errorf("approve: %w", suggestion.ErrAlreadyReviewed), wantStatus: http.StatusConflict, wantCode: "ALREADY_REVIEWED"},
		{name: "place without location", err: suggestion.ErrInvalidLocation, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "blank place name", err: place.ErrInvalidName, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "place name too long", err: place.ErrNameTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "place address too long", err: place.ErrAddressTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "place operating hours too long", err: place.ErrOperatingHoursTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "place estimated price range too long", err: place.ErrEstimatedPriceRangeTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "place outside Indonesia", err: place.ErrInvalidLocation, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "place attributes invalid", err: place.ErrInvalidAttributes, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "wrapped place attributes invalid", err: fmt.Errorf("%w: %q is required", place.ErrInvalidAttributes, "contact"), wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "unknown sub-district", err: place.ErrSubDistrictNotFound, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "unknown category", err: place.ErrCategoryNotFound, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "storage failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().Approve(gomock.Any(), int64(3), int64(99), gomock.Any()).Return(nil, tc.err)

			rec := serve(t, newRouter(svc), adminRequest(t, http.MethodPost, "/suggestions/3/approve", approveBody, 99))
			assertError(t, rec, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestHandler_ApproveSuggestion_Success(t *testing.T) {
	svc := mock.NewMockService(gomock.NewController(t))
	svc.EXPECT().Approve(gomock.Any(), int64(3), int64(99), gomock.Any()).Return(ptr(linApproved()), nil)

	rec := serve(t, newRouter(svc), adminRequest(t, http.MethodPost, "/suggestions/3/approve", approveBody, 99))
	assertData(t, rec, http.StatusOK, linApprovedJSON)
}

// -----------------------------------------------------------------------------
// POST /suggestions/{id}/reject
// -----------------------------------------------------------------------------

func TestHandler_RejectSuggestion_RejectsInvalidID(t *testing.T) {
	for _, tc := range invalidIDs {
		t.Run(tc.name, func(t *testing.T) {
			// No expectations: an invalid id must never reach the service.
			svc := mock.NewMockService(gomock.NewController(t))
			rec := serve(t, newRouter(svc), adminRequest(t, http.MethodPost, "/suggestions/"+tc.id+"/reject", `{}`, 99))
			assertError(t, rec, http.StatusBadRequest, "INVALID_SUGGESTION_ID")
		})
	}
}

func TestHandler_RejectSuggestion_RejectsMalformedBodies(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "empty body", body: ``},
		{name: "body is null", body: `null`},
		{name: "truncated JSON", body: `{"reason":`},
		{name: "array instead of object", body: `[]`},
		{name: "reason is a number", body: `{"reason":5}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// No expectations: a malformed body must never reach the service.
			svc := mock.NewMockService(gomock.NewController(t))
			rec := serve(t, newRouter(svc), adminRequest(t, http.MethodPost, "/suggestions/3/reject", tc.body, 99))
			assertError(t, rec, http.StatusBadRequest, "INVALID_BODY")
		})
	}
}

func TestHandler_RejectSuggestion_WithoutUser(t *testing.T) {
	// No expectations: without a user there is no reviewer to record.
	svc := mock.NewMockService(gomock.NewController(t))
	r := chi.NewRouter()
	r.Post("/suggestions/{id}/reject", suggestion.NewHandler(svc).RejectSuggestion)

	rec := serve(t, r, jsonRequest(http.MethodPost, "/suggestions/3/reject", `{}`))
	assertError(t, rec, http.StatusUnauthorized, "UNAUTHORIZED")
}

func TestHandler_RejectSuggestion_PassesReasonToService(t *testing.T) {
	tests := []struct {
		name string
		body string
		want *string
	}{
		{name: "reason", body: `{"reason":"Sudah tutup"}`, want: ptr("Sudah tutup")},
		{name: "reason is passed on untrimmed", body: `{"reason":"  Sudah tutup  "}`, want: ptr("  Sudah tutup  ")},
		{name: "no reason", body: `{}`, want: nil},
		{name: "null reason", body: `{"reason":null}`, want: nil},
		{name: "unknown keys are ignored", body: `{"reason":"Sudah tutup","status":"approved"}`, want: ptr("Sudah tutup")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The suggestion comes from the path and the reviewer from the token.
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().Reject(gomock.Any(), int64(3), int64(99), tc.want).Return(ptr(linRejected()), nil)

			rec := serve(t, newRouter(svc), adminRequest(t, http.MethodPost, "/suggestions/3/reject", tc.body, 99))
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body)
			}
		})
	}
}

func TestHandler_RejectSuggestion_MapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "unknown suggestion", err: suggestion.ErrSuggestionNotFound, wantStatus: http.StatusNotFound, wantCode: "SUGGESTION_NOT_FOUND"},
		{name: "already reviewed", err: suggestion.ErrAlreadyReviewed, wantStatus: http.StatusConflict, wantCode: "ALREADY_REVIEWED"},
		{name: "reason too long", err: suggestion.ErrReasonTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "storage failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().Reject(gomock.Any(), int64(3), int64(99), gomock.Any()).Return(nil, tc.err)

			rec := serve(t, newRouter(svc), adminRequest(t, http.MethodPost, "/suggestions/3/reject", `{"reason":"Sudah tutup"}`, 99))
			assertError(t, rec, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestHandler_RejectSuggestion_Success(t *testing.T) {
	svc := mock.NewMockService(gomock.NewController(t))
	svc.EXPECT().Reject(gomock.Any(), int64(3), int64(99), gomock.Any()).Return(ptr(linRejected()), nil)

	rec := serve(t, newRouter(svc), adminRequest(t, http.MethodPost, "/suggestions/3/reject", `{"reason":"Sudah tutup"}`, 99))
	assertData(t, rec, http.StatusOK, linRejectedJSON)
}

// -----------------------------------------------------------------------------
// Cross-cutting: context and internal details
// -----------------------------------------------------------------------------

func TestHandler_ContextReachesService(t *testing.T) {
	tests := []struct {
		name   string
		newCtx func() (context.Context, context.CancelFunc)
	}{
		{
			name: "client disconnected",
			newCtx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, cancel
			},
		},
		{
			name: "deadline passes while storage is slow",
			newCtx: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 20*time.Millisecond)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := tc.newCtx()
			defer cancel()

			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
				func(got context.Context, _ *suggestion.Suggestion) (*suggestion.Suggestion, error) {
					select {
					case <-got.Done():
						return nil, got.Err()
					case <-time.After(2 * time.Second):
						t.Error("service was not given the request's context")
						return nil, errors.New("timed out")
					}
				})

			req := authRequest(t, http.MethodPost, "/suggestions", minimalBody, 7).WithContext(ctx)
			rec := serve(t, newRouter(svc), req)
			if rec.Code == http.StatusCreated {
				t.Error("status 201 for a request whose context had ended")
			}
		})
	}
}

func TestHandler_InternalErrorsDoNotLeakDetails(t *testing.T) {
	dbErr := errors.New(`pq: relation "place_suggestions" does not exist`)

	tests := []struct {
		name string
		req  func(t *testing.T) *http.Request
		fail func(svc *mock.MockService)
	}{
		{
			name: "create suggestion",
			req: func(t *testing.T) *http.Request {
				return authRequest(t, http.MethodPost, "/suggestions", minimalBody, 7)
			},
			fail: func(s *mock.MockService) {
				s.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, dbErr)
			},
		},
		{
			name: "list my suggestions",
			req: func(t *testing.T) *http.Request {
				return authRequest(t, http.MethodGet, "/suggestions/mine", "", 7)
			},
			fail: func(s *mock.MockService) {
				s.EXPECT().GetByUser(gomock.Any(), gomock.Any()).Return(nil, dbErr)
			},
		},
		{
			name: "list suggestions",
			req: func(t *testing.T) *http.Request {
				return adminRequest(t, http.MethodGet, "/suggestions", "", 99)
			},
			fail: func(s *mock.MockService) {
				s.EXPECT().GetMany(gomock.Any(), gomock.Any()).Return(nil, dbErr)
			},
		},
		{
			name: "get suggestion",
			req: func(t *testing.T) *http.Request {
				return adminRequest(t, http.MethodGet, "/suggestions/3", "", 99)
			},
			fail: func(s *mock.MockService) {
				s.EXPECT().GetSuggestion(gomock.Any(), gomock.Any()).Return(nil, dbErr)
			},
		},
		{
			name: "approve suggestion",
			req: func(t *testing.T) *http.Request {
				return adminRequest(t, http.MethodPost, "/suggestions/3/approve", approveBody, 99)
			},
			fail: func(s *mock.MockService) {
				s.EXPECT().Approve(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, dbErr)
			},
		},
		{
			name: "reject suggestion",
			req: func(t *testing.T) *http.Request {
				return adminRequest(t, http.MethodPost, "/suggestions/3/reject", `{}`, 99)
			},
			fail: func(s *mock.MockService) {
				s.EXPECT().Reject(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, dbErr)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			tc.fail(svc)

			rec := serve(t, newRouter(svc), tc.req(t))
			assertError(t, rec, http.StatusInternalServerError, "INTERNAL_ERROR")
			if strings.Contains(rec.Body.String(), "pq:") || strings.Contains(rec.Body.String(), "relation") {
				t.Errorf("500 body exposes storage detail: %s", rec.Body)
			}
		})
	}
}
