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
	// Remove takes key off the queue.
	Remove(ctx context.Context, key string) error
	// GetDue returns up to limit keys that are due, longest due first.
	GetDue(ctx context.Context, limit int) ([]string, error)
	// Reschedule records a failed attempt to delete key and makes it due again
	// after retryIn.
	Reschedule(ctx context.Context, key string, retryIn time.Duration, lastErr string) error
}

type postgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository builds a Repository that keeps the deletion queue in PostgreSQL.
func NewPostgresRepository(db *sqlx.DB) Repository {
	return &postgresRepository{
		db: db,
	}
}

// -----------------------------------------------------------------------------
// postgresRepository
// -----------------------------------------------------------------------------

// Remove takes key off the queue. Removing a key that is not queued is not an
// error.
func (r *postgresRepository) Remove(ctx context.Context, key string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM STORAGE_DELETIONS WHERE key = $1`, key)
	return err
}

// GetDue returns up to limit due keys, longest due first. It does not change
// the queue, so runs that overlap get the same keys.
func (r *postgresRepository) GetDue(ctx context.Context, limit int) ([]string, error) {
	var keys []string

	err := r.db.SelectContext(
		ctx,
		&keys,
		`SELECT key
		FROM STORAGE_DELETIONS
		WHERE next_attempt_at <= NOW()
		ORDER BY next_attempt_at, key
		LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, err
	}

	return keys, nil
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
