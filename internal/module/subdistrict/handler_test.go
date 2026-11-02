package subdistrict_test

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

	"rantau-go/internal/module/subdistrict"
	"rantau-go/internal/module/subdistrict/mock"
	"rantau-go/internal/storage"
)

// newRouter serves the sub-district handlers on the paths the API uses for
// them, so URL parameters resolve as they do in production.
func newRouter(svc subdistrict.Service) http.Handler {
	h := subdistrict.NewHandler(svc, testImageURL)
	r := chi.NewRouter()
	r.Get("/subdistricts", h.GetSubDistricts)
	r.Get("/subdistricts/{id}", h.GetSubDistrict)
	r.Post("/subdistricts", h.CreateSubDistrict)
	r.Patch("/subdistricts/{id}", h.PatchSubDistrict)
	r.Delete("/subdistricts/{id}", h.DeleteSubDistrict)
	r.Post("/subdistricts/{id}/image/upload-url", h.CreateImageUpload(subdistrict.SlotImage))
	r.Put("/subdistricts/{id}/image", h.SetImage(subdistrict.SlotImage))
	r.Post("/subdistricts/{id}/static-map/upload-url", h.CreateImageUpload(subdistrict.SlotStaticMap))
	r.Put("/subdistricts/{id}/static-map", h.SetImage(subdistrict.SlotStaticMap))
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
// GET /subdistricts
// -----------------------------------------------------------------------------

func TestHandler_GetSubDistricts_ResponseShape(t *testing.T) {
	tests := []struct {
		name         string
		subDistricts []subdistrict.SubDistrict
		want         string
	}{
		{name: "no sub-districts", subDistricts: []subdistrict.SubDistrict{}, want: `[]`},
		{
			name: "every field is listed, unset ones as null",
			subDistricts: []subdistrict.SubDistrict{
				{ID: 1, Name: "Menteng", Slug: "menteng", FloodHistory: ptr("Banjir 2020"),
					EstimatedDailySpending: ptr(1500000.5), ImageKey: ptr("subdistricts/1/image/a.webp"),
					StaticMapImageKey: ptr("subdistricts/1/static-map/a.png")},
				{ID: 2, Name: "Kemang", Slug: "kemang"},
			},
			want: `[
				{"id": 1, "name": "Menteng", "slug": "menteng", "flood_history": "Banjir 2020",
				 "safety_crime_information": null, "estimated_daily_spending": 1500000.5,
				 "image_url": "https://img.test/subdistricts/1/image/a.webp",
				 "static_map_image_url": "https://img.test/subdistricts/1/static-map/a.png"},
				{"id": 2, "name": "Kemang", "slug": "kemang", "flood_history": null,
				 "safety_crime_information": null, "estimated_daily_spending": null,
				 "image_url": null, "static_map_image_url": null}
			]`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().GetMany(gomock.Any()).Return(tc.subDistricts, nil)

			rec := serve(t, newRouter(svc), jsonRequest(http.MethodGet, "/subdistricts", ""))
			assertData(t, rec, http.StatusOK, tc.want)
		})
	}
}

// -----------------------------------------------------------------------------
// GET /subdistricts/{id}
// -----------------------------------------------------------------------------

func TestHandler_GetSubDistrict_RejectsInvalidID(t *testing.T) {
	tests := []struct {
		name   string
		target string
	}{
		{name: "id is not a number", target: "/subdistricts/abc"},
		{name: "id is fractional", target: "/subdistricts/1.5"},
		{name: "id is zero", target: "/subdistricts/0"},
		{name: "id is negative", target: "/subdistricts/-1"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// No expectations: an invalid id must never reach the service.
			svc := mock.NewMockService(gomock.NewController(t))
			rec := serve(t, newRouter(svc), jsonRequest(http.MethodGet, tc.target, ""))
			assertError(t, rec, http.StatusBadRequest, "INVALID_SUB_DISTRICT_ID")
		})
	}
}

