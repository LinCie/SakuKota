package review

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"

	"rantau-go/internal/database"
)

// Repository defines the persistence operations required by the service.
//
//go:generate mockgen -source=repository.go -destination=mock/repository.go -package=mock
type Repository interface {
	// Create stores the review and adds its rating to the place's totals in
	// the same statement. It returns the stored review with its author, or
	// ErrPlaceNotFound or ErrAlreadyReviewed.
	Create(ctx context.Context, r *Review) (*Review, error)
	// GetByPlace returns the place's reviews with their authors, newest
	// first, or ErrPlaceNotFound.
	GetByPlace(ctx context.Context, placeID int64) ([]Review, error)
}

type postgresRepository struct {
	db *sqlx.DB
}

func NewPostgresRepository(db *sqlx.DB) *postgresRepository {
	return &postgresRepository{
		db: db,
	}
}

// -----------------------------------------------------------------------------
// postgresRepository
// -----------------------------------------------------------------------------

// reviewColumns selects a review from `r` and its author from `u`, aliased to
// match the nested Review.User fields.
const reviewColumns = `r.id, r.place_id, r.user_id, r.rating, r.comment, r.created_at,
	u.id AS "user.id", u.name AS "user.name"`

// Create inserts the review and adds its rating to the place's totals. Both
// happen in one statement, so a review that fails to insert leaves the totals
// untouched. The totals are incremented rather than recounted, so concurrent
// reviews of the same place wait on the place's row and none is lost.
func (r *postgresRepository) Create(ctx context.Context, review *Review) (*Review, error) {
	var created Review

	err := r.db.GetContext(
		ctx,
		&created,
		`WITH inserted AS (
			INSERT INTO PLACE_REVIEWS (place_id, user_id, rating, comment)
			VALUES ($1, $2, $3, $4)
			RETURNING id, place_id, user_id, rating, comment, created_at
		), rated AS (
			UPDATE PLACES p
			SET rating_count = p.rating_count + 1, rating_sum = p.rating_sum + i.rating
			FROM inserted i
			WHERE p.id = i.place_id
		)
		SELECT `+reviewColumns+`
		FROM inserted r
		JOIN USERS u ON u.id = r.user_id`,
		review.PlaceID,
		review.UserID,
		review.Rating,
		review.Comment,
	)
	if err != nil {
		return nil, mapCreateError(err)
	}

	return &created, nil
}

// GetByPlace returns the place's reviews, newest first, or ErrPlaceNotFound.
// Reviews created at the same moment list the later one first.
func (r *postgresRepository) GetByPlace(ctx context.Context, placeID int64) ([]Review, error) {
	var exists bool
	err := r.db.GetContext(
		ctx,
		&exists,
		`SELECT EXISTS (SELECT 1 FROM PLACES WHERE id = $1)`,
		placeID,
	)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrPlaceNotFound
	}

	reviews := make([]Review, 0)

	err = r.db.SelectContext(
		ctx,
		&reviews,
		`SELECT `+reviewColumns+`
		FROM PLACE_REVIEWS r
		JOIN USERS u ON u.id = r.user_id
		WHERE r.place_id = $1
		ORDER BY r.created_at DESC, r.id DESC`,
		placeID,
	)
	if err != nil {
		return nil, err
	}

	return reviews, nil
}

// mapCreateError translates a second review by the same user into
// ErrAlreadyReviewed and a missing place into ErrPlaceNotFound. Other errors
// are returned unchanged.
func mapCreateError(err error) error {
	if database.IsCode(err, database.UniqueViolation) {
		return ErrAlreadyReviewed
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == database.ForeignKeyViolation &&
		pgErr.ConstraintName == "place_reviews_place_id_fkey" {
		return ErrPlaceNotFound
	}

	return err
}
