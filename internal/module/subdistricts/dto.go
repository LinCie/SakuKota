package subdistricts

import (
	"rantau-go/internal/pkg/postgis"
	"rantau-go/internal/storage"
)

// SubDistrictRequest is the API payload for creating a SubDistrict or
// replacing one. An update sends every field: keys left out are cleared.
type SubDistrictRequest struct {
	Name                   string   `json:"name"`
	Slug                   string   `json:"slug"`
	FloodHistory           *string  `json:"flood_history"`
	SafetyCrimeInformation *string  `json:"safety_crime_information"`
	EstimatedDailySpending *float64 `json:"estimated_daily_spending"`
}

// SubDistrictResponse is the API representation of a SubDistrict.
type SubDistrictResponse struct {
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

// -----------------------------------------------------------------------------
// Conversions
// -----------------------------------------------------------------------------

// toSubDistrict turns the request body into a SubDistrict.
func toSubDistrict(req SubDistrictRequest) *SubDistrict {
	return &SubDistrict{
		Name:                   req.Name,
		Slug:                   req.Slug,
		FloodHistory:           req.FloodHistory,
		SafetyCrimeInformation: req.SafetyCrimeInformation,
		EstimatedDailySpending: req.EstimatedDailySpending,
	}
}

// toSubDistrictResponse turns sd into the JSON shape sent to clients.
// Media keys become full links using mediaURL.
func toSubDistrictResponse(sd SubDistrict, mediaURL func(key string) string) SubDistrictResponse {
	return SubDistrictResponse{
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
