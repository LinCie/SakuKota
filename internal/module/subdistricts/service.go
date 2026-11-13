package subdistricts

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
	// Create checks a new sub-district's data and saves it.
	Create(ctx context.Context, sd *SubDistrict) (*SubDistrict, error)
	// GetMany returns every sub-district.
	GetMany(ctx context.Context) ([]SubDistrict, error)
	// GetSubDistrict returns the sub-district with the given id.
	GetSubDistrict(ctx context.Context, id int64) (*SubDistrict, error)
	// Update overwrites the sub-district with the given id with the details in sd.
	Update(ctx context.Context, id int64, sd *SubDistrict) (*SubDistrict, error)
	// Delete removes the sub-district with the given id.
	Delete(ctx context.Context, id int64) error
	// ImageUploadURL gives a temporary link to upload a new image in the given slot.
	ImageUploadURL(ctx context.Context, id int64, slot ImageSlot, contentType string) (storage.Upload, error)
	// AddMedia attaches an uploaded photo to the sub-district.
	AddMedia(ctx context.Context, id int64, uploadKey string) (*SubDistrict, error)
	// DeleteMedia removes a photo from the sub-district.
	DeleteMedia(ctx context.Context, id, mediaID int64) error
	// SetStaticMap sets the uploaded image as the static map, replacing any previous one.
	SetStaticMap(ctx context.Context, id int64, uploadKey string) (*SubDistrict, error)
}

type service struct {
	repo   Repository
	images ImageStorage
}

// NewService builds the sub-district service from the repository and the image storage.
func NewService(repo Repository, images ImageStorage) Service {
	return &service{
		repo:   repo,
		images: images,
	}
}

// -----------------------------------------------------------------------------
// service
// -----------------------------------------------------------------------------

// Create checks the new sub-district's data, then saves it.
func (s *service) Create(ctx context.Context, sd *SubDistrict) (*SubDistrict, error) {
	if err := validate(sd); err != nil {
		return nil, err
	}

	return s.repo.Create(ctx, sd)
}

// GetMany returns every sub-district.
func (s *service) GetMany(ctx context.Context) ([]SubDistrict, error) {
	return s.repo.GetMany(ctx)
}

// GetSubDistrict returns the sub-district with the given id.
func (s *service) GetSubDistrict(ctx context.Context, id int64) (*SubDistrict, error) {
	return s.repo.GetSubDistrict(ctx, id)
}

// Update checks the new details, then overwrites the sub-district with the given id.
// If there is no such sub-district, the repository reports ErrSubDistrictNotFound.
func (s *service) Update(ctx context.Context, id int64, sd *SubDistrict) (*SubDistrict, error) {
	if err := validate(sd); err != nil {
		return nil, err
	}
	sd.ID = id

	return s.repo.Update(ctx, sd)
}

// Delete removes the sub-district with the given id.
// The repository also queues its photos and static map for deletion from storage.
func (s *service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

// ImageUploadURL checks the sub-district exists, then gives a temporary link to upload an image.
// The image is stored under the sub-district's folder for that slot.
func (s *service) ImageUploadURL(ctx context.Context, id int64, slot ImageSlot, contentType string) (storage.Upload, error) {
	if _, err := s.repo.GetSubDistrict(ctx, id); err != nil {
		return storage.Upload{}, err
	}

	return s.images.PresignUpload(ctx, imagePrefix(id, slot), contentType)
}

// AddMedia attaches an uploaded photo to the sub-district's photo list.
// If the photo cannot be attached, for example because the list is full, its file is deleted again.
func (s *service) AddMedia(ctx context.Context, id int64, uploadKey string) (*SubDistrict, error) {
	return s.storeImage(ctx, id, SlotMedias, uploadKey, s.repo.AddMedia)
}

// DeleteMedia removes a photo from the sub-district.
// The repository also queues its file for deletion from storage.
func (s *service) DeleteMedia(ctx context.Context, id, mediaID int64) error {
	return s.repo.DeleteMedia(ctx, id, mediaID)
}

// SetStaticMap moves the uploaded image into place as the sub-district's static map.
// The repository queues the previous static map for deletion.
func (s *service) SetStaticMap(ctx context.Context, id int64, uploadKey string) (*SubDistrict, error) {
	return s.storeImage(ctx, id, SlotStaticMap, uploadKey, s.repo.SetStaticMap)
}

// storeImage moves the upload into the slot, then saves it with store.
// If saving fails, the moved file is deleted again, and a failed delete is ignored.
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

// imagePrefix returns the storage folder for the images in a slot of the sub-district with id.
func imagePrefix(id int64, slot ImageSlot) string {
	return "subdistricts/" + strconv.FormatInt(id, 10) + "/" + string(slot)
}

// -----------------------------------------------------------------------------
// validation
// -----------------------------------------------------------------------------

// validate cleans up the text fields, then checks every field against its limits.
// Blank optional text is saved as nothing.
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
