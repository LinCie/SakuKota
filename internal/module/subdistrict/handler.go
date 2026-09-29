package subdistrict

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"rantau-go/internal/httpx"
)

// Service defines the operations required by the handler.
type Service interface {
	GetMany(ctx context.Context) ([]SubDistrict, error)
	Create(ctx context.Context, sd *SubDistrict) (*SubDistrict, error)
}

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
		httpx.RespondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toSubDistrictResponses(subDistricts))
}

func (h *Handler) CreateSubDistrict(w http.ResponseWriter, r *http.Request) {
	var req CreateSubDistrictRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.RespondError(w, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	sd, err := h.service.Create(r.Context(), toSubDistrict(req))
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidName),
			errors.Is(err, ErrInvalidSlug):
			httpx.RespondError(w, http.StatusBadRequest, "validation_error", err.Error())
		case errors.Is(err, ErrSubDistrictExists):
			httpx.RespondError(w, http.StatusConflict, "sub_district_exists", err.Error())
		default:
			httpx.RespondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		}
		return
	}

	httpx.RespondJSON(w, http.StatusCreated, toSubDistrictResponse(*sd))
}
