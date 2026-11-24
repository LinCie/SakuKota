package suggestions_test

import (
	"bytes"
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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"sakukota-go/internal/config"
	"sakukota-go/internal/domain"
	"sakukota-go/internal/middleware"
	"sakukota-go/internal/module/suggestions"
	"sakukota-go/internal/module/suggestions/mock"
	"sakukota-go/internal/pkg/jwt"
	"sakukota-go/internal/pkg/postgis"
)

// testConfig signs and checks the access tokens in these tests.
var testConfig = config.Config{JWTSecret: "test-secret", JWTExpiry: time.Hour}

// newRouter serves the suggestion handlers on the paths the API uses for them.
// Every route requires an access token, as it does in production. The admin
// role is checked by the API router in front of the admin routes, not by the
// handlers.
func newRouter(svc *mock.MockService) http.Handler {
	h := suggestions.NewHandler(svc)
	r := chi.NewRouter()
	r.Route("/suggestions", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(middleware.WithAuth(testConfig))

			// Any signed-in user.
			r.Post("/", h.Create)
			r.Get("/mine", h.ListMine)

			// Admins only.
			r.Get("/", h.List)
			r.Get("/{id}", h.Show)
			r.Post("/{id}/approve", h.Approve)
			r.Post("/{id}/reject", h.Reject)
		})
	})
	return r
}

// newMockService returns a mock suggestion service that fails the test on any
// call without an expectation.
func newMockService(t *testing.T) *mock.MockService {
	t.Helper()
	return mock.NewMockService(gomock.NewController(t))
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
	require.NoError(t, err)

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

// diffError describes how rec differs from the status and the { code, message }
// error envelope. It returns "" if it matches.
func diffError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) string {
	t.Helper()
	if rec.Code != status {
		return fmt.Sprintf("status = %d, want %d; body: %s", rec.Code, status, rec.Body)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		return fmt.Sprintf("Content-Type = %q, want application/json", ct)
	}

	var body errorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		return fmt.Sprintf("error body is not JSON: %v; body: %s", err, rec.Body)
	}
	if body.Code != code {
		return fmt.Sprintf("code = %q, want %q", body.Code, code)
	}
	if body.Message == "" {
		return "message is empty"
	}

	return ""
}

// responseData returns the value under the success envelope's "data" key.
func responseData(t *testing.T, rec *httptest.ResponseRecorder) json.RawMessage {
	t.Helper()
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env), "response is not JSON: %s", rec.Body)
	return env.Data
}

// diffData describes how rec differs from the status and the response data
// being the JSON document want. It returns "" if it matches.
func diffData(t *testing.T, rec *httptest.ResponseRecorder, status int, want string) string {
	t.Helper()
	if rec.Code != status {
		return fmt.Sprintf("status = %d, want %d; body: %s", rec.Code, status, rec.Body)
	}
	if data := responseData(t, rec); !jsonEqual(t, want, data) {
		return fmt.Sprintf("data = %s, want %s", data, want)
	}

	return ""
}

