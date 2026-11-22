package subdistricts_test

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

	"rantau-go/internal/domain"
	"rantau-go/internal/module/subdistricts"
	"rantau-go/internal/module/subdistricts/mock"
	"rantau-go/internal/storage"
)

// newRouter serves the sub-district handlers on the paths the API uses for
// them, so URL parameters resolve as they do in production. It returns the
// router and the mock of the service behind it.
func newRouter(t *testing.T) (http.Handler, *mock.MocksubDistrictService) {
	t.Helper()

	svc := mock.NewMocksubDistrictService(gomock.NewController(t))
	h := subdistricts.NewHandler(svc, testMediaURL)
	r := chi.NewRouter()
	r.Route("/subdistricts", func(r chi.Router) {
		r.Get("/", h.List)
		r.Get("/{id}", h.Show)
		r.Post("/", h.Create)
		r.Put("/{id}", h.Replace)
		r.Delete("/{id}", h.Delete)
		r.Post("/{id}/medias/upload-url", h.CreateMediaUpload)
		r.Post("/{id}/medias", h.AddMedia)
		r.Delete("/{id}/medias/{mediaId}", h.DeleteMedia)
		r.Put("/{id}/boundary", h.SetBoundary)
	})
	return r, svc
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

// errorResult is what a client sees of an error response. The message is only
// checked for being present.
type errorResult struct {
	Status     int
	JSON       bool
	Code       string
	HasMessage bool
}

// wantError is the errorResult of a well-formed error response.
func wantError(status int, code string) errorResult {
	return errorResult{Status: status, JSON: true, Code: code, HasMessage: true}
}

// readError reads the status and the { code, message } error envelope of rec.
func readError(t *testing.T, rec *httptest.ResponseRecorder) errorResult {
	t.Helper()

	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), "error body is not JSON; status %d, body: %s", rec.Code, rec.Body)

	return errorResult{
		Status:     rec.Code,
		JSON:       strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json"),
		Code:       body.Code,
		HasMessage: body.Message != "",
	}
}

// dataResult is what a client sees of a success response.
type dataResult struct {
	Status int
	Data   any
}

// readData reads the status and the success envelope's "data" value of rec.
func readData(t *testing.T, rec *httptest.ResponseRecorder) dataResult {
	t.Helper()

	var env struct {
		Data any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env), "response is not JSON; status %d, body: %s", rec.Code, rec.Body)

	return dataResult{Status: rec.Code, Data: env.Data}
}

// wantData is the dataResult of a response whose data is the JSON document doc.
func wantData(t *testing.T, status int, doc string) dataResult {
	t.Helper()

	var data any
	require.NoError(t, json.Unmarshal([]byte(doc), &data), "want data %s is not JSON", doc)

	return dataResult{Status: status, Data: data}
}

func TestHandler_List(t *testing.T) {
	t.Run("response-shape", func(t *testing.T) {
		tests := []struct {
			name         string
			subDistricts []domain.SubDistrict
			want         string
		}{
			{name: "none", subDistricts: []domain.SubDistrict{}, want: `[]`},
			{
				name: "every-field",
				subDistricts: []domain.SubDistrict{
					{ID: 1, Name: "Menteng", Slug: "menteng", FloodHistory: new("Banjir 2020"),
						EstimatedDailySpending: new(1500000.5),
						Medias: []domain.Media{
							{ID: 4, Key: "subdistricts/1/medias/a.webp", ContentType: "image/webp", Size: 2048, Position: 1},
						},
						Boundary: mustBoundary(t, squareJSON)},
					{ID: 2, Name: "Kemang", Slug: "kemang"},
				},
				want: `[
					{"id": 1, "name": "Menteng", "slug": "menteng", "flood_history": "Banjir 2020",
					 "safety_crime_information": null, "estimated_daily_spending": 1500000.5,
					 "medias": [{"id": 4, "url": "https://img.test/subdistricts/1/medias/a.webp",
					             "content_type": "image/webp", "size": 2048, "position": 1}],
					 "boundary": ` + squareJSON + `},
					{"id": 2, "name": "Kemang", "slug": "kemang", "flood_history": null,
					 "safety_crime_information": null, "estimated_daily_spending": null,
					 "medias": [], "boundary": null}
				]`,
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				router, svc := newRouter(t)
				svc.EXPECT().List(gomock.Any()).Return(tc.subDistricts, nil)

				rec := serve(t, router, jsonRequest(http.MethodGet, "/subdistricts", ""))
				assert.Equal(t, wantData(t, http.StatusOK, tc.want), readData(t, rec), "GET /subdistricts response")
			})
		}
	})
}

