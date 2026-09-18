package subdistrict

import (
	"context"

	"github.com/jmoiron/sqlx"
)

type Repository interface {
	GetMany(ctx context.Context) ([]SubDistrict, error)
}

type PostgresRepository struct {
	db *sqlx.DB
}

var _ Repository = (*PostgresRepository)(nil)

func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{
		db: db,
	}
}

// -----------------------------------------------------------------------------
// PostgresRepository
// -----------------------------------------------------------------------------

func (r *PostgresRepository) GetMany(ctx context.Context) ([]SubDistrict, error) {
	subDistricts := make([]SubDistrict, 0)

	err := r.db.SelectContext(
		ctx,
		&subDistricts,
		`SELECT id, name, slug, flood_history, safety_crime_information,
			estimated_daily_spending, static_map_image_url
		FROM SUB_DISTRICTS`,
	)
	if err != nil {
		return nil, err
	}

	return subDistricts, nil
}