func TestHandler_GetSubDistrict_MapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "unknown sub-district", err: subdistrict.ErrSubDistrictNotFound, wantStatus: http.StatusNotFound, wantCode: "SUB_DISTRICT_NOT_FOUND"},
		{name: "wrapped unknown sub-district", err: fmt.Errorf("get: %w", subdistrict.ErrSubDistrictNotFound), wantStatus: http.StatusNotFound, wantCode: "SUB_DISTRICT_NOT_FOUND"},
		{name: "storage failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().GetSubDistrict(gomock.Any(), int64(5)).Return(nil, tc.err)

			rec := serve(t, newRouter(svc), jsonRequest(http.MethodGet, "/subdistricts/5", ""))
			assertError(t, rec, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestHandler_GetSubDistrict_Success(t *testing.T) {
	svc := mock.NewMockService(gomock.NewController(t))
	svc.EXPECT().GetSubDistrict(gomock.Any(), int64(3)).Return(&subdistrict.SubDistrict{
		ID:           3,
		Name:         "Menteng",
		Slug:         "menteng",
		FloodHistory: ptr("Banjir 2020"),
	}, nil)

	rec := serve(t, newRouter(svc), jsonRequest(http.MethodGet, "/subdistricts/3", ""))
	assertData(t, rec, http.StatusOK, `{
		"id": 3,
		"name": "Menteng",
		"slug": "menteng",
		"flood_history": "Banjir 2020",
		"safety_crime_information": null,
		"estimated_daily_spending": null,
		"image_url": null,
		"static_map_image_url": null
	}`)
}

// -----------------------------------------------------------------------------
// POST /subdistricts
// -----------------------------------------------------------------------------

func TestHandler_CreateSubDistrict_RejectsMalformedBodies(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "empty body", body: ``},
		{name: "body is null", body: `null`},
		{name: "truncated JSON", body: `{"name":"Menteng"`},
		{name: "name is a number", body: `{"name":5,"slug":"menteng"}`},
		{name: "estimated daily spending is a string", body: `{"name":"Menteng","slug":"menteng","estimated_daily_spending":"1.5"}`},
		{name: "body is an array", body: `[]`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// No expectations: a malformed body must never reach the service.
			svc := mock.NewMockService(gomock.NewController(t))
			rec := serve(t, newRouter(svc), jsonRequest(http.MethodPost, "/subdistricts", tc.body))
			assertError(t, rec, http.StatusBadRequest, "INVALID_BODY")
		})
	}
}

func TestHandler_CreateSubDistrict_MapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "invalid name", err: subdistrict.ErrInvalidName, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "invalid slug", err: subdistrict.ErrInvalidSlug, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "invalid estimated daily spending", err: subdistrict.ErrInvalidEstimatedDailySpending, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "duplicate name or slug", err: subdistrict.ErrSubDistrictExists, wantStatus: http.StatusConflict, wantCode: "SUB_DISTRICT_EXISTS"},
		{name: "wrapped duplicate", err: fmt.Errorf("create: %w", subdistrict.ErrSubDistrictExists), wantStatus: http.StatusConflict, wantCode: "SUB_DISTRICT_EXISTS"},
		{name: "storage failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, tc.err)

			rec := serve(t, newRouter(svc), jsonRequest(http.MethodPost, "/subdistricts",
				`{"name":"Menteng","slug":"menteng"}`))
			assertError(t, rec, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestHandler_CreateSubDistrict_Success(t *testing.T) {
	svc := mock.NewMockService(gomock.NewController(t))

	var got *subdistrict.SubDistrict
	svc.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, sd *subdistrict.SubDistrict) (*subdistrict.SubDistrict, error) {
			got = sd
			return &subdistrict.SubDistrict{
				ID:                     42,
				Name:                   sd.Name,
				Slug:                   sd.Slug,
				FloodHistory:           sd.FloodHistory,
				SafetyCrimeInformation: sd.SafetyCrimeInformation,
				EstimatedDailySpending: sd.EstimatedDailySpending,
			}, nil
		})

	body := `{"name":"Menteng","slug":"menteng","flood_history":"Banjir 2020","safety_crime_information":null,` +
		`"estimated_daily_spending":1500000.5,"static_map_image_url":"https://example.com/menteng.png",` +
		`"id":77,"created_at":"yesterday"}`
	rec := serve(t, newRouter(svc), jsonRequest(http.MethodPost, "/subdistricts", body))

	require.NotNil(t, got, "service was not called")
	want := subdistrict.SubDistrict{
		Name:                   "Menteng",
		Slug:                   "menteng",
		FloodHistory:           ptr("Banjir 2020"),
		EstimatedDailySpending: ptr(1500000.5),
	}
	assert.Equal(t, want, *got, "a client-sent id or unknown key reached the service")

	assertData(t, rec, http.StatusCreated, `{
		"id": 42,
		"name": "Menteng",
		"slug": "menteng",
		"flood_history": "Banjir 2020",
		"safety_crime_information": null,
		"estimated_daily_spending": 1500000.5,
		"image_url": null,
		"static_map_image_url": null
	}`)
}

