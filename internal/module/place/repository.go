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

	if err := r.attachCategories(ctx, places); err != nil {
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

	if err := r.attachCategories(ctx, places); err != nil {
		return nil, err
	}

	return places, nil
}

// attachCategories loads the categories of every given place in one query
// and assigns them to each place's Categories field.
func (r *postgresRepository) attachCategories(ctx context.Context, places []Place) error {
	if len(places) == 0 {
		return nil
	}

	placeIDs := make([]int64, 0, len(places))
	for _, p := range places {
		placeIDs = append(placeIDs, p.ID)
	}

	rows := make([]struct {
		PlaceID int64 `db:"place_id"`
		Category
	}, 0)

	err := r.db.SelectContext(
		ctx,
		&rows,
		`SELECT pc.place_id, c.id, c.name, c.slug
		FROM PLACE_CATEGORIES pc
		JOIN CATEGORIES c ON c.id = pc.category_id
		WHERE pc.place_id = ANY($1)
		ORDER BY c.id`,
		placeIDs,
	)
	if err != nil {
		return err
	}

	categoriesByPlace := make(map[int64][]Category, len(places))
	for _, row := range rows {
		categoriesByPlace[row.PlaceID] = append(categoriesByPlace[row.PlaceID], row.Category)
	}

	for i := range places {
		places[i].Categories = categoriesByPlace[places[i].ID]
		if places[i].Categories == nil {
			places[i].Categories = make([]Category, 0)
		}
	}

	return nil
}
