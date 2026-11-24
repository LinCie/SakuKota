package reports

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"rantau-go/internal/database"
	"rantau-go/internal/domain"
)

// PostgresRepository stores reports in PostgreSQL. It is safe for concurrent use.
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository returns a PostgresRepository that uses db.
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{
		db: db,
	}
}

// Create saves the report as pending and returns it with its author.
// It locks the user's row while counting, so two reports at once cannot both
// pass the limit.
func (r *PostgresRepository) Create(ctx context.Context, rp *domain.Report) (*domain.Report, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `SELECT 1 FROM USERS WHERE id = $1 FOR UPDATE`, rp.UserID); err != nil {
		return nil, err
	}

	var pending int
	err = tx.GetContext(
		ctx,
		&pending,
		`SELECT COUNT(*) FROM REPORTS WHERE user_id = $1 AND status = 'pending'`,
		rp.UserID,
	)
	if err != nil {
		return nil, err
	}
	if pending >= MaxPendingPerUser {
		return nil, domain.ErrReportLimitReached
	}

	// Inserts nothing unless the place exists and, for a review report, the
	// review belongs to it.
	var created domain.Report
	err = tx.GetContext(
		ctx,
		&created,
		`WITH inserted AS (
			INSERT INTO REPORTS (user_id, place_id, review_id, reason, note)
			SELECT $1, p.id, $3::BIGINT, $4, $5
			FROM PLACES p
			WHERE p.id = $2
				AND ($3::BIGINT IS NULL OR EXISTS (
					SELECT 1 FROM PLACE_REVIEWS pr WHERE pr.id = $3 AND pr.place_id = p.id
				))
			RETURNING *
		)
		SELECT `+reportColumns+`
		FROM inserted rp
		JOIN USERS u ON u.id = rp.user_id`,
		rp.UserID,
		rp.PlaceID,
		rp.ReviewID,
		string(rp.Reason),
		rp.Note,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows) && rp.ReviewID == nil:
			return nil, domain.ErrPlaceNotFound
		case errors.Is(err, sql.ErrNoRows):
			return nil, domain.ErrReviewNotFound
		case database.IsCode(err, database.UniqueViolation):
			return nil, domain.ErrAlreadyReported
		}
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &created, nil
}

// List returns the reports matching the filter with their authors, oldest
// first. Reports created at the same moment list the earlier one first.
func (r *PostgresRepository) List(ctx context.Context, filter Filter) ([]domain.Report, error) {
	query := `SELECT ` + reportColumns + `
		FROM REPORTS rp JOIN USERS u ON u.id = rp.user_id`

	var args []any
	if filter.Status != "" {
		query += ` WHERE rp.status = $1`
		args = append(args, string(filter.Status))
	}
	query += ` ORDER BY rp.created_at, rp.id`

	reports := make([]domain.Report, 0)

	if err := r.db.SelectContext(ctx, &reports, query, args...); err != nil {
		return nil, err
	}

	return reports, nil
}

// Fetch returns the report with its author, or domain.ErrReportNotFound.
func (r *PostgresRepository) Fetch(ctx context.Context, id int64) (*domain.Report, error) {
	var report domain.Report

	err := r.db.GetContext(
		ctx,
		&report,
		`SELECT `+reportColumns+`
		FROM REPORTS rp JOIN USERS u ON u.id = rp.user_id
		WHERE rp.id = $1`,
		id,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrReportNotFound
	}
	if err != nil {
		return nil, err
	}

	return &report, nil
}

// Resolve marks the pending report with status and returns it with its author,
// in the transaction carried by ctx if there is one. Only a pending report is
// updated, so two admins cannot both resolve it.
func (r *PostgresRepository) Resolve(ctx context.Context, id, adminID int64, status domain.ReportStatus) (*domain.Report, error) {
	var resolved domain.Report

	err := database.Conn(ctx, r.db).GetContext(
		ctx,
		&resolved,
		`WITH updated AS (
			UPDATE REPORTS
			SET status = $2, resolved_by = $3, resolved_at = NOW()
			WHERE id = $1 AND status = 'pending'
			RETURNING *
		)
		SELECT `+reportColumns+`
		FROM updated rp
		JOIN USERS u ON u.id = rp.user_id`,
		id,
		string(status),
		adminID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, r.unresolvableError(ctx, id)
		}
		return nil, err
	}

	return &resolved, nil
}

// ResolveContent marks removed every pending report of the place, and of its
// reviews, when reviewID is nil; otherwise of the review only. It runs in the
// transaction carried by ctx if there is one.
func (r *PostgresRepository) ResolveContent(ctx context.Context, placeID int64, reviewID *int64, adminID int64) error {
	_, err := database.Conn(ctx, r.db).ExecContext(
		ctx,
		`UPDATE REPORTS
		SET status = 'removed', resolved_by = $3, resolved_at = NOW()
		WHERE status = 'pending' AND place_id = $1
			AND ($2::BIGINT IS NULL OR review_id = $2)`,
		placeID,
		reviewID,
		adminID,
	)

	return err
}

// reportColumns selects a report from `rp` and its author from `u`, aliased
// to match the nested Report.User fields.
const reportColumns = `rp.id, rp.user_id, rp.place_id, rp.review_id, rp.reason, rp.note,
	rp.status, rp.resolved_by, rp.resolved_at, rp.created_at,
	u.id AS "user.id", u.name AS "user.name"`

// unresolvableError explains why report id was not resolved: it does not
// exist, or it is no longer pending.
func (r *PostgresRepository) unresolvableError(ctx context.Context, id int64) error {
	var exists bool
	err := database.Conn(ctx, r.db).GetContext(
		ctx,
		&exists,
		`SELECT EXISTS (SELECT 1 FROM REPORTS WHERE id = $1)`,
		id,
	)
	if err != nil {
		return err
	}
	if !exists {
		return domain.ErrReportNotFound
	}

	return domain.ErrReportAlreadyResolved
}
