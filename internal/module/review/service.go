package review

import (
	"context"

	"rantau-go/internal/pkg/text"
)

// Service defines the operations required by the handler.
//
//go:generate mockgen -source=service.go -destination=mock/service.go -package=mock
type Service interface {
	// Create checks the rating (1 to 5) and the comment (at most 1000 characters),
	// then saves the review. A blank comment is saved as nothing.
	Create(ctx context.Context, r *Review) (*Review, error)
	// GetByPlace returns the place's reviews, newest first.
	GetByPlace(ctx context.Context, placeID int64) ([]Review, error)
}

type service struct {
	repo Repository
}

// NewService builds the review service on top of the repository.
func NewService(repo Repository) Service {
	return &service{
		repo: repo,
	}
}

// -----------------------------------------------------------------------------
// service
// -----------------------------------------------------------------------------

// Create checks the rating, trims the comment, then saves the review.
func (s *service) Create(ctx context.Context, r *Review) (*Review, error) {
	if r.Rating < minRating || r.Rating > maxRating {
		return nil, ErrInvalidRating
	}

	comment, ok := text.TrimOptionalMax(r.Comment, maxCommentLength)
	if !ok {
		return nil, ErrCommentTooLong
	}
	r.Comment = comment

	return s.repo.Create(ctx, r)
}

// GetByPlace returns the reviews of the place, newest first.
func (s *service) GetByPlace(ctx context.Context, placeID int64) ([]Review, error) {
	return s.repo.GetByPlace(ctx, placeID)
}
