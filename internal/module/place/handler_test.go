package place_test

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

	"rantau-go/internal/module/place"
	"rantau-go/internal/module/place/mock"
	"rantau-go/internal/postgis"
	"rantau-go/internal/storage"
)

// newRouter serves the place handlers on the paths the API uses for them, so
// URL parameters resolve as they do in production.
func newRouter(svc place.Service) http.Handler {
	h := place.NewHandler(svc, testImageURL)
	r := chi.NewRouter()
	r.Get("/places", h.GetPlaces)
	r.Post("/places", h.CreatePlace)
	r.Get("/places/{id}", h.GetPlace)
	r.Put("/places/{id}", h.PutPlace)
	r.Delete("/places/{id}", h.DeletePlace)
	r.Post("/places/{id}/image/upload-url", h.CreateImageUpload)
	r.Put("/places/{id}/image", h.SetImage)
	r.Get("/categories", h.GetCategories)
	r.Get("/categories/{id}", h.GetCategory)
	return r
}

// testImageURL stands in for the storage's public URLs.
func testImageURL(key string) string {
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

// -----------------------------------------------------------------------------
// GET /places
// -----------------------------------------------------------------------------

func TestHandler_GetPlaces_RejectsMalformedFilters(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		wantCode string
	}{
		{name: "sub_district_id is not a number", target: "/places?sub_district_id=abc", wantCode: "INVALID_SUB_DISTRICT_ID"},
		{name: "sub_district_id is fractional", target: "/places?sub_district_id=1.5", wantCode: "INVALID_SUB_DISTRICT_ID"},
		{name: "sub_district_id is zero", target: "/places?sub_district_id=0", wantCode: "INVALID_SUB_DISTRICT_ID"},
		{name: "sub_district_id is negative", target: "/places?sub_district_id=-3", wantCode: "INVALID_SUB_DISTRICT_ID"},
		{name: "sub_district_id overflows 64 bits", target: "/places?sub_district_id=99999999999999999999", wantCode: "INVALID_SUB_DISTRICT_ID"},
		{name: "category_id has a non-number", target: "/places?category_id=1,x", wantCode: "INVALID_CATEGORY_ID"},
		{name: "category_id has an empty element", target: "/places?category_id=1,,2", wantCode: "INVALID_CATEGORY_ID"},
		{name: "category_id ends with a comma", target: "/places?category_id=1,", wantCode: "INVALID_CATEGORY_ID"},
		{name: "category_id is zero", target: "/places?category_id=0", wantCode: "INVALID_CATEGORY_ID"},
		{name: "category_id is negative", target: "/places?category_id=-1", wantCode: "INVALID_CATEGORY_ID"},
		{name: "category_id has a zero element", target: "/places?category_id=1,0", wantCode: "INVALID_CATEGORY_ID"},
		{name: "limit is not a number", target: "/places?limit=abc", wantCode: "INVALID_LIMIT"},
		{name: "limit is fractional", target: "/places?limit=2.5", wantCode: "INVALID_LIMIT"},
		{name: "limit is zero", target: "/places?limit=0", wantCode: "INVALID_LIMIT"},
		{name: "limit is negative", target: "/places?limit=-3", wantCode: "INVALID_LIMIT"},
		{name: "limit overflows 64 bits", target: "/places?limit=99999999999999999999", wantCode: "INVALID_LIMIT"},
		{name: "bbox is not numbers", target: "/places?bbox=abc", wantCode: "INVALID_BBOX"},
		{name: "bbox has three values", target: "/places?bbox=106.8,-6.3,106.9", wantCode: "INVALID_BBOX"},
		{name: "bbox has five values", target: "/places?bbox=106.8,-6.3,106.9,-6.1,1", wantCode: "INVALID_BBOX"},
		{name: "bbox has an empty element", target: "/places?bbox=106.8,,106.9,-6.1", wantCode: "INVALID_BBOX"},
		{name: "bbox has a non-number", target: "/places?bbox=106.8,x,106.9,-6.1", wantCode: "INVALID_BBOX"},
		{name: "bbox has NaN", target: "/places?bbox=NaN,-6.3,106.9,-6.1", wantCode: "INVALID_BBOX"},
		{name: "bbox has infinity", target: "/places?bbox=106.8,-6.3,Inf,-6.1", wantCode: "INVALID_BBOX"},
		{name: "bbox west is east of east", target: "/places?bbox=106.9,-6.3,106.8,-6.1", wantCode: "INVALID_BBOX"},
		{name: "bbox south is north of north", target: "/places?bbox=106.8,-6.1,106.9,-6.3", wantCode: "INVALID_BBOX"},
		{name: "bbox longitude out of range", target: "/places?bbox=-181,-6.3,106.9,-6.1", wantCode: "INVALID_BBOX"},
		{name: "bbox latitude out of range", target: "/places?bbox=106.8,-91,106.9,-6.1", wantCode: "INVALID_BBOX"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// No expectations: a malformed filter must never reach the service.
			svc := mock.NewMockService(gomock.NewController(t))
			rec := serve(t, newRouter(svc), jsonRequest(http.MethodGet, tc.target, ""))
			assertError(t, rec, http.StatusBadRequest, tc.wantCode)
		})
	}
}

