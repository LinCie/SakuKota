package storagecleanup

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
)

// Repository defines the persistence operations required by the service.
//
//go:generate mockgen -source=repository.go -destination=mock/repository.go -package=mock
type Repository interface {
	// ClaimDue returns up to limit deletions that are due and holds them back
	// from other workers until they are removed or rescheduled.
	ClaimDue(ctx context.Context, limit int) ([]Deletion, error)
	// Remove takes key off the queue.
	Remove(ctx context.Context, key string) error
	// Reschedule records a failed attempt to delete key and makes it due again
	// after retryIn.
	Reschedule(ctx context.Context, key string, retryIn time.Duration, lastErr string) error
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

// ClaimDue returns up to limit due deletions, longest due first, and holds
// them back by moving their next attempt claimLease ahead. Rows claimed by a
// concurrent worker are skipped.
func (r *postgresRepository) ClaimDue(ctx context.Context, limit int) ([]Deletion, error) {
	deletions := make([]Deletion, 0)

	err := r.db.SelectContext(
		ctx,
		&deletions,
		`WITH due AS (
			SELECT key, next_attempt_at AS due_at
			FROM STORAGE_DELETIONS
			WHERE next_attempt_at <= NOW()
			ORDER BY next_attempt_at, key
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		), claimed AS (
			UPDATE STORAGE_DELETIONS d
			SET next_attempt_at = NOW() + make_interval(secs => $2)
			FROM due
			WHERE d.key = due.key
			RETURNING d.key, d.attempts, due.due_at
		)
		SELECT key, attempts FROM claimed ORDER BY due_at, key`,
		limit,
		claimLease.Seconds(),
	)
	if err != nil {
		return nil, err
	}

	return deletions, nil
}

// Remove takes key off the queue. Removing a key that is not queued is not an
// error.
func (r *postgresRepository) Remove(ctx context.Context, key string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM STORAGE_DELETIONS WHERE key = $1`, key)
	return err
}

// Reschedule records a failed attempt to delete key and makes it due again
// after retryIn. The error message is stored in last_error.
func (r *postgresRepository) Reschedule(ctx context.Context, key string, retryIn time.Duration, lastErr string) error {
	_, err := r.db.ExecContext(
		ctx,
		`UPDATE STORAGE_DELETIONS
		SET attempts = attempts + 1,
			last_error = $2,
			next_attempt_at = NOW() + make_interval(secs => $3)
		WHERE key = $1`,
		key,
		lastErr,
		retryIn.Seconds(),
	)
	return err
}
