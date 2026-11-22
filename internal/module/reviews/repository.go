package reviews

import (
	"context"

	"github.com/jmoiron/sqlx"

	"rantau-go/internal/database"
	"rantau-go/internal/domain"
)

// PostgresRepository stores reviews in PostgreSQL. It is safe for concurrent use.
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository returns a PostgresRepository that uses db.
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Create saves the review, in the transaction carried by ctx if there is one.
func (r *PostgresRepository) Create(ctx context.Context, review *domain.Review) (*domain.Review, error) {
	var created domain.Review

	err := database.Conn(ctx, r.db).GetContext(
		ctx,
		&created,
		`WITH inserted AS (
			INSERT INTO PLACE_REVIEWS (place_id, user_id, rating, comment)
			VALUES ($1, $2, $3, $4)
			RETURNING *
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

// ListByPlace returns the place's reviews, newest first, or domain.ErrPlaceNotFound.
// Reviews created at the same moment list the later one first.
func (r *PostgresRepository) ListByPlace(ctx context.Context, placeID int64) ([]domain.Review, error) {
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
		return nil, domain.ErrPlaceNotFound
	}

	reviews := make([]domain.Review, 0)

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

// reviewColumns selects a review from `r` and its author from `u`, aliased to
// match the nested domain.Review.User fields.
const reviewColumns = `r.id, r.place_id, r.user_id, r.rating, r.comment, r.created_at,
	u.id AS "user.id", u.name AS "user.name"`

// mapCreateError maps a repeat review to domain.ErrPlaceAlreadyReviewed and a missing place to domain.ErrPlaceNotFound.
// Any other error is returned unchanged.
func mapCreateError(err error) error {
	if database.IsCode(err, database.UniqueViolation) && database.ConstraintName(err) == "place_reviews_place_id_user_id_key" {
		return domain.ErrPlaceAlreadyReviewed
	}

	if database.IsCode(err, database.ForeignKeyViolation) && database.ConstraintName(err) == "place_reviews_place_id_fkey" {
		return domain.ErrPlaceNotFound
	}

	return err
}
