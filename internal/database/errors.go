package database

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// UniqueViolation is the PostgreSQL error code for unique constraint violations.
const UniqueViolation = "23505"

// ForeignKeyViolation is the PostgreSQL error code for foreign key constraint violations.
const ForeignKeyViolation = "23503"

// IsCode reports whether err is a PostgreSQL error with the given SQLSTATE code.
func IsCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}
