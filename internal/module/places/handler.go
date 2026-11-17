package places

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"

	"rantau-go/internal/pkg/httpx"
	"rantau-go/internal/pkg/utils"
	"rantau-go/internal/storage"
)

type Handler struct {
	service Service

	// mediaURL turns a stored media key into its public URL.
	mediaURL func(key string) string
}

// -----------------------------------------------------------------------------
// Exported
// -----------------------------------------------------------------------------

func NewHandler(service Service, mediaURL func(key string) string) *Handler {
	return &Handler{
		service:  service,
		mediaURL: mediaURL,
	}
}

// CreatePlace creates a place from the request body.
func (h *Handler) CreatePlace(w http.ResponseWriter, r *http.Request) {
	req, ok := httpx.DecodeJSON[PlaceRequest](w, r)
	if !ok {
		return
	}

	p, err := h.service.Create(r.Context(), ToPlace(*req))
	if err != nil {
		respondWriteError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusCreated, toPlaceResponse(*p, h.mediaURL))
}

// CreateMediaUpload returns a temporary link the client uploads a photo to directly.
func (h *Handler) CreateMediaUpload(w http.ResponseWriter, r *http.Request) {
	id, ok := placeID(w, r)
	if !ok {
		return
	}

	req, ok := httpx.DecodeJSON[storage.MediaUploadRequest](w, r)
	if !ok {
		return
	}

	upload, err := h.service.MediaUploadURL(r.Context(), id, req.ContentType)
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, storage.ToMediaUploadResponse(upload))
}

// AddMedia attaches an already uploaded photo to the place and returns the place.
func (h *Handler) AddMedia(w http.ResponseWriter, r *http.Request) {
	id, ok := placeID(w, r)
	if !ok {
		return
	}

	req, ok := httpx.DecodeJSON[storage.ConfirmUploadRequest](w, r)
	if !ok {
		return
	}

	p, err := h.service.AddMedia(r.Context(), id, req.UploadKey)
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toPlaceResponse(*p, h.mediaURL))
}

// GetPlaces returns one page of places matching the URL filters, plus the total match count.
func (h *Handler) GetPlaces(w http.ResponseWriter, r *http.Request) {
	filter, ok := parseFilter(w, r)
	if !ok {
		return
	}

	places, total, err := h.service.GetMany(r.Context(), filter)
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, PlacePageResponse{
		Places: utils.Map(places, func(p Place) PlaceResponse { return toPlaceResponse(p, h.mediaURL) }),
		Total:  total,
	})
}

// GetPlace returns the place with the id in the URL.
func (h *Handler) GetPlace(w http.ResponseWriter, r *http.Request) {
	id, ok := placeID(w, r)
	if !ok {
		return
	}

	p, err := h.service.GetPlace(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toPlaceResponse(*p, h.mediaURL))
}

// GetCategories returns every category with its custom fields.
func (h *Handler) GetCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.service.GetCategories(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, utils.Map(categories, toCategoryDetailResponse))
}

// GetCategory returns the category with the id in the URL.
func (h *Handler) GetCategory(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.PathID(w, r, "id", httpx.CodeInvalidCategoryID, "invalid category_id")
	if !ok {
		return
	}

	c, err := h.service.GetCategory(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toCategoryDetailResponse(*c))
}

// PutPlace replaces the place with the request body. Fields left out are cleared.
func (h *Handler) PutPlace(w http.ResponseWriter, r *http.Request) {
	id, ok := placeID(w, r)
	if !ok {
		return
	}

	req, ok := httpx.DecodeJSON[PlaceRequest](w, r)
	if !ok {
		return
	}

	p, err := h.service.Update(r.Context(), id, ToPlace(*req))
	if err != nil {
		respondWriteError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toPlaceResponse(*p, h.mediaURL))
}

