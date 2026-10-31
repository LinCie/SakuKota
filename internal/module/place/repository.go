package place

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"

	"rantau-go/internal/database"
)

// Repository defines the persistence operations required by the service.
//
//go:generate mockgen -source=repository.go -destination=mock/repository.go -package=mock
type Repository interface {
	GetMany(ctx context.Context, filter PlaceFilter) ([]Place, int, error)
	Create(ctx context.Context, p *Place) (*Place, error)
	GetCategories(ctx context.Context) ([]Category, error)
	GetCategory(ctx context.Context, id int64) (*Category, error)
	GetPlace(ctx context.Context, id int64) (*Place, error)
	Update(ctx context.Context, p *Place) (*Place, error)
	// SetImageKey stores key as the place's image and returns the place.
	SetImageKey(ctx context.Context, id int64, key string) (*Place, error)
	// Delete removes the place and returns the key of its image, if any.
	Delete(ctx context.Context, id int64) (*string, error)
}

type postgresRepository struct {
	db *sqlx.DB
}

func NewPostgresRepository(db *sqlx.DB) *postgresRepository {
	return &postgresRepository{
		db: db,
	}
}

// -----------------------------------------------------------------------------
// postgresRepository
// -----------------------------------------------------------------------------

// likeEscaper escapes LIKE wildcards so the search query matches literally.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// placeColumns selects a place from `p` and its category from `c`, aliased to
// match the nested Place.Category fields.
const placeColumns = `p.id, p.sub_district_id, p.category_id, p.name, p.address,
	p.location, p.operating_hours, p.estimated_price_range, p.short_description,
	p.attributes, p.rating_count, p.rating_sum, p.image_key, c.id AS "category.id", c.name AS "category.name", c.slug AS "category.slug",
	c.code AS "category.code", c.label AS "category.label", c.color AS "category.color"`

// placeRowColumns are the PLACES columns that placeColumns reads from `p`. The
// RETURNING clause of a write lists them so its row can be joined with its
// category and selected with placeColumns.
const placeRowColumns = `id, sub_district_id, category_id, name, address,
	location, operating_hours, estimated_price_range, short_description,
	attributes, rating_count, rating_sum, image_key`

// checkFilterTargets returns ErrSubDistrictNotFound or ErrCategoryNotFound when
// the filter names a sub-district or category that does not exist.
func (r *postgresRepository) checkFilterTargets(ctx context.Context, filter PlaceFilter) error {
	if filter.SubDistrictID != 0 {
		var exists bool
		err := r.db.GetContext(
			ctx,
			&exists,
			`SELECT EXISTS (SELECT 1 FROM SUB_DISTRICTS WHERE id = $1)`,
			filter.SubDistrictID,
		)
		if err != nil {
			return err
		}
		if !exists {
			return ErrSubDistrictNotFound
		}
	}

	if len(filter.CategoryIDs) == 0 {
		return nil
	}

	var found []int64
	err := r.db.SelectContext(
		ctx,
		&found,
		`SELECT id FROM CATEGORIES WHERE id = ANY($1)`,
		filter.CategoryIDs,
	)
	if err != nil {
		return err
	}

	for _, id := range filter.CategoryIDs {
		if !slices.Contains(found, id) {
			return ErrCategoryNotFound
		}
	}

	return nil
}

// placeFrom is the FROM clause the place filters are written against.
const placeFrom = ` FROM PLACES p JOIN CATEGORIES c ON c.id = p.category_id`