// linSuggestion is a stored, pending suggestion by Lin with most fields set.
func linSuggestion() domain.Suggestion {
	return domain.Suggestion{
		ID:            3,
		UserID:        7,
		SubDistrictID: 1,
		CategoryID:    2,
		Name:          "Warung Bu Ani",
		Address:       new("Jl. Sabang 12"),
		Location:      postgis.MustPoint(106.8, -6.2),
		Attributes:    json.RawMessage(`{"halal":true}`),
		Note:          new("Murah, porsi besar"),
		Status:        domain.StatusPending,
		CreatedAt:     time.Date(2026, 10, 4, 12, 30, 0, 0, time.UTC),
		User:          domain.Author{ID: 7, Name: "Lin"},
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
func linApproved() domain.Suggestion {
	s := linSuggestion()
	s.Status = domain.StatusApproved
	s.PlaceID = new(int64(42))
	s.ReviewedBy = new(int64(99))
	s.ReviewedAt = new(time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC))
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
func linRejected() domain.Suggestion {
	s := linSuggestion()
	s.Status = domain.StatusRejected
	s.RejectionReason = new("Sudah tutup")
	s.ReviewedBy = new(int64(99))
	s.ReviewedAt = new(time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC))
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

// jsonEqual reports whether got is the JSON document want, ignoring formatting.
func jsonEqual(t *testing.T, want string, got []byte) bool {
	t.Helper()

	var w, g any
	require.NoError(t, json.NewDecoder(strings.NewReader(want)).Decode(&w), "invalid JSON %s", want)
	require.NoError(t, json.NewDecoder(bytes.NewReader(got)).Decode(&g), "invalid JSON %s", got)

	return reflect.DeepEqual(w, g)
}

// returnOne returns a pointer to a copy of s, as a service returns it.
func returnOne(s domain.Suggestion) *domain.Suggestion { return &s }

func TestHandler_Create(t *testing.T) {
	t.Run("rejects malformed bodies", func(t *testing.T) {
		const prefix = `{"name":"Warung","sub_district_id":1,"category_id":2,"location":`

		tests := []struct {
			name string
			body string
		}{
			{name: "empty_body", body: ``},
			{name: "body_is_null", body: `null`},
			{name: "truncated_JSON", body: `{"name":"Warung"`},
			{name: "not_JSON", body: `warung enak`},
			{name: "array_instead_of_object", body: `[]`},
			{name: "name_is_a_number", body: `{"name":5,"sub_district_id":1,"category_id":2}`},
			{name: "sub_district_id_is_a_string", body: `{"name":"Warung","sub_district_id":"one","category_id":2}`},
			{name: "category_id_is_fractional", body: `{"name":"Warung","sub_district_id":1,"category_id":2.5}`},
			{name: "note_is_a_number", body: `{"name":"Warung","sub_district_id":1,"category_id":2,"note":5}`},
			{name: "location_is_a_line", body: prefix + `{"type":"LineString","coordinates":[[106.8,-6.2],[106.9,-6.3]]}}`},
			{name: "coordinates_are_swapped", body: prefix + `{"type":"Point","coordinates":[-6.2,106.8]}}`},
			{name: "latitude_above_90", body: prefix + `{"type":"Point","coordinates":[106.8,91]}}`},
			{name: "longitude_above_180", body: prefix + `{"type":"Point","coordinates":[181,-6.2]}}`},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// A malformed body must never reach the service.
				svc := newMockService(t)
				rec := serve(t, newRouter(svc), authRequest(t, http.MethodPost, "/suggestions", tc.body, 7))
				assert.Empty(t, diffError(t, rec, http.StatusBadRequest, "INVALID_BODY"), "POST /suggestions %s", tc.body)
			})
		}
	})

	t.Run("without_user", func(t *testing.T) {
		// Without WithAuth in front there is no user to attribute the
		// suggestion to, so it must never reach the service.
		svc := newMockService(t)
		r := chi.NewRouter()
		r.Post("/suggestions", suggestions.NewHandler(svc).Create)

		rec := serve(t, r, jsonRequest(http.MethodPost, "/suggestions", minimalBody))
		assert.Empty(t, diffError(t, rec, http.StatusUnauthorized, "UNAUTHORIZED"), "POST /suggestions without user")
	})

	t.Run("without_token", func(t *testing.T) {
		// An anonymous request must never reach the service.
		svc := newMockService(t)
		rec := serve(t, newRouter(svc), jsonRequest(http.MethodPost, "/suggestions", minimalBody))
		assert.Equal(t, http.StatusUnauthorized, rec.Code, "POST /suggestions without token; body: %s", rec.Body)
	})

	t.Run("passes_suggestion_to_service", func(t *testing.T) {
		tests := []struct {
			name string
			body string
			want *domain.Suggestion
		}{
			{
				name: "required_fields_only",
				body: minimalBody,
				want: &domain.Suggestion{
					UserID: 7, SubDistrictID: 1, CategoryID: 2, Name: "Warung Bu Ani",
					Location: postgis.MustPoint(106.8, -6.2),
				},
			},
			{
				name: "every_field",
				body: `{"name":"Warung Bu Ani","sub_district_id":1,"category_id":2,` +
					`"location":{"type":"Point","coordinates":[106.8,-6.2]},` +
					`"address":"Jl. Sabang 12","operating_hours":"07:00-21:00",` +
					`"estimated_price_range":"Rp15.000-30.000","short_description":"Masakan Padang",` +
					`"attributes":{"halal":true},"note":"Murah, porsi besar"}`,
				want: &domain.Suggestion{
					UserID: 7, SubDistrictID: 1, CategoryID: 2, Name: "Warung Bu Ani",
					Location:            postgis.MustPoint(106.8, -6.2),
					Address:             new("Jl. Sabang 12"),
					OperatingHours:      new("07:00-21:00"),
					EstimatedPriceRange: new("Rp15.000-30.000"),
					ShortDescription:    new("Masakan Padang"),
					Attributes:          json.RawMessage(`{"halal":true}`),
					Note:                new("Murah, porsi besar"),
				},
			},
			{
				name: "text_passed_on_untrimmed",
				body: `{"name":"  Warung Bu Ani  ","sub_district_id":1,"category_id":2,` +
					`"location":{"type":"Point","coordinates":[106.8,-6.2]},"note":"  Murah  "}`,
				want: &domain.Suggestion{
					UserID: 7, SubDistrictID: 1, CategoryID: 2, Name: "  Warung Bu Ani  ",
					Location: postgis.MustPoint(106.8, -6.2), Note: new("  Murah  "),
				},
			},
			{
				name: "null_optional_fields_absent",
				body: `{"name":"Warung Bu Ani","sub_district_id":1,"category_id":2,` +
					`"location":{"type":"Point","coordinates":[106.8,-6.2]},"address":null,"note":null}`,
				want: &domain.Suggestion{
					UserID: 7, SubDistrictID: 1, CategoryID: 2, Name: "Warung Bu Ani",
					Location: postgis.MustPoint(106.8, -6.2),
				},
			},
			{
				name: "attributes_passed_on_as_sent",
				body: `{"name":"Warung Bu Ani","sub_district_id":1,"category_id":2,` +
					`"location":{"type":"Point","coordinates":[106.8,-6.2]},"attributes":[1,2]}`,
				want: &domain.Suggestion{
					UserID: 7, SubDistrictID: 1, CategoryID: 2, Name: "Warung Bu Ani",
					Location: postgis.MustPoint(106.8, -6.2), Attributes: json.RawMessage(`[1,2]`),
				},
			},
			{
				name: "missing_location_left_to_service",
				body: `{"name":"Warung Bu Ani","sub_district_id":1,"category_id":2}`,
				want: &domain.Suggestion{UserID: 7, SubDistrictID: 1, CategoryID: 2, Name: "Warung Bu Ani"},
			},
			{
				name: "author_from_token_review_fields_ignored",
				body: `{"name":"Warung Bu Ani","sub_district_id":1,"category_id":2,` +
					`"location":{"type":"Point","coordinates":[106.8,-6.2]},` +
					`"id":99,"user_id":1,"status":"approved","place_id":5,"reviewed_by":1,` +
					`"rejection_reason":"x","reviewed_at":"yesterday","created_at":"yesterday"}`,
				want: &domain.Suggestion{
					UserID: 7, SubDistrictID: 1, CategoryID: 2, Name: "Warung Bu Ani",
					Location: postgis.MustPoint(106.8, -6.2),
				},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				svc := newMockService(t)
				svc.EXPECT().Create(gomock.Any(), tc.want).Return(returnOne(linSuggestion()), nil)

				rec := serve(t, newRouter(svc), authRequest(t, http.MethodPost, "/suggestions", tc.body, 7))
				assert.Equal(t, http.StatusCreated, rec.Code, "POST /suggestions; body: %s", rec.Body)
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
			{name: "blank_name", err: domain.ErrPlaceNameRequired, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "name_too_long", err: domain.ErrPlaceNameTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "invalid_location", err: domain.ErrSuggestionLocationInvalid, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "note_too_long", err: domain.ErrSuggestionNoteTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "address_too_long", err: domain.ErrPlaceAddressTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "operating_hours_too_long", err: domain.ErrPlaceOperatingHoursTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "price_range_too_long", err: domain.ErrPlaceEstimatedPriceRangeTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "attributes_not_object", err: domain.ErrSuggestionAttributesInvalid, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "unknown_sub_district", err: domain.ErrSubDistrictNotFound, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "unknown_category", err: domain.ErrCategoryNotFound, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "too_many_pending", err: domain.ErrSuggestionLimitReached, wantStatus: http.StatusTooManyRequests, wantCode: "TOO_MANY_PENDING_SUGGESTIONS"},
			{name: "wrapped_too_many_pending", err: fmt.Errorf("create: %w", domain.ErrSuggestionLimitReached), wantStatus: http.StatusTooManyRequests, wantCode: "TOO_MANY_PENDING_SUGGESTIONS"},
			{name: "storage_failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				svc := newMockService(t)
				svc.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, tc.err)

				rec := serve(t, newRouter(svc), authRequest(t, http.MethodPost, "/suggestions", minimalBody, 7))
				assert.Empty(t, diffError(t, rec, tc.wantStatus, tc.wantCode), "POST /suggestions with service error %v", tc.err)
			})
		}
	})

	t.Run("success", func(t *testing.T) {
		svc := newMockService(t)
		svc.EXPECT().Create(gomock.Any(), gomock.Any()).Return(returnOne(linSuggestion()), nil)

		rec := serve(t, newRouter(svc), authRequest(t, http.MethodPost, "/suggestions", minimalBody, 7))
		assert.Empty(t, diffData(t, rec, http.StatusCreated, linSuggestionJSON), "POST /suggestions")
	})

	t.Run("context_reaches_service", func(t *testing.T) {
		tests := []struct {
			name   string
			newCtx func(parent context.Context) (context.Context, context.CancelFunc)
		}{
			{
				name: "client_disconnected",
				newCtx: func(parent context.Context) (context.Context, context.CancelFunc) {
					ctx, cancel := context.WithCancel(parent)
					cancel()
					return ctx, cancel
				},
			},
			{
				name: "deadline_passes",
				newCtx: func(parent context.Context) (context.Context, context.CancelFunc) {
					return context.WithTimeout(parent, 20*time.Millisecond)
				},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				ctx, cancel := tc.newCtx(t.Context())
				defer cancel()

				svc := newMockService(t)
				svc.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(got context.Context, _ *domain.Suggestion) (*domain.Suggestion, error) {
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
				assert.NotEqual(t, http.StatusCreated, rec.Code, "POST /suggestions for a request whose context had ended")
			})
		}
	})
}

