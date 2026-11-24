package users

import (
	"context"
	"strconv"
	"strings"

	"sakukota-go/internal/domain"
	"sakukota-go/internal/storage"
)

//go:generate mockgen -source=service.go -destination=mock/service.go -package=mock

// Repository is the persistence the service needs.
type Repository interface {
	// Create saves a new user and returns it with its generated fields, or
	// domain.ErrEmailTaken.
	Create(ctx context.Context, u *domain.User) (*domain.User, error)
	// FetchByEmail returns the user with the email, or domain.ErrUserNotFound.
	FetchByEmail(ctx context.Context, email string) (*domain.User, error)
	// Fetch returns the user with the id, or domain.ErrUserNotFound.
	Fetch(ctx context.Context, id int64) (*domain.User, error)
	// Rename sets the user's name and returns the user, or
	// domain.ErrUserNotFound.
	Rename(ctx context.Context, id int64, name string) (*domain.User, error)
	// SetAvatar replaces the user's profile photo, queues the old file for
	// deletion, and returns the user, or domain.ErrUserNotFound.
	SetAvatar(ctx context.Context, id int64, m domain.Media) (*domain.User, error)
}

// MediaStorage is the file storage the service needs.
type MediaStorage interface {
	// PresignUpload returns a temporary link for uploading a file under prefix.
	PresignUpload(ctx context.Context, prefix, contentType string) (storage.Upload, error)
	// Promote moves the uploaded file under uploadKey into prefix and describes it as a photo.
	Promote(ctx context.Context, prefix, uploadKey string) (domain.Media, error)
	// Delete removes the file stored under key.
	Delete(ctx context.Context, key string) error
	// PublicURL returns the link a browser can use to open the file stored under key.
	PublicURL(key string) string
}

// service provides user accounts and profiles.
type service struct {
	repo  Repository
	media MediaStorage
}

// NewService returns a service that stores users in repo and their profile
// photos in media.
func NewService(repo Repository, media MediaStorage) Service {
	return &service{repo: repo, media: media}
}

// Create saves a new user, or returns domain.ErrEmailTaken.
func (s *service) Create(ctx context.Context, u *domain.User) (*domain.User, error) {
	return s.repo.Create(ctx, u)
}

// FetchByEmail returns the user with the email, or domain.ErrUserNotFound.
func (s *service) FetchByEmail(ctx context.Context, email string) (*domain.User, error) {
	return s.repo.FetchByEmail(ctx, email)
}

// Fetch returns the user with the id, or domain.ErrUserNotFound.
func (s *service) Fetch(ctx context.Context, id int64) (*domain.User, error) {
	return s.repo.Fetch(ctx, id)
}

// Rename checks the name as registering does, then sets it as the user's name.
func (s *service) Rename(ctx context.Context, id int64, name string) (*domain.User, error) {
	name, err := domain.NormalizeUserName(name)
	if err != nil {
		return nil, err
	}

	return s.repo.Rename(ctx, id, name)
}

// AvatarUploadURL returns a temporary link to upload a profile photo of the
// user. Only images are accepted: other types fail with
// domain.ErrAvatarNotImage.
func (s *service) AvatarUploadURL(ctx context.Context, id int64, contentType string) (storage.Upload, error) {
	if !strings.HasPrefix(contentType, "image/") {
		return storage.Upload{}, domain.ErrAvatarNotImage
	}

	return s.media.PresignUpload(ctx, avatarPrefix(id), contentType)
}

// SetAvatar makes the uploaded photo the user's profile photo and returns the
// user. The previous photo is deleted.
func (s *service) SetAvatar(ctx context.Context, id int64, uploadKey string) (*domain.User, error) {
	promoted, err := s.media.Promote(ctx, avatarPrefix(id), uploadKey)
	if err != nil {
		return nil, err
	}

	updated, err := s.repo.SetAvatar(ctx, id, promoted)
	if err != nil {
		// Storing failed, so remove the file just moved.
		// WithoutCancel: the cleanup must run even if the request was cancelled.
		_ = s.media.Delete(context.WithoutCancel(ctx), promoted.Key)
		return nil, err
	}

	return updated, nil
}

// avatarPrefix returns the storage folder of the user's profile photos. Only
// images are presigned there, so Promote never finds a video in it.
func avatarPrefix(id int64) string {
	return "avatars/" + strconv.FormatInt(id, 10)
}
