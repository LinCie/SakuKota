package places_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"sakukota-go/internal/domain"
	"sakukota-go/internal/module/places"
	"sakukota-go/internal/module/places/mock"
	"sakukota-go/internal/pkg/postgis"
	"sakukota-go/internal/storage"
)

// newRouter serves the place handlers on the paths the API uses for them, so
// URL parameters resolve as they do in production.
func newRouter(svc *mock.MockService) http.Handler {
	h := places.NewHandler(svc, testMediaURL)
	r := chi.NewRouter()
	r.Route("/places", func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Post("/bulk", h.CreateMany)
		r.Get("/{id}", h.Show)
		r.Put("/{id}", h.Replace)
		r.Delete("/{id}", h.Delete)
		r.Post("/{id}/medias/upload-url", h.CreateMediaUpload)
		r.Post("/{id}/medias", h.AddMedia)
		r.Delete("/{id}/medias/{mediaId}", h.DeleteMedia)
	})
	r.Route("/categories", func(r chi.Router) {
		r.Get("/", h.ListCategories)
		r.Get("/{id}", h.ShowCategory)
	})
	return r
}

// newService returns a mock service that fails the test when it ends if an
// expected call did not happen, or when an unexpected one does.
func newService(t *testing.T) *mock.MockService {
	t.Helper()

	return mock.NewMockService(gomock.NewController(t))
}

// testMediaURL stands in for the storage's public URLs.
func testMediaURL(key string) string {
	return "https://img.test/" + key
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

// errorResponse is what a test checks of an error response: its status, whether
// it is JSON, its code, and whether it has a message.
type errorResponse struct {
	Status     int
	JSON       bool
	Code       string
	HasMessage bool
	// Body is set only when the response is not the { code, message } envelope.
	Body string
}

// wantError returns the error response with the status and code.
func wantError(status int, code string) errorResponse {
	return errorResponse{Status: status, JSON: true, Code: code, HasMessage: true}
}

// gotError describes rec as an error response.
func gotError(rec *httptest.ResponseRecorder) errorResponse {
	got := errorResponse{
		Status: rec.Code,
		JSON:   strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json"),
	}

	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		got.Body = rec.Body.String()
		return got
	}
	got.Code = body.Code
	got.HasMessage = body.Message != ""

	return got
}

// responseData returns the JSON under the success envelope's "data" key, or
// the whole body if it is not the envelope.
func responseData(rec *httptest.ResponseRecorder) string {
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil || env.Data == nil {
		return rec.Body.String()
	}

	return string(env.Data)
}

// kopiCategory is the category of the places the handler tests return.
var kopiCategory = domain.Category{ID: 2, Name: "Kopi", Slug: "kopi", Code: "K", Label: "Kopi", Color: "plum"}

const kopiCategoryJSON = `{"id": 2, "name": "Kopi", "slug": "kopi", "code": "K", "label": "Kopi", "color": "plum"}`

// errorCase is a request whose failure the handler maps to a response.
type errorCase struct {
	name       string
	err        error
	wantStatus int
	wantCode   string
}

