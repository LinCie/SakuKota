package users

import (
	"context"

	"rantau-go/internal/domain"
)

//go:generate mockgen -source=service.go -destination=mock/service.go -package=mock

// Repository is the persistence the Service needs.
type Repository interface {
	// Create saves a new user and returns it with its generated fields, or
	// domain.ErrEmailTaken.
	Create(ctx context.Context, u *domain.User) (*domain.User, error)
	// FetchByEmail returns the user with the email, or domain.ErrUserNotFound.
	FetchByEmail(ctx context.Context, email string) (*domain.User, error)
	// Fetch returns the user with the id, or domain.ErrUserNotFound.
	Fetch(ctx context.Context, id int64) (*domain.User, error)
}

// Service provides the user operations used by other modules.
type Service struct {
	repo Repository
}

// NewService builds a Service over repo.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Create saves a new user, or returns domain.ErrEmailTaken.
func (s *Service) Create(ctx context.Context, u *domain.User) (*domain.User, error) {
	return s.repo.Create(ctx, u)
}

// FetchByEmail returns the user with the email, or domain.ErrUserNotFound.
func (s *Service) FetchByEmail(ctx context.Context, email string) (*domain.User, error) {
	return s.repo.FetchByEmail(ctx, email)
}

// Fetch returns the user with the id, or domain.ErrUserNotFound.
func (s *Service) Fetch(ctx context.Context, id int64) (*domain.User, error) {
	return s.repo.Fetch(ctx, id)
}
