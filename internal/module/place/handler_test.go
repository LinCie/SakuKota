package place_test

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

	"rantau-go/internal/module/place"
	"rantau-go/internal/module/place/mock"
	"rantau-go/internal/postgis"
)

// newRouter serves the place handlers on the paths the API uses for them, so
// URL parameters resolve as they do in production.
func newRouter(svc place.Service) http.Handler {
	h := place.NewHandler(svc)
	r := chi.NewRouter()
	r.Get("/places", h.GetPlaces)
	r.Post("/places", h.CreatePlace)
	r.Patch("/places/{id}", h.PatchPlace)
	r.Delete("/places/{id}", h.DeletePlace)
	r.Get("/categories", h.GetCategories)
	r.Post("/categories", h.CreateCategory)
	r.Post("/categories/{id}/fields", h.CreateCategoryField)
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

// -----------------------------------------------------------------------------
// GET /places
// -----------------------------------------------------------------------------

func TestHandler_GetPlaces_RejectsMalformedFilters(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		wantCode string
	}{
		{name: "sub_district_id is not a number", target: "/places?sub_district_id=abc", wantCode: "invalid_sub_district_id"},
		{name: "sub_district_id is fractional", target: "/places?sub_district_id=1.5", wantCode: "invalid_sub_district_id"},
		{name: "sub_district_id is zero", target: "/places?sub_district_id=0", wantCode: "invalid_sub_district_id"},
		{name: "sub_district_id is negative", target: "/places?sub_district_id=-3", wantCode: "invalid_sub_district_id"},
		{name: "sub_district_id overflows 64 bits", target: "/places?sub_district_id=99999999999999999999", wantCode: "invalid_sub_district_id"},
		{name: "category_id has a non-number", target: "/places?category_id=1,x", wantCode: "invalid_category_id"},
		{name: "category_id has an empty element", target: "/places?category_id=1,,2", wantCode: "invalid_category_id"},
		{name: "category_id ends with a comma", target: "/places?category_id=1,", wantCode: "invalid_category_id"},
		{name: "category_id is zero", target: "/places?category_id=0", wantCode: "invalid_category_id"},
		{name: "category_id is negative", target: "/places?category_id=-1", wantCode: "invalid_category_id"},
		{name: "category_id has a zero element", target: "/places?category_id=1,0", wantCode: "invalid_category_id"},
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
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().GetMany(gomock.Any(), tc.want).Return([]place.Place{}, nil)

			rec := serve(t, newRouter(svc), jsonRequest(http.MethodGet, tc.target, ""))
			assertData(t, rec, http.StatusOK, `[]`)
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
		{name: "unknown sub-district", err: place.ErrSubDistrictNotFound, wantStatus: http.StatusNotFound, wantCode: "sub_district_not_found"},
		{name: "unknown category", err: place.ErrCategoryNotFound, wantStatus: http.StatusNotFound, wantCode: "category_not_found"},
		{name: "wrapped unknown category", err: fmt.Errorf("filter: %w", place.ErrCategoryNotFound), wantStatus: http.StatusNotFound, wantCode: "category_not_found"},
		{name: "storage failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().GetMany(gomock.Any(), gomock.Any()).Return(nil, tc.err)

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
		Category:      place.Category{ID: 2, Name: "Kopi", Slug: "kopi"},
	}}, nil)

	rec := serve(t, newRouter(svc), jsonRequest(http.MethodGet, "/places", ""))
	assertData(t, rec, http.StatusOK, `[{
		"id": 3,
		"sub_district_id": 1,
		"name": "Kopi Tubruk",
		"address": null,
		"location": null,
		"operating_hours": null,
		"estimated_price_range": null,
		"short_description": null,
		"attributes": {},
		"category": {"id": 2, "name": "Kopi", "slug": "kopi"}
	}]`)
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
			categories: []place.Category{{ID: 1, Name: "Warung", Slug: "warung"}},
			want:       `[{"id":1,"name":"Warung","slug":"warung","fields":[]}]`,
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
// POST /categories
// -----------------------------------------------------------------------------

func TestHandler_CreateCategory_RejectsMalformedBodies(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "empty body", body: ``},
		{name: "truncated JSON", body: `{"name":"Kopi","slug":"kopi"`},
		{name: "name is a number", body: `{"name":5,"slug":"kopi"}`},
		{name: "body is an array", body: `[]`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			rec := serve(t, newRouter(svc), jsonRequest(http.MethodPost, "/categories", tc.body))
			assertError(t, rec, http.StatusBadRequest, "invalid_body")
		})
	}
}

