package database

import (
	"context"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

// NewPostgresDB opens a PostgreSQL connection pool and checks that the database answers.
// The pool keeps at most 10 open connections.
func NewPostgresDB(
	ctx context.Context,
	databaseURL string,
) (*sqlx.DB, error) {
	config, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	// Traces and times every query through the global OpenTelemetry
	// providers, which do nothing until a command sets them.
	config.Tracer = otelpgx.NewTracer()

	db := sqlx.NewDb(
		stdlib.OpenDB(*config),
		"pgx",
	)

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}
