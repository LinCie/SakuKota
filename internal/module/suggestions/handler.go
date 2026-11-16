package suggestions

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"rantau-go/internal/middleware"
	"rantau-go/internal/pkg/httpx"
)

type Handler struct {
	service Service
}

// NewHandler creates a Handler that passes its work to the suggestion service.
func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

// -----------------------------------------------------------------------------
// Handler
// -----------------------------------------------------------------------------

// CreateSuggestion saves the logged-in user's suggestion as pending review.
// The route must sit behind middleware.WithAuth, so the user is known.
func (h *Handler) CreateSuggestion(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.RespondError(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "missing user")
		return
	}

	req, ok := httpx.DecodeJSON[CreateSuggestionRequest](w, r)
	if !ok {
		return
	}

	created, err := h.service.Create(r.Context(), toSuggestion(userID, *req))
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusCreated, toSuggestionResponse(*created))
}

// GetMySuggestions sends back the logged-in user's suggestions, newest first.
// The route must sit behind middleware.WithAuth, so the user is known.
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

// GetSuggestions sends back all suggestions, oldest first.
// If the status query is set, only suggestions with that status are sent.
func (h *Handler) GetSuggestions(w http.ResponseWriter, r *http.Request) {
	filter := SuggestionFilter{Status: Status(r.URL.Query().Get("status"))}

	suggestions, err := h.service.GetMany(r.Context(), filter)
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toSuggestionResponses(suggestions))
}

// GetSuggestion sends back the suggestion whose id is in the URL.
// It sends an error if the id is invalid or the suggestion does not exist.
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

// ApproveSuggestion makes the suggestion in the URL a place, using the body as its details.
// The route must sit behind middleware.WithAuth, so the admin is known.
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

	// The body is the place to create, in the shape of POST /places.
	req, ok := httpx.DecodeJSON[ApproveSuggestionRequest](w, r)
	if !ok {
		return
	}

	approved, err := h.service.Approve(r.Context(), id, adminID, toPlace(*req))
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toSuggestionResponse(*approved))
}

// RejectSuggestion marks the suggestion in the URL as rejected, with an optional reason.
// The route must sit behind middleware.WithAuth, so the admin is known.
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

	req, ok := httpx.DecodeJSON[RejectSuggestionRequest](w, r)
	if !ok {
		return
	}

	rejected, err := h.service.Reject(r.Context(), id, adminID, req.Reason)
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toSuggestionResponse(*rejected))
}

// respondError sends the right error response for a failed service call.
// Approving can also fail with place errors, which are shown as validation errors.
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
		errors.Is(err, ErrPlaceInvalidLocation),
		errors.Is(err, ErrPlaceInvalidAttributes):
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeValidationError, err.Error())
	default:
		httpx.RespondInternalError(w)
	}
}
