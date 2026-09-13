package place

import (
	"net/http"
	"strconv"

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
	if categoryParam := r.URL.Query().Get("category_id"); categoryParam != "" {
		categoryID, err := strconv.ParseInt(categoryParam, 10, 64)
		if err != nil {
			httpx.RespondError(w, http.StatusBadRequest, "invalid_category_id", "invalid category_id")
			return
		}

		places, err := h.service.GetManyByCategory(r.Context(), categoryID)
		if err != nil {
			httpx.RespondError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}

		httpx.RespondJSON(w, http.StatusOK, places)
		return
	}

	places, err := h.service.GetMany(r.Context())
	if err != nil {
		httpx.RespondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	httpx.RespondJSON(w, http.StatusOK, places)
}

func (h *Handler) GetCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.service.GetCategories(r.Context())
	if err != nil {
		httpx.RespondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	httpx.RespondJSON(w, http.StatusOK, categories)
}