// placeWhere returns the WHERE clause for the filter and its arguments. Only
// the filters that are set become conditions, so cached statements get plans
// that can use the indexes (a catch-all `$1 IS NULL OR ...` cannot).
func placeWhere(filter PlaceFilter) (string, []any) {
	var conditions []string
	var args []any
	arg := func(v any) string {
		args = append(args, v)
		return "$" + strconv.Itoa(len(args))
	}

	if filter.SubDistrictID != 0 {
		conditions = append(conditions, "p.sub_district_id = "+arg(filter.SubDistrictID))
	}

	if len(filter.CategoryIDs) > 0 {
		conditions = append(conditions, "p.category_id = ANY("+arg(filter.CategoryIDs)+")")
	}

	if filter.Query != "" {
		pattern := arg("%" + likeEscaper.Replace(filter.Query) + "%")
		conditions = append(conditions, fmt.Sprintf(
			"(p.name ILIKE %[1]s OR p.address ILIKE %[1]s OR p.short_description ILIKE %[1]s OR c.name ILIKE %[1]s)",
			pattern,
		))
	}

	if b := filter.Bounds; b != nil {
		conditions = append(conditions, fmt.Sprintf(
			"p.location && ST_MakeEnvelope(%s, %s, %s, %s, 4326)",
			arg(b.MinLng), arg(b.MinLat), arg(b.MaxLng), arg(b.MaxLat),
		))
	}

	if len(conditions) == 0 {
		return "", args
	}

	return " WHERE " + strings.Join(conditions, " AND "), args
}

