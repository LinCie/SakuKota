package subdistrict

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	GetMany(ctx context.Context) ([]SubDistrict, error)
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

var _ Repository = (*PostgresRepository)(nil)

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{
		pool: pool,
	}
}

// -----------------------------------------------------------------------------
// PostgresRepository
// -----------------------------------------------------------------------------

func (r *PostgresRepository) GetMany(ctx context.Context) ([]SubDistrict, error) {
	rows, err := r.pool.Query(
		ctx,
		`SELECT id, name, slug, flood_history, safety_crime_information,
			estimated_daily_spending, static_map_image_url
		FROM SUB_DISTRICTS`,
	)
	if err != nil {
		return nil, err
	}

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (SubDistrict, error) {
		return scanSubDistrict(ctx, row)
	})
}

// -----------------------------------------------------------------------------
// Scans
// -----------------------------------------------------------------------------

func scanSubDistrict(ctx context.Context, row pgx.CollectableRow) (SubDistrict, error) {
	if err := ctx.Err(); err != nil {
		return SubDistrict{}, err
	}

	var s SubDistrict

	err := row.Scan(
		&s.ID,
		&s.Name,
		&s.Slug,
		&s.FloodHistory,
		&s.SafetyCrimeInformation,
		&s.EstimatedDailySpending,
		&s.StaticMapImageURL,
	)

	return s, err
}
