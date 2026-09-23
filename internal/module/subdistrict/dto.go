package subdistrict

// SubDistrictResponse is the API representation of a SubDistrict.
type SubDistrictResponse struct {
	ID                     int64    `json:"id"`
	Name                   string   `json:"name"`
	Slug                   string   `json:"slug"`
	FloodHistory           *string  `json:"flood_history"`
	SafetyCrimeInformation *string  `json:"safety_crime_information"`
	EstimatedDailySpending *float64 `json:"estimated_daily_spending"`
	StaticMapImageURL      *string  `json:"static_map_image_url"`
}

// -----------------------------------------------------------------------------
// Conversions
// -----------------------------------------------------------------------------

func toSubDistrictResponse(sd SubDistrict) SubDistrictResponse {
	return SubDistrictResponse(sd)
}

func toSubDistrictResponses(subDistricts []SubDistrict) []SubDistrictResponse {
	responses := make([]SubDistrictResponse, 0, len(subDistricts))

	for _, sd := range subDistricts {
		responses = append(responses, toSubDistrictResponse(sd))
	}

	return responses
}
