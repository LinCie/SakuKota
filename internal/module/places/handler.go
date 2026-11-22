package places

import (
	"context"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"

	"rantau-go/internal/domain"
	"rantau-go/internal/pkg/httpx"
	"rantau-go/internal/pkg/utils"
	"rantau-go/internal/storage"
)

// Codes of the error responses only places sends.
const (
	codeInvalidPlaceID       = "INVALID_PLACE_ID"
	codePlaceNotFound        = "PLACE_NOT_FOUND"
	codeInvalidSubDistrictID = "INVALID_SUB_DISTRICT_ID"
	codeSubDistrictNotFound  = "SUB_DISTRICT_NOT_FOUND"
	codeInvalidCategoryID    = "INVALID_CATEGORY_ID"
	codeCategoryNotFound     = "CATEGORY_NOT_FOUND"
	codeInvalidLimit         = "INVALID_LIMIT"
	codeInvalidOffset        = "INVALID_OFFSET"
	codeInvalidBBox          = "INVALID_BBOX"
	codeInvalidMediaID       = "INVALID_MEDIA_ID"
	codeMediaNotFound        = "MEDIA_NOT_FOUND"
	codeMediaLimitReached    = "MEDIA_LIMIT_REACHED"
)

//go:generate mockgen -source=handler.go -destination=mock/handler.go -package=mock

// placeService is the service the handler calls.
type placeService interface {
	Create(ctx context.Context, p *domain.Place) (*domain.Place, error)
	MediaUploadURL(ctx context.Context, id int64, contentType string) (storage.Upload, error)
	AddMedia(ctx context.Context, id int64, uploadKey string) (*domain.Place, error)
	List(ctx context.Context, filter Filter) ([]domain.Place, int, error)
	Fetch(ctx context.Context, id int64) (*domain.Place, error)
	ListCategories(ctx context.Context) ([]domain.Category, error)
	FetchCategory(ctx context.Context, id int64) (*domain.Category, error)
	Update(ctx context.Context, id int64, p *domain.Place) (*domain.Place, error)
	Delete(ctx context.Context, id int64) error
	DeleteMedia(ctx context.Context, id, mediaID int64) error
}

// Handler serves the place and category endpoints.
type Handler struct {
	service placeService

	// mediaURL turns a stored media key into its public URL.
	mediaURL func(key string) string
}

// NewHandler returns a Handler that calls service and builds media links with mediaURL.
func NewHandler(service placeService, mediaURL func(key string) string) *Handler {
	return &Handler{
		service:  service,
		mediaURL: mediaURL,
	}
}

// Create creates a place from the request body.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	req, ok := httpx.DecodeJSON[Request](w, r)
	if !ok {
		return
	}

	p, err := h.service.Create(r.Context(), toPlace(*req))
	if err != nil {
		respondWriteError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusCreated, toResponse(*p, h.mediaURL))
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

	httpx.RespondJSON(w, http.StatusOK, toResponse(*p, h.mediaURL))
}

// List returns one page of places matching the URL filters, plus the total match count.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	filter, ok := parseFilter(w, r)
	if !ok {
		return
	}

	places, total, err := h.service.List(r.Context(), filter)
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, PageResponse{
		Places: utils.Map(places, func(p domain.Place) Response { return toResponse(p, h.mediaURL) }),
		Total:  total,
	})
}

// Show returns the place with the id in the URL.
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	id, ok := placeID(w, r)
	if !ok {
		return
	}

	p, err := h.service.Fetch(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toResponse(*p, h.mediaURL))
}

// ListCategories returns every category with its custom fields.
func (h *Handler) ListCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.service.ListCategories(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, utils.Map(categories, toCategoryDetailResponse))
}

// ShowCategory returns the category with the id in the URL.
func (h *Handler) ShowCategory(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.PathID(w, r, "id", codeInvalidCategoryID, "invalid category_id")
	if !ok {
		return
	}

	c, err := h.service.FetchCategory(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toCategoryDetailResponse(*c))
}

