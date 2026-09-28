package place

import (
	"context"
	"strings"

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

// likeEscaper escapes LIKE wildcards so the search query matches literally.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func (r *postgresRepository) GetMany(ctx context.Context, filter PlaceFilter) ([]Place, error) {
	places := make([]Place, 0)

	pattern := ""
	if filter.Query != "" {
		pattern = "%" + likeEscaper.Replace(filter.Query) + "%"
	}

	err := r.db.SelectContext(
		ctx,
		&places,
		`SELECT p.id, p.sub_district_id, p.name, p.address, p.location,
			p.operating_hours, p.estimated_price_range, p.short_description
		FROM PLACES p
		WHERE ($1::bigint = 0 OR p.sub_district_id = $1)
			AND (COALESCE(CARDINALITY($2::bigint[]), 0) = 0 OR EXISTS (
				SELECT 1 FROM PLACE_CATEGORIES pc
				WHERE pc.place_id = p.id AND pc.category_id = ANY($2)
			))
			AND ($3::text = ''
				OR p.name ILIKE $3
				OR p.address ILIKE $3
				OR p.short_description ILIKE $3
				OR EXISTS (
					SELECT 1 FROM PLACE_CATEGORIES pc
					JOIN CATEGORIES c ON c.id = pc.category_id
					WHERE pc.place_id = p.id AND c.name ILIKE $3
				))
		ORDER BY p.id`,
		filter.SubDistrictID,
		filter.CategoryIDs,
		pattern,
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
