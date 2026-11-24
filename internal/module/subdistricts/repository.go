package subdistricts

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"sakukota-go/internal/database"
	"sakukota-go/internal/domain"
	"sakukota-go/internal/pkg/postgis"
)

// PostgresRepository stores sub-districts in PostgreSQL.
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository builds a PostgresRepository over db.
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{
		db: db,
	}
}

// Create inserts the sub-district and returns it with its photos.
// It returns domain.ErrSubDistrictExists if the name or slug is already taken.
func (r *PostgresRepository) Create(ctx context.Context, sd *domain.SubDistrict) (*domain.SubDistrict, error) {
	var created domain.SubDistrict

	err := r.db.GetContext(
		ctx,
		&created,
		`WITH sd AS (
			INSERT INTO SUB_DISTRICTS (name, slug, flood_history,
				safety_crime_information, estimated_daily_spending)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING *
		)
		SELECT `+subDistrictColumns+` FROM sd`,
		sd.Name,
		sd.Slug,
		sd.FloodHistory,
		sd.SafetyCrimeInformation,
		sd.EstimatedDailySpending,
	)
	if err != nil {
		if database.IsCode(err, database.UniqueViolation) {
			return nil, domain.ErrSubDistrictExists
		}
		return nil, err
	}

	return r.withMedias(ctx, created)
}

// AddMedia stores the photo after the sub-district's last one and returns the sub-district.
// It locks the sub-district row so two uploads at once cannot take the same slot or go over the limit.
func (r *PostgresRepository) AddMedia(ctx context.Context, id int64, m domain.Media) (*domain.SubDistrict, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var locked int64
	err = tx.GetContext(ctx, &locked, `SELECT id FROM SUB_DISTRICTS WHERE id = $1 FOR NO KEY UPDATE`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrSubDistrictNotFound
		}
		return nil, err
	}

	// DeleteMedia closes gaps, so the positions in use are 1..count and the next free one is count+1.
	var count int
	err = tx.GetContext(ctx, &count, `SELECT COUNT(*) FROM SUB_DISTRICT_MEDIAS WHERE sub_district_id = $1`, id)
	if err != nil {
		return nil, err
	}
	if count >= domain.MaxMedias {
		return nil, domain.ErrMediaLimitReached
	}

	_, err = tx.ExecContext(
		ctx,
		`WITH media AS (
			INSERT INTO MEDIAS (key, content_type, size_bytes)
			VALUES ($2, $3, $4)
			RETURNING id
		)
		INSERT INTO SUB_DISTRICT_MEDIAS (sub_district_id, media_id, position)
		SELECT $1, id, $5 FROM media`,
		id,
		m.Key,
		m.ContentType,
		m.Size,
		count+1,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return r.Fetch(ctx, id)
}

// List returns every sub-district in id order, with its photos.
func (r *PostgresRepository) List(ctx context.Context) ([]domain.SubDistrict, error) {
	subDistricts := make([]domain.SubDistrict, 0)

	err := r.db.SelectContext(
		ctx,
		&subDistricts,
		`SELECT `+subDistrictColumns+` FROM SUB_DISTRICTS sd ORDER BY sd.id`,
	)
	if err != nil {
		return nil, err
	}

	if err := r.attachMedias(ctx, subDistricts); err != nil {
		return nil, err
	}

	return subDistricts, nil
}

// Fetch returns the sub-district with the id, or domain.ErrSubDistrictNotFound.
func (r *PostgresRepository) Fetch(ctx context.Context, id int64) (*domain.SubDistrict, error) {
	var sd domain.SubDistrict

	err := r.db.GetContext(
		ctx,
		&sd,
		`SELECT `+subDistrictColumns+` FROM SUB_DISTRICTS sd WHERE sd.id = $1`,
		id,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrSubDistrictNotFound
		}
		return nil, err
	}

	return r.withMedias(ctx, sd)
}

// Update saves the sub-district's details and returns it with its photos.
// It returns domain.ErrSubDistrictNotFound, or domain.ErrSubDistrictExists if the name or slug is taken.
func (r *PostgresRepository) Update(ctx context.Context, sd *domain.SubDistrict) (*domain.SubDistrict, error) {
	var updated domain.SubDistrict

	err := r.db.GetContext(
		ctx,
		&updated,
		`WITH sd AS (
			UPDATE SUB_DISTRICTS
			SET name = $2, slug = $3, flood_history = $4, safety_crime_information = $5,
				estimated_daily_spending = $6
			WHERE id = $1
			RETURNING *
		)
		SELECT `+subDistrictColumns+` FROM sd`,
		sd.ID,
		sd.Name,
		sd.Slug,
		sd.FloodHistory,
		sd.SafetyCrimeInformation,
		sd.EstimatedDailySpending,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrSubDistrictNotFound
		}
		if database.IsCode(err, database.UniqueViolation) {
			return nil, domain.ErrSubDistrictExists
		}
		return nil, err
	}

	return r.withMedias(ctx, updated)
}

