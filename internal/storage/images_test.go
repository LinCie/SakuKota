package storage_test

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"rantau-go/internal/storage"
	"rantau-go/internal/storage/mock"
)

const (
	publicBaseURL = "http://rantau-images.web.localhost:3902"
	uploadKey     = "uploads/places/7/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp"
	promotedKey   = "places/7/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp"
)

func newImages(t *testing.T) (*storage.Images, *mock.MockObjectStore) {
	t.Helper()

	store := mock.NewMockObjectStore(gomock.NewController(t))
	return storage.NewImages(store, publicBaseURL), store
}

// -----------------------------------------------------------------------------
// Presigning uploads
// -----------------------------------------------------------------------------

func TestImages_PresignUpload(t *testing.T) {
	tests := []struct {
		contentType string
		wantKey     *regexp.Regexp
	}{
		{contentType: "image/jpeg", wantKey: uploadKeyPattern("jpg")},
		{contentType: "image/png", wantKey: uploadKeyPattern("png")},
		{contentType: "image/webp", wantKey: uploadKeyPattern("webp")},
	}

	for _, tc := range tests {
		t.Run(tc.contentType, func(t *testing.T) {
			images, store := newImages(t)
			presigned := storage.PresignedUpload{
				URL:       "http://localhost:3900/rantau-images/signed",
				Method:    "PUT",
				Headers:   map[string]string{"Content-Type": tc.contentType},
				ExpiresAt: time.Date(2026, 10, 31, 12, 5, 0, 0, time.UTC),
			}

			var presignedKey string
			store.EXPECT().PresignPut(gomock.Any(), gomock.Any(), tc.contentType, storage.UploadTTL).DoAndReturn(
				func(_ context.Context, key, _ string, _ time.Duration) (storage.PresignedUpload, error) {
					presignedKey = key
					return presigned, nil
				})

			got, err := images.PresignUpload(context.Background(), "places/7", tc.contentType)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tc.wantKey.MatchString(presignedKey) {
				t.Errorf("presigned key = %q, want a match for %s", presignedKey, tc.wantKey)
			}
			if got.Key != presignedKey {
				t.Errorf("returned key = %q, want the presigned key %q", got.Key, presignedKey)
			}
			if got.URL != presigned.URL || got.Method != presigned.Method ||
				got.Headers["Content-Type"] != tc.contentType || !got.ExpiresAt.Equal(presigned.ExpiresAt) {
				t.Errorf("returned upload = %+v, want the presigned request %+v", got.PresignedUpload, presigned)
			}
		})
	}
}

// uploadKeyPattern matches an upload key for place 7 with a random UUID name.
func uploadKeyPattern(ext string) *regexp.Regexp {
	return regexp.MustCompile(`^uploads/places/7/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\.` + ext + `$`)
}

func TestImages_PresignUpload_KeysAreUnique(t *testing.T) {
	images, store := newImages(t)
	store.EXPECT().PresignPut(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(storage.PresignedUpload{}, nil).Times(2)

	first, err := images.PresignUpload(context.Background(), "places/7", "image/png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	second, err := images.PresignUpload(context.Background(), "places/7", "image/png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if first.Key == "" || first.Key == second.Key {
		t.Errorf("keys = %q and %q, want two different keys", first.Key, second.Key)
	}
}

func TestImages_PresignUpload_RejectedType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
	}{
		{name: "empty", contentType: ""},
		{name: "GIF", contentType: "image/gif"},
		{name: "SVG", contentType: "image/svg+xml"},
		{name: "HTML", contentType: "text/html"},
		{name: "PDF", contentType: "application/pdf"},
		{name: "allowed type with parameters", contentType: "image/png; charset=utf-8"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The mock fails the test if anything is presigned.
			images, _ := newImages(t)

			_, err := images.PresignUpload(context.Background(), "places/7", tc.contentType)
			if !errors.Is(err, storage.ErrInvalidImageType) {
				t.Fatalf("err = %v, want ErrInvalidImageType", err)
			}
		})
	}
}

func TestImages_PresignUpload_StoreFailure(t *testing.T) {
	boom := errors.New("connection reset")
	images, store := newImages(t)
	store.EXPECT().PresignPut(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(storage.PresignedUpload{}, boom)

	_, err := images.PresignUpload(context.Background(), "places/7", "image/png")
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want %v", err, boom)
	}
}

// -----------------------------------------------------------------------------
// Promoting uploads
// -----------------------------------------------------------------------------