func TestHandler_GetPlaces_PassesFilterToService(t *testing.T) {
	tests := []struct {
		name   string
		target string
		want   place.PlaceFilter
	}{
		{name: "no filters", target: "/places", want: place.PlaceFilter{}},
		{name: "query is trimmed", target: "/places?q=%20%20kopi%20%20", want: place.PlaceFilter{Query: "kopi"}},
		{name: "whitespace-only query means no search", target: "/places?q=%20%20%20", want: place.PlaceFilter{}},
		{name: "empty parameters are absent", target: "/places?sub_district_id=&category_id=", want: place.PlaceFilter{}},
		{name: "all filters together", target: "/places?sub_district_id=7&category_id=1,2&q=kopi",
			want: place.PlaceFilter{SubDistrictID: 7, CategoryIDs: []int64{1, 2}, Query: "kopi"}},
		{name: "limit is passed", target: "/places?limit=25", want: place.PlaceFilter{Limit: 25}},
		{name: "empty limit is absent", target: "/places?limit=", want: place.PlaceFilter{}},
		{name: "all filters together with limit", target: "/places?sub_district_id=7&category_id=1,2&q=kopi&limit=25",
			want: place.PlaceFilter{SubDistrictID: 7, CategoryIDs: []int64{1, 2}, Query: "kopi", Limit: 25}},
		{name: "bbox is passed as west, south, east, north", target: "/places?bbox=106.8,-6.3,106.9,-6.1",
			want: place.PlaceFilter{Bounds: &place.BoundingBox{MinLng: 106.8, MinLat: -6.3, MaxLng: 106.9, MaxLat: -6.1}}},
		{name: "empty bbox is absent", target: "/places?bbox=", want: place.PlaceFilter{}},
		{name: "bbox combines with limit", target: "/places?bbox=106.8,-6.3,106.9,-6.1&limit=25",
			want: place.PlaceFilter{Bounds: &place.BoundingBox{MinLng: 106.8, MinLat: -6.3, MaxLng: 106.9, MaxLat: -6.1}, Limit: 25}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().GetMany(gomock.Any(), tc.want).Return([]place.Place{}, 0, nil)

			rec := serve(t, newRouter(svc), jsonRequest(http.MethodGet, tc.target, ""))
			assertData(t, rec, http.StatusOK, `{"places": [], "total": 0}`)
		})
	}
}