// Replace replaces the place with the request body. Fields left out are cleared.
func (h *Handler) Replace(w http.ResponseWriter, r *http.Request) {
	id, ok := placeID(w, r)
	if !ok {
		return
	}

	req, ok := httpx.DecodeJSON[Request](w, r)
	if !ok {
		return
	}

	p, err := h.service.Update(r.Context(), id, toPlace(*req))
	if err != nil {
		respondWriteError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toResponse(*p, h.mediaURL))
}

// Delete deletes the place with the id in the URL.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
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

	mediaID, ok := httpx.PathID(w, r, "mediaId", codeInvalidMediaID, "invalid media_id")
	if !ok {
		return
	}

	if err := h.service.DeleteMedia(r.Context(), id, mediaID); err != nil {
		respondError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// placeID reads the place id from the URL, or answers 400.
func placeID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	return httpx.PathID(w, r, "id", codeInvalidPlaceID, "invalid place_id")
}

// parseFilter reads the list filters from the query string, or answers 400.
func parseFilter(w http.ResponseWriter, r *http.Request) (Filter, bool) {
	query := r.URL.Query()
	filter := Filter{Query: strings.TrimSpace(query.Get("q"))}

	if param := query.Get("sub_district_id"); param != "" {
		id, ok := httpx.ParseID(param)
		if !ok {
			httpx.RespondError(w, http.StatusBadRequest, codeInvalidSubDistrictID, "invalid sub_district_id")
			return filter, false
		}
		filter.SubDistrictID = id
	}

	// category_id is a comma-separated list.
	if param := query.Get("category_id"); param != "" {
		for _, part := range strings.Split(param, ",") {
			id, ok := httpx.ParseID(part)
			if !ok {
				httpx.RespondError(w, http.StatusBadRequest, codeInvalidCategoryID, "invalid category_id")
				return filter, false
			}
			filter.CategoryIDs = append(filter.CategoryIDs, id)
		}
	}

	if param := query.Get("limit"); param != "" {
		limit, err := strconv.Atoi(param)
		if err != nil || limit <= 0 {
			httpx.RespondError(w, http.StatusBadRequest, codeInvalidLimit, "invalid limit")
			return filter, false
		}
		filter.Limit = limit
	}

	if param := query.Get("offset"); param != "" {
		offset, err := strconv.Atoi(param)
		if err != nil || offset < 0 {
			httpx.RespondError(w, http.StatusBadRequest, codeInvalidOffset, "invalid offset")
			return filter, false
		}
		filter.Offset = offset
	}

	if param := query.Get("bbox"); param != "" {
		bounds, ok := parseBoundingBox(param)
		if !ok {
			httpx.RespondError(w, http.StatusBadRequest, codeInvalidBBox, "invalid bbox")
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
	if b.MinLng < -180 || b.MaxLng > 180 || b.MinLat < -90 || b.MaxLat > 90 || b.MinLng > b.MaxLng || b.MinLat > b.MaxLat {
		return nil, false
	}

	return &b, true
}

// respondWriteError answers a failed create or update.
// Here an unknown sub-district or category is bad input (400), not a missing resource.
func respondWriteError(w http.ResponseWriter, err error) {
	if errors.Is(err, domain.ErrSubDistrictNotFound) || errors.Is(err, domain.ErrCategoryNotFound) {
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeValidationError, err.Error())
		return
	}

	respondError(w, err)
}

// respondError answers a failed request. Unknown errors become a generic 500,
// so internals never reach the client.
func respondError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrPlaceNotFound):
		httpx.RespondError(w, http.StatusNotFound, codePlaceNotFound, err.Error())
	case errors.Is(err, domain.ErrSubDistrictNotFound):
		httpx.RespondError(w, http.StatusNotFound, codeSubDistrictNotFound, err.Error())
	case errors.Is(err, domain.ErrCategoryNotFound):
		httpx.RespondError(w, http.StatusNotFound, codeCategoryNotFound, err.Error())
	case errors.Is(err, domain.ErrMediaNotFound):
		httpx.RespondError(w, http.StatusNotFound, codeMediaNotFound, err.Error())
	case errors.Is(err, domain.ErrMediaLimitReached):
		httpx.RespondError(w, http.StatusConflict, codeMediaLimitReached, err.Error())
	case domain.IsPlaceValidationError(err), storage.IsInvalidMedia(err):
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeValidationError, err.Error())
	default:
		httpx.RespondInternalError(w)
	}
}