func TestHandler_List(t *testing.T) {
	t.Run("malformed_filters", func(t *testing.T) {
		tests := []struct {
			name     string
			target   string
			wantCode string
		}{
			{name: "sub_district_not_number", target: "/places?sub_district_id=abc", wantCode: "INVALID_SUB_DISTRICT_ID"},
			{name: "sub_district_fractional", target: "/places?sub_district_id=1.5", wantCode: "INVALID_SUB_DISTRICT_ID"},
			{name: "sub_district_zero", target: "/places?sub_district_id=0", wantCode: "INVALID_SUB_DISTRICT_ID"},
			{name: "sub_district_negative", target: "/places?sub_district_id=-3", wantCode: "INVALID_SUB_DISTRICT_ID"},
			{name: "sub_district_overflow", target: "/places?sub_district_id=99999999999999999999", wantCode: "INVALID_SUB_DISTRICT_ID"},
			{name: "category_non_number", target: "/places?category_id=1,x", wantCode: "INVALID_CATEGORY_ID"},
			{name: "category_empty_element", target: "/places?category_id=1,,2", wantCode: "INVALID_CATEGORY_ID"},
			{name: "category_trailing_comma", target: "/places?category_id=1,", wantCode: "INVALID_CATEGORY_ID"},
			{name: "category_zero", target: "/places?category_id=0", wantCode: "INVALID_CATEGORY_ID"},
			{name: "category_negative", target: "/places?category_id=-1", wantCode: "INVALID_CATEGORY_ID"},
			{name: "category_zero_element", target: "/places?category_id=1,0", wantCode: "INVALID_CATEGORY_ID"},
			{name: "limit_not_number", target: "/places?limit=abc", wantCode: "INVALID_LIMIT"},
			{name: "limit_fractional", target: "/places?limit=2.5", wantCode: "INVALID_LIMIT"},
			{name: "limit_zero", target: "/places?limit=0", wantCode: "INVALID_LIMIT"},
			{name: "limit_negative", target: "/places?limit=-3", wantCode: "INVALID_LIMIT"},
			{name: "limit_overflow", target: "/places?limit=99999999999999999999", wantCode: "INVALID_LIMIT"},
			{name: "offset_not_number", target: "/places?offset=abc", wantCode: "INVALID_OFFSET"},
			{name: "offset_fractional", target: "/places?offset=2.5", wantCode: "INVALID_OFFSET"},
			{name: "offset_negative", target: "/places?offset=-3", wantCode: "INVALID_OFFSET"},
			{name: "offset_overflow", target: "/places?offset=99999999999999999999", wantCode: "INVALID_OFFSET"},
			{name: "bbox_not_numbers", target: "/places?bbox=abc", wantCode: "INVALID_BBOX"},
			{name: "bbox_three_values", target: "/places?bbox=106.8,-6.3,106.9", wantCode: "INVALID_BBOX"},
			{name: "bbox_five_values", target: "/places?bbox=106.8,-6.3,106.9,-6.1,1", wantCode: "INVALID_BBOX"},
			{name: "bbox_empty_element", target: "/places?bbox=106.8,,106.9,-6.1", wantCode: "INVALID_BBOX"},
			{name: "bbox_non_number", target: "/places?bbox=106.8,x,106.9,-6.1", wantCode: "INVALID_BBOX"},
			{name: "bbox_nan", target: "/places?bbox=NaN,-6.3,106.9,-6.1", wantCode: "INVALID_BBOX"},
			{name: "bbox_infinity", target: "/places?bbox=106.8,-6.3,Inf,-6.1", wantCode: "INVALID_BBOX"},
			{name: "bbox_west_past_east", target: "/places?bbox=106.9,-6.3,106.8,-6.1", wantCode: "INVALID_BBOX"},
			{name: "bbox_south_past_north", target: "/places?bbox=106.8,-6.1,106.9,-6.3", wantCode: "INVALID_BBOX"},
			{name: "bbox_longitude_range", target: "/places?bbox=-181,-6.3,106.9,-6.1", wantCode: "INVALID_BBOX"},
			{name: "bbox_latitude_range", target: "/places?bbox=106.8,-91,106.9,-6.1", wantCode: "INVALID_BBOX"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// No calls are set up: a malformed filter must never reach the service.
				rec := serve(t, newRouter(newService(t)), jsonRequest(http.MethodGet, tc.target, ""))
				assert.Equal(t, wantError(http.StatusBadRequest, tc.wantCode), gotError(rec), "GET %s", tc.target)
			})
		}
	})

	t.Run("passes_filter", func(t *testing.T) {
		tests := []struct {
			name   string
			target string
			want   places.Filter
		}{
			{name: "none", target: "/places", want: places.Filter{}},
			{name: "query_trimmed", target: "/places?q=%20%20kopi%20%20", want: places.Filter{Query: "kopi"}},
			{name: "blank_query", target: "/places?q=%20%20%20", want: places.Filter{}},
			{name: "empty_parameters", target: "/places?sub_district_id=&category_id=", want: places.Filter{}},
			{name: "all_together", target: "/places?sub_district_id=7&category_id=1,2&q=kopi",
				want: places.Filter{SubDistrictID: 7, CategoryIDs: []int64{1, 2}, Query: "kopi"}},
			{name: "limit", target: "/places?limit=25", want: places.Filter{Limit: 25}},
			{name: "empty_limit", target: "/places?limit=", want: places.Filter{}},
			{name: "offset", target: "/places?limit=25&offset=50", want: places.Filter{Limit: 25, Offset: 50}},
			{name: "zero_offset", target: "/places?offset=0", want: places.Filter{}},
			{name: "empty_offset", target: "/places?offset=", want: places.Filter{}},
			{name: "all_with_limit", target: "/places?sub_district_id=7&category_id=1,2&q=kopi&limit=25",
				want: places.Filter{SubDistrictID: 7, CategoryIDs: []int64{1, 2}, Query: "kopi", Limit: 25}},
			{name: "bbox", target: "/places?bbox=106.8,-6.3,106.9,-6.1",
				want: places.Filter{Bounds: &places.BoundingBox{MinLng: 106.8, MinLat: -6.3, MaxLng: 106.9, MaxLat: -6.1}}},
			{name: "empty_bbox", target: "/places?bbox=", want: places.Filter{}},
			{name: "bbox_with_limit", target: "/places?bbox=106.8,-6.3,106.9,-6.1&limit=25",
				want: places.Filter{Bounds: &places.BoundingBox{MinLng: 106.8, MinLat: -6.3, MaxLng: 106.9, MaxLat: -6.1}, Limit: 25}},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				svc := newService(t)
				svc.EXPECT().List(gomock.Any(), tc.want).Return([]domain.Place{}, 0, nil)

				rec := serve(t, newRouter(svc), jsonRequest(http.MethodGet, tc.target, ""))
				assert.Equal(t, http.StatusOK, rec.Code, "GET %s status; body: %s", rec.Body)
				assert.JSONEq(t, `{"places": [], "total": 0}`, responseData(rec), "GET %s", tc.target)
			})
		}
	})

	t.Run("service_errors", func(t *testing.T) {
		tests := []errorCase{
			{name: "unknown_sub_district", err: domain.ErrSubDistrictNotFound, wantStatus: http.StatusNotFound, wantCode: "SUB_DISTRICT_NOT_FOUND"},
			{name: "unknown_category", err: domain.ErrCategoryNotFound, wantStatus: http.StatusNotFound, wantCode: "CATEGORY_NOT_FOUND"},
			{name: "wrapped_unknown_category", err: fmt.Errorf("filter: %w", domain.ErrCategoryNotFound), wantStatus: http.StatusNotFound, wantCode: "CATEGORY_NOT_FOUND"},
			{name: "storage_failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				svc := newService(t)
				svc.EXPECT().List(gomock.Any(), places.Filter{SubDistrictID: 7}).Return(nil, 0, tc.err)

				rec := serve(t, newRouter(svc), jsonRequest(http.MethodGet, "/places?sub_district_id=7", ""))
				assert.Equal(t, wantError(tc.wantStatus, tc.wantCode), gotError(rec), "GET /places with error %v", tc.err)
			})
		}
	})

	t.Run("response_shape", func(t *testing.T) {
		svc := newService(t)
		svc.EXPECT().List(gomock.Any(), places.Filter{}).Return([]domain.Place{{
			ID:            3,
			SubDistrictID: 1,
			Name:          "Kopi Tubruk",
			Attributes:    json.RawMessage(`{}`),
			Category:      kopiCategory,
		}}, 12, nil)

		// total counts every match, not just the places in this response.
		rec := serve(t, newRouter(svc), jsonRequest(http.MethodGet, "/places", ""))
		want := `{"total": 12, "places": [{
			"id": 3,
			"sub_district_id": 1,
			"name": "Kopi Tubruk",
			"address": null,
			"location": null,
			"operating_hours": null,
			"estimated_price_range": null,
			"short_description": null,
			"attributes": {},
			"rating_average": null,
			"rating_count": 0,
			"medias": [],
			"category": ` + kopiCategoryJSON + `
		}]}`
		assert.Equal(t, http.StatusOK, rec.Code, "GET /places status; body: %s", rec.Body)
		assert.JSONEq(t, want, responseData(rec), "GET /places")
	})
}

func TestHandler_ListCategories(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tests := []struct {
			name       string
			categories []domain.Category
			want       string
		}{
			{name: "none", categories: []domain.Category{}, want: `[]`},
			{
				name:       "without_fields",
				categories: []domain.Category{{ID: 1, Name: "Warung", Slug: "warung", Code: "W", Label: "Warung", Color: "chili"}},
				want:       `[{"id":1,"name":"Warung","slug":"warung","code":"W","label":"Warung","color":"chili","fields":[]}]`,
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				svc := newService(t)
				svc.EXPECT().ListCategories(gomock.Any()).Return(tc.categories, nil)

				rec := serve(t, newRouter(svc), jsonRequest(http.MethodGet, "/categories", ""))
				assert.Equal(t, http.StatusOK, rec.Code, "GET /categories status; body: %s", rec.Body)
				assert.JSONEq(t, tc.want, responseData(rec), "GET /categories")
			})
		}
	})
}

