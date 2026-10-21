package place

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
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
// `category_id` (comma-separated, matches any), and a `q` search term.
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

	places, err := h.service.GetMany(r.Context(), filter)
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

	httpx.RespondJSON(w, http.StatusOK, toPlaceResponses(places))
}

func (h *Handler) GetCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.service.GetCategories(r.Context())
	if err != nil {
		httpx.RespondInternalError(w)
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
			httpx.RespondInternalError(w)
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
			httpx.RespondInternalError(w)
		}
		return
	}

	httpx.RespondJSON(w, http.StatusCreated, toCategoryFieldResponse(*f))
}

func (h *Handler) CreatePlace(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		httpx.RespondError(w, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	var req CreatePlaceRequest
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&req); err != nil {
		httpx.RespondError(w, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	in := toPlace(req)
	if key := r.Header.Get("Idempotency-Key"); key != "" {
		// The hash tells a retry of the same request from a reuse of the key.
		sum := sha256.Sum256(body)
		in.IdempotencyKey = key
		in.RequestHash = hex.EncodeToString(sum[:])
	}

	p, err := h.service.Create(r.Context(), in)
	if err == nil && p == nil {
		err = errors.New("service returned no place")
	}
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
	if err == nil && p == nil {
		err = errors.New("service returned no place")
	}
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
	case errors.Is(err, ErrIdempotencyKeyReused):
		httpx.RespondError(w, http.StatusConflict, "idempotency_key_reused", err.Error())
	default:
		httpx.RespondInternalError(w)
	}
}