func TestHandler_ListMine(t *testing.T) {
	t.Run("without_user", func(t *testing.T) {
		// Without a user there is nobody to list for.
		svc := newMockService(t)
		r := chi.NewRouter()
		r.Get("/suggestions/mine", suggestions.NewHandler(svc).ListMine)

		rec := serve(t, r, jsonRequest(http.MethodGet, "/suggestions/mine", ""))
		assert.Empty(t, diffError(t, rec, http.StatusUnauthorized, "UNAUTHORIZED"), "GET /suggestions/mine without user")
	})

	t.Run("maps_service_errors", func(t *testing.T) {
		svc := newMockService(t)
		svc.EXPECT().ListByUser(gomock.Any(), int64(7)).Return(nil, errors.New("storage failure"))

		rec := serve(t, newRouter(svc), authRequest(t, http.MethodGet, "/suggestions/mine", "", 7))
		assert.Empty(t, diffError(t, rec, http.StatusInternalServerError, "INTERNAL_ERROR"), "GET /suggestions/mine")
	})

	t.Run("success", func(t *testing.T) {
		tests := []struct {
			name        string
			suggestions []domain.Suggestion
			want        string
		}{
			{name: "no_suggestions", suggestions: []domain.Suggestion{}, want: `[]`},
			{
				name:        "suggestions_in_given_order",
				suggestions: []domain.Suggestion{linRejected(), linSuggestion()},
				want:        `[` + linRejectedJSON + `, ` + linSuggestionJSON + `]`,
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// The user to list for comes from the token.
				svc := newMockService(t)
				svc.EXPECT().ListByUser(gomock.Any(), int64(7)).Return(tc.suggestions, nil)

				rec := serve(t, newRouter(svc), authRequest(t, http.MethodGet, "/suggestions/mine", "", 7))
				assert.Empty(t, diffData(t, rec, http.StatusOK, tc.want), "GET /suggestions/mine")
			})
		}
	})
}

