package reports

import (
	"context"
	"errors"
	"net/http"

	"sakukota-go/internal/domain"
	"sakukota-go/internal/middleware"
	"sakukota-go/internal/pkg/httpx"
	"sakukota-go/internal/pkg/utils"
)

// Error codes of the report endpoints.
const (
	codeInvalidPlaceID        = "INVALID_PLACE_ID"
	codeInvalidReviewID       = "INVALID_REVIEW_ID"
	codeInvalidReportID       = "INVALID_REPORT_ID"
	codePlaceNotFound         = "PLACE_NOT_FOUND"
	codeReviewNotFound        = "REVIEW_NOT_FOUND"
	codeReportNotFound        = "REPORT_NOT_FOUND"
	codeAlreadyReported       = "ALREADY_REPORTED"
	codeAlreadyResolved       = "ALREADY_RESOLVED"
	codeTooManyPendingReports = "TOO_MANY_PENDING_REPORTS"
	codeInvalidStatus         = "INVALID_STATUS"
)

//go:generate mockgen -source=handler.go -destination=mock/handler.go -package=mock

// Service is the service the Handler uses.
type Service interface {
	Create(ctx context.Context, r *domain.Report) (*domain.Report, error)
	List(ctx context.Context, filter Filter) ([]domain.Report, error)
	Fetch(ctx context.Context, id int64) (*domain.Report, error)
	Remove(ctx context.Context, id, adminID int64) (*domain.Report, error)
	Dismiss(ctx context.Context, id, adminID int64) (*domain.Report, error)
}

// Handler serves the report endpoints.
type Handler struct {
	service Service

	// mediaURL turns a stored media key into its public URL.
	mediaURL func(key string) string
}

// NewHandler returns a Handler that calls svc and builds media links with mediaURL.
func NewHandler(svc Service, mediaURL func(key string) string) *Handler {
	return &Handler{
		service:  svc,
		mediaURL: mediaURL,
	}
}

// CreateForPlace saves the logged-in user's report of the place in the URL.
// The route must sit behind middleware.WithAuth, so the user is known.
func (h *Handler) CreateForPlace(w http.ResponseWriter, r *http.Request) {
	placeID, ok := httpx.PathID(w, r, "id", codeInvalidPlaceID, "invalid place_id")
	if !ok {
		return
	}

	h.create(w, r, placeID, nil)
}

// CreateForReview saves the logged-in user's report of the review in the URL,
// which must belong to the place in the URL. The route must sit behind
// middleware.WithAuth, so the user is known.
func (h *Handler) CreateForReview(w http.ResponseWriter, r *http.Request) {
	placeID, ok := httpx.PathID(w, r, "id", codeInvalidPlaceID, "invalid place_id")
	if !ok {
		return
	}

	reviewID, ok := httpx.PathID(w, r, "reviewId", codeInvalidReviewID, "invalid review_id")
	if !ok {
		return
	}

	h.create(w, r, placeID, &reviewID)
}

// create saves the logged-in user's report of the place, or of its review
// when reviewID is not nil.
func (h *Handler) create(w http.ResponseWriter, r *http.Request, placeID int64, reviewID *int64) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.RespondError(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "missing user")
		return
	}

	req, ok := httpx.DecodeJSON[CreateRequest](w, r)
	if !ok {
		return
	}

	created, err := h.service.Create(r.Context(), toReport(userID, placeID, reviewID, *req))
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusCreated, toResponse(*created))
}

// List sends back all reports, oldest first.
// If the status query is set, only reports with that status are sent.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	filter := Filter{Status: domain.ReportStatus(r.URL.Query().Get("status"))}

	reports, err := h.service.List(r.Context(), filter)
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, utils.Map(reports, toResponse))
}

// Show sends back the report in the URL with its content as it is now.
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	id, ok := reportID(w, r)
	if !ok {
		return
	}

	report, err := h.service.Fetch(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toDetailResponse(*report, h.mediaURL))
}

// Remove deletes the content reported by the report in the URL and resolves
// the report as removed. The route must sit behind middleware.WithAuth, so the
// admin is known.
func (h *Handler) Remove(w http.ResponseWriter, r *http.Request) {
	h.resolve(w, r, h.service.Remove)
}

// Dismiss resolves the report in the URL as dismissed, keeping the content.
// The route must sit behind middleware.WithAuth, so the admin is known.
func (h *Handler) Dismiss(w http.ResponseWriter, r *http.Request) {
	h.resolve(w, r, h.service.Dismiss)
}

// resolve resolves the report in the URL as the logged-in admin with fn.
func (h *Handler) resolve(w http.ResponseWriter, r *http.Request, fn func(ctx context.Context, id, adminID int64) (*domain.Report, error)) {
	id, ok := reportID(w, r)
	if !ok {
		return
	}

	adminID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.RespondError(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "missing user")
		return
	}

	resolved, err := fn(r.Context(), id, adminID)
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toResponse(*resolved))
}

// reportID reads the report id from the URL, or answers 400.
func reportID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	return httpx.PathID(w, r, "id", codeInvalidReportID, "invalid report_id")
}

// respondError sends the right error response for a failed service call.
func respondError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrReportReasonInvalid), errors.Is(err, domain.ErrReportNoteTooLong):
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeValidationError, err.Error())
	case errors.Is(err, domain.ErrReportStatusInvalid):
		httpx.RespondError(w, http.StatusBadRequest, codeInvalidStatus, err.Error())
	case errors.Is(err, domain.ErrPlaceNotFound):
		httpx.RespondError(w, http.StatusNotFound, codePlaceNotFound, err.Error())
	case errors.Is(err, domain.ErrReviewNotFound):
		httpx.RespondError(w, http.StatusNotFound, codeReviewNotFound, err.Error())
	case errors.Is(err, domain.ErrReportNotFound):
		httpx.RespondError(w, http.StatusNotFound, codeReportNotFound, err.Error())
	case errors.Is(err, domain.ErrAlreadyReported):
		httpx.RespondError(w, http.StatusConflict, codeAlreadyReported, err.Error())
	case errors.Is(err, domain.ErrReportAlreadyResolved):
		httpx.RespondError(w, http.StatusConflict, codeAlreadyResolved, err.Error())
	case errors.Is(err, domain.ErrReportLimitReached):
		httpx.RespondError(w, http.StatusTooManyRequests, codeTooManyPendingReports, err.Error())
	default:
		httpx.RespondInternalError(w)
	}
}
