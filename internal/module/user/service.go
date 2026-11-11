package user

import (
	"context"
)

// Service defines the user operations exposed to other modules.
//
//go:generate mockgen -source=service.go -destination=mock/service.go -package=mock
type Service interface {
	Create(ctx context.Context, u *User) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetUser(ctx context.Context, id int64) (*User, error)
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

func (s *service) Create(ctx context.Context, u *User) (*User, error) {
	return s.repo.Create(ctx, u)
}

func (s *service) GetByEmail(ctx context.Context, email string) (*User, error) {
	return s.repo.GetByEmail(ctx, email)
}

func (s *service) GetUser(ctx context.Context, id int64) (*User, error) {
	return s.repo.GetUser(ctx, id)
}