func TestHandler_CreateCategory_MapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "invalid name", err: place.ErrInvalidCategoryName, wantStatus: http.StatusBadRequest, wantCode: "validation_error"},
		{name: "invalid slug", err: place.ErrInvalidSlug, wantStatus: http.StatusBadRequest, wantCode: "validation_error"},
		{name: "duplicate name or slug", err: place.ErrCategoryExists, wantStatus: http.StatusConflict, wantCode: "category_exists"},
		{name: "wrapped duplicate", err: fmt.Errorf("create: %w", place.ErrCategoryExists), wantStatus: http.StatusConflict, wantCode: "category_exists"},
		{name: "storage failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().CreateCategory(gomock.Any(), gomock.Any()).Return(nil, tc.err)

			rec := serve(t, newRouter(svc), jsonRequest(http.MethodPost, "/categories", `{"name":"Kopi","slug":"kopi"}`))
			assertError(t, rec, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestHandler_CreateCategory_IgnoresUnknownKeys(t *testing.T) {
	svc := mock.NewMockService(gomock.NewController(t))

	var got *place.Category
	svc.EXPECT().CreateCategory(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, c *place.Category) (*place.Category, error) {
			got = c
			return &place.Category{ID: 5, Name: c.Name, Slug: c.Slug}, nil
		})

	body := `{"name":"Kopi","slug":"kopi","id":99,"fields":[{"key":"halal"}],"unexpected":true}`
	rec := serve(t, newRouter(svc), jsonRequest(http.MethodPost, "/categories", body))

	if got == nil {
		t.Fatal("service was not called")
	}
	if got.Name != "Kopi" || got.Slug != "kopi" {
		t.Errorf("service got name %q slug %q, want Kopi and kopi", got.Name, got.Slug)
	}
	if got.ID != 0 || len(got.Fields) != 0 {
		t.Errorf("client-sent id or fields reached the service: id %d, %d fields", got.ID, len(got.Fields))
	}
	assertData(t, rec, http.StatusCreated, `{"id":5,"name":"Kopi","slug":"kopi","fields":[]}`)
}

// -----------------------------------------------------------------------------
// POST /categories/{id}/fields
// -----------------------------------------------------------------------------

func TestHandler_CreateCategoryField_RejectsMalformedRequests(t *testing.T) {
	validBody := `{"key":"halal","label":"Halal","data_type":"boolean"}`

	tests := []struct {
		name     string
		target   string
		body     string
		wantCode string
	}{
		{name: "category id is not a number", target: "/categories/abc/fields", body: validBody, wantCode: "invalid_category_id"},
		{name: "truncated JSON", target: "/categories/1/fields", body: `{"key":"halal"`, wantCode: "invalid_body"},
		{name: "options is not a list", target: "/categories/1/fields", body: `{"key":"halal","label":"Halal","data_type":"select","options":"cash"}`, wantCode: "invalid_body"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			rec := serve(t, newRouter(svc), jsonRequest(http.MethodPost, tc.target, tc.body))
			assertError(t, rec, http.StatusBadRequest, tc.wantCode)
		})
	}
}