func TestHandler_List(t *testing.T) {
	t.Run("passes_filter_to_service", func(t *testing.T) {
		tests := []struct {
			name   string
			target string
			want   suggestions.Filter
		}{
			{name: "no_filter", target: "/suggestions", want: suggestions.Filter{}},
			{name: "empty_status_is_absent", target: "/suggestions?status=", want: suggestions.Filter{}},
			{name: "pending", target: "/suggestions?status=pending", want: suggestions.Filter{Status: domain.StatusPending}},
			{name: "approved", target: "/suggestions?status=approved", want: suggestions.Filter{Status: domain.StatusApproved}},
			{name: "rejected", target: "/suggestions?status=rejected", want: suggestions.Filter{Status: domain.StatusRejected}},
			{name: "unknown_status_left_to_service", target: "/suggestions?status=deleted", want: suggestions.Filter{Status: "deleted"}},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				svc := newMockService(t)
				svc.EXPECT().List(gomock.Any(), tc.want).Return([]domain.Suggestion{}, nil)

				rec := serve(t, newRouter(svc), adminRequest(t, http.MethodGet, tc.target, "", 99))
				assert.Equal(t, http.StatusOK, rec.Code, "GET %s; body: %s", tc.target, rec.Body)
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
			{name: "unknown_status", err: domain.ErrSuggestionStatusInvalid, wantStatus: http.StatusBadRequest, wantCode: "INVALID_STATUS"},
			{name: "storage_failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				svc := newMockService(t)
				svc.EXPECT().List(gomock.Any(), suggestions.Filter{Status: domain.StatusPending}).Return(nil, tc.err)

				rec := serve(t, newRouter(svc), adminRequest(t, http.MethodGet, "/suggestions?status=pending", "", 99))
				assert.Empty(t, diffError(t, rec, tc.wantStatus, tc.wantCode), "GET /suggestions with service error %v", tc.err)
			})
		}
	})

	t.Run("success", func(t *testing.T) {
		tests := []struct {
			name        string
			suggestions []domain.Suggestion
			want        string
		}{
			{name: "no_suggestions", suggestions: []domain.Suggestion{}, want: `[]`},
			{
				name:        "suggestions_in_given_order",
				suggestions: []domain.Suggestion{linSuggestion(), linApproved()},
				want:        `[` + linSuggestionJSON + `, ` + linApprovedJSON + `]`,
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				svc := newMockService(t)
				svc.EXPECT().List(gomock.Any(), suggestions.Filter{}).Return(tc.suggestions, nil)

				rec := serve(t, newRouter(svc), adminRequest(t, http.MethodGet, "/suggestions", "", 99))
				assert.Empty(t, diffData(t, rec, http.StatusOK, tc.want), "GET /suggestions")
			})
		}
	})
}

