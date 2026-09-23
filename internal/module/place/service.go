package place

import "context"

// Repository defines the persistence operations required by the service.
type Repository interface {
	GetMany(ctx context.Context) ([]Place, error)
	GetCategories(ctx context.Context) ([]Category, error)
	GetManyByCategory(ctx context.Context, categoryID int64) ([]Place, error)
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

func (s *service) GetMany(ctx context.Context) ([]Place, error) {
	return s.repo.GetMany(ctx)
}

func (s *service) GetCategories(ctx context.Context) ([]Category, error) {
	return s.repo.GetCategories(ctx)
}

func (s *service) GetManyByCategory(ctx context.Context, categoryID int64) ([]Place, error) {
	return s.repo.GetManyByCategory(ctx, categoryID)
}
