package place

import "context"

// Repository defines the persistence operations required by the service.
type Repository interface {
	GetMany(ctx context.Context, filter PlaceFilter) ([]Place, error)
	GetCategories(ctx context.Context) ([]Category, error)
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
