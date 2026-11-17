package storage

import (
	"context"
	"errors"
	"time"
)

// ObjectStore defines the object operations MediaStore is built on.
//
//go:generate mockgen -source=store.go -destination=mock/store.go -package=mock
type ObjectStore interface {
	// PresignPut returns a temporary link that uploads key with contentType until UploadTTL passes.
	PresignPut(ctx context.Context, key, contentType string) (Upload, error)
	// Stat returns the object's size and type, or ErrObjectNotFound.
	Stat(ctx context.Context, key string) (ObjectInfo, error)
	// Copy copies the object at srcKey to dstKey.
	Copy(ctx context.Context, srcKey, dstKey string) error
	// Delete removes the object under key.
	Delete(ctx context.Context, key string) error
}

// Upload is a temporary link a client can use to upload an object without
// credentials, and the key the object is stored under until it is promoted.
type Upload struct {
	Key    string
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