func TestHandler_GetPlaces_MapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "unknown sub-district", err: place.ErrSubDistrictNotFound, wantStatus: http.StatusNotFound, wantCode: "SUB_DISTRICT_NOT_FOUND"},
		{name: "unknown category", err: place.ErrCategoryNotFound, wantStatus: http.StatusNotFound, wantCode: "CATEGORY_NOT_FOUND"},
		{name: "wrapped unknown category", err: fmt.Errorf("filter: %w", place.ErrCategoryNotFound), wantStatus: http.StatusNotFound, wantCode: "CATEGORY_NOT_FOUND"},
		{name: "storage failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().GetMany(gomock.Any(), gomock.Any()).Return(nil, 0, tc.err)

			rec := serve(t, newRouter(svc), jsonRequest(http.MethodGet, "/places?sub_district_id=7", ""))
			assertError(t, rec, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestHandler_GetPlaces_ResponseShape(t *testing.T) {
	svc := mock.NewMockService(gomock.NewController(t))
	svc.EXPECT().GetMany(gomock.Any(), gomock.Any()).Return([]place.Place{{
		ID:            3,
		SubDistrictID: 1,
		Name:          "Kopi Tubruk",
		Attributes:    json.RawMessage(`{}`),
		Category:      place.Category{ID: 2, Name: "Kopi", Slug: "kopi", Code: "K", Label: "Kopi", Color: "plum"},
	}}, 12, nil)

	// total counts every match, not just the places in this response.
	rec := serve(t, newRouter(svc), jsonRequest(http.MethodGet, "/places", ""))
	assertData(t, rec, http.StatusOK, `{"total": 12, "places": [{
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
		"image_url": null,
		"category": {"id": 2, "name": "Kopi", "slug": "kopi", "code": "K", "label": "Kopi", "color": "plum"}
	}]}`)
}

// -----------------------------------------------------------------------------
// GET /categories
// -----------------------------------------------------------------------------

func TestHandler_GetCategories_Success(t *testing.T) {
	tests := []struct {
		name       string
		categories []place.Category
		want       string
	}{
		{name: "no categories", categories: []place.Category{}, want: `[]`},
		{
			name:       "a category without fields lists an empty fields array",
			categories: []place.Category{{ID: 1, Name: "Warung", Slug: "warung", Code: "W", Label: "Warung", Color: "chili"}},
			want:       `[{"id":1,"name":"Warung","slug":"warung","code":"W","label":"Warung","color":"chili","fields":[]}]`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().GetCategories(gomock.Any()).Return(tc.categories, nil)

			rec := serve(t, newRouter(svc), jsonRequest(http.MethodGet, "/categories", ""))
			assertData(t, rec, http.StatusOK, tc.want)
		})
	}
}

// -----------------------------------------------------------------------------
// POST /places
// -----------------------------------------------------------------------------

func TestHandler_CreatePlace_RejectsMalformedBodies(t *testing.T) {
	const prefix = `{"name":"Kopi","sub_district_id":1,"category_id":2,"location":`

	tests := []struct {
		name string
		body string
	}{
		{name: "empty body", body: ``},
		{name: "body is null", body: `null`},
		{name: "truncated JSON", body: `{"name":"Kopi"`},
		{name: "sub_district_id is a string", body: `{"name":"Kopi","sub_district_id":"one","category_id":2}`},
		{name: "location is a line", body: prefix + `{"type":"LineString","coordinates":[[106.8,-6.2],[106.9,-6.3]]}}`},
		{name: "coordinates are swapped", body: prefix + `{"type":"Point","coordinates":[-6.2,106.8]}}`},
		{name: "latitude above 90", body: prefix + `{"type":"Point","coordinates":[106.8,91]}}`},
		{name: "longitude above 180", body: prefix + `{"type":"Point","coordinates":[181,-6.2]}}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			rec := serve(t, newRouter(svc), jsonRequest(http.MethodPost, "/places", tc.body))
			assertError(t, rec, http.StatusBadRequest, "INVALID_BODY")
		})
	}
}

func TestHandler_CreatePlace_MapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "blank name", err: place.ErrInvalidName, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "name too long", err: place.ErrNameTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "address too long", err: place.ErrAddressTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "operating hours too long", err: place.ErrOperatingHoursTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "estimated price range too long", err: place.ErrEstimatedPriceRangeTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "unknown sub-district", err: place.ErrSubDistrictNotFound, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "unknown category", err: place.ErrCategoryNotFound, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "invalid attributes", err: place.ErrInvalidAttributes, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "wrapped invalid attributes", err: fmt.Errorf("%w: %q is required", place.ErrInvalidAttributes, "contact"), wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "storage failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, tc.err)

			rec := serve(t, newRouter(svc), jsonRequest(http.MethodPost, "/places",
				`{"name":"Kopi","sub_district_id":1,"category_id":2}`))
			assertError(t, rec, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestHandler_CreatePlace_Success(t *testing.T) {
	svc := mock.NewMockService(gomock.NewController(t))

	var got *place.Place
	svc.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, p *place.Place) (*place.Place, error) {
			got = p
			return &place.Place{
				ID:            42,
				SubDistrictID: p.SubDistrictID,
				Name:          p.Name,
				Location:      p.Location,
				Attributes:    json.RawMessage(`{}`),
				Category:      place.Category{ID: 2, Name: "Kopi", Slug: "kopi", Code: "K", Label: "Kopi", Color: "plum"},
			}, nil
		})

	body := `{"name":"Kopi Senja","sub_district_id":1,"category_id":2,` +
		`"location":{"type":"Point","coordinates":[106.8,-6.2]},"id":77,"created_at":"yesterday"}`
	rec := serve(t, newRouter(svc), jsonRequest(http.MethodPost, "/places", body))

	require.NotNil(t, got, "service was not called")
	assert.Zero(t, got.ID, "client-sent id reached the service")
	assert.Nil(t, got.Address, "absent optional field reached the service")
	assert.Nil(t, got.Attributes, "absent optional field reached the service")
	require.NotNil(t, got.Location, "location did not reach the service")
	assert.Equal(t, 106.8, got.Location.X())
	assert.Equal(t, -6.2, got.Location.Y())

	assertData(t, rec, http.StatusCreated, `{
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
		"image_url": null,
		"category": {"id": 2, "name": "Kopi", "slug": "kopi", "code": "K", "label": "Kopi", "color": "plum"}
	}`)
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
				func(got context.Context, _ *place.Place) (*place.Place, error) {
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
			assert.NotEqual(t, http.StatusCreated, rec.Code, "status 201 for a request whose context had ended")
		})
	}
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
			name:   "list places",
			method: http.MethodGet,
			target: "/places",
			body:   "",
			fail: func(s *mock.MockService) {
				s.EXPECT().GetMany(gomock.Any(), gomock.Any()).Return(nil, 0, dbErr)
			},
		},
		{
			name:   "list categories",
			method: http.MethodGet,
			target: "/categories",
			body:   "",
			fail: func(s *mock.MockService) {
				s.EXPECT().GetCategories(gomock.Any()).Return(nil, dbErr)
			},
		},
		{
			name:   "create place",
			method: http.MethodPost,
			target: "/places",
			body:   `{"name":"Kopi","sub_district_id":1,"category_id":2}`,
			fail: func(s *mock.MockService) {
				s.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, dbErr)
			},
		},
		{
			name:   "get place",
			method: http.MethodGet,
			target: "/places/1",
			body:   "",
			fail: func(s *mock.MockService) {
				s.EXPECT().GetPlace(gomock.Any(), int64(1)).Return(nil, dbErr)
			},
		},
		{
			name:   "get category",
			method: http.MethodGet,
			target: "/categories/1",
			body:   "",
			fail: func(s *mock.MockService) {
				s.EXPECT().GetCategory(gomock.Any(), int64(1)).Return(nil, dbErr)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			tc.fail(svc)

			rec := serve(t, newRouter(svc), jsonRequest(tc.method, tc.target, tc.body))
			assertError(t, rec, http.StatusInternalServerError, "INTERNAL_ERROR")
			assert.NotContains(t, rec.Body.String(), "pq:", "500 body exposes storage detail")
			assert.NotContains(t, rec.Body.String(), "relation", "500 body exposes storage detail")
		})
	}
}

// -----------------------------------------------------------------------------
// PUT /places/{id}
// -----------------------------------------------------------------------------

func TestHandler_PutPlace_RejectsInvalidRequests(t *testing.T) {
	const valid = `{"name":"Kopi","sub_district_id":1,"category_id":2}`

	tests := []struct {
		name     string
		target   string
		body     string
		wantCode string
	}{
		{name: "place id is not a number", target: "/places/abc", body: valid, wantCode: "INVALID_PLACE_ID"},
		{name: "place id is fractional", target: "/places/1.5", body: valid, wantCode: "INVALID_PLACE_ID"},
		{name: "place id is zero", target: "/places/0", body: valid, wantCode: "INVALID_PLACE_ID"},
		{name: "place id is negative", target: "/places/-1", body: valid, wantCode: "INVALID_PLACE_ID"},
		{name: "empty body", target: "/places/1", body: ``, wantCode: "INVALID_BODY"},
		{name: "truncated JSON", target: "/places/1", body: `{"name":"Kopi"`, wantCode: "INVALID_BODY"},
		{name: "body is an array", target: "/places/1", body: `[]`, wantCode: "INVALID_BODY"},
		{name: "body is null", target: "/places/1", body: `null`, wantCode: "INVALID_BODY"},
		{name: "name is a number", target: "/places/1", body: `{"name":5,"sub_district_id":1,"category_id":2}`, wantCode: "INVALID_BODY"},
		{name: "sub_district_id is a string", target: "/places/1", body: `{"name":"Kopi","sub_district_id":"one","category_id":2}`, wantCode: "INVALID_BODY"},
		{name: "location is a line", target: "/places/1",
			body: `{"name":"Kopi","sub_district_id":1,"category_id":2,"location":{"type":"LineString","coordinates":[[106.8,-6.2],[106.9,-6.3]]}}`, wantCode: "INVALID_BODY"},
		{name: "location coordinates are swapped", target: "/places/1",
			body: `{"name":"Kopi","sub_district_id":1,"category_id":2,"location":{"type":"Point","coordinates":[-6.2,106.8]}}`, wantCode: "INVALID_BODY"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// No expectations: an invalid request must never reach the service.
			svc := mock.NewMockService(gomock.NewController(t))
			rec := serve(t, newRouter(svc), jsonRequest(http.MethodPut, tc.target, tc.body))
			assertError(t, rec, http.StatusBadRequest, tc.wantCode)
		})
	}
}

func TestHandler_PutPlace_PassesPlaceToService(t *testing.T) {
	tests := []struct {
		name string
		body string
		want *place.Place
	}{
		{name: "every field is passed as sent",
			body: `{"sub_district_id":2,"category_id":3,"name":"  Kopi Senja  ","address":"Jl. Sudirman 9",
				"location":{"type":"Point","coordinates":[106.8,-6.2]},"operating_hours":"08:00-17:00",
				"estimated_price_range":"Rp20.000-50.000","short_description":"Kopi tubruk",
				"attributes":{"contact":"0812"}}`,
			want: &place.Place{
				SubDistrictID:       2,
				CategoryID:          3,
				Name:                "  Kopi Senja  ",
				Address:             ptr("Jl. Sudirman 9"),
				Location:            postgis.MustPoint(106.8, -6.2),
				OperatingHours:      ptr("08:00-17:00"),
				EstimatedPriceRange: ptr("Rp20.000-50.000"),
				ShortDescription:    ptr("Kopi tubruk"),
				Attributes:          json.RawMessage(`{"contact":"0812"}`),
			}},
		{name: "left-out keys are passed as absent",
			body: `{"sub_district_id":2,"category_id":3,"name":"Kopi"}`,
			want: &place.Place{SubDistrictID: 2, CategoryID: 3, Name: "Kopi"}},
		{name: "client id and unknown keys are ignored",
			body: `{"id":99,"created_at":"yesterday","sub_district_id":2,"category_id":3,"name":"Kopi"}`,
			want: &place.Place{SubDistrictID: 2, CategoryID: 3, Name: "Kopi"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().Update(gomock.Any(), int64(9), tc.want).Return(&place.Place{ID: 9}, nil)

			rec := serve(t, newRouter(svc), jsonRequest(http.MethodPut, "/places/9", tc.body))
			require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body)
		})
	}
}

func TestHandler_PutPlace_MapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "unknown place", err: place.ErrPlaceNotFound, wantStatus: http.StatusNotFound, wantCode: "PLACE_NOT_FOUND"},
		{name: "wrapped unknown place", err: fmt.Errorf("update: %w", place.ErrPlaceNotFound), wantStatus: http.StatusNotFound, wantCode: "PLACE_NOT_FOUND"},
		{name: "blank name", err: place.ErrInvalidName, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "name too long", err: place.ErrNameTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "address too long", err: place.ErrAddressTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "operating hours too long", err: place.ErrOperatingHoursTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "estimated price range too long", err: place.ErrEstimatedPriceRangeTooLong, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "unknown sub-district", err: place.ErrSubDistrictNotFound, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "unknown category", err: place.ErrCategoryNotFound, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "invalid attributes", err: place.ErrInvalidAttributes, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "wrapped invalid attributes", err: fmt.Errorf("%w: %q is required", place.ErrInvalidAttributes, "contact"), wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "location outside Indonesia", err: place.ErrInvalidLocation, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "storage failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().Update(gomock.Any(), int64(1), gomock.Any()).Return(nil, tc.err)

			rec := serve(t, newRouter(svc), jsonRequest(http.MethodPut, "/places/1", `{"name":"Kopi","sub_district_id":1,"category_id":2}`))
			assertError(t, rec, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestHandler_PutPlace_Success(t *testing.T) {
	svc := mock.NewMockService(gomock.NewController(t))
	svc.EXPECT().Update(gomock.Any(), int64(3), &place.Place{SubDistrictID: 1, CategoryID: 2, Name: "Kopi Senja"}).Return(&place.Place{
		ID:            3,
		SubDistrictID: 1,
		Name:          "Kopi Senja",
		Attributes:    json.RawMessage(`{}`),
		Category:      place.Category{ID: 2, Name: "Kopi", Slug: "kopi", Code: "K", Label: "Kopi", Color: "plum"},
	}, nil)

	rec := serve(t, newRouter(svc), jsonRequest(http.MethodPut, "/places/3", `{"name":"Kopi Senja","sub_district_id":1,"category_id":2}`))
	assertData(t, rec, http.StatusOK, `{
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
		"image_url": null,
		"category": {"id": 2, "name": "Kopi", "slug": "kopi", "code": "K", "label": "Kopi", "color": "plum"}
	}`)
}

// -----------------------------------------------------------------------------
// DELETE /places/{id}
// -----------------------------------------------------------------------------

func TestHandler_DeletePlace_RejectsInvalidID(t *testing.T) {
	tests := []struct {
		name   string
		target string
	}{
		{name: "place id is not a number", target: "/places/abc"},
		{name: "place id is fractional", target: "/places/1.5"},
		{name: "place id is zero", target: "/places/0"},
		{name: "place id is negative", target: "/places/-1"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// No expectations: an invalid id must never reach the service.
			svc := mock.NewMockService(gomock.NewController(t))
			rec := serve(t, newRouter(svc), jsonRequest(http.MethodDelete, tc.target, ""))
			assertError(t, rec, http.StatusBadRequest, "INVALID_PLACE_ID")
		})
	}
}

func TestHandler_DeletePlace_MapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "unknown place", err: place.ErrPlaceNotFound, wantStatus: http.StatusNotFound, wantCode: "PLACE_NOT_FOUND"},
		{name: "wrapped unknown place", err: fmt.Errorf("delete: %w", place.ErrPlaceNotFound), wantStatus: http.StatusNotFound, wantCode: "PLACE_NOT_FOUND"},
		{name: "storage failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().Delete(gomock.Any(), int64(4)).Return(tc.err)

			rec := serve(t, newRouter(svc), jsonRequest(http.MethodDelete, "/places/4", ""))
			assertError(t, rec, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestHandler_DeletePlace_Success(t *testing.T) {
	svc := mock.NewMockService(gomock.NewController(t))
	svc.EXPECT().Delete(gomock.Any(), int64(4)).Return(nil)

	rec := serve(t, newRouter(svc), jsonRequest(http.MethodDelete, "/places/4", ""))
	require.Equal(t, http.StatusNoContent, rec.Code, "body: %s", rec.Body)
	assert.Empty(t, rec.Body.String(), "body should be empty")
}

// -----------------------------------------------------------------------------
// GET /places/{id}
// -----------------------------------------------------------------------------

func TestHandler_GetPlace_RejectsInvalidID(t *testing.T) {
	tests := []struct {
		name   string
		target string
	}{
		{name: "place id is not a number", target: "/places/abc"},
		{name: "place id is fractional", target: "/places/1.5"},
		{name: "place id is zero", target: "/places/0"},
		{name: "place id is negative", target: "/places/-1"},
		{name: "place id overflows 64 bits", target: "/places/99999999999999999999"},
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

func TestHandler_GetPlace_MapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "unknown place", err: place.ErrPlaceNotFound, wantStatus: http.StatusNotFound, wantCode: "PLACE_NOT_FOUND"},
		{name: "wrapped unknown place", err: fmt.Errorf("get: %w", place.ErrPlaceNotFound), wantStatus: http.StatusNotFound, wantCode: "PLACE_NOT_FOUND"},
		{name: "storage failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().GetPlace(gomock.Any(), int64(3)).Return(nil, tc.err)

			rec := serve(t, newRouter(svc), jsonRequest(http.MethodGet, "/places/3", ""))
			assertError(t, rec, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestHandler_GetPlace_Success(t *testing.T) {
	svc := mock.NewMockService(gomock.NewController(t))
	svc.EXPECT().GetPlace(gomock.Any(), int64(3)).Return(&place.Place{
		ID:            3,
		SubDistrictID: 1,
		CategoryID:    2,
		Name:          "Kopi Senja",
		Address:       ptr("Jl. Sudirman 9"),
		Location:      postgis.MustPoint(106.8, -6.2),
		Attributes:    json.RawMessage(`{"halal":true}`),
		Category:      place.Category{ID: 2, Name: "Kopi", Slug: "kopi", Code: "K", Label: "Kopi", Color: "plum"},
	}, nil)

	rec := serve(t, newRouter(svc), jsonRequest(http.MethodGet, "/places/3", ""))
	assertData(t, rec, http.StatusOK, `{
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
		"image_url": null,
		"category": {"id": 2, "name": "Kopi", "slug": "kopi", "code": "K", "label": "Kopi", "color": "plum"}
	}`)
}

func TestHandler_GetPlace_Rating(t *testing.T) {
	tests := []struct {
		name        string
		count       int
		sum         int
		wantAverage string // as JSON
	}{
		{name: "not rated yet has no average", count: 0, sum: 0, wantAverage: `null`},
		{name: "one rating is the average", count: 1, sum: 4, wantAverage: `4`},
		{name: "average of several ratings", count: 4, sum: 17, wantAverage: `4.25`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().GetPlace(gomock.Any(), int64(3)).Return(&place.Place{
				ID:          3,
				Name:        "Kopi Senja",
				Attributes:  json.RawMessage(`{}`),
				RatingCount: tc.count,
				RatingSum:   tc.sum,
			}, nil)

			rec := serve(t, newRouter(svc), jsonRequest(http.MethodGet, "/places/3", ""))
			require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body)

			var got struct {
				RatingAverage json.RawMessage `json:"rating_average"`
				RatingCount   int             `json:"rating_count"`
			}
			require.NoError(t, json.Unmarshal(responseData(t, rec), &got), "decode")
			assert.Equal(t, tc.wantAverage, string(got.RatingAverage))
			assert.Equal(t, tc.count, got.RatingCount)
		})
	}
}

// -----------------------------------------------------------------------------
// GET /categories/{id}
// -----------------------------------------------------------------------------

func TestHandler_GetCategory_RejectsInvalidID(t *testing.T) {
	tests := []struct {
		name   string
		target string
	}{
		{name: "category id is not a number", target: "/categories/abc"},
		{name: "category id is fractional", target: "/categories/1.5"},
		{name: "category id is zero", target: "/categories/0"},
		{name: "category id is negative", target: "/categories/-1"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			rec := serve(t, newRouter(svc), jsonRequest(http.MethodGet, tc.target, ""))
			assertError(t, rec, http.StatusBadRequest, "INVALID_CATEGORY_ID")
		})
	}
}

func TestHandler_GetCategory_MapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "unknown category", err: place.ErrCategoryNotFound, wantStatus: http.StatusNotFound, wantCode: "CATEGORY_NOT_FOUND"},
		{name: "wrapped unknown category", err: fmt.Errorf("get: %w", place.ErrCategoryNotFound), wantStatus: http.StatusNotFound, wantCode: "CATEGORY_NOT_FOUND"},
		{name: "storage failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().GetCategory(gomock.Any(), int64(2)).Return(nil, tc.err)

			rec := serve(t, newRouter(svc), jsonRequest(http.MethodGet, "/categories/2", ""))
			assertError(t, rec, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestHandler_GetCategory_Success(t *testing.T) {
	tests := []struct {
		name     string
		category place.Category
		want     string
	}{
		{
			name:     "a category without fields has an empty fields array",
			category: place.Category{ID: 2, Name: "Kopi", Slug: "kopi", Code: "K", Label: "Kopi", Color: "plum"},
			want:     `{"id":2,"name":"Kopi","slug":"kopi","code":"K","label":"Kopi","color":"plum","fields":[]}`,
		},
		{
			name: "a category lists its fields",
			category: place.Category{ID: 2, Name: "Kopi", Slug: "kopi", Code: "K", Label: "Kopi", Color: "plum", Fields: []place.CategoryField{
				{ID: 9, CategoryID: 2, Key: "halal", Label: "Halal", DataType: place.FieldTypeBoolean, Required: true, SortOrder: 1},
			}},
			want: `{"id":2,"name":"Kopi","slug":"kopi","code":"K","label":"Kopi","color":"plum","fields":[
				{"id":9,"category_id":2,"key":"halal","label":"Halal","data_type":"boolean","required":true,"options":null,"sort_order":1}
			]}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().GetCategory(gomock.Any(), int64(2)).Return(&tc.category, nil)

			rec := serve(t, newRouter(svc), jsonRequest(http.MethodGet, "/categories/2", ""))
			assertData(t, rec, http.StatusOK, tc.want)
		})
	}
}

func TestHandler_GetPlace_ImageURL(t *testing.T) {
	svc := mock.NewMockService(gomock.NewController(t))
	svc.EXPECT().GetPlace(gomock.Any(), int64(3)).Return(&place.Place{
		ID:         3,
		Name:       "Kopi Senja",
		Attributes: json.RawMessage(`{}`),
		ImageKey:   ptr("places/3/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp"),
	}, nil)

	rec := serve(t, newRouter(svc), jsonRequest(http.MethodGet, "/places/3", ""))
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body)

	var got struct {
		ImageURL *string `json:"image_url"`
	}
	require.NoError(t, json.Unmarshal(responseData(t, rec), &got), "decode")
	require.NotNil(t, got.ImageURL, "image_url is missing")
	assert.Equal(t, "https://img.test/places/3/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp", *got.ImageURL)
}

