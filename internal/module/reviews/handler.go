package reviews

import (
	"context"
	"errors"
	"net/http"

	"rantau-go/internal/domain"
	"rantau-go/internal/middleware"
	"rantau-go/internal/pkg/httpx"
	"rantau-go/internal/pkg/utils"
)

// Error codes sent to clients, in addition to the shared ones in httpx.
const (
	codeInvalidPlaceID  = "INVALID_PLACE_ID"
	codePlaceNotFound   = "PLACE_NOT_FOUND"
	codeAlreadyReviewed = "ALREADY_REVIEWED"
)

//go:generate mockgen -source=handler.go -destination=mock/handler.go -package=mock

// reviewService is the part of Service the Handler uses.
type reviewService interface {
	Create(ctx context.Context, r *domain.Review) (*domain.Review, error)
	ListByPlace(ctx context.Context, placeID int64) ([]domain.Review, error)
}

// Handler serves the review endpoints.
type Handler struct {
	service reviewService
}

// NewHandler returns a Handler that uses service.
func NewHandler(service reviewService) *Handler {
	return &Handler{service: service}
}

// Create saves the logged-in user's review of the place in the URL.
// The route must sit behind middleware.WithAuth, so the user is known.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	id, ok := placeID(w, r)
	if !ok {
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.RespondError(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "missing user")
		return
	}

	req, ok := httpx.DecodeJSON[CreateRequest](w, r)
	if !ok {
		return
	}

	created, err := h.service.Create(r.Context(), toReview(id, userID, *req))
	if err != nil {
		switch {
		case isValidationError(err):
			httpx.RespondError(w, http.StatusBadRequest, httpx.CodeValidationError, err.Error())
		case errors.Is(err, domain.ErrPlaceNotFound):
			httpx.RespondError(w, http.StatusNotFound, codePlaceNotFound, err.Error())
		case errors.Is(err, domain.ErrPlaceAlreadyReviewed):
			httpx.RespondError(w, http.StatusConflict, codeAlreadyReviewed, err.Error())
		default:
			httpx.RespondInternalError(w)
		}
		return
	}

	httpx.RespondJSON(w, http.StatusCreated, toResponse(*created))
}

// List sends back the reviews of the place in the URL, newest first.
// If there is no such place, it sends a not-found error.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	id, ok := placeID(w, r)
	if !ok {
		return
	}

	reviews, err := h.service.ListByPlace(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrPlaceNotFound) {
			httpx.RespondError(w, http.StatusNotFound, codePlaceNotFound, err.Error())
			return
		}
		httpx.RespondInternalError(w)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, utils.Map(reviews, toResponse))
}

// placeID reads the place id from the URL, or answers 400.
func placeID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	return httpx.PathID(w, r, "id", codeInvalidPlaceID, "invalid place_id")
}

// isValidationError reports whether err means the review's data was refused.
func isValidationError(err error) bool {
	return errors.Is(err, domain.ErrReviewRatingInvalid) || errors.Is(err, domain.ErrReviewCommentTooLong)
}
