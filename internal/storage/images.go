package storage

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	// MaxImageSize is the largest image, in bytes, that Promote accepts.
	MaxImageSize = 5 << 20
	// UploadTTL is how long a temporary upload link stays valid.
	UploadTTL = 5 * time.Minute
	// MaxMedias is the most photos a place or sub-district can have. The
	// position CHECKs of PLACE_MEDIAS and SUB_DISTRICT_MEDIAS match it.
	MaxMedias = 10
)

var (
	ErrInvalidImageType = errors.New("image must be JPEG, PNG, or WebP")
	ErrImageTooLarge    = errors.New("image must be 5 MB or smaller")
	ErrImageNotUploaded = errors.New("image has not been uploaded")
	ErrInvalidImageKey  = errors.New("invalid upload key")

	ErrMediaNotFound     = errors.New("media not found")
	ErrMediaLimitReached = fmt.Errorf("at most %d medias are allowed", MaxMedias)
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

// Upload is a temporary upload link for a new image, and the key it is stored under
// until it is promoted.
type Upload struct {
	Key string
	PresignedUpload
}

// Media is a stored image: a MEDIAS row, and its position when it is linked
// to a place or sub-district.
type Media struct {
	ID          int64  `db:"id"`
	Key         string `db:"key"`
	ContentType string `db:"content_type"`
	Size        int64  `db:"size_bytes"`
	Position    int    `db:"position"`
}

// Images stores images uploaded directly by clients. An image is uploaded
// under uploads/{prefix}/ and promoted to {prefix}/ once it has been checked.
type Images struct {
	store         ObjectStore
	publicBaseURL string
}

// NewImages builds an Images that stores files in store and builds public links from publicBaseURL.
func NewImages(store ObjectStore, publicBaseURL string) *Images {
	return &Images{
		store:         store,
		publicBaseURL: publicBaseURL,
	}
}

// -----------------------------------------------------------------------------
// Images
// -----------------------------------------------------------------------------

// PresignUpload checks contentType and returns a temporary link to upload to
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

// Promote checks that uploadKey was issued for prefix and holds an allowed image.
// It moves the image to {prefix}/{uuid}.{ext} and returns its key, type, and size.
func (i *Images) Promote(ctx context.Context, prefix, uploadKey string) (Media, error) {
	name, ok := strings.CutPrefix(uploadKey, uploadDir(prefix))
	match := uploadName.FindStringSubmatch(name)
	if !ok || match == nil {
		return Media{}, ErrInvalidImageKey
	}

	info, err := i.store.Stat(ctx, uploadKey)
	if errors.Is(err, ErrObjectNotFound) {
		return Media{}, ErrImageNotUploaded
	}
	if err != nil {
		return Media{}, err
	}

	// The upload is deleted once it is rejected or copied. Uploads expire
	// anyway, so a failed delete is ignored.
	if err := checkImage(info, imageTypes[match[1]]); err != nil {
		_ = i.store.Delete(ctx, uploadKey)
		return Media{}, err
	}

	key := prefix + "/" + name
	if err := i.store.Copy(ctx, uploadKey, key); err != nil {
		return Media{}, err
	}
	_ = i.store.Delete(ctx, uploadKey)

	return Media{Key: key, ContentType: info.ContentType, Size: info.Size}, nil
}

// Delete removes the file stored under key.
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
