package reviews

import (
	"errors"
	"net/http"

	"rantau-go/internal/middleware"
	"rantau-go/internal/pkg/httpx"
	"rantau-go/internal/pkg/utils"
)

type Handler struct {
	service Service
}

// -----------------------------------------------------------------------------
// Exported
// -----------------------------------------------------------------------------

func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

// CreateReview saves the logged-in user's review of the place in the URL.
// The route must sit behind middleware.WithAuth, so the user is known.
func (h *Handler) CreateReview(w http.ResponseWriter, r *http.Request) {
	id, ok := placeID(w, r)
	if !ok {
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.RespondError(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "missing user")
		return
	}

	req, ok := httpx.DecodeJSON[CreateReviewRequest](w, r)
	if !ok {
		return
	}

	created, err := h.service.Create(r.Context(), toReview(id, userID, *req))
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidRating),
			errors.Is(err, ErrCommentTooLong):
			httpx.RespondError(w, http.StatusBadRequest, httpx.CodeValidationError, err.Error())
		case errors.Is(err, ErrPlaceNotFound):
			httpx.RespondError(w, http.StatusNotFound, httpx.CodePlaceNotFound, err.Error())
		case errors.Is(err, ErrAlreadyReviewed):
			httpx.RespondError(w, http.StatusConflict, httpx.CodeAlreadyReviewed, err.Error())
		default:
			httpx.RespondInternalError(w)
		}
		return
	}

	httpx.RespondJSON(w, http.StatusCreated, toReviewResponse(*created))
}

// GetReviews sends back the reviews of the place in the URL, newest first.
// If there is no such place, it sends a not-found error.
func (h *Handler) GetReviews(w http.ResponseWriter, r *http.Request) {
	id, ok := placeID(w, r)
	if !ok {
		return
	}

	reviews, err := h.service.GetByPlace(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrPlaceNotFound) {
			httpx.RespondError(w, http.StatusNotFound, httpx.CodePlaceNotFound, err.Error())
			return
		}
		httpx.RespondInternalError(w)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, utils.Map(reviews, toReviewResponse))
}

// -----------------------------------------------------------------------------
// Unexported
// -----------------------------------------------------------------------------

// placeID reads the place id from the URL, or answers 400.
func placeID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	return httpx.PathID(w, r, "id", httpx.CodeInvalidPlaceID, "invalid place_id")
}
