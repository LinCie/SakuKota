package subdistricts

import "rantau-go/internal/storage"

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
	// StaticMapImageURL is where the static map is served, or null if the
	// sub-district has none.
	StaticMapImageURL *string `json:"static_map_image_url"`
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
// Image keys become full links using imageURL.
func toSubDistrictResponse(sd SubDistrict, imageURL func(key string) string) SubDistrictResponse {
	var staticMap *string
	if sd.StaticMapImageKey != nil {
		url := imageURL(*sd.StaticMapImageKey)
		staticMap = &url
	}

	return SubDistrictResponse{
		ID:                     sd.ID,
		Name:                   sd.Name,
		Slug:                   sd.Slug,
		FloodHistory:           sd.FloodHistory,
		SafetyCrimeInformation: sd.SafetyCrimeInformation,
		EstimatedDailySpending: sd.EstimatedDailySpending,
		Medias:                 storage.ToMediaResponses(sd.Medias, imageURL),
		StaticMapImageURL:      staticMap,
	}
}
