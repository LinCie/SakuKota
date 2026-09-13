package place

import "context"

type Service struct {
	repo Repository
}

var _ Repository = (*Service)(nil)

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// -----------------------------------------------------------------------------
// Service
// -----------------------------------------------------------------------------

func (s *Service) GetMany(ctx context.Context) ([]Place, error) {
	return s.repo.GetMany(ctx)
}

func (s *Service) GetCategories(ctx context.Context) ([]Category, error) {
	return s.repo.GetCategories(ctx)
}

func (s *Service) GetManyByCategory(ctx context.Context, categoryID int64) ([]Place, error) {
	return s.repo.GetManyByCategory(ctx, categoryID)
}