func TestHandler_Show(t *testing.T) {
	t.Run("invalid-id", func(t *testing.T) {
		tests := []struct {
			name   string
			target string
		}{
			{name: "not-a-number", target: "/subdistricts/abc"},
			{name: "fractional", target: "/subdistricts/1.5"},
			{name: "zero", target: "/subdistricts/0"},
			{name: "negative", target: "/subdistricts/-1"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// No expectations: an invalid id must never reach the service.
				router, _ := newRouter(t)
				rec := serve(t, router, jsonRequest(http.MethodGet, tc.target, ""))
				assert.Equal(t, wantError(http.StatusBadRequest, "INVALID_SUB_DISTRICT_ID"), readError(t, rec), "GET %s response", tc.target)
			})
		}
	})

	t.Run("service-errors", func(t *testing.T) {
		tests := []struct {
			name       string
			err        error
			wantStatus int
			wantCode   string
		}{
			{name: "unknown", err: domain.ErrSubDistrictNotFound, wantStatus: http.StatusNotFound, wantCode: "SUB_DISTRICT_NOT_FOUND"},
			{name: "wrapped-unknown", err: fmt.Errorf("get: %w", domain.ErrSubDistrictNotFound), wantStatus: http.StatusNotFound, wantCode: "SUB_DISTRICT_NOT_FOUND"},
			{name: "storage-failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				router, svc := newRouter(t)
				svc.EXPECT().Fetch(gomock.Any(), int64(5)).Return(nil, tc.err)

				rec := serve(t, router, jsonRequest(http.MethodGet, "/subdistricts/5", ""))
				assert.Equal(t, wantError(tc.wantStatus, tc.wantCode), readError(t, rec), "GET /subdistricts/5 response")
			})
		}
	})

	t.Run("success", func(t *testing.T) {
		router, svc := newRouter(t)
		svc.EXPECT().Fetch(gomock.Any(), int64(3)).
			Return(&domain.SubDistrict{ID: 3, Name: "Menteng", Slug: "menteng", FloodHistory: new("Banjir 2020")}, nil)

		rec := serve(t, router, jsonRequest(http.MethodGet, "/subdistricts/3", ""))
		want := wantData(t, http.StatusOK, `{
			"id": 3,
			"name": "Menteng",
			"slug": "menteng",
			"flood_history": "Banjir 2020",
			"safety_crime_information": null,
			"estimated_daily_spending": null,
			"medias": [],
			"boundary": null
		}`)
		assert.Equal(t, want, readData(t, rec), "GET /subdistricts/3 response")
	})
}

func TestHandler_Create(t *testing.T) {
	t.Run("malformed-body", func(t *testing.T) {
		tests := []struct {
			name string
			body string
		}{
			{name: "empty", body: ``},
			{name: "null", body: `null`},
			{name: "truncated", body: `{"name":"Menteng"`},
			{name: "name-number", body: `{"name":5,"slug":"menteng"}`},
			{name: "spending-string", body: `{"name":"Menteng","slug":"menteng","estimated_daily_spending":"1.5"}`},
			{name: "array", body: `[]`},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// No expectations: a malformed body must never reach the service.
				router, _ := newRouter(t)
				rec := serve(t, router, jsonRequest(http.MethodPost, "/subdistricts", tc.body))
				assert.Equal(t, wantError(http.StatusBadRequest, "INVALID_BODY"), readError(t, rec), "POST /subdistricts %q response", tc.body)
			})
		}
	})

	t.Run("service-errors", func(t *testing.T) {
		tests := []struct {
			name       string
			err        error
			wantStatus int
			wantCode   string
		}{
			{name: "invalid-name", err: domain.ErrSubDistrictNameInvalid, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "invalid-slug", err: domain.ErrSubDistrictSlugInvalid, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "invalid-spending", err: domain.ErrSubDistrictSpendingInvalid, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "duplicate", err: domain.ErrSubDistrictExists, wantStatus: http.StatusConflict, wantCode: "SUB_DISTRICT_EXISTS"},
			{name: "wrapped-duplicate", err: fmt.Errorf("create: %w", domain.ErrSubDistrictExists), wantStatus: http.StatusConflict, wantCode: "SUB_DISTRICT_EXISTS"},
			{name: "storage-failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				router, svc := newRouter(t)
				svc.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, tc.err)

				rec := serve(t, router, jsonRequest(http.MethodPost, "/subdistricts", `{"name":"Menteng","slug":"menteng"}`))
				assert.Equal(t, wantError(tc.wantStatus, tc.wantCode), readError(t, rec), "POST /subdistricts response")
			})
		}
	})

	t.Run("success", func(t *testing.T) {
		// A client-sent id or unknown key must be ignored.
		want := &domain.SubDistrict{
			Name:                   "Menteng",
			Slug:                   "menteng",
			FloodHistory:           new("Banjir 2020"),
			EstimatedDailySpending: new(1500000.5),
		}
		router, svc := newRouter(t)
		svc.EXPECT().Create(gomock.Any(), want).Return(&domain.SubDistrict{
			ID:                     42,
			Name:                   want.Name,
			Slug:                   want.Slug,
			FloodHistory:           want.FloodHistory,
			SafetyCrimeInformation: want.SafetyCrimeInformation,
			EstimatedDailySpending: want.EstimatedDailySpending,
		}, nil)

		body := `{"name":"Menteng","slug":"menteng","flood_history":"Banjir 2020","safety_crime_information":null,` +
			`"estimated_daily_spending":1500000.5,"boundary":` + squareJSON + `,` +
			`"id":77,"created_at":"yesterday"}`
		rec := serve(t, router, jsonRequest(http.MethodPost, "/subdistricts", body))

		wantBody := wantData(t, http.StatusCreated, `{
			"id": 42,
			"name": "Menteng",
			"slug": "menteng",
			"flood_history": "Banjir 2020",
			"safety_crime_information": null,
			"estimated_daily_spending": 1500000.5,
			"medias": [],
			"boundary": null
		}`)
		assert.Equal(t, wantBody, readData(t, rec), "POST /subdistricts response")
	})

	t.Run("context-reaches-service", func(t *testing.T) {
		tests := []struct {
			name   string
			newCtx func(t *testing.T) (context.Context, context.CancelFunc)
		}{
			{
				name: "client-disconnected",
				newCtx: func(t *testing.T) (context.Context, context.CancelFunc) {
					ctx, cancel := context.WithCancel(t.Context())
					cancel()
					return ctx, cancel
				},
			},
			{
				name: "deadline-passes",
				newCtx: func(t *testing.T) (context.Context, context.CancelFunc) {
					return context.WithTimeout(t.Context(), 20*time.Millisecond)
				},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				ctx, cancel := tc.newCtx(t)
				defer cancel()

				router, svc := newRouter(t)
				svc.EXPECT().Create(gomock.Any(), gomock.Any()).
					DoAndReturn(func(got context.Context, _ *domain.SubDistrict) (*domain.SubDistrict, error) {
						select {
						case <-got.Done():
							return nil, got.Err()
						case <-time.After(2 * time.Second):
							assert.Fail(t, "service was not given the request's context")
							return nil, errors.New("timed out")
						}
					})

				req := jsonRequest(http.MethodPost, "/subdistricts", `{"name":"Menteng","slug":"menteng"}`).WithContext(ctx)
				rec := serve(t, router, req)
				assert.NotEqual(t, http.StatusCreated, rec.Code, "POST /subdistricts status for a request whose context had ended")
			})
		}
	})
}

