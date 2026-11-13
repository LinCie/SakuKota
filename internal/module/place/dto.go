package place

import (
	"encoding/json"

	"rantau-go/internal/pkg/postgis"
	"rantau-go/internal/storage"
)

// PlaceRequest is the API payload for creating a Place or replacing one. An
// update sends every field: keys left out are cleared.
type PlaceRequest struct {
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

// PlaceResponse is the API representation of a Place.
type PlaceResponse struct {
	ID                  int64           `json:"id"`
	SubDistrictID       int64           `json:"sub_district_id"`
	Name                string          `json:"name"`
	Address             *string         `json:"address"`
	Location            *postgis.Point  `json:"location"`
	OperatingHours      *string         `json:"operating_hours"`
	EstimatedPriceRange *string         `json:"estimated_price_range"`
	ShortDescription    *string         `json:"short_description"`
	Attributes          json.RawMessage `json:"attributes"`
	// RatingAverage is the mean star rating, or null before the first review.
	RatingAverage *float64 `json:"rating_average"`
	RatingCount   int      `json:"rating_count"`
	// Medias are the place's images in position order, or [] if it has none.
	Medias   []storage.MediaResponse `json:"medias"`
	Category CategoryResponse        `json:"category"`
}

// PlacePageResponse is the API representation of the places matching a
// filter. Total counts every match, including those cut off by the limit.
type PlacePageResponse struct {
	Places []PlaceResponse `json:"places"`
	Total  int             `json:"total"`
}

// CategoryResponse is the API representation of the Category of a place.
type CategoryResponse struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Slug  string `json:"slug"`
	Code  string `json:"code"`
	Label string `json:"label"`
	Color string `json:"color"`
}

// CategoryDetailResponse is the API representation of a Category with the
// custom fields its places carry.
type CategoryDetailResponse struct {
	ID     int64                   `json:"id"`
	Name   string                  `json:"name"`
	Slug   string                  `json:"slug"`
	Code   string                  `json:"code"`
	Label  string                  `json:"label"`
	Color  string                  `json:"color"`
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

// ToPlace turns the request body into a Place.
func ToPlace(req PlaceRequest) *Place {
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

// toPlaceResponse turns p into the JSON shape sent to clients.
// Photo keys become full links using imageURL.
func toPlaceResponse(p Place, imageURL func(key string) string) PlaceResponse {
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
		RatingAverage:       ratingAverage(p.RatingCount, p.RatingSum),
		RatingCount:         p.RatingCount,
		Medias:              storage.ToMediaResponses(p.Medias, imageURL),
		Category:            toCategoryResponse(p.Category),
	}
}

// ratingAverage returns the average star rating, or nil if there are no ratings yet.
func ratingAverage(count, sum int) *float64 {
	if count == 0 {
		return nil
	}

	average := float64(sum) / float64(count)
	return &average
}

// toPlaceResponses turns each place into its response shape.
func toPlaceResponses(places []Place, imageURL func(key string) string) []PlaceResponse {
	responses := make([]PlaceResponse, 0, len(places))

	for _, p := range places {
		responses = append(responses, toPlaceResponse(p, imageURL))
	}

	return responses
}

// toCategoryResponse turns c into its response shape.
func toCategoryResponse(c Category) CategoryResponse {
	return CategoryResponse{
		ID:    c.ID,
		Name:  c.Name,
		Slug:  c.Slug,
		Code:  c.Code,
		Label: c.Label,
		Color: c.Color,
	}
}

// toCategoryDetailResponse turns c and its fields into its response shape.
func toCategoryDetailResponse(c Category) CategoryDetailResponse {
	fields := make([]CategoryFieldResponse, 0, len(c.Fields))

	for _, f := range c.Fields {
		fields = append(fields, toCategoryFieldResponse(f))
	}

	return CategoryDetailResponse{
		ID:     c.ID,
		Name:   c.Name,
		Slug:   c.Slug,
		Code:   c.Code,
		Label:  c.Label,
		Color:  c.Color,
		Fields: fields,
	}
}

// toCategoryDetailResponses turns each category into its response shape.
func toCategoryDetailResponses(categories []Category) []CategoryDetailResponse {
	responses := make([]CategoryDetailResponse, 0, len(categories))

	for _, c := range categories {
		responses = append(responses, toCategoryDetailResponse(c))
	}

	return responses
}

// toCategoryFieldResponse turns f into its response shape.
func toCategoryFieldResponse(f CategoryField) CategoryFieldResponse {
	return CategoryFieldResponse{
		ID:         f.ID,
		CategoryID: f.CategoryID,
		Key:        f.Key,
		Label:      f.Label,
		DataType:   f.DataType,
		Required:   f.Required,
		Options:    f.Options,
		SortOrder:  f.SortOrder,
	}
}