// invalidIDs are suggestion paths whose id is not a positive 64-bit integer.
var invalidIDs = []struct {
	name string
	id   string
}{
	{name: "not_a_number", id: "abc"},
	{name: "fractional", id: "1.5"},
	{name: "zero", id: "0"},
	{name: "negative", id: "-1"},
	{name: "overflows_64_bits", id: "99999999999999999999"},
}

func TestHandler_Show(t *testing.T) {
	t.Run("rejects_invalid_id", func(t *testing.T) {
		for _, tc := range invalidIDs {
			t.Run(tc.name, func(t *testing.T) {
				// An invalid id must never reach the service.
				svc := newMockService(t)
				rec := serve(t, newRouter(svc), adminRequest(t, http.MethodGet, "/suggestions/"+tc.id, "", 99))
				assert.Empty(t, diffError(t, rec, http.StatusBadRequest, "INVALID_SUGGESTION_ID"), "GET /suggestions/%s", tc.id)
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
			{name: "unknown_suggestion", err: domain.ErrSuggestionNotFound, wantStatus: http.StatusNotFound, wantCode: "SUGGESTION_NOT_FOUND"},
			{name: "wrapped_unknown_suggestion", err: fmt.Errorf("get: %w", domain.ErrSuggestionNotFound), wantStatus: http.StatusNotFound, wantCode: "SUGGESTION_NOT_FOUND"},
			{name: "storage_failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				svc := newMockService(t)
				svc.EXPECT().Fetch(gomock.Any(), int64(3)).Return(nil, tc.err)

				rec := serve(t, newRouter(svc), adminRequest(t, http.MethodGet, "/suggestions/3", "", 99))
				assert.Empty(t, diffError(t, rec, tc.wantStatus, tc.wantCode), "GET /suggestions/3 with service error %v", tc.err)
			})
		}
	})

	t.Run("success", func(t *testing.T) {
		svc := newMockService(t)
		svc.EXPECT().Fetch(gomock.Any(), int64(3)).Return(returnOne(linSuggestion()), nil)

		rec := serve(t, newRouter(svc), adminRequest(t, http.MethodGet, "/suggestions/3", "", 99))
		assert.Empty(t, diffData(t, rec, http.StatusOK, linSuggestionJSON), "GET /suggestions/3")
	})
}

// approveBody is the final place an admin approves a suggestion as. It has the
// same shape as the body of POST /places.
const approveBody = `{"name":"Warung Bu Ani","sub_district_id":1,"category_id":2,` +
	`"location":{"type":"Point","coordinates":[106.8,-6.2]}}`

func TestHandler_Approve(t *testing.T) {
	t.Run("rejects_invalid_id", func(t *testing.T) {
		for _, tc := range invalidIDs {
			t.Run(tc.name, func(t *testing.T) {
				// An invalid id must never reach the service.
				svc := newMockService(t)
				rec := serve(t, newRouter(svc), adminRequest(t, http.MethodPost, "/suggestions/"+tc.id+"/approve", approveBody, 99))
				assert.Empty(t, diffError(t, rec, http.StatusBadRequest, "INVALID_SUGGESTION_ID"), "POST /suggestions/%s/approve", tc.id)
			})
		}
	})

	t.Run("rejects_malformed_bodies", func(t *testing.T) {
		const prefix = `{"name":"Warung","sub_district_id":1,"category_id":2,"location":`

		tests := []struct {
			name string
			body string
		}{
			{name: "empty_body", body: ``},
			{name: "body_is_null", body: `null`},
			{name: "truncated_JSON", body: `{"name":"Warung"`},
			{name: "array_instead_of_object", body: `[]`},
			{name: "sub_district_id_is_a_string", body: `{"name":"Warung","sub_district_id":"one","category_id":2}`},
			{name: "location_is_a_line", body: prefix + `{"type":"LineString","coordinates":[[106.8,-6.2],[106.9,-6.3]]}}`},
			{name: "coordinates_are_swapped", body: prefix + `{"type":"Point","coordinates":[-6.2,106.8]}}`},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// A malformed body must never reach the service.
				svc := newMockService(t)
				rec := serve(t, newRouter(svc), adminRequest(t, http.MethodPost, "/suggestions/3/approve", tc.body, 99))
				assert.Empty(t, diffError(t, rec, http.StatusBadRequest, "INVALID_BODY"), "POST /suggestions/3/approve %s", tc.body)
			})
		}
	})

	t.Run("without_user", func(t *testing.T) {
		// Without a user there is no reviewer to record.
		svc := newMockService(t)
		r := chi.NewRouter()
		r.Post("/suggestions/{id}/approve", suggestions.NewHandler(svc).Approve)

		rec := serve(t, r, jsonRequest(http.MethodPost, "/suggestions/3/approve", approveBody))
		assert.Empty(t, diffError(t, rec, http.StatusUnauthorized, "UNAUTHORIZED"), "POST /suggestions/3/approve without user")
	})

	t.Run("passes_place_to_service", func(t *testing.T) {
		tests := []struct {
			name string
			body string
			want *domain.Place
		}{
			{
				name: "required_fields_only",
				body: approveBody,
				want: &domain.Place{
					Name: "Warung Bu Ani", SubDistrictID: 1, CategoryID: 2,
					Location: postgis.MustPoint(106.8, -6.2),
				},
			},
			{
				name: "every_field",
				body: `{"name":"Warung Bu Ani","sub_district_id":1,"category_id":2,` +
					`"location":{"type":"Point","coordinates":[106.8,-6.2]},` +
					`"address":"Jl. Sabang 12","operating_hours":"07:00-21:00",` +
					`"estimated_price_range":"Rp15.000-30.000","short_description":"Masakan Padang",` +
					`"attributes":{"halal":true}}`,
				want: &domain.Place{
					Name: "Warung Bu Ani", SubDistrictID: 1, CategoryID: 2,
					Location:            postgis.MustPoint(106.8, -6.2),
					Address:             new("Jl. Sabang 12"),
					OperatingHours:      new("07:00-21:00"),
					EstimatedPriceRange: new("Rp15.000-30.000"),
					ShortDescription:    new("Masakan Padang"),
					Attributes:          json.RawMessage(`{"halal":true}`),
				},
			},
			{
				name: "missing_location_left_to_service",
				body: `{"name":"Warung Bu Ani","sub_district_id":1,"category_id":2}`,
				want: &domain.Place{Name: "Warung Bu Ani", SubDistrictID: 1, CategoryID: 2},
			},
			{
				name: "client_id_and_unknown_keys_ignored",
				body: `{"name":"Warung Bu Ani","sub_district_id":1,"category_id":2,` +
					`"location":{"type":"Point","coordinates":[106.8,-6.2]},"id":77,"created_at":"yesterday"}`,
				want: &domain.Place{
					Name: "Warung Bu Ani", SubDistrictID: 1, CategoryID: 2,
					Location: postgis.MustPoint(106.8, -6.2),
				},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// The suggestion comes from the path and the reviewer from the token.
				svc := newMockService(t)
				svc.EXPECT().Approve(gomock.Any(), int64(3), int64(99), tc.want).Return(returnOne(linApproved()), nil)

				rec := serve(t, newRouter(svc), adminRequest(t, http.MethodPost, "/suggestions/3/approve", tc.body, 99))
				assert.Equal(t, http.StatusOK, rec.Code, "POST /suggestions/3/approve; body: %s", rec.Body)
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
			{name: "unknown_suggestion", err: domain.ErrSuggestionNotFound, wantStatus: http.StatusNotFound, wantCode: "SUGGESTION_NOT_FOUND"},
			{name: "already_reviewed", err: domain.ErrSuggestionAlreadyReviewed, wantStatus: http.StatusConflict, wantCode: "ALREADY_REVIEWED"},
			{name: "wrapped_already_reviewed", err: fmt.Errorf("approve: %w", domain.ErrSuggestionAlreadyReviewed), wantStatus: http.StatusConflict, wantCode: "ALREADY_REVIEWED"},
			{name: "place_without_location", err: domain.ErrSuggestionLocationInvalid, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "blank_place_name", err: domain.ErrPlaceNameRequired, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "place_name_too_long", err: domain.ErrPlaceNameTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "place_address_too_long", err: domain.ErrPlaceAddressTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "place_operating_hours_too_long", err: domain.ErrPlaceOperatingHoursTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "place_price_range_too_long", err: domain.ErrPlaceEstimatedPriceRangeTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "place_outside_Indonesia", err: domain.ErrPlaceLocationInvalid, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "place_attributes_invalid", err: domain.ErrPlaceAttributesInvalid, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "wrapped_place_attributes_invalid", err: fmt.Errorf("%w: %q is required", domain.ErrPlaceAttributesInvalid, "contact"), wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "unknown_sub_district", err: domain.ErrSubDistrictNotFound, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "unknown_category", err: domain.ErrCategoryNotFound, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "storage_failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				svc := newMockService(t)
				svc.EXPECT().Approve(gomock.Any(), int64(3), int64(99), gomock.Any()).Return(nil, tc.err)

				rec := serve(t, newRouter(svc), adminRequest(t, http.MethodPost, "/suggestions/3/approve", approveBody, 99))
				assert.Empty(t, diffError(t, rec, tc.wantStatus, tc.wantCode), "POST /suggestions/3/approve with service error %v", tc.err)
			})
		}
	})

	t.Run("success", func(t *testing.T) {
		svc := newMockService(t)
		svc.EXPECT().Approve(gomock.Any(), int64(3), int64(99), gomock.Any()).Return(returnOne(linApproved()), nil)

		rec := serve(t, newRouter(svc), adminRequest(t, http.MethodPost, "/suggestions/3/approve", approveBody, 99))
		assert.Empty(t, diffData(t, rec, http.StatusOK, linApprovedJSON), "POST /suggestions/3/approve")
	})
}

