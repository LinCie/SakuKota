package subdistrict

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrInvalidName       = errors.New("name is required")
	ErrInvalidSlug       = errors.New("slug is required")
	ErrSubDistrictExists = errors.New("sub-district with this name or slug already exists")
)

// Repository defines the persistence operations required by the service.
type Repository interface {
	GetMany(ctx context.Context) ([]SubDistrict, error)
	Create(ctx context.Context, sd *SubDistrict) (*SubDistrict, error)
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

func (s *service) Create(ctx context.Context, sd *SubDistrict) (*SubDistrict, error) {
	sd.Name = strings.TrimSpace(sd.Name)
	if sd.Name == "" {
		return nil, ErrInvalidName
	}

	sd.Slug = strings.TrimSpace(sd.Slug)
	if sd.Slug == "" {
		return nil, ErrInvalidSlug
	}

	return s.repo.Create(ctx, sd)
}