func TestHandler_Create(t *testing.T) {
	t.Run("malformed_bodies", func(t *testing.T) {
		const prefix = `{"name":"Kopi","sub_district_id":1,"category_id":2,"location":`

		tests := []struct {
			name string
			body string
		}{
			{name: "empty", body: ``},
			{name: "null", body: `null`},
			{name: "truncated", body: `{"name":"Kopi"`},
			{name: "sub_district_string", body: `{"name":"Kopi","sub_district_id":"one","category_id":2}`},
			{name: "location_line", body: prefix + `{"type":"LineString","coordinates":[[106.8,-6.2],[106.9,-6.3]]}}`},
			{name: "coordinates_swapped", body: prefix + `{"type":"Point","coordinates":[-6.2,106.8]}}`},
			{name: "latitude_above_90", body: prefix + `{"type":"Point","coordinates":[106.8,91]}}`},
			{name: "longitude_above_180", body: prefix + `{"type":"Point","coordinates":[181,-6.2]}}`},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				rec := serve(t, newRouter(newService(t)), jsonRequest(http.MethodPost, "/places", tc.body))
				assert.Equal(t, wantError(http.StatusBadRequest, "INVALID_BODY"), gotError(rec), "POST /places %s", tc.body)
			})
		}
	})

	t.Run("service_errors", func(t *testing.T) {
		tests := []errorCase{
			{name: "blank_name", err: domain.ErrPlaceNameRequired, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "name_too_long", err: domain.ErrPlaceNameTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "address_too_long", err: domain.ErrPlaceAddressTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "hours_too_long", err: domain.ErrPlaceOperatingHoursTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "price_too_long", err: domain.ErrPlaceEstimatedPriceRangeTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "unknown_sub_district", err: domain.ErrSubDistrictNotFound, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "unknown_category", err: domain.ErrCategoryNotFound, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "invalid_attributes", err: domain.ErrPlaceAttributesInvalid, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "wrapped_attributes", err: fmt.Errorf("%w: %q is required", domain.ErrPlaceAttributesInvalid, "contact"), wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "storage_failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				svc := newService(t)
				svc.EXPECT().Create(gomock.Any(), &domain.Place{Name: "Kopi", SubDistrictID: 1, CategoryID: 2}).Return(nil, tc.err)

				rec := serve(t, newRouter(svc), jsonRequest(http.MethodPost, "/places",
					`{"name":"Kopi","sub_district_id":1,"category_id":2}`))
				assert.Equal(t, wantError(tc.wantStatus, tc.wantCode), gotError(rec), "POST /places with error %v", tc.err)
			})
		}
	})

	t.Run("success", func(t *testing.T) {
		svc := newService(t)

		var got *domain.Place
		svc.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, p *domain.Place) (*domain.Place, error) {
			got = p
			return &domain.Place{
				ID:            42,
				SubDistrictID: p.SubDistrictID,
				Name:          p.Name,
				Location:      p.Location,
				Attributes:    json.RawMessage(`{}`),
				Category:      kopiCategory,
			}, nil
		})

		body := `{"name":"Kopi Senja","sub_district_id":1,"category_id":2,` +
			`"location":{"type":"Point","coordinates":[106.8,-6.2]},"id":77,"created_at":"yesterday"}`
		rec := serve(t, newRouter(svc), jsonRequest(http.MethodPost, "/places", body))

		require.NotNil(t, got, "POST /places must call the service")
		assert.Zero(t, got.ID, "POST /places must ignore a client-sent id")
		assert.Nil(t, got.Address, "POST /places address passed to the service")
		assert.Nil(t, got.Attributes, "POST /places attributes passed to the service")
		require.NotNil(t, got.Location, "POST /places must pass the location to the service")
		assert.Equal(t, 106.8, got.Location.X(), "POST /places longitude passed to the service")
		assert.Equal(t, -6.2, got.Location.Y(), "POST /places latitude passed to the service")

		want := `{
			"id": 42,
			"sub_district_id": 1,
			"name": "Kopi Senja",
			"address": null,
			"location": {"type": "Point", "coordinates": [106.8, -6.2]},
			"operating_hours": null,
			"estimated_price_range": null,
			"short_description": null,
			"attributes": {},
			"rating_average": null,
			"rating_count": 0,
			"medias": [],
			"category": ` + kopiCategoryJSON + `
		}`
		assert.Equal(t, http.StatusCreated, rec.Code, "POST /places status; body: %s", rec.Body)
		assert.JSONEq(t, want, responseData(rec), "POST /places")
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

				svc := newService(t)
				svc.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(got context.Context, _ *domain.Place) (*domain.Place, error) {
					select {
					case <-got.Done():
						return nil, got.Err()
					case <-time.After(2 * time.Second):
						assert.Fail(t, "service was not given the request's context")
						return nil, errors.New("timed out")
					}
				})

				req := jsonRequest(http.MethodPost, "/places", `{"name":"Kopi","sub_district_id":1,"category_id":2}`).WithContext(ctx)
				rec := serve(t, newRouter(svc), req)
				assert.NotEqual(t, http.StatusCreated, rec.Code, "POST /places status for a request whose context had ended")
			})
		}
	})
}

