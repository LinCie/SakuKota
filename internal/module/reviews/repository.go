package reviews

import (
	"context"

	"github.com/jmoiron/sqlx"

	"rantau-go/internal/database"
)

// Repository defines the persistence operations required by the service.
//
//go:generate mockgen -source=repository.go -destination=mock/repository.go -package=mock
type Repository interface {
	// Create saves the review and adds its rating to the place's totals.
	// It returns the saved review with its author, or ErrPlaceNotFound or ErrAlreadyReviewed.
	Create(ctx context.Context, r *Review) (*Review, error)
	// GetByPlace returns the place's reviews with their authors, newest
	// first, or ErrPlaceNotFound.
	GetByPlace(ctx context.Context, placeID int64) ([]Review, error)
}

type postgresRepository struct {
	db *sqlx.DB
}

// -----------------------------------------------------------------------------
// Exported
// -----------------------------------------------------------------------------

func NewPostgresRepository(db *sqlx.DB) Repository {
	return &postgresRepository{
		db: db,
	}
}

// Create saves the review and adds its rating to the place's totals in one statement.
// Adding to the totals, not recounting them, means two reviews at once both count.
func (r *postgresRepository) Create(ctx context.Context, review *Review) (*Review, error) {
	var created Review

	err := r.db.GetContext(
		ctx,
		&created,
		`WITH inserted AS (
			INSERT INTO PLACE_REVIEWS (place_id, user_id, rating, comment)
			VALUES ($1, $2, $3, $4)
			RETURNING *
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

// -----------------------------------------------------------------------------
// Unexported
// -----------------------------------------------------------------------------

// reviewColumns selects a review from `r` and its author from `u`, aliased to
// match the nested Review.User fields.
const reviewColumns = `r.id, r.place_id, r.user_id, r.rating, r.comment, r.created_at,
	u.id AS "user.id", u.name AS "user.name"`

// mapCreateError maps a repeat review to ErrAlreadyReviewed and a missing place to ErrPlaceNotFound.
// Any other error is returned unchanged.
func mapCreateError(err error) error {
	if database.IsCode(err, database.UniqueViolation) &&
		database.ConstraintName(err) == "place_reviews_place_id_user_id_key" {
		return ErrAlreadyReviewed
	}

	if database.IsCode(err, database.ForeignKeyViolation) &&
		database.ConstraintName(err) == "place_reviews_place_id_fkey" {
		return ErrPlaceNotFound
	}

	return err
}
