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

// NewHandler creates a Handler that passes its work to the service.
// imageURL turns a stored image key into a link the browser can open.
func NewHandler(service Service, imageURL func(key string) string) *Handler {
	return &Handler{
		service:  service,
		imageURL: imageURL,
	}
}

// -----------------------------------------------------------------------------
// Handler
// -----------------------------------------------------------------------------

// CreateSubDistrict reads a new sub-district from the request body and saves it.
// It sends back the saved sub-district, or an error that says what is wrong.
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

// CreateImageUpload returns a handler that gives a temporary upload link for a new image.
// The image goes in the slot (photo list or static map) of the sub-district in the URL.
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

// GetSubDistricts sends back every sub-district, in id order.
func (h *Handler) GetSubDistricts(w http.ResponseWriter, r *http.Request) {
	subDistricts, err := h.service.GetMany(r.Context())
	if err != nil {
		httpx.RespondInternalError(w)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toSubDistrictResponses(subDistricts, h.imageURL))
}

// GetSubDistrict sends back the sub-district whose id is in the URL.
// If there is no such sub-district, it sends a not-found error.
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

// PutSubDistrict replaces the sub-district whose id is in the URL with the details in the body.
// Any field left out of the body is cleared.
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

// DeleteSubDistrict removes the sub-district whose id is in the URL.
// It sends no content on success, or a not-found or conflict error.
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

// AddMedia attaches an uploaded photo to the sub-district whose id is in the URL.
// It sends back the updated sub-district.
func (h *Handler) AddMedia(w http.ResponseWriter, r *http.Request) {
	h.storeImage(w, r, h.service.AddMedia)
}

// SetStaticMap makes the uploaded image the static map of the sub-district whose id is in the URL.
// It sends back the updated sub-district.
func (h *Handler) SetStaticMap(w http.ResponseWriter, r *http.Request) {
	h.storeImage(w, r, h.service.SetStaticMap)
}

// storeImage reads the upload key from the body and passes it to store for the sub-district in the URL.
// It sends back the updated sub-district, or the matching error.
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

// DeleteMedia removes the photo mediaId from the sub-district id, both taken from the URL.
// It sends no content on success, or a not-found error if the photo is missing.
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

// respondImageError sends the right error response for a failed image upload or attach.
// Any unexpected error becomes a generic 500 response.
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

// respondSubDistrictError sends the right error response for a failed create or update.
// Any unexpected error becomes a generic 500 response.
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
