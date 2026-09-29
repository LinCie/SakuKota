package subdistrict

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"

	"rantau-go/internal/database"
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

func (r *postgresRepository) Create(ctx context.Context, sd *SubDistrict) (*SubDistrict, error) {
	var created SubDistrict

	err := r.db.GetContext(
		ctx,
		&created,
		`INSERT INTO SUB_DISTRICTS (name, slug, flood_history,
			safety_crime_information, estimated_daily_spending, static_map_image_url)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, slug, flood_history, safety_crime_information,
			estimated_daily_spending, static_map_image_url`,
		sd.Name,
		sd.Slug,
		sd.FloodHistory,
		sd.SafetyCrimeInformation,
		sd.EstimatedDailySpending,
		sd.StaticMapImageURL,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == database.UniqueViolation {
			return nil, ErrSubDistrictExists
		}
		return nil, err
	}

	return &created, nil
}