func TestHandler_CreateMany(t *testing.T) {
	t.Run("malformed_bodies", func(t *testing.T) {
		tests := []struct {
			name string
			body string
		}{
			{name: "empty", body: ``},
			{name: "null", body: `null`},
			{name: "array", body: `[{"name":"Kopi","sub_district_id":1,"category_id":2}]`},
			{name: "places_object", body: `{"places":{"name":"Kopi"}}`},
			{name: "sub_district_string", body: `{"places":[{"name":"Kopi","sub_district_id":"one","category_id":2}]}`},
			{name: "location_line", body: `{"places":[{"name":"Kopi","sub_district_id":1,"category_id":2,` +
				`"location":{"type":"LineString","coordinates":[[106.8,-6.2],[106.9,-6.3]]}}]}`},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				rec := serve(t, newRouter(newService(t)), jsonRequest(http.MethodPost, "/places/bulk", tc.body))
				assert.Equal(t, wantError(http.StatusBadRequest, "INVALID_BODY"), gotError(rec), "POST /places/bulk %s", tc.body)
			})
		}
	})

	t.Run("service_errors", func(t *testing.T) {
		tests := []errorCase{
			{name: "batch_size", err: domain.ErrPlaceBatchSizeInvalid, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "blank_name", err: &domain.PlaceBatchError{Position: 2, Err: domain.ErrPlaceNameRequired}, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "invalid_attributes", err: &domain.PlaceBatchError{Position: 1, Err: domain.ErrPlaceAttributesInvalid}, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "unknown_sub_district", err: &domain.PlaceBatchError{Position: 2, Err: domain.ErrSubDistrictNotFound}, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "unknown_category", err: &domain.PlaceBatchError{Position: 1, Err: domain.ErrCategoryNotFound}, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "storage_failure", err: &domain.PlaceBatchError{Position: 1, Err: errors.New(`pq: relation "places" does not exist`)}, wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				svc := newService(t)
				svc.EXPECT().CreateMany(gomock.Any(), []domain.Place{
					{Name: "Kopi", SubDistrictID: 1, CategoryID: 2},
					{Name: "Teh", SubDistrictID: 1, CategoryID: 2},
				}).Return(nil, tc.err)

				rec := serve(t, newRouter(svc), jsonRequest(http.MethodPost, "/places/bulk",
					`{"places":[{"name":"Kopi","sub_district_id":1,"category_id":2},{"name":"Teh","sub_district_id":1,"category_id":2}]}`))
				assert.Equal(t, wantError(tc.wantStatus, tc.wantCode), gotError(rec), "POST /places/bulk with error %v", tc.err)
				assert.NotContains(t, rec.Body.String(), "pq:", "POST /places/bulk body must not show storage detail")
			})
		}
	})

	t.Run("empty_places_reach_service", func(t *testing.T) {
		svc := newService(t)
		svc.EXPECT().CreateMany(gomock.Any(), gomock.Len(0)).Return(nil, domain.ErrPlaceBatchSizeInvalid)

		rec := serve(t, newRouter(svc), jsonRequest(http.MethodPost, "/places/bulk", `{"places":[]}`))
		assert.Equal(t, wantError(http.StatusBadRequest, "VALIDATION_ERROR"), gotError(rec), "POST /places/bulk with no places")
	})

	t.Run("success", func(t *testing.T) {
		svc := newService(t)
		svc.EXPECT().CreateMany(gomock.Any(), []domain.Place{
			{Name: "Kopi Senja", SubDistrictID: 1, CategoryID: 2, Location: postgis.MustPoint(106.8, -6.2), Attributes: json.RawMessage(`{"halal":true}`)},
			{Name: "Kopi Pagi", SubDistrictID: 1, CategoryID: 2},
		}).DoAndReturn(func(_ context.Context, ps []domain.Place) ([]domain.Place, error) {
			created := make([]domain.Place, 0, len(ps))
			for i, p := range ps {
				p.ID = int64(41 + i)
				p.Category = kopiCategory
				if p.Attributes == nil {
					p.Attributes = json.RawMessage(`{}`)
				}
				created = append(created, p)
			}
			return created, nil
		})

		body := `{"places":[` +
			`{"name":"Kopi Senja","sub_district_id":1,"category_id":2,"location":{"type":"Point","coordinates":[106.8,-6.2]},"attributes":{"halal":true},"id":77},` +
			`{"name":"Kopi Pagi","sub_district_id":1,"category_id":2}` +
			`]}`
		rec := serve(t, newRouter(svc), jsonRequest(http.MethodPost, "/places/bulk", body))

		want := `[{
			"id": 41,
			"sub_district_id": 1,
			"name": "Kopi Senja",
			"address": null,
			"location": {"type": "Point", "coordinates": [106.8, -6.2]},
			"operating_hours": null,
			"estimated_price_range": null,
			"short_description": null,
			"attributes": {"halal": true},
			"rating_average": null,
			"rating_count": 0,
			"medias": [],
			"category": ` + kopiCategoryJSON + `
		}, {
			"id": 42,
			"sub_district_id": 1,
			"name": "Kopi Pagi",
			"address": null,
			"location": null,
			"operating_hours": null,
			"estimated_price_range": null,
			"short_description": null,
			"attributes": {},
			"rating_average": null,
			"rating_count": 0,
			"medias": [],
			"category": ` + kopiCategoryJSON + `
		}]`
		assert.Equal(t, http.StatusCreated, rec.Code, "POST /places/bulk status; body: %s", rec.Body)
		assert.JSONEq(t, want, responseData(rec), "POST /places/bulk")
	})
}

