package suggestions

import (
	"context"
	"errors"
	"net/http"

	"rantau-go/internal/domain"
	"rantau-go/internal/middleware"
	"rantau-go/internal/pkg/httpx"
	"rantau-go/internal/pkg/utils"
)

// Error codes of the suggestion endpoints.
const (
	codeInvalidSuggestionID       = "INVALID_SUGGESTION_ID"
	codeSuggestionNotFound        = "SUGGESTION_NOT_FOUND"
	codeAlreadyReviewed           = "ALREADY_REVIEWED"
	codeTooManyPendingSuggestions = "TOO_MANY_PENDING_SUGGESTIONS"
	codeInvalidStatus             = "INVALID_STATUS"
)

//go:generate mockgen -source=handler.go -destination=mock/handler.go -package=mock

// Service is the service the Handler uses.
type Service interface {
	Create(ctx context.Context, s *domain.Suggestion) (*domain.Suggestion, error)
	Fetch(ctx context.Context, id int64) (*domain.Suggestion, error)
	ListByUser(ctx context.Context, userID int64) ([]domain.Suggestion, error)
	List(ctx context.Context, filter Filter) ([]domain.Suggestion, error)
	Approve(ctx context.Context, id, adminID int64, p *domain.Place) (*domain.Suggestion, error)
	Reject(ctx context.Context, id, adminID int64, reason *string) (*domain.Suggestion, error)
}

// Handler serves the suggestion endpoints.
type Handler struct {
	service Service
}

// NewHandler returns a Handler that calls svc.
func NewHandler(svc Service) *Handler {
	return &Handler{
		service: svc,
	}
}

// Create saves the logged-in user's suggestion as pending review.
// The route must sit behind middleware.WithAuth, so the user is known.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.RespondError(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "missing user")
		return
	}

	req, ok := httpx.DecodeJSON[CreateRequest](w, r)
	if !ok {
		return
	}

	created, err := h.service.Create(r.Context(), toSuggestion(userID, *req))
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusCreated, toResponse(*created))
}

// ListMine sends back the logged-in user's suggestions, newest first.
// The route must sit behind middleware.WithAuth, so the user is known.
func (h *Handler) ListMine(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.RespondError(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "missing user")
		return
	}

	suggestions, err := h.service.ListByUser(r.Context(), userID)
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, utils.Map(suggestions, toResponse))
}

// List sends back all suggestions, oldest first.
// If the status query is set, only suggestions with that status are sent.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	filter := Filter{Status: domain.Status(r.URL.Query().Get("status"))}

	suggestions, err := h.service.List(r.Context(), filter)
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, utils.Map(suggestions, toResponse))
}

// Show sends back the suggestion whose id is in the URL.
// It sends an error if the id is invalid or the suggestion does not exist.
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	id, ok := suggestionID(w, r)
	if !ok {
		return
	}

	s, err := h.service.Fetch(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toResponse(*s))
}

// Approve makes the suggestion in the URL a place, using the body as its details.
// The route must sit behind middleware.WithAuth, so the admin is known.
func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
	id, ok := suggestionID(w, r)
	if !ok {
		return
	}

	adminID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.RespondError(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "missing user")
		return
	}

	req, ok := httpx.DecodeJSON[ApproveRequest](w, r)
	if !ok {
		return
	}

	approved, err := h.service.Approve(r.Context(), id, adminID, toPlace(*req))
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toResponse(*approved))
}

// Reject marks the suggestion in the URL as rejected, with an optional reason.
// The route must sit behind middleware.WithAuth, so the admin is known.
func (h *Handler) Reject(w http.ResponseWriter, r *http.Request) {
	id, ok := suggestionID(w, r)
	if !ok {
		return
	}

	adminID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.RespondError(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "missing user")
		return
	}

	req, ok := httpx.DecodeJSON[RejectRequest](w, r)
	if !ok {
		return
	}

	rejected, err := h.service.Reject(r.Context(), id, adminID, req.Reason)
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toResponse(*rejected))
}

// suggestionID reads the suggestion id from the URL, or answers 400.
func suggestionID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	return httpx.PathID(w, r, "id", codeInvalidSuggestionID, "invalid suggestion_id")
}

// isValidationError reports whether err is a refused suggestion or place detail.
// Approving can also fail with place errors, which are shown as validation errors.
func isValidationError(err error) bool {
	switch {
	case errors.Is(err, domain.ErrSuggestionLocationInvalid):
		return true
	case errors.Is(err, domain.ErrSuggestionNoteTooLong):
		return true
	case errors.Is(err, domain.ErrSuggestionAttributesInvalid):
		return true
	case errors.Is(err, domain.ErrSuggestionReasonTooLong):
		return true
	case errors.Is(err, domain.ErrSubDistrictNotFound):
		return true
	case errors.Is(err, domain.ErrCategoryNotFound):
		return true
	}

	return domain.IsPlaceValidationError(err)
}

// respondError sends the right error response for a failed service call.
func respondError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrSuggestionNotFound):
		httpx.RespondError(w, http.StatusNotFound, codeSuggestionNotFound, err.Error())
	case errors.Is(err, domain.ErrSuggestionAlreadyReviewed):
		httpx.RespondError(w, http.StatusConflict, codeAlreadyReviewed, err.Error())
	case errors.Is(err, domain.ErrSuggestionLimitReached):
		httpx.RespondError(w, http.StatusTooManyRequests, codeTooManyPendingSuggestions, err.Error())
	case errors.Is(err, domain.ErrSuggestionStatusInvalid):
		httpx.RespondError(w, http.StatusBadRequest, codeInvalidStatus, err.Error())
	case isValidationError(err):
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeValidationError, err.Error())
	default:
		httpx.RespondInternalError(w)
	}
}