func TestHandler_Reject(t *testing.T) {
	t.Run("rejects_invalid_id", func(t *testing.T) {
		for _, tc := range invalidIDs {
			t.Run(tc.name, func(t *testing.T) {
				// An invalid id must never reach the service.
				svc := newMockService(t)
				rec := serve(t, newRouter(svc), adminRequest(t, http.MethodPost, "/suggestions/"+tc.id+"/reject", `{}`, 99))
				assert.Empty(t, diffError(t, rec, http.StatusBadRequest, "INVALID_SUGGESTION_ID"), "POST /suggestions/%s/reject", tc.id)
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
			{name: "truncated_JSON", body: `{"reason":`},
			{name: "array_instead_of_object", body: `[]`},
			{name: "reason_is_a_number", body: `{"reason":5}`},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// A malformed body must never reach the service.
				svc := newMockService(t)
				rec := serve(t, newRouter(svc), adminRequest(t, http.MethodPost, "/suggestions/3/reject", tc.body, 99))
				assert.Empty(t, diffError(t, rec, http.StatusBadRequest, "INVALID_BODY"), "POST /suggestions/3/reject %s", tc.body)
			})
		}
	})

	t.Run("without_user", func(t *testing.T) {
		// Without a user there is no reviewer to record.
		svc := newMockService(t)
		r := chi.NewRouter()
		r.Post("/suggestions/{id}/reject", suggestions.NewHandler(svc).Reject)

		rec := serve(t, r, jsonRequest(http.MethodPost, "/suggestions/3/reject", `{}`))
		assert.Empty(t, diffError(t, rec, http.StatusUnauthorized, "UNAUTHORIZED"), "POST /suggestions/3/reject without user")
	})

	t.Run("passes_reason_to_service", func(t *testing.T) {
		tests := []struct {
			name string
			body string
			want *string
		}{
			{name: "reason", body: `{"reason":"Sudah tutup"}`, want: new("Sudah tutup")},
			{name: "reason_passed_on_untrimmed", body: `{"reason":"  Sudah tutup  "}`, want: new("  Sudah tutup  ")},
			{name: "no_reason", body: `{}`, want: nil},
			{name: "null_reason", body: `{"reason":null}`, want: nil},
			{name: "unknown_keys_ignored", body: `{"reason":"Sudah tutup","status":"approved"}`, want: new("Sudah tutup")},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// The suggestion comes from the path and the reviewer from the token.
				svc := newMockService(t)
				svc.EXPECT().Reject(gomock.Any(), int64(3), int64(99), tc.want).Return(returnOne(linRejected()), nil)

				rec := serve(t, newRouter(svc), adminRequest(t, http.MethodPost, "/suggestions/3/reject", tc.body, 99))
				assert.Equal(t, http.StatusOK, rec.Code, "POST /suggestions/3/reject %s; body: %s", tc.body, rec.Body)
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
			{name: "unknown_suggestion", err: domain.ErrSuggestionNotFound, wantStatus: http.StatusNotFound, wantCode: "SUGGESTION_NOT_FOUND"},
			{name: "already_reviewed", err: domain.ErrSuggestionAlreadyReviewed, wantStatus: http.StatusConflict, wantCode: "ALREADY_REVIEWED"},
			{name: "reason_too_long", err: domain.ErrSuggestionReasonTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "storage_failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				svc := newMockService(t)
				svc.EXPECT().Reject(gomock.Any(), int64(3), int64(99), new("Sudah tutup")).Return(nil, tc.err)

				rec := serve(t, newRouter(svc), adminRequest(t, http.MethodPost, "/suggestions/3/reject", `{"reason":"Sudah tutup"}`, 99))
				assert.Empty(t, diffError(t, rec, tc.wantStatus, tc.wantCode), "POST /suggestions/3/reject with service error %v", tc.err)
			})
		}
	})

	t.Run("success", func(t *testing.T) {
		svc := newMockService(t)
		svc.EXPECT().Reject(gomock.Any(), int64(3), int64(99), new("Sudah tutup")).Return(returnOne(linRejected()), nil)

		rec := serve(t, newRouter(svc), adminRequest(t, http.MethodPost, "/suggestions/3/reject", `{"reason":"Sudah tutup"}`, 99))
		assert.Empty(t, diffData(t, rec, http.StatusOK, linRejectedJSON), "POST /suggestions/3/reject")
	})
}

