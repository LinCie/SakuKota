package storage_test

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"sakukota-go/internal/domain"
	"sakukota-go/internal/storage"
	"sakukota-go/internal/storage/mock"
)

const (
	publicBaseURL = "http://sakukota-media.web.localhost:3902"
	uploadKey     = "uploads/places/7/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp"
	promotedKey   = "places/7/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp"
)

func newMediaStore(t *testing.T) (*storage.MediaStore, *mock.MockObjectStore) {
	t.Helper()

	store := mock.NewMockObjectStore(gomock.NewController(t))
	return storage.NewMediaStore(store, publicBaseURL), store
}

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
		t.Run(strings.ReplaceAll(tc.contentType, "/", "_"), func(t *testing.T) {
			media, store := newMediaStore(t)
			presigned := storage.Upload{
				URL:       "http://localhost:3900/sakukota-images/signed",
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

			got, err := media.PresignUpload(t.Context(), "places/7", tc.contentType)
			require.NoError(t, err)
			assert.Regexp(t, tc.wantKey, presignedKey)
			assert.Equal(t, presignedKey, got.Key)
			assert.Equal(t, presigned.URL, got.URL)
			assert.Equal(t, presigned.Method, got.Method)
			assert.Equal(t, tc.contentType, got.Headers["Content-Type"])
			assert.True(t, presigned.ExpiresAt.Equal(got.ExpiresAt), "ExpiresAt = %v, want %v", got.ExpiresAt, presigned.ExpiresAt)
		})
	}
}

// uploadKeyPattern matches an upload key for place 7 with a random UUID name.
func uploadKeyPattern(ext string) *regexp.Regexp {
	return regexp.MustCompile(`^uploads/places/7/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\.` + ext + `$`)
}

func TestMediaStore_PresignUpload_KeysAreUnique(t *testing.T) {
	media, store := newMediaStore(t)
	store.EXPECT().PresignPut(gomock.Any(), gomock.Any(), "image/png").DoAndReturn(
		func(_ context.Context, key, _ string) (storage.Upload, error) {
			return storage.Upload{Key: key}, nil
		}).Times(2)

	first, err := media.PresignUpload(t.Context(), "places/7", "image/png")
	require.NoError(t, err)
	second, err := media.PresignUpload(t.Context(), "places/7", "image/png")
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
		{name: "QuickTime", contentType: "video/quicktime"},
		{name: "VideoParams", contentType: "video/mp4; codecs=avc1"},
		{name: "AllowedParams", contentType: "image/png; charset=utf-8"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The mock fails the test if anything is presigned.
			media, _ := newMediaStore(t)

			_, err := media.PresignUpload(t.Context(), "places/7", tc.contentType)
			assert.ErrorIs(t, err, storage.ErrInvalidMediaType)
		})
	}
}

func TestMediaStore_PresignUpload_StoreFailure(t *testing.T) {
	boom := errors.New("connection reset")
	media, store := newMediaStore(t)
	store.EXPECT().PresignPut(gomock.Any(), gomock.Any(), "image/png").Return(storage.Upload{}, boom)

	_, err := media.PresignUpload(t.Context(), "places/7", "image/png")
	assert.ErrorIs(t, err, boom)
}

