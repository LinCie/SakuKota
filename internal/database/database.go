package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"

	"rantau-go/internal/postgis"
)

func NewDB(
	ctx context.Context,
	databaseURL string,
) (*sqlx.DB, error) {
	config, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}

	db := sqlx.NewDb(
		stdlib.OpenDB(
			*config,
			stdlib.OptionAfterConnect(postgis.Register),
		),
		"pgx",
	)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
