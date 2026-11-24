package users_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"sakukota-go/internal/domain"
	"sakukota-go/internal/module/users"
	"sakukota-go/internal/module/users/mock"
	"sakukota-go/internal/storage"
)

const testEmail = "ana@example.com"

var errBoom = errors.New("boom")

func newRepo(t *testing.T) *mock.MockRepository {
	t.Helper()
	return mock.NewMockRepository(gomock.NewController(t))
}

func TestService_Create_ReturnsCreatedUser(t *testing.T) {
	repo := newRepo(t)
	svc := users.NewService(repo, nil)
	in := &domain.User{Name: "Ana", Email: testEmail, PasswordHash: "hash"}
	want := &domain.User{ID: 1, Name: "Ana", Email: testEmail, PasswordHash: "hash", Role: "user"}
	repo.EXPECT().Create(gomock.Any(), in).Return(want, nil)

	got, err := svc.Create(t.Context(), in)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestService_Create_PropagatesErrors(t *testing.T) {
	tests := []struct {
		name string
		want error
	}{
		{name: "EmailTaken", want: domain.ErrEmailTaken},
		{name: "Boom", want: errBoom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newRepo(t)
			svc := users.NewService(repo, nil)
			in := &domain.User{Name: "Ana", Email: testEmail}
			repo.EXPECT().Create(gomock.Any(), in).Return(nil, tc.want)

			_, err := svc.Create(t.Context(), in)
			assert.ErrorIs(t, err, tc.want)
		})
	}
}

func TestService_FetchByEmail_ReturnsUser(t *testing.T) {
	repo := newRepo(t)
	svc := users.NewService(repo, nil)
	want := &domain.User{ID: 7, Email: testEmail}
	repo.EXPECT().FetchByEmail(gomock.Any(), testEmail).Return(want, nil)

	got, err := svc.FetchByEmail(t.Context(), testEmail)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestService_FetchByEmail_PropagatesErrors(t *testing.T) {
	tests := []struct {
		name string
		want error
	}{
		{name: "NotFound", want: domain.ErrUserNotFound},
		{name: "Boom", want: errBoom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newRepo(t)
			svc := users.NewService(repo, nil)
			repo.EXPECT().FetchByEmail(gomock.Any(), testEmail).Return(nil, tc.want)

			_, err := svc.FetchByEmail(t.Context(), testEmail)
			assert.ErrorIs(t, err, tc.want)
		})
	}
}

func TestService_Fetch_ReturnsUser(t *testing.T) {
	repo := newRepo(t)
	svc := users.NewService(repo, nil)
	want := &domain.User{ID: 7, Email: testEmail}
	repo.EXPECT().Fetch(gomock.Any(), int64(7)).Return(want, nil)

	got, err := svc.Fetch(t.Context(), 7)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestService_Fetch_PropagatesErrors(t *testing.T) {
	tests := []struct {
		name string
		want error
	}{
		{name: "NotFound", want: domain.ErrUserNotFound},
		{name: "Boom", want: errBoom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newRepo(t)
			svc := users.NewService(repo, nil)
			repo.EXPECT().Fetch(gomock.Any(), int64(7)).Return(nil, tc.want)

			_, err := svc.Fetch(t.Context(), 7)
			assert.ErrorIs(t, err, tc.want)
		})
	}
}

func TestService_Rename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantName string // stored name; empty when nothing may be stored
		wantErr  error
	}{
		{name: "Trimmed", input: "  Ana Lestari ", wantName: "Ana Lestari"},
		{name: "MaxLength", input: strings.Repeat("é", domain.MaxUserNameLength), wantName: strings.Repeat("é", domain.MaxUserNameLength)},
		{name: "Blank", input: "   ", wantErr: domain.ErrUserNameRequired},
		{name: "TooLong", input: strings.Repeat("é", domain.MaxUserNameLength+1), wantErr: domain.ErrUserNameTooLong},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newRepo(t)
			want := &domain.User{ID: 7, Name: tc.wantName}
			if tc.wantErr == nil {
				repo.EXPECT().Rename(gomock.Any(), int64(7), tc.wantName).Return(want, nil)
			}

			got, err := users.NewService(repo, nil).Rename(t.Context(), 7, tc.input)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Same(t, want, got)
		})
	}
}

func TestService_Rename_PropagatesErrors(t *testing.T) {
	for _, repoErr := range []error{domain.ErrUserNotFound, errBoom} {
		repo := newRepo(t)
		repo.EXPECT().Rename(gomock.Any(), int64(7), "Ana").Return(nil, repoErr)

		_, err := users.NewService(repo, nil).Rename(t.Context(), 7, "Ana")
		assert.ErrorIs(t, err, repoErr)
	}
}

// avatarUploadKey is an upload key issued for user 7's profile photo.
const avatarUploadKey = "uploads/avatars/7/0f8b2c1e-3d4a-4b5c-8d6e-7f8091a2b3c4.webp"