func TestHandler_InternalErrorsDoNotLeakDetails(t *testing.T) {
	dbErr := errors.New(`pq: relation "sub_districts" does not exist`)

	tests := []struct {
		name   string
		method string
		target string
		body   string
		fail   func(svc *mock.MocksubDistrictService)
	}{
		{
			name:   "list",
			method: http.MethodGet,
			target: "/subdistricts",
			body:   "",
			fail: func(s *mock.MocksubDistrictService) {
				s.EXPECT().List(gomock.Any()).Return(nil, dbErr)
			},
		},
		{
			name:   "show",
			method: http.MethodGet,
			target: "/subdistricts/1",
			body:   "",
			fail: func(s *mock.MocksubDistrictService) {
				s.EXPECT().Fetch(gomock.Any(), int64(1)).Return(nil, dbErr)
			},
		},
		{
			name:   "create",
			method: http.MethodPost,
			target: "/subdistricts",
			body:   `{"name":"Menteng","slug":"menteng"}`,
			fail: func(s *mock.MocksubDistrictService) {
				s.EXPECT().Create(gomock.Any(), &domain.SubDistrict{Name: "Menteng", Slug: "menteng"}).Return(nil, dbErr)
			},
		},
		{
			name:   "update",
			method: http.MethodPut,
			target: "/subdistricts/1",
			body:   `{"name":"Menteng Atas","slug":"menteng"}`,
			fail: func(s *mock.MocksubDistrictService) {
				s.EXPECT().Update(gomock.Any(), int64(1), &domain.SubDistrict{Name: "Menteng Atas", Slug: "menteng"}).Return(nil, dbErr)
			},
		},
		{
			name:   "set-boundary",
			method: http.MethodPut,
			target: "/subdistricts/1/boundary",
			body:   squareJSON,
			fail: func(s *mock.MocksubDistrictService) {
				s.EXPECT().SetBoundary(gomock.Any(), int64(1), mustBoundary(t, squareJSON)).Return(nil, dbErr)
			},
		},
		{
			name:   "delete",
			method: http.MethodDelete,
			target: "/subdistricts/1",
			body:   "",
			fail: func(s *mock.MocksubDistrictService) {
				s.EXPECT().Delete(gomock.Any(), int64(1)).Return(dbErr)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router, svc := newRouter(t)
			tc.fail(svc)

			rec := serve(t, router, jsonRequest(tc.method, tc.target, tc.body))
			assert.Equal(t, wantError(http.StatusInternalServerError, "INTERNAL_ERROR"), readError(t, rec), "%s %s response", tc.method, tc.target)
			assert.NotContains(t, rec.Body.String(), "pq:", "%s %s 500 body exposes storage detail", tc.method, tc.target)
			assert.NotContains(t, rec.Body.String(), "relation", "%s %s 500 body exposes storage detail", tc.method, tc.target)
		})
	}
}

