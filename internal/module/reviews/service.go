package reviews

import (
	"context"

	"rantau-go/internal/domain"
	"rantau-go/internal/pkg/text"
)

//go:generate mockgen -source=service.go -destination=mock/service.go -package=mock

// Repository is the persistence the Service needs.
type Repository interface {
	// Create saves the review. It returns the saved review with its author,
	// or domain.ErrPlaceNotFound or domain.ErrPlaceAlreadyReviewed.
	Create(ctx context.Context, r *domain.Review) (*domain.Review, error)
	// ListByPlace returns the place's reviews with their authors, newest
	// first, or domain.ErrPlaceNotFound.
	ListByPlace(ctx context.Context, placeID int64) ([]domain.Review, error)
}

// Transactor runs functions in one database transaction.
type Transactor interface {
	// WithinTx runs fn in a transaction carried by the context passed to fn.
	// It commits if fn returns nil, and otherwise rolls back and returns fn's
	// error.
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// PlaceService is the place operations that reviewing requires.
type PlaceService interface {
	// AddRating adds one review rating to the place's rating totals.
	AddRating(ctx context.Context, placeID int64, rating int) error
}

// Service holds the business rules of reviews.
type Service struct {
	repo   Repository
	tx     Transactor
	places PlaceService
}

// NewService returns a Service that stores reviews in repo and adds their
// ratings through places, both in one transaction run by tx.
func NewService(repo Repository, tx Transactor, places PlaceService) *Service {
	return &Service{
		repo:   repo,
		tx:     tx,
		places: places,
	}
}

// Create checks the rating (domain.MinRating to domain.MaxRating) and the
// comment (at most domain.MaxReviewCommentLength characters), then saves the
// review and adds its rating to the place's totals in one transaction. A blank
// comment is saved as nothing.
func (s *Service) Create(ctx context.Context, r *domain.Review) (*domain.Review, error) {
	if r.Rating < domain.MinRating || r.Rating > domain.MaxRating {
		return nil, domain.ErrReviewRatingInvalid
	}

	comment, ok := text.TrimOptionalMax(r.Comment, domain.MaxReviewCommentLength)
	if !ok {
		return nil, domain.ErrReviewCommentTooLong
	}
	r.Comment = comment

	var created *domain.Review
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		created, err = s.repo.Create(ctx, r)
		if err != nil {
			return err
		}
		return s.places.AddRating(ctx, r.PlaceID, r.Rating)
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

// ListByPlace returns the place's reviews, newest first.
func (s *Service) ListByPlace(ctx context.Context, placeID int64) ([]domain.Review, error) {
	return s.repo.ListByPlace(ctx, placeID)
}
