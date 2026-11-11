package place

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"

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

func (h *Handler) CreatePlace(w http.ResponseWriter, r *http.Request) {
	req, ok := httpx.DecodeJSON[PlaceRequest](w, r)
	if !ok {
		return
	}

	p, err := h.service.Create(r.Context(), ToPlace(*req))
	if err != nil {
		respondPlaceError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusCreated, toPlaceResponse(*p, h.imageURL))
}

// CreateImageUpload returns a presigned upload for a new image of the place
// given by the `id` path parameter.
func (h *Handler) CreateImageUpload(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.ParseID(chi.URLParam(r, "id"))
	if !ok {
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeInvalidPlaceID, "invalid place_id")
		return
	}

	req, ok := httpx.DecodeJSON[storage.ImageUploadRequest](w, r)
	if !ok {
		return
	}

	upload, err := h.service.ImageUploadURL(r.Context(), id, req.ContentType)
	if err != nil {
		respondImageError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, storage.ToImageUploadResponse(upload))
}

// GetPlaces lists places, optionally filtered by `sub_district_id`,
// `category_id` (comma-separated, matches any), a `q` search term, and a
// `bbox` (west,south,east,north), and capped to the `limit` lowest ids after
// skipping the first `offset`. The response carries the total number of matches alongside the places.
func (h *Handler) GetPlaces(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	filter := PlaceFilter{Query: strings.TrimSpace(query.Get("q"))}

	if param := query.Get("sub_district_id"); param != "" {
		id, ok := httpx.ParseID(param)
		if !ok {
			httpx.RespondError(w, http.StatusBadRequest, httpx.CodeInvalidSubDistrictID, "invalid sub_district_id")
			return
		}
		filter.SubDistrictID = id
	}

	if param := query.Get("category_id"); param != "" {
		for part := range strings.SplitSeq(param, ",") {
			id, ok := httpx.ParseID(part)
			if !ok {
				httpx.RespondError(w, http.StatusBadRequest, httpx.CodeInvalidCategoryID, "invalid category_id")
				return
			}
			filter.CategoryIDs = append(filter.CategoryIDs, id)
		}
	}

	if param := query.Get("limit"); param != "" {
		limit, err := strconv.Atoi(param)
		if err != nil || limit <= 0 {
			httpx.RespondError(w, http.StatusBadRequest, httpx.CodeInvalidLimit, "invalid limit")
			return
		}
		filter.Limit = limit
	}

	if param := query.Get("offset"); param != "" {
		offset, err := strconv.Atoi(param)
		if err != nil || offset < 0 {
			httpx.RespondError(w, http.StatusBadRequest, httpx.CodeInvalidOffset, "invalid offset")
			return
		}
		filter.Offset = offset
	}

	if param := query.Get("bbox"); param != "" {
		bounds, ok := parseBoundingBox(param)
		if !ok {
			httpx.RespondError(w, http.StatusBadRequest, httpx.CodeInvalidBBox, "invalid bbox")
			return
		}
		filter.Bounds = bounds
	}

	places, total, err := h.service.GetMany(r.Context(), filter)
	if err != nil {
		switch {
		case errors.Is(err, ErrSubDistrictNotFound):
			httpx.RespondError(w, http.StatusNotFound, httpx.CodeSubDistrictNotFound, err.Error())
		case errors.Is(err, ErrCategoryNotFound):
			httpx.RespondError(w, http.StatusNotFound, httpx.CodeCategoryNotFound, err.Error())
		default:
			httpx.RespondInternalError(w)
		}
		return
	}

	httpx.RespondJSON(w, http.StatusOK, PlacePageResponse{
		Places: toPlaceResponses(places, h.imageURL),
		Total:  total,
	})
}

// GetPlace returns the place given by the `id` path parameter.
func (h *Handler) GetPlace(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.ParseID(chi.URLParam(r, "id"))
	if !ok {
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeInvalidPlaceID, "invalid place_id")
		return
	}

	p, err := h.service.GetPlace(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrPlaceNotFound) {
			httpx.RespondError(w, http.StatusNotFound, httpx.CodePlaceNotFound, err.Error())
			return
		}
		httpx.RespondInternalError(w)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toPlaceResponse(*p, h.imageURL))
}

// GetCategories lists every category with its fields.
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
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeInvalidCategoryID, "invalid category_id")
		return
	}

	c, err := h.service.GetCategory(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrCategoryNotFound) {
			httpx.RespondError(w, http.StatusNotFound, httpx.CodeCategoryNotFound, err.Error())
			return
		}
		httpx.RespondInternalError(w)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toCategoryDetailResponse(*c))
}

// PutPlace replaces the place given by the `id` path parameter.
func (h *Handler) PutPlace(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.ParseID(chi.URLParam(r, "id"))
	if !ok {
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeInvalidPlaceID, "invalid place_id")
		return
	}

	req, ok := httpx.DecodeJSON[PlaceRequest](w, r)
	if !ok {
		return
	}

	p, err := h.service.Update(r.Context(), id, ToPlace(*req))
	if err != nil {
		respondPlaceError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toPlaceResponse(*p, h.imageURL))
}

// DeletePlace deletes the place given by the `id` path parameter.
func (h *Handler) DeletePlace(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.ParseID(chi.URLParam(r, "id"))
	if !ok {
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeInvalidPlaceID, "invalid place_id")
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		if errors.Is(err, ErrPlaceNotFound) {
			httpx.RespondError(w, http.StatusNotFound, httpx.CodePlaceNotFound, err.Error())
			return
		}
		httpx.RespondInternalError(w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SetImage makes the uploaded image the image of the place given by the `id`
// path parameter.
func (h *Handler) SetImage(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.ParseID(chi.URLParam(r, "id"))
	if !ok {
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeInvalidPlaceID, "invalid place_id")
		return
	}

	req, ok := httpx.DecodeJSON[storage.SetImageRequest](w, r)
	if !ok {
		return
	}

	p, err := h.service.SetImage(r.Context(), id, req.UploadKey)
	if err != nil {
		respondImageError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toPlaceResponse(*p, h.imageURL))
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

// respondImageError writes the response for an error from uploading a place's image.
func respondImageError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrPlaceNotFound):
		httpx.RespondError(w, http.StatusNotFound, httpx.CodePlaceNotFound, err.Error())
	case storage.IsInvalidImage(err):
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeValidationError, err.Error())
	default:
		httpx.RespondInternalError(w)
	}
}

// respondPlaceError writes the response for an error from creating or updating a place.
func respondPlaceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrPlaceNotFound):
		httpx.RespondError(w, http.StatusNotFound, httpx.CodePlaceNotFound, err.Error())
	case errors.Is(err, ErrInvalidName),
		errors.Is(err, ErrNameTooLong),
		errors.Is(err, ErrAddressTooLong),
		errors.Is(err, ErrOperatingHoursTooLong),
		errors.Is(err, ErrEstimatedPriceRangeTooLong),
		errors.Is(err, ErrSubDistrictNotFound),
		errors.Is(err, ErrCategoryNotFound),
		errors.Is(err, ErrInvalidAttributes),
		errors.Is(err, ErrInvalidLocation):
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeValidationError, err.Error())
	default:
		httpx.RespondInternalError(w)
	}
}
