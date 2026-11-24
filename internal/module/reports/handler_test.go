package reports_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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
	"sakukota-go/internal/module/reports"
	"sakukota-go/internal/module/reports/mock"
	"sakukota-go/internal/pkg/jwt"
)

// testConfig signs and checks the access tokens in these tests.
var testConfig = config.Config{JWTSecret: "test-secret", JWTExpiry: time.Hour}

// newRouter serves the report handlers on the paths the API uses for them,
// behind WithAuth as in production. The admin role is checked by the API
// router, not by the handlers.
func newRouter(svc *mock.MockService) http.Handler {
	h := reports.NewHandler(svc, mediaURL)
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(middleware.WithAuth(testConfig))
		r.Post("/places/{id}/reports", h.CreateForPlace)
		r.Post("/places/{id}/reviews/{reviewId}/reports", h.CreateForReview)
		r.Get("/reports", h.List)
		r.Get("/reports/{id}", h.Show)
		r.Post("/reports/{id}/remove", h.Remove)
		r.Post("/reports/{id}/dismiss", h.Dismiss)
	})
	return r
}

// mediaURL builds the public URL of a stored media key.
func mediaURL(key string) string { return "https://cdn.test/" + key }

// newMockService returns a mock report service that fails the test on any
// call without an expectation.
func newMockService(t *testing.T) *mock.MockService {
	t.Helper()
	return mock.NewMockService(gomock.NewController(t))
}

// request sends a JSON request with an access token for user 7 through h.
func request(t *testing.T, h http.Handler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	token, err := jwt.GenerateToken(testConfig, 7, "user")
	require.NoError(t, err)

	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// errorCode returns the code of the error response in rec.
func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), "body: %s", rec.Body)
	assert.NotEmpty(t, body.Message)
	return body.Code
}

// linReport is a pending report by Lin of review 9 of place 4.
func linReport() *domain.Report {
	return &domain.Report{
		ID:        3,
		UserID:    7,
		PlaceID:   4,
		ReviewID:  new(int64(9)),
		Reason:    domain.ReportReasonSpam,
		Note:      new("Iklan"),
		Status:    domain.ReportStatusPending,
		CreatedAt: time.Date(2026, 10, 4, 12, 30, 0, 0, time.UTC),
		User:      domain.Author{ID: 7, Name: "Lin"},
	}
}

// linReportJSON is linReport as the API returns it.
const linReportJSON = `{
	"id": 3,
	"place_id": 4,
	"review_id": 9,
	"reason": "spam",
	"note": "Iklan",
	"status": "pending",
	"resolved_at": null,
	"created_at": "2026-10-04T12:30:00Z",
	"user": {"id": 7, "name": "Lin"}
}`