// promotedAvatar is avatarUploadKey once promoted.
var promotedAvatar = domain.Media{Key: "avatars/7/0f8b2c1e-3d4a-4b5c-8d6e-7f8091a2b3c4.webp", ContentType: "image/webp", Size: 1024}

func newMediaMocks(t *testing.T) (*mock.MockRepository, *mock.MockMediaStorage, users.Service) {
	t.Helper()
	ctrl := gomock.NewController(t)
	repo := mock.NewMockRepository(ctrl)
	media := mock.NewMockMediaStorage(ctrl)
	return repo, media, users.NewService(repo, media)
}

func TestService_AvatarUploadURL(t *testing.T) {
	_, media, svc := newMediaMocks(t)
	upload := storage.Upload{Key: avatarUploadKey, URL: "http://localhost:3900/signed", Method: "PUT"}
	media.EXPECT().PresignUpload(gomock.Any(), "avatars/7", "image/webp").Return(upload, nil)

	got, err := svc.AvatarUploadURL(t.Context(), 7, "image/webp")

	require.NoError(t, err)
	assert.Equal(t, upload, got)
}

func TestService_AvatarUploadURL_OnlyImages(t *testing.T) {
	for _, contentType := range []string{"video/mp4", "video/webm", ""} {
		// Nothing may be presigned: the mock has no expectations.
		_, _, svc := newMediaMocks(t)

		_, err := svc.AvatarUploadURL(t.Context(), 7, contentType)

		assert.ErrorIs(t, err, domain.ErrAvatarNotImage, "content type %q", contentType)
	}
}

func TestService_AvatarUploadURL_PresignFailure(t *testing.T) {
	for _, presignErr := range []error{storage.ErrInvalidMediaType, errBoom} {
		_, media, svc := newMediaMocks(t)
		media.EXPECT().PresignUpload(gomock.Any(), "avatars/7", "image/gif").Return(storage.Upload{}, presignErr)

		_, err := svc.AvatarUploadURL(t.Context(), 7, "image/gif")

		assert.ErrorIs(t, err, presignErr)
	}
}

func TestService_SetAvatar(t *testing.T) {
	// The promoted photo is stored as it is, and nothing is deleted.
	repo, media, svc := newMediaMocks(t)
	updated := &domain.User{ID: 7, AvatarKey: &promotedAvatar.Key}
	gomock.InOrder(
		media.EXPECT().Promote(gomock.Any(), "avatars/7", avatarUploadKey).Return(promotedAvatar, nil),
		repo.EXPECT().SetAvatar(gomock.Any(), int64(7), promotedAvatar).Return(updated, nil),
	)

	got, err := svc.SetAvatar(t.Context(), 7, avatarUploadKey)

	require.NoError(t, err)
	assert.Same(t, updated, got)
}

func TestService_SetAvatar_PromoteFailure(t *testing.T) {
	for _, promoteErr := range []error{storage.ErrInvalidUploadKey, storage.ErrMediaNotUploaded, storage.ErrImageTooLarge, errBoom} {
		// Nothing may be stored.
		_, media, svc := newMediaMocks(t)
		media.EXPECT().Promote(gomock.Any(), "avatars/7", avatarUploadKey).Return(domain.Media{}, promoteErr)

		_, err := svc.SetAvatar(t.Context(), 7, avatarUploadKey)

		assert.ErrorIs(t, err, promoteErr)
	}
}

func TestService_SetAvatar_StoreFailure(t *testing.T) {
	tests := []struct {
		name      string
		storeErr  error
		deleteErr error
	}{
		{name: "UserDeleted", storeErr: domain.ErrUserNotFound},
		{name: "StorageFailure", storeErr: errBoom},
		{name: "DeleteFailureIgnored", storeErr: errBoom, deleteErr: errors.New("timeout")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The promoted file is removed, even once the request is cancelled.
			ctx, cancel := context.WithCancel(t.Context())
			repo, media, svc := newMediaMocks(t)
			gomock.InOrder(
				media.EXPECT().Promote(gomock.Any(), "avatars/7", avatarUploadKey).Return(promotedAvatar, nil),
				repo.EXPECT().SetAvatar(gomock.Any(), int64(7), promotedAvatar).DoAndReturn(func(context.Context, int64, domain.Media) (*domain.User, error) {
					cancel()
					return nil, tc.storeErr
				}),
				media.EXPECT().Delete(gomock.Any(), promotedAvatar.Key).DoAndReturn(func(ctx context.Context, _ string) error {
					assert.NoError(t, ctx.Err(), "cleanup ran with a cancelled context")
					return tc.deleteErr
				}),
			)

			_, err := svc.SetAvatar(ctx, 7, avatarUploadKey)

			assert.ErrorIs(t, err, tc.storeErr)
		})
	}
}
