package subdistricts

import (
	"context"
	"errors"
	"net/http"

	"rantau-go/internal/domain"
	"rantau-go/internal/pkg/httpx"
	"rantau-go/internal/pkg/postgis"
	"rantau-go/internal/pkg/utils"
	"rantau-go/internal/storage"
)

// Error codes of the sub-district API.
const (
	codeInvalidSubDistrictID = "INVALID_SUB_DISTRICT_ID"
	codeSubDistrictNotFound  = "SUB_DISTRICT_NOT_FOUND"
	codeSubDistrictExists    = "SUB_DISTRICT_EXISTS"
	codeSubDistrictInUse     = "SUB_DISTRICT_IN_USE"
	codeInvalidMediaID       = "INVALID_MEDIA_ID"
	codeMediaNotFound        = "MEDIA_NOT_FOUND"
	codeMediaLimitReached    = "MEDIA_LIMIT_REACHED"
)

//go:generate mockgen -source=handler.go -destination=mock/handler.go -package=mock

// Service is the service the Handler uses.
type Service interface {
	Create(ctx context.Context, sd *domain.SubDistrict) (*domain.SubDistrict, error)
	MediaUploadURL(ctx context.Context, id int64, slot MediaSlot, contentType string) (storage.Upload, error)
	AddMedia(ctx context.Context, id int64, uploadKey string) (*domain.SubDistrict, error)
	List(ctx context.Context) ([]domain.SubDistrict, error)
	Fetch(ctx context.Context, id int64) (*domain.SubDistrict, error)
	Update(ctx context.Context, id int64, sd *domain.SubDistrict) (*domain.SubDistrict, error)
	SetBoundary(ctx context.Context, id int64, boundary *postgis.MultiPolygon) (*domain.SubDistrict, error)
	Delete(ctx context.Context, id int64) error
	DeleteMedia(ctx context.Context, id, mediaID int64) error
}

// Handler serves the sub-district HTTP endpoints.
type Handler struct {
	service Service

	// mediaURL returns the public URL of the file stored under a key.
	mediaURL func(key string) string
}

// NewHandler builds a Handler over svc. It uses mediaURL to build image links.
func NewHandler(svc Service, mediaURL func(key string) string) *Handler {
	return &Handler{
		service:  svc,
		mediaURL: mediaURL,
	}
}

// Create creates a sub-district from the request body.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	req, ok := httpx.DecodeJSON[Request](w, r)
	if !ok {
		return
	}

	sd, err := h.service.Create(r.Context(), toSubDistrict(*req))
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusCreated, toResponse(*sd, h.mediaURL))
}

// CreateMediaUpload returns a temporary link the client uploads a photo to directly.
func (h *Handler) CreateMediaUpload(w http.ResponseWriter, r *http.Request) {
	id, ok := subDistrictID(w, r)
	if !ok {
		return
	}

	req, ok := httpx.DecodeJSON[storage.MediaUploadRequest](w, r)
	if !ok {
		return
	}

	upload, err := h.service.MediaUploadURL(r.Context(), id, SlotMedias, req.ContentType)
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, storage.ToMediaUploadResponse(upload))
}

// AddMedia attaches an uploaded photo to the sub-district whose id is in the URL.
// It sends back the updated sub-district.
func (h *Handler) AddMedia(w http.ResponseWriter, r *http.Request) {
	id, ok := subDistrictID(w, r)
	if !ok {
		return
	}

	req, ok := httpx.DecodeJSON[storage.ConfirmUploadRequest](w, r)
	if !ok {
		return
	}

	sd, err := h.service.AddMedia(r.Context(), id, req.UploadKey)
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toResponse(*sd, h.mediaURL))
}

// List sends back every sub-district, in id order.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	subDistricts, err := h.service.List(r.Context())
	if err != nil {
		httpx.RespondInternalError(w)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, utils.Map(subDistricts, func(sd domain.SubDistrict) Response {
		return toResponse(sd, h.mediaURL)
	}))
}

