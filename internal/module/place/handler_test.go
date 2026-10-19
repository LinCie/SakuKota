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
		{"sub_district_id is not a number", "/places?sub_district_id=abc", "invalid_sub_district_id"},
		{"sub_district_id is fractional", "/places?sub_district_id=1.5", "invalid_sub_district_id"},
		{"sub_district_id is zero", "/places?sub_district_id=0", "invalid_sub_district_id"},
		{"sub_district_id is negative", "/places?sub_district_id=-3", "invalid_sub_district_id"},
		{"sub_district_id overflows 64 bits", "/places?sub_district_id=99999999999999999999", "invalid_sub_district_id"},
		{"category_id has a non-number", "/places?category_id=1,x", "invalid_category_id"},
		{"category_id has an empty element", "/places?category_id=1,,2", "invalid_category_id"},
		{"category_id ends with a comma", "/places?category_id=1,", "invalid_category_id"},
		{"category_id is zero", "/places?category_id=0", "invalid_category_id"},
		{"category_id is negative", "/places?category_id=-1", "invalid_category_id"},
		{"category_id has a zero element", "/places?category_id=1,0", "invalid_category_id"},
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
		{"no filters", "/places", place.PlaceFilter{}},
		{"query is trimmed", "/places?q=%20%20kopi%20%20", place.PlaceFilter{Query: "kopi"}},
		{"whitespace-only query means no search", "/places?q=%20%20%20", place.PlaceFilter{}},
		{"empty parameters are absent", "/places?sub_district_id=&category_id=", place.PlaceFilter{}},
		{"all filters together", "/places?sub_district_id=7&category_id=1,2&q=kopi",
			place.PlaceFilter{SubDistrictID: 7, CategoryIDs: []int64{1, 2}, Query: "kopi"}},
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
		{"unknown sub-district", place.ErrSubDistrictNotFound, http.StatusNotFound, "sub_district_not_found"},
		{"unknown category", place.ErrCategoryNotFound, http.StatusNotFound, "category_not_found"},
		{"wrapped unknown category", fmt.Errorf("filter: %w", place.ErrCategoryNotFound), http.StatusNotFound, "category_not_found"},
		{"storage failure", errors.New("storage failure"), http.StatusInternalServerError, "internal_error"},
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
		{"no categories", []place.Category{}, `[]`},
		{
			"a category without fields lists an empty fields array",
			[]place.Category{{ID: 1, Name: "Warung", Slug: "warung"}},
			`[{"id":1,"name":"Warung","slug":"warung","fields":[]}]`,
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
		{"empty body", ``},
		{"truncated JSON", `{"name":"Kopi","slug":"kopi"`},
		{"name is a number", `{"name":5,"slug":"kopi"}`},
		{"body is an array", `[]`},
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
		{"invalid name", place.ErrInvalidCategoryName, http.StatusBadRequest, "validation_error"},
		{"invalid slug", place.ErrInvalidSlug, http.StatusBadRequest, "validation_error"},
		{"duplicate name or slug", place.ErrCategoryExists, http.StatusConflict, "category_exists"},
		{"wrapped duplicate", fmt.Errorf("create: %w", place.ErrCategoryExists), http.StatusConflict, "category_exists"},
		{"storage failure", errors.New("storage failure"), http.StatusInternalServerError, "internal_error"},
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
		{"category id is not a number", "/categories/abc/fields", validBody, "invalid_category_id"},
		{"truncated JSON", "/categories/1/fields", `{"key":"halal"`, "invalid_body"},
		{"options is not a list", "/categories/1/fields", `{"key":"halal","label":"Halal","data_type":"select","options":"cash"}`, "invalid_body"},
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
		{"invalid key", place.ErrInvalidFieldKey, http.StatusBadRequest, "validation_error"},
		{"invalid label", place.ErrInvalidFieldLabel, http.StatusBadRequest, "validation_error"},
		{"invalid sort order", place.ErrInvalidFieldSortOrder, http.StatusBadRequest, "validation_error"},
		{"invalid data type", place.ErrInvalidFieldDataType, http.StatusBadRequest, "validation_error"},
		{"select without options", place.ErrFieldOptionsRequired, http.StatusBadRequest, "validation_error"},
		{"options on a non-select", place.ErrFieldOptionsNotAllowed, http.StatusBadRequest, "validation_error"},
		{"unknown category", place.ErrCategoryNotFound, http.StatusNotFound, "category_not_found"},
		{"duplicate key", place.ErrCategoryFieldExists, http.StatusConflict, "category_field_exists"},
		{"storage failure", errors.New("storage failure"), http.StatusInternalServerError, "internal_error"},
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
		{"empty body", ``},
		{"truncated JSON", `{"name":"Kopi"`},
		{"sub_district_id is a string", `{"name":"Kopi","sub_district_id":"one","category_id":2}`},
		{"location is a line", prefix + `{"type":"LineString","coordinates":[[106.8,-6.2],[106.9,-6.3]]}}`},
		{"coordinates are swapped", prefix + `{"type":"Point","coordinates":[-6.2,106.8]}}`},
		{"latitude above 90", prefix + `{"type":"Point","coordinates":[106.8,91]}}`},
		{"longitude above 180", prefix + `{"type":"Point","coordinates":[181,-6.2]}}`},
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
		{"blank name", place.ErrInvalidName, http.StatusBadRequest, "validation_error"},
		{"unknown sub-district", place.ErrSubDistrictNotFound, http.StatusBadRequest, "validation_error"},
		{"unknown category", place.ErrCategoryNotFound, http.StatusBadRequest, "validation_error"},
		{"invalid attributes", place.ErrInvalidAttributes, http.StatusBadRequest, "validation_error"},
		{"wrapped invalid attributes", fmt.Errorf("%w: %q is required", place.ErrInvalidAttributes, "contact"), http.StatusBadRequest, "validation_error"},
		{"storage failure", errors.New("storage failure"), http.StatusInternalServerError, "internal_error"},
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
		{"client disconnected", func() (context.Context, context.CancelFunc) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			return ctx, cancel
		}},
		{"deadline passes while storage is slow", func() (context.Context, context.CancelFunc) {
			return context.WithTimeout(context.Background(), 20*time.Millisecond)
		}},
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
		{"list places", http.MethodGet, "/places", "", func(s *mock.MockService) {
			s.EXPECT().GetMany(gomock.Any(), gomock.Any()).Return(nil, dbErr)
		}},
		{"list categories", http.MethodGet, "/categories", "", func(s *mock.MockService) {
			s.EXPECT().GetCategories(gomock.Any()).Return(nil, dbErr)
		}},
		{"create category", http.MethodPost, "/categories", `{"name":"Kopi","slug":"kopi"}`, func(s *mock.MockService) {
			s.EXPECT().CreateCategory(gomock.Any(), gomock.Any()).Return(nil, dbErr)
		}},
		{"create field", http.MethodPost, "/categories/1/fields", `{"key":"halal","label":"Halal","data_type":"boolean"}`, func(s *mock.MockService) {
			s.EXPECT().CreateCategoryField(gomock.Any(), gomock.Any()).Return(nil, dbErr)
		}},
		{"create place", http.MethodPost, "/places", `{"name":"Kopi","sub_district_id":1,"category_id":2}`, func(s *mock.MockService) {
			s.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, dbErr)
		}},
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
		{"place id is not a number", "/places/abc", `{"name":"Kopi"}`, "invalid_place_id"},
		{"place id is fractional", "/places/1.5", `{"name":"Kopi"}`, "invalid_place_id"},
		{"place id is zero", "/places/0", `{"name":"Kopi"}`, "invalid_place_id"},
		{"place id is negative", "/places/-1", `{"name":"Kopi"}`, "invalid_place_id"},
		{"empty body", "/places/1", ``, "invalid_body"},
		{"truncated JSON", "/places/1", `{"name":"Kopi"`, "invalid_body"},
		{"body is an array", "/places/1", `[]`, "invalid_body"},
		{"body is null", "/places/1", `null`, "invalid_body"},
		{"name is a number", "/places/1", `{"name":5}`, "invalid_body"},
		{"name is null", "/places/1", `{"name":null}`, "invalid_body"},
		{"sub_district_id is null", "/places/1", `{"sub_district_id":null}`, "invalid_body"},
		{"category_id is null", "/places/1", `{"category_id":null}`, "invalid_body"},
		{"sub_district_id is a string", "/places/1", `{"sub_district_id":"one"}`, "invalid_body"},
		{"location is a line", "/places/1",
			`{"location":{"type":"LineString","coordinates":[[106.8,-6.2],[106.9,-6.3]]}}`, "invalid_body"},
		{"location coordinates are swapped", "/places/1",
			`{"location":{"type":"Point","coordinates":[-6.2,106.8]}}`, "invalid_body"},
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
		{"empty body changes nothing", "/places/9", `{}`, 9, place.PlacePatch{}},
		{"name is passed as sent", "/places/9", `{"name":"  Kopi Senja  "}`, 9,
			place.PlacePatch{Name: present("  Kopi Senja  ")}},
		{"null clears an optional field", "/places/9", `{"address":null}`, 9,
			place.PlacePatch{Address: null[string]()}},
		{"text is replaced", "/places/9", `{"operating_hours":"08:00-17:00"}`, 9,
			place.PlacePatch{OperatingHours: present("08:00-17:00")}},
		{"location is replaced", "/places/9", `{"location":{"type":"Point","coordinates":[106.8,-6.2]}}`, 9,
			place.PlacePatch{Location: present(*postgis.MustPoint(106.8, -6.2))}},
		{"null clears location", "/places/9", `{"location":null}`, 9,
			place.PlacePatch{Location: null[postgis.Point]()}},
		{"sub-district and category are replaced", "/places/9", `{"sub_district_id":2,"category_id":3}`, 9,
			place.PlacePatch{SubDistrictID: present(int64(2)), CategoryID: present(int64(3))}},
		{"attributes are passed as sent", "/places/9", `{"attributes":{"contact":"0812"}}`, 9,
			place.PlacePatch{Attributes: present(json.RawMessage(`{"contact":"0812"}`))}},
		{"null clears attributes", "/places/9", `{"attributes":null}`, 9,
			place.PlacePatch{Attributes: null[json.RawMessage]()}},
		{"client id and unknown keys are ignored", "/places/9",
			`{"id":99,"created_at":"yesterday","name":"Kopi"}`, 9,
			place.PlacePatch{Name: present("Kopi")}},
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
		{"unknown place", place.ErrPlaceNotFound, http.StatusNotFound, "place_not_found"},
		{"wrapped unknown place", fmt.Errorf("update: %w", place.ErrPlaceNotFound), http.StatusNotFound, "place_not_found"},
		{"blank name", place.ErrInvalidName, http.StatusBadRequest, "validation_error"},
		{"name too long", place.ErrPlaceNameTooLong, http.StatusBadRequest, "validation_error"},
		{"unknown sub-district", place.ErrSubDistrictNotFound, http.StatusBadRequest, "validation_error"},
		{"unknown category", place.ErrCategoryNotFound, http.StatusBadRequest, "validation_error"},
		{"invalid attributes", place.ErrInvalidAttributes, http.StatusBadRequest, "validation_error"},
		{"wrapped invalid attributes", fmt.Errorf("%w: %q is required", place.ErrInvalidAttributes, "contact"), http.StatusBadRequest, "validation_error"},
		{"location outside Indonesia", place.ErrInvalidLocation, http.StatusBadRequest, "validation_error"},
		{"storage failure", errors.New("storage failure"), http.StatusInternalServerError, "internal_error"},
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
		{"place id is not a number", "/places/abc"},
		{"place id is fractional", "/places/1.5"},
		{"place id is zero", "/places/0"},
		{"place id is negative", "/places/-1"},
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
		{"unknown place", place.ErrPlaceNotFound, http.StatusNotFound, "place_not_found"},
		{"wrapped unknown place", fmt.Errorf("delete: %w", place.ErrPlaceNotFound), http.StatusNotFound, "place_not_found"},
		{"storage failure", errors.New("storage failure"), http.StatusInternalServerError, "internal_error"},
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