func TestHandler_CreateCategoryField_MapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "invalid key", err: place.ErrInvalidFieldKey, wantStatus: http.StatusBadRequest, wantCode: "validation_error"},
		{name: "invalid label", err: place.ErrInvalidFieldLabel, wantStatus: http.StatusBadRequest, wantCode: "validation_error"},
		{name: "invalid sort order", err: place.ErrInvalidFieldSortOrder, wantStatus: http.StatusBadRequest, wantCode: "validation_error"},
		{name: "invalid data type", err: place.ErrInvalidFieldDataType, wantStatus: http.StatusBadRequest, wantCode: "validation_error"},
		{name: "select without options", err: place.ErrFieldOptionsRequired, wantStatus: http.StatusBadRequest, wantCode: "validation_error"},
		{name: "options on a non-select", err: place.ErrFieldOptionsNotAllowed, wantStatus: http.StatusBadRequest, wantCode: "validation_error"},
		{name: "unknown category", err: place.ErrCategoryNotFound, wantStatus: http.StatusNotFound, wantCode: "category_not_found"},
		{name: "duplicate key", err: place.ErrCategoryFieldExists, wantStatus: http.StatusConflict, wantCode: "category_field_exists"},
		{name: "storage failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().CreateCategoryField(gomock.Any(), gomock.Any()).Return(nil, tc.err)

			rec := serve(t, newRouter(svc), jsonRequest(http.MethodPost, "/categories/3/fields",
				`{"key":"halal","label":"Halal","data_type":"boolean"}`))
			assertError(t, rec, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestHandler_CreateCategoryField_UsesPathIDAndIgnoresUnknownKeys(t *testing.T) {
	svc := mock.NewMockService(gomock.NewController(t))

	var got *place.CategoryField
	svc.EXPECT().CreateCategoryField(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, f *place.CategoryField) (*place.CategoryField, error) {
			got = f
			return &place.CategoryField{ID: 9, CategoryID: f.CategoryID, Key: f.Key, Label: f.Label, DataType: f.DataType}, nil
		})

	body := `{"key":"halal","label":"Halal","data_type":"boolean","category_id":99,"id":5,"unexpected":1}`
	rec := serve(t, newRouter(svc), jsonRequest(http.MethodPost, "/categories/3/fields", body))

	if got == nil {
		t.Fatal("service was not called")
	}
	if got.CategoryID != 3 {
		t.Errorf("category id = %d, want 3 from the path", got.CategoryID)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body: %s", rec.Code, rec.Body)
	}

	var created map[string]any
	if err := json.Unmarshal(responseData(t, rec), &created); err != nil {
		t.Fatalf("data is not JSON: %v", err)
	}
	if created["id"] != float64(9) || created["category_id"] != float64(3) || created["key"] != "halal" {
		t.Errorf("created field = %v, want id 9, category_id 3, key halal", created)
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
			assertError(t, rec, http.StatusBadRequest, "invalid_body")
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
		{name: "blank name", err: place.ErrInvalidName, wantStatus: http.StatusBadRequest, wantCode: "validation_error"},
		{name: "unknown sub-district", err: place.ErrSubDistrictNotFound, wantStatus: http.StatusBadRequest, wantCode: "validation_error"},
		{name: "unknown category", err: place.ErrCategoryNotFound, wantStatus: http.StatusBadRequest, wantCode: "validation_error"},
		{name: "invalid attributes", err: place.ErrInvalidAttributes, wantStatus: http.StatusBadRequest, wantCode: "validation_error"},
		{name: "wrapped invalid attributes", err: fmt.Errorf("%w: %q is required", place.ErrInvalidAttributes, "contact"), wantStatus: http.StatusBadRequest, wantCode: "validation_error"},
		{name: "storage failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
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

func TestHandler_CreatePlace_NilResultIsAnInternalError(t *testing.T) {
	svc := mock.NewMockService(gomock.NewController(t))
	svc.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, nil)

	rec := serve(t, newRouter(svc), jsonRequest(http.MethodPost, "/places",
		`{"name":"Kopi","sub_district_id":1,"category_id":2}`))
	assertError(t, rec, http.StatusInternalServerError, "internal_error")
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
				Category:      place.Category{ID: 2, Name: "Kopi", Slug: "kopi"},
			}, nil
		})

	body := `{"name":"Kopi Senja","sub_district_id":1,"category_id":2,` +
		`"location":{"type":"Point","coordinates":[106.8,-6.2]},"id":77,"created_at":"yesterday"}`
	rec := serve(t, newRouter(svc), jsonRequest(http.MethodPost, "/places", body))

	if got == nil {
		t.Fatal("service was not called")
	}
	if got.ID != 0 {
		t.Errorf("client-sent id reached the service: %d", got.ID)
	}
	if got.Address != nil || got.Attributes != nil {
		t.Errorf("absent optional fields reached the service as %v / %s", got.Address, got.Attributes)
	}
	if got.Location == nil || got.Location.X() != 106.8 || got.Location.Y() != -6.2 {
		t.Fatalf("location reached the service as %v, want [106.8, -6.2]", got.Location)
	}

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
		"category": {"id": 2, "name": "Kopi", "slug": "kopi"}
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
						t.Error("service was not given the request's context")
						return nil, errors.New("timed out")
					}
				})

			req := jsonRequest(http.MethodPost, "/places", `{"name":"Kopi","sub_district_id":1,"category_id":2}`).WithContext(ctx)
			rec := serve(t, newRouter(svc), req)
			if rec.Code == http.StatusCreated {
				t.Error("status 201 for a request whose context had ended")
			}
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
				s.EXPECT().GetMany(gomock.Any(), gomock.Any()).Return(nil, dbErr)
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
			name:   "create category",
			method: http.MethodPost,
			target: "/categories",
			body:   `{"name":"Kopi","slug":"kopi"}`,
			fail: func(s *mock.MockService) {
				s.EXPECT().CreateCategory(gomock.Any(), gomock.Any()).Return(nil, dbErr)
			},
		},
		{
			name:   "create field",
			method: http.MethodPost,
			target: "/categories/1/fields",
			body:   `{"key":"halal","label":"Halal","data_type":"boolean"}`,
			fail: func(s *mock.MockService) {
				s.EXPECT().CreateCategoryField(gomock.Any(), gomock.Any()).Return(nil, dbErr)
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
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			tc.fail(svc)

			rec := serve(t, newRouter(svc), jsonRequest(tc.method, tc.target, tc.body))
			assertError(t, rec, http.StatusInternalServerError, "internal_error")
			if strings.Contains(rec.Body.String(), "pq:") || strings.Contains(rec.Body.String(), "relation") {
				t.Errorf("500 body exposes storage detail: %s", rec.Body)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// PATCH /places/{id}
// -----------------------------------------------------------------------------

func TestHandler_PatchPlace_RejectsInvalidRequests(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		body     string
		wantCode string
	}{
		{name: "place id is not a number", target: "/places/abc", body: `{"name":"Kopi"}`, wantCode: "invalid_place_id"},
		{name: "place id is fractional", target: "/places/1.5", body: `{"name":"Kopi"}`, wantCode: "invalid_place_id"},
		{name: "place id is zero", target: "/places/0", body: `{"name":"Kopi"}`, wantCode: "invalid_place_id"},
		{name: "place id is negative", target: "/places/-1", body: `{"name":"Kopi"}`, wantCode: "invalid_place_id"},
		{name: "empty body", target: "/places/1", body: ``, wantCode: "invalid_body"},
		{name: "truncated JSON", target: "/places/1", body: `{"name":"Kopi"`, wantCode: "invalid_body"},
		{name: "body is an array", target: "/places/1", body: `[]`, wantCode: "invalid_body"},
		{name: "body is null", target: "/places/1", body: `null`, wantCode: "invalid_body"},
		{name: "name is a number", target: "/places/1", body: `{"name":5}`, wantCode: "invalid_body"},
		{name: "name is null", target: "/places/1", body: `{"name":null}`, wantCode: "invalid_body"},
		{name: "sub_district_id is null", target: "/places/1", body: `{"sub_district_id":null}`, wantCode: "invalid_body"},
		{name: "category_id is null", target: "/places/1", body: `{"category_id":null}`, wantCode: "invalid_body"},
		{name: "sub_district_id is a string", target: "/places/1", body: `{"sub_district_id":"one"}`, wantCode: "invalid_body"},
		{name: "location is a line", target: "/places/1",
			body: `{"location":{"type":"LineString","coordinates":[[106.8,-6.2],[106.9,-6.3]]}}`, wantCode: "invalid_body"},
		{name: "location coordinates are swapped", target: "/places/1",
			body: `{"location":{"type":"Point","coordinates":[-6.2,106.8]}}`, wantCode: "invalid_body"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// No expectations: an invalid request must never reach the service.
			svc := mock.NewMockService(gomock.NewController(t))
			rec := serve(t, newRouter(svc), jsonRequest(http.MethodPatch, tc.target, tc.body))
			assertError(t, rec, http.StatusBadRequest, tc.wantCode)
		})
	}
}

func TestHandler_PatchPlace_PassesPatchToService(t *testing.T) {
	tests := []struct {
		name   string
		target string
		body   string
		wantID int64
		want   place.PlacePatch
	}{
		{name: "empty body changes nothing", target: "/places/9", body: `{}`, wantID: 9, want: place.PlacePatch{}},
		{name: "name is passed as sent", target: "/places/9", body: `{"name":"  Kopi Senja  "}`, wantID: 9,
			want: place.PlacePatch{Name: present("  Kopi Senja  ")}},
		{name: "null clears an optional field", target: "/places/9", body: `{"address":null}`, wantID: 9,
			want: place.PlacePatch{Address: null[string]()}},
		{name: "text is replaced", target: "/places/9", body: `{"operating_hours":"08:00-17:00"}`, wantID: 9,
			want: place.PlacePatch{OperatingHours: present("08:00-17:00")}},
		{name: "location is replaced", target: "/places/9", body: `{"location":{"type":"Point","coordinates":[106.8,-6.2]}}`, wantID: 9,
			want: place.PlacePatch{Location: present(*postgis.MustPoint(106.8, -6.2))}},
		{name: "null clears location", target: "/places/9", body: `{"location":null}`, wantID: 9,
			want: place.PlacePatch{Location: null[postgis.Point]()}},
		{name: "sub-district and category are replaced", target: "/places/9", body: `{"sub_district_id":2,"category_id":3}`, wantID: 9,
			want: place.PlacePatch{SubDistrictID: present(int64(2)), CategoryID: present(int64(3))}},
		{name: "attributes are passed as sent", target: "/places/9", body: `{"attributes":{"contact":"0812"}}`, wantID: 9,
			want: place.PlacePatch{Attributes: present(json.RawMessage(`{"contact":"0812"}`))}},
		{name: "null clears attributes", target: "/places/9", body: `{"attributes":null}`, wantID: 9,
			want: place.PlacePatch{Attributes: null[json.RawMessage]()}},
		{name: "client id and unknown keys are ignored", target: "/places/9",
			body: `{"id":99,"created_at":"yesterday","name":"Kopi"}`, wantID: 9,
			want: place.PlacePatch{Name: present("Kopi")}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().Update(gomock.Any(), tc.wantID, tc.want).Return(&place.Place{ID: tc.wantID}, nil)

			rec := serve(t, newRouter(svc), jsonRequest(http.MethodPatch, tc.target, tc.body))
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body)
			}
		})
	}
}

