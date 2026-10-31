package storage

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	// MaxImageSize is the largest image, in bytes, that Promote accepts.
	MaxImageSize = 5 << 20
	// UploadTTL is how long a presigned upload stays valid.
	UploadTTL = 5 * time.Minute
)

var (
	ErrInvalidImageType = errors.New("image must be JPEG, PNG, or WebP")
	ErrImageTooLarge    = errors.New("image must be 5 MB or smaller")
	ErrImageNotUploaded = errors.New("image has not been uploaded")
	ErrInvalidImageKey  = errors.New("invalid upload key")
)

// imageTypes maps the extension of each allowed image type to its content type.
var imageTypes = map[string]string{
	"jpg":  "image/jpeg",
	"png":  "image/png",
	"webp": "image/webp",
}

// uploadName matches the name of an uploaded image: a lowercase UUID and the
// extension of an allowed type.
var uploadName = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\.(jpg|png|webp)$`)

// Upload is a presigned upload of a new image, and the key it is stored under
// until it is promoted.
type Upload struct {
	Key string
	PresignedUpload
}

// Images stores images uploaded directly by clients. An image is uploaded
// under uploads/{prefix}/ and promoted to {prefix}/ once it has been checked.
type Images struct {
	store         ObjectStore
	publicBaseURL string
}

func NewImages(store ObjectStore, publicBaseURL string) *Images {
	return &Images{
		store:         store,
		publicBaseURL: publicBaseURL,
	}
}

// -----------------------------------------------------------------------------
// Images
// -----------------------------------------------------------------------------

// PresignUpload checks contentType and returns an upload to
// uploads/{prefix}/{uuid}.{ext}.
func (i *Images) PresignUpload(ctx context.Context, prefix, contentType string) (Upload, error) {
	ext, ok := extensionOf(contentType)
	if !ok {
		return Upload{}, ErrInvalidImageType
	}

	key := uploadDir(prefix) + uuid.NewString() + "." + ext
	presigned, err := i.store.PresignPut(ctx, key, contentType, UploadTTL)
	if err != nil {
		return Upload{}, err
	}

	return Upload{Key: key, PresignedUpload: presigned}, nil
}

// Promote checks that uploadKey was presigned for prefix and holds an allowed
// image, then moves it to {prefix}/{uuid}.{ext} and returns that key.
func (i *Images) Promote(ctx context.Context, prefix, uploadKey string) (string, error) {
	name, ok := strings.CutPrefix(uploadKey, uploadDir(prefix))
	match := uploadName.FindStringSubmatch(name)
	if !ok || match == nil {
		return "", ErrInvalidImageKey
	}

	info, err := i.store.Stat(ctx, uploadKey)
	if errors.Is(err, ErrObjectNotFound) {
		return "", ErrImageNotUploaded
	}
	if err != nil {
		return "", err
	}

	// The upload is deleted once it is rejected or copied. Uploads expire
	// anyway, so a failed delete is ignored.
	if err := checkImage(info, imageTypes[match[1]]); err != nil {
		_ = i.store.Delete(ctx, uploadKey)
		return "", err
	}

	key := prefix + "/" + name
	if err := i.store.Copy(ctx, uploadKey, key); err != nil {
		return "", err
	}
	_ = i.store.Delete(ctx, uploadKey)

	return key, nil
}

func (i *Images) Delete(ctx context.Context, key string) error {
	return i.store.Delete(ctx, key)
}

// PublicURL returns the URL key is served from.
func (i *Images) PublicURL(key string) string {
	return strings.TrimSuffix(i.publicBaseURL, "/") + "/" + key
}

// uploadDir returns the directory that images for prefix are uploaded to.
func uploadDir(prefix string) string {
	return "uploads/" + prefix + "/"
}

// extensionOf returns the extension of an allowed image content type.
func extensionOf(contentType string) (string, bool) {
	for ext, t := range imageTypes {
		if t == contentType {
			return ext, true
		}
	}

	return "", false
}

// checkImage returns an error unless the stored object is an image of
// contentType within MaxImageSize.
func checkImage(info ObjectInfo, contentType string) error {
	switch {
	case info.ContentType != contentType:
		return ErrInvalidImageType
	case info.Size > MaxImageSize:
		return ErrImageTooLarge
	}

	return nil
}
