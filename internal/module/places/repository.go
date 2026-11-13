package places

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"

	"rantau-go/internal/database"
	"rantau-go/internal/pkg/postgis"
	"rantau-go/internal/storage"
)

// Repository defines the persistence operations required by the service.
//
//go:generate mockgen -source=repository.go -destination=mock/repository.go -package=mock
type Repository interface {
	// Create saves the place and returns it with its category and photos.
	Create(ctx context.Context, p *Place) (*Place, error)
	// GetMany returns one page of places matching the filter, plus the total number of matches.
	GetMany(ctx context.Context, filter PlaceFilter) ([]Place, int, error)
	// GetCategories returns every category with its fields.
	GetCategories(ctx context.Context) ([]Category, error)
	// GetCategory returns the category with its fields.
	GetCategory(ctx context.Context, id int64) (*Category, error)
	// GetPlace returns the place with its category and photos.
	GetPlace(ctx context.Context, id int64) (*Place, error)
	// Update saves the place's new details and returns it with its category and photos.
	Update(ctx context.Context, p *Place) (*Place, error)
	// Delete removes the place and its photos, and queues their files for deletion.
	// It returns ErrPlaceNotFound if there is no such place.
	Delete(ctx context.Context, id int64) error
	// AddMedia stores the photo in the place's first free slot and returns the place.
	// It returns ErrPlaceNotFound, or ErrMediaLimitReached if the place is already full.
	AddMedia(ctx context.Context, id int64, m storage.Media) (*Place, error)
	// DeleteMedia removes the photo and queues its file for deletion.
	// Later photos move up one slot to close the gap; ErrMediaNotFound if there is no such photo.
	DeleteMedia(ctx context.Context, id, mediaID int64) error
}

type postgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository builds a Repository that stores places in PostgreSQL.
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
	p.attributes, p.rating_count, p.rating_sum,
	c.id AS "category.id", c.name AS "category.name", c.slug AS "category.slug",
	c.code AS "category.code", c.label AS "category.label", c.color AS "category.color"`

// placeRowColumns are the PLACES columns that placeColumns reads from `p`. The
// RETURNING clause of a write lists them so its row can be joined with its
// category and selected with placeColumns.
const placeRowColumns = `id, sub_district_id, category_id, name, address,
	location, operating_hours, estimated_price_range, short_description,
	attributes, rating_count, rating_sum`

// Create inserts the place and returns it with its category and photos.
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
		return nil, mapForeignKeyError(err)
	}

	return r.withMedias(ctx, created)
}

// GetMany returns one page of places matching the filter, plus the total number of matches.
// If the first page is not full, it already holds every match, so no count query runs.
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
	if filter.Offset > 0 {
		pageArgs = append(slices.Clip(pageArgs), filter.Offset)
		query += ` OFFSET $` + strconv.Itoa(len(pageArgs))
	}

	places := make([]Place, 0)
	if err := r.db.SelectContext(ctx, &places, query, pageArgs...); err != nil {
		return nil, 0, err
	}

	if err := r.attachMedias(ctx, places); err != nil {
		return nil, 0, err
	}

	// A first page with fewer places than the limit means none were cut off.
	if filter.Offset == 0 && (filter.Limit == 0 || len(places) < filter.Limit) {
		return places, len(places), nil
	}

	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*)`+placeFrom+where, args...); err != nil {
		return nil, 0, err
	}

	return places, total, nil
}

// GetCategories returns every category with its fields.
func (r *postgresRepository) GetCategories(ctx context.Context) ([]Category, error) {
	categories := make([]Category, 0)

	err := r.db.SelectContext(
		ctx,
		&categories,
		`SELECT id, name, slug, code, label, color FROM CATEGORIES ORDER BY id`,
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

// GetPlace returns the place with its category and photos, or ErrPlaceNotFound.
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

	return r.withMedias(ctx, p)
}

// Update saves the place's details and returns it with its category and photos.
// It returns ErrPlaceNotFound if there is no such place.
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
		return nil, mapForeignKeyError(err)
	}

	return r.withMedias(ctx, updated)
}

// Delete removes the place with its photos and queues their files for deletion.
// It returns ErrPlaceNotFound if there is no such place.
func (r *postgresRepository) Delete(ctx context.Context, id int64) error {
	var deleted int

	err := r.db.GetContext(
		ctx,
		&deleted,
		`WITH medias AS (
			DELETE FROM MEDIAS
			WHERE id IN (SELECT media_id FROM PLACE_MEDIAS WHERE place_id = $1)
			RETURNING key
		), deleted AS (
			DELETE FROM PLACES WHERE id = $1 RETURNING id
		), queued AS (
			INSERT INTO STORAGE_DELETIONS (key)
			SELECT key FROM medias
			ON CONFLICT (key) DO NOTHING
		)
		SELECT COUNT(*) FROM deleted`,
		id,
	)
	if err != nil {
		return err
	}
	if deleted == 0 {
		return ErrPlaceNotFound
	}

	return nil
}

// checkFilterTargets returns ErrSubDistrictNotFound or ErrCategoryNotFound for a missing target.
// This is deliberate: an unknown target gives a 404, not an empty list.
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

