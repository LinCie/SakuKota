package suggestion

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"rantau-go/internal/database"
)

// Repository defines the persistence operations required by the service.
//
//go:generate mockgen -source=repository.go -destination=mock/repository.go -package=mock
type Repository interface {
	// Create stores the suggestion as pending and returns it with its author.
	// It returns ErrTooManyPending if the user already has MaxPendingPerUser
	// pending suggestions, or ErrSubDistrictNotFound or ErrCategoryNotFound.
	Create(ctx context.Context, s *Suggestion) (*Suggestion, error)
	// GetSuggestion returns the suggestion with its author, or ErrSuggestionNotFound.
	GetSuggestion(ctx context.Context, id int64) (*Suggestion, error)
	// GetByUser returns the user's suggestions with their author, newest first.
	GetByUser(ctx context.Context, userID int64) ([]Suggestion, error)
	// GetMany returns the suggestions matching the filter with their authors,
	// oldest first.
	GetMany(ctx context.Context, filter SuggestionFilter) ([]Suggestion, error)
	// MarkApproved marks the suggestion approved as the place placeID. It
	// returns ErrSuggestionNotFound, or ErrAlreadyReviewed if the suggestion is
	// not pending.
	MarkApproved(ctx context.Context, id, placeID, adminID int64) (*Suggestion, error)
	// Reject marks the suggestion rejected with the reason. It returns
	// ErrSuggestionNotFound, or ErrAlreadyReviewed if the suggestion is not pending.
	Reject(ctx context.Context, id, adminID int64, reason *string) (*Suggestion, error)
}

type postgresRepository struct {
	db *sqlx.DB
}

func NewPostgresRepository(db *sqlx.DB) Repository {
	return &postgresRepository{
		db: db,
	}
}

// -----------------------------------------------------------------------------
// postgresRepository
// -----------------------------------------------------------------------------

// suggestionColumns selects a suggestion from `s` and its author from `u`,
// aliased to match the nested Suggestion.User fields.
const suggestionColumns = `s.id, s.user_id, s.sub_district_id, s.category_id, s.name, s.address,
	s.location, s.operating_hours, s.estimated_price_range, s.short_description,
	s.attributes, s.note, s.status, s.rejection_reason, s.place_id, s.reviewed_by,
	s.reviewed_at, s.created_at, u.id AS "user.id", u.name AS "user.name"`

// suggestionFrom is the FROM clause that reads suggestions from `s` with their authors from `u`.
const suggestionFrom = `FROM PLACE_SUGGESTIONS s JOIN USERS u ON u.id = s.user_id`

// suggestionRowColumns are the PLACE_SUGGESTIONS columns that suggestionColumns
// reads from `s`. The RETURNING clause of a write lists them so its row can be
// joined with its author.
const suggestionRowColumns = `id, user_id, sub_district_id, category_id, name, address,
	location, operating_hours, estimated_price_range, short_description,
	attributes, note, status, rejection_reason, place_id, reviewed_by,
	reviewed_at, created_at`

// Create stores the suggestion as pending and returns it with its author. The
// user's row is locked while the pending suggestions are counted, so concurrent
// submissions by the same user cannot both pass the limit.
func (r *postgresRepository) Create(ctx context.Context, s *Suggestion) (*Suggestion, error) {
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
		return nil, ErrTooManyPending
	}

	var created Suggestion
	err = tx.GetContext(
		ctx,
		&created,
		`WITH inserted AS (
			INSERT INTO PLACE_SUGGESTIONS (user_id, sub_district_id, category_id, name, address,
				location, operating_hours, estimated_price_range, short_description,
				attributes, note)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			RETURNING `+suggestionRowColumns+`
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
		s.EstimatedPriceRange,
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

// GetSuggestion returns the suggestion with its author, or ErrSuggestionNotFound.
func (r *postgresRepository) GetSuggestion(ctx context.Context, id int64) (*Suggestion, error) {
	var s Suggestion

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
			return nil, ErrSuggestionNotFound
		}
		return nil, err
	}

	return &s, nil
}

// GetByUser returns the user's suggestions with their author, newest first.
// Suggestions created at the same moment list the later one first.
func (r *postgresRepository) GetByUser(ctx context.Context, userID int64) ([]Suggestion, error) {
	suggestions := make([]Suggestion, 0)

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

// GetMany returns the suggestions matching the filter with their authors,
// oldest first. Suggestions created at the same moment list the earlier one first.
func (r *postgresRepository) GetMany(ctx context.Context, filter SuggestionFilter) ([]Suggestion, error) {
	query := `SELECT ` + suggestionColumns + `
		` + suggestionFrom

	var args []any
	if filter.Status != "" {
		query += ` WHERE s.status = $1`
		args = append(args, string(filter.Status))
	}
	query += ` ORDER BY s.created_at, s.id`

	suggestions := make([]Suggestion, 0)

	if err := r.db.SelectContext(ctx, &suggestions, query, args...); err != nil {
		return nil, err
	}

	return suggestions, nil
}

// MarkApproved marks the suggestion approved as placeID.
func (r *postgresRepository) MarkApproved(ctx context.Context, id, placeID, adminID int64) (*Suggestion, error) {
	return r.review(ctx, id, StatusApproved, &placeID, nil, adminID)
}

// Reject marks the suggestion rejected with the reason.
func (r *postgresRepository) Reject(ctx context.Context, id, adminID int64, reason *string) (*Suggestion, error) {
	return r.review(ctx, id, StatusRejected, nil, reason, adminID)
}

// review records the outcome of suggestion id and returns it with its author.
// The update only applies to a pending suggestion, so a suggestion is reviewed
// at most once even when admins act at the same time.
func (r *postgresRepository) review(
	ctx context.Context,
	id int64,
	status Status,
	placeID *int64,
	reason *string,
	adminID int64,
) (*Suggestion, error) {
	var reviewed Suggestion

	err := r.db.GetContext(
		ctx,
		&reviewed,
		`WITH updated AS (
			UPDATE PLACE_SUGGESTIONS
			SET status = $2, place_id = $3, rejection_reason = $4,
				reviewed_by = $5, reviewed_at = NOW()
			WHERE id = $1 AND status = 'pending'
			RETURNING `+suggestionRowColumns+`
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
func (r *postgresRepository) unreviewableError(ctx context.Context, id int64) error {
	var exists bool
	err := r.db.GetContext(
		ctx,
		&exists,
		`SELECT EXISTS (SELECT 1 FROM PLACE_SUGGESTIONS WHERE id = $1)`,
		id,
	)
	if err != nil {
		return err
	}
	if !exists {
		return ErrSuggestionNotFound
	}

	return ErrAlreadyReviewed
}

// mapForeignKeyError translates a foreign key violation on
// PLACE_SUGGESTIONS into the sub-district or category it refers to. Other
// errors are returned unchanged.
func mapForeignKeyError(err error) error {
	if database.IsCode(err, database.ForeignKeyViolation) {
		switch database.ConstraintName(err) {
		case "place_suggestions_sub_district_id_fkey":
			return ErrSubDistrictNotFound
		case "place_suggestions_category_id_fkey":
			return ErrCategoryNotFound
		}
	}

	return err
}
