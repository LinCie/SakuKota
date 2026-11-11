package review

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

func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

// -----------------------------------------------------------------------------
// Handler
// -----------------------------------------------------------------------------

// CreateReview stores the authenticated user's review of the place given by
// the `id` path parameter. It must be served behind middleware.WithAuth.
func (h *Handler) CreateReview(w http.ResponseWriter, r *http.Request) {
	placeID, ok := httpx.ParseID(chi.URLParam(r, "id"))
	if !ok {
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeInvalidPlaceID, "invalid place_id")
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

	created, err := h.service.Create(r.Context(), toReview(placeID, userID, *req))
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

// GetReviews lists the reviews of the place given by the `id` path parameter,
// newest first.
func (h *Handler) GetReviews(w http.ResponseWriter, r *http.Request) {
	placeID, ok := httpx.ParseID(chi.URLParam(r, "id"))
	if !ok {
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeInvalidPlaceID, "invalid place_id")
		return
	}

	reviews, err := h.service.GetByPlace(r.Context(), placeID)
	if err != nil {
		if errors.Is(err, ErrPlaceNotFound) {
			httpx.RespondError(w, http.StatusNotFound, httpx.CodePlaceNotFound, err.Error())
			return
		}
		httpx.RespondInternalError(w)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toReviewResponses(reviews))
}
