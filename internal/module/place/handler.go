package place

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"rantau-go/internal/httpx"
)

// Service defines the operations required by the handler.
type Service interface {
	GetMany(ctx context.Context, filter PlaceFilter) ([]Place, error)
	GetCategories(ctx context.Context) ([]Category, error)
	Create(ctx context.Context, p *Place, categoryIDs []int64) (*Place, error)
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

// GetPlaces lists places, optionally filtered by `sub_district_id`,
// `category_id` (comma-separated, matches any), and a `q` search term.
func (h *Handler) GetPlaces(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	filter := PlaceFilter{Query: strings.TrimSpace(query.Get("q"))}

	if param := query.Get("sub_district_id"); param != "" {
		id, err := strconv.ParseInt(param, 10, 64)
		if err != nil {
			httpx.RespondError(w, http.StatusBadRequest, "invalid_sub_district_id", "invalid sub_district_id")
			return
		}
		filter.SubDistrictID = id
	}

	if param := query.Get("category_id"); param != "" {
		for part := range strings.SplitSeq(param, ",") {
			id, err := strconv.ParseInt(part, 10, 64)
			if err != nil {
				httpx.RespondError(w, http.StatusBadRequest, "invalid_category_id", "invalid category_id")
				return
			}
			filter.CategoryIDs = append(filter.CategoryIDs, id)
		}
	}

	places, err := h.service.GetMany(r.Context(), filter)
	if err != nil {
		httpx.RespondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toPlaceResponses(places))
}

func (h *Handler) GetCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.service.GetCategories(r.Context())
	if err != nil {
		httpx.RespondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toCategoryResponses(categories))
}

func (h *Handler) CreatePlace(w http.ResponseWriter, r *http.Request) {
	var req CreatePlaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.RespondError(w, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	p, err := h.service.Create(r.Context(), toPlace(req), req.CategoryIDs)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidName),
			errors.Is(err, ErrSubDistrictNotFound),
			errors.Is(err, ErrCategoryNotFound):
			httpx.RespondError(w, http.StatusBadRequest, "validation_error", err.Error())
		default:
			httpx.RespondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		}
		return
	}

	httpx.RespondJSON(w, http.StatusCreated, toPlaceResponse(*p))
}
