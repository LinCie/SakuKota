package subdistricts

import (
	"context"
	"errors"
	"net/http"

	"rantau-go/internal/pkg/httpx"
	"rantau-go/internal/pkg/postgis"
	"rantau-go/internal/pkg/utils"
	"rantau-go/internal/storage"
)

type Handler struct {
	service Service

	// mediaURL returns the public URL of the file stored under a key.
	mediaURL func(key string) string
}

// -----------------------------------------------------------------------------
// Exported
// -----------------------------------------------------------------------------

func NewHandler(service Service, mediaURL func(key string) string) *Handler {
	return &Handler{
		service:  service,
		mediaURL: mediaURL,
	}
}

// CreateSubDistrict creates a sub-district from the request body.
func (h *Handler) CreateSubDistrict(w http.ResponseWriter, r *http.Request) {
	req, ok := httpx.DecodeJSON[SubDistrictRequest](w, r)
	if !ok {
		return
	}

	sd, err := h.service.Create(r.Context(), toSubDistrict(*req))
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusCreated, toSubDistrictResponse(*sd, h.mediaURL))
}

// CreateMediaUpload returns a temporary link the client uploads a photo to directly.
func (h *Handler) CreateMediaUpload(w http.ResponseWriter, r *http.Request) {
	h.createMediaUpload(w, r, SlotMedias)
}

// AddMedia attaches an uploaded photo to the sub-district whose id is in the URL.
// It sends back the updated sub-district.
func (h *Handler) AddMedia(w http.ResponseWriter, r *http.Request) {
	h.storeMedia(w, r, h.service.AddMedia)
}

// GetSubDistricts sends back every sub-district, in id order.
func (h *Handler) GetSubDistricts(w http.ResponseWriter, r *http.Request) {
	subDistricts, err := h.service.GetMany(r.Context())
	if err != nil {
		httpx.RespondInternalError(w)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, utils.Map(subDistricts, func(sd SubDistrict) SubDistrictResponse {
		return toSubDistrictResponse(sd, h.mediaURL)
	}))
}

// GetSubDistrict sends back the sub-district whose id is in the URL.
// If there is no such sub-district, it sends a not-found error.
func (h *Handler) GetSubDistrict(w http.ResponseWriter, r *http.Request) {
	id, ok := subDistrictID(w, r)
	if !ok {
		return
	}

	sd, err := h.service.GetSubDistrict(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toSubDistrictResponse(*sd, h.mediaURL))
}

// PutSubDistrict replaces the sub-district whose id is in the URL with the details in the body.
// Any field left out of the body is cleared.
func (h *Handler) PutSubDistrict(w http.ResponseWriter, r *http.Request) {
	id, ok := subDistrictID(w, r)
	if !ok {
		return
	}

	req, ok := httpx.DecodeJSON[SubDistrictRequest](w, r)
	if !ok {
		return
	}

	sd, err := h.service.Update(r.Context(), id, toSubDistrict(*req))
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toSubDistrictResponse(*sd, h.mediaURL))
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

	httpx.RespondJSON(w, http.StatusOK, toSubDistrictResponse(*sd, h.mediaURL))
}

// DeleteSubDistrict removes the sub-district whose id is in the URL.
// It sends no content on success, or a not-found or conflict error.
func (h *Handler) DeleteSubDistrict(w http.ResponseWriter, r *http.Request) {
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

	mediaID, ok := httpx.PathID(w, r, "mediaId", httpx.CodeInvalidMediaID, "invalid media_id")
	if !ok {
		return
	}

	if err := h.service.DeleteMedia(r.Context(), id, mediaID); err != nil {
		respondError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// -----------------------------------------------------------------------------
// Unexported
// -----------------------------------------------------------------------------

// createMediaUpload sends a temporary upload link for a new file in the slot
// of the sub-district in the URL.
func (h *Handler) createMediaUpload(w http.ResponseWriter, r *http.Request, slot MediaSlot) {
	id, ok := subDistrictID(w, r)
	if !ok {
		return
	}

	req, ok := httpx.DecodeJSON[storage.MediaUploadRequest](w, r)
	if !ok {
		return
	}

	upload, err := h.service.MediaUploadURL(r.Context(), id, slot, req.ContentType)
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, storage.ToMediaUploadResponse(upload))
}

// subDistrictID reads the sub-district id from the URL, or answers 400.
func subDistrictID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	return httpx.PathID(w, r, "id", httpx.CodeInvalidSubDistrictID, "invalid sub_district_id")
}

// storeMedia reads the upload key from the body and passes it to store for the sub-district in the URL.
// It sends back the updated sub-district, or the matching error.
func (h *Handler) storeMedia(w http.ResponseWriter, r *http.Request,
	store func(ctx context.Context, id int64, uploadKey string) (*SubDistrict, error),
) {
	id, ok := subDistrictID(w, r)
	if !ok {
		return
	}

	req, ok := httpx.DecodeJSON[storage.ConfirmUploadRequest](w, r)
	if !ok {
		return
	}

	sd, err := store(r.Context(), id, req.UploadKey)
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toSubDistrictResponse(*sd, h.mediaURL))
}

// respondError sends the right error response for a failed service call.
// Any unexpected error becomes a generic 500 response.
func respondError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrSubDistrictNotFound):
		httpx.RespondError(w, http.StatusNotFound, httpx.CodeSubDistrictNotFound, err.Error())
	case errors.Is(err, storage.ErrMediaNotFound):
		httpx.RespondError(w, http.StatusNotFound, httpx.CodeMediaNotFound, err.Error())
	case errors.Is(err, ErrSubDistrictExists):
		httpx.RespondError(w, http.StatusConflict, httpx.CodeSubDistrictExists, err.Error())
	case errors.Is(err, ErrSubDistrictInUse):
		httpx.RespondError(w, http.StatusConflict, httpx.CodeSubDistrictInUse, err.Error())
	case errors.Is(err, storage.ErrMediaLimitReached):
		httpx.RespondError(w, http.StatusConflict, httpx.CodeMediaLimitReached, err.Error())
	case errors.Is(err, ErrInvalidName),
		errors.Is(err, ErrInvalidSlug),
		errors.Is(err, ErrInvalidEstimatedDailySpending),
		errors.Is(err, ErrInvalidBoundary),
		storage.IsInvalidMedia(err):
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeValidationError, err.Error())
	default:
		httpx.RespondInternalError(w)
	}
}
