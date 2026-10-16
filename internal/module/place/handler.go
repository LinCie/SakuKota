package place

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

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

const internalErrorMessage = "internal server error"

func respondInternal(w http.ResponseWriter) {
	httpx.RespondError(w, http.StatusInternalServerError, "internal_error", internalErrorMessage)
}

// GetPlaces lists places, optionally filtered by `sub_district_id`,
// `category_id` (comma-separated, matches any), and a `q` search term.
func (h *Handler) GetPlaces(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	filter := PlaceFilter{Query: strings.TrimSpace(query.Get("q"))}

	if param := query.Get("sub_district_id"); param != "" {
		id, err := strconv.ParseInt(param, 10, 64)
		if err != nil || id <= 0 {
			httpx.RespondError(w, http.StatusBadRequest, "invalid_sub_district_id", "invalid sub_district_id")
			return
		}
		filter.SubDistrictID = id
	}

	if param := query.Get("category_id"); param != "" {
		for part := range strings.SplitSeq(param, ",") {
			id, err := strconv.ParseInt(part, 10, 64)
			if err != nil || id <= 0 {
				httpx.RespondError(w, http.StatusBadRequest, "invalid_category_id", "invalid category_id")
				return
			}
			filter.CategoryIDs = append(filter.CategoryIDs, id)
		}
	}

	places, err := h.service.GetMany(r.Context(), filter)
	if err != nil {
		switch {
		case errors.Is(err, ErrSubDistrictNotFound):
			httpx.RespondError(w, http.StatusNotFound, "sub_district_not_found", err.Error())
		case errors.Is(err, ErrCategoryNotFound):
			httpx.RespondError(w, http.StatusNotFound, "category_not_found", err.Error())
		default:
			respondInternal(w)
		}
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toPlaceResponses(places))
}

func (h *Handler) GetCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.service.GetCategories(r.Context())
	if err != nil {
		respondInternal(w)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toCategoryDetailResponses(categories))
}

func (h *Handler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var req CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.RespondError(w, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	c, err := h.service.CreateCategory(r.Context(), toCategory(req))
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCategoryName),
			errors.Is(err, ErrInvalidSlug):
			httpx.RespondError(w, http.StatusBadRequest, "validation_error", err.Error())
		case errors.Is(err, ErrCategoryExists):
			httpx.RespondError(w, http.StatusConflict, "category_exists", err.Error())
		default:
			respondInternal(w)
		}
		return
	}

	httpx.RespondJSON(w, http.StatusCreated, toCategoryDetailResponse(*c))
}

// CreateCategoryField adds a custom field to the category given by the `id`
// path parameter.
func (h *Handler) CreateCategoryField(w http.ResponseWriter, r *http.Request) {
	categoryID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httpx.RespondError(w, http.StatusBadRequest, "invalid_category_id", "invalid category_id")
		return
	}

	var req CreateCategoryFieldRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.RespondError(w, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	f, err := h.service.CreateCategoryField(r.Context(), toCategoryField(categoryID, req))
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidFieldKey),
			errors.Is(err, ErrInvalidFieldLabel),
			errors.Is(err, ErrInvalidFieldSortOrder),
			errors.Is(err, ErrInvalidFieldDataType),
			errors.Is(err, ErrFieldOptionsRequired),
			errors.Is(err, ErrFieldOptionsNotAllowed):
			httpx.RespondError(w, http.StatusBadRequest, "validation_error", err.Error())
		case errors.Is(err, ErrCategoryNotFound):
			httpx.RespondError(w, http.StatusNotFound, "category_not_found", err.Error())
		case errors.Is(err, ErrCategoryFieldExists):
			httpx.RespondError(w, http.StatusConflict, "category_field_exists", err.Error())
		default:
			respondInternal(w)
		}
		return
	}

	httpx.RespondJSON(w, http.StatusCreated, toCategoryFieldResponse(*f))
}

func (h *Handler) CreatePlace(w http.ResponseWriter, r *http.Request) {
	var req CreatePlaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.RespondError(w, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	p, err := h.service.Create(r.Context(), toPlace(req))
	if err == nil && p == nil {
		err = errors.New("service returned no place")
	}
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidName),
			errors.Is(err, ErrSubDistrictNotFound),
			errors.Is(err, ErrCategoryNotFound),
			errors.Is(err, ErrInvalidAttributes),
			errors.Is(err, ErrInvalidLocation):
			httpx.RespondError(w, http.StatusBadRequest, "validation_error", err.Error())
		default:
			respondInternal(w)
		}
		return
	}

	httpx.RespondJSON(w, http.StatusCreated, toPlaceResponse(*p))
}
