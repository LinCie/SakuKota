package subdistricts

import (
	"sakukota-go/internal/domain"
	"sakukota-go/internal/pkg/postgis"
	"sakukota-go/internal/storage"
)

// Request is the API payload for creating a sub-district or replacing one. An
// update sends every field: keys left out are cleared.
type Request struct {
	Name                   string   `json:"name"`
	Slug                   string   `json:"slug"`
	FloodHistory           *string  `json:"flood_history"`
	SafetyCrimeInformation *string  `json:"safety_crime_information"`
	EstimatedDailySpending *float64 `json:"estimated_daily_spending"`
}

// Response is the API representation of a sub-district.
type Response struct {
	ID                     int64    `json:"id"`
	Name                   string   `json:"name"`
	Slug                   string   `json:"slug"`
	FloodHistory           *string  `json:"flood_history"`
	SafetyCrimeInformation *string  `json:"safety_crime_information"`
	EstimatedDailySpending *float64 `json:"estimated_daily_spending"`
	// Medias are the sub-district's images in position order, or [] if it
	// has none.
	Medias []storage.MediaResponse `json:"medias"`
	// Boundary is the sub-district's area as a GeoJSON MultiPolygon, or null
	// if it has none.
	Boundary *postgis.MultiPolygon `json:"boundary"`
}

// toSubDistrict turns the request body into a sub-district.
func toSubDistrict(req Request) *domain.SubDistrict {
	return &domain.SubDistrict{
		Name:                   req.Name,
		Slug:                   req.Slug,
		FloodHistory:           req.FloodHistory,
		SafetyCrimeInformation: req.SafetyCrimeInformation,
		EstimatedDailySpending: req.EstimatedDailySpending,
	}
}

// toResponse turns sd into the JSON shape sent to clients.
// Media keys become full links using mediaURL.
func toResponse(sd domain.SubDistrict, mediaURL func(key string) string) Response {
	return Response{
		ID:                     sd.ID,
		Name:                   sd.Name,
		Slug:                   sd.Slug,
		FloodHistory:           sd.FloodHistory,
		SafetyCrimeInformation: sd.SafetyCrimeInformation,
		EstimatedDailySpending: sd.EstimatedDailySpending,
		Medias:                 storage.ToMediaResponses(sd.Medias, mediaURL),
		Boundary:               sd.Boundary,
	}
}
