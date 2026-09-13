package database

import (
	"context"
	"rantau-go/internal/postgis"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(
	ctx context.Context,
	databaseURL string,
) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}

	config.AfterConnect = func(
		ctx context.Context,
		conn *pgx.Conn,
	) error {
		return postgis.Register(ctx, conn)
	}

	return pgxpool.NewWithConfig(ctx, config)
}