// GetMany returns the places matching the filter and the number of matches
// before the limit is applied.
func (r *postgresRepository) GetMany(ctx context.Context, filter PlaceFilter) ([]Place, int, error) {
	if err := r.checkFilterTargets(ctx, filter); err != nil {
		return nil, 0, err
	}

	where, args := placeWhere(filter)

	query := `SELECT ` + placeColumns + placeFrom + where + ` ORDER BY p.id`
	pageArgs := args
	if filter.Limit > 0 {
		// Clip so the limit is not written into args, which the count reuses.
		pageArgs = append(slices.Clip(args), filter.Limit)
		query += ` LIMIT $` + strconv.Itoa(len(pageArgs))
	}

	places := make([]Place, 0)
	if err := r.db.SelectContext(ctx, &places, query, pageArgs...); err != nil {
		return nil, 0, err
	}

	// Fewer places than the limit means none were cut off.
	if filter.Limit == 0 || len(places) < filter.Limit {
		return places, len(places), nil
	}

	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*)`+placeFrom+where, args...); err != nil {
		return nil, 0, err
	}

	return places, total, nil
}

func (r *postgresRepository) GetCategories(ctx context.Context) ([]Category, error) {
	categories := make([]Category, 0)

	err := r.db.SelectContext(
		ctx,
		&categories,
		`SELECT id, name, slug, code, label, color FROM CATEGORIES`,
	)
	if err != nil {
		return nil, err
	}

	if err := r.attachFields(ctx, categories); err != nil {
		return nil, err
	}

	return categories, nil
}

// GetCategory returns the category with its fields, or ErrCategoryNotFound.
func (r *postgresRepository) GetCategory(ctx context.Context, id int64) (*Category, error) {
	var category Category

	err := r.db.GetContext(
		ctx,
		&category,
		`SELECT id, name, slug, code, label, color FROM CATEGORIES WHERE id = $1`,
		id,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCategoryNotFound
		}
		return nil, err
	}

	categories := []Category{category}
	if err := r.attachFields(ctx, categories); err != nil {
		return nil, err
	}

	return &categories[0], nil
}

// Create inserts the place and returns it joined with its category.
func (r *postgresRepository) Create(ctx context.Context, p *Place) (*Place, error) {
	var created Place

	err := r.db.GetContext(
		ctx,
		&created,
		`WITH inserted AS (
			INSERT INTO PLACES (sub_district_id, category_id, name, address,
				location, operating_hours, estimated_price_range, short_description,
				attributes)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING `+placeRowColumns+`
		)
		SELECT `+placeColumns+`
		FROM inserted p
		JOIN CATEGORIES c ON c.id = p.category_id`,
		p.SubDistrictID,
		p.CategoryID,
		p.Name,
		p.Address,
		p.Location,
		p.OperatingHours,
		p.EstimatedPriceRange,
		p.ShortDescription,
		p.Attributes,
	)
	if err != nil {
		return nil, mapPlaceForeignKeyError(err)
	}

	return &created, nil
}

// GetPlace returns the place with its category, or ErrPlaceNotFound.
func (r *postgresRepository) GetPlace(ctx context.Context, id int64) (*Place, error) {
	var p Place

	err := r.db.GetContext(
		ctx,
		&p,
		`SELECT `+placeColumns+`
		FROM PLACES p
		JOIN CATEGORIES c ON c.id = p.category_id
		WHERE p.id = $1`,
		id,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPlaceNotFound
		}
		return nil, err
	}

	return &p, nil
}

// Update stores the place's fields and returns it joined with its category.
func (r *postgresRepository) Update(ctx context.Context, p *Place) (*Place, error) {
	var updated Place

	err := r.db.GetContext(
		ctx,
		&updated,
		`WITH updated AS (
			UPDATE PLACES
			SET sub_district_id = $2, category_id = $3, name = $4, address = $5,
				location = $6, operating_hours = $7, estimated_price_range = $8,
				short_description = $9, attributes = $10
			WHERE id = $1
			RETURNING `+placeRowColumns+`
		)
		SELECT `+placeColumns+`
		FROM updated p
		JOIN CATEGORIES c ON c.id = p.category_id`,
		p.ID,
		p.SubDistrictID,
		p.CategoryID,
		p.Name,
		p.Address,
		p.Location,
		p.OperatingHours,
		p.EstimatedPriceRange,
		p.ShortDescription,
		p.Attributes,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPlaceNotFound
		}
		return nil, mapPlaceForeignKeyError(err)
	}

	return &updated, nil
}

// SetImageKey stores key as the place's image and returns the place joined
// with its category, or ErrPlaceNotFound.
func (r *postgresRepository) SetImageKey(ctx context.Context, id int64, key string) (*Place, error) {
	var updated Place

	err := r.db.GetContext(
		ctx,
		&updated,
		`WITH updated AS (
			UPDATE PLACES
			SET image_key = $2
			WHERE id = $1
			RETURNING `+placeRowColumns+`
		)
		SELECT `+placeColumns+`
		FROM updated p
		JOIN CATEGORIES c ON c.id = p.category_id`,
		id,
		key,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPlaceNotFound
		}
		return nil, err
	}

	return &updated, nil
}

// Delete removes the place and returns its image key, or returns
// ErrPlaceNotFound if there is no such place.
func (r *postgresRepository) Delete(ctx context.Context, id int64) (*string, error) {
	var imageKey *string

	err := r.db.GetContext(ctx, &imageKey, `DELETE FROM PLACES WHERE id = $1 RETURNING image_key`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPlaceNotFound
		}
		return nil, err
	}

	return imageKey, nil
}

// mapPlaceForeignKeyError translates a foreign key violation on PLACES into the
// sub-district or category it refers to. Other errors are returned unchanged.
func mapPlaceForeignKeyError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == database.ForeignKeyViolation {
		switch pgErr.ConstraintName {
		case "places_sub_district_id_fkey":
			return ErrSubDistrictNotFound
		case "places_category_id_fkey":
			return ErrCategoryNotFound
		}
	}

	return err
}

// attachFields loads the fields of every given category in one query and
// assigns them to each category's Fields field.
func (r *postgresRepository) attachFields(ctx context.Context, categories []Category) error {
	if len(categories) == 0 {
		return nil
	}

	categoryIDs := make([]int64, 0, len(categories))
	for _, c := range categories {
		categoryIDs = append(categoryIDs, c.ID)
	}

	fields := make([]CategoryField, 0)

	err := r.db.SelectContext(
		ctx,
		&fields,
		`SELECT id, category_id, key, label, data_type, required, options, sort_order
		FROM CATEGORY_FIELDS
		WHERE category_id = ANY($1)
		ORDER BY sort_order, id`,
		categoryIDs,
	)
	if err != nil {
		return err
	}

	fieldsByCategory := make(map[int64][]CategoryField, len(categories))
	for _, f := range fields {
		fieldsByCategory[f.CategoryID] = append(fieldsByCategory[f.CategoryID], f)
	}

	for i := range categories {
		categories[i].Fields = fieldsByCategory[categories[i].ID]
		if categories[i].Fields == nil {
			categories[i].Fields = make([]CategoryField, 0)
		}
	}

	return nil
}