// -----------------------------------------------------------------------------
// POST /places/{id}/image/upload-url
// -----------------------------------------------------------------------------

func TestHandler_CreateImageUpload_RejectsInvalidRequests(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		body     string
		wantCode string
	}{
		{name: "place id is not a number", target: "/places/abc/image/upload-url", body: `{"content_type":"image/png"}`, wantCode: "INVALID_PLACE_ID"},
		{name: "place id is zero", target: "/places/0/image/upload-url", body: `{"content_type":"image/png"}`, wantCode: "INVALID_PLACE_ID"},
		{name: "body is not JSON", target: "/places/3/image/upload-url", body: `content_type=image/png`, wantCode: "INVALID_BODY"},
		{name: "body is null", target: "/places/3/image/upload-url", body: `null`, wantCode: "INVALID_BODY"},
		{name: "body is an array", target: "/places/3/image/upload-url", body: `[]`, wantCode: "INVALID_BODY"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// No expectations: an invalid request must never reach the service.
			svc := mock.NewMockService(gomock.NewController(t))
			rec := serve(t, newRouter(svc), jsonRequest(http.MethodPost, tc.target, tc.body))
			assertError(t, rec, http.StatusBadRequest, tc.wantCode)
		})
	}
}

func TestHandler_CreateImageUpload_MapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "unknown place", err: place.ErrPlaceNotFound, wantStatus: http.StatusNotFound, wantCode: "PLACE_NOT_FOUND"},
		{name: "image type not allowed", err: storage.ErrInvalidImageType, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "storage failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().ImageUploadURL(gomock.Any(), int64(3), "image/gif").Return(storage.Upload{}, tc.err)

			rec := serve(t, newRouter(svc), jsonRequest(http.MethodPost, "/places/3/image/upload-url", `{"content_type":"image/gif"}`))
			assertError(t, rec, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestHandler_CreateImageUpload_Success(t *testing.T) {
	svc := mock.NewMockService(gomock.NewController(t))
	svc.EXPECT().ImageUploadURL(gomock.Any(), int64(3), "image/webp").Return(storage.Upload{
		Key: "uploads/places/3/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp",
		PresignedUpload: storage.PresignedUpload{
			URL:       "http://localhost:3900/rantau-images/uploads/places/3/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp?X-Amz-Signature=abc",
			Method:    "PUT",
			Headers:   map[string]string{"Content-Type": "image/webp"},
			ExpiresAt: time.Date(2026, 10, 31, 12, 5, 0, 0, time.UTC),
		},
	}, nil)

	rec := serve(t, newRouter(svc), jsonRequest(http.MethodPost, "/places/3/image/upload-url", `{"content_type":"image/webp"}`))
	assertData(t, rec, http.StatusOK, `{
		"upload_key": "uploads/places/3/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp",
		"upload_url": "http://localhost:3900/rantau-images/uploads/places/3/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp?X-Amz-Signature=abc",
		"method": "PUT",
		"headers": {"Content-Type": "image/webp"},
		"expires_at": "2026-10-31T12:05:00Z"
	}`)
}

// -----------------------------------------------------------------------------
// PUT /places/{id}/image
// -----------------------------------------------------------------------------

func TestHandler_SetImage_RejectsInvalidRequests(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		body     string
		wantCode string
	}{
		{name: "place id is not a number", target: "/places/abc/image", body: `{"upload_key":"k"}`, wantCode: "INVALID_PLACE_ID"},
		{name: "place id is negative", target: "/places/-1/image", body: `{"upload_key":"k"}`, wantCode: "INVALID_PLACE_ID"},
		{name: "body is not JSON", target: "/places/3/image", body: `upload_key=k`, wantCode: "INVALID_BODY"},
		{name: "body is null", target: "/places/3/image", body: `null`, wantCode: "INVALID_BODY"},
		{name: "body is an array", target: "/places/3/image", body: `[]`, wantCode: "INVALID_BODY"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// No expectations: an invalid request must never reach the service.
			svc := mock.NewMockService(gomock.NewController(t))
			rec := serve(t, newRouter(svc), jsonRequest(http.MethodPut, tc.target, tc.body))
			assertError(t, rec, http.StatusBadRequest, tc.wantCode)
		})
	}
}

