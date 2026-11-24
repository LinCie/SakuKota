package places

import (
	"encoding/json"

	"sakukota-go/internal/domain"
	"sakukota-go/internal/pkg/postgis"
	"sakukota-go/internal/pkg/utils"
	"sakukota-go/internal/storage"
)

// Request is the body for creating or replacing a place.
// A replace clears every field left out.
type Request struct {
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

// BulkRequest is the body for creating many places at once.
type BulkRequest struct {
	Places []Request `json:"places"`
}

// Response is the API representation of a place.
type Response struct {
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

// PageResponse is one page of places. Total counts every match, including
// those beyond the page.
type PageResponse struct {
	Places []Response `json:"places"`
	Total  int        `json:"total"`
}

// CategoryResponse is the API representation of a place's category.
type CategoryResponse struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Slug  string `json:"slug"`
	Code  string `json:"code"`
	Label string `json:"label"`
	Color string `json:"color"`
}

// CategoryDetailResponse is a category with its custom fields.
type CategoryDetailResponse struct {
	CategoryResponse
	Fields []CategoryFieldResponse `json:"fields"`
}

// CategoryFieldResponse is the API representation of a category field.
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

// toPlace turns the request body into a place.
func toPlace(req Request) *domain.Place {
	return &domain.Place{
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

// toResponse turns p into its response, with media keys turned into URLs by mediaURL.
func toResponse(p domain.Place, mediaURL func(key string) string) Response {
	return Response{
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
		Medias:              storage.ToMediaResponses(p.Medias, mediaURL),
		Category:            toCategoryResponse(p.Category),
	}
}

// ratingAverage returns the mean rating, or nil before the first rating.
func ratingAverage(count, sum int) *float64 {
	if count == 0 {
		return nil
	}

	average := float64(sum) / float64(count)
	return &average
}

// toCategoryResponse turns c into its response.
func toCategoryResponse(c domain.Category) CategoryResponse {
	return CategoryResponse{
		ID:    c.ID,
		Name:  c.Name,
		Slug:  c.Slug,
		Code:  c.Code,
		Label: c.Label,
		Color: c.Color,
	}
}

// toCategoryDetailResponse turns c and its fields into its response.
func toCategoryDetailResponse(c domain.Category) CategoryDetailResponse {
	return CategoryDetailResponse{
		CategoryResponse: toCategoryResponse(c),
		Fields:           utils.Map(c.Fields, toCategoryFieldResponse),
	}
}

// toCategoryFieldResponse turns f into its response.
func toCategoryFieldResponse(f domain.CategoryField) CategoryFieldResponse {
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
