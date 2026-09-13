package subdistrict

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

func (s *Service) GetMany(ctx context.Context) ([]SubDistrict, error) {
	return s.repo.GetMany(ctx)
}
