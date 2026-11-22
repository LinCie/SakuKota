package subdistricts

import (
	"context"
	"strconv"
	"strings"
	"unicode/utf8"

	"rantau-go/internal/pkg/postgis"
	"rantau-go/internal/pkg/text"
	"rantau-go/internal/storage"
)

// Service defines the operations required by the handler.
//
//go:generate mockgen -source=service.go -destination=mock/service.go -package=mock
type Service interface {
	// Create checks a new sub-district's data and saves it.
	Create(ctx context.Context, sd *SubDistrict) (*SubDistrict, error)
	// MediaUploadURL gives a temporary link to upload a new file in the given slot.
	MediaUploadURL(ctx context.Context, id int64, slot MediaSlot, contentType string) (storage.Upload, error)
	// AddMedia attaches an uploaded photo to the sub-district.
	AddMedia(ctx context.Context, id int64, uploadKey string) (*SubDistrict, error)
	// GetMany returns every sub-district.
	GetMany(ctx context.Context) ([]SubDistrict, error)
	// GetSubDistrict returns the sub-district with the given id.
	GetSubDistrict(ctx context.Context, id int64) (*SubDistrict, error)
	// Update overwrites the sub-district with the given id with the details in sd.
	Update(ctx context.Context, id int64, sd *SubDistrict) (*SubDistrict, error)
	// SetBoundary checks the area and saves it as the sub-district's boundary, replacing any previous one.
	SetBoundary(ctx context.Context, id int64, boundary *postgis.MultiPolygon) (*SubDistrict, error)
	// Delete removes the sub-district with the given id.
	Delete(ctx context.Context, id int64) error
	// DeleteMedia removes a photo from the sub-district.
	DeleteMedia(ctx context.Context, id, mediaID int64) error
}

type service struct {
	repo  Repository
	media MediaStorage
}

// -----------------------------------------------------------------------------
// Exported
// -----------------------------------------------------------------------------

func NewService(repo Repository, media MediaStorage) Service {
	return &service{
		repo:  repo,
		media: media,
	}
}

func (s *service) Create(ctx context.Context, sd *SubDistrict) (*SubDistrict, error) {
	if err := validate(sd); err != nil {
		return nil, err
	}

	return s.repo.Create(ctx, sd)
}

func (s *service) MediaUploadURL(ctx context.Context, id int64, slot MediaSlot, contentType string) (storage.Upload, error) {
	if _, err := s.repo.GetSubDistrict(ctx, id); err != nil {
		return storage.Upload{}, err
	}

	return s.media.PresignUpload(ctx, mediaPrefix(id, slot), contentType)
}

func (s *service) AddMedia(ctx context.Context, id int64, uploadKey string) (*SubDistrict, error) {
	return s.storeMedia(ctx, id, SlotMedias, uploadKey, s.repo.AddMedia)
}

func (s *service) GetMany(ctx context.Context) ([]SubDistrict, error) {
	return s.repo.GetMany(ctx)
}

func (s *service) GetSubDistrict(ctx context.Context, id int64) (*SubDistrict, error) {
	return s.repo.GetSubDistrict(ctx, id)
}

func (s *service) Update(ctx context.Context, id int64, sd *SubDistrict) (*SubDistrict, error) {
	if err := validate(sd); err != nil {
		return nil, err
	}
	sd.ID = id

	return s.repo.Update(ctx, sd)
}

// SetBoundary only checks that the area lies in Indonesia; the database rejects
// an area that crosses itself.
func (s *service) SetBoundary(ctx context.Context, id int64, boundary *postgis.MultiPolygon) (*SubDistrict, error) {
	if !postgis.MultiPolygonInIndonesia(boundary) {
		return nil, ErrInvalidBoundary
	}

	return s.repo.SetBoundary(ctx, id, boundary)
}

// Delete also queues the sub-district's photos for deletion, in the repository.
func (s *service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

// DeleteMedia also queues the photo file for deletion, in the repository.
func (s *service) DeleteMedia(ctx context.Context, id, mediaID int64) error {
	return s.repo.DeleteMedia(ctx, id, mediaID)
}

// -----------------------------------------------------------------------------
// Unexported
// -----------------------------------------------------------------------------

// storeMedia moves the upload into the slot, then saves it with store.
// If saving fails, the moved file is deleted again, and a failed delete is ignored.
func (s *service) storeMedia(ctx context.Context, id int64, slot MediaSlot, uploadKey string,
	store func(ctx context.Context, id int64, m storage.Media) (*SubDistrict, error),
) (*SubDistrict, error) {
	if _, err := s.repo.GetSubDistrict(ctx, id); err != nil {
		return nil, err
	}

	promoted, err := s.media.Promote(ctx, mediaPrefix(id, slot), uploadKey)
	if err != nil {
		return nil, err
	}

	updated, err := store(ctx, id, promoted)
	if err != nil {
		_ = s.media.Delete(context.WithoutCancel(ctx), promoted.Key)
		return nil, err
	}

	return updated, nil
}

// mediaPrefix returns the storage folder for the files in a slot of the sub-district with id.
func mediaPrefix(id int64, slot MediaSlot) string {
	return "subdistricts/" + strconv.FormatInt(id, 10) + "/" + string(slot)
}

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
