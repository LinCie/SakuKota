package subdistricts

import (
	"context"
	"strconv"
	"strings"
	"unicode/utf8"

	"rantau-go/internal/domain"
	"rantau-go/internal/pkg/postgis"
	"rantau-go/internal/pkg/text"
	"rantau-go/internal/storage"
)

//go:generate mockgen -source=service.go -destination=mock/service.go -package=mock

// Repository defines the persistence operations required by the service.
type Repository interface {
	// Create saves the sub-district and returns it with its photos.
	Create(ctx context.Context, sd *domain.SubDistrict) (*domain.SubDistrict, error)
	// AddMedia stores the photo in the sub-district's first free slot and returns the sub-district.
	// It returns domain.ErrSubDistrictNotFound, or domain.ErrMediaLimitReached if the sub-district is already full.
	AddMedia(ctx context.Context, id int64, m domain.Media) (*domain.SubDistrict, error)
	// List returns every sub-district with its photos.
	List(ctx context.Context) ([]domain.SubDistrict, error)
	// Fetch returns the sub-district with its photos.
	Fetch(ctx context.Context, id int64) (*domain.SubDistrict, error)
	// Update saves the sub-district's details and returns it with its photos. The boundary is kept.
	Update(ctx context.Context, sd *domain.SubDistrict) (*domain.SubDistrict, error)
	// SetBoundary saves the area as the sub-district's boundary and returns the sub-district.
	// It returns domain.ErrSubDistrictNotFound, or domain.ErrSubDistrictBoundaryInvalid if the area crosses itself.
	SetBoundary(ctx context.Context, id int64, boundary *postgis.MultiPolygon) (*domain.SubDistrict, error)
	// Delete removes the sub-district with its photos, and queues their files for deletion.
	// It returns domain.ErrSubDistrictNotFound, or domain.ErrSubDistrictInUse if places still use it.
	Delete(ctx context.Context, id int64) error
	// DeleteMedia removes the photo and queues its file for deletion.
	// Later photos move up one slot to close the gap; it returns domain.ErrMediaNotFound if there is no such photo.
	DeleteMedia(ctx context.Context, id, mediaID int64) error
}

// MediaStorage defines the media operations required by the service.
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

// Service holds the business rules of sub-districts. It is safe for
// concurrent use if its repository and media storage are.
type Service struct {
	repo  Repository
	media MediaStorage
}

// NewService builds a Service over repo and media.
func NewService(repo Repository, media MediaStorage) *Service {
	return &Service{
		repo:  repo,
		media: media,
	}
}

// Create checks a new sub-district's data and saves it.
func (s *Service) Create(ctx context.Context, sd *domain.SubDistrict) (*domain.SubDistrict, error) {
	if err := validate(sd); err != nil {
		return nil, err
	}

	return s.repo.Create(ctx, sd)
}

// MediaUploadURL gives a temporary link to upload a new file in the given slot.
func (s *Service) MediaUploadURL(ctx context.Context, id int64, slot MediaSlot, contentType string) (storage.Upload, error) {
	if _, err := s.repo.Fetch(ctx, id); err != nil {
		return storage.Upload{}, err
	}

	return s.media.PresignUpload(ctx, mediaPrefix(id, slot), contentType)
}

// AddMedia attaches an uploaded photo to the sub-district.
func (s *Service) AddMedia(ctx context.Context, id int64, uploadKey string) (*domain.SubDistrict, error) {
	return s.storeMedia(ctx, id, SlotMedias, uploadKey, s.repo.AddMedia)
}

// List returns every sub-district.
func (s *Service) List(ctx context.Context) ([]domain.SubDistrict, error) {
	return s.repo.List(ctx)
}

// Fetch returns the sub-district with the given id.
func (s *Service) Fetch(ctx context.Context, id int64) (*domain.SubDistrict, error) {
	return s.repo.Fetch(ctx, id)
}

// Update overwrites the sub-district with the given id with the details in sd.
func (s *Service) Update(ctx context.Context, id int64, sd *domain.SubDistrict) (*domain.SubDistrict, error) {
	if err := validate(sd); err != nil {
		return nil, err
	}
	sd.ID = id

	return s.repo.Update(ctx, sd)
}

// SetBoundary checks the area and saves it as the sub-district's boundary,
// replacing any previous one. It only checks that the area lies in Indonesia;
// the database rejects an area that crosses itself.
func (s *Service) SetBoundary(ctx context.Context, id int64, boundary *postgis.MultiPolygon) (*domain.SubDistrict, error) {
	if !postgis.MultiPolygonInIndonesia(boundary) {
		return nil, domain.ErrSubDistrictBoundaryInvalid
	}

	return s.repo.SetBoundary(ctx, id, boundary)
}

// Delete removes the sub-district with the given id. The repository also
// queues the sub-district's photos for deletion.
func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

// DeleteMedia removes a photo from the sub-district. The repository also
// queues the photo file for deletion.
func (s *Service) DeleteMedia(ctx context.Context, id, mediaID int64) error {
	return s.repo.DeleteMedia(ctx, id, mediaID)
}

// storeMedia moves the upload into the slot, then saves it with store.
// If saving fails, the moved file is deleted again, and a failed delete is ignored.
func (s *Service) storeMedia(ctx context.Context, id int64, slot MediaSlot, uploadKey string, store func(ctx context.Context, id int64, m domain.Media) (*domain.SubDistrict, error)) (*domain.SubDistrict, error) {
	if _, err := s.repo.Fetch(ctx, id); err != nil {
		return nil, err
	}

	promoted, err := s.media.Promote(ctx, mediaPrefix(id, slot), uploadKey)
	if err != nil {
		return nil, err
	}

	updated, err := store(ctx, id, promoted)
	if err != nil {
		_ = s.media.Delete(context.WithoutCancel(ctx), promoted.Key) // the original error matters more; uploads are cleaned up later
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
func validate(sd *domain.SubDistrict) error {
	sd.Name = strings.TrimSpace(sd.Name)
	if n := utf8.RuneCountInString(sd.Name); n == 0 || n > domain.MaxSubDistrictNameLength {
		return domain.ErrSubDistrictNameInvalid
	}

	sd.Slug = strings.TrimSpace(sd.Slug)
	if n := utf8.RuneCountInString(sd.Slug); n == 0 || n > domain.MaxSubDistrictNameLength {
		return domain.ErrSubDistrictSlugInvalid
	}

	sd.FloodHistory = text.TrimOptional(sd.FloodHistory)
	sd.SafetyCrimeInformation = text.TrimOptional(sd.SafetyCrimeInformation)

	if v := sd.EstimatedDailySpending; v != nil && (*v < 0 || *v > domain.MaxEstimatedDailySpending) {
		return domain.ErrSubDistrictSpendingInvalid
	}

	return nil
}
