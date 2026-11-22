package storagecleanup

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
)

// PostgresRepository keeps the deletion queue in PostgreSQL.
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository builds a PostgresRepository over db.
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Remove takes key off the queue. Removing a key that is not queued is not an
// error.
func (r *PostgresRepository) Remove(ctx context.Context, key string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM STORAGE_DELETIONS WHERE key = $1`, key)
	return err
}

// ListDue returns up to limit due keys, longest due first. It does not change
// the queue, so runs that overlap get the same keys.
func (r *PostgresRepository) ListDue(ctx context.Context, limit int) ([]string, error) {
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
func (r *PostgresRepository) Reschedule(ctx context.Context, key string, retryIn time.Duration, lastErr string) error {
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