func TestImages_Promote(t *testing.T) {
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
			images, store := newImages(t)
			gomock.InOrder(
				store.EXPECT().Stat(gomock.Any(), uploadKey).
					Return(storage.ObjectInfo{Size: tc.size, ContentType: "image/webp"}, nil),
				store.EXPECT().Copy(gomock.Any(), uploadKey, promotedKey).Return(nil),
				store.EXPECT().Delete(gomock.Any(), uploadKey).Return(tc.deleteErr),
			)

			got, err := images.Promote(context.Background(), "places/7", uploadKey)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != promotedKey {
				t.Errorf("key = %q, want %q", got, promotedKey)
			}
		})
	}
}

func TestImages_Promote_InvalidKey(t *testing.T) {
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
		{name: "no extension", key: "uploads/places/7/" + id},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The mock fails the test if the store is touched.
			images, _ := newImages(t)

			_, err := images.Promote(context.Background(), "places/7", tc.key)
			if !errors.Is(err, storage.ErrInvalidImageKey) {
				t.Fatalf("err = %v, want ErrInvalidImageKey", err)
			}
		})
	}
}

func TestImages_Promote_NotUploaded(t *testing.T) {
	images, store := newImages(t)
	store.EXPECT().Stat(gomock.Any(), uploadKey).Return(storage.ObjectInfo{}, storage.ErrObjectNotFound)

	_, err := images.Promote(context.Background(), "places/7", uploadKey)
	if !errors.Is(err, storage.ErrImageNotUploaded) {
		t.Fatalf("err = %v, want ErrImageNotUploaded", err)
	}
}

func TestImages_Promote_RejectedImage(t *testing.T) {
	tests := []struct {
		name    string
		info    storage.ObjectInfo
		wantErr error
	}{
		{name: "too large", info: storage.ObjectInfo{Size: storage.MaxImageSize + 1, ContentType: "image/webp"}, wantErr: storage.ErrImageTooLarge},
		{name: "type not allowed", info: storage.ObjectInfo{Size: 1024, ContentType: "text/html"}, wantErr: storage.ErrInvalidImageType},
		{name: "type does not match the extension", info: storage.ObjectInfo{Size: 1024, ContentType: "image/png"}, wantErr: storage.ErrInvalidImageType},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			images, store := newImages(t)
			store.EXPECT().Stat(gomock.Any(), uploadKey).Return(tc.info, nil)
			store.EXPECT().Delete(gomock.Any(), uploadKey).Return(nil)

			_, err := images.Promote(context.Background(), "places/7", uploadKey)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestImages_Promote_RejectedImageCleanupFailure(t *testing.T) {
	images, store := newImages(t)
	store.EXPECT().Stat(gomock.Any(), uploadKey).
		Return(storage.ObjectInfo{Size: storage.MaxImageSize + 1, ContentType: "image/webp"}, nil)
	store.EXPECT().Delete(gomock.Any(), uploadKey).Return(errors.New("connection reset"))

	_, err := images.Promote(context.Background(), "places/7", uploadKey)
	if !errors.Is(err, storage.ErrImageTooLarge) {
		t.Fatalf("err = %v, want ErrImageTooLarge", err)
	}
}

func TestImages_Promote_StoreFailure(t *testing.T) {
	boom := errors.New("connection reset")

	t.Run("stat", func(t *testing.T) {
		images, store := newImages(t)
		store.EXPECT().Stat(gomock.Any(), uploadKey).Return(storage.ObjectInfo{}, boom)

		_, err := images.Promote(context.Background(), "places/7", uploadKey)
		if !errors.Is(err, boom) {
			t.Fatalf("err = %v, want %v", err, boom)
		}
	})

	t.Run("copy", func(t *testing.T) {
		images, store := newImages(t)
		store.EXPECT().Stat(gomock.Any(), uploadKey).
			Return(storage.ObjectInfo{Size: 1024, ContentType: "image/webp"}, nil)
		store.EXPECT().Copy(gomock.Any(), uploadKey, promotedKey).Return(boom)

		_, err := images.Promote(context.Background(), "places/7", uploadKey)
		if !errors.Is(err, boom) {
			t.Fatalf("err = %v, want %v", err, boom)
		}
	})
}

// -----------------------------------------------------------------------------
// Deleting and serving images
// -----------------------------------------------------------------------------

func TestImages_Delete(t *testing.T) {
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
			images, store := newImages(t)
			store.EXPECT().Delete(gomock.Any(), promotedKey).Return(tc.storeErr)

			err := images.Delete(context.Background(), promotedKey)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestImages_PublicURL(t *testing.T) {
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

			got := storage.NewImages(store, tc.baseURL).PublicURL(promotedKey)
			if got != tc.want {
				t.Errorf("PublicURL = %q, want %q", got, tc.want)
			}
		})
	}
}
