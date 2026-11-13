package place

import (
	"context"

	"rantau-go/internal/storage"
)

// ImageStorage defines the image operations required by the service.
//
//go:generate mockgen -source=storage.go -destination=mock/storage.go -package=mock
type ImageStorage interface {
	PresignUpload(ctx context.Context, prefix, contentType string) (storage.Upload, error)
	Promote(ctx context.Context, prefix, uploadKey string) (storage.Media, error)
	Delete(ctx context.Context, key string) error
	PublicURL(key string) string
}
