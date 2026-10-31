package subdistrict

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"rantau-go/internal/httpx"
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

func (h *Handler) GetSubDistricts(w http.ResponseWriter, r *http.Request) {
	subDistricts, err := h.service.GetMany(r.Context())
	if err != nil {
		httpx.RespondInternalError(w)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toSubDistrictResponses(subDistricts))
}

// GetSubDistrict returns the sub-district given by the `id` path parameter.
func (h *Handler) GetSubDistrict(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.ParseID(chi.URLParam(r, "id"))
	if !ok {
		httpx.RespondError(w, http.StatusBadRequest, CodeInvalidSubDistrictID, "invalid sub_district_id")
		return
	}

	sd, err := h.service.GetSubDistrict(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrSubDistrictNotFound) {
			httpx.RespondError(w, http.StatusNotFound, CodeSubDistrictNotFound, err.Error())
			return
		}
		httpx.RespondInternalError(w)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toSubDistrictResponse(*sd))
}

func (h *Handler) CreateSubDistrict(w http.ResponseWriter, r *http.Request) {
	var req CreateSubDistrictRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeInvalidBody, "invalid request body")
		return
	}

	sd, err := h.service.Create(r.Context(), toSubDistrict(req))
	if err != nil {
		respondSubDistrictError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusCreated, toSubDistrictResponse(*sd))
}

// PatchSubDistrict partially updates the sub-district given by the `id` path parameter.
func (h *Handler) PatchSubDistrict(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.ParseID(chi.URLParam(r, "id"))
	if !ok {
		httpx.RespondError(w, http.StatusBadRequest, CodeInvalidSubDistrictID, "invalid sub_district_id")
		return
	}

	// A body of null decodes to a nil pointer and is rejected with the others.
	var req *UpdateSubDistrictRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req == nil || req.clearsRequiredField() {
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeInvalidBody, "invalid request body")
		return
	}

	sd, err := h.service.Update(r.Context(), id, SubDistrictPatch(*req))
	if err != nil {
		respondSubDistrictError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toSubDistrictResponse(*sd))
}

// DeleteSubDistrict deletes the sub-district given by the `id` path parameter.
func (h *Handler) DeleteSubDistrict(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.ParseID(chi.URLParam(r, "id"))
	if !ok {
		httpx.RespondError(w, http.StatusBadRequest, CodeInvalidSubDistrictID, "invalid sub_district_id")
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		switch {
		case errors.Is(err, ErrSubDistrictNotFound):
			httpx.RespondError(w, http.StatusNotFound, CodeSubDistrictNotFound, err.Error())
		case errors.Is(err, ErrSubDistrictInUse):
			httpx.RespondError(w, http.StatusConflict, CodeSubDistrictInUse, err.Error())
		default:
			httpx.RespondInternalError(w)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// respondSubDistrictError writes the response for an error from creating or updating a sub-district.
func respondSubDistrictError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrSubDistrictNotFound):
		httpx.RespondError(w, http.StatusNotFound, CodeSubDistrictNotFound, err.Error())
	case errors.Is(err, ErrInvalidName),
		errors.Is(err, ErrInvalidSlug),
		errors.Is(err, ErrInvalidEstimatedDailySpending),
		errors.Is(err, ErrInvalidStaticMapImageURL):
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeValidationError, err.Error())
	case errors.Is(err, ErrSubDistrictExists):
		httpx.RespondError(w, http.StatusConflict, CodeSubDistrictExists, err.Error())
	default:
		httpx.RespondInternalError(w)
	}
}
