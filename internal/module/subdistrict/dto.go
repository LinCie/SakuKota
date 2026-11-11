package subdistrict

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
	// ImageURL and StaticMapImageURL are where the sub-district's images are
	// served, or null for an image it does not have.
	ImageURL          *string `json:"image_url"`
	StaticMapImageURL *string `json:"static_map_image_url"`
}

// -----------------------------------------------------------------------------
// Conversions
// -----------------------------------------------------------------------------

func toSubDistrict(req SubDistrictRequest) *SubDistrict {
	return &SubDistrict{
		Name:                   req.Name,
		Slug:                   req.Slug,
		FloodHistory:           req.FloodHistory,
		SafetyCrimeInformation: req.SafetyCrimeInformation,
		EstimatedDailySpending: req.EstimatedDailySpending,
	}
}

// toSubDistrictResponse converts sd, turning its image keys into URLs with imageURL.
func toSubDistrictResponse(sd SubDistrict, imageURL func(key string) string) SubDistrictResponse {
	urlOf := func(key *string) *string {
		if key == nil {
			return nil
		}
		url := imageURL(*key)
		return &url
	}

	return SubDistrictResponse{
		ID:                     sd.ID,
		Name:                   sd.Name,
		Slug:                   sd.Slug,
		FloodHistory:           sd.FloodHistory,
		SafetyCrimeInformation: sd.SafetyCrimeInformation,
		EstimatedDailySpending: sd.EstimatedDailySpending,
		ImageURL:               urlOf(sd.ImageKey),
		StaticMapImageURL:      urlOf(sd.StaticMapImageKey),
	}
}

func toSubDistrictResponses(subDistricts []SubDistrict, imageURL func(key string) string) []SubDistrictResponse {
	responses := make([]SubDistrictResponse, 0, len(subDistricts))

	for _, sd := range subDistricts {
		responses = append(responses, toSubDistrictResponse(sd, imageURL))
	}

	return responses
}