func TestHandler_Create(t *testing.T) {
	t.Run("place_report", func(t *testing.T) {
		svc := newMockService(t)
		want := &domain.Report{UserID: 7, PlaceID: 4, Reason: domain.ReportReasonFake, Note: new("  Tutup  ")}
		svc.EXPECT().Create(gomock.Any(), want).Return(linReport(), nil)

		// The author comes from the token and the target from the URL.
		body := `{"reason":"fake","note":"  Tutup  ","user_id":1,"place_id":2,"review_id":3,"status":"removed"}`
		rec := request(t, newRouter(svc), http.MethodPost, "/places/4/reports", body)
		assert.Equal(t, http.StatusCreated, rec.Code, "body: %s", rec.Body)
		assert.JSONEq(t, `{"data": `+linReportJSON+`}`, rec.Body.String())
	})

	t.Run("review_report", func(t *testing.T) {
		svc := newMockService(t)
		want := &domain.Report{UserID: 7, PlaceID: 4, ReviewID: new(int64(9)), Reason: domain.ReportReasonSpam}
		svc.EXPECT().Create(gomock.Any(), want).Return(linReport(), nil)

		rec := request(t, newRouter(svc), http.MethodPost, "/places/4/reviews/9/reports", `{"reason":"spam"}`)
		assert.Equal(t, http.StatusCreated, rec.Code, "body: %s", rec.Body)
	})

	t.Run("refused_before_service", func(t *testing.T) {
		tests := []struct {
			name, target, body string
			status             int
			code               string
		}{
			{name: "bad_place_id", target: "/places/0/reports", body: `{"reason":"spam"}`, status: http.StatusBadRequest, code: "INVALID_PLACE_ID"},
			{name: "bad_review_place_id", target: "/places/x/reviews/9/reports", body: `{"reason":"spam"}`, status: http.StatusBadRequest, code: "INVALID_PLACE_ID"},
			{name: "bad_review_id", target: "/places/4/reviews/-1/reports", body: `{"reason":"spam"}`, status: http.StatusBadRequest, code: "INVALID_REVIEW_ID"},
			{name: "empty_body", target: "/places/4/reports", body: ``, status: http.StatusBadRequest, code: "INVALID_BODY"},
			{name: "null_body", target: "/places/4/reports", body: `null`, status: http.StatusBadRequest, code: "INVALID_BODY"},
			{name: "reason_is_a_number", target: "/places/4/reports", body: `{"reason":1}`, status: http.StatusBadRequest, code: "INVALID_BODY"},
			{name: "note_is_a_number", target: "/places/4/reports", body: `{"reason":"spam","note":1}`, status: http.StatusBadRequest, code: "INVALID_BODY"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				rec := request(t, newRouter(newMockService(t)), http.MethodPost, tc.target, tc.body)
				assert.Equal(t, tc.status, rec.Code, "body: %s", rec.Body)
				assert.Equal(t, tc.code, errorCode(t, rec))
			})
		}
	})

	t.Run("without_user", func(t *testing.T) {
		svc := newMockService(t)
		r := chi.NewRouter()
		r.Post("/places/{id}/reports", reports.NewHandler(svc, mediaURL).CreateForPlace)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/places/4/reports", strings.NewReader(`{"reason":"spam"}`)))
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Equal(t, "UNAUTHORIZED", errorCode(t, rec))
	})

	t.Run("maps_service_errors", func(t *testing.T) {
		tests := []struct {
			err    error
			status int
			code   string
		}{
			{err: domain.ErrReportReasonInvalid, status: http.StatusBadRequest, code: "VALIDATION_ERROR"},
			{err: domain.ErrReportNoteTooLong, status: http.StatusBadRequest, code: "VALIDATION_ERROR"},
			{err: domain.ErrPlaceNotFound, status: http.StatusNotFound, code: "PLACE_NOT_FOUND"},
			{err: domain.ErrReviewNotFound, status: http.StatusNotFound, code: "REVIEW_NOT_FOUND"},
			{err: domain.ErrAlreadyReported, status: http.StatusConflict, code: "ALREADY_REPORTED"},
			{err: domain.ErrReportLimitReached, status: http.StatusTooManyRequests, code: "TOO_MANY_PENDING_REPORTS"},
			{err: errors.New("pq: connection reset"), status: http.StatusInternalServerError, code: "INTERNAL_ERROR"},
		}

		for _, tc := range tests {
			t.Run(tc.code, func(t *testing.T) {
				svc := newMockService(t)
				svc.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, tc.err)

				rec := request(t, newRouter(svc), http.MethodPost, "/places/4/reports", `{"reason":"spam"}`)
				assert.Equal(t, tc.status, rec.Code, "body: %s", rec.Body)
				assert.Equal(t, tc.code, errorCode(t, rec))
				assert.NotContains(t, rec.Body.String(), "pq:")
			})
		}
	})
}

func TestHandler_List(t *testing.T) {
	t.Run("passes_status_filter", func(t *testing.T) {
		svc := newMockService(t)
		svc.EXPECT().List(gomock.Any(), reports.Filter{Status: domain.ReportStatusPending}).Return([]domain.Report{*linReport()}, nil)

		rec := request(t, newRouter(svc), http.MethodGet, "/reports?status=pending", "")
		assert.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body)
		assert.JSONEq(t, `{"data": [`+linReportJSON+`]}`, rec.Body.String())
	})

	t.Run("empty_list_is_an_array", func(t *testing.T) {
		svc := newMockService(t)
		svc.EXPECT().List(gomock.Any(), reports.Filter{}).Return(nil, nil)

		rec := request(t, newRouter(svc), http.MethodGet, "/reports", "")
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `{"data": []}`, rec.Body.String())
	})

	t.Run("invalid_status", func(t *testing.T) {
		svc := newMockService(t)
		svc.EXPECT().List(gomock.Any(), reports.Filter{Status: "open"}).Return(nil, domain.ErrReportStatusInvalid)

		rec := request(t, newRouter(svc), http.MethodGet, "/reports?status=open", "")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "INVALID_STATUS", errorCode(t, rec))
	})
}

