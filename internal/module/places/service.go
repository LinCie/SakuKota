package places

import (
	"context"
	"strconv"

	"rantau-go/internal/domain"
	"rantau-go/internal/pkg/postgis"
	"rantau-go/internal/storage"
)

//go:generate mockgen -source=service.go -destination=mock/service.go -package=mock

// Repository is the persistence the service needs.
type Repository interface {
	// Create saves the place and returns it with its category and photos.
	Create(ctx context.Context, p *domain.Place) (*domain.Place, error)
	// AddMedia stores the photo in the place's first free position and returns the place.
	// It returns domain.ErrPlaceNotFound, or domain.ErrMediaLimitReached if the place is full.
	AddMedia(ctx context.Context, id int64, m domain.Media) (*domain.Place, error)
	// List returns one page of places matching the filter, plus the total match count.
	List(ctx context.Context, filter Filter) ([]domain.Place, int, error)
	// ListLocations returns the id, category, and coordinates of every place matching the
	// filter's sub-district, categories, and query. Bounds, limit, and offset are ignored.
	ListLocations(ctx context.Context, filter Filter) ([]Location, error)
	// ListByIDs returns the places with the given ids, in the order of ids.
	// Ids that match no place are skipped.
	ListByIDs(ctx context.Context, ids []int64) ([]domain.Place, error)
	// ListCategories returns every category with its fields.
	ListCategories(ctx context.Context) ([]domain.Category, error)
	// FetchCategory returns the category with its fields.
	FetchCategory(ctx context.Context, id int64) (*domain.Category, error)
	// Fetch returns the place with its category and photos.
	Fetch(ctx context.Context, id int64) (*domain.Place, error)
	// Update overwrites the place and returns it with its category and photos.
	Update(ctx context.Context, p *domain.Place) (*domain.Place, error)
	// Delete removes the place and its photos and queues the photo files for deletion.
	// It returns domain.ErrPlaceNotFound if there is no such place.
	Delete(ctx context.Context, id int64) error
	// DeleteMedia removes the photo and queues its file for deletion.
	// It returns domain.ErrMediaNotFound if there is no such photo.
	DeleteMedia(ctx context.Context, id, mediaID int64) error
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

// Service validates places and coordinates their storage. It is safe for
// concurrent use if its repository and media storage are.
type Service struct {
	repo  Repository
	media MediaStorage
}

// NewService returns a Service that stores places in repo and photos in media.
func NewService(repo Repository, media MediaStorage) *Service {
	return &Service{
		repo:  repo,
		media: media,
	}
}

// Create validates and saves a new place.
func (s *Service) Create(ctx context.Context, p *domain.Place) (*domain.Place, error) {
	if err := s.validate(ctx, p); err != nil {
		return nil, err
	}

	return s.repo.Create(ctx, p)
}

// MediaUploadURL returns a temporary link to upload a photo for the place.
func (s *Service) MediaUploadURL(ctx context.Context, id int64, contentType string) (storage.Upload, error) {
	if _, err := s.repo.Fetch(ctx, id); err != nil {
		return storage.Upload{}, err
	}

	return s.media.PresignUpload(ctx, mediaPrefix(id), contentType)
}

// AddMedia attaches an uploaded photo to the place and returns the place.
func (s *Service) AddMedia(ctx context.Context, id int64, uploadKey string) (*domain.Place, error) {
	if _, err := s.repo.Fetch(ctx, id); err != nil {
		return nil, err
	}

	promoted, err := s.media.Promote(ctx, mediaPrefix(id), uploadKey)
	if err != nil {
		return nil, err
	}

	updated, err := s.repo.AddMedia(ctx, id, promoted)
	if err != nil {
		// Attaching failed (e.g. the place is full), so remove the file just moved.
		// WithoutCancel: the cleanup must run even if the request was cancelled.
		_ = s.media.Delete(context.WithoutCancel(ctx), promoted.Key)
		return nil, err
	}

	return updated, nil
}

// List returns one page of places matching the filter, plus the total match
// count. It caps the page size so no list is unbounded.
func (s *Service) List(ctx context.Context, filter Filter) ([]domain.Place, int, error) {
	if filter.Limit <= 0 || filter.Limit > maxLimit {
		filter.Limit = maxLimit
	}

	return s.repo.List(ctx, filter)
}

// Fetch returns the place with the given id.
func (s *Service) Fetch(ctx context.Context, id int64) (*domain.Place, error) {
	return s.repo.Fetch(ctx, id)
}

// ListCategories returns every category with its fields.
func (s *Service) ListCategories(ctx context.Context) ([]domain.Category, error) {
	return s.repo.ListCategories(ctx)
}

// FetchCategory returns the category with the given id.
func (s *Service) FetchCategory(ctx context.Context, id int64) (*domain.Category, error) {
	return s.repo.FetchCategory(ctx, id)
}

// Update validates p and overwrites the place with the given id.
func (s *Service) Update(ctx context.Context, id int64, p *domain.Place) (*domain.Place, error) {
	if err := s.validate(ctx, p); err != nil {
		return nil, err
	}
	p.ID = id

	return s.repo.Update(ctx, p)
}

// Delete removes the place with the given id. The repository also queues the
// place's photo files for deletion.
func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

// DeleteMedia removes a photo from the place. The repository also queues the
// photo file for deletion.
func (s *Service) DeleteMedia(ctx context.Context, id, mediaID int64) error {
	return s.repo.DeleteMedia(ctx, id, mediaID)
}

// validate cleans the place's text and checks its location and its attributes
// against the fields of its category.
func (s *Service) validate(ctx context.Context, p *domain.Place) error {
	name, err := domain.NormalizePlaceName(p.Name)
	if err != nil {
		return err
	}
	p.Name = name

	if err := domain.NormalizePlaceFields(p); err != nil {
		return err
	}

	// The location is optional.
	if p.Location != nil && !postgis.InIndonesia(p.Location) {
		return domain.ErrPlaceLocationInvalid
	}

	category, err := s.repo.FetchCategory(ctx, p.CategoryID)
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

// mediaPrefix returns the storage folder of the place's medias.
func mediaPrefix(id int64) string {
	return "places/" + strconv.FormatInt(id, 10)
}
