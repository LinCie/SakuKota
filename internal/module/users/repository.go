package users

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"sakukota-go/internal/database"
	"sakukota-go/internal/domain"
)

// PostgresRepository keeps users in PostgreSQL.
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository builds a PostgresRepository over db.
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// userSelect selects users from `u` with the key of their profile photo.
const userSelect = `SELECT u.id, u.name, u.email, u.password_hash, u.role, u.created_at, u.updated_at, m.key AS avatar_key
	FROM USERS u
	LEFT JOIN MEDIAS m ON m.id = u.avatar_media_id`

// Create inserts the user and returns it with its generated fields, or
// domain.ErrEmailTaken if the email is already registered.
func (r *PostgresRepository) Create(ctx context.Context, u *domain.User) (*domain.User, error) {
	var created domain.User

	err := r.db.GetContext(
		ctx,
		&created,
		`INSERT INTO USERS (name, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, name, email, password_hash, role, created_at, updated_at`,
		u.Name,
		u.Email,
		u.PasswordHash,
	)
	if database.IsCode(err, database.UniqueViolation) {
		return nil, domain.ErrEmailTaken
	}
	if err != nil {
		return nil, err
	}

	return &created, nil
}

// FetchByEmail returns the user with the email, or domain.ErrUserNotFound.
func (r *PostgresRepository) FetchByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.fetch(ctx, `u.email = $1`, email)
}

// Fetch returns the user with the id, or domain.ErrUserNotFound.
func (r *PostgresRepository) Fetch(ctx context.Context, id int64) (*domain.User, error) {
	return r.fetch(ctx, `u.id = $1`, id)
}

// fetch returns the one user matching where, or domain.ErrUserNotFound.
func (r *PostgresRepository) fetch(ctx context.Context, where string, args ...any) (*domain.User, error) {
	var u domain.User

	err := r.db.GetContext(ctx, &u, userSelect+` WHERE `+where, args...)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return &u, nil
}

// Rename sets the user's name and returns the user, or domain.ErrUserNotFound.
func (r *PostgresRepository) Rename(ctx context.Context, id int64, name string) (*domain.User, error) {
	// An unknown user updates nothing, and Fetch then reports it.
	if _, err := r.db.ExecContext(ctx, `UPDATE USERS SET name = $2, updated_at = NOW() WHERE id = $1`, id, name); err != nil {
		return nil, err
	}

	return r.Fetch(ctx, id)
}

// SetAvatar stores the photo as the user's profile photo and returns the user,
// or domain.ErrUserNotFound. The previous photo is deleted and its file queued
// for deletion. The user row is locked so concurrent changes queue every
// replaced file.
func (r *PostgresRepository) SetAvatar(ctx context.Context, id int64, m domain.Media) (*domain.User, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	// FOR NO KEY UPDATE: only non-key data changes, so foreign key inserts elsewhere are not blocked.
	var oldID *int64
	err = tx.GetContext(ctx, &oldID, `SELECT avatar_media_id FROM USERS WHERE id = $1 FOR NO KEY UPDATE`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	_, err = tx.ExecContext(
		ctx,
		`WITH media AS (
			INSERT INTO MEDIAS (key, content_type, size_bytes)
			VALUES ($2, $3, $4)
			RETURNING id
		)
		UPDATE USERS SET avatar_media_id = (SELECT id FROM media), updated_at = NOW()
		WHERE id = $1`,
		id,
		m.Key,
		m.ContentType,
		m.Size,
	)
	if err != nil {
		return nil, err
	}

	if oldID != nil {
		// The file is queued in the same statement as the row delete, so both commit or neither does.
		_, err = tx.ExecContext(
			ctx,
			`WITH deleted AS (
				DELETE FROM MEDIAS WHERE id = $1
				RETURNING key
			)
			INSERT INTO STORAGE_DELETIONS (key)
			SELECT key FROM deleted
			ON CONFLICT (key) DO NOTHING`,
			*oldID,
		)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return r.Fetch(ctx, id)
}
