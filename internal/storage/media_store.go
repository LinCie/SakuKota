package storage

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"sakukota-go/internal/domain"
)

const (
	// MaxImageSize is the largest image, in bytes, that Promote accepts.
	MaxImageSize = 5 << 20
	// MaxVideoSize is the largest video, in bytes, that Promote accepts.
	MaxVideoSize = 50 << 20
	// UploadTTL is how long a temporary upload link stays valid.
	UploadTTL = 5 * time.Minute
)

// Errors returned when a client's upload is refused. IsInvalidMedia matches all of them.
var (
	ErrInvalidMediaType = errors.New("file must be a JPEG, PNG, or WebP image, or an MP4 or WebM video")
	ErrImageTooLarge    = errors.New("image must be 5 MB or smaller")
	ErrVideoTooLarge    = errors.New("video must be 50 MB or smaller")
	ErrMediaNotUploaded = errors.New("file has not been uploaded")
	ErrInvalidUploadKey = errors.New("invalid upload key")
)

// mediaType is what an uploaded file with a given extension must be.
type mediaType struct {
	contentType string
	maxSize     int64
	// tooLarge is the error for a file over maxSize.
	tooLarge error
}

// mediaTypes is every allowed file type, by extension.
var mediaTypes = map[string]mediaType{
	"jpg":  {"image/jpeg", MaxImageSize, ErrImageTooLarge},
	"png":  {"image/png", MaxImageSize, ErrImageTooLarge},
	"webp": {"image/webp", MaxImageSize, ErrImageTooLarge},
	"mp4":  {"video/mp4", MaxVideoSize, ErrVideoTooLarge},
	"webm": {"video/webm", MaxVideoSize, ErrVideoTooLarge},
}

// lowercaseUUID matches the name, without its extension, of an uploaded file.
var lowercaseUUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// MediaStore stores images and videos uploaded directly by clients. A file is
// uploaded under uploads/{prefix}/ and promoted to {prefix}/ once it has been checked.
type MediaStore struct {
	store         ObjectStore
	publicBaseURL string
}

// NewMediaStore builds a MediaStore that stores files in store and builds public links from publicBaseURL.
func NewMediaStore(store ObjectStore, publicBaseURL string) *MediaStore {
	return &MediaStore{
		store:         store,
		publicBaseURL: publicBaseURL,
	}
}

// PresignUpload checks contentType and returns a temporary link to upload to
// uploads/{prefix}/{uuid}.{ext}.
func (m *MediaStore) PresignUpload(ctx context.Context, prefix, contentType string) (Upload, error) {
	for ext, t := range mediaTypes {
		if t.contentType == contentType {
			return m.store.PresignPut(ctx, uploadDir(prefix)+uuid.NewString()+"."+ext, contentType)
		}
	}

	return Upload{}, ErrInvalidMediaType
}

// Promote checks that uploadKey was issued for prefix and holds an allowed file.
// It moves the file to {prefix}/{uuid}.{ext} and returns its key, type, and size.
func (m *MediaStore) Promote(ctx context.Context, prefix, uploadKey string) (domain.Media, error) {
	name, inUploadDir := strings.CutPrefix(uploadKey, uploadDir(prefix))
	id, ext, _ := strings.Cut(name, ".")
	allowed, isAllowed := mediaTypes[ext]
	if !inUploadDir || !isAllowed || !lowercaseUUID.MatchString(id) {
		return domain.Media{}, ErrInvalidUploadKey
	}

	info, err := m.store.Stat(ctx, uploadKey)
	if errors.Is(err, ErrObjectNotFound) {
		return domain.Media{}, ErrMediaNotUploaded
	}
	if err != nil {
		return domain.Media{}, err
	}

	// The upload is deleted once it is rejected or copied. Uploads expire
	// anyway, so a failed delete is ignored.
	if info.ContentType != allowed.contentType {
		_ = m.store.Delete(ctx, uploadKey)
		return domain.Media{}, ErrInvalidMediaType
	}
	if info.Size > allowed.maxSize {
		_ = m.store.Delete(ctx, uploadKey)
		return domain.Media{}, allowed.tooLarge
	}

	key := prefix + "/" + name
	if err := m.store.Copy(ctx, uploadKey, key); err != nil {
		return domain.Media{}, err
	}
	_ = m.store.Delete(ctx, uploadKey)

	return domain.Media{Key: key, ContentType: info.ContentType, Size: info.Size}, nil
}

// Delete removes the file stored under key.
func (m *MediaStore) Delete(ctx context.Context, key string) error {
	return m.store.Delete(ctx, key)
}

// PublicURL returns the URL key is served from.
func (m *MediaStore) PublicURL(key string) string {
	return strings.TrimSuffix(m.publicBaseURL, "/") + "/" + key
}

// uploadDir returns the directory that files for prefix are uploaded to.
func uploadDir(prefix string) string {
	return "uploads/" + prefix + "/"
}