func TestMediaStore_Promote(t *testing.T) {
	tests := []struct {
		name      string
		size      int64
		deleteErr error
	}{
		{name: "MovesUpload", size: 1024, deleteErr: nil},
		{name: "LargestImage", size: storage.MaxImageSize, deleteErr: nil},
		{name: "CleanupFailureIgnored", size: 1024, deleteErr: errors.New("connection reset")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			media, store := newMediaStore(t)
			gomock.InOrder(
				store.EXPECT().Stat(gomock.Any(), uploadKey).Return(storage.ObjectInfo{Size: tc.size, ContentType: "image/webp"}, nil),
				store.EXPECT().Copy(gomock.Any(), uploadKey, promotedKey).Return(nil),
				store.EXPECT().Delete(gomock.Any(), uploadKey).Return(tc.deleteErr),
			)

			got, err := media.Promote(t.Context(), "places/7", uploadKey)
			require.NoError(t, err)

			// The promoted media carries the new key and the stored type and size.
			want := domain.Media{Key: promotedKey, ContentType: "image/webp", Size: tc.size}
			assert.Equal(t, want, got)
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
				store.EXPECT().Stat(gomock.Any(), src).Return(storage.ObjectInfo{Size: size, ContentType: tc.contentType}, nil),
				store.EXPECT().Copy(gomock.Any(), src, dst).Return(nil),
				store.EXPECT().Delete(gomock.Any(), src).Return(nil),
			)

			got, err := media.Promote(t.Context(), "places/7", src)
			require.NoError(t, err)

			want := domain.Media{Key: dst, ContentType: tc.contentType, Size: size}
			assert.Equal(t, want, got)
		})
	}
}

func TestMediaStore_Promote_InvalidKey(t *testing.T) {
	const id = "0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b"

	tests := []struct {
		name string
		key  string
	}{
		{name: "Empty", key: ""},
		{name: "AnotherPlace", key: "uploads/places/8/" + id + ".webp"},
		{name: "SharedIDPrefix", key: "uploads/places/70/" + id + ".webp"},
		{name: "AlreadyPromoted", key: "places/7/" + id + ".webp"},
		{name: "NestedDirectory", key: "uploads/places/7/x/" + id + ".webp"},
		{name: "PathTraversal", key: "uploads/places/7/../8/" + id + ".webp"},
		{name: "NotUUID", key: "uploads/places/7/photo.webp"},
		{name: "UppercaseUUID", key: "uploads/places/7/0B1E6D4C-3F2A-4B8E-9C7D-5A6F1E2D3C4B.webp"},
		{name: "ExtensionNotAllowed", key: "uploads/places/7/" + id + ".gif"},
		{name: "VideoExtensionNotAllowed", key: "uploads/places/7/" + id + ".mov"},
		{name: "NoExtension", key: "uploads/places/7/" + id},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The mock fails the test if the store is touched.
			media, _ := newMediaStore(t)

			_, err := media.Promote(t.Context(), "places/7", tc.key)
			assert.ErrorIs(t, err, storage.ErrInvalidUploadKey)
		})
	}
}

func TestMediaStore_Promote_NotUploaded(t *testing.T) {
	media, store := newMediaStore(t)
	store.EXPECT().Stat(gomock.Any(), uploadKey).Return(storage.ObjectInfo{}, storage.ErrObjectNotFound)

	_, err := media.Promote(t.Context(), "places/7", uploadKey)
	assert.ErrorIs(t, err, storage.ErrMediaNotUploaded)
}

