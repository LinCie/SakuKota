package storage

import (
	"errors"
	"time"
)

// ImageUploadRequest is the API payload for requesting an image upload.
type ImageUploadRequest struct {
	ContentType string `json:"content_type"`
}

// ImageUploadResponse is a temporary upload link the client sends the image with.
// The client then confirms it by sending UploadKey in a SetImageRequest.
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

// MediaResponse is the API representation of a Media.
type MediaResponse struct {
	ID          int64  `json:"id"`
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	Position    int    `json:"position"`
}

// ToMediaResponses turns each photo into its response shape, with its key made into a URL.
// It never returns nil, so an empty list is sent as [] instead of null.
func ToMediaResponses(medias []Media, imageURL func(key string) string) []MediaResponse {
	responses := make([]MediaResponse, 0, len(medias))

	for _, m := range medias {
		responses = append(responses, MediaResponse{
			ID:          m.ID,
			URL:         imageURL(m.Key),
			ContentType: m.ContentType,
			Size:        m.Size,
			Position:    m.Position,
		})
	}

	return responses
}

// ToImageUploadResponse turns u into the JSON shape sent to the client.
func ToImageUploadResponse(u Upload) ImageUploadResponse {
	return ImageUploadResponse{
		UploadKey: u.Key,
		UploadURL: u.URL,
		Method:    u.Method,
		Headers:   u.Headers,
		ExpiresAt: u.ExpiresAt,
	}
}

// IsInvalidImage reports whether err means the client's image was refused.
// That covers a wrong type, a file that is too large, a missing upload, or a bad upload key.
func IsInvalidImage(err error) bool {
	return errors.Is(err, ErrInvalidImageType) ||
		errors.Is(err, ErrImageTooLarge) ||
		errors.Is(err, ErrVideoTooLarge) ||
		errors.Is(err, ErrImageNotUploaded) ||
		errors.Is(err, ErrInvalidImageKey)
}
