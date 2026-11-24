package reviews

import (
	"context"
	"errors"
	"net/http"

	"sakukota-go/internal/domain"
	"sakukota-go/internal/middleware"
	"sakukota-go/internal/pkg/httpx"
	"sakukota-go/internal/pkg/utils"
	"sakukota-go/internal/storage"
)

// Error codes sent to clients, in addition to the shared ones in httpx.
const (
	codeInvalidPlaceID    = "INVALID_PLACE_ID"
	codePlaceNotFound     = "PLACE_NOT_FOUND"
	codeAlreadyReviewed   = "ALREADY_REVIEWED"
	codeReviewNotFound    = "REVIEW_NOT_FOUND"
	codeInvalidReviewID   = "INVALID_REVIEW_ID"
	codeInvalidMediaID    = "INVALID_MEDIA_ID"
	codeMediaNotFound     = "MEDIA_NOT_FOUND"
	codeMediaLimitReached = "MEDIA_LIMIT_REACHED"
)

//go:generate mockgen -source=handler.go -destination=mock/handler.go -package=mock

// Service is the service the Handler uses. FetchByID is not called by the
// Handler; it is part of Service so Module.Service can be wired into reports.
type Service interface {
	Create(ctx context.Context, r *domain.Review) (*domain.Review, error)
	MediaUploadURL(ctx context.Context, placeID, userID int64, contentType string) (storage.Upload, error)
	AddMedia(ctx context.Context, placeID, userID int64, uploadKey string) (*domain.Review, error)
	ListByPlace(ctx context.Context, placeID int64) ([]domain.Review, error)
	FetchByID(ctx context.Context, placeID, reviewID int64) (*domain.Review, error)
	DeleteMedia(ctx context.Context, placeID, userID, mediaID int64) error
	Delete(ctx context.Context, placeID, reviewID int64) error
	DeleteMine(ctx context.Context, placeID, userID int64) error
}

// Handler serves the review endpoints.
type Handler struct {
	service Service

	// mediaURL turns a stored media key into its public URL.
	mediaURL func(key string) string
}

// NewHandler returns a Handler that uses svc and builds media links with mediaURL.
func NewHandler(svc Service, mediaURL func(key string) string) *Handler {
	return &Handler{
		service:  svc,
		mediaURL: mediaURL,
	}
}

// Create saves the logged-in user's review of the place in the URL.
// The route must sit behind middleware.WithAuth, so the user is known.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	id, userID, ok := placeAndUser(w, r)
	if !ok {
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

	httpx.RespondJSON(w, http.StatusCreated, toResponse(*created, h.mediaURL))
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

	httpx.RespondJSON(w, http.StatusOK, utils.Map(reviews, func(r domain.Review) Response { return toResponse(r, h.mediaURL) }))
}

// CreateMediaUpload returns a temporary link the logged-in user uploads a
// photo of their review of the place in the URL to directly.
func (h *Handler) CreateMediaUpload(w http.ResponseWriter, r *http.Request) {
	id, userID, ok := placeAndUser(w, r)
	if !ok {
		return
	}

	req, ok := httpx.DecodeJSON[storage.MediaUploadRequest](w, r)
	if !ok {
		return
	}

	upload, err := h.service.MediaUploadURL(r.Context(), id, userID, req.ContentType)
	if err != nil {
		respondMediaError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, storage.ToMediaUploadResponse(upload))
}

// AddMedia attaches an already uploaded photo to the logged-in user's review
// of the place in the URL and returns the review.
func (h *Handler) AddMedia(w http.ResponseWriter, r *http.Request) {
	id, userID, ok := placeAndUser(w, r)
	if !ok {
		return
	}

	req, ok := httpx.DecodeJSON[storage.ConfirmUploadRequest](w, r)
	if !ok {
		return
	}

	review, err := h.service.AddMedia(r.Context(), id, userID, req.UploadKey)
	if err != nil {
		respondMediaError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toResponse(*review, h.mediaURL))
}

// DeleteMedia deletes one photo of the logged-in user's review of the place
// in the URL.
func (h *Handler) DeleteMedia(w http.ResponseWriter, r *http.Request) {
	id, userID, ok := placeAndUser(w, r)
	if !ok {
		return
	}

	mediaID, ok := httpx.PathID(w, r, "mediaId", codeInvalidMediaID, "invalid media_id")
	if !ok {
		return
	}

	if err := h.service.DeleteMedia(r.Context(), id, userID, mediaID); err != nil {
		respondMediaError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Delete removes the review in the URL from the place in the URL. The route
// must be limited to admins.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := placeID(w, r)
	if !ok {
		return
	}

	reviewID, ok := httpx.PathID(w, r, "reviewId", codeInvalidReviewID, "invalid review_id")
	if !ok {
		return
	}

	respondDeleted(w, h.service.Delete(r.Context(), id, reviewID))
}

// DeleteMine removes the logged-in user's review of the place in the URL.
// The route must sit behind middleware.WithAuth, so the user is known.
func (h *Handler) DeleteMine(w http.ResponseWriter, r *http.Request) {
	id, userID, ok := placeAndUser(w, r)
	if !ok {
		return
	}

	respondDeleted(w, h.service.DeleteMine(r.Context(), id, userID))
}

// respondDeleted answers the outcome err of a review delete.
func respondDeleted(w http.ResponseWriter, err error) {
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, domain.ErrReviewNotFound):
		httpx.RespondError(w, http.StatusNotFound, codeReviewNotFound, err.Error())
	default:
		httpx.RespondInternalError(w)
	}
}

// placeID reads the place id from the URL, or answers 400.
func placeID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	return httpx.PathID(w, r, "id", codeInvalidPlaceID, "invalid place_id")
}

// placeAndUser reads the place id from the URL and the logged-in user, or
// answers 400 or 401. The route must sit behind middleware.WithAuth.
func placeAndUser(w http.ResponseWriter, r *http.Request) (int64, int64, bool) {
	id, ok := placeID(w, r)
	if !ok {
		return 0, 0, false
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.RespondError(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "missing user")
		return 0, 0, false
	}

	return id, userID, true
}

// isValidationError reports whether err means the review's data was refused.
func isValidationError(err error) bool {
	return errors.Is(err, domain.ErrReviewRatingInvalid) || errors.Is(err, domain.ErrReviewCommentTooLong)
}

// respondMediaError answers an error from a review media operation.
func respondMediaError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrReviewNotFound):
		httpx.RespondError(w, http.StatusNotFound, codeReviewNotFound, err.Error())
	case errors.Is(err, domain.ErrMediaNotFound):
		httpx.RespondError(w, http.StatusNotFound, codeMediaNotFound, err.Error())
	case errors.Is(err, domain.ErrMediaLimitReached):
		httpx.RespondError(w, http.StatusConflict, codeMediaLimitReached, err.Error())
	case storage.IsInvalidMedia(err):
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeValidationError, err.Error())
	default:
		httpx.RespondInternalError(w)
	}
}