func TestHandler_Replace(t *testing.T) {
	t.Run("invalid-request", func(t *testing.T) {
		const valid = `{"name":"Kemang","slug":"kemang"}`

		tests := []struct {
			name     string
			target   string
			body     string
			wantCode string
		}{
			{name: "id-not-a-number", target: "/subdistricts/abc", body: valid, wantCode: "INVALID_SUB_DISTRICT_ID"},
			{name: "id-fractional", target: "/subdistricts/1.5", body: valid, wantCode: "INVALID_SUB_DISTRICT_ID"},
			{name: "id-zero", target: "/subdistricts/0", body: valid, wantCode: "INVALID_SUB_DISTRICT_ID"},
			{name: "id-negative", target: "/subdistricts/-1", body: valid, wantCode: "INVALID_SUB_DISTRICT_ID"},
			{name: "empty-body", target: "/subdistricts/1", body: ``, wantCode: "INVALID_BODY"},
			{name: "truncated", target: "/subdistricts/1", body: `{"name":"Kemang"`, wantCode: "INVALID_BODY"},
			{name: "array", target: "/subdistricts/1", body: `[]`, wantCode: "INVALID_BODY"},
			{name: "null", target: "/subdistricts/1", body: `null`, wantCode: "INVALID_BODY"},
			{name: "name-number", target: "/subdistricts/1", body: `{"name":5,"slug":"kemang"}`, wantCode: "INVALID_BODY"},
			{name: "spending-string", target: "/subdistricts/1",
				body: `{"name":"Kemang","slug":"kemang","estimated_daily_spending":"1.5"}`, wantCode: "INVALID_BODY"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// No expectations: an invalid request must never reach the service.
				router, _ := newRouter(t)
				rec := serve(t, router, jsonRequest(http.MethodPut, tc.target, tc.body))
				assert.Equal(t, wantError(http.StatusBadRequest, tc.wantCode), readError(t, rec), "PUT %s %q response", tc.target, tc.body)
			})
		}
	})

	t.Run("passes-sub-district", func(t *testing.T) {
		tests := []struct {
			name string
			body string
			want *domain.SubDistrict
		}{
			{name: "every-field",
				body: `{"name":"  Menteng Atas  ","slug":"menteng-atas","flood_history":"Banjir 2024",
					"safety_crime_information":"Aman","estimated_daily_spending":750000.25}`,
				want: &domain.SubDistrict{
					Name:                   "  Menteng Atas  ",
					Slug:                   "menteng-atas",
					FloodHistory:           new("Banjir 2024"),
					SafetyCrimeInformation: new("Aman"),
					EstimatedDailySpending: new(750000.25),
				}},
			{name: "null-and-left-out",
				body: `{"name":"Menteng","slug":"menteng","flood_history":null}`,
				want: &domain.SubDistrict{Name: "Menteng", Slug: "menteng"}},
			{name: "boundary-ignored",
				body: `{"name":"Menteng","slug":"menteng","boundary":` + squareJSON + `}`,
				want: &domain.SubDistrict{Name: "Menteng", Slug: "menteng"}},
			{name: "id-and-unknown-ignored",
				body: `{"id":99,"created_at":"yesterday","name":"Menteng","slug":"menteng"}`,
				want: &domain.SubDistrict{Name: "Menteng", Slug: "menteng"}},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				router, svc := newRouter(t)
				svc.EXPECT().Update(gomock.Any(), int64(9), tc.want).Return(&domain.SubDistrict{ID: 9}, nil)

				rec := serve(t, router, jsonRequest(http.MethodPut, "/subdistricts/9", tc.body))
				assert.Equal(t, http.StatusOK, rec.Code, "PUT /subdistricts/9 status; body: %s", rec.Body)
			})
		}
	})

	t.Run("service-errors", func(t *testing.T) {
		tests := []struct {
			name       string
			err        error
			wantStatus int
			wantCode   string
		}{
			{name: "unknown", err: domain.ErrSubDistrictNotFound, wantStatus: http.StatusNotFound, wantCode: "SUB_DISTRICT_NOT_FOUND"},
			{name: "wrapped-unknown", err: fmt.Errorf("update: %w", domain.ErrSubDistrictNotFound), wantStatus: http.StatusNotFound, wantCode: "SUB_DISTRICT_NOT_FOUND"},
			{name: "invalid-name", err: domain.ErrSubDistrictNameInvalid, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "invalid-slug", err: domain.ErrSubDistrictSlugInvalid, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "invalid-spending", err: domain.ErrSubDistrictSpendingInvalid, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "duplicate", err: domain.ErrSubDistrictExists, wantStatus: http.StatusConflict, wantCode: "SUB_DISTRICT_EXISTS"},
			{name: "wrapped-duplicate", err: fmt.Errorf("update: %w", domain.ErrSubDistrictExists), wantStatus: http.StatusConflict, wantCode: "SUB_DISTRICT_EXISTS"},
			{name: "storage-failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				router, svc := newRouter(t)
				svc.EXPECT().Update(gomock.Any(), int64(1), gomock.Any()).Return(nil, tc.err)

				rec := serve(t, router, jsonRequest(http.MethodPut, "/subdistricts/1", `{"name":"Menteng","slug":"menteng"}`))
				assert.Equal(t, wantError(tc.wantStatus, tc.wantCode), readError(t, rec), "PUT /subdistricts/1 response")
			})
		}
	})

	t.Run("success", func(t *testing.T) {
		router, svc := newRouter(t)
		svc.EXPECT().Update(gomock.Any(), int64(3), &domain.SubDistrict{Name: "Menteng Atas", Slug: "menteng"}).
			Return(&domain.SubDistrict{ID: 3, Name: "Menteng Atas", Slug: "menteng", Boundary: mustBoundary(t, squareJSON)}, nil)

		rec := serve(t, router, jsonRequest(http.MethodPut, "/subdistricts/3", `{"name":"Menteng Atas","slug":"menteng"}`))
		want := wantData(t, http.StatusOK, `{
			"id": 3,
			"name": "Menteng Atas",
			"slug": "menteng",
			"flood_history": null,
			"safety_crime_information": null,
			"estimated_daily_spending": null,
			"medias": [],
			"boundary": `+squareJSON+`
		}`)
		assert.Equal(t, want, readData(t, rec), "PUT /subdistricts/3 response")
	})
}

