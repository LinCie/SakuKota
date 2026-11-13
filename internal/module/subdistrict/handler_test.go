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
	r.Route("/subdistricts", func(r chi.Router) {
		r.Get("/", h.GetSubDistricts)
		r.Get("/{id}", h.GetSubDistrict)
		r.Post("/", h.CreateSubDistrict)
		r.Put("/{id}", h.PutSubDistrict)
		r.Delete("/{id}", h.DeleteSubDistrict)
		r.Post("/{id}/medias/upload-url", h.CreateImageUpload(subdistrict.SlotMedias))
		r.Post("/{id}/medias", h.AddMedia)
		r.Delete("/{id}/medias/{mediaId}", h.DeleteMedia)
		r.Post("/{id}/static-map/upload-url", h.CreateImageUpload(subdistrict.SlotStaticMap))
		r.Put("/{id}/static-map", h.SetStaticMap)
	})
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

func TestHandler_GetSubDistricts(t *testing.T) {
	t.Run("response shape", func(t *testing.T) {
		tests := []struct {
			name         string
			subDistricts []subdistrict.SubDistrict
			want         string
		}{
			{name: "no sub-districts", subDistricts: []subdistrict.SubDistrict{}, want: `[]`},
			{
				name: "every field is listed, unset ones as null",
				subDistricts: []subdistrict.SubDistrict{
					{ID: 1, Name: "Menteng", Slug: "menteng", FloodHistory: new("Banjir 2020"),
						EstimatedDailySpending: new(1500000.5),
						Medias: []storage.Media{
							{ID: 4, Key: "subdistricts/1/medias/a.webp", ContentType: "image/webp", Size: 2048, Position: 1},
						},
						StaticMapImageKey: new("subdistricts/1/static-map/a.png")},
					{ID: 2, Name: "Kemang", Slug: "kemang"},
				},
				want: `[
					{"id": 1, "name": "Menteng", "slug": "menteng", "flood_history": "Banjir 2020",
					 "safety_crime_information": null, "estimated_daily_spending": 1500000.5,
					 "medias": [{"id": 4, "url": "https://img.test/subdistricts/1/medias/a.webp",
					             "content_type": "image/webp", "size": 2048, "position": 1}],
					 "static_map_image_url": "https://img.test/subdistricts/1/static-map/a.png"},
					{"id": 2, "name": "Kemang", "slug": "kemang", "flood_history": null,
					 "safety_crime_information": null, "estimated_daily_spending": null,
					 "medias": [], "static_map_image_url": null}
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
	})
}

func TestHandler_GetSubDistrict(t *testing.T) {
	t.Run("rejects invalid id", func(t *testing.T) {
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
	})

	t.Run("maps service errors", func(t *testing.T) {
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
	})

	t.Run("success", func(t *testing.T) {
		svc := mock.NewMockService(gomock.NewController(t))
		svc.EXPECT().GetSubDistrict(gomock.Any(), int64(3)).Return(&subdistrict.SubDistrict{
			ID:           3,
			Name:         "Menteng",
			Slug:         "menteng",
			FloodHistory: new("Banjir 2020"),
		}, nil)

		rec := serve(t, newRouter(svc), jsonRequest(http.MethodGet, "/subdistricts/3", ""))
		assertData(t, rec, http.StatusOK, `{
			"id": 3,
			"name": "Menteng",
			"slug": "menteng",
			"flood_history": "Banjir 2020",
			"safety_crime_information": null,
			"estimated_daily_spending": null,
			"medias": [],
			"static_map_image_url": null
		}`)
	})
}

func TestHandler_CreateSubDistrict(t *testing.T) {
	t.Run("rejects malformed bodies", func(t *testing.T) {
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
	})

	t.Run("maps service errors", func(t *testing.T) {
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
	})

	t.Run("success", func(t *testing.T) {
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
			FloodHistory:           new("Banjir 2020"),
			EstimatedDailySpending: new(1500000.5),
		}
		assert.Equal(t, want, *got, "a client-sent id or unknown key reached the service")

		assertData(t, rec, http.StatusCreated, `{
			"id": 42,
			"name": "Menteng",
			"slug": "menteng",
			"flood_history": "Banjir 2020",
			"safety_crime_information": null,
			"estimated_daily_spending": 1500000.5,
			"medias": [],
			"static_map_image_url": null
		}`)
	})

	t.Run("context reaches service", func(t *testing.T) {
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
	})
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
			method: http.MethodPut,
			target: "/subdistricts/1",
			body:   `{"name":"Menteng Atas","slug":"menteng"}`,
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

func TestHandler_PutSubDistrict(t *testing.T) {
	t.Run("rejects invalid requests", func(t *testing.T) {
		const valid = `{"name":"Kemang","slug":"kemang"}`

		tests := []struct {
			name     string
			target   string
			body     string
			wantCode string
		}{
			{name: "id is not a number", target: "/subdistricts/abc", body: valid, wantCode: "INVALID_SUB_DISTRICT_ID"},
			{name: "id is fractional", target: "/subdistricts/1.5", body: valid, wantCode: "INVALID_SUB_DISTRICT_ID"},
			{name: "id is zero", target: "/subdistricts/0", body: valid, wantCode: "INVALID_SUB_DISTRICT_ID"},
			{name: "id is negative", target: "/subdistricts/-1", body: valid, wantCode: "INVALID_SUB_DISTRICT_ID"},
			{name: "empty body", target: "/subdistricts/1", body: ``, wantCode: "INVALID_BODY"},
			{name: "truncated JSON", target: "/subdistricts/1", body: `{"name":"Kemang"`, wantCode: "INVALID_BODY"},
			{name: "body is an array", target: "/subdistricts/1", body: `[]`, wantCode: "INVALID_BODY"},
			{name: "body is null", target: "/subdistricts/1", body: `null`, wantCode: "INVALID_BODY"},
			{name: "name is a number", target: "/subdistricts/1", body: `{"name":5,"slug":"kemang"}`, wantCode: "INVALID_BODY"},
			{name: "estimated daily spending is a string", target: "/subdistricts/1",
				body: `{"name":"Kemang","slug":"kemang","estimated_daily_spending":"1.5"}`, wantCode: "INVALID_BODY"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// No expectations: an invalid request must never reach the service.
				svc := mock.NewMockService(gomock.NewController(t))
				rec := serve(t, newRouter(svc), jsonRequest(http.MethodPut, tc.target, tc.body))
				assertError(t, rec, http.StatusBadRequest, tc.wantCode)
			})
		}
	})

	t.Run("passes sub district to service", func(t *testing.T) {
		tests := []struct {
			name string
			body string
			want *subdistrict.SubDistrict
		}{
			{name: "every field is passed as sent",
				body: `{"name":"  Menteng Atas  ","slug":"menteng-atas","flood_history":"Banjir 2024",
					"safety_crime_information":"Aman","estimated_daily_spending":750000.25}`,
				want: &subdistrict.SubDistrict{
					Name:                   "  Menteng Atas  ",
					Slug:                   "menteng-atas",
					FloodHistory:           new("Banjir 2024"),
					SafetyCrimeInformation: new("Aman"),
					EstimatedDailySpending: new(750000.25),
				}},
			{name: "null and left-out keys are passed as absent",
				body: `{"name":"Menteng","slug":"menteng","flood_history":null}`,
				want: &subdistrict.SubDistrict{Name: "Menteng", Slug: "menteng"}},
			{name: "map image URL is ignored",
				body: `{"name":"Menteng","slug":"menteng","static_map_image_url":"https://evil.test/x.png"}`,
				want: &subdistrict.SubDistrict{Name: "Menteng", Slug: "menteng"}},
			{name: "client id and unknown keys are ignored",
				body: `{"id":99,"created_at":"yesterday","name":"Menteng","slug":"menteng"}`,
				want: &subdistrict.SubDistrict{Name: "Menteng", Slug: "menteng"}},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				svc := mock.NewMockService(gomock.NewController(t))
				svc.EXPECT().Update(gomock.Any(), int64(9), tc.want).Return(&subdistrict.SubDistrict{ID: 9}, nil)

				rec := serve(t, newRouter(svc), jsonRequest(http.MethodPut, "/subdistricts/9", tc.body))
				require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body)
			})
		}
	})

	t.Run("maps service errors", func(t *testing.T) {
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

				rec := serve(t, newRouter(svc), jsonRequest(http.MethodPut, "/subdistricts/1", `{"name":"Menteng","slug":"menteng"}`))
				assertError(t, rec, tc.wantStatus, tc.wantCode)
			})
		}
	})

	t.Run("success", func(t *testing.T) {
		svc := mock.NewMockService(gomock.NewController(t))
		svc.EXPECT().Update(gomock.Any(), int64(3), &subdistrict.SubDistrict{Name: "Menteng Atas", Slug: "menteng"}).Return(&subdistrict.SubDistrict{
			ID:                3,
			Name:              "Menteng Atas",
			Slug:              "menteng",
			StaticMapImageKey: new("subdistricts/3/static-map/a.png"),
		}, nil)

		rec := serve(t, newRouter(svc), jsonRequest(http.MethodPut, "/subdistricts/3", `{"name":"Menteng Atas","slug":"menteng"}`))
		assertData(t, rec, http.StatusOK, `{
			"id": 3,
			"name": "Menteng Atas",
			"slug": "menteng",
			"flood_history": null,
			"safety_crime_information": null,
			"estimated_daily_spending": null,
			"medias": [],
			"static_map_image_url": "https://img.test/subdistricts/3/static-map/a.png"
		}`)
	})
}

func TestHandler_DeleteSubDistrict(t *testing.T) {
	t.Run("rejects invalid id", func(t *testing.T) {
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
	})

	t.Run("maps service errors", func(t *testing.T) {
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
	})

	t.Run("success", func(t *testing.T) {
		svc := mock.NewMockService(gomock.NewController(t))
		svc.EXPECT().Delete(gomock.Any(), int64(4)).Return(nil)

		rec := serve(t, newRouter(svc), jsonRequest(http.MethodDelete, "/subdistricts/4", ""))
		require.Equal(t, http.StatusNoContent, rec.Code, "body: %s", rec.Body)
		assert.Empty(t, rec.Body.String(), "body should be empty")
	})
}

// -----------------------------------------------------------------------------
// Sub-district images
// -----------------------------------------------------------------------------

// handlerSlots are the image slots with their path segment.
var handlerSlots = []struct {
	slot subdistrict.ImageSlot
	path string
}{
	{slot: subdistrict.SlotMedias, path: "medias"},
	{slot: subdistrict.SlotStaticMap, path: "static-map"},
}

func TestHandler_CreateImageUpload(t *testing.T) {
	t.Run("rejects invalid requests", func(t *testing.T) {
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
	})

	t.Run("maps service errors", func(t *testing.T) {
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
				svc.EXPECT().ImageUploadURL(gomock.Any(), int64(3), subdistrict.SlotMedias, "image/gif").Return(storage.Upload{}, tc.err)

				rec := serve(t, newRouter(svc), jsonRequest(http.MethodPost, "/subdistricts/3/medias/upload-url", `{"content_type":"image/gif"}`))
				assertError(t, rec, tc.wantStatus, tc.wantCode)
			})
		}
	})

	t.Run("success", func(t *testing.T) {
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
	})
}

// storedImage is a sub-district with a media and a static map, and its JSON.
var storedImage = &subdistrict.SubDistrict{
	ID:   3,
	Name: "Menteng",
	Slug: "menteng",
	Medias: []storage.Media{
		{ID: 9, Key: "subdistricts/3/medias/a.png", ContentType: "image/png", Size: 1024, Position: 1},
	},
	StaticMapImageKey: new("subdistricts/3/static-map/a.png"),
}

const storedImageJSON = `{
	"id": 3,
	"name": "Menteng",
	"slug": "menteng",
	"flood_history": null,
	"safety_crime_information": null,
	"estimated_daily_spending": null,
	"medias": [{"id": 9, "url": "https://img.test/subdistricts/3/medias/a.png", "content_type": "image/png", "size": 1024, "position": 1}],
	"static_map_image_url": "https://img.test/subdistricts/3/static-map/a.png"
}`

// uploadErrors are the service errors from storing an upload, with their responses.
var uploadErrors = []struct {
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

// invalidUploadRequests are requests to store an upload that must never reach
// the service.
var invalidUploadRequests = []struct {
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

func TestHandler_AddMedia(t *testing.T) {
	t.Run("rejects invalid requests", func(t *testing.T) {
		for _, tc := range invalidUploadRequests {
			t.Run(tc.name, func(t *testing.T) {
				svc := mock.NewMockService(gomock.NewController(t))
				rec := serve(t, newRouter(svc), jsonRequest(http.MethodPost, "/subdistricts/"+tc.id+"/medias", tc.body))
				assertError(t, rec, http.StatusBadRequest, tc.wantCode)
			})
		}
	})

	t.Run("maps the media limit", func(t *testing.T) {
		svc := mock.NewMockService(gomock.NewController(t))
		svc.EXPECT().AddMedia(gomock.Any(), int64(3), "uploads/subdistricts/3/medias/a.png").Return(nil, storage.ErrMediaLimitReached)

		rec := serve(t, newRouter(svc), jsonRequest(http.MethodPost, "/subdistricts/3/medias",
			`{"upload_key":"uploads/subdistricts/3/medias/a.png"}`))
		assertError(t, rec, http.StatusConflict, "MEDIA_LIMIT_REACHED")
	})

	t.Run("maps service errors", func(t *testing.T) {
		for _, tc := range uploadErrors {
			t.Run(tc.name, func(t *testing.T) {
				svc := mock.NewMockService(gomock.NewController(t))
				svc.EXPECT().AddMedia(gomock.Any(), int64(3), "uploads/subdistricts/3/medias/a.png").Return(nil, tc.err)

				rec := serve(t, newRouter(svc), jsonRequest(http.MethodPost, "/subdistricts/3/medias",
					`{"upload_key":"uploads/subdistricts/3/medias/a.png"}`))
				assertError(t, rec, tc.wantStatus, tc.wantCode)
			})
		}
	})

	t.Run("success", func(t *testing.T) {
		svc := mock.NewMockService(gomock.NewController(t))
		svc.EXPECT().AddMedia(gomock.Any(), int64(3), "uploads/subdistricts/3/medias/a.png").Return(storedImage, nil)

		rec := serve(t, newRouter(svc), jsonRequest(http.MethodPost, "/subdistricts/3/medias",
			`{"upload_key":"uploads/subdistricts/3/medias/a.png"}`))
		assertData(t, rec, http.StatusOK, storedImageJSON)
	})
}

func TestHandler_DeleteMedia(t *testing.T) {
	t.Run("rejects invalid ids", func(t *testing.T) {
		tests := []struct {
			name     string
			target   string
			wantCode string
		}{
			{name: "sub-district id is not a number", target: "/subdistricts/abc/medias/5", wantCode: "INVALID_SUB_DISTRICT_ID"},
			{name: "sub-district id is zero", target: "/subdistricts/0/medias/5", wantCode: "INVALID_SUB_DISTRICT_ID"},
			{name: "media id is not a number", target: "/subdistricts/3/medias/abc", wantCode: "INVALID_MEDIA_ID"},
			{name: "media id is fractional", target: "/subdistricts/3/medias/1.5", wantCode: "INVALID_MEDIA_ID"},
			{name: "media id is zero", target: "/subdistricts/3/medias/0", wantCode: "INVALID_MEDIA_ID"},
			{name: "media id is negative", target: "/subdistricts/3/medias/-1", wantCode: "INVALID_MEDIA_ID"},
			{name: "media id overflows 64 bits", target: "/subdistricts/3/medias/99999999999999999999", wantCode: "INVALID_MEDIA_ID"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// No expectations: an invalid id must never reach the service.
				svc := mock.NewMockService(gomock.NewController(t))
				rec := serve(t, newRouter(svc), jsonRequest(http.MethodDelete, tc.target, ""))
				assertError(t, rec, http.StatusBadRequest, tc.wantCode)
			})
		}
	})

	t.Run("maps service errors", func(t *testing.T) {
		tests := []struct {
			name       string
			err        error
			wantStatus int
			wantCode   string
		}{
			{name: "unknown media", err: storage.ErrMediaNotFound, wantStatus: http.StatusNotFound, wantCode: "MEDIA_NOT_FOUND"},
			{name: "storage failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				svc := mock.NewMockService(gomock.NewController(t))
				svc.EXPECT().DeleteMedia(gomock.Any(), int64(3), int64(5)).Return(tc.err)

				rec := serve(t, newRouter(svc), jsonRequest(http.MethodDelete, "/subdistricts/3/medias/5", ""))
				assertError(t, rec, tc.wantStatus, tc.wantCode)
			})
		}
	})

	t.Run("success", func(t *testing.T) {
		svc := mock.NewMockService(gomock.NewController(t))
		svc.EXPECT().DeleteMedia(gomock.Any(), int64(3), int64(5)).Return(nil)

		rec := serve(t, newRouter(svc), jsonRequest(http.MethodDelete, "/subdistricts/3/medias/5", ""))
		require.Equal(t, http.StatusNoContent, rec.Code, "body: %s", rec.Body)
		assert.Empty(t, rec.Body.String(), "body should be empty")
	})
}

func TestHandler_SetStaticMap(t *testing.T) {
	t.Run("rejects invalid requests", func(t *testing.T) {
		for _, tc := range invalidUploadRequests {
			t.Run(tc.name, func(t *testing.T) {
				svc := mock.NewMockService(gomock.NewController(t))
				rec := serve(t, newRouter(svc), jsonRequest(http.MethodPut, "/subdistricts/"+tc.id+"/static-map", tc.body))
				assertError(t, rec, http.StatusBadRequest, tc.wantCode)
			})
		}
	})

	t.Run("maps service errors", func(t *testing.T) {
		for _, tc := range uploadErrors {
			t.Run(tc.name, func(t *testing.T) {
				svc := mock.NewMockService(gomock.NewController(t))
				svc.EXPECT().SetStaticMap(gomock.Any(), int64(3), "uploads/subdistricts/3/static-map/a.png").Return(nil, tc.err)

				rec := serve(t, newRouter(svc), jsonRequest(http.MethodPut, "/subdistricts/3/static-map",
					`{"upload_key":"uploads/subdistricts/3/static-map/a.png"}`))
				assertError(t, rec, tc.wantStatus, tc.wantCode)
			})
		}
	})

	t.Run("success", func(t *testing.T) {
		svc := mock.NewMockService(gomock.NewController(t))
		svc.EXPECT().SetStaticMap(gomock.Any(), int64(3), "uploads/subdistricts/3/static-map/a.png").Return(storedImage, nil)

		rec := serve(t, newRouter(svc), jsonRequest(http.MethodPut, "/subdistricts/3/static-map",
			`{"upload_key":"uploads/subdistricts/3/static-map/a.png"}`))
		assertData(t, rec, http.StatusOK, storedImageJSON)
	})
}
