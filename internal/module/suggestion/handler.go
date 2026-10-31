package suggestion

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"rantau-go/internal/httpx"
	"rantau-go/internal/middleware"
	"rantau-go/internal/module/place"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

// -----------------------------------------------------------------------------
// Handler
// -----------------------------------------------------------------------------

// CreateSuggestion stores the authenticated user's suggestion as pending
// review. It must be served behind middleware.WithAuth.
func (h *Handler) CreateSuggestion(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.RespondError(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "missing user")
		return
	}

	var req *CreateSuggestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req == nil {
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeInvalidBody, "invalid request body")
		return
	}

	created, err := h.service.Create(r.Context(), toSuggestion(userID, *req))
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusCreated, toSuggestionResponse(*created))
}

// GetMySuggestions lists the authenticated user's suggestions, newest first.
// It must be served behind middleware.WithAuth.
func (h *Handler) GetMySuggestions(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.RespondError(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "missing user")
		return
	}

	suggestions, err := h.service.GetByUser(r.Context(), userID)
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toSuggestionResponses(suggestions))
}

// GetSuggestions lists suggestions, optionally filtered by the `status` query
// parameter, oldest first.
func (h *Handler) GetSuggestions(w http.ResponseWriter, r *http.Request) {
	filter := SuggestionFilter{Status: Status(r.URL.Query().Get("status"))}

	suggestions, err := h.service.GetMany(r.Context(), filter)
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toSuggestionResponses(suggestions))
}

// GetSuggestion returns the suggestion given by the `id` path parameter.
func (h *Handler) GetSuggestion(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.ParseID(chi.URLParam(r, "id"))
	if !ok {
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeInvalidSuggestionID, "invalid suggestion_id")
		return
	}

	s, err := h.service.GetSuggestion(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toSuggestionResponse(*s))
}

// ApproveSuggestion creates the place given in the body from the suggestion
// given by the `id` path parameter, and marks the suggestion approved. It must
// be served behind middleware.WithAuth.
func (h *Handler) ApproveSuggestion(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.ParseID(chi.URLParam(r, "id"))
	if !ok {
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeInvalidSuggestionID, "invalid suggestion_id")
		return
	}

	adminID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.RespondError(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "missing user")
		return
	}

	var req *ApproveSuggestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req == nil {
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeInvalidBody, "invalid request body")
		return
	}

	approved, err := h.service.Approve(r.Context(), id, adminID, toPlace(*req))
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toSuggestionResponse(*approved))
}

// RejectSuggestion rejects the suggestion given by the `id` path parameter,
// with the optional reason in the body. It must be served behind
// middleware.WithAuth.
func (h *Handler) RejectSuggestion(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.ParseID(chi.URLParam(r, "id"))
	if !ok {
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeInvalidSuggestionID, "invalid suggestion_id")
		return
	}

	adminID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.RespondError(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "missing user")
		return
	}

	var req *RejectSuggestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req == nil {
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeInvalidBody, "invalid request body")
		return
	}

	rejected, err := h.service.Reject(r.Context(), id, adminID, req.Reason)
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toSuggestionResponse(*rejected))
}

// respondError writes the response for an error from the service. Approving
// also returns the errors of creating the place, which are validation errors.
func respondError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrSuggestionNotFound):
		httpx.RespondError(w, http.StatusNotFound, httpx.CodeSuggestionNotFound, err.Error())
	case errors.Is(err, ErrAlreadyReviewed):
		httpx.RespondError(w, http.StatusConflict, httpx.CodeAlreadyReviewed, err.Error())
	case errors.Is(err, ErrTooManyPending):
		httpx.RespondError(w, http.StatusTooManyRequests, httpx.CodeTooManyPendingSuggestions, err.Error())
	case errors.Is(err, ErrInvalidStatus):
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeInvalidStatus, err.Error())
	case errors.Is(err, ErrInvalidName),
		errors.Is(err, ErrNameTooLong),
		errors.Is(err, ErrAddressTooLong),
		errors.Is(err, ErrOperatingHoursTooLong),
		errors.Is(err, ErrEstimatedPriceRangeTooLong),
		errors.Is(err, ErrInvalidLocation),
		errors.Is(err, ErrNoteTooLong),
		errors.Is(err, ErrInvalidAttributes),
		errors.Is(err, ErrReasonTooLong),
		errors.Is(err, ErrSubDistrictNotFound),
		errors.Is(err, ErrCategoryNotFound),
		errors.Is(err, place.ErrInvalidName),
		errors.Is(err, place.ErrNameTooLong),
		errors.Is(err, place.ErrAddressTooLong),
		errors.Is(err, place.ErrOperatingHoursTooLong),
		errors.Is(err, place.ErrEstimatedPriceRangeTooLong),
		errors.Is(err, place.ErrInvalidLocation),
		errors.Is(err, place.ErrInvalidAttributes),
		errors.Is(err, place.ErrSubDistrictNotFound),
		errors.Is(err, place.ErrCategoryNotFound):
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeValidationError, err.Error())
	default:
		httpx.RespondInternalError(w)
	}
}
