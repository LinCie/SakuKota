package place

import (
	"context"
	"database/sql"
	"errors"
	"strings"

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

// likeEscaper escapes LIKE wildcards so the search query matches literally.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// placeColumns selects a place from `p` and its category from `c`, aliased to
// match the nested Place.Category fields.
const placeColumns = `p.id, p.sub_district_id, p.category_id, p.name, p.address,
	p.location, p.operating_hours, p.estimated_price_range, p.short_description,
	p.attributes, c.id AS "category.id", c.name AS "category.name", c.slug AS "category.slug"`

func (r *postgresRepository) GetMany(ctx context.Context, filter PlaceFilter) ([]Place, error) {
	places := make([]Place, 0)

	pattern := ""
	if filter.Query != "" {
		pattern = "%" + likeEscaper.Replace(filter.Query) + "%"
	}

	err := r.db.SelectContext(
		ctx,
		&places,
		`SELECT `+placeColumns+`
		FROM PLACES p
		JOIN CATEGORIES c ON c.id = p.category_id
		WHERE ($1::bigint = 0 OR p.sub_district_id = $1)
			AND (COALESCE(CARDINALITY($2::bigint[]), 0) = 0 OR p.category_id = ANY($2))
			AND ($3::text = ''
				OR p.name ILIKE $3
				OR p.address ILIKE $3
				OR p.short_description ILIKE $3
				OR c.name ILIKE $3)
		ORDER BY p.id`,
		filter.SubDistrictID,
		filter.CategoryIDs,
		pattern,
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
		`SELECT id, name, slug FROM CATEGORIES WHERE id = $1`,
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

func (r *postgresRepository) CreateCategory(ctx context.Context, c *Category) (*Category, error) {
	var created Category

	err := r.db.GetContext(
		ctx,
		&created,
		`INSERT INTO CATEGORIES (name, slug)
		VALUES ($1, $2)
		RETURNING id, name, slug`,
		c.Name,
		c.Slug,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == database.UniqueViolation {
			return nil, ErrCategoryExists
		}
		return nil, err
	}

	created.Fields = make([]CategoryField, 0)

	return &created, nil
}

// CreateCategoryField inserts the field at its SortOrder position, moving the
// category's fields from that position on down, or last if SortOrder is zero.
func (r *postgresRepository) CreateCategoryField(ctx context.Context, f *CategoryField) (*CategoryField, error) {
	var created CategoryField

	err := r.db.GetContext(
		ctx,
		&created,
		`WITH shifted AS (
			UPDATE CATEGORY_FIELDS
			SET sort_order = sort_order + 1
			WHERE category_id = $1 AND $7 > 0 AND sort_order >= $7
		)
		INSERT INTO CATEGORY_FIELDS (category_id, key, label, data_type, required, options, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, CASE WHEN $7 > 0 THEN $7 ELSE (
			SELECT COALESCE(MAX(sort_order), 0) + 1
			FROM CATEGORY_FIELDS
			WHERE category_id = $1
		) END)
		RETURNING id, category_id, key, label, data_type, required, options, sort_order`,
		f.CategoryID,
		f.Key,
		f.Label,
		f.DataType,
		f.Required,
		f.Options,
		f.SortOrder,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case database.ForeignKeyViolation:
				return nil, ErrCategoryNotFound
			case database.UniqueViolation:
				return nil, ErrCategoryFieldExists
			}
		}
		return nil, err
	}

	return &created, nil
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
			RETURNING id, sub_district_id, category_id, name, address,
				location, operating_hours, estimated_price_range, short_description,
				attributes
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
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == database.ForeignKeyViolation {
			switch pgErr.ConstraintName {
			case "places_sub_district_id_fkey":
				return nil, ErrSubDistrictNotFound
			case "places_category_id_fkey":
				return nil, ErrCategoryNotFound
			}
		}
		return nil, err
	}

	return &created, nil
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