// SetBoundary saves the area as the sub-district's boundary and returns the sub-district.
func (r *PostgresRepository) SetBoundary(ctx context.Context, id int64, boundary *postgis.MultiPolygon) (*domain.SubDistrict, error) {
	var updated domain.SubDistrict

	err := r.db.GetContext(
		ctx,
		&updated,
		`WITH sd AS (
			UPDATE SUB_DISTRICTS SET boundary = $2 WHERE id = $1 RETURNING *
		)
		SELECT `+subDistrictColumns+` FROM sd`,
		id,
		boundary,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrSubDistrictNotFound
		}
		if database.IsCode(err, database.CheckViolation) && database.ConstraintName(err) == "sub_districts_boundary_check" {
			return nil, domain.ErrSubDistrictBoundaryInvalid
		}
		return nil, err
	}

	return r.withMedias(ctx, updated)
}

// Delete removes the sub-district with its photos, and queues their files for deletion.
// It returns domain.ErrSubDistrictNotFound, or domain.ErrSubDistrictInUse if places still use it, and removes nothing.
func (r *PostgresRepository) Delete(ctx context.Context, id int64) error {
	var deleted int

	err := r.db.GetContext(
		ctx,
		&deleted,
		`WITH medias AS (
			DELETE FROM MEDIAS
			WHERE id IN (SELECT media_id FROM SUB_DISTRICT_MEDIAS WHERE sub_district_id = $1)
			RETURNING key
		), deleted AS (
			DELETE FROM SUB_DISTRICTS WHERE id = $1 RETURNING id
		), queued AS (
			INSERT INTO STORAGE_DELETIONS (key)
			SELECT key FROM medias
			ON CONFLICT (key) DO NOTHING
		)
		SELECT COUNT(*) FROM deleted`,
		id,
	)
	if err != nil {
		if database.IsCode(err, database.ForeignKeyViolation) && database.ConstraintName(err) == "places_sub_district_id_fkey" {
			return domain.ErrSubDistrictInUse
		}
		return err
	}
	if deleted == 0 {
		return domain.ErrSubDistrictNotFound
	}

	return nil
}

// DeleteMedia removes the photo, queues its file for deletion, and shifts later photos up one slot.
// The shift closes the gap, so the next added photo goes to the end, not into the hole.
func (r *PostgresRepository) DeleteMedia(ctx context.Context, id, mediaID int64) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var locked int64
	err = tx.GetContext(ctx, &locked, `SELECT id FROM SUB_DISTRICTS WHERE id = $1 FOR NO KEY UPDATE`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrMediaNotFound
		}
		return err
	}

	var position int
	err = tx.GetContext(
		ctx,
		&position,
		`WITH deleted AS (
			DELETE FROM MEDIAS m
			USING SUB_DISTRICT_MEDIAS sdm
			WHERE sdm.media_id = m.id AND sdm.sub_district_id = $1 AND m.id = $2
			RETURNING m.key, sdm.position
		), queued AS (
			INSERT INTO STORAGE_DELETIONS (key)
			SELECT key FROM deleted
			ON CONFLICT (key) DO NOTHING
		)
		SELECT position FROM deleted`,
		id,
		mediaID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrMediaNotFound
		}
		return err
	}

	_, err = tx.ExecContext(
		ctx,
		`UPDATE SUB_DISTRICT_MEDIAS SET position = position - 1 WHERE sub_district_id = $1 AND position > $2`,
		id,
		position,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// subDistrictColumns selects a sub-district from `sd`.
const subDistrictColumns = `sd.id, sd.name, sd.slug, sd.flood_history,
	sd.safety_crime_information, sd.estimated_daily_spending, sd.boundary`

// withMedias returns sd with its photos attached.
func (r *PostgresRepository) withMedias(ctx context.Context, sd domain.SubDistrict) (*domain.SubDistrict, error) {
	subDistricts := []domain.SubDistrict{sd}
	if err := r.attachMedias(ctx, subDistricts); err != nil {
		return nil, err
	}

	return &subDistricts[0], nil
}

// attachMedias loads the photos of all the given sub-districts in one query.
// Each sub-district gets its photos in position order.
func (r *PostgresRepository) attachMedias(ctx context.Context, subDistricts []domain.SubDistrict) error {
	if len(subDistricts) == 0 {
		return nil
	}

	ids := make([]int64, 0, len(subDistricts))
	for _, sd := range subDistricts {
		ids = append(ids, sd.ID)
	}

	var rows []struct {
		SubDistrictID int64 `db:"sub_district_id"`
		domain.Media
	}

	err := r.db.SelectContext(
		ctx,
		&rows,
		`SELECT sdm.sub_district_id, m.id, m.key, m.content_type, m.size_bytes, sdm.position
		FROM SUB_DISTRICT_MEDIAS sdm
		JOIN MEDIAS m ON m.id = sdm.media_id
		WHERE sdm.sub_district_id = ANY($1)
		ORDER BY sdm.sub_district_id, sdm.position`,
		ids,
	)
	if err != nil {
		return err
	}

	mediasBySubDistrict := make(map[int64][]domain.Media, len(subDistricts))
	for _, row := range rows {
		mediasBySubDistrict[row.SubDistrictID] = append(mediasBySubDistrict[row.SubDistrictID], row.Media)
	}

	for i := range subDistricts {
		subDistricts[i].Medias = mediasBySubDistrict[subDistricts[i].ID]
		if subDistricts[i].Medias == nil {
			subDistricts[i].Medias = make([]domain.Media, 0)
		}
	}

	return nil
}
