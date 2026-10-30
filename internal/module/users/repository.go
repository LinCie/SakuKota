package users

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"rantau-go/internal/database"
)

var (
	ErrNotFound   = errors.New("user not found")
	ErrEmailTaken = errors.New("email already registered")
)

// Repository defines the persistence operations for users.
//
//go:generate mockgen -source=repository.go -destination=mock/repository.go -package=mock
type Repository interface {
	Create(ctx context.Context, u *User) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id int64) (*User, error)
}

type postgresRepository struct {
	db *sqlx.DB
}

func NewPostgresRepository(db *sqlx.DB) *postgresRepository {
	return &postgresRepository{
		db: db,
	}
}

// -----------------------------------------------------------------------------
// postgresRepository
// -----------------------------------------------------------------------------

func (r *postgresRepository) Create(ctx context.Context, u *User) (*User, error) {
	var created User

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
	if err != nil {
		if database.IsCode(err, database.UniqueViolation) {
			return nil, ErrEmailTaken
		}
		return nil, err
	}

	return &created, nil
}

func (r *postgresRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	var u User

	err := r.db.GetContext(
		ctx,
		&u,
		`SELECT id, name, email, password_hash, role, created_at, updated_at
		FROM USERS
		WHERE email = $1`,
		email,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &u, nil
}

func (r *postgresRepository) GetByID(ctx context.Context, id int64) (*User, error) {
	var u User

	err := r.db.GetContext(
		ctx,
		&u,
		`SELECT id, name, email, password_hash, role, created_at, updated_at
		FROM USERS
		WHERE id = $1`,
		id,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &u, nil
}
