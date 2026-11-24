package reviews

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"sakukota-go/internal/database"
	"sakukota-go/internal/domain"
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
		`+reviewJoins,
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

// Fetch returns the user's review of the place with its author and medias,
// or domain.ErrReviewNotFound.
func (r *PostgresRepository) Fetch(ctx context.Context, placeID, userID int64) (*domain.Review, error) {
	return r.fetch(ctx, `r.place_id = $1 AND r.user_id = $2`, placeID, userID)
}

// FetchByID returns the place's review with its author and medias, or
// domain.ErrReviewNotFound.
func (r *PostgresRepository) FetchByID(ctx context.Context, placeID, reviewID int64) (*domain.Review, error) {
	return r.fetch(ctx, `r.place_id = $1 AND r.id = $2`, placeID, reviewID)
}

// fetch returns the one review matching where with its author and medias, or
// domain.ErrReviewNotFound.
func (r *PostgresRepository) fetch(ctx context.Context, where string, args ...any) (*domain.Review, error) {
	var review domain.Review

	err := r.db.GetContext(
		ctx,
		&review,
		`SELECT `+reviewColumns+`
		FROM PLACE_REVIEWS r
		`+reviewJoins+`
		WHERE `+where,
		args...,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrReviewNotFound
	}
	if err != nil {
		return nil, err
	}

	reviews := []domain.Review{review}
	if err := r.attachMedias(ctx, reviews); err != nil {
		return nil, err
	}

	return &reviews[0], nil
}

// AddMedia stores the photo in the first free position of the user's review of
// the place and returns the review. The review row is locked so concurrent
// uploads cannot take the same position or exceed the limit.
func (r *PostgresRepository) AddMedia(ctx context.Context, placeID, userID int64, m domain.Media) (*domain.Review, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	reviewID, err := lockReview(ctx, tx, placeID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrReviewNotFound
	}
	if err != nil {
		return nil, err
	}

	// The lowest unused position; NULL means every position is taken.
	var position *int
	err = tx.GetContext(
		ctx,
		&position,
		`SELECT MIN(free) FROM generate_series(1, $2::int) AS free
		WHERE free NOT IN (SELECT position FROM PLACE_REVIEW_MEDIAS WHERE review_id = $1)`,
		reviewID,
		domain.MaxMedias,
	)
	if err != nil {
		return nil, err
	}
	if position == nil {
		return nil, domain.ErrMediaLimitReached
	}

	_, err = tx.ExecContext(
		ctx,
		`WITH media AS (
			INSERT INTO MEDIAS (key, content_type, size_bytes)
			VALUES ($2, $3, $4)
			RETURNING id
		)
		INSERT INTO PLACE_REVIEW_MEDIAS (review_id, media_id, position)
		SELECT $1, id, $5 FROM media`,
		reviewID,
		m.Key,
		m.ContentType,
		m.Size,
		*position,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return r.Fetch(ctx, placeID, userID)
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
		`+reviewJoins+`
		WHERE r.place_id = $1
		ORDER BY r.created_at DESC, r.id DESC`,
		placeID,
	)
	if err != nil {
		return nil, err
	}

	if err := r.attachMedias(ctx, reviews); err != nil {
		return nil, err
	}

	return reviews, nil
}

