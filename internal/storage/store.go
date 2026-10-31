package storage

import (
	"context"
	"errors"
	"time"
)

// ObjectStore defines the object operations Images is built on.
//
//go:generate mockgen -source=store.go -destination=mock/store.go -package=mock
type ObjectStore interface {
	// PresignPut returns a request that uploads key with contentType until ttl passes.
	PresignPut(ctx context.Context, key, contentType string, ttl time.Duration) (PresignedUpload, error)
	// Stat returns the object's size and type, or ErrObjectNotFound.
	Stat(ctx context.Context, key string) (ObjectInfo, error)
	Copy(ctx context.Context, srcKey, dstKey string) error
	Delete(ctx context.Context, key string) error
}

// PresignedUpload is a request a client can send to upload an object without
// credentials.
type PresignedUpload struct {
	URL    string
	Method string
	// Headers must be sent with the request exactly as given.
	Headers   map[string]string
	ExpiresAt time.Time
}

// ObjectInfo describes a stored object.
type ObjectInfo struct {
	Size        int64
	ContentType string
}

var ErrObjectNotFound = errors.New("object not found")
