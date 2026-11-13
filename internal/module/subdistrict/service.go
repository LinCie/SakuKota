package subdistrict

import (
	"context"
	"strconv"
	"strings"
	"unicode/utf8"

	"rantau-go/internal/pkg/text"
	"rantau-go/internal/storage"
)

// Service defines the operations required by the handler.
//
//go:generate mockgen -source=service.go -destination=mock/service.go -package=mock
type Service interface {
	Create(ctx context.Context, sd *SubDistrict) (*SubDistrict, error)
	GetMany(ctx context.Context) ([]SubDistrict, error)
	GetSubDistrict(ctx context.Context, id int64) (*SubDistrict, error)
	// Update replaces the sub-district's editable fields with those of sd.
	Update(ctx context.Context, id int64, sd *SubDistrict) (*SubDistrict, error)
	Delete(ctx context.Context, id int64) error
	// ImageUploadURL returns an upload for a new image in the sub-district's slot.
	ImageUploadURL(ctx context.Context, id int64, slot ImageSlot, contentType string) (storage.Upload, error)
	// AddMedia adds the uploaded image to the sub-district's medias.
	AddMedia(ctx context.Context, id int64, uploadKey string) (*SubDistrict, error)
	// DeleteMedia removes the media from the sub-district's medias.
	DeleteMedia(ctx context.Context, id, mediaID int64) error
	// SetStaticMap makes the uploaded image the sub-district's static map,
	// replacing any previous one.
	SetStaticMap(ctx context.Context, id int64, uploadKey string) (*SubDistrict, error)
}

type service struct {
	repo   Repository
	images ImageStorage
}

func NewService(repo Repository, images ImageStorage) Service {
	return &service{
		repo:   repo,
		images: images,
	}
}

// -----------------------------------------------------------------------------
// service
// -----------------------------------------------------------------------------

func (s *service) Create(ctx context.Context, sd *SubDistrict) (*SubDistrict, error) {
	if err := validate(sd); err != nil {
		return nil, err
	}

	return s.repo.Create(ctx, sd)
}

func (s *service) GetMany(ctx context.Context) ([]SubDistrict, error) {
	return s.repo.GetMany(ctx)
}

func (s *service) GetSubDistrict(ctx context.Context, id int64) (*SubDistrict, error) {
	return s.repo.GetSubDistrict(ctx, id)
}

// Update replaces the stored sub-district's fields with those of sd. The
// repository reports ErrSubDistrictNotFound when there is no such sub-district.
func (s *service) Update(ctx context.Context, id int64, sd *SubDistrict) (*SubDistrict, error) {
	if err := validate(sd); err != nil {
		return nil, err
	}
	sd.ID = id

	return s.repo.Update(ctx, sd)
}

// Delete removes the sub-district. The repository queues its medias and
// static map for deletion.
func (s *service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

// ImageUploadURL returns an upload for a new image in slot of the sub-district.
func (s *service) ImageUploadURL(ctx context.Context, id int64, slot ImageSlot, contentType string) (storage.Upload, error) {
	if _, err := s.repo.GetSubDistrict(ctx, id); err != nil {
		return storage.Upload{}, err
	}

	return s.images.PresignUpload(ctx, imagePrefix(id, slot), contentType)
}

// AddMedia promotes the upload and adds it to the sub-district's medias.
func (s *service) AddMedia(ctx context.Context, id int64, uploadKey string) (*SubDistrict, error) {
	return s.storeImage(ctx, id, SlotMedias, uploadKey, s.repo.AddMedia)
}

// DeleteMedia removes the media from the sub-district. The repository queues
// its image for deletion.
func (s *service) DeleteMedia(ctx context.Context, id, mediaID int64) error {
	return s.repo.DeleteMedia(ctx, id, mediaID)
}

// SetStaticMap promotes the upload and stores it as the sub-district's static
// map. The repository queues the previous static map for deletion.
func (s *service) SetStaticMap(ctx context.Context, id int64, uploadKey string) (*SubDistrict, error) {
	return s.storeImage(ctx, id, SlotStaticMap, uploadKey, s.repo.SetStaticMap)
}

// storeImage promotes the upload to slot and stores it with store. If the
// promoted image cannot be stored, for example because the sub-district
// already has storage.MaxMedias, its delete is attempted once and a failure is
// ignored: it only leaves an unused object behind.
func (s *service) storeImage(ctx context.Context, id int64, slot ImageSlot, uploadKey string,
	store func(ctx context.Context, id int64, m storage.Media) (*SubDistrict, error),
) (*SubDistrict, error) {
	if _, err := s.repo.GetSubDistrict(ctx, id); err != nil {
		return nil, err
	}

	promoted, err := s.images.Promote(ctx, imagePrefix(id, slot), uploadKey)
	if err != nil {
		return nil, err
	}

	updated, err := store(ctx, id, promoted)
	if err != nil {
		_ = s.images.Delete(context.WithoutCancel(ctx), promoted.Key)
		return nil, err
	}

	return updated, nil
}

// imagePrefix is the storage prefix of the images in slot of sub-district id.
func imagePrefix(id int64, slot ImageSlot) string {
	return "subdistricts/" + strconv.FormatInt(id, 10) + "/" + string(slot)
}

// -----------------------------------------------------------------------------
// validation
// -----------------------------------------------------------------------------

// validate trims the text fields, then checks every field against its limits.
// Blank optional text is stored as absent.
func validate(sd *SubDistrict) error {
	sd.Name = strings.TrimSpace(sd.Name)
	if n := utf8.RuneCountInString(sd.Name); n == 0 || n > maxNameLength {
		return ErrInvalidName
	}

	sd.Slug = strings.TrimSpace(sd.Slug)
	if n := utf8.RuneCountInString(sd.Slug); n == 0 || n > maxNameLength {
		return ErrInvalidSlug
	}

	sd.FloodHistory = text.TrimOptional(sd.FloodHistory)
	sd.SafetyCrimeInformation = text.TrimOptional(sd.SafetyCrimeInformation)

	if v := sd.EstimatedDailySpending; v != nil && (*v < 0 || *v > maxEstimatedDailySpending) {
		return ErrInvalidEstimatedDailySpending
	}

	return nil
}
