package storage

import (
	"errors"
	"time"

	"sakukota-go/internal/domain"
)

// MediaUploadRequest is the API payload for requesting an image or video upload.
type MediaUploadRequest struct {
	ContentType string `json:"content_type"`
}

// MediaUploadResponse is a temporary upload link the client sends the file with.
// The client then confirms it by sending UploadKey in a ConfirmUploadRequest.
type MediaUploadResponse struct {
	UploadKey string            `json:"upload_key"`
	UploadURL string            `json:"upload_url"`
	Method    string            `json:"method"`
	Headers   map[string]string `json:"headers"`
	ExpiresAt time.Time         `json:"expires_at"`
}

// ConfirmUploadRequest is the API payload for confirming an uploaded file.
type ConfirmUploadRequest struct {
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

// ToMediaResponses turns each media into its response shape, with its key made into a URL.
// It never returns nil, so an empty list is sent as [] instead of null.
func ToMediaResponses(medias []domain.Media, mediaURL func(key string) string) []MediaResponse {
	responses := make([]MediaResponse, 0, len(medias))

	for _, m := range medias {
		responses = append(responses, MediaResponse{
			ID:          m.ID,
			URL:         mediaURL(m.Key),
			ContentType: m.ContentType,
			Size:        m.Size,
			Position:    m.Position,
		})
	}

	return responses
}

// OptionalURL returns the URL mediaURL builds for key, or nil when key is nil.
func OptionalURL(key *string, mediaURL func(key string) string) *string {
	if key == nil {
		return nil
	}
	return new(mediaURL(*key))
}

// ToMediaUploadResponse turns u into the JSON shape sent to the client.
func ToMediaUploadResponse(u Upload) MediaUploadResponse {
	return MediaUploadResponse{
		UploadKey: u.Key,
		UploadURL: u.URL,
		Method:    u.Method,
		Headers:   u.Headers,
		ExpiresAt: u.ExpiresAt,
	}
}

// IsInvalidMedia reports whether err means the client's image or video was refused.
// That covers a wrong type, a file that is too large, a missing upload, or a bad upload key.
func IsInvalidMedia(err error) bool {
	return errors.Is(err, ErrInvalidMediaType) ||
		errors.Is(err, ErrImageTooLarge) ||
		errors.Is(err, ErrVideoTooLarge) ||
		errors.Is(err, ErrMediaNotUploaded) ||
		errors.Is(err, ErrInvalidUploadKey)
}