// Show sends back the sub-district whose id is in the URL.
// If there is no such sub-district, it sends a not-found error.
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	id, ok := subDistrictID(w, r)
	if !ok {
		return
	}

	sd, err := h.service.Fetch(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toResponse(*sd, h.mediaURL))
}

// Replace replaces the sub-district whose id is in the URL with the details in the body.
// Any field left out of the body is cleared.
func (h *Handler) Replace(w http.ResponseWriter, r *http.Request) {
	id, ok := subDistrictID(w, r)
	if !ok {
		return
	}

	req, ok := httpx.DecodeJSON[Request](w, r)
	if !ok {
		return
	}

	sd, err := h.service.Update(r.Context(), id, toSubDistrict(*req))
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toResponse(*sd, h.mediaURL))
}

// SetBoundary makes the GeoJSON Polygon or MultiPolygon in the body the boundary
// of the sub-district whose id is in the URL. It sends back the updated sub-district.
func (h *Handler) SetBoundary(w http.ResponseWriter, r *http.Request) {
	id, ok := subDistrictID(w, r)
	if !ok {
		return
	}

	boundary, ok := httpx.DecodeJSON[postgis.MultiPolygon](w, r)
	if !ok {
		return
	}

	sd, err := h.service.SetBoundary(r.Context(), id, boundary)
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toResponse(*sd, h.mediaURL))
}

// Delete removes the sub-district whose id is in the URL.
// It sends no content on success, or a not-found or conflict error.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := subDistrictID(w, r)
	if !ok {
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		respondError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteMedia removes the photo mediaId from the sub-district id, both taken from the URL.
// It sends no content on success, or a not-found error if the photo is missing.
func (h *Handler) DeleteMedia(w http.ResponseWriter, r *http.Request) {
	id, ok := subDistrictID(w, r)
	if !ok {
		return
	}

	mediaID, ok := httpx.PathID(w, r, "mediaId", codeInvalidMediaID, "invalid media_id")
	if !ok {
		return
	}

	if err := h.service.DeleteMedia(r.Context(), id, mediaID); err != nil {
		respondError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// subDistrictID reads the sub-district id from the URL, or answers 400.
func subDistrictID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	return httpx.PathID(w, r, "id", codeInvalidSubDistrictID, "invalid sub_district_id")
}

// isValidationError reports whether err means the request's data was refused.
func isValidationError(err error) bool {
	return errors.Is(err, domain.ErrSubDistrictNameInvalid) ||
		errors.Is(err, domain.ErrSubDistrictSlugInvalid) ||
		errors.Is(err, domain.ErrSubDistrictSpendingInvalid) ||
		errors.Is(err, domain.ErrSubDistrictBoundaryInvalid) ||
		storage.IsInvalidMedia(err)
}

// respondError sends the right error response for a failed service call.
// Any unexpected error becomes a generic 500 response.
func respondError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrSubDistrictNotFound):
		httpx.RespondError(w, http.StatusNotFound, codeSubDistrictNotFound, err.Error())
	case errors.Is(err, domain.ErrMediaNotFound):
		httpx.RespondError(w, http.StatusNotFound, codeMediaNotFound, err.Error())
	case errors.Is(err, domain.ErrSubDistrictExists):
		httpx.RespondError(w, http.StatusConflict, codeSubDistrictExists, err.Error())
	case errors.Is(err, domain.ErrSubDistrictInUse):
		httpx.RespondError(w, http.StatusConflict, codeSubDistrictInUse, err.Error())
	case errors.Is(err, domain.ErrMediaLimitReached):
		httpx.RespondError(w, http.StatusConflict, codeMediaLimitReached, err.Error())
	case isValidationError(err):
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeValidationError, err.Error())
	default:
		httpx.RespondInternalError(w)
	}
}