func TestHandler_InternalErrorsDoNotLeakDetails(t *testing.T) {
	dbErr := errors.New(`pq: relation "place_suggestions" does not exist`)

	tests := []struct {
		name string
		req  func(t *testing.T) *http.Request
		fail func(svc *mock.MockService)
	}{
		{
			name: "create_suggestion",
			req: func(t *testing.T) *http.Request {
				return authRequest(t, http.MethodPost, "/suggestions", minimalBody, 7)
			},
			fail: func(s *mock.MockService) {
				s.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, dbErr)
			},
		},
		{
			name: "list_my_suggestions",
			req: func(t *testing.T) *http.Request {
				return authRequest(t, http.MethodGet, "/suggestions/mine", "", 7)
			},
			fail: func(s *mock.MockService) {
				s.EXPECT().ListByUser(gomock.Any(), int64(7)).Return(nil, dbErr)
			},
		},
		{
			name: "list_suggestions",
			req: func(t *testing.T) *http.Request {
				return adminRequest(t, http.MethodGet, "/suggestions", "", 99)
			},
			fail: func(s *mock.MockService) {
				s.EXPECT().List(gomock.Any(), suggestions.Filter{}).Return(nil, dbErr)
			},
		},
		{
			name: "show_suggestion",
			req: func(t *testing.T) *http.Request {
				return adminRequest(t, http.MethodGet, "/suggestions/3", "", 99)
			},
			fail: func(s *mock.MockService) {
				s.EXPECT().Fetch(gomock.Any(), int64(3)).Return(nil, dbErr)
			},
		},
		{
			name: "approve_suggestion",
			req: func(t *testing.T) *http.Request {
				return adminRequest(t, http.MethodPost, "/suggestions/3/approve", approveBody, 99)
			},
			fail: func(s *mock.MockService) {
				s.EXPECT().Approve(gomock.Any(), int64(3), int64(99), gomock.Any()).Return(nil, dbErr)
			},
		},
		{
			name: "reject_suggestion",
			req: func(t *testing.T) *http.Request {
				return adminRequest(t, http.MethodPost, "/suggestions/3/reject", `{}`, 99)
			},
			fail: func(s *mock.MockService) {
				s.EXPECT().Reject(gomock.Any(), int64(3), int64(99), nil).Return(nil, dbErr)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := newMockService(t)
			tc.fail(svc)

			rec := serve(t, newRouter(svc), tc.req(t))
			assert.Empty(t, diffError(t, rec, http.StatusInternalServerError, "INTERNAL_ERROR"), "response")
			assert.NotContains(t, rec.Body.String(), "pq:")
			assert.NotContains(t, rec.Body.String(), "relation")
		})
	}
}
