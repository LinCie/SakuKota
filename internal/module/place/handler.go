package place

import (
	"encoding/json"
	"errors"
	"math"
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

// GetPlaces lists places, optionally filtered by `sub_district_id`,
// `category_id` (comma-separated, matches any), a `q` search term, and a
// `bbox` (west,south,east,north), and capped to the `limit` lowest ids. The
// response carries the total number of matches alongside the places.
func (h *Handler) GetPlaces(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	filter := PlaceFilter{Query: strings.TrimSpace(query.Get("q"))}

	if param := query.Get("sub_district_id"); param != "" {
		id, ok := httpx.ParseID(param)
		if !ok {
			httpx.RespondError(w, http.StatusBadRequest, "invalid_sub_district_id", "invalid sub_district_id")
			return
		}
		filter.SubDistrictID = id
	}

	if param := query.Get("category_id"); param != "" {
		for part := range strings.SplitSeq(param, ",") {
			id, ok := httpx.ParseID(part)
			if !ok {
				httpx.RespondError(w, http.StatusBadRequest, "invalid_category_id", "invalid category_id")
				return
			}
			filter.CategoryIDs = append(filter.CategoryIDs, id)
		}
	}

	if param := query.Get("limit"); param != "" {
		limit, err := strconv.Atoi(param)
		if err != nil || limit <= 0 {
			httpx.RespondError(w, http.StatusBadRequest, "invalid_limit", "invalid limit")
			return
		}
		filter.Limit = limit
	}

	if param := query.Get("bbox"); param != "" {
		bounds, ok := parseBoundingBox(param)
		if !ok {
			httpx.RespondError(w, http.StatusBadRequest, "invalid_bbox", "invalid bbox")
			return
		}
		filter.Bounds = bounds
	}

	places, total, err := h.service.GetMany(r.Context(), filter)
	if err != nil {
		switch {
		case errors.Is(err, ErrSubDistrictNotFound):
			httpx.RespondError(w, http.StatusNotFound, "sub_district_not_found", err.Error())
		case errors.Is(err, ErrCategoryNotFound):
			httpx.RespondError(w, http.StatusNotFound, "category_not_found", err.Error())
		default:
			httpx.RespondInternalError(w)
		}
		return
	}

	httpx.RespondJSON(w, http.StatusOK, PlacePageResponse{
		Places: toPlaceResponses(places),
		Total:  total,
	})
}

// parseBoundingBox parses "west,south,east,north" in degrees. It reports false
// unless there are four finite coordinates in range with west <= east and
// south <= north.
func parseBoundingBox(param string) (*BoundingBox, bool) {
	parts := strings.Split(param, ",")
	if len(parts) != 4 {
		return nil, false
	}

	var coords [4]float64
	for i, part := range parts {
		f, err := strconv.ParseFloat(part, 64)
		if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
			return nil, false
		}
		coords[i] = f
	}

	b := BoundingBox{MinLng: coords[0], MinLat: coords[1], MaxLng: coords[2], MaxLat: coords[3]}
	if b.MinLng < -180 || b.MaxLng > 180 || b.MinLat < -90 || b.MaxLat > 90 ||
		b.MinLng > b.MaxLng || b.MinLat > b.MaxLat {
		return nil, false
	}

	return &b, true
}

// GetPlace returns the place given by the `id` path parameter.
func (h *Handler) GetPlace(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.ParseID(chi.URLParam(r, "id"))
	if !ok {
		httpx.RespondError(w, http.StatusBadRequest, "invalid_place_id", "invalid place_id")
		return
	}

	p, err := h.service.GetPlace(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrPlaceNotFound) {
			httpx.RespondError(w, http.StatusNotFound, "place_not_found", err.Error())
			return
		}
		httpx.RespondInternalError(w)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toPlaceResponse(*p))
}

func (h *Handler) GetCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.service.GetCategories(r.Context())
	if err != nil {
		httpx.RespondInternalError(w)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toCategoryDetailResponses(categories))
}

// GetCategory returns the category given by the `id` path parameter.
func (h *Handler) GetCategory(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.ParseID(chi.URLParam(r, "id"))
	if !ok {
		httpx.RespondError(w, http.StatusBadRequest, "invalid_category_id", "invalid category_id")
		return
	}

	c, err := h.service.GetCategory(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrCategoryNotFound) {
			httpx.RespondError(w, http.StatusNotFound, "category_not_found", err.Error())
			return
		}
		httpx.RespondInternalError(w)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toCategoryDetailResponse(*c))
}

func (h *Handler) CreatePlace(w http.ResponseWriter, r *http.Request) {
	var req CreatePlaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.RespondError(w, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	p, err := h.service.Create(r.Context(), toPlace(req))
	if err != nil {
		respondPlaceError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusCreated, toPlaceResponse(*p))
}

// PatchPlace partially updates the place given by the `id` path parameter.
func (h *Handler) PatchPlace(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.ParseID(chi.URLParam(r, "id"))
	if !ok {
		httpx.RespondError(w, http.StatusBadRequest, "invalid_place_id", "invalid place_id")
		return
	}

	// A body of null decodes to a nil pointer and is rejected with the others.
	var req *UpdatePlaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req == nil || req.clearsRequiredField() {
		httpx.RespondError(w, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	p, err := h.service.Update(r.Context(), id, PlacePatch(*req))
	if err != nil {
		respondPlaceError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toPlaceResponse(*p))
}

// DeletePlace deletes the place given by the `id` path parameter.
func (h *Handler) DeletePlace(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.ParseID(chi.URLParam(r, "id"))
	if !ok {
		httpx.RespondError(w, http.StatusBadRequest, "invalid_place_id", "invalid place_id")
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		if errors.Is(err, ErrPlaceNotFound) {
			httpx.RespondError(w, http.StatusNotFound, "place_not_found", err.Error())
			return
		}
		httpx.RespondInternalError(w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// respondPlaceError writes the response for an error from creating or updating a place.
func respondPlaceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrPlaceNotFound):
		httpx.RespondError(w, http.StatusNotFound, "place_not_found", err.Error())
	case errors.Is(err, ErrInvalidName),
		errors.Is(err, ErrPlaceNameTooLong),
		errors.Is(err, ErrSubDistrictNotFound),
		errors.Is(err, ErrCategoryNotFound),
		errors.Is(err, ErrInvalidAttributes),
		errors.Is(err, ErrInvalidLocation):
		httpx.RespondError(w, http.StatusBadRequest, "validation_error", err.Error())
	default:
		httpx.RespondInternalError(w)
	}
}