func TestHandler_Show(t *testing.T) {
	t.Run("review_report_with_its_content", func(t *testing.T) {
		svc := newMockService(t)
		report := linReport()
		report.Place = &domain.Place{
			ID:               4,
			Name:             "Warung Bu Ani",
			Address:          new("Jl. Sabang 12"),
			ShortDescription: nil,
			Category:         domain.Category{ID: 2, Name: "Warung", Slug: "warung", Code: "W", Label: "Warung", Color: "chili"},
			Location:         nil,
		}
		report.Review = &domain.Review{
			ID:        9,
			PlaceID:   4,
			Rating:    1,
			Comment:   new("Pinjol murah!"),
			CreatedAt: time.Date(2026, 10, 3, 8, 0, 0, 0, time.UTC),
			User:      domain.Author{ID: 5, Name: "Budi"},
			Medias:    []domain.Media{{ID: 11, Key: "reviews/4/5/a.webp", ContentType: "image/webp", Size: 2048, Position: 1}},
		}
		svc.EXPECT().Fetch(gomock.Any(), int64(3)).Return(report, nil)

		rec := request(t, newRouter(svc), http.MethodGet, "/reports/3", "")
		assert.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body)
		assert.JSONEq(t, `{"data": `+strings.TrimSuffix(linReportJSON, "}")+`,
			"place": {
				"id": 4,
				"name": "Warung Bu Ani",
				"address": "Jl. Sabang 12",
				"short_description": null,
				"category": {"id": 2, "name": "Warung", "slug": "warung", "code": "W", "label": "Warung", "color": "chili"},
				"medias": []
			},
			"review": {
				"id": 9,
				"rating": 1,
				"comment": "Pinjol murah!",
				"created_at": "2026-10-03T08:00:00Z",
				"user": {"id": 5, "name": "Budi"},
				"medias": [{"id": 11, "url": "https://cdn.test/reviews/4/5/a.webp", "content_type": "image/webp", "size": 2048, "position": 1}]
			}
		}}`, rec.Body.String())
	})

	t.Run("deleted_content_is_null", func(t *testing.T) {
		svc := newMockService(t)
		svc.EXPECT().Fetch(gomock.Any(), int64(3)).Return(linReport(), nil)

		rec := request(t, newRouter(svc), http.MethodGet, "/reports/3", "")
		assert.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body)
		assert.JSONEq(t, `{"data": `+strings.TrimSuffix(linReportJSON, "}")+`, "place": null, "review": null}}`, rec.Body.String())
	})

	t.Run("bad_id", func(t *testing.T) {
		rec := request(t, newRouter(newMockService(t)), http.MethodGet, "/reports/0", "")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "INVALID_REPORT_ID", errorCode(t, rec))
	})

	t.Run("not_found", func(t *testing.T) {
		svc := newMockService(t)
		svc.EXPECT().Fetch(gomock.Any(), int64(3)).Return(nil, domain.ErrReportNotFound)

		rec := request(t, newRouter(svc), http.MethodGet, "/reports/3", "")
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Equal(t, "REPORT_NOT_FOUND", errorCode(t, rec))
	})
}

func TestHandler_Resolve(t *testing.T) {
	for _, action := range []string{"remove", "dismiss"} {
		t.Run(action, func(t *testing.T) {
			// expect sets the service call the action makes for report 3 by admin 7.
			expect := func(svc *mock.MockService) *gomock.Call {
				if action == "remove" {
					return svc.EXPECT().Remove(gomock.Any(), int64(3), int64(7))
				}
				return svc.EXPECT().Dismiss(gomock.Any(), int64(3), int64(7))
			}

			t.Run("returns_report", func(t *testing.T) {
				svc := newMockService(t)
				resolved := linReport()
				if action == "remove" {
					resolved.Status = domain.ReportStatusRemoved
				} else {
					resolved.Status = domain.ReportStatusDismissed
				}
				resolved.ResolvedBy = new(int64(7))
				resolved.ResolvedAt = new(time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC))
				expect(svc).Return(resolved, nil)

				rec := request(t, newRouter(svc), http.MethodPost, "/reports/3/"+action, "")
				assert.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body)
				assert.Contains(t, rec.Body.String(), `"resolved_at":"2026-10-05T09:00:00Z"`)
				assert.NotContains(t, rec.Body.String(), "resolved_by")
			})

			t.Run("bad_id", func(t *testing.T) {
				rec := request(t, newRouter(newMockService(t)), http.MethodPost, "/reports/abc/"+action, "")
				assert.Equal(t, http.StatusBadRequest, rec.Code)
				assert.Equal(t, "INVALID_REPORT_ID", errorCode(t, rec))
			})

			tests := []struct {
				err    error
				status int
				code   string
			}{
				{err: domain.ErrReportNotFound, status: http.StatusNotFound, code: "REPORT_NOT_FOUND"},
				{err: domain.ErrReportAlreadyResolved, status: http.StatusConflict, code: "ALREADY_RESOLVED"},
				{err: errors.New("boom"), status: http.StatusInternalServerError, code: "INTERNAL_ERROR"},
			}
			for _, tc := range tests {
				t.Run(tc.code, func(t *testing.T) {
					svc := newMockService(t)
					expect(svc).Return(nil, tc.err)

					rec := request(t, newRouter(svc), http.MethodPost, "/reports/3/"+action, "")
					assert.Equal(t, tc.status, rec.Code, "body: %s", rec.Body)
					assert.Equal(t, tc.code, errorCode(t, rec))
				})
			}
		})
	}
}