func TestHandler_Replace(t *testing.T) {
	t.Run("invalid_requests", func(t *testing.T) {
		const valid = `{"name":"Kopi","sub_district_id":1,"category_id":2}`

		tests := []struct {
			name     string
			target   string
			body     string
			wantCode string
		}{
			{name: "id_not_number", target: "/places/abc", body: valid, wantCode: "INVALID_PLACE_ID"},
			{name: "id_fractional", target: "/places/1.5", body: valid, wantCode: "INVALID_PLACE_ID"},
			{name: "id_zero", target: "/places/0", body: valid, wantCode: "INVALID_PLACE_ID"},
			{name: "id_negative", target: "/places/-1", body: valid, wantCode: "INVALID_PLACE_ID"},
			{name: "empty_body", target: "/places/1", body: ``, wantCode: "INVALID_BODY"},
			{name: "truncated", target: "/places/1", body: `{"name":"Kopi"`, wantCode: "INVALID_BODY"},
			{name: "array", target: "/places/1", body: `[]`, wantCode: "INVALID_BODY"},
			{name: "null", target: "/places/1", body: `null`, wantCode: "INVALID_BODY"},
			{name: "trailing_data", target: "/places/1", body: valid + ` trailing`, wantCode: "INVALID_BODY"},
			{name: "two_objects", target: "/places/1", body: valid + valid, wantCode: "INVALID_BODY"},
			{name: "name_number", target: "/places/1", body: `{"name":5,"sub_district_id":1,"category_id":2}`, wantCode: "INVALID_BODY"},
			{name: "sub_district_string", target: "/places/1", body: `{"name":"Kopi","sub_district_id":"one","category_id":2}`, wantCode: "INVALID_BODY"},
			{name: "location_line", target: "/places/1",
				body: `{"name":"Kopi","sub_district_id":1,"category_id":2,"location":{"type":"LineString","coordinates":[[106.8,-6.2],[106.9,-6.3]]}}`, wantCode: "INVALID_BODY"},
			{name: "coordinates_swapped", target: "/places/1",
				body: `{"name":"Kopi","sub_district_id":1,"category_id":2,"location":{"type":"Point","coordinates":[-6.2,106.8]}}`, wantCode: "INVALID_BODY"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// No calls are set up: an invalid request must never reach the service.
				rec := serve(t, newRouter(newService(t)), jsonRequest(http.MethodPut, tc.target, tc.body))
				assert.Equal(t, wantError(http.StatusBadRequest, tc.wantCode), gotError(rec), "PUT %s %s", tc.target, tc.body)
			})
		}
	})

	t.Run("passes_place", func(t *testing.T) {
		tests := []struct {
			name string
			body string
			want *domain.Place
		}{
			{name: "every_field",
				body: `{"sub_district_id":2,"category_id":3,"name":"  Kopi Senja  ","address":"Jl. Sudirman 9",
					"location":{"type":"Point","coordinates":[106.8,-6.2]},"operating_hours":"08:00-17:00",
					"estimated_price_range":"Rp20.000-50.000","short_description":"Kopi tubruk",
					"attributes":{"contact":"0812"}}`,
				want: &domain.Place{
					SubDistrictID:       2,
					CategoryID:          3,
					Name:                "  Kopi Senja  ",
					Address:             new("Jl. Sudirman 9"),
					Location:            postgis.MustPoint(106.8, -6.2),
					OperatingHours:      new("08:00-17:00"),
					EstimatedPriceRange: new("Rp20.000-50.000"),
					ShortDescription:    new("Kopi tubruk"),
					Attributes:          json.RawMessage(`{"contact":"0812"}`),
				}},
			{name: "left_out_keys_absent",
				body: `{"sub_district_id":2,"category_id":3,"name":"Kopi"}`,
				want: &domain.Place{SubDistrictID: 2, CategoryID: 3, Name: "Kopi"}},
			{name: "client_id_ignored",
				body: `{"id":99,"created_at":"yesterday","sub_district_id":2,"category_id":3,"name":"Kopi"}`,
				want: &domain.Place{SubDistrictID: 2, CategoryID: 3, Name: "Kopi"}},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				svc := newService(t)
				svc.EXPECT().Update(gomock.Any(), int64(9), tc.want).Return(&domain.Place{ID: 9}, nil)

				rec := serve(t, newRouter(svc), jsonRequest(http.MethodPut, "/places/9", tc.body))
				assert.Equal(t, http.StatusOK, rec.Code, "PUT /places/9 status; body: %s", rec.Body)
			})
		}
	})

	t.Run("service_errors", func(t *testing.T) {
		tests := []errorCase{
			{name: "unknown_place", err: domain.ErrPlaceNotFound, wantStatus: http.StatusNotFound, wantCode: "PLACE_NOT_FOUND"},
			{name: "wrapped_unknown_place", err: fmt.Errorf("update: %w", domain.ErrPlaceNotFound), wantStatus: http.StatusNotFound, wantCode: "PLACE_NOT_FOUND"},
			{name: "blank_name", err: domain.ErrPlaceNameRequired, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "name_too_long", err: domain.ErrPlaceNameTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "address_too_long", err: domain.ErrPlaceAddressTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "hours_too_long", err: domain.ErrPlaceOperatingHoursTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "price_too_long", err: domain.ErrPlaceEstimatedPriceRangeTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "unknown_sub_district", err: domain.ErrSubDistrictNotFound, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "unknown_category", err: domain.ErrCategoryNotFound, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "invalid_attributes", err: domain.ErrPlaceAttributesInvalid, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "wrapped_attributes", err: fmt.Errorf("%w: %q is required", domain.ErrPlaceAttributesInvalid, "contact"), wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "location_outside", err: domain.ErrPlaceLocationInvalid, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "storage_failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				svc := newService(t)
				svc.EXPECT().Update(gomock.Any(), int64(1), &domain.Place{SubDistrictID: 1, CategoryID: 2, Name: "Kopi"}).Return(nil, tc.err)

				rec := serve(t, newRouter(svc), jsonRequest(http.MethodPut, "/places/1", `{"name":"Kopi","sub_district_id":1,"category_id":2}`))
				assert.Equal(t, wantError(tc.wantStatus, tc.wantCode), gotError(rec), "PUT /places/1 with error %v", tc.err)
			})
		}
	})

	t.Run("success", func(t *testing.T) {
		svc := newService(t)
		svc.EXPECT().Update(gomock.Any(), int64(3), &domain.Place{SubDistrictID: 1, CategoryID: 2, Name: "Kopi Senja"}).Return(&domain.Place{
			ID:            3,
			SubDistrictID: 1,
			Name:          "Kopi Senja",
			Attributes:    json.RawMessage(`{}`),
			Category:      kopiCategory,
		}, nil)

		rec := serve(t, newRouter(svc), jsonRequest(http.MethodPut, "/places/3", `{"name":"Kopi Senja","sub_district_id":1,"category_id":2}`))
		want := `{
			"id": 3,
			"sub_district_id": 1,
			"name": "Kopi Senja",
			"address": null,
			"location": null,
			"operating_hours": null,
			"estimated_price_range": null,
			"short_description": null,
			"attributes": {},
			"rating_average": null,
			"rating_count": 0,
			"medias": [],
			"category": ` + kopiCategoryJSON + `
		}`
		assert.Equal(t, http.StatusOK, rec.Code, "PUT /places/3 status; body: %s", rec.Body)
		assert.JSONEq(t, want, responseData(rec), "PUT /places/3")
	})
}

func TestHandler_Delete(t *testing.T) {
	t.Run("invalid_id", func(t *testing.T) {
		tests := []struct {
			name   string
			target string
		}{
			{name: "not_number", target: "/places/abc"},
			{name: "fractional", target: "/places/1.5"},
			{name: "zero", target: "/places/0"},
			{name: "negative", target: "/places/-1"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// No calls are set up: an invalid id must never reach the service.
				rec := serve(t, newRouter(newService(t)), jsonRequest(http.MethodDelete, tc.target, ""))
				assert.Equal(t, wantError(http.StatusBadRequest, "INVALID_PLACE_ID"), gotError(rec), "DELETE %s", tc.target)
			})
		}
	})

	t.Run("service_errors", func(t *testing.T) {
		tests := []errorCase{
			{name: "unknown_place", err: domain.ErrPlaceNotFound, wantStatus: http.StatusNotFound, wantCode: "PLACE_NOT_FOUND"},
			{name: "wrapped_unknown_place", err: fmt.Errorf("delete: %w", domain.ErrPlaceNotFound), wantStatus: http.StatusNotFound, wantCode: "PLACE_NOT_FOUND"},
			{name: "storage_failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				svc := newService(t)
				svc.EXPECT().Delete(gomock.Any(), int64(4)).Return(tc.err)

				rec := serve(t, newRouter(svc), jsonRequest(http.MethodDelete, "/places/4", ""))
				assert.Equal(t, wantError(tc.wantStatus, tc.wantCode), gotError(rec), "DELETE /places/4 with error %v", tc.err)
			})
		}
	})

	t.Run("success", func(t *testing.T) {
		svc := newService(t)
		svc.EXPECT().Delete(gomock.Any(), int64(4)).Return(nil)

		rec := serve(t, newRouter(svc), jsonRequest(http.MethodDelete, "/places/4", ""))
		assert.Equal(t, http.StatusNoContent, rec.Code, "DELETE /places/4 status; body: %s", rec.Body)
		assert.Zero(t, rec.Body.Len(), "DELETE /places/4 body = %q, want empty", rec.Body)
	})
}

