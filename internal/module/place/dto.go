package place

import (
	"encoding/json"

	"rantau-go/internal/postgis"
)

// CreatePlaceRequest is the API payload for creating a Place.
type CreatePlaceRequest struct {
	SubDistrictID       int64          `json:"sub_district_id"`
	CategoryID          int64          `json:"category_id"`
	Name                string         `json:"name"`
	Address             *string        `json:"address"`
	Location            *postgis.Point `json:"location"`
	OperatingHours      *string        `json:"operating_hours"`
	EstimatedPriceRange *string        `json:"estimated_price_range"`
	ShortDescription    *string        `json:"short_description"`
	// Attributes is a JSON object keyed by the category's field keys.
	Attributes json.RawMessage `json:"attributes"`
}

// UpdatePlaceRequest is the API payload for partially updating a Place. Keys
// left out keep their current value. Nullable fields accept null to clear them.
type UpdatePlaceRequest struct {
	SubDistrictID       Optional[int64]           `json:"sub_district_id"`
	CategoryID          Optional[int64]           `json:"category_id"`
	Name                Optional[string]          `json:"name"`
	Address             Optional[string]          `json:"address"`
	Location            Optional[postgis.Point]   `json:"location"`
	OperatingHours      Optional[string]          `json:"operating_hours"`
	EstimatedPriceRange Optional[string]          `json:"estimated_price_range"`
	ShortDescription    Optional[string]          `json:"short_description"`
	Attributes          Optional[json.RawMessage] `json:"attributes"`
}

// clearsRequiredField reports whether the client sent null for a field that
// cannot be cleared.
func (r *UpdatePlaceRequest) clearsRequiredField() bool {
	return (r.SubDistrictID.Set && r.SubDistrictID.Value == nil) ||
		(r.CategoryID.Set && r.CategoryID.Value == nil) ||
		(r.Name.Set && r.Name.Value == nil)
}

// CreateCategoryRequest is the API payload for creating a Category.
type CreateCategoryRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// CreateCategoryFieldRequest is the API payload for adding a custom field to
// a Category.
type CreateCategoryFieldRequest struct {
	Key      string   `json:"key"`
	Label    string   `json:"label"`
	DataType string   `json:"data_type"`
	Required bool     `json:"required"`
	Options  []string `json:"options"`
	// SortOrder is the 1-based position to insert the field at, moving the
	// fields from there on down. Omitted, the field goes last.
	SortOrder *int32 `json:"sort_order"`
}

// PlaceResponse is the API representation of a Place.
type PlaceResponse struct {
	ID                  int64            `json:"id"`
	SubDistrictID       int64            `json:"sub_district_id"`
	Name                string           `json:"name"`
	Address             *string          `json:"address"`
	Location            *postgis.Point   `json:"location"`
	OperatingHours      *string          `json:"operating_hours"`
	EstimatedPriceRange *string          `json:"estimated_price_range"`
	ShortDescription    *string          `json:"short_description"`
	Attributes          json.RawMessage  `json:"attributes"`
	Category            CategoryResponse `json:"category"`
}

// CategoryResponse is the API representation of the Category of a place.
type CategoryResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// CategoryDetailResponse is the API representation of a Category with the
// custom fields its places carry.
type CategoryDetailResponse struct {
	ID     int64                   `json:"id"`
	Name   string                  `json:"name"`
	Slug   string                  `json:"slug"`
	Fields []CategoryFieldResponse `json:"fields"`
}

// CategoryFieldResponse is the API representation of a CategoryField.
type CategoryFieldResponse struct {
	ID         int64    `json:"id"`
	CategoryID int64    `json:"category_id"`
	Key        string   `json:"key"`
	Label      string   `json:"label"`
	DataType   string   `json:"data_type"`
	Required   bool     `json:"required"`
	Options    []string `json:"options"`
	SortOrder  int32    `json:"sort_order"`
}

// -----------------------------------------------------------------------------
// Conversions
// -----------------------------------------------------------------------------

func toPlace(req CreatePlaceRequest) *Place {
	return &Place{
		SubDistrictID:       req.SubDistrictID,
		CategoryID:          req.CategoryID,
		Name:                req.Name,
		Address:             req.Address,
		Location:            req.Location,
		OperatingHours:      req.OperatingHours,
		EstimatedPriceRange: req.EstimatedPriceRange,
		ShortDescription:    req.ShortDescription,
		Attributes:          req.Attributes,
	}
}

func toPlacePatch(req UpdatePlaceRequest) PlacePatch {
	return PlacePatch{
		SubDistrictID:       req.SubDistrictID,
		CategoryID:          req.CategoryID,
		Name:                req.Name,
		Address:             req.Address,
		Location:            req.Location,
		OperatingHours:      req.OperatingHours,
		EstimatedPriceRange: req.EstimatedPriceRange,
		ShortDescription:    req.ShortDescription,
		Attributes:          req.Attributes,
	}
}

func toCategory(req CreateCategoryRequest) *Category {
	return &Category{
		Name: req.Name,
		Slug: req.Slug,
	}
}

func toCategoryField(categoryID int64, req CreateCategoryFieldRequest) *CategoryField {
	return &CategoryField{
		CategoryID: categoryID,
		Key:        req.Key,
		Label:      req.Label,
		DataType:   req.DataType,
		Required:   req.Required,
		Options:    req.Options,
		SortOrder:  req.SortOrder,
	}
}

func toPlaceResponse(p Place) PlaceResponse {
	return PlaceResponse{
		ID:                  p.ID,
		SubDistrictID:       p.SubDistrictID,
		Name:                p.Name,
		Address:             p.Address,
		Location:            p.Location,
		OperatingHours:      p.OperatingHours,
		EstimatedPriceRange: p.EstimatedPriceRange,
		ShortDescription:    p.ShortDescription,
		Attributes:          p.Attributes,
		Category:            toCategoryResponse(p.Category),
	}
}

func toPlaceResponses(places []Place) []PlaceResponse {
	responses := make([]PlaceResponse, 0, len(places))

	for _, p := range places {
		responses = append(responses, toPlaceResponse(p))
	}

	return responses
}

func toCategoryResponse(c Category) CategoryResponse {
	return CategoryResponse{
		ID:   c.ID,
		Name: c.Name,
		Slug: c.Slug,
	}
}

func toCategoryDetailResponse(c Category) CategoryDetailResponse {
	fields := make([]CategoryFieldResponse, 0, len(c.Fields))

	for _, f := range c.Fields {
		fields = append(fields, toCategoryFieldResponse(f))
	}

	return CategoryDetailResponse{
		ID:     c.ID,
		Name:   c.Name,
		Slug:   c.Slug,
		Fields: fields,
	}
}

func toCategoryDetailResponses(categories []Category) []CategoryDetailResponse {
	responses := make([]CategoryDetailResponse, 0, len(categories))

	for _, c := range categories {
		responses = append(responses, toCategoryDetailResponse(c))
	}

	return responses
}

func toCategoryFieldResponse(f CategoryField) CategoryFieldResponse {
	var sortOrder int32
	if f.SortOrder != nil {
		sortOrder = *f.SortOrder
	}

	return CategoryFieldResponse{
		ID:         f.ID,
		CategoryID: f.CategoryID,
		Key:        f.Key,
		Label:      f.Label,
		DataType:   f.DataType,
		Required:   f.Required,
		Options:    f.Options,
		SortOrder:  sortOrder,
	}
}