// DeletePlace deletes the place with the id in the URL.
func (h *Handler) DeletePlace(w http.ResponseWriter, r *http.Request) {
	id, ok := placeID(w, r)
	if !ok {
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		respondError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteMedia deletes one photo of the place.
func (h *Handler) DeleteMedia(w http.ResponseWriter, r *http.Request) {
	id, ok := placeID(w, r)
	if !ok {
		return
	}

	mediaID, ok := httpx.PathID(w, r, "mediaId", httpx.CodeInvalidMediaID, "invalid media_id")
	if !ok {
		return
	}

	if err := h.service.DeleteMedia(r.Context(), id, mediaID); err != nil {
		respondError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// -----------------------------------------------------------------------------
// Unexported
// -----------------------------------------------------------------------------

// placeID reads the place id from the URL, or answers 400.
func placeID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	return httpx.PathID(w, r, "id", httpx.CodeInvalidPlaceID, "invalid place_id")
}

// parseFilter reads the list filters from the query string, or answers 400.
func parseFilter(w http.ResponseWriter, r *http.Request) (PlaceFilter, bool) {
	query := r.URL.Query()
	filter := PlaceFilter{Query: strings.TrimSpace(query.Get("q"))}

	if param := query.Get("sub_district_id"); param != "" {
		id, ok := httpx.ParseID(param)
		if !ok {
			httpx.RespondError(w, http.StatusBadRequest, httpx.CodeInvalidSubDistrictID, "invalid sub_district_id")
			return filter, false
		}
		filter.SubDistrictID = id
	}

	// category_id is a comma-separated list.
	if param := query.Get("category_id"); param != "" {
		for _, part := range strings.Split(param, ",") {
			id, ok := httpx.ParseID(part)
			if !ok {
				httpx.RespondError(w, http.StatusBadRequest, httpx.CodeInvalidCategoryID, "invalid category_id")
				return filter, false
			}
			filter.CategoryIDs = append(filter.CategoryIDs, id)
		}
	}

	if param := query.Get("limit"); param != "" {
		limit, err := strconv.Atoi(param)
		if err != nil || limit <= 0 {
			httpx.RespondError(w, http.StatusBadRequest, httpx.CodeInvalidLimit, "invalid limit")
			return filter, false
		}
		filter.Limit = limit
	}

	if param := query.Get("offset"); param != "" {
		offset, err := strconv.Atoi(param)
		if err != nil || offset < 0 {
			httpx.RespondError(w, http.StatusBadRequest, httpx.CodeInvalidOffset, "invalid offset")
			return filter, false
		}
		filter.Offset = offset
	}

	if param := query.Get("bbox"); param != "" {
		bounds, ok := parseBoundingBox(param)
		if !ok {
			httpx.RespondError(w, http.StatusBadRequest, httpx.CodeInvalidBBox, "invalid bbox")
			return filter, false
		}
		filter.Bounds = bounds
	}

	return filter, true
}

// parseBoundingBox reads "west,south,east,north" in degrees.
// It fails unless the four numbers are finite, in range, and in order.
func parseBoundingBox(param string) (*BoundingBox, bool) {
	parts := strings.Split(param, ",")
	if len(parts) != 4 {
		return nil, false
	}

	var coords [4]float64
	for i, part := range parts {
		f, err := strconv.ParseFloat(part, 64)
		// ParseFloat accepts "NaN" and "Inf", which are not coordinates.
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

// respondWriteError answers a failed create or update.
// Here an unknown sub-district or category is bad input (400), not a missing resource.
func respondWriteError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrSubDistrictNotFound) || errors.Is(err, ErrCategoryNotFound) {
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeValidationError, err.Error())
		return
	}

	respondError(w, err)
}

// respondError answers a failed request. Unknown errors become a generic 500,
// so internals never reach the client.
func respondError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrPlaceNotFound):
		httpx.RespondError(w, http.StatusNotFound, httpx.CodePlaceNotFound, err.Error())
	case errors.Is(err, ErrSubDistrictNotFound):
		httpx.RespondError(w, http.StatusNotFound, httpx.CodeSubDistrictNotFound, err.Error())
	case errors.Is(err, ErrCategoryNotFound):
		httpx.RespondError(w, http.StatusNotFound, httpx.CodeCategoryNotFound, err.Error())
	case errors.Is(err, storage.ErrMediaNotFound):
		httpx.RespondError(w, http.StatusNotFound, httpx.CodeMediaNotFound, err.Error())
	case errors.Is(err, storage.ErrMediaLimitReached):
		httpx.RespondError(w, http.StatusConflict, httpx.CodeMediaLimitReached, err.Error())
	case IsValidationError(err), storage.IsInvalidMedia(err):
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeValidationError, err.Error())
	default:
		httpx.RespondInternalError(w)
	}
}
