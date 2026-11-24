package suggestions

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"sakukota-go/internal/database"
	"sakukota-go/internal/domain"
)

// PostgresRepository stores suggestions in PostgreSQL. It is safe for concurrent use.
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository returns a PostgresRepository that uses db.
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{
		db: db,
	}
}

// Create saves the suggestion as pending and returns it with its author.
// It locks the user's row while counting, so two submissions at once cannot both pass the limit.
func (r *PostgresRepository) Create(ctx context.Context, s *domain.Suggestion) (*domain.Suggestion, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `SELECT 1 FROM USERS WHERE id = $1 FOR UPDATE`, s.UserID); err != nil {
		return nil, err
	}

	var pending int
	err = tx.GetContext(
		ctx,
		&pending,
		`SELECT COUNT(*) FROM PLACE_SUGGESTIONS WHERE user_id = $1 AND status = 'pending'`,
		s.UserID,
	)
	if err != nil {
		return nil, err
	}
	if pending >= MaxPendingPerUser {
		return nil, domain.ErrSuggestionLimitReached
	}

	var created domain.Suggestion
	err = tx.GetContext(
		ctx,
		&created,
		`WITH inserted AS (
			INSERT INTO PLACE_SUGGESTIONS (user_id, sub_district_id, category_id, name, address,
				location, operating_hours, price_min, price_max, short_description,
				attributes, note)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			RETURNING *
		)
		SELECT `+suggestionColumns+`
		FROM inserted s
		JOIN USERS u ON u.id = s.user_id`,
		s.UserID,
		s.SubDistrictID,
		s.CategoryID,
		s.Name,
		s.Address,
		s.Location,
		s.OperatingHours,
		s.PriceMin,
		s.PriceMax,
		s.ShortDescription,
		s.Attributes,
		s.Note,
	)
	if err != nil {
		return nil, mapForeignKeyError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &created, nil
}

// Fetch returns the suggestion with its author, or domain.ErrSuggestionNotFound.
func (r *PostgresRepository) Fetch(ctx context.Context, id int64) (*domain.Suggestion, error) {
	var s domain.Suggestion

	err := r.db.GetContext(
		ctx,
		&s,
		`SELECT `+suggestionColumns+`
		`+suggestionFrom+`
		WHERE s.id = $1`,
		id,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrSuggestionNotFound
		}
		return nil, err
	}

	return &s, nil
}

// ListByUser returns the user's suggestions with their author, newest first.
// Suggestions created at the same moment list the later one first.
func (r *PostgresRepository) ListByUser(ctx context.Context, userID int64) ([]domain.Suggestion, error) {
	suggestions := make([]domain.Suggestion, 0)

	err := r.db.SelectContext(
		ctx,
		&suggestions,
		`SELECT `+suggestionColumns+`
		`+suggestionFrom+`
		WHERE s.user_id = $1
		ORDER BY s.created_at DESC, s.id DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}

	return suggestions, nil
}

// List returns the suggestions matching the filter with their authors,
// oldest first. Suggestions created at the same moment list the earlier one first.
func (r *PostgresRepository) List(ctx context.Context, filter Filter) ([]domain.Suggestion, error) {
	query := `SELECT ` + suggestionColumns + `
		` + suggestionFrom

	var args []any
	if filter.Status != "" {
		query += ` WHERE s.status = $1`
		args = append(args, string(filter.Status))
	}
	query += ` ORDER BY s.created_at, s.id`

	suggestions := make([]domain.Suggestion, 0)

	if err := r.db.SelectContext(ctx, &suggestions, query, args...); err != nil {
		return nil, err
	}

	return suggestions, nil
}

// MarkApproved marks the suggestion approved as placeID.
func (r *PostgresRepository) MarkApproved(ctx context.Context, id, placeID, adminID int64) (*domain.Suggestion, error) {
	return r.review(ctx, id, domain.StatusApproved, &placeID, nil, adminID)
}

// Reject marks the suggestion rejected with the reason.
func (r *PostgresRepository) Reject(ctx context.Context, id, adminID int64, reason *string) (*domain.Suggestion, error) {
	return r.review(ctx, id, domain.StatusRejected, nil, reason, adminID)
}

// suggestionColumns selects a suggestion from `s` and its author from `u`,
// aliased to match the nested Suggestion.User fields.
const suggestionColumns = `s.id, s.user_id, s.sub_district_id, s.category_id, s.name, s.address,
	s.location, s.operating_hours, s.price_min, s.price_max, s.short_description,
	s.attributes, s.note, s.status, s.rejection_reason, s.place_id, s.reviewed_by,
	s.reviewed_at, s.created_at, u.id AS "user.id", u.name AS "user.name"`

// suggestionFrom is the FROM clause that reads suggestions from `s` with their authors from `u`.
const suggestionFrom = `FROM PLACE_SUGGESTIONS s JOIN USERS u ON u.id = s.user_id`

// review records the outcome of suggestion id and returns it with its author,
// in the transaction carried by ctx if there is one.
// Only a pending suggestion is updated, so two admins cannot both review it.
func (r *PostgresRepository) review(ctx context.Context, id int64, status domain.Status, placeID *int64, reason *string, adminID int64) (*domain.Suggestion, error) {
	var reviewed domain.Suggestion

	err := database.Conn(ctx, r.db).GetContext(
		ctx,
		&reviewed,
		`WITH updated AS (
			UPDATE PLACE_SUGGESTIONS
			SET status = $2, place_id = $3, rejection_reason = $4,
				reviewed_by = $5, reviewed_at = NOW()
			WHERE id = $1 AND status = 'pending'
			RETURNING *
		)
		SELECT `+suggestionColumns+`
		FROM updated s
		JOIN USERS u ON u.id = s.user_id`,
		id,
		string(status),
		placeID,
		reason,
		adminID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, r.unreviewableError(ctx, id)
		}
		return nil, err
	}

	return &reviewed, nil
}

// unreviewableError explains why suggestion id was not reviewed: it does not
// exist, or it is no longer pending.
func (r *PostgresRepository) unreviewableError(ctx context.Context, id int64) error {
	var exists bool
	err := database.Conn(ctx, r.db).GetContext(
		ctx,
		&exists,
		`SELECT EXISTS (SELECT 1 FROM PLACE_SUGGESTIONS WHERE id = $1)`,
		id,
	)
	if err != nil {
		return err
	}
	if !exists {
		return domain.ErrSuggestionNotFound
	}

	return domain.ErrSuggestionAlreadyReviewed
}

// mapForeignKeyError maps a bad sub-district or category reference to its not-found error.
// Any other error is returned unchanged.
func mapForeignKeyError(err error) error {
	if database.IsCode(err, database.ForeignKeyViolation) {
		switch database.ConstraintName(err) {
		case "place_suggestions_sub_district_id_fkey":
			return domain.ErrSubDistrictNotFound
		case "place_suggestions_category_id_fkey":
			return domain.ErrCategoryNotFound
		}
	}

	return err
}