func TestHandler_Delete(t *testing.T) {
	t.Run("invalid-id", func(t *testing.T) {
		tests := []struct {
			name   string
			target string
		}{
			{name: "not-a-number", target: "/subdistricts/abc"},
			{name: "fractional", target: "/subdistricts/1.5"},
			{name: "zero", target: "/subdistricts/0"},
			{name: "negative", target: "/subdistricts/-1"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// No expectations: an invalid id must never reach the service.
				router, _ := newRouter(t)
				rec := serve(t, router, jsonRequest(http.MethodDelete, tc.target, ""))
				assert.Equal(t, wantError(http.StatusBadRequest, "INVALID_SUB_DISTRICT_ID"), readError(t, rec), "DELETE %s response", tc.target)
			})
		}
	})

	t.Run("service-errors", func(t *testing.T) {
		tests := []struct {
			name       string
			err        error
			wantStatus int
			wantCode   string
		}{
			{name: "unknown", err: domain.ErrSubDistrictNotFound, wantStatus: http.StatusNotFound, wantCode: "SUB_DISTRICT_NOT_FOUND"},
			{name: "wrapped-unknown", err: fmt.Errorf("delete: %w", domain.ErrSubDistrictNotFound), wantStatus: http.StatusNotFound, wantCode: "SUB_DISTRICT_NOT_FOUND"},
			{name: "in-use", err: domain.ErrSubDistrictInUse, wantStatus: http.StatusConflict, wantCode: "SUB_DISTRICT_IN_USE"},
			{name: "storage-failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				router, svc := newRouter(t)
				svc.EXPECT().Delete(gomock.Any(), int64(4)).Return(tc.err)

				rec := serve(t, router, jsonRequest(http.MethodDelete, "/subdistricts/4", ""))
				assert.Equal(t, wantError(tc.wantStatus, tc.wantCode), readError(t, rec), "DELETE /subdistricts/4 response")
			})
		}
	})

	t.Run("success", func(t *testing.T) {
		router, svc := newRouter(t)
		svc.EXPECT().Delete(gomock.Any(), int64(4)).Return(nil)

		rec := serve(t, router, jsonRequest(http.MethodDelete, "/subdistricts/4", ""))
		assert.Equal(t, http.StatusNoContent, rec.Code, "DELETE /subdistricts/4 status; body: %s", rec.Body)
		assert.Zero(t, rec.Body.Len(), "DELETE /subdistricts/4 body = %q, want empty", rec.Body)
	})
}

// handlerSlots are the image slots with their path segment.
var handlerSlots = []struct {
	slot subdistricts.MediaSlot
	path string
}{
	{slot: subdistricts.SlotMedias, path: "medias"},
}