func TestHandler_CreateSubDistrict_ContextReachesService(t *testing.T) {
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
				func(got context.Context, _ *subdistrict.SubDistrict) (*subdistrict.SubDistrict, error) {
					select {
					case <-got.Done():
						return nil, got.Err()
					case <-time.After(2 * time.Second):
						assert.Fail(t, "service was not given the request's context")
						return nil, errors.New("timed out")
					}
				})

			req := jsonRequest(http.MethodPost, "/subdistricts", `{"name":"Menteng","slug":"menteng"}`).WithContext(ctx)
			rec := serve(t, newRouter(svc), req)
			assert.NotEqual(t, http.StatusCreated, rec.Code, "status 201 for a request whose context had ended")
		})
	}
}

// -----------------------------------------------------------------------------
// Cross-cutting: internal details
// -----------------------------------------------------------------------------

func TestHandler_InternalErrorsDoNotLeakDetails(t *testing.T) {
	dbErr := errors.New(`pq: relation "sub_districts" does not exist`)

	tests := []struct {
		name   string
		method string
		target string
		body   string
		fail   func(svc *mock.MockService)
	}{
		{
			name:   "list sub-districts",
			method: http.MethodGet,
			target: "/subdistricts",
			body:   "",
			fail: func(s *mock.MockService) {
				s.EXPECT().GetMany(gomock.Any()).Return(nil, dbErr)
			},
		},
		{
			name:   "get sub-district",
			method: http.MethodGet,
			target: "/subdistricts/1",
			body:   "",
			fail: func(s *mock.MockService) {
				s.EXPECT().GetSubDistrict(gomock.Any(), int64(1)).Return(nil, dbErr)
			},
		},
		{
			name:   "create sub-district",
			method: http.MethodPost,
			target: "/subdistricts",
			body:   `{"name":"Menteng","slug":"menteng"}`,
			fail: func(s *mock.MockService) {
				s.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, dbErr)
			},
		},
		{
			name:   "update sub-district",
			method: http.MethodPatch,
			target: "/subdistricts/1",
			body:   `{"name":"Menteng Atas"}`,
			fail: func(s *mock.MockService) {
				s.EXPECT().Update(gomock.Any(), int64(1), gomock.Any()).Return(nil, dbErr)
			},
		},
		{
			name:   "delete sub-district",
			method: http.MethodDelete,
			target: "/subdistricts/1",
			body:   "",
			fail: func(s *mock.MockService) {
				s.EXPECT().Delete(gomock.Any(), int64(1)).Return(dbErr)
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
// PATCH /subdistricts/{id}
// -----------------------------------------------------------------------------

func TestHandler_PatchSubDistrict_RejectsInvalidRequests(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		body     string
		wantCode string
	}{
		{name: "id is not a number", target: "/subdistricts/abc", body: `{"name":"Kemang"}`, wantCode: "INVALID_SUB_DISTRICT_ID"},
		{name: "id is fractional", target: "/subdistricts/1.5", body: `{"name":"Kemang"}`, wantCode: "INVALID_SUB_DISTRICT_ID"},
		{name: "id is zero", target: "/subdistricts/0", body: `{"name":"Kemang"}`, wantCode: "INVALID_SUB_DISTRICT_ID"},
		{name: "id is negative", target: "/subdistricts/-1", body: `{"name":"Kemang"}`, wantCode: "INVALID_SUB_DISTRICT_ID"},
		{name: "empty body", target: "/subdistricts/1", body: ``, wantCode: "INVALID_BODY"},
		{name: "truncated JSON", target: "/subdistricts/1", body: `{"name":"Kemang"`, wantCode: "INVALID_BODY"},
		{name: "body is an array", target: "/subdistricts/1", body: `[]`, wantCode: "INVALID_BODY"},
		{name: "body is null", target: "/subdistricts/1", body: `null`, wantCode: "INVALID_BODY"},
		{name: "name is a number", target: "/subdistricts/1", body: `{"name":5}`, wantCode: "INVALID_BODY"},
		{name: "name is null", target: "/subdistricts/1", body: `{"name":null}`, wantCode: "INVALID_BODY"},
		{name: "slug is null", target: "/subdistricts/1", body: `{"slug":null}`, wantCode: "INVALID_BODY"},
		{name: "estimated daily spending is a string", target: "/subdistricts/1", body: `{"estimated_daily_spending":"1.5"}`, wantCode: "INVALID_BODY"},
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

func TestHandler_PatchSubDistrict_PassesPatchToService(t *testing.T) {
	tests := []struct {
		name   string
		target string
		body   string
		wantID int64
		want   subdistrict.SubDistrictPatch
	}{
		{name: "empty body changes nothing", target: "/subdistricts/9", body: `{}`, wantID: 9,
			want: subdistrict.SubDistrictPatch{}},
		{name: "name is passed as sent", target: "/subdistricts/9", body: `{"name":"  Menteng Atas  "}`, wantID: 9,
			want: subdistrict.SubDistrictPatch{Name: present("  Menteng Atas  ")}},
		{name: "slug is replaced", target: "/subdistricts/9", body: `{"slug":"menteng-atas"}`, wantID: 9,
			want: subdistrict.SubDistrictPatch{Slug: present("menteng-atas")}},
		{name: "text is replaced", target: "/subdistricts/9", body: `{"safety_crime_information":"Aman"}`, wantID: 9,
			want: subdistrict.SubDistrictPatch{SafetyCrimeInformation: present("Aman")}},
		{name: "null clears an optional text field", target: "/subdistricts/9", body: `{"flood_history":null}`, wantID: 9,
			want: subdistrict.SubDistrictPatch{FloodHistory: null[string]()}},
		{name: "number is replaced", target: "/subdistricts/9", body: `{"estimated_daily_spending":750000.25}`, wantID: 9,
			want: subdistrict.SubDistrictPatch{EstimatedDailySpending: present(750000.25)}},
		{name: "null clears the number", target: "/subdistricts/9", body: `{"estimated_daily_spending":null}`, wantID: 9,
			want: subdistrict.SubDistrictPatch{EstimatedDailySpending: null[float64]()}},
		{name: "map image URL can no longer be patched", target: "/subdistricts/9", body: `{"static_map_image_url":null}`, wantID: 9,
			want: subdistrict.SubDistrictPatch{}},
		{name: "client id and unknown keys are ignored", target: "/subdistricts/9",
			body: `{"id":99,"created_at":"yesterday","slug":"menteng-atas"}`, wantID: 9,
			want: subdistrict.SubDistrictPatch{Slug: present("menteng-atas")}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().Update(gomock.Any(), tc.wantID, tc.want).Return(&subdistrict.SubDistrict{ID: tc.wantID}, nil)

			rec := serve(t, newRouter(svc), jsonRequest(http.MethodPatch, tc.target, tc.body))
			require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body)
		})
	}
}

func TestHandler_PatchSubDistrict_MapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "unknown sub-district", err: subdistrict.ErrSubDistrictNotFound, wantStatus: http.StatusNotFound, wantCode: "SUB_DISTRICT_NOT_FOUND"},
		{name: "wrapped unknown sub-district", err: fmt.Errorf("update: %w", subdistrict.ErrSubDistrictNotFound), wantStatus: http.StatusNotFound, wantCode: "SUB_DISTRICT_NOT_FOUND"},
		{name: "invalid name", err: subdistrict.ErrInvalidName, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "invalid slug", err: subdistrict.ErrInvalidSlug, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "invalid estimated daily spending", err: subdistrict.ErrInvalidEstimatedDailySpending, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "duplicate name or slug", err: subdistrict.ErrSubDistrictExists, wantStatus: http.StatusConflict, wantCode: "SUB_DISTRICT_EXISTS"},
		{name: "wrapped duplicate", err: fmt.Errorf("update: %w", subdistrict.ErrSubDistrictExists), wantStatus: http.StatusConflict, wantCode: "SUB_DISTRICT_EXISTS"},
		{name: "storage failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().Update(gomock.Any(), int64(1), gomock.Any()).Return(nil, tc.err)

			rec := serve(t, newRouter(svc), jsonRequest(http.MethodPatch, "/subdistricts/1", `{"name":"Menteng"}`))
			assertError(t, rec, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestHandler_PatchSubDistrict_Success(t *testing.T) {
	svc := mock.NewMockService(gomock.NewController(t))
	svc.EXPECT().Update(gomock.Any(), int64(3), subdistrict.SubDistrictPatch{Name: present("Menteng Atas")}).Return(&subdistrict.SubDistrict{
		ID:                3,
		Name:              "Menteng Atas",
		Slug:              "menteng",
		StaticMapImageKey: ptr("subdistricts/3/static-map/a.png"),
	}, nil)

	rec := serve(t, newRouter(svc), jsonRequest(http.MethodPatch, "/subdistricts/3", `{"name":"Menteng Atas"}`))
	assertData(t, rec, http.StatusOK, `{
		"id": 3,
		"name": "Menteng Atas",
		"slug": "menteng",
		"flood_history": null,
		"safety_crime_information": null,
		"estimated_daily_spending": null,
		"image_url": null,
		"static_map_image_url": "https://img.test/subdistricts/3/static-map/a.png"
	}`)
}

// -----------------------------------------------------------------------------
// DELETE /subdistricts/{id}
// -----------------------------------------------------------------------------

func TestHandler_DeleteSubDistrict_RejectsInvalidID(t *testing.T) {
	tests := []struct {
		name   string
		target string
	}{
		{name: "id is not a number", target: "/subdistricts/abc"},
		{name: "id is fractional", target: "/subdistricts/1.5"},
		{name: "id is zero", target: "/subdistricts/0"},
		{name: "id is negative", target: "/subdistricts/-1"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// No expectations: an invalid id must never reach the service.
			svc := mock.NewMockService(gomock.NewController(t))
			rec := serve(t, newRouter(svc), jsonRequest(http.MethodDelete, tc.target, ""))
			assertError(t, rec, http.StatusBadRequest, "INVALID_SUB_DISTRICT_ID")
		})
	}
}

func TestHandler_DeleteSubDistrict_MapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "unknown sub-district", err: subdistrict.ErrSubDistrictNotFound, wantStatus: http.StatusNotFound, wantCode: "SUB_DISTRICT_NOT_FOUND"},
		{name: "wrapped unknown sub-district", err: fmt.Errorf("delete: %w", subdistrict.ErrSubDistrictNotFound), wantStatus: http.StatusNotFound, wantCode: "SUB_DISTRICT_NOT_FOUND"},
		{name: "still referenced by places", err: subdistrict.ErrSubDistrictInUse, wantStatus: http.StatusConflict, wantCode: "SUB_DISTRICT_IN_USE"},
		{name: "storage failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().Delete(gomock.Any(), int64(4)).Return(tc.err)

			rec := serve(t, newRouter(svc), jsonRequest(http.MethodDelete, "/subdistricts/4", ""))
			assertError(t, rec, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestHandler_DeleteSubDistrict_Success(t *testing.T) {
	svc := mock.NewMockService(gomock.NewController(t))
	svc.EXPECT().Delete(gomock.Any(), int64(4)).Return(nil)

	rec := serve(t, newRouter(svc), jsonRequest(http.MethodDelete, "/subdistricts/4", ""))
	require.Equal(t, http.StatusNoContent, rec.Code, "body: %s", rec.Body)
	assert.Empty(t, rec.Body.String(), "body should be empty")
}

// -----------------------------------------------------------------------------
// Sub-district images
// -----------------------------------------------------------------------------

// handlerSlots are the image slots with their path segment.
var handlerSlots = []struct {
	slot subdistrict.ImageSlot
	path string
}{
	{slot: subdistrict.SlotImage, path: "image"},
	{slot: subdistrict.SlotStaticMap, path: "static-map"},
}

func TestHandler_CreateImageUpload_RejectsInvalidRequests(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		body     string
		wantCode string
	}{
		{name: "sub-district id is not a number", id: "abc", body: `{"content_type":"image/png"}`, wantCode: "INVALID_SUB_DISTRICT_ID"},
		{name: "sub-district id is zero", id: "0", body: `{"content_type":"image/png"}`, wantCode: "INVALID_SUB_DISTRICT_ID"},
		{name: "body is not JSON", id: "3", body: `content_type=image/png`, wantCode: "INVALID_BODY"},
		{name: "body is null", id: "3", body: `null`, wantCode: "INVALID_BODY"},
		{name: "body is an array", id: "3", body: `[]`, wantCode: "INVALID_BODY"},
	}

	for _, hs := range handlerSlots {
		for _, tc := range tests {
			t.Run(hs.path+"/"+tc.name, func(t *testing.T) {
				// No expectations: an invalid request must never reach the service.
				svc := mock.NewMockService(gomock.NewController(t))
				target := "/subdistricts/" + tc.id + "/" + hs.path + "/upload-url"
				rec := serve(t, newRouter(svc), jsonRequest(http.MethodPost, target, tc.body))
				assertError(t, rec, http.StatusBadRequest, tc.wantCode)
			})
		}
	}
}

func TestHandler_CreateImageUpload_MapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "unknown sub-district", err: subdistrict.ErrSubDistrictNotFound, wantStatus: http.StatusNotFound, wantCode: "SUB_DISTRICT_NOT_FOUND"},
		{name: "image type not allowed", err: storage.ErrInvalidImageType, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "storage failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().ImageUploadURL(gomock.Any(), int64(3), subdistrict.SlotImage, "image/gif").Return(storage.Upload{}, tc.err)

			rec := serve(t, newRouter(svc), jsonRequest(http.MethodPost, "/subdistricts/3/image/upload-url", `{"content_type":"image/gif"}`))
			assertError(t, rec, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestHandler_CreateImageUpload_Success(t *testing.T) {
	for _, hs := range handlerSlots {
		t.Run(hs.path, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().ImageUploadURL(gomock.Any(), int64(3), hs.slot, "image/png").Return(storage.Upload{
				Key: "uploads/subdistricts/3/" + hs.path + "/a.png",
				PresignedUpload: storage.PresignedUpload{
					URL:       "http://localhost:3900/rantau-images/uploads/subdistricts/3/" + hs.path + "/a.png?X-Amz-Signature=abc",
					Method:    "PUT",
					Headers:   map[string]string{"Content-Type": "image/png"},
					ExpiresAt: time.Date(2026, 10, 31, 12, 5, 0, 0, time.UTC),
				},
			}, nil)

			rec := serve(t, newRouter(svc), jsonRequest(http.MethodPost, "/subdistricts/3/"+hs.path+"/upload-url", `{"content_type":"image/png"}`))
			assertData(t, rec, http.StatusOK, `{
				"upload_key": "uploads/subdistricts/3/`+hs.path+`/a.png",
				"upload_url": "http://localhost:3900/rantau-images/uploads/subdistricts/3/`+hs.path+`/a.png?X-Amz-Signature=abc",
				"method": "PUT",
				"headers": {"Content-Type": "image/png"},
				"expires_at": "2026-10-31T12:05:00Z"
			}`)
		})
	}
}

func TestHandler_SetImage_RejectsInvalidRequests(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		body     string
		wantCode string
	}{
		{name: "sub-district id is not a number", id: "abc", body: `{"upload_key":"k"}`, wantCode: "INVALID_SUB_DISTRICT_ID"},
		{name: "sub-district id is negative", id: "-1", body: `{"upload_key":"k"}`, wantCode: "INVALID_SUB_DISTRICT_ID"},
		{name: "body is not JSON", id: "3", body: `upload_key=k`, wantCode: "INVALID_BODY"},
		{name: "body is null", id: "3", body: `null`, wantCode: "INVALID_BODY"},
		{name: "body is an array", id: "3", body: `[]`, wantCode: "INVALID_BODY"},
	}

	for _, hs := range handlerSlots {
		for _, tc := range tests {
			t.Run(hs.path+"/"+tc.name, func(t *testing.T) {
				// No expectations: an invalid request must never reach the service.
				svc := mock.NewMockService(gomock.NewController(t))
				target := "/subdistricts/" + tc.id + "/" + hs.path
				rec := serve(t, newRouter(svc), jsonRequest(http.MethodPut, target, tc.body))
				assertError(t, rec, http.StatusBadRequest, tc.wantCode)
			})
		}
	}
}

func TestHandler_SetImage_MapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "unknown sub-district", err: subdistrict.ErrSubDistrictNotFound, wantStatus: http.StatusNotFound, wantCode: "SUB_DISTRICT_NOT_FOUND"},
		{name: "invalid upload key", err: storage.ErrInvalidImageKey, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "image not uploaded", err: storage.ErrImageNotUploaded, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "image type not allowed", err: storage.ErrInvalidImageType, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "image too large", err: storage.ErrImageTooLarge, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "storage failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().SetImage(gomock.Any(), int64(3), subdistrict.SlotStaticMap, "uploads/subdistricts/3/static-map/a.png").Return(nil, tc.err)

			rec := serve(t, newRouter(svc), jsonRequest(http.MethodPut, "/subdistricts/3/static-map",
				`{"upload_key":"uploads/subdistricts/3/static-map/a.png"}`))
			assertError(t, rec, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestHandler_SetImage_Success(t *testing.T) {
	for _, hs := range handlerSlots {
		t.Run(hs.path, func(t *testing.T) {
			svc := mock.NewMockService(gomock.NewController(t))
			svc.EXPECT().SetImage(gomock.Any(), int64(3), hs.slot, "uploads/subdistricts/3/"+hs.path+"/a.png").Return(&subdistrict.SubDistrict{
				ID:                3,
				Name:              "Menteng",
				Slug:              "menteng",
				ImageKey:          ptr("subdistricts/3/image/a.png"),
				StaticMapImageKey: ptr("subdistricts/3/static-map/a.png"),
			}, nil)

			rec := serve(t, newRouter(svc), jsonRequest(http.MethodPut, "/subdistricts/3/"+hs.path,
				`{"upload_key":"uploads/subdistricts/3/`+hs.path+`/a.png"}`))
			assertData(t, rec, http.StatusOK, `{
				"id": 3,
				"name": "Menteng",
				"slug": "menteng",
				"flood_history": null,
				"safety_crime_information": null,
				"estimated_daily_spending": null,
				"image_url": "https://img.test/subdistricts/3/image/a.png",
				"static_map_image_url": "https://img.test/subdistricts/3/static-map/a.png"
			}`)
		})
	}
}
