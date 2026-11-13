package subdistrict

import (
	"context"

	"rantau-go/internal/storage"
)

// ImageStorage defines the image operations required by the service.
//
//go:generate mockgen -source=storage.go -destination=mock/storage.go -package=mock
type ImageStorage interface {
	// PresignUpload returns a temporary link for uploading a file under prefix.
	PresignUpload(ctx context.Context, prefix, contentType string) (storage.Upload, error)
	// Promote moves the uploaded file under uploadKey into prefix and describes it as a photo.
	Promote(ctx context.Context, prefix, uploadKey string) (storage.Media, error)
	// Delete removes the file stored under key.
	Delete(ctx context.Context, key string) error
	// PublicURL returns the link a browser can use to open the file stored under key.
	PublicURL(key string) string
}