func TestHandler_CreateMediaUpload(t *testing.T) {
	t.Run("invalid-request", func(t *testing.T) {
		tests := []struct {
			name     string
			id       string
			body     string
			wantCode string
		}{
			{name: "id-not-a-number", id: "abc", body: `{"content_type":"image/png"}`, wantCode: "INVALID_SUB_DISTRICT_ID"},
			{name: "id-zero", id: "0", body: `{"content_type":"image/png"}`, wantCode: "INVALID_SUB_DISTRICT_ID"},
			{name: "body-not-json", id: "3", body: `content_type=image/png`, wantCode: "INVALID_BODY"},
			{name: "body-null", id: "3", body: `null`, wantCode: "INVALID_BODY"},
			{name: "body-array", id: "3", body: `[]`, wantCode: "INVALID_BODY"},
		}

		for _, hs := range handlerSlots {
			for _, tc := range tests {
				t.Run(hs.path+"-"+tc.name, func(t *testing.T) {
					// No expectations: an invalid request must never reach the service.
					router, _ := newRouter(t)
					target := "/subdistricts/" + tc.id + "/" + hs.path + "/upload-url"
					rec := serve(t, router, jsonRequest(http.MethodPost, target, tc.body))
					assert.Equal(t, wantError(http.StatusBadRequest, tc.wantCode), readError(t, rec), "POST %s %q response", target, tc.body)
				})
			}
		}
	})

	t.Run("service-errors", func(t *testing.T) {
		tests := []struct {
			name       string
			err        error
			wantStatus int
			wantCode   string
		}{
			{name: "unknown", err: domain.ErrSubDistrictNotFound, wantStatus: http.StatusNotFound, wantCode: "SUB_DISTRICT_NOT_FOUND"},
			{name: "type-not-allowed", err: storage.ErrInvalidMediaType, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "storage-failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				router, svc := newRouter(t)
				svc.EXPECT().MediaUploadURL(gomock.Any(), int64(3), subdistricts.SlotMedias, "image/gif").Return(storage.Upload{}, tc.err)

				rec := serve(t, router, jsonRequest(http.MethodPost, "/subdistricts/3/medias/upload-url", `{"content_type":"image/gif"}`))
				assert.Equal(t, wantError(tc.wantStatus, tc.wantCode), readError(t, rec), "POST /subdistricts/3/medias/upload-url response")
			})
		}
	})

	t.Run("success", func(t *testing.T) {
		for _, hs := range handlerSlots {
			t.Run(hs.path, func(t *testing.T) {
				router, svc := newRouter(t)
				svc.EXPECT().MediaUploadURL(gomock.Any(), int64(3), hs.slot, "image/png").Return(storage.Upload{
					Key:       "uploads/subdistricts/3/" + hs.path + "/a.png",
					URL:       "http://localhost:3900/rantau-images/uploads/subdistricts/3/" + hs.path + "/a.png?X-Amz-Signature=abc",
					Method:    "PUT",
					Headers:   map[string]string{"Content-Type": "image/png"},
					ExpiresAt: time.Date(2026, 10, 31, 12, 5, 0, 0, time.UTC),
				}, nil)

				rec := serve(t, router, jsonRequest(http.MethodPost, "/subdistricts/3/"+hs.path+"/upload-url", `{"content_type":"image/png"}`))
				want := wantData(t, http.StatusOK, `{
					"upload_key": "uploads/subdistricts/3/`+hs.path+`/a.png",
					"upload_url": "http://localhost:3900/rantau-images/uploads/subdistricts/3/`+hs.path+`/a.png?X-Amz-Signature=abc",
					"method": "PUT",
					"headers": {"Content-Type": "image/png"},
					"expires_at": "2026-10-31T12:05:00Z"
				}`)
				assert.Equal(t, want, readData(t, rec), "POST /subdistricts/3/%s/upload-url response", hs.path)
			})
		}
	})
}

// storedImage is a sub-district with a media, and its JSON.
var storedImage = &domain.SubDistrict{
	ID:   3,
	Name: "Menteng",
	Slug: "menteng",
	Medias: []domain.Media{
		{ID: 9, Key: "subdistricts/3/medias/a.png", ContentType: "image/png", Size: 1024, Position: 1},
	},
}

const storedImageJSON = `{
	"id": 3,
	"name": "Menteng",
	"slug": "menteng",
	"flood_history": null,
	"safety_crime_information": null,
	"estimated_daily_spending": null,
	"medias": [{"id": 9, "url": "https://img.test/subdistricts/3/medias/a.png", "content_type": "image/png", "size": 1024, "position": 1}],
	"boundary": null
}`

