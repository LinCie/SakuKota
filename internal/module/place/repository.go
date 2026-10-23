package place

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"

	"rantau-go/internal/database"
)

// Repository defines the persistence operations required by the service.
//
//go:generate mockgen -source=repository.go -destination=mock/repository.go -package=mock
type Repository interface {
	GetMany(ctx context.Context, filter PlaceFilter) ([]Place, error)
	Create(ctx context.Context, p *Place) (*Place, error)
	GetCategories(ctx context.Context) ([]Category, error)
	GetCategory(ctx context.Context, id int64) (*Category, error)
	CreateCategory(ctx context.Context, c *Category) (*Category, error)
	CreateCategoryField(ctx context.Context, f *CategoryField) (*CategoryField, error)
	GetPlace(ctx context.Context, id int64) (*Place, error)
	Update(ctx context.Context, p *Place) (*Place, error)
	Delete(ctx context.Context, id int64) error
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
	p.attributes, c.id AS "category.id", c.name AS "category.name", c.slug AS "category.slug"`

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

func (r *postgresRepository) GetMany(ctx context.Context, filter PlaceFilter) ([]Place, error) {
	if err := r.checkFilterTargets(ctx, filter); err != nil {
		return nil, err
	}

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
		ORDER BY p.id
		LIMIT NULLIF($4::bigint, 0)`,
		filter.SubDistrictID,
		filter.CategoryIDs,
		pattern,
		filter.Limit,
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

// CreateCategoryField adds the field at its SortOrder position, moving the
// category's fields from that position on down, or last if SortOrder is nil.
// The category row is locked for the duration, so concurrent adds to the same
// category take turns and each sees the positions the others took.
func (r *postgresRepository) CreateCategoryField(ctx context.Context, f *CategoryField) (*CategoryField, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var categoryID int64
	err = tx.GetContext(ctx, &categoryID, `SELECT id FROM CATEGORIES WHERE id = $1 FOR UPDATE`, f.CategoryID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCategoryNotFound
		}
		return nil, err
	}

	var count int32
	err = tx.GetContext(ctx, &count, `SELECT COUNT(*) FROM CATEGORY_FIELDS WHERE category_id = $1`, f.CategoryID)
	if err != nil {
		return nil, err
	}

	sortOrder := count + 1
	if f.SortOrder != nil {
		if *f.SortOrder > sortOrder {
			return nil, ErrInvalidFieldSortOrder
		}

		sortOrder = *f.SortOrder
		_, err = tx.ExecContext(
			ctx,
			`UPDATE CATEGORY_FIELDS SET sort_order = sort_order + 1
			WHERE category_id = $1 AND sort_order >= $2`,
			f.CategoryID,
			sortOrder,
		)
		if err != nil {
			return nil, err
		}
	}

	var created CategoryField
	err = tx.GetContext(
		ctx,
		&created,
		`INSERT INTO CATEGORY_FIELDS (category_id, key, label, data_type, required, options, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, category_id, key, label, data_type, required, options, sort_order`,
		f.CategoryID,
		f.Key,
		f.Label,
		f.DataType,
		f.Required,
		f.Options,
		sortOrder,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == database.UniqueViolation {
			return nil, ErrCategoryFieldExists
		}
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &created, nil
}

// Create inserts the place and returns it joined with its category. A request
// with an idempotency key is stored once and replayed on retries.
func (r *postgresRepository) Create(ctx context.Context, p *Place) (*Place, error) {
	if p.IdempotencyKey == "" {
		return insertPlace(ctx, r.db, p)
	}

	return r.createOnce(ctx, p)
}

// insertPlace inserts the place through q and returns it joined with its category.
func insertPlace(ctx context.Context, q sqlx.QueryerContext, p *Place) (*Place, error) {
	var created Place

	err := sqlx.GetContext(
		ctx,
		q,
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
		return nil, mapPlaceForeignKeyError(err)
	}

	return &created, nil
}

// createOnce stores the place together with its idempotency key. If the key is
// already taken, the place holding it is returned when the request matches, and
// ErrIdempotencyKeyReused is returned when it does not.
func (r *postgresRepository) createOnce(ctx context.Context, p *Place) (*Place, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	created, err := insertPlace(ctx, tx, p)
	if err != nil {
		return nil, err
	}

	// A concurrent request with the same key waits here until this transaction
	// ends, then inserts nothing.
	result, err := tx.ExecContext(
		ctx,
		`INSERT INTO PLACE_IDEMPOTENCY_KEYS (key, request_hash, place_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (key) DO NOTHING`,
		p.IdempotencyKey,
		p.RequestHash,
		created.ID,
	)
	if err != nil {
		return nil, err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rows == 1 {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return created, nil
	}

	// Discard this request's copy of the place and answer with the one that
	// holds the key.
	if err := tx.Rollback(); err != nil {
		return nil, err
	}

	return r.replay(ctx, p)
}

// replay returns the place holding the request's idempotency key, or
// ErrIdempotencyKeyReused if the key belongs to a different request.
func (r *postgresRepository) replay(ctx context.Context, p *Place) (*Place, error) {
	var stored struct {
		RequestHash string `db:"request_hash"`
		PlaceID     int64  `db:"place_id"`
	}

	err := r.db.GetContext(
		ctx,
		&stored,
		`SELECT request_hash, place_id FROM PLACE_IDEMPOTENCY_KEYS WHERE key = $1`,
		p.IdempotencyKey,
	)
	if err != nil {
		return nil, err
	}
	if stored.RequestHash != p.RequestHash {
		return nil, ErrIdempotencyKeyReused
	}

	return r.GetPlace(ctx, stored.PlaceID)
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
			RETURNING id, sub_district_id, category_id, name, address,
				location, operating_hours, estimated_price_range, short_description,
				attributes
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

// Delete removes the place, or returns ErrPlaceNotFound if there is none.
func (r *postgresRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM PLACES WHERE id = $1`, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrPlaceNotFound
	}

	return nil
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
