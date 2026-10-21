package subdistrict

import "rantau-go/internal/httpx"

// CreateSubDistrictRequest is the API payload for creating a SubDistrict.
type CreateSubDistrictRequest struct {
	Name                   string   `json:"name"`
	Slug                   string   `json:"slug"`
	FloodHistory           *string  `json:"flood_history"`
	SafetyCrimeInformation *string  `json:"safety_crime_information"`
	EstimatedDailySpending *float64 `json:"estimated_daily_spending"`
	StaticMapImageURL      *string  `json:"static_map_image_url"`
}

// UpdateSubDistrictRequest is the API payload for partially updating a SubDistrict.
// Keys left out keep their current value. Nullable fields accept null to clear them.
type UpdateSubDistrictRequest struct {
	Name                   httpx.Optional[string]  `json:"name"`
	Slug                   httpx.Optional[string]  `json:"slug"`
	FloodHistory           httpx.Optional[string]  `json:"flood_history"`
	SafetyCrimeInformation httpx.Optional[string]  `json:"safety_crime_information"`
	EstimatedDailySpending httpx.Optional[float64] `json:"estimated_daily_spending"`
	StaticMapImageURL      httpx.Optional[string]  `json:"static_map_image_url"`
}

// clearsRequiredField reports whether the client sent null for a field that
// cannot be cleared.
func (r *UpdateSubDistrictRequest) clearsRequiredField() bool {
	return (r.Name.Set && r.Name.Value == nil) ||
		(r.Slug.Set && r.Slug.Value == nil)
}

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

func toSubDistrict(req CreateSubDistrictRequest) *SubDistrict {
	return &SubDistrict{
		Name:                   req.Name,
		Slug:                   req.Slug,
		FloodHistory:           req.FloodHistory,
		SafetyCrimeInformation: req.SafetyCrimeInformation,
		EstimatedDailySpending: req.EstimatedDailySpending,
		StaticMapImageURL:      req.StaticMapImageURL,
	}
}

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
