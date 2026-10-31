package storagecleanup

import "context"

// ImageStorage defines the storage operations required by the service.
//
//go:generate mockgen -source=storage.go -destination=mock/storage.go -package=mock
type ImageStorage interface {
	// Delete removes the object. Deleting a missing object is not an error.
	Delete(ctx context.Context, key string) error
}