func TestMediaStore_Promote_RejectedImage(t *testing.T) {
	tests := []struct {
		name    string
		info    storage.ObjectInfo
		wantErr error
	}{
		{name: "TooLarge", info: storage.ObjectInfo{Size: storage.MaxImageSize + 1, ContentType: "image/webp"}, wantErr: storage.ErrImageTooLarge},
		{name: "TypeNotAllowed", info: storage.ObjectInfo{Size: 1024, ContentType: "text/html"}, wantErr: storage.ErrInvalidMediaType},
		{name: "TypeMismatch", info: storage.ObjectInfo{Size: 1024, ContentType: "image/png"}, wantErr: storage.ErrInvalidMediaType},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			media, store := newMediaStore(t)
			gomock.InOrder(
				store.EXPECT().Stat(gomock.Any(), uploadKey).Return(tc.info, nil),
				store.EXPECT().Delete(gomock.Any(), uploadKey).Return(nil),
			)

			_, err := media.Promote(t.Context(), "places/7", uploadKey)
			assert.ErrorIs(t, err, tc.wantErr)
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
		{name: "TooLarge", info: storage.ObjectInfo{Size: storage.MaxVideoSize + 1, ContentType: "video/mp4"}, wantErr: storage.ErrVideoTooLarge},
		{name: "TypeMismatch", info: storage.ObjectInfo{Size: 1024, ContentType: "video/webm"}, wantErr: storage.ErrInvalidMediaType},
		{name: "ImageTypeForVideo", info: storage.ObjectInfo{Size: 1024, ContentType: "image/png"}, wantErr: storage.ErrInvalidMediaType},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			media, store := newMediaStore(t)
			gomock.InOrder(
				store.EXPECT().Stat(gomock.Any(), videoKey).Return(tc.info, nil),
				store.EXPECT().Delete(gomock.Any(), videoKey).Return(nil),
			)

			_, err := media.Promote(t.Context(), "places/7", videoKey)
			assert.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestMediaStore_Promote_VideoContentTypeForImageExtension(t *testing.T) {
	media, store := newMediaStore(t)
	gomock.InOrder(
		store.EXPECT().Stat(gomock.Any(), uploadKey).Return(storage.ObjectInfo{Size: 1024, ContentType: "video/mp4"}, nil),
		store.EXPECT().Delete(gomock.Any(), uploadKey).Return(nil),
	)

	_, err := media.Promote(t.Context(), "places/7", uploadKey)
	assert.ErrorIs(t, err, storage.ErrInvalidMediaType)
}

func TestMediaStore_Promote_RejectedImageCleanupFailure(t *testing.T) {
	media, store := newMediaStore(t)
	gomock.InOrder(
		store.EXPECT().Stat(gomock.Any(), uploadKey).Return(storage.ObjectInfo{Size: storage.MaxImageSize + 1, ContentType: "image/webp"}, nil),
		store.EXPECT().Delete(gomock.Any(), uploadKey).Return(errors.New("connection reset")),
	)

	_, err := media.Promote(t.Context(), "places/7", uploadKey)
	assert.ErrorIs(t, err, storage.ErrImageTooLarge)
}

func TestMediaStore_Promote_StoreFailure(t *testing.T) {
	boom := errors.New("connection reset")

	t.Run("Stat", func(t *testing.T) {
		media, store := newMediaStore(t)
		store.EXPECT().Stat(gomock.Any(), uploadKey).Return(storage.ObjectInfo{}, boom)

		_, err := media.Promote(t.Context(), "places/7", uploadKey)
		assert.ErrorIs(t, err, boom)
	})

	t.Run("Copy", func(t *testing.T) {
		media, store := newMediaStore(t)
		gomock.InOrder(
			store.EXPECT().Stat(gomock.Any(), uploadKey).Return(storage.ObjectInfo{Size: 1024, ContentType: "image/webp"}, nil),
			store.EXPECT().Copy(gomock.Any(), uploadKey, promotedKey).Return(boom),
		)

		_, err := media.Promote(t.Context(), "places/7", uploadKey)
		assert.ErrorIs(t, err, boom)
	})
}

func TestMediaStore_Delete(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name     string
		storeErr error
		wantErr  error
	}{
		{name: "Deletes", storeErr: nil, wantErr: nil},
		{name: "StoreFailure", storeErr: boom, wantErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			media, store := newMediaStore(t)
			store.EXPECT().Delete(gomock.Any(), promotedKey).Return(tc.storeErr)

			err := media.Delete(t.Context(), promotedKey)
			assert.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestMediaStore_PublicURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		want    string
	}{
		{name: "JoinsBaseAndKey", baseURL: publicBaseURL, want: publicBaseURL + "/" + promotedKey},
		{name: "TrailingSlash", baseURL: publicBaseURL + "/", want: publicBaseURL + "/" + promotedKey},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := mock.NewMockObjectStore(gomock.NewController(t))
			got := storage.NewMediaStore(store, tc.baseURL).PublicURL(promotedKey)
			assert.Equal(t, tc.want, got)
		})
	}
}