// uploadErrors are the service errors from storing an upload, with their responses.
var uploadErrors = []struct {
	name       string
	err        error
	wantStatus int
	wantCode   string
}{
	{name: "unknown", err: domain.ErrSubDistrictNotFound, wantStatus: http.StatusNotFound, wantCode: "SUB_DISTRICT_NOT_FOUND"},
	{name: "invalid-key", err: storage.ErrInvalidUploadKey, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
	{name: "not-uploaded", err: storage.ErrMediaNotUploaded, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
	{name: "type-not-allowed", err: storage.ErrInvalidMediaType, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
	{name: "too-large", err: storage.ErrImageTooLarge, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
	{name: "storage-failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
}

// invalidUploadRequests are requests to store an upload that must never reach
// the service.
var invalidUploadRequests = []struct {
	name     string
	id       string
	body     string
	wantCode string
}{
	{name: "id-not-a-number", id: "abc", body: `{"upload_key":"k"}`, wantCode: "INVALID_SUB_DISTRICT_ID"},
	{name: "id-negative", id: "-1", body: `{"upload_key":"k"}`, wantCode: "INVALID_SUB_DISTRICT_ID"},
	{name: "body-not-json", id: "3", body: `upload_key=k`, wantCode: "INVALID_BODY"},
	{name: "body-null", id: "3", body: `null`, wantCode: "INVALID_BODY"},
	{name: "body-array", id: "3", body: `[]`, wantCode: "INVALID_BODY"},
}

func TestHandler_AddMedia(t *testing.T) {
	const body = `{"upload_key":"uploads/subdistricts/3/medias/a.png"}`
	const uploadKey = "uploads/subdistricts/3/medias/a.png"

	t.Run("invalid-request", func(t *testing.T) {
		for _, tc := range invalidUploadRequests {
			t.Run(tc.name, func(t *testing.T) {
				// No expectations: an invalid request must never reach the service.
				router, _ := newRouter(t)
				target := "/subdistricts/" + tc.id + "/medias"
				rec := serve(t, router, jsonRequest(http.MethodPost, target, tc.body))
				assert.Equal(t, wantError(http.StatusBadRequest, tc.wantCode), readError(t, rec), "POST %s %q response", target, tc.body)
			})
		}
	})

	t.Run("media-limit", func(t *testing.T) {
		router, svc := newRouter(t)
		svc.EXPECT().AddMedia(gomock.Any(), int64(3), uploadKey).Return(nil, domain.ErrMediaLimitReached)

		rec := serve(t, router, jsonRequest(http.MethodPost, "/subdistricts/3/medias", body))
		assert.Equal(t, wantError(http.StatusConflict, "MEDIA_LIMIT_REACHED"), readError(t, rec), "POST /subdistricts/3/medias response")
	})

	t.Run("service-errors", func(t *testing.T) {
		for _, tc := range uploadErrors {
			t.Run(tc.name, func(t *testing.T) {
				router, svc := newRouter(t)
				svc.EXPECT().AddMedia(gomock.Any(), int64(3), uploadKey).Return(nil, tc.err)

				rec := serve(t, router, jsonRequest(http.MethodPost, "/subdistricts/3/medias", body))
				assert.Equal(t, wantError(tc.wantStatus, tc.wantCode), readError(t, rec), "POST /subdistricts/3/medias response")
			})
		}
	})

	t.Run("success", func(t *testing.T) {
		router, svc := newRouter(t)
		svc.EXPECT().AddMedia(gomock.Any(), int64(3), uploadKey).Return(storedImage, nil)

		rec := serve(t, router, jsonRequest(http.MethodPost, "/subdistricts/3/medias", body))
		assert.Equal(t, wantData(t, http.StatusOK, storedImageJSON), readData(t, rec), "POST /subdistricts/3/medias response")
	})
}

func TestHandler_DeleteMedia(t *testing.T) {
	t.Run("invalid-ids", func(t *testing.T) {
		tests := []struct {
			name     string
			target   string
			wantCode string
		}{
			{name: "id-not-a-number", target: "/subdistricts/abc/medias/5", wantCode: "INVALID_SUB_DISTRICT_ID"},
			{name: "id-zero", target: "/subdistricts/0/medias/5", wantCode: "INVALID_SUB_DISTRICT_ID"},
			{name: "media-not-a-number", target: "/subdistricts/3/medias/abc", wantCode: "INVALID_MEDIA_ID"},
			{name: "media-fractional", target: "/subdistricts/3/medias/1.5", wantCode: "INVALID_MEDIA_ID"},
			{name: "media-zero", target: "/subdistricts/3/medias/0", wantCode: "INVALID_MEDIA_ID"},
			{name: "media-negative", target: "/subdistricts/3/medias/-1", wantCode: "INVALID_MEDIA_ID"},
			{name: "media-overflow", target: "/subdistricts/3/medias/99999999999999999999", wantCode: "INVALID_MEDIA_ID"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// No expectations: an invalid id must never reach the service.
				router, _ := newRouter(t)
				rec := serve(t, router, jsonRequest(http.MethodDelete, tc.target, ""))
				assert.Equal(t, wantError(http.StatusBadRequest, tc.wantCode), readError(t, rec), "DELETE %s response", tc.target)
			})
		}
	})

	t.Run("service-errors", func(t *testing.T) {
		tests := []struct {
			name       string
			err        error
			wantStatus int
			wantCode   string
		}{
			{name: "unknown-media", err: domain.ErrMediaNotFound, wantStatus: http.StatusNotFound, wantCode: "MEDIA_NOT_FOUND"},
			{name: "storage-failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				router, svc := newRouter(t)
				svc.EXPECT().DeleteMedia(gomock.Any(), int64(3), int64(5)).Return(tc.err)

				rec := serve(t, router, jsonRequest(http.MethodDelete, "/subdistricts/3/medias/5", ""))
				assert.Equal(t, wantError(tc.wantStatus, tc.wantCode), readError(t, rec), "DELETE /subdistricts/3/medias/5 response")
			})
		}
	})

	t.Run("success", func(t *testing.T) {
		router, svc := newRouter(t)
		svc.EXPECT().DeleteMedia(gomock.Any(), int64(3), int64(5)).Return(nil)

		rec := serve(t, router, jsonRequest(http.MethodDelete, "/subdistricts/3/medias/5", ""))
		assert.Equal(t, http.StatusNoContent, rec.Code, "DELETE /subdistricts/3/medias/5 status; body: %s", rec.Body)
		assert.Zero(t, rec.Body.Len(), "DELETE /subdistricts/3/medias/5 body = %q, want empty", rec.Body)
	})
}

func TestHandler_SetBoundary(t *testing.T) {
	t.Run("invalid-request", func(t *testing.T) {
		tests := []struct {
			name     string
			id       string
			body     string
			wantCode string
		}{
			{name: "id-not-a-number", id: "abc", body: squareJSON, wantCode: "INVALID_SUB_DISTRICT_ID"},
			{name: "id-zero", id: "0", body: squareJSON, wantCode: "INVALID_SUB_DISTRICT_ID"},
			{name: "empty-body", id: "3", body: ``, wantCode: "INVALID_BODY"},
			{name: "null", id: "3", body: `null`, wantCode: "INVALID_BODY"},
			{name: "array", id: "3", body: `[]`, wantCode: "INVALID_BODY"},
			{name: "trailing-data", id: "3", body: squareJSON + `{}`, wantCode: "INVALID_BODY"},
			{name: "point", id: "3", body: `{"type":"Point","coordinates":[106.8,-6.2]}`, wantCode: "INVALID_BODY"},
			{name: "no-type", id: "3", body: `{"coordinates":[[[[106.8,-6.2],[106.9,-6.2],[106.9,-6.3],[106.8,-6.2]]]]}`, wantCode: "INVALID_BODY"},
			{name: "no-polygons", id: "3", body: `{"type":"MultiPolygon","coordinates":[]}`, wantCode: "INVALID_BODY"},
			{name: "ring-not-closed", id: "3",
				body: `{"type":"Polygon","coordinates":[[[106.8,-6.2],[106.9,-6.2],[106.9,-6.3],[106.8,-6.3]]]}`, wantCode: "INVALID_BODY"},
			{name: "ring-three-positions", id: "3",
				body: `{"type":"Polygon","coordinates":[[[106.8,-6.2],[106.9,-6.2],[106.8,-6.2]]]}`, wantCode: "INVALID_BODY"},
			{name: "latitude-out-of-range", id: "3",
				body: `{"type":"Polygon","coordinates":[[[-6.2,106.8],[-6.2,106.9],[-6.3,106.9],[-6.2,106.8]]]}`, wantCode: "INVALID_BODY"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// No expectations: an invalid request must never reach the service.
				router, _ := newRouter(t)
				target := "/subdistricts/" + tc.id + "/boundary"
				rec := serve(t, router, jsonRequest(http.MethodPut, target, tc.body))
				assert.Equal(t, wantError(http.StatusBadRequest, tc.wantCode), readError(t, rec), "PUT %s %q response", target, tc.body)
			})
		}
	})

	t.Run("service-errors", func(t *testing.T) {
		tests := []struct {
			name       string
			err        error
			wantStatus int
			wantCode   string
		}{
			{name: "unknown", err: domain.ErrSubDistrictNotFound, wantStatus: http.StatusNotFound, wantCode: "SUB_DISTRICT_NOT_FOUND"},
			{name: "invalid-boundary", err: domain.ErrSubDistrictBoundaryInvalid, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "wrapped-invalid-boundary", err: fmt.Errorf("set: %w", domain.ErrSubDistrictBoundaryInvalid), wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
			{name: "storage-failure", err: errors.New("storage failure"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				router, svc := newRouter(t)
				svc.EXPECT().SetBoundary(gomock.Any(), int64(3), gomock.Any()).Return(nil, tc.err)

				rec := serve(t, router, jsonRequest(http.MethodPut, "/subdistricts/3/boundary", squareJSON))
				assert.Equal(t, wantError(tc.wantStatus, tc.wantCode), readError(t, rec), "PUT /subdistricts/3/boundary response")
			})
		}
	})

	t.Run("success", func(t *testing.T) {
		polygon := `{"type":"Polygon","coordinates":[[[106.8,-6.2],[106.9,-6.2],[106.9,-6.3],[106.8,-6.3],[106.8,-6.2]]]}`
		want := mustBoundary(t, squareJSON)

		router, svc := newRouter(t)
		svc.EXPECT().SetBoundary(gomock.Any(), int64(3), want).
			Return(&domain.SubDistrict{ID: 3, Name: "Menteng", Slug: "menteng", Boundary: want}, nil)

		rec := serve(t, router, jsonRequest(http.MethodPut, "/subdistricts/3/boundary", polygon))
		wantBody := wantData(t, http.StatusOK, `{
			"id": 3,
			"name": "Menteng",
			"slug": "menteng",
			"flood_history": null,
			"safety_crime_information": null,
			"estimated_daily_spending": null,
			"medias": [],
			"boundary": `+squareJSON+`
		}`)
		assert.Equal(t, wantBody, readData(t, rec), "PUT /subdistricts/3/boundary response")
	})
}
