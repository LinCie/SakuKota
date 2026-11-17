package storage_test

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"rantau-go/internal/storage"
	"rantau-go/internal/storage/mock"
)

const (
	publicBaseURL = "http://rantau-media.web.localhost:3902"
	uploadKey     = "uploads/places/7/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp"
	promotedKey   = "places/7/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp"
)

func newMediaStore(t *testing.T) (*storage.MediaStore, *mock.MockObjectStore) {
	t.Helper()

	store := mock.NewMockObjectStore(gomock.NewController(t))
	return storage.NewMediaStore(store, publicBaseURL), store
}

// -----------------------------------------------------------------------------
// Presigning uploads
// -----------------------------------------------------------------------------

func TestMediaStore_PresignUpload(t *testing.T) {
	tests := []struct {
		contentType string
		wantKey     *regexp.Regexp
	}{
		{contentType: "image/jpeg", wantKey: uploadKeyPattern("jpg")},
		{contentType: "image/png", wantKey: uploadKeyPattern("png")},
		{contentType: "image/webp", wantKey: uploadKeyPattern("webp")},
		{contentType: "video/mp4", wantKey: uploadKeyPattern("mp4")},
		{contentType: "video/webm", wantKey: uploadKeyPattern("webm")},
	}

	for _, tc := range tests {
		t.Run(tc.contentType, func(t *testing.T) {
			media, store := newMediaStore(t)
			presigned := storage.Upload{
				URL:       "http://localhost:3900/rantau-images/signed",
				Method:    "PUT",
				Headers:   map[string]string{"Content-Type": tc.contentType},
				ExpiresAt: time.Date(2026, 10, 31, 12, 5, 0, 0, time.UTC),
			}

			var presignedKey string
			store.EXPECT().PresignPut(gomock.Any(), gomock.Any(), tc.contentType).DoAndReturn(
				func(_ context.Context, key, _ string) (storage.Upload, error) {
					presignedKey = key
					presigned.Key = key
					return presigned, nil
				})

			got, err := media.PresignUpload(context.Background(), "places/7", tc.contentType)
			require.NoError(t, err)
			assert.Regexp(t, tc.wantKey, presignedKey)
			assert.Equal(t, presignedKey, got.Key, "returned key should be the presigned key")
			assert.Equal(t, presigned.URL, got.URL)
			assert.Equal(t, presigned.Method, got.Method)
			assert.Equal(t, tc.contentType, got.Headers["Content-Type"])
			assert.True(t, got.ExpiresAt.Equal(presigned.ExpiresAt), "ExpiresAt = %v, want %v", got.ExpiresAt, presigned.ExpiresAt)
		})
	}
}

// uploadKeyPattern matches an upload key for place 7 with a random UUID name.
func uploadKeyPattern(ext string) *regexp.Regexp {
	return regexp.MustCompile(`^uploads/places/7/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\.` + ext + `$`)
}

func TestMediaStore_PresignUpload_KeysAreUnique(t *testing.T) {
	media, store := newMediaStore(t)
	store.EXPECT().PresignPut(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, key, _ string) (storage.Upload, error) {
			return storage.Upload{Key: key}, nil
		}).Times(2)

	first, err := media.PresignUpload(context.Background(), "places/7", "image/png")
	require.NoError(t, err)
	second, err := media.PresignUpload(context.Background(), "places/7", "image/png")
	require.NoError(t, err)

	assert.NotEmpty(t, first.Key)
	assert.NotEqual(t, first.Key, second.Key)
}

func TestMediaStore_PresignUpload_RejectedType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
	}{
		{name: "empty", contentType: ""},
		{name: "GIF", contentType: "image/gif"},
		{name: "SVG", contentType: "image/svg+xml"},
		{name: "HTML", contentType: "text/html"},
		{name: "PDF", contentType: "application/pdf"},
		{name: "QuickTime video", contentType: "video/quicktime"},
		{name: "video with parameters", contentType: "video/mp4; codecs=avc1"},
		{name: "allowed type with parameters", contentType: "image/png; charset=utf-8"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The mock fails the test if anything is presigned.
			media, _ := newMediaStore(t)

			_, err := media.PresignUpload(context.Background(), "places/7", tc.contentType)
			require.ErrorIs(t, err, storage.ErrInvalidMediaType)
		})
	}
}

