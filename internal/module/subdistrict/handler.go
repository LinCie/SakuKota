package subdistrict

import (
	"context"
	"net/http"

	"rantau-go/internal/httpx"
)

// Service defines the operations required by the handler.
type Service interface {
	GetMany(ctx context.Context) ([]SubDistrict, error)
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
