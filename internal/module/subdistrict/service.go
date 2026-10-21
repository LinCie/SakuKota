package subdistrict

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"
)

var (
	ErrInvalidName                   = errors.New("name must be 1-100 characters")
	ErrInvalidSlug                   = errors.New("slug must be 1-100 characters")
	ErrSubDistrictExists             = errors.New("sub-district with this name or slug already exists")
	ErrSubDistrictNotFound           = errors.New("sub-district not found")
	ErrSubDistrictInUse              = errors.New("sub-district is referenced by places")
	ErrInvalidEstimatedDailySpending = errors.New("estimated_daily_spending must be between 0 and 99999999.99")
	ErrInvalidStaticMapImageURL      = errors.New("static_map_image_url must be 255 characters or fewer")
)

const (
	// maxNameLength is the size of the VARCHAR(100) name and slug columns.
	// VARCHAR counts characters, not bytes.
	maxNameLength = 100

	// maxStaticMapImageURLLength is the size of the VARCHAR(255) URL column.
	maxStaticMapImageURLLength = 255

	// maxEstimatedDailySpending is the largest value a DECIMAL(10,2) column holds.
	maxEstimatedDailySpending = 99999999.99
)

// Service defines the operations required by the handler.
//
//go:generate mockgen -source=service.go -destination=mock/service.go -package=mock
type Service interface {
	GetMany(ctx context.Context) ([]SubDistrict, error)
	GetSubDistrict(ctx context.Context, id int64) (*SubDistrict, error)
	Create(ctx context.Context, sd *SubDistrict) (*SubDistrict, error)
	Update(ctx context.Context, id int64, patch SubDistrictPatch) (*SubDistrict, error)
	Delete(ctx context.Context, id int64) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{
		repo: repo,
	}
}

// -----------------------------------------------------------------------------
// service
// -----------------------------------------------------------------------------

func (s *service) GetMany(ctx context.Context) ([]SubDistrict, error) {
	return s.repo.GetMany(ctx)
}

func (s *service) GetSubDistrict(ctx context.Context, id int64) (*SubDistrict, error) {
	return s.repo.GetSubDistrict(ctx, id)
}

func (s *service) Create(ctx context.Context, sd *SubDistrict) (*SubDistrict, error) {
	if err := validate(sd); err != nil {
		return nil, err
	}

	return s.repo.Create(ctx, sd)
}

// Update applies the patch to the stored sub-district. The result is validated
// as a whole, so a patch cannot leave the sub-district invalid.
func (s *service) Update(ctx context.Context, id int64, patch SubDistrictPatch) (*SubDistrict, error) {
	sd, err := s.repo.GetSubDistrict(ctx, id)
	if err != nil {
		return nil, err
	}

	if patch.Name.Set && patch.Name.Value != nil {
		sd.Name = *patch.Name.Value
	}

	if patch.Slug.Set && patch.Slug.Value != nil {
		sd.Slug = *patch.Slug.Value
	}

	if patch.FloodHistory.Set {
		sd.FloodHistory = patch.FloodHistory.Value
	}

	if patch.SafetyCrimeInformation.Set {
		sd.SafetyCrimeInformation = patch.SafetyCrimeInformation.Value
	}

	if patch.EstimatedDailySpending.Set {
		sd.EstimatedDailySpending = patch.EstimatedDailySpending.Value
	}

	if patch.StaticMapImageURL.Set {
		sd.StaticMapImageURL = patch.StaticMapImageURL.Value
	}

	if err := validate(sd); err != nil {
		return nil, err
	}

	return s.repo.Update(ctx, sd)
}

func (s *service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

// -----------------------------------------------------------------------------
// validation
// -----------------------------------------------------------------------------

// validate trims the name and slug, then checks every field against its limits.
func validate(sd *SubDistrict) error {
	sd.Name = strings.TrimSpace(sd.Name)
	if n := utf8.RuneCountInString(sd.Name); n == 0 || n > maxNameLength {
		return ErrInvalidName
	}

	sd.Slug = strings.TrimSpace(sd.Slug)
	if n := utf8.RuneCountInString(sd.Slug); n == 0 || n > maxNameLength {
		return ErrInvalidSlug
	}

	if v := sd.EstimatedDailySpending; v != nil && (*v < 0 || *v > maxEstimatedDailySpending) {
		return ErrInvalidEstimatedDailySpending
	}

	if u := sd.StaticMapImageURL; u != nil && utf8.RuneCountInString(*u) > maxStaticMapImageURLLength {
		return ErrInvalidStaticMapImageURL
	}

	return nil
}