func TestHandler_PatchPlace_MapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "unknown place", err: place.ErrPlaceNotFound, wantStatus: http.StatusNotFound, wantCode: "place_not_found"},
		{name: "wrapped unknown place", err: fmt.Errorf("update: %w", place.ErrPlaceNotFound), wantStatus: http.StatusNotFound, wantCode: "place_not_found"},
		{name: "blank name", err: place.ErrInvalidName, wantStatus: http.StatusBadRequest, wantCode: "validation_error"},
		{name: "name too long", err: place.ErrPlaceNameTooLong, wantStatus: http.StatusBadRequest, wantCode: "validation_error"},
		{name: "unknown sub-district", err: place.ErrSubDistrictNotFound, wantStatus: http.StatusBadRequest, wantCode: "validation_error"},
		{name: "unknown category", err: place.ErrCategoryNotFound, wantStatus: http.StatusBadRequest, wantCode: "validation_error"},
		{name: "invalid attributes", err: place.ErrInvalidAttributes, wantStatus: http.StatusBadRequest, wantCode: "validation_error"},
		{name: "wrapped invalid attributes", err: fmt.Errorf("%w: %q is required", place.ErrInvalidAttributes, "contact"), wantStatus: http.StatusBadRequest, wantCode: "validation_error"},
		{name: "location outside Indonesia", err: place.ErrInvalidLocation, wantStatus: http.StatusBadRequest, wantCode: "validation_error"},
		{name: "storage failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().Update(gomock.Any(), int64(1), gomock.Any()).Return(nil, tc.err)

			rec := serve(t, newRouter(svc), jsonRequest(http.MethodPatch, "/places/1", `{"name":"Kopi"}`))
			assertError(t, rec, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestHandler_PatchPlace_NilResultIsAnInternalError(t *testing.T) {
	svc := mock.NewMockService(gomock.NewController(t))
	svc.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)

	rec := serve(t, newRouter(svc), jsonRequest(http.MethodPatch, "/places/1", `{"name":"Kopi"}`))
	assertError(t, rec, http.StatusInternalServerError, "internal_error")
}

func TestHandler_PatchPlace_Success(t *testing.T) {
	svc := mock.NewMockService(gomock.NewController(t))
	svc.EXPECT().Update(gomock.Any(), int64(3), place.PlacePatch{Name: present("Kopi Senja")}).Return(&place.Place{
		ID:            3,
		SubDistrictID: 1,
		Name:          "Kopi Senja",
		Attributes:    json.RawMessage(`{}`),
		Category:      place.Category{ID: 2, Name: "Kopi", Slug: "kopi"},
	}, nil)

	rec := serve(t, newRouter(svc), jsonRequest(http.MethodPatch, "/places/3", `{"name":"Kopi Senja"}`))
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
		"category": {"id": 2, "name": "Kopi", "slug": "kopi"}
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
			assertError(t, rec, http.StatusBadRequest, "invalid_place_id")
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
		{name: "unknown place", err: place.ErrPlaceNotFound, wantStatus: http.StatusNotFound, wantCode: "place_not_found"},
		{name: "wrapped unknown place", err: fmt.Errorf("delete: %w", place.ErrPlaceNotFound), wantStatus: http.StatusNotFound, wantCode: "place_not_found"},
		{name: "storage failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
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
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204; body: %s", rec.Code, rec.Body)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", rec.Body)
	}
}