// AddMedia stores the photo in the place's first free slot and returns the place.
// It locks the place row so two uploads at once cannot take the same slot or go over the limit.
func (r *postgresRepository) AddMedia(ctx context.Context, id int64, m storage.Media) (*Place, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	// FOR NO KEY UPDATE: only non-key data changes, so foreign key inserts elsewhere are not blocked.
	var locked int64
	err = tx.GetContext(ctx, &locked, `SELECT id FROM PLACES WHERE id = $1 FOR NO KEY UPDATE`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPlaceNotFound
		}
		return nil, err
	}

	var position *int
	err = tx.GetContext(
		ctx,
		&position,
		`SELECT MIN(free) FROM generate_series(1, $2::int) AS free
		WHERE free NOT IN (SELECT position FROM PLACE_MEDIAS WHERE place_id = $1)`,
		id,
		storage.MaxMedias,
	)
	if err != nil {
		return nil, err
	}
	if position == nil {
		return nil, storage.ErrMediaLimitReached
	}

	_, err = tx.ExecContext(
		ctx,
		`WITH media AS (
			INSERT INTO MEDIAS (key, content_type, size_bytes)
			VALUES ($2, $3, $4)
			RETURNING id
		)
		INSERT INTO PLACE_MEDIAS (place_id, media_id, position)
		SELECT $1, id, $5 FROM media`,
		id,
		m.Key,
		m.ContentType,
		m.Size,
		*position,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return r.GetPlace(ctx, id)
}

// DeleteMedia removes the photo, queues its file for deletion, and shifts later photos up one slot.
// The shift closes the gap, so the next added photo goes to the end, not into the hole.
func (r *postgresRepository) DeleteMedia(ctx context.Context, id, mediaID int64) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// FOR NO KEY UPDATE: only non-key data changes, so foreign key inserts elsewhere are not blocked.
	var locked int64
	err = tx.GetContext(ctx, &locked, `SELECT id FROM PLACES WHERE id = $1 FOR NO KEY UPDATE`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.ErrMediaNotFound
		}
		return err
	}

	// The file is queued in the same statement as the row delete, so both commit or neither does.
	// storagecleanup then retries the file delete until it succeeds.
	var position *int
	err = tx.GetContext(
		ctx,
		&position,
		`WITH deleted AS (
			DELETE FROM MEDIAS m
			USING PLACE_MEDIAS pm
			WHERE pm.media_id = m.id AND pm.place_id = $1 AND m.id = $2
			RETURNING m.key, pm.position
		), queued AS (
			INSERT INTO STORAGE_DELETIONS (key)
			SELECT key FROM deleted
			ON CONFLICT (key) DO NOTHING
		)
		SELECT MAX(position) FROM deleted`,
		id,
		mediaID,
	)
	if err != nil {
		return err
	}
	if position == nil {
		return storage.ErrMediaNotFound
	}

	_, err = tx.ExecContext(
		ctx,
		`UPDATE PLACE_MEDIAS SET position = position - 1 WHERE place_id = $1 AND position > $2`,
		id,
		*position,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// withMedias returns p with its photos attached.
func (r *postgresRepository) withMedias(ctx context.Context, p Place) (*Place, error) {
	places := []Place{p}
	if err := r.attachMedias(ctx, places); err != nil {
		return nil, err
	}

	return &places[0], nil
}

// attachMedias loads the photos of all the given places in one query.
// Each place gets its photos in position order.
func (r *postgresRepository) attachMedias(ctx context.Context, places []Place) error {
	if len(places) == 0 {
		return nil
	}

	placeIDs := make([]int64, 0, len(places))
	for _, p := range places {
		placeIDs = append(placeIDs, p.ID)
	}

	var rows []struct {
		PlaceID int64 `db:"place_id"`
		storage.Media
	}

	err := r.db.SelectContext(
		ctx,
		&rows,
		`SELECT pm.place_id, m.id, m.key, m.content_type, m.size_bytes, pm.position
		FROM PLACE_MEDIAS pm
		JOIN MEDIAS m ON m.id = pm.media_id
		WHERE pm.place_id = ANY($1)
		ORDER BY pm.place_id, pm.position`,
		placeIDs,
	)
	if err != nil {
		return err
	}

	mediasByPlace := make(map[int64][]storage.Media, len(places))
	for _, row := range rows {
		mediasByPlace[row.PlaceID] = append(mediasByPlace[row.PlaceID], row.Media)
	}

	for i := range places {
		places[i].Medias = mediasByPlace[places[i].ID]
		if places[i].Medias == nil {
			places[i].Medias = make([]storage.Media, 0)
		}
	}

	return nil
}

// attachFields loads the custom fields of all the given categories in one query.
// Each category gets its fields in sort order.
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

// placeFrom is the FROM clause the place filters are written against.
const placeFrom = ` FROM PLACES p JOIN CATEGORIES c ON c.id = p.category_id`

// placeWhere builds the WHERE clause for the filter, along with its values.
// Only the filters that are set become conditions, so the database can use its indexes.
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
			"p.location && ST_MakeEnvelope(%s, %s, %s, %s, %d)",
			arg(b.MinLng), arg(b.MinLat), arg(b.MaxLng), arg(b.MaxLat), postgis.SRID,
		))
	}

	if len(conditions) == 0 {
		return "", args
	}

	return " WHERE " + strings.Join(conditions, " AND "), args
}

// mapForeignKeyError turns a foreign key error on PLACES into the matching not-found error.
// Any other error is returned unchanged.
func mapForeignKeyError(err error) error {
	if database.IsCode(err, database.ForeignKeyViolation) {
		switch database.ConstraintName(err) {
		case "places_sub_district_id_fkey":
			return ErrSubDistrictNotFound
		case "places_category_id_fkey":
			return ErrCategoryNotFound
		}
	}

	return err
}
