package users

import (
	"context"

	"rantau-go/internal/domain"
)

//go:generate mockgen -source=service.go -destination=mock/service.go -package=mock

// Repository is the persistence the service needs.
type Repository interface {
	// Create saves a new user and returns it with its generated fields, or
	// domain.ErrEmailTaken.
	Create(ctx context.Context, u *domain.User) (*domain.User, error)
	// FetchByEmail returns the user with the email, or domain.ErrUserNotFound.
	FetchByEmail(ctx context.Context, email string) (*domain.User, error)
	// Fetch returns the user with the id, or domain.ErrUserNotFound.
	Fetch(ctx context.Context, id int64) (*domain.User, error)
}

// service provides the user operations used by other modules.
type service struct {
	repo Repository
}

// NewService builds a service over repo.
func NewService(repo Repository) *service {
	return &service{repo: repo}
}

// Create saves a new user, or returns domain.ErrEmailTaken.
func (s *service) Create(ctx context.Context, u *domain.User) (*domain.User, error) {
	return s.repo.Create(ctx, u)
}

// FetchByEmail returns the user with the email, or domain.ErrUserNotFound.
func (s *service) FetchByEmail(ctx context.Context, email string) (*domain.User, error) {
	return s.repo.FetchByEmail(ctx, email)
}

// Fetch returns the user with the id, or domain.ErrUserNotFound.
func (s *service) Fetch(ctx context.Context, id int64) (*domain.User, error) {
	return s.repo.Fetch(ctx, id)
}