func TestMediaStore_PresignUpload_StoreFailure(t *testing.T) {
	boom := errors.New("connection reset")
	media, store := newMediaStore(t)
	store.EXPECT().PresignPut(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(storage.Upload{}, boom)

	_, err := media.PresignUpload(context.Background(), "places/7", "image/png")
	require.ErrorIs(t, err, boom)
}

// -----------------------------------------------------------------------------
// Promoting uploads
// -----------------------------------------------------------------------------

func TestMediaStore_Promote(t *testing.T) {
	tests := []struct {
		name      string
		size      int64
		deleteErr error
	}{
		{name: "moves the upload", size: 1024, deleteErr: nil},
		{name: "largest image", size: storage.MaxImageSize, deleteErr: nil},
		{name: "upload cleanup failure is ignored", size: 1024, deleteErr: errors.New("connection reset")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			media, store := newMediaStore(t)
			gomock.InOrder(
				store.EXPECT().Stat(gomock.Any(), uploadKey).
					Return(storage.ObjectInfo{Size: tc.size, ContentType: "image/webp"}, nil),
				store.EXPECT().Copy(gomock.Any(), uploadKey, promotedKey).Return(nil),
				store.EXPECT().Delete(gomock.Any(), uploadKey).Return(tc.deleteErr),
			)

			got, err := media.Promote(context.Background(), "places/7", uploadKey)
			require.NoError(t, err)
			assert.Equal(t, storage.Media{Key: promotedKey, ContentType: "image/webp", Size: tc.size}, got,
				"the promoted media should carry the new key and the stored type and size")
		})
	}
}

func TestMediaStore_Promote_Video(t *testing.T) {
	tests := []struct {
		ext         string
		contentType string
	}{
		{ext: "mp4", contentType: "video/mp4"},
		{ext: "webm", contentType: "video/webm"},
	}

	for _, tc := range tests {
		t.Run(tc.ext, func(t *testing.T) {
			const size = storage.MaxVideoSize // larger than an image may be
			src := "uploads/places/7/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b." + tc.ext
			dst := "places/7/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b." + tc.ext

			media, store := newMediaStore(t)
			gomock.InOrder(
				store.EXPECT().Stat(gomock.Any(), src).
					Return(storage.ObjectInfo{Size: size, ContentType: tc.contentType}, nil),
				store.EXPECT().Copy(gomock.Any(), src, dst).Return(nil),
				store.EXPECT().Delete(gomock.Any(), src).Return(nil),
			)

			got, err := media.Promote(context.Background(), "places/7", src)
			require.NoError(t, err)
			assert.Equal(t, storage.Media{Key: dst, ContentType: tc.contentType, Size: size}, got)
		})
	}
}

func TestMediaStore_Promote_InvalidKey(t *testing.T) {
	const id = "0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b"

	tests := []struct {
		name string
		key  string
	}{
		{name: "empty", key: ""},
		{name: "another place", key: "uploads/places/8/" + id + ".webp"},
		{name: "place sharing the id prefix", key: "uploads/places/70/" + id + ".webp"},
		{name: "already promoted", key: "places/7/" + id + ".webp"},
		{name: "nested directory", key: "uploads/places/7/x/" + id + ".webp"},
		{name: "path traversal", key: "uploads/places/7/../8/" + id + ".webp"},
		{name: "name is not a UUID", key: "uploads/places/7/photo.webp"},
		{name: "uppercase UUID", key: "uploads/places/7/0B1E6D4C-3F2A-4B8E-9C7D-5A6F1E2D3C4B.webp"},
		{name: "extension not allowed", key: "uploads/places/7/" + id + ".gif"},
		{name: "video extension not allowed", key: "uploads/places/7/" + id + ".mov"},
		{name: "no extension", key: "uploads/places/7/" + id},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The mock fails the test if the store is touched.
			media, _ := newMediaStore(t)

			_, err := media.Promote(context.Background(), "places/7", tc.key)
			require.ErrorIs(t, err, storage.ErrInvalidUploadKey)
		})
	}
}

func TestMediaStore_Promote_NotUploaded(t *testing.T) {
	media, store := newMediaStore(t)
	store.EXPECT().Stat(gomock.Any(), uploadKey).Return(storage.ObjectInfo{}, storage.ErrObjectNotFound)

	_, err := media.Promote(context.Background(), "places/7", uploadKey)
	require.ErrorIs(t, err, storage.ErrMediaNotUploaded)
}

