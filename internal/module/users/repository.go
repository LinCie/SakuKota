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
	var u domain.User

	err := r.db.GetContext(
		ctx,
		&u,
		`SELECT id, name, email, password_hash, role, created_at, updated_at
		FROM USERS
		WHERE email = $1`,
		email,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return &u, nil
}

// Fetch returns the user with the id, or domain.ErrUserNotFound.
func (r *PostgresRepository) Fetch(ctx context.Context, id int64) (*domain.User, error) {
	var u domain.User

	err := r.db.GetContext(
		ctx,
		&u,
		`SELECT id, name, email, password_hash, role, created_at, updated_at
		FROM USERS
		WHERE id = $1`,
		id,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return &u, nil
}
