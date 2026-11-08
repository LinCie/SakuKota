package subdistrict

import (
	"time"

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
	// ImageURL and StaticMapImageURL are where the sub-district's images are
	// served, or null for an image it does not have.
	ImageURL          *string `json:"image_url"`
	StaticMapImageURL *string `json:"static_map_image_url"`
}

// ImageUploadRequest is the API payload for requesting an image upload.
type ImageUploadRequest struct {
	ContentType string `json:"content_type"`
}

// ImageUploadResponse is a presigned upload the client sends the image with.
// The client then confirms it by sending UploadKey to SetImage.
type ImageUploadResponse struct {
	UploadKey string            `json:"upload_key"`
	UploadURL string            `json:"upload_url"`
	Method    string            `json:"method"`
	Headers   map[string]string `json:"headers"`
	ExpiresAt time.Time         `json:"expires_at"`
}

// SetImageRequest is the API payload for confirming an uploaded image.
type SetImageRequest struct {
	UploadKey string `json:"upload_key"`
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

func toImageUploadResponse(u storage.Upload) ImageUploadResponse {
	return ImageUploadResponse{
		UploadKey: u.Key,
		UploadURL: u.URL,
		Method:    u.Method,
		Headers:   u.Headers,
		ExpiresAt: u.ExpiresAt,
	}
}
