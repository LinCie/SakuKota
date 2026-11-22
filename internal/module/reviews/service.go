package reviews

import (
	"context"

	"rantau-go/internal/domain"
	"rantau-go/internal/pkg/text"
)

//go:generate mockgen -source=service.go -destination=mock/service.go -package=mock

// Repository is the persistence the Service needs.
type Repository interface {
	// Create saves the review and adds its rating to the place's totals.
	// It returns the saved review with its author, or domain.ErrPlaceNotFound or
	// domain.ErrPlaceAlreadyReviewed.
	Create(ctx context.Context, r *domain.Review) (*domain.Review, error)
	// ListByPlace returns the place's reviews with their authors, newest
	// first, or domain.ErrPlaceNotFound.
	ListByPlace(ctx context.Context, placeID int64) ([]domain.Review, error)
}

// Service holds the business rules of reviews.
type Service struct {
	repo Repository
}

// NewService returns a Service that stores reviews in repo.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Create checks the rating (domain.MinRating to domain.MaxRating) and the
// comment (at most domain.MaxReviewCommentLength characters), then saves the
// review. A blank comment is saved as nothing.
func (s *Service) Create(ctx context.Context, r *domain.Review) (*domain.Review, error) {
	if r.Rating < domain.MinRating || r.Rating > domain.MaxRating {
		return nil, domain.ErrReviewRatingInvalid
	}

	comment, ok := text.TrimOptionalMax(r.Comment, domain.MaxReviewCommentLength)
	if !ok {
		return nil, domain.ErrReviewCommentTooLong
	}
	r.Comment = comment

	return s.repo.Create(ctx, r)
}

// ListByPlace returns the place's reviews, newest first.
func (s *Service) ListByPlace(ctx context.Context, placeID int64) ([]domain.Review, error) {
	return s.repo.ListByPlace(ctx, placeID)
}