func TestHandler_Show(t *testing.T) {
	t.Run("invalid_id", func(t *testing.T) {
		tests := []struct {
			name   string
			target string
		}{
			{name: "not_number", target: "/places/abc"},
			{name: "fractional", target: "/places/1.5"},
			{name: "zero", target: "/places/0"},
			{name: "negative", target: "/places/-1"},
			{name: "overflow", target: "/places/99999999999999999999"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// No calls are set up: an invalid id must never reach the service.
				rec := serve(t, newRouter(newService(t)), jsonRequest(http.MethodGet, tc.target, ""))
				assert.Equal(t, wantError(http.StatusBadRequest, "INVALID_PLACE_ID"), gotError(rec), "GET %s", tc.target)
			})
		}
	})

	t.Run("service_errors", func(t *testing.T) {
		tests := []errorCase{
			{name: "unknown_place", err: domain.ErrPlaceNotFound, wantStatus: http.StatusNotFound, wantCode: "PLACE_NOT_FOUND"},
			{name: "wrapped_unknown_place", err: fmt.Errorf("get: %w", domain.ErrPlaceNotFound), wantStatus: http.StatusNotFound, wantCode: "PLACE_NOT_FOUND"},
			{name: "storage_failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				svc := newService(t)
				svc.EXPECT().Fetch(gomock.Any(), int64(3)).Return(nil, tc.err)

				rec := serve(t, newRouter(svc), jsonRequest(http.MethodGet, "/places/3", ""))
				assert.Equal(t, wantError(tc.wantStatus, tc.wantCode), gotError(rec), "GET /places/3 with error %v", tc.err)
			})
		}
	})

	t.Run("success", func(t *testing.T) {
		svc := newService(t)
		svc.EXPECT().Fetch(gomock.Any(), int64(3)).Return(&domain.Place{
			ID:            3,
			SubDistrictID: 1,
			CategoryID:    2,
			Name:          "Kopi Senja",
			Address:       new("Jl. Sudirman 9"),
			Location:      postgis.MustPoint(106.8, -6.2),
			Attributes:    json.RawMessage(`{"halal":true}`),
			Category:      kopiCategory,
		}, nil)

		rec := serve(t, newRouter(svc), jsonRequest(http.MethodGet, "/places/3", ""))
		want := `{
			"id": 3,
			"sub_district_id": 1,
			"name": "Kopi Senja",
			"address": "Jl. Sudirman 9",
			"location": {"type": "Point", "coordinates": [106.8, -6.2]},
			"operating_hours": null,
			"estimated_price_range": null,
			"short_description": null,
			"attributes": {"halal": true},
			"rating_average": null,
			"rating_count": 0,
			"medias": [],
			"category": ` + kopiCategoryJSON + `
		}`
		assert.Equal(t, http.StatusOK, rec.Code, "GET /places/3 status; body: %s", rec.Body)
		assert.JSONEq(t, want, responseData(rec), "GET /places/3")
	})

	t.Run("rating", func(t *testing.T) {
		tests := []struct {
			name        string
			count       int
			sum         int
			wantAverage string // as JSON
		}{
			{name: "unrated", count: 0, sum: 0, wantAverage: `null`},
			{name: "one_rating", count: 1, sum: 4, wantAverage: `4`},
			{name: "several_ratings", count: 4, sum: 17, wantAverage: `4.25`},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				svc := newService(t)
				svc.EXPECT().Fetch(gomock.Any(), int64(3)).Return(&domain.Place{
					ID:          3,
					Name:        "Kopi Senja",
					Attributes:  json.RawMessage(`{}`),
					RatingCount: tc.count,
					RatingSum:   tc.sum,
				}, nil)

				rec := serve(t, newRouter(svc), jsonRequest(http.MethodGet, "/places/3", ""))
				require.Equal(t, http.StatusOK, rec.Code, "GET /places/3 status; body: %s", rec.Body)

				var got struct {
					RatingAverage json.RawMessage `json:"rating_average"`
					RatingCount   int             `json:"rating_count"`
				}
				data := responseData(rec)
				require.NoError(t, json.Unmarshal([]byte(data), &got), "decode %s", data)
				assert.Equal(t, tc.wantAverage, string(got.RatingAverage), "GET /places/3 rating_average")
				assert.Equal(t, tc.count, got.RatingCount, "GET /places/3 rating_count")
			})
		}
	})

	t.Run("medias", func(t *testing.T) {
		svc := newService(t)
		svc.EXPECT().Fetch(gomock.Any(), int64(3)).Return(&domain.Place{
			ID:         3,
			Name:       "Kopi Senja",
			Attributes: json.RawMessage(`{}`),
			Medias: []domain.Media{
				{ID: 8, Key: "places/3/a.webp", ContentType: "image/webp", Size: 2048, Position: 1},
				{ID: 5, Key: "places/3/b.png", ContentType: "image/png", Size: 1024, Position: 2},
			},
		}, nil)

		rec := serve(t, newRouter(svc), jsonRequest(http.MethodGet, "/places/3", ""))
		require.Equal(t, http.StatusOK, rec.Code, "GET /places/3 status; body: %s", rec.Body)

		var got struct {
			Medias json.RawMessage `json:"medias"`
		}
		data := responseData(rec)
		require.NoError(t, json.Unmarshal([]byte(data), &got), "decode %s", data)

		want := `[
			{"id": 8, "url": "https://img.test/places/3/a.webp", "content_type": "image/webp", "size": 2048, "position": 1},
			{"id": 5, "url": "https://img.test/places/3/b.png", "content_type": "image/png", "size": 1024, "position": 2}
		]`
		assert.JSONEq(t, want, string(got.Medias), "GET /places/3 medias")
	})
}

func TestHandler_ShowCategory(t *testing.T) {
	t.Run("invalid_id", func(t *testing.T) {
		tests := []struct {
			name   string
			target string
		}{
			{name: "not_number", target: "/categories/abc"},
			{name: "fractional", target: "/categories/1.5"},
			{name: "zero", target: "/categories/0"},
			{name: "negative", target: "/categories/-1"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				rec := serve(t, newRouter(newService(t)), jsonRequest(http.MethodGet, tc.target, ""))
				assert.Equal(t, wantError(http.StatusBadRequest, "INVALID_CATEGORY_ID"), gotError(rec), "GET %s", tc.target)
			})
		}
	})

	t.Run("service_errors", func(t *testing.T) {
		tests := []errorCase{
			{name: "unknown_category", err: domain.ErrCategoryNotFound, wantStatus: http.StatusNotFound, wantCode: "CATEGORY_NOT_FOUND"},
			{name: "wrapped_unknown_category", err: fmt.Errorf("get: %w", domain.ErrCategoryNotFound), wantStatus: http.StatusNotFound, wantCode: "CATEGORY_NOT_FOUND"},
			{name: "storage_failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				svc := newService(t)
				svc.EXPECT().FetchCategory(gomock.Any(), int64(2)).Return(nil, tc.err)

				rec := serve(t, newRouter(svc), jsonRequest(http.MethodGet, "/categories/2", ""))
				assert.Equal(t, wantError(tc.wantStatus, tc.wantCode), gotError(rec), "GET /categories/2 with error %v", tc.err)
			})
		}
	})

	t.Run("success", func(t *testing.T) {
		tests := []struct {
			name     string
			category domain.Category
			want     string
		}{
			{
				name:     "without_fields",
				category: domain.Category{ID: 2, Name: "Kopi", Slug: "kopi", Code: "K", Label: "Kopi", Color: "plum"},
				want:     `{"id":2,"name":"Kopi","slug":"kopi","code":"K","label":"Kopi","color":"plum","fields":[]}`,
			},
			{
				name: "with_fields",
				category: domain.Category{ID: 2, Name: "Kopi", Slug: "kopi", Code: "K", Label: "Kopi", Color: "plum", Fields: []domain.CategoryField{
					{ID: 9, CategoryID: 2, Key: "halal", Label: "Halal", DataType: domain.FieldTypeBoolean, Required: true, SortOrder: 1},
				}},
				want: `{"id":2,"name":"Kopi","slug":"kopi","code":"K","label":"Kopi","color":"plum","fields":[
					{"id":9,"category_id":2,"key":"halal","label":"Halal","data_type":"boolean","required":true,"options":null,"sort_order":1}
				]}`,
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				svc := newService(t)
				svc.EXPECT().FetchCategory(gomock.Any(), int64(2)).Return(&tc.category, nil)

				rec := serve(t, newRouter(svc), jsonRequest(http.MethodGet, "/categories/2", ""))
				assert.Equal(t, http.StatusOK, rec.Code, "GET /categories/2 status; body: %s", rec.Body)
				assert.JSONEq(t, tc.want, responseData(rec), "GET /categories/2")
			})
		}
	})
}

