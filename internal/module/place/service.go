package place

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrInvalidName         = errors.New("name is required")
	ErrSubDistrictNotFound = errors.New("sub-district not found")
	ErrCategoryNotFound    = errors.New("category not found")
)

// Repository defines the persistence operations required by the service.
type Repository interface {
	GetMany(ctx context.Context, filter PlaceFilter) ([]Place, error)
	GetCategories(ctx context.Context) ([]Category, error)
	Create(ctx context.Context, p *Place, categoryIDs []int64) (*Place, error)
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

func (s *service) GetMany(ctx context.Context, filter PlaceFilter) ([]Place, error) {
	return s.repo.GetMany(ctx, filter)
}

func (s *service) GetCategories(ctx context.Context) ([]Category, error) {
	return s.repo.GetCategories(ctx)
}

func (s *service) Create(ctx context.Context, p *Place, categoryIDs []int64) (*Place, error) {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return nil, ErrInvalidName
	}

	return s.repo.Create(ctx, p, categoryIDs)
}
