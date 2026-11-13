package storage

import (
	"errors"
	"time"
)

// ImageUploadRequest is the API payload for requesting an image upload.
type ImageUploadRequest struct {
	ContentType string `json:"content_type"`
}

// ImageUploadResponse is a presigned upload the client sends the image with.
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

// ToMediaResponses converts medias in order, turning each key into a URL with
// imageURL. It never returns nil, so no medias encode as [].
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

func ToImageUploadResponse(u Upload) ImageUploadResponse {
	return ImageUploadResponse{
		UploadKey: u.Key,
		UploadURL: u.URL,
		Method:    u.Method,
		Headers:   u.Headers,
		ExpiresAt: u.ExpiresAt,
	}
}

// IsInvalidImage reports whether err rejects the client's image: a disallowed
// type, a key that was not presigned, a missing upload, or an image that is too
// large.
func IsInvalidImage(err error) bool {
	return errors.Is(err, ErrInvalidImageType) ||
		errors.Is(err, ErrImageTooLarge) ||
		errors.Is(err, ErrImageNotUploaded) ||
		errors.Is(err, ErrInvalidImageKey)
}