func TestHandler_CreateMediaUpload(t *testing.T) {
	t.Run("invalid_requests", func(t *testing.T) {
		tests := []struct {
			name     string
			target   string
			body     string
			wantCode string
		}{
			{name: "id_not_number", target: "/places/abc/medias/upload-url", body: `{"content_type":"image/png"}`, wantCode: "INVALID_PLACE_ID"},
			{name: "id_zero", target: "/places/0/medias/upload-url", body: `{"content_type":"image/png"}`, wantCode: "INVALID_PLACE_ID"},
			{name: "body_not_json", target: "/places/3/medias/upload-url", body: `content_type=image/png`, wantCode: "INVALID_BODY"},
			{name: "body_null", target: "/places/3/medias/upload-url", body: `null`, wantCode: "INVALID_BODY"},
			{name: "body_array", target: "/places/3/medias/upload-url", body: `[]`, wantCode: "INVALID_BODY"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// No calls are set up: an invalid request must never reach the service.
				rec := serve(t, newRouter(newService(t)), jsonRequest(http.MethodPost, tc.target, tc.body))
				assert.Equal(t, wantError(http.StatusBadRequest, tc.wantCode), gotError(rec), "POST %s %s", tc.target, tc.body)
			})
		}
	})

	t.Run("service_errors", func(t *testing.T) {
		tests := []errorCase{
			{name: "unknown_place", err: domain.ErrPlaceNotFound, wantStatus: http.StatusNotFound, wantCode: "PLACE_NOT_FOUND"},
			{name: "type_not_allowed", err: storage.ErrInvalidMediaType, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "storage_failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				svc := newService(t)
				svc.EXPECT().MediaUploadURL(gomock.Any(), int64(3), "image/gif").Return(storage.Upload{}, tc.err)

				rec := serve(t, newRouter(svc), jsonRequest(http.MethodPost, "/places/3/medias/upload-url", `{"content_type":"image/gif"}`))
				assert.Equal(t, wantError(tc.wantStatus, tc.wantCode), gotError(rec), "POST /places/3/medias/upload-url with error %v", tc.err)
			})
		}
	})

	t.Run("success", func(t *testing.T) {
		svc := newService(t)
		svc.EXPECT().MediaUploadURL(gomock.Any(), int64(3), "image/webp").Return(storage.Upload{
			Key:       "uploads/places/3/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp",
			URL:       "http://localhost:3900/sakukota-images/uploads/places/3/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp?X-Amz-Signature=abc",
			Method:    "PUT",
			Headers:   map[string]string{"Content-Type": "image/webp"},
			ExpiresAt: time.Date(2026, 10, 31, 12, 5, 0, 0, time.UTC),
		}, nil)

		rec := serve(t, newRouter(svc), jsonRequest(http.MethodPost, "/places/3/medias/upload-url", `{"content_type":"image/webp"}`))
		want := `{
			"upload_key": "uploads/places/3/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp",
			"upload_url": "http://localhost:3900/sakukota-images/uploads/places/3/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp?X-Amz-Signature=abc",
			"method": "PUT",
			"headers": {"Content-Type": "image/webp"},
			"expires_at": "2026-10-31T12:05:00Z"
		}`
		assert.Equal(t, http.StatusOK, rec.Code, "POST /places/3/medias/upload-url status; body: %s", rec.Body)
		assert.JSONEq(t, want, responseData(rec), "POST /places/3/medias/upload-url")
	})
}

func TestHandler_AddMedia(t *testing.T) {
	t.Run("invalid_requests", func(t *testing.T) {
		tests := []struct {
			name     string
			target   string
			body     string
			wantCode string
		}{
			{name: "id_not_number", target: "/places/abc/medias", body: `{"upload_key":"k"}`, wantCode: "INVALID_PLACE_ID"},
			{name: "id_negative", target: "/places/-1/medias", body: `{"upload_key":"k"}`, wantCode: "INVALID_PLACE_ID"},
			{name: "body_not_json", target: "/places/3/medias", body: `upload_key=k`, wantCode: "INVALID_BODY"},
			{name: "body_null", target: "/places/3/medias", body: `null`, wantCode: "INVALID_BODY"},
			{name: "body_array", target: "/places/3/medias", body: `[]`, wantCode: "INVALID_BODY"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// No calls are set up: an invalid request must never reach the service.
				rec := serve(t, newRouter(newService(t)), jsonRequest(http.MethodPost, tc.target, tc.body))
				assert.Equal(t, wantError(http.StatusBadRequest, tc.wantCode), gotError(rec), "POST %s %s", tc.target, tc.body)
			})
		}
	})

	t.Run("service_errors", func(t *testing.T) {
		tests := []errorCase{
			{name: "unknown_place", err: domain.ErrPlaceNotFound, wantStatus: http.StatusNotFound, wantCode: "PLACE_NOT_FOUND"},
			{name: "invalid_key", err: storage.ErrInvalidUploadKey, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "not_uploaded", err: storage.ErrMediaNotUploaded, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "type_not_allowed", err: storage.ErrInvalidMediaType, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "image_too_large", err: storage.ErrImageTooLarge, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "video_too_large", err: storage.ErrVideoTooLarge, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "media_limit", err: domain.ErrMediaLimitReached, wantStatus: http.StatusConflict, wantCode: "MEDIA_LIMIT_REACHED"},
			{name: "storage_failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				svc := newService(t)
				svc.EXPECT().AddMedia(gomock.Any(), int64(3), "uploads/places/3/x.webp").Return(nil, tc.err)

				rec := serve(t, newRouter(svc), jsonRequest(http.MethodPost, "/places/3/medias", `{"upload_key":"uploads/places/3/x.webp"}`))
				assert.Equal(t, wantError(tc.wantStatus, tc.wantCode), gotError(rec), "POST /places/3/medias with error %v", tc.err)
			})
		}
	})

	t.Run("success", func(t *testing.T) {
		svc := newService(t)
		svc.EXPECT().AddMedia(gomock.Any(), int64(3), "uploads/places/3/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp").Return(&domain.Place{
			ID:            3,
			SubDistrictID: 1,
			Name:          "Kopi Senja",
			Attributes:    json.RawMessage(`{}`),
			Medias: []domain.Media{
				{ID: 9, Key: "places/3/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp", ContentType: "image/webp", Size: 2048, Position: 1},
			},
			Category: kopiCategory,
		}, nil)

		rec := serve(t, newRouter(svc), jsonRequest(http.MethodPost, "/places/3/medias",
			`{"upload_key":"uploads/places/3/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp"}`))
		want := `{
			"id": 3,
			"sub_district_id": 1,
			"name": "Kopi Senja",
			"address": null,
			"location": null,
			"operating_hours": null,
			"estimated_price_range": null,
			"short_description": null,
			"attributes": {},
			"rating_average": null,
			"rating_count": 0,
			"medias": [{
				"id": 9,
				"url": "https://img.test/places/3/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp",
				"content_type": "image/webp",
				"size": 2048,
				"position": 1
			}],
			"category": ` + kopiCategoryJSON + `
		}`
		assert.Equal(t, http.StatusOK, rec.Code, "POST /places/3/medias status; body: %s", rec.Body)
		assert.JSONEq(t, want, responseData(rec), "POST /places/3/medias")
	})
}