func TestHandler_SetImage_MapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "unknown place", err: place.ErrPlaceNotFound, wantStatus: http.StatusNotFound, wantCode: "PLACE_NOT_FOUND"},
		{name: "invalid upload key", err: storage.ErrInvalidImageKey, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "image not uploaded", err: storage.ErrImageNotUploaded, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "image type not allowed", err: storage.ErrInvalidImageType, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "image too large", err: storage.ErrImageTooLarge, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "storage failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().SetImage(gomock.Any(), int64(3), "uploads/places/3/x.webp").Return(nil, tc.err)

			rec := serve(t, newRouter(svc), jsonRequest(http.MethodPut, "/places/3/image", `{"upload_key":"uploads/places/3/x.webp"}`))
			assertError(t, rec, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestHandler_SetImage_Success(t *testing.T) {
	svc := mock.NewMockService(gomock.NewController(t))
	svc.EXPECT().SetImage(gomock.Any(), int64(3), "uploads/places/3/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp").Return(&place.Place{
		ID:            3,
		SubDistrictID: 1,
		Name:          "Kopi Senja",
		Attributes:    json.RawMessage(`{}`),
		ImageKey:      ptr("places/3/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp"),
		Category:      place.Category{ID: 2, Name: "Kopi", Slug: "kopi", Code: "K", Label: "Kopi", Color: "plum"},
	}, nil)

	rec := serve(t, newRouter(svc), jsonRequest(http.MethodPut, "/places/3/image",
		`{"upload_key":"uploads/places/3/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp"}`))
	assertData(t, rec, http.StatusOK, `{
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
		"image_url": "https://img.test/places/3/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp",
		"category": {"id": 2, "name": "Kopi", "slug": "kopi", "code": "K", "label": "Kopi", "color": "plum"}
	}`)
}
