package places

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
	// Create validates and saves a new place.
	Create(ctx context.Context, p *Place) (*Place, error)
	// MediaUploadURL returns a temporary link to upload a photo for the place.
	MediaUploadURL(ctx context.Context, id int64, contentType string) (storage.Upload, error)
	// AddMedia attaches an uploaded photo to the place.
	AddMedia(ctx context.Context, id int64, uploadKey string) (*Place, error)
	// GetMany returns one page of places matching the filter, plus the total match count.
	GetMany(ctx context.Context, filter PlaceFilter) ([]Place, int, error)
	// GetPlace returns the place with the given id.
	GetPlace(ctx context.Context, id int64) (*Place, error)
	// GetCategories returns every category.
	GetCategories(ctx context.Context) ([]Category, error)
	// GetCategory returns the category with the given id.
	GetCategory(ctx context.Context, id int64) (*Category, error)
	// Update validates p and overwrites the place with the given id.
	Update(ctx context.Context, id int64, p *Place) (*Place, error)
	// Delete removes the place with the given id.
	Delete(ctx context.Context, id int64) error
	// DeleteMedia removes a photo from the place.
	DeleteMedia(ctx context.Context, id, mediaID int64) error
}

type service struct {
	repo   Repository
	images ImageStorage
}

// -----------------------------------------------------------------------------
// Exported
// -----------------------------------------------------------------------------

func NewService(repo Repository, images ImageStorage) Service {
	return &service{
		repo:   repo,
		images: images,
	}
}

// NormalizeName trims name and rejects it if it is empty or too long.
func NormalizeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		return "", ErrInvalidName
	case utf8.RuneCountInString(name) > MaxNameLength:
		return "", ErrNameTooLong
	}

	return name, nil
}

// NormalizeOptionalFields trims the optional text fields of p and turns blank ones into nil.
// It rejects any that exceed their column length.
func NormalizeOptionalFields(p *Place) error {
	var ok bool

	if p.Address, ok = text.TrimOptionalMax(p.Address, MaxAddressLength); !ok {
		return ErrAddressTooLong
	}

	if p.OperatingHours, ok = text.TrimOptionalMax(p.OperatingHours, MaxOperatingHoursLength); !ok {
		return ErrOperatingHoursTooLong
	}

	if p.EstimatedPriceRange, ok = text.TrimOptionalMax(p.EstimatedPriceRange, MaxEstimatedPriceRangeLength); !ok {
		return ErrEstimatedPriceRangeTooLong
	}

	p.ShortDescription = text.TrimOptional(p.ShortDescription)

	return nil
}

func (s *service) Create(ctx context.Context, p *Place) (*Place, error) {
	if err := s.validate(ctx, p); err != nil {
		return nil, err
	}

	return s.repo.Create(ctx, p)
}

func (s *service) MediaUploadURL(ctx context.Context, id int64, contentType string) (storage.Upload, error) {
	if _, err := s.repo.GetPlace(ctx, id); err != nil {
		return storage.Upload{}, err
	}

	return s.images.PresignUpload(ctx, imagePrefix(id), contentType)
}

func (s *service) AddMedia(ctx context.Context, id int64, uploadKey string) (*Place, error) {
	if _, err := s.repo.GetPlace(ctx, id); err != nil {
		return nil, err
	}

	promoted, err := s.images.Promote(ctx, imagePrefix(id), uploadKey)
	if err != nil {
		return nil, err
	}

	updated, err := s.repo.AddMedia(ctx, id, promoted)
	if err != nil {
		// Attaching failed (e.g. the place is full), so remove the file just moved.
		// WithoutCancel: the cleanup must run even if the request was cancelled.
		_ = s.images.Delete(context.WithoutCancel(ctx), promoted.Key)
		return nil, err
	}

	return updated, nil
}

// GetMany caps the page size so no list is unbounded.
func (s *service) GetMany(ctx context.Context, filter PlaceFilter) ([]Place, int, error) {
	if filter.Limit <= 0 || filter.Limit > maxLimit {
		filter.Limit = maxLimit
	}

	return s.repo.GetMany(ctx, filter)
}

func (s *service) GetPlace(ctx context.Context, id int64) (*Place, error) {
	return s.repo.GetPlace(ctx, id)
}

func (s *service) GetCategories(ctx context.Context) ([]Category, error) {
	return s.repo.GetCategories(ctx)
}

func (s *service) GetCategory(ctx context.Context, id int64) (*Category, error) {
	return s.repo.GetCategory(ctx, id)
}

func (s *service) Update(ctx context.Context, id int64, p *Place) (*Place, error) {
	if err := s.validate(ctx, p); err != nil {
		return nil, err
	}
	p.ID = id

	return s.repo.Update(ctx, p)
}

// Delete also queues the place's photo files for deletion, in the repository.
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

// validate cleans the place's text and checks its location and its attributes
// against the fields of its category.
func (s *service) validate(ctx context.Context, p *Place) error {
	name, err := NormalizeName(p.Name)
	if err != nil {
		return err
	}
	p.Name = name

	if err := NormalizeOptionalFields(p); err != nil {
		return err
	}

	// The location is optional.
	if p.Location != nil && !postgis.InIndonesia(p.Location) {
		return ErrInvalidLocation
	}

	category, err := s.repo.GetCategory(ctx, p.CategoryID)
	if err != nil {
		return err
	}

	attributes, err := validateAttributes(category.Fields, p.Attributes)
	if err != nil {
		return err
	}
	p.Attributes = attributes

	return nil
}

// imagePrefix returns the storage folder of the place's photos.
func imagePrefix(id int64) string {
	return "places/" + strconv.FormatInt(id, 10)
}