func TestMediaStore_Promote_RejectedImage(t *testing.T) {
	tests := []struct {
		name    string
		info    storage.ObjectInfo
		wantErr error
	}{
		{name: "too large", info: storage.ObjectInfo{Size: storage.MaxImageSize + 1, ContentType: "image/webp"}, wantErr: storage.ErrImageTooLarge},
		{name: "type not allowed", info: storage.ObjectInfo{Size: 1024, ContentType: "text/html"}, wantErr: storage.ErrInvalidMediaType},
		{name: "type does not match the extension", info: storage.ObjectInfo{Size: 1024, ContentType: "image/png"}, wantErr: storage.ErrInvalidMediaType},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			media, store := newMediaStore(t)
			store.EXPECT().Stat(gomock.Any(), uploadKey).Return(tc.info, nil)
			store.EXPECT().Delete(gomock.Any(), uploadKey).Return(nil)

			_, err := media.Promote(context.Background(), "places/7", uploadKey)
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestMediaStore_Promote_RejectedVideo(t *testing.T) {
	const videoKey = "uploads/places/7/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.mp4"

	tests := []struct {
		name    string
		info    storage.ObjectInfo
		wantErr error
	}{
		{name: "too large", info: storage.ObjectInfo{Size: storage.MaxVideoSize + 1, ContentType: "video/mp4"}, wantErr: storage.ErrVideoTooLarge},
		{name: "type does not match the extension", info: storage.ObjectInfo{Size: 1024, ContentType: "video/webm"}, wantErr: storage.ErrInvalidMediaType},
		{name: "image type for a video extension", info: storage.ObjectInfo{Size: 1024, ContentType: "image/png"}, wantErr: storage.ErrInvalidMediaType},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			media, store := newMediaStore(t)
			store.EXPECT().Stat(gomock.Any(), videoKey).Return(tc.info, nil)
			store.EXPECT().Delete(gomock.Any(), videoKey).Return(nil)

			_, err := media.Promote(context.Background(), "places/7", videoKey)
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestMediaStore_Promote_VideoContentTypeForImageExtension(t *testing.T) {
	media, store := newMediaStore(t)
	store.EXPECT().Stat(gomock.Any(), uploadKey).
		Return(storage.ObjectInfo{Size: 1024, ContentType: "video/mp4"}, nil)
	store.EXPECT().Delete(gomock.Any(), uploadKey).Return(nil)

	_, err := media.Promote(context.Background(), "places/7", uploadKey)
	require.ErrorIs(t, err, storage.ErrInvalidMediaType)
}

func TestMediaStore_Promote_RejectedImageCleanupFailure(t *testing.T) {
	media, store := newMediaStore(t)
	store.EXPECT().Stat(gomock.Any(), uploadKey).
		Return(storage.ObjectInfo{Size: storage.MaxImageSize + 1, ContentType: "image/webp"}, nil)
	store.EXPECT().Delete(gomock.Any(), uploadKey).Return(errors.New("connection reset"))

	_, err := media.Promote(context.Background(), "places/7", uploadKey)
	require.ErrorIs(t, err, storage.ErrImageTooLarge)
}

func TestMediaStore_Promote_StoreFailure(t *testing.T) {
	boom := errors.New("connection reset")

	t.Run("stat", func(t *testing.T) {
		media, store := newMediaStore(t)
		store.EXPECT().Stat(gomock.Any(), uploadKey).Return(storage.ObjectInfo{}, boom)

		_, err := media.Promote(context.Background(), "places/7", uploadKey)
		require.ErrorIs(t, err, boom)
	})

	t.Run("copy", func(t *testing.T) {
		media, store := newMediaStore(t)
		store.EXPECT().Stat(gomock.Any(), uploadKey).
			Return(storage.ObjectInfo{Size: 1024, ContentType: "image/webp"}, nil)
		store.EXPECT().Copy(gomock.Any(), uploadKey, promotedKey).Return(boom)

		_, err := media.Promote(context.Background(), "places/7", uploadKey)
		require.ErrorIs(t, err, boom)
	})
}

// -----------------------------------------------------------------------------
// Deleting and serving images
// -----------------------------------------------------------------------------

func TestMediaStore_Delete(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name     string
		storeErr error
		wantErr  error
	}{
		{name: "deletes the object", storeErr: nil, wantErr: nil},
		{name: "store failure is returned", storeErr: boom, wantErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			media, store := newMediaStore(t)
			store.EXPECT().Delete(gomock.Any(), promotedKey).Return(tc.storeErr)

			err := media.Delete(context.Background(), promotedKey)
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestMediaStore_PublicURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		want    string
	}{
		{name: "joins the base URL and the key", baseURL: publicBaseURL, want: publicBaseURL + "/" + promotedKey},
		{name: "base URL with a trailing slash", baseURL: publicBaseURL + "/", want: publicBaseURL + "/" + promotedKey},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := mock.NewMockObjectStore(gomock.NewController(t))

			got := storage.NewMediaStore(store, tc.baseURL).PublicURL(promotedKey)
			assert.Equal(t, tc.want, got)
		})
	}
}