func TestHandler_DeleteMedia(t *testing.T) {
	t.Run("invalid_ids", func(t *testing.T) {
		tests := []struct {
			name     string
			target   string
			wantCode string
		}{
			{name: "place_not_number", target: "/places/abc/medias/5", wantCode: "INVALID_PLACE_ID"},
			{name: "place_zero", target: "/places/0/medias/5", wantCode: "INVALID_PLACE_ID"},
			{name: "media_not_number", target: "/places/3/medias/abc", wantCode: "INVALID_MEDIA_ID"},
			{name: "media_fractional", target: "/places/3/medias/1.5", wantCode: "INVALID_MEDIA_ID"},
			{name: "media_zero", target: "/places/3/medias/0", wantCode: "INVALID_MEDIA_ID"},
			{name: "media_negative", target: "/places/3/medias/-1", wantCode: "INVALID_MEDIA_ID"},
			{name: "media_overflow", target: "/places/3/medias/99999999999999999999", wantCode: "INVALID_MEDIA_ID"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// No calls are set up: an invalid id must never reach the service.
				rec := serve(t, newRouter(newService(t)), jsonRequest(http.MethodDelete, tc.target, ""))
				assert.Equal(t, wantError(http.StatusBadRequest, tc.wantCode), gotError(rec), "DELETE %s", tc.target)
			})
		}
	})

	t.Run("service_errors", func(t *testing.T) {
		tests := []errorCase{
			{name: "unknown_media", err: domain.ErrMediaNotFound, wantStatus: http.StatusNotFound, wantCode: "MEDIA_NOT_FOUND"},
			{name: "wrapped_unknown_media", err: fmt.Errorf("delete: %w", domain.ErrMediaNotFound), wantStatus: http.StatusNotFound, wantCode: "MEDIA_NOT_FOUND"},
			{name: "storage_failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				svc := newService(t)
				svc.EXPECT().DeleteMedia(gomock.Any(), int64(3), int64(5)).Return(tc.err)

				rec := serve(t, newRouter(svc), jsonRequest(http.MethodDelete, "/places/3/medias/5", ""))
				assert.Equal(t, wantError(tc.wantStatus, tc.wantCode), gotError(rec), "DELETE /places/3/medias/5 with error %v", tc.err)
			})
		}
	})

	t.Run("success", func(t *testing.T) {
		svc := newService(t)
		svc.EXPECT().DeleteMedia(gomock.Any(), int64(3), int64(5)).Return(nil)

		rec := serve(t, newRouter(svc), jsonRequest(http.MethodDelete, "/places/3/medias/5", ""))
		assert.Equal(t, http.StatusNoContent, rec.Code, "DELETE /places/3/medias/5 status; body: %s", rec.Body)
		assert.Zero(t, rec.Body.Len(), "DELETE /places/3/medias/5 body = %q, want empty", rec.Body)
	})
}

func TestHandler_InternalErrorsDoNotLeakDetails(t *testing.T) {
	dbErr := errors.New(`pq: relation "places" does not exist`)

	tests := []struct {
		name   string
		method string
		target string
		body   string
		fail   func(svc *mock.MockService)
	}{
		{
			name:   "list_places",
			method: http.MethodGet,
			target: "/places",
			body:   "",
			fail: func(s *mock.MockService) {
				s.EXPECT().List(gomock.Any(), places.Filter{}).Return(nil, 0, dbErr)
			},
		},
		{
			name:   "list_categories",
			method: http.MethodGet,
			target: "/categories",
			body:   "",
			fail: func(s *mock.MockService) {
				s.EXPECT().ListCategories(gomock.Any()).Return(nil, dbErr)
			},
		},
		{
			name:   "create_place",
			method: http.MethodPost,
			target: "/places",
			body:   `{"name":"Kopi","sub_district_id":1,"category_id":2}`,
			fail: func(s *mock.MockService) {
				s.EXPECT().Create(gomock.Any(), &domain.Place{Name: "Kopi", SubDistrictID: 1, CategoryID: 2}).Return(nil, dbErr)
			},
		},
		{
			name:   "show_place",
			method: http.MethodGet,
			target: "/places/1",
			body:   "",
			fail: func(s *mock.MockService) {
				s.EXPECT().Fetch(gomock.Any(), int64(1)).Return(nil, dbErr)
			},
		},
		{
			name:   "show_category",
			method: http.MethodGet,
			target: "/categories/1",
			body:   "",
			fail: func(s *mock.MockService) {
				s.EXPECT().FetchCategory(gomock.Any(), int64(1)).Return(nil, dbErr)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := newService(t)
			tc.fail(svc)

			rec := serve(t, newRouter(svc), jsonRequest(tc.method, tc.target, tc.body))
			assert.Equal(t, wantError(http.StatusInternalServerError, "INTERNAL_ERROR"), gotError(rec), "%s %s", tc.method, tc.target)
			for _, leak := range []string{"pq:", "relation"} {
				assert.NotContains(t, rec.Body.String(), leak, "%s %s 500 body must not show storage detail", tc.method, tc.target)
			}
		})
	}
}
