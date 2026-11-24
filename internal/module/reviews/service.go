package reviews

import (
	"context"
	"strconv"

	"sakukota-go/internal/domain"
	"sakukota-go/internal/pkg/text"
	"sakukota-go/internal/storage"
)

//go:generate mockgen -source=service.go -destination=mock/service.go -package=mock

// Repository is the persistence the service needs.
type Repository interface {
	// Create saves the review. It returns the saved review with its author,
	// or domain.ErrPlaceNotFound or domain.ErrPlaceAlreadyReviewed.
	Create(ctx context.Context, r *domain.Review) (*domain.Review, error)
	// Fetch returns the user's review of the place with its author and medias,
	// or domain.ErrReviewNotFound.
	Fetch(ctx context.Context, placeID, userID int64) (*domain.Review, error)
	// FetchByID returns the place's review with its author and medias, or
	// domain.ErrReviewNotFound.
	FetchByID(ctx context.Context, placeID, reviewID int64) (*domain.Review, error)
	// AddMedia stores the photo in the first free position of the user's review
	// of the place and returns the review. It returns domain.ErrReviewNotFound,
	// or domain.ErrMediaLimitReached if the review is full.
	AddMedia(ctx context.Context, placeID, userID int64, m domain.Media) (*domain.Review, error)
	// ListByPlace returns the place's reviews with their authors and medias,
	// newest first, or domain.ErrPlaceNotFound.
	ListByPlace(ctx context.Context, placeID int64) ([]domain.Review, error)
	// ListByUser returns the user's reviews with their authors and medias,
	// newest first.
	ListByUser(ctx context.Context, userID int64) ([]domain.Review, error)
	// DeleteMedia removes the photo from the user's review of the place and
	// queues its file for deletion. It returns domain.ErrMediaNotFound if
	// there is no such photo.
	DeleteMedia(ctx context.Context, placeID, userID, mediaID int64) error
	// Delete removes the place's review and its photos, queues the photo files
	// for deletion, and returns the review's rating. It returns
	// domain.ErrReviewNotFound if the place has no such review.
	Delete(ctx context.Context, placeID, reviewID int64) (int, error)
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
	// RemoveRating takes one review rating off the place's rating totals.
	RemoveRating(ctx context.Context, placeID int64, rating int) error
}

// MediaStorage is the file storage the service needs.
type MediaStorage interface {
	// PresignUpload returns a temporary link for uploading a file under prefix.
	PresignUpload(ctx context.Context, prefix, contentType string) (storage.Upload, error)
	// Promote moves the uploaded file under uploadKey into prefix and describes it as a photo.
	Promote(ctx context.Context, prefix, uploadKey string) (domain.Media, error)
	// Delete removes the file stored under key.
	Delete(ctx context.Context, key string) error
	// PublicURL returns the link a browser can use to open the file stored under key.
	PublicURL(key string) string
}

// service holds the business rules of reviews.
type service struct {
	repo   Repository
	tx     Transactor
	places PlaceService
	media  MediaStorage
}

// NewService returns a service that stores reviews in repo and adds their
// ratings through places, both in one transaction run by tx. Review photos
// are kept in media.
func NewService(repo Repository, tx Transactor, places PlaceService, media MediaStorage) Service {
	return &service{
		repo:   repo,
		tx:     tx,
		places: places,
		media:  media,
	}
}

// Create checks the rating (domain.MinRating to domain.MaxRating) and the
// comment (at most domain.MaxReviewCommentLength characters), then saves the
// review and adds its rating to the place's totals in one transaction. A blank
// comment is saved as nothing.
func (s *service) Create(ctx context.Context, r *domain.Review) (*domain.Review, error) {
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

// MediaUploadURL returns a temporary link to upload a photo for the user's
// review of the place.
func (s *service) MediaUploadURL(ctx context.Context, placeID, userID int64, contentType string) (storage.Upload, error) {
	if _, err := s.repo.Fetch(ctx, placeID, userID); err != nil {
		return storage.Upload{}, err
	}

	return s.media.PresignUpload(ctx, mediaPrefix(placeID, userID), contentType)
}

// AddMedia attaches an uploaded photo to the user's review of the place and
// returns the review.
func (s *service) AddMedia(ctx context.Context, placeID, userID int64, uploadKey string) (*domain.Review, error) {
	if _, err := s.repo.Fetch(ctx, placeID, userID); err != nil {
		return nil, err
	}

	promoted, err := s.media.Promote(ctx, mediaPrefix(placeID, userID), uploadKey)
	if err != nil {
		return nil, err
	}

	updated, err := s.repo.AddMedia(ctx, placeID, userID, promoted)
	if err != nil {
		// Attaching failed (e.g. the review is full), so remove the file just moved.
		// WithoutCancel: the cleanup must run even if the request was cancelled.
		_ = s.media.Delete(context.WithoutCancel(ctx), promoted.Key)
		return nil, err
	}

	return updated, nil
}

// ListByPlace returns the place's reviews, newest first.
func (s *service) ListByPlace(ctx context.Context, placeID int64) ([]domain.Review, error) {
	return s.repo.ListByPlace(ctx, placeID)
}

// ListByUser returns the user's reviews, newest first.
func (s *service) ListByUser(ctx context.Context, userID int64) ([]domain.Review, error) {
	return s.repo.ListByUser(ctx, userID)
}

// FetchByID returns the place's review with its author and medias, or
// domain.ErrReviewNotFound.
func (s *service) FetchByID(ctx context.Context, placeID, reviewID int64) (*domain.Review, error) {
	return s.repo.FetchByID(ctx, placeID, reviewID)
}

// DeleteMedia removes a photo from the user's review of the place. The
// repository also queues the photo file for deletion.
func (s *service) DeleteMedia(ctx context.Context, placeID, userID, mediaID int64) error {
	return s.repo.DeleteMedia(ctx, placeID, userID, mediaID)
}

// Delete removes the place's review and its photos, and takes its rating off
// the place's totals in one transaction. It returns domain.ErrReviewNotFound
// if the place has no such review.
func (s *service) Delete(ctx context.Context, placeID, reviewID int64) error {
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		rating, err := s.repo.Delete(ctx, placeID, reviewID)
		if err != nil {
			return err
		}
		return s.places.RemoveRating(ctx, placeID, rating)
	})
}

// DeleteMine deletes the user's review of the place as Delete does. It returns
// domain.ErrReviewNotFound if the user has not reviewed the place.
func (s *service) DeleteMine(ctx context.Context, placeID, userID int64) error {
	review, err := s.repo.Fetch(ctx, placeID, userID)
	if err != nil {
		return err
	}

	return s.Delete(ctx, placeID, review.ID)
}

// mediaPrefix returns the storage folder of the photos of the user's review of
// the place. A user reviews a place once, so the pair names one review.
func mediaPrefix(placeID, userID int64) string {
	return "reviews/" + strconv.FormatInt(placeID, 10) + "/" + strconv.FormatInt(userID, 10)
}
