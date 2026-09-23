package subdistrict

import (
	"context"

	"github.com/jmoiron/sqlx"
)

type postgresRepository struct {
	db *sqlx.DB
}

func NewPostgresRepository(db *sqlx.DB) Repository {
	return &postgresRepository{
		db: db,
	}
}

// -----------------------------------------------------------------------------
// postgresRepository
// -----------------------------------------------------------------------------

func (r *postgresRepository) GetMany(ctx context.Context) ([]SubDistrict, error) {
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
