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
	"rantau-go/internal/domain"
	"rantau-go/internal/pkg/postgis"
)

// PostgresRepository stores places in PostgreSQL. It is safe for concurrent use.
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository returns a PostgresRepository that uses db.
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{
		db: db,
	}
}

// Create inserts the place and returns it with its category and photos, in the
// transaction carried by ctx if there is one.
// The CTE returns the new row joined with its category in one query.
func (r *PostgresRepository) Create(ctx context.Context, p *domain.Place) (*domain.Place, error) {
	var created domain.Place

	err := database.Conn(ctx, r.db).GetContext(
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

// AddMedia stores the photo in the place's first free position and returns the place.
// The place row is locked so concurrent uploads cannot take the same position or exceed the limit.
func (r *PostgresRepository) AddMedia(ctx context.Context, id int64, m domain.Media) (*domain.Place, error) {
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
			return nil, domain.ErrPlaceNotFound
		}
		return nil, err
	}

	// The lowest unused position; NULL means every position is taken.
	var position *int
	err = tx.GetContext(
		ctx,
		&position,
		`SELECT MIN(free) FROM generate_series(1, $2::int) AS free
		WHERE free NOT IN (SELECT position FROM PLACE_MEDIAS WHERE place_id = $1)`,
		id,
		domain.MaxMedias,
	)
	if err != nil {
		return nil, err
	}
	if position == nil {
		return nil, domain.ErrMediaLimitReached
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

	return r.Fetch(ctx, id)
}

// List returns one page of places matching the filter, plus the total match count.
// filter.Limit must be positive; the service guarantees it.
func (r *PostgresRepository) List(ctx context.Context, filter Filter) ([]domain.Place, int, error) {
	if err := r.checkFilterTargets(ctx, filter); err != nil {
		return nil, 0, err
	}

	where, args := placeWhere(filter)

	// Clip so the page values are not written into args, which the count reuses.
	pageArgs := slices.Clip(args)

	var query string
	if b := filter.Bounds; b != nil {
		// With bounds, the places are spread over the box, as spreadPlaces orders them.
		longitude, latitude := b.Center()
		n := len(pageArgs)
		pageArgs = append(pageArgs, b.gridCell(filter.Limit), longitude, latitude, filter.Limit, filter.Offset)
		query = fmt.Sprintf(spreadQuery, where, postgis.SRID, n+1, n+2, n+3, n+4, n+5)
	} else {
		pageArgs = append(pageArgs, filter.Limit, filter.Offset)
		query = fmt.Sprintf(`SELECT %s%s%s ORDER BY p.id LIMIT $%d OFFSET $%d`,
			placeColumns, placeFrom, where, len(pageArgs)-1, len(pageArgs))
	}

	places := make([]domain.Place, 0)
	if err := r.db.SelectContext(ctx, &places, query, pageArgs...); err != nil {
		return nil, 0, err
	}

	if err := r.attachMedias(ctx, places); err != nil {
		return nil, 0, err
	}

	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*)`+placeFrom+where, args...); err != nil {
		return nil, 0, err
	}

	return places, total, nil
}

// ListLocations returns the id, category, and coordinates of every place matching the
// filter's sub-district, categories, and query. Bounds, limit, and offset are ignored.
func (r *PostgresRepository) ListLocations(ctx context.Context, filter Filter) ([]Location, error) {
	if err := r.checkFilterTargets(ctx, filter); err != nil {
		return nil, err
	}

	filter.Bounds = nil
	where, args := placeWhere(filter)

	locations := make([]Location, 0)
	err := r.db.SelectContext(
		ctx,
		&locations,
		`SELECT p.id, p.category_id, ST_X(p.location) AS longitude, ST_Y(p.location) AS latitude`+
			placeFrom+where+` ORDER BY p.id`,
		args...,
	)
	if err != nil {
		return nil, err
	}

	return locations, nil
}

// ListByIDs returns the places with the given ids, in the order of ids, with category and photos.
func (r *PostgresRepository) ListByIDs(ctx context.Context, ids []int64) ([]domain.Place, error) {
	places := make([]domain.Place, 0, len(ids))

	err := r.db.SelectContext(
		ctx,
		&places,
		`SELECT `+placeColumns+placeFrom+` WHERE p.id = ANY($1) ORDER BY array_position($1, p.id)`,
		ids,
	)
	if err != nil {
		return nil, err
	}

	if err := r.attachMedias(ctx, places); err != nil {
		return nil, err
	}

	return places, nil
}

// ListCategories returns every category with its fields.
func (r *PostgresRepository) ListCategories(ctx context.Context) ([]domain.Category, error) {
	categories := make([]domain.Category, 0)

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

// FetchCategory returns the category with its fields, or domain.ErrCategoryNotFound.
func (r *PostgresRepository) FetchCategory(ctx context.Context, id int64) (*domain.Category, error) {
	var category domain.Category

	err := r.db.GetContext(
		ctx,
		&category,
		`SELECT id, name, slug, code, label, color FROM CATEGORIES WHERE id = $1`,
		id,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrCategoryNotFound
		}
		return nil, err
	}

	categories := []domain.Category{category}
	if err := r.attachFields(ctx, categories); err != nil {
		return nil, err
	}

	return &categories[0], nil
}

// Fetch returns the place with its category and photos, or domain.ErrPlaceNotFound.
func (r *PostgresRepository) Fetch(ctx context.Context, id int64) (*domain.Place, error) {
	var p domain.Place

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
			return nil, domain.ErrPlaceNotFound
		}
		return nil, err
	}

	return r.withMedias(ctx, p)
}

// Update overwrites the place and returns it with its category and photos.
// Like Create, one CTE writes the row and joins its category.
// It returns domain.ErrPlaceNotFound if there is no such place.
func (r *PostgresRepository) Update(ctx context.Context, p *domain.Place) (*domain.Place, error) {
	var updated domain.Place

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
			return nil, domain.ErrPlaceNotFound
		}
		return nil, mapForeignKeyError(err)
	}

	return r.withMedias(ctx, updated)
}

// AddRating adds one review rating to the place's rating totals, in the
// transaction carried by ctx if there is one. Adding to the totals, not
// recounting them, means two ratings at once both count.
// It returns domain.ErrPlaceNotFound if there is no such place.
func (r *PostgresRepository) AddRating(ctx context.Context, id int64, rating int) error {
	res, err := database.Conn(ctx, r.db).ExecContext(
		ctx,
		`UPDATE PLACES SET rating_count = rating_count + 1, rating_sum = rating_sum + $2 WHERE id = $1`,
		id,
		rating,
	)
	if err != nil {
		return err
	}

	updated, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if updated == 0 {
		return domain.ErrPlaceNotFound
	}

	return nil
}

// Delete removes the place and its photos and queues the photo files for deletion.
// It returns domain.ErrPlaceNotFound if there is no such place.
func (r *PostgresRepository) Delete(ctx context.Context, id int64) error {
	// One statement deletes the photos and the place and queues the photo files,
	// so either all of it commits or none of it does.
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
		return domain.ErrPlaceNotFound
	}

	return nil
}

// DeleteMedia removes the photo, queues its file for deletion, and moves later photos up one position.
// The shift closes the gap, so the next upload goes to the end.
func (r *PostgresRepository) DeleteMedia(ctx context.Context, id, mediaID int64) error {
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
			return domain.ErrMediaNotFound
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
	// No row was deleted: the place has no such photo.
	if position == nil {
		return domain.ErrMediaNotFound
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

// placeColumns selects a place from alias p and its category from alias c,
// named to fill Place.Category.
const placeColumns = `p.id, p.sub_district_id, p.category_id, p.name, p.address,
	p.location, p.operating_hours, p.estimated_price_range, p.short_description,
	p.attributes, p.rating_count, p.rating_sum,
	c.id AS "category.id", c.name AS "category.name", c.slug AS "category.slug",
	c.code AS "category.code", c.label AS "category.label", c.color AS "category.color"`

// placeRowColumns are the PLACES columns that placeColumns reads. A write's RETURNING clause
// lists them so the row can be joined with its category and selected with placeColumns.
const placeRowColumns = `id, sub_district_id, category_id, name, address,
	location, operating_hours, estimated_price_range, short_description,
	attributes, rating_count, rating_sum`

// placeFrom is the FROM clause the filters are written against.
const placeFrom = ` FROM PLACES p JOIN CATEGORIES c ON c.id = p.category_id`

// spreadQuery is a page of places spread over a bounding box, ordered as
// spreadPlaces orders them. Its fmt arguments are the WHERE clause, the SRID,
// and the parameter numbers of the grid cell size, the longitude and latitude
// of the box's centre, the limit, and the offset.
const spreadQuery = `WITH located AS (
	SELECT p.id, p.category_id, p.location,
		floor(ST_X(p.location) / $%[3]d::float8)::bigint AS cell_x,
		floor(ST_Y(p.location) / $%[3]d::float8)::bigint AS cell_y` + placeFrom + `%[1]s
), measured AS (
	SELECT id, category_id, cell_x, cell_y,
		location <-> ST_SetSRID(ST_MakePoint(
			(cell_x + 0.5::float8) * $%[3]d::float8, (cell_y + 0.5::float8) * $%[3]d::float8), %[2]d) AS cell_distance,
		location <-> ST_SetSRID(ST_MakePoint($%[4]d::float8, $%[5]d::float8), %[2]d) AS center_distance,
		dense_rank() OVER (PARTITION BY cell_x, cell_y ORDER BY category_id) - 1 AS category_index
	FROM located
), counted AS (
	SELECT *,
		row_number() OVER (PARTITION BY cell_x, cell_y, category_id ORDER BY cell_distance, id) AS category_round,
		max(category_index) OVER (PARTITION BY cell_x, cell_y) + 1 AS category_count
	FROM measured
), ranked AS (
	SELECT id, center_distance,
		row_number() OVER (
			PARTITION BY cell_x, cell_y
			ORDER BY category_round,
				((category_index - cell_x - cell_y) %% category_count + category_count) %% category_count,
				cell_distance, id
		) AS round
	FROM counted
)
SELECT ` + placeColumns + placeFrom + ` JOIN ranked r ON r.id = p.id
ORDER BY r.round, r.center_distance, p.id
LIMIT $%[6]d OFFSET $%[7]d`

// checkFilterTargets returns domain.ErrSubDistrictNotFound or domain.ErrCategoryNotFound if the filter
// names one that does not exist. An unknown target is a 404, not an empty list.
func (r *PostgresRepository) checkFilterTargets(ctx context.Context, filter Filter) error {
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
			return domain.ErrSubDistrictNotFound
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

	// ANY returns each existing id once, so check every requested id against it.
	for _, id := range filter.CategoryIDs {
		if !slices.Contains(found, id) {
			return domain.ErrCategoryNotFound
		}
	}

	return nil
}

// likeEscaper escapes LIKE wildcards so a search term matches literally.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// placeWhere builds the WHERE clause and its arguments from the filter.
// Only the filters that are set become conditions, so the database can use its indexes.
func placeWhere(filter Filter) (string, []any) {
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
			// ST_Covers keeps the edges and, unlike a bare &&, which compares
			// float4 boxes, drops points just outside. It still uses the GiST index.
			"ST_Covers(ST_MakeEnvelope(%s, %s, %s, %s, %d), p.location)",
			arg(b.MinLng), arg(b.MinLat), arg(b.MaxLng), arg(b.MaxLat), postgis.SRID,
		))
	}

	if len(conditions) == 0 {
		return "", args
	}

	return " WHERE " + strings.Join(conditions, " AND "), args
}

// withMedias returns p with its photos attached.
func (r *PostgresRepository) withMedias(ctx context.Context, p domain.Place) (*domain.Place, error) {
	places := []domain.Place{p}
	if err := r.attachMedias(ctx, places); err != nil {
		return nil, err
	}

	return &places[0], nil
}

// attachMedias loads the photos of all places in one query, in position order.
func (r *PostgresRepository) attachMedias(ctx context.Context, places []domain.Place) error {
	if len(places) == 0 {
		return nil
	}

	placeIDs := make([]int64, 0, len(places))
	for _, p := range places {
		placeIDs = append(placeIDs, p.ID)
	}

	var rows []struct {
		PlaceID int64 `db:"place_id"`
		domain.Media
	}

	err := database.Conn(ctx, r.db).SelectContext(
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

	mediasByPlace := make(map[int64][]domain.Media, len(places))
	for _, row := range rows {
		mediasByPlace[row.PlaceID] = append(mediasByPlace[row.PlaceID], row.Media)
	}

	for i := range places {
		places[i].Medias = mediasByPlace[places[i].ID]
		// Never nil, so it encodes as [].
		if places[i].Medias == nil {
			places[i].Medias = make([]domain.Media, 0)
		}
	}

	return nil
}

// attachFields loads the fields of all categories in one query, in sort order.
func (r *PostgresRepository) attachFields(ctx context.Context, categories []domain.Category) error {
	if len(categories) == 0 {
		return nil
	}

	categoryIDs := make([]int64, 0, len(categories))
	for _, c := range categories {
		categoryIDs = append(categoryIDs, c.ID)
	}

	fields := make([]domain.CategoryField, 0)

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

	fieldsByCategory := make(map[int64][]domain.CategoryField, len(categories))
	for _, f := range fields {
		fieldsByCategory[f.CategoryID] = append(fieldsByCategory[f.CategoryID], f)
	}

	for i := range categories {
		categories[i].Fields = fieldsByCategory[categories[i].ID]
		// Never nil, so it encodes as [].
		if categories[i].Fields == nil {
			categories[i].Fields = make([]domain.CategoryField, 0)
		}
	}

	return nil
}

// mapForeignKeyError turns a PLACES foreign key violation into the matching not-found error.
func mapForeignKeyError(err error) error {
	if database.IsCode(err, database.ForeignKeyViolation) {
		switch database.ConstraintName(err) {
		case "places_sub_district_id_fkey":
			return domain.ErrSubDistrictNotFound
		case "places_category_id_fkey":
			return domain.ErrCategoryNotFound
		}
	}

	return err
}
