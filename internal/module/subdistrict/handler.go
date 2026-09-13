package subdistrict

import (
	"net/http"

	"rantau-go/internal/httpx"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// -----------------------------------------------------------------------------
// Handler
// -----------------------------------------------------------------------------

func (h *Handler) GetMany(w http.ResponseWriter, r *http.Request) {
	subDistricts, err := h.service.GetMany(r.Context())
	if err != nil {
		httpx.RespondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	httpx.RespondJSON(w, http.StatusOK, subDistricts)
}
