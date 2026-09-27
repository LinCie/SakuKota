package place

import "rantau-go/internal/postgis"

// PlaceResponse is the API representation of a Place.
type PlaceResponse struct {
	ID                  int64              `json:"id"`
	SubDistrictID       int64              `json:"sub_district_id"`
	Name                string             `json:"name"`
	Address             *string            `json:"address"`
	Location            *postgis.Point     `json:"location"`
	OperatingHours      *string            `json:"operating_hours"`
	EstimatedPriceRange *string            `json:"estimated_price_range"`
	ShortDescription    *string            `json:"short_description"`
	Categories          []CategoryResponse `json:"categories"`
}

// CategoryResponse is the API representation of a Category.
type CategoryResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// -----------------------------------------------------------------------------
// Conversions
// -----------------------------------------------------------------------------

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
		Categories:          toCategoryResponses(p.Categories),
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
	return CategoryResponse(c)
}

func toCategoryResponses(categories []Category) []CategoryResponse {
	responses := make([]CategoryResponse, 0, len(categories))

	for _, c := range categories {
		responses = append(responses, toCategoryResponse(c))
	}

	return responses
}
