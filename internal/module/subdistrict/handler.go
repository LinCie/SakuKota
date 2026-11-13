package subdistrict

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"rantau-go/internal/pkg/httpx"
	"rantau-go/internal/storage"
)

type Handler struct {
	service Service
	// imageURL returns the public URL of the image stored under a key.
	imageURL func(key string) string
}

func NewHandler(service Service, imageURL func(key string) string) *Handler {
	return &Handler{
		service:  service,
		imageURL: imageURL,
	}
}

// -----------------------------------------------------------------------------
// Handler
// -----------------------------------------------------------------------------

// CreateSubDistrict stores the sub-district given in the body.
func (h *Handler) CreateSubDistrict(w http.ResponseWriter, r *http.Request) {
	req, ok := httpx.DecodeJSON[SubDistrictRequest](w, r)
	if !ok {
		return
	}

	sd, err := h.service.Create(r.Context(), toSubDistrict(*req))
	if err != nil {
		respondSubDistrictError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusCreated, toSubDistrictResponse(*sd, h.imageURL))
}

// CreateImageUpload returns a handler that presigns an upload for a new image
// in slot of the sub-district given by the `id` path parameter.
func (h *Handler) CreateImageUpload(slot ImageSlot) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := httpx.ParseID(chi.URLParam(r, "id"))
		if !ok {
			httpx.RespondError(w, http.StatusBadRequest, httpx.CodeInvalidSubDistrictID, "invalid sub_district_id")
			return
		}

		req, ok := httpx.DecodeJSON[storage.ImageUploadRequest](w, r)
		if !ok {
			return
		}

		upload, err := h.service.ImageUploadURL(r.Context(), id, slot, req.ContentType)
		if err != nil {
			respondImageError(w, err)
			return
		}

		httpx.RespondJSON(w, http.StatusOK, storage.ToImageUploadResponse(upload))
	}
}

// GetSubDistricts lists every sub-district in id order.
func (h *Handler) GetSubDistricts(w http.ResponseWriter, r *http.Request) {
	subDistricts, err := h.service.GetMany(r.Context())
	if err != nil {
		httpx.RespondInternalError(w)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toSubDistrictResponses(subDistricts, h.imageURL))
}

// GetSubDistrict returns the sub-district given by the `id` path parameter.
func (h *Handler) GetSubDistrict(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.ParseID(chi.URLParam(r, "id"))
	if !ok {
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeInvalidSubDistrictID, "invalid sub_district_id")
		return
	}

	sd, err := h.service.GetSubDistrict(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrSubDistrictNotFound) {
			httpx.RespondError(w, http.StatusNotFound, httpx.CodeSubDistrictNotFound, err.Error())
			return
		}
		httpx.RespondInternalError(w)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toSubDistrictResponse(*sd, h.imageURL))
}

// PutSubDistrict replaces the sub-district given by the `id` path parameter.
func (h *Handler) PutSubDistrict(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.ParseID(chi.URLParam(r, "id"))
	if !ok {
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeInvalidSubDistrictID, "invalid sub_district_id")
		return
	}

	req, ok := httpx.DecodeJSON[SubDistrictRequest](w, r)
	if !ok {
		return
	}

	sd, err := h.service.Update(r.Context(), id, toSubDistrict(*req))
	if err != nil {
		respondSubDistrictError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toSubDistrictResponse(*sd, h.imageURL))
}

// DeleteSubDistrict deletes the sub-district given by the `id` path parameter.
func (h *Handler) DeleteSubDistrict(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.ParseID(chi.URLParam(r, "id"))
	if !ok {
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeInvalidSubDistrictID, "invalid sub_district_id")
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		switch {
		case errors.Is(err, ErrSubDistrictNotFound):
			httpx.RespondError(w, http.StatusNotFound, httpx.CodeSubDistrictNotFound, err.Error())
		case errors.Is(err, ErrSubDistrictInUse):
			httpx.RespondError(w, http.StatusConflict, httpx.CodeSubDistrictInUse, err.Error())
		default:
			httpx.RespondInternalError(w)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AddMedia adds the uploaded image to the medias of the sub-district given by
// the `id` path parameter.
func (h *Handler) AddMedia(w http.ResponseWriter, r *http.Request) {
	h.storeImage(w, r, h.service.AddMedia)
}

// SetStaticMap makes the uploaded image the static map of the sub-district
// given by the `id` path parameter.
func (h *Handler) SetStaticMap(w http.ResponseWriter, r *http.Request) {
	h.storeImage(w, r, h.service.SetStaticMap)
}

// storeImage stores the upload in the body for the sub-district given by the
// `id` path parameter with store, and responds with the sub-district.
func (h *Handler) storeImage(w http.ResponseWriter, r *http.Request,
	store func(ctx context.Context, id int64, uploadKey string) (*SubDistrict, error),
) {
	id, ok := httpx.ParseID(chi.URLParam(r, "id"))
	if !ok {
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeInvalidSubDistrictID, "invalid sub_district_id")
		return
	}

	req, ok := httpx.DecodeJSON[storage.SetImageRequest](w, r)
	if !ok {
		return
	}

	sd, err := store(r.Context(), id, req.UploadKey)
	if err != nil {
		respondImageError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toSubDistrictResponse(*sd, h.imageURL))
}

// DeleteMedia deletes the media given by the `mediaId` path parameter from the
// sub-district given by the `id` path parameter.
func (h *Handler) DeleteMedia(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.ParseID(chi.URLParam(r, "id"))
	if !ok {
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeInvalidSubDistrictID, "invalid sub_district_id")
		return
	}

	mediaID, ok := httpx.ParseID(chi.URLParam(r, "mediaId"))
	if !ok {
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeInvalidMediaID, "invalid media_id")
		return
	}

	if err := h.service.DeleteMedia(r.Context(), id, mediaID); err != nil {
		if errors.Is(err, storage.ErrMediaNotFound) {
			httpx.RespondError(w, http.StatusNotFound, httpx.CodeMediaNotFound, err.Error())
			return
		}
		httpx.RespondInternalError(w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// respondImageError writes the response for an error from uploading a sub-district's image.
func respondImageError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrSubDistrictNotFound):
		httpx.RespondError(w, http.StatusNotFound, httpx.CodeSubDistrictNotFound, err.Error())
	case storage.IsInvalidImage(err):
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeValidationError, err.Error())
	case errors.Is(err, storage.ErrMediaLimitReached):
		httpx.RespondError(w, http.StatusConflict, httpx.CodeMediaLimitReached, err.Error())
	default:
		httpx.RespondInternalError(w)
	}
}

// respondSubDistrictError writes the response for an error from creating or updating a sub-district.
func respondSubDistrictError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrSubDistrictNotFound):
		httpx.RespondError(w, http.StatusNotFound, httpx.CodeSubDistrictNotFound, err.Error())
	case errors.Is(err, ErrInvalidName),
		errors.Is(err, ErrInvalidSlug),
		errors.Is(err, ErrInvalidEstimatedDailySpending):
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeValidationError, err.Error())
	case errors.Is(err, ErrSubDistrictExists):
		httpx.RespondError(w, http.StatusConflict, httpx.CodeSubDistrictExists, err.Error())
	default:
		httpx.RespondInternalError(w)
	}
}