// ListByUser returns the user's reviews, newest first. Reviews created at the
// same moment list the later one first.
func (r *PostgresRepository) ListByUser(ctx context.Context, userID int64) ([]domain.Review, error) {
	reviews := make([]domain.Review, 0)

	err := r.db.SelectContext(
		ctx,
		&reviews,
		`SELECT `+reviewColumns+`
		FROM PLACE_REVIEWS r
		`+reviewJoins+`
		WHERE r.user_id = $1
		ORDER BY r.created_at DESC, r.id DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}

	if err := r.attachMedias(ctx, reviews); err != nil {
		return nil, err
	}

	return reviews, nil
}

// attachMedias loads the medias of all reviews in one query, in position order.
func (r *PostgresRepository) attachMedias(ctx context.Context, reviews []domain.Review) error {
	if len(reviews) == 0 {
		return nil
	}

	reviewIDs := make([]int64, 0, len(reviews))
	for _, review := range reviews {
		reviewIDs = append(reviewIDs, review.ID)
	}

	var rows []struct {
		ReviewID int64 `db:"review_id"`
		domain.Media
	}

	err := r.db.SelectContext(
		ctx,
		&rows,
		`SELECT rm.review_id, m.id, m.key, m.content_type, m.size_bytes, rm.position
		FROM PLACE_REVIEW_MEDIAS rm
		JOIN MEDIAS m ON m.id = rm.media_id
		WHERE rm.review_id = ANY($1)
		ORDER BY rm.review_id, rm.position`,
		reviewIDs,
	)
	if err != nil {
		return err
	}

	mediasByReview := make(map[int64][]domain.Media, len(reviews))
	for _, row := range rows {
		mediasByReview[row.ReviewID] = append(mediasByReview[row.ReviewID], row.Media)
	}

	for i := range reviews {
		reviews[i].Medias = mediasByReview[reviews[i].ID]
	}

	return nil
}

// DeleteMedia removes the photo from the user's review of the place, queues its
// file for deletion, and moves later photos up one position. The shift closes
// the gap, so the next upload goes to the end.
func (r *PostgresRepository) DeleteMedia(ctx context.Context, placeID, userID, mediaID int64) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	reviewID, err := lockReview(ctx, tx, placeID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrMediaNotFound
	}
	if err != nil {
		return err
	}

	// The file is queued in the same statement as the row delete, so both commit or neither does.
	// storagecleanup then retries the file delete until it succeeds.
	var position *int
	err = tx.GetContext(
		ctx,
		&position,
		`WITH deleted AS (
			DELETE FROM MEDIAS m
			USING PLACE_REVIEW_MEDIAS rm
			WHERE rm.media_id = m.id AND rm.review_id = $1 AND m.id = $2
			RETURNING m.key, rm.position
		), queued AS (
			INSERT INTO STORAGE_DELETIONS (key)
			SELECT key FROM deleted
			ON CONFLICT (key) DO NOTHING
		)
		SELECT MAX(position) FROM deleted`,
		reviewID,
		mediaID,
	)
	if err != nil {
		return err
	}
	// No row was deleted: the review has no such photo.
	if position == nil {
		return domain.ErrMediaNotFound
	}

	_, err = tx.ExecContext(
		ctx,
		`UPDATE PLACE_REVIEW_MEDIAS SET position = position - 1 WHERE review_id = $1 AND position > $2`,
		reviewID,
		*position,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// Delete removes the place's review with the given id and its medias, queues
// the media files for deletion, and returns the review's rating. It joins the
// transaction carried by ctx if there is one. It returns
// domain.ErrReviewNotFound if the place has no such review.
func (r *PostgresRepository) Delete(ctx context.Context, placeID, reviewID int64) (int, error) {
	var rating int
	err := database.NewTransactor(r.db).WithinTx(ctx, func(ctx context.Context) error {
		conn := database.Conn(ctx, r.db)

		// Lock first so the delete below starts after any media being added has
		// committed, and sees it; otherwise its file would never be queued.
		var id int64
		err := conn.GetContext(ctx, &id, `SELECT id FROM PLACE_REVIEWS WHERE id = $1 AND place_id = $2 FOR UPDATE`, reviewID, placeID)
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrReviewNotFound
		}
		if err != nil {
			return err
		}

		// The files are queued in the same statement as the row deletes, so both commit or neither does.
		return conn.GetContext(
			ctx,
			&rating,
			`WITH medias AS (
				DELETE FROM MEDIAS
				WHERE id IN (SELECT media_id FROM PLACE_REVIEW_MEDIAS WHERE review_id = $1)
				RETURNING key
			), queued AS (
				INSERT INTO STORAGE_DELETIONS (key)
				SELECT key FROM medias
				ON CONFLICT (key) DO NOTHING
			)
			DELETE FROM PLACE_REVIEWS WHERE id = $1
			RETURNING rating`,
			id,
		)
	})

	return rating, err
}

// lockReview locks the user's review of the place in tx and returns its id,
// or sql.ErrNoRows if there is none.
func lockReview(ctx context.Context, tx *sqlx.Tx, placeID, userID int64) (int64, error) {
	var id int64
	// FOR NO KEY UPDATE: only non-key data changes, so foreign key inserts elsewhere are not blocked.
	err := tx.GetContext(ctx, &id, `SELECT id FROM PLACE_REVIEWS WHERE place_id = $1 AND user_id = $2 FOR NO KEY UPDATE`, placeID, userID)
	return id, err
}

// reviewColumns selects a review from `r`, its place's name from `p`, and its
// author from `u` with the photo from `a`, aliased to match the nested
// domain.Review.User fields.
const reviewColumns = `r.id, r.place_id, r.user_id, r.rating, r.comment, r.created_at, p.name AS place_name,
	u.id AS "user.id", u.name AS "user.name", a.key AS "user.avatar_key"`

// reviewJoins joins a review `r` to the tables reviewColumns reads.
const reviewJoins = `JOIN PLACES p ON p.id = r.place_id
	JOIN USERS u ON u.id = r.user_id
	LEFT JOIN MEDIAS a ON a.id = u.avatar_media_id`

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
