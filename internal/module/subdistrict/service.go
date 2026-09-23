package subdistrict

import "context"

// Repository defines the persistence operations required by the service.
type Repository interface {
	GetMany(ctx context.Context) ([]SubDistrict, error)
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
