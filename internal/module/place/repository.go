package place

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

func (r *postgresRepository) GetMany(ctx context.Context) ([]Place, error) {
	places := make([]Place, 0)

	err := r.db.SelectContext(
		ctx,
		&places,
		`SELECT id, sub_district_id, name, address, location,
			operating_hours, estimated_price_range, short_description
		FROM PLACES`,
	)
	if err != nil {
		return nil, err
	}

	return places, nil
}

func (r *postgresRepository) GetCategories(ctx context.Context) ([]Category, error) {
	categories := make([]Category, 0)

	err := r.db.SelectContext(
		ctx,
		&categories,
		`SELECT id, name, slug FROM CATEGORIES`,
	)
	if err != nil {
		return nil, err
	}

	return categories, nil
}

func (r *postgresRepository) GetManyByCategory(
	ctx context.Context,
	categoryID int64,
) ([]Place, error) {
	places := make([]Place, 0)

	err := r.db.SelectContext(
		ctx,
		&places,
		`SELECT p.id, p.sub_district_id, p.name, p.address, p.location,
			p.operating_hours, p.estimated_price_range, p.short_description
		FROM PLACES p
		JOIN PLACE_CATEGORIES pc ON pc.place_id = p.id
		WHERE pc.category_id = $1`,
		categoryID,
	)
	if err != nil {
		return nil, err
	}

	return places, nil
}
