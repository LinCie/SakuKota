package subdistrict

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"rantau-go/internal/database"
	"rantau-go/internal/storage"
)

// Repository defines the persistence operations required by the service.
//
//go:generate mockgen -source=repository.go -destination=mock/repository.go -package=mock
type Repository interface {
	Create(ctx context.Context, sd *SubDistrict) (*SubDistrict, error)
	GetMany(ctx context.Context) ([]SubDistrict, error)
	GetSubDistrict(ctx context.Context, id int64) (*SubDistrict, error)
	Update(ctx context.Context, sd *SubDistrict) (*SubDistrict, error)
	// Delete removes the sub-district with its medias and static map, and
	// queues their images for deletion. It returns ErrSubDistrictNotFound, or
	// ErrSubDistrictInUse if places still reference it.
	Delete(ctx context.Context, id int64) error
	// AddMedia stores m at the sub-district's lowest free position and returns
	// the sub-district. It returns ErrSubDistrictNotFound, or
	// storage.ErrMediaLimitReached if it already has storage.MaxMedias.
	AddMedia(ctx context.Context, id int64, m storage.Media) (*SubDistrict, error)
	// DeleteMedia removes the sub-district's media, queues its image for
	// deletion, and shifts the later medias down to close the gap. It returns
	// storage.ErrMediaNotFound if the sub-district has no such media.
	DeleteMedia(ctx context.Context, id, mediaID int64) error
	// SetStaticMap stores m as the sub-district's static map and returns the
	// sub-district. The previous static map is removed and its image queued
	// for deletion.
	SetStaticMap(ctx context.Context, id int64, m storage.Media) (*SubDistrict, error)
}

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

// subDistrictColumns selects a sub-district from `sd` and the key of its
// static map from `sm`. Queries add staticMapJoin to provide `sm`.
const subDistrictColumns = `sd.id, sd.name, sd.slug, sd.flood_history,
	sd.safety_crime_information, sd.estimated_daily_spending,
	sm.key AS static_map_image_key`

// staticMapJoin joins the static map of `sd` as `sm`.
const staticMapJoin = ` LEFT JOIN MEDIAS sm ON sm.id = sd.static_map_media_id`

// Create inserts the sub-district and returns it, or ErrSubDistrictExists if the
// name or slug is taken.
func (r *postgresRepository) Create(ctx context.Context, sd *SubDistrict) (*SubDistrict, error) {
	var created SubDistrict

	err := r.db.GetContext(
		ctx,
		&created,
		`WITH sd AS (
			INSERT INTO SUB_DISTRICTS (name, slug, flood_history,
				safety_crime_information, estimated_daily_spending)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING *
		)
		SELECT `+subDistrictColumns+` FROM sd`+staticMapJoin,
		sd.Name,
		sd.Slug,
		sd.FloodHistory,
		sd.SafetyCrimeInformation,
		sd.EstimatedDailySpending,
	)
	if err != nil {
		if database.IsCode(err, database.UniqueViolation) {
			return nil, ErrSubDistrictExists
		}
		return nil, err
	}

	return r.withMedias(ctx, created)
}

// GetMany returns every sub-district in id order.
func (r *postgresRepository) GetMany(ctx context.Context) ([]SubDistrict, error) {
	subDistricts := make([]SubDistrict, 0)

	err := r.db.SelectContext(
		ctx,
		&subDistricts,
		`SELECT `+subDistrictColumns+` FROM SUB_DISTRICTS sd`+staticMapJoin+` ORDER BY sd.id`,
	)
	if err != nil {
		return nil, err
	}

	if err := r.attachMedias(ctx, subDistricts); err != nil {
		return nil, err
	}

	return subDistricts, nil
}

// GetSubDistrict returns the sub-district with the id, or ErrSubDistrictNotFound.
func (r *postgresRepository) GetSubDistrict(ctx context.Context, id int64) (*SubDistrict, error) {
	var sd SubDistrict

	err := r.db.GetContext(
		ctx,
		&sd,
		`SELECT `+subDistrictColumns+` FROM SUB_DISTRICTS sd`+staticMapJoin+` WHERE sd.id = $1`,
		id,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSubDistrictNotFound
		}
		return nil, err
	}

	return r.withMedias(ctx, sd)
}

// Update stores the sub-district's fields and returns it, or
// ErrSubDistrictNotFound or ErrSubDistrictExists.
func (r *postgresRepository) Update(ctx context.Context, sd *SubDistrict) (*SubDistrict, error) {
	var updated SubDistrict

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
		SELECT `+subDistrictColumns+` FROM sd`+staticMapJoin,
		sd.ID,
		sd.Name,
		sd.Slug,
		sd.FloodHistory,
		sd.SafetyCrimeInformation,
		sd.EstimatedDailySpending,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSubDistrictNotFound
		}
		if database.IsCode(err, database.UniqueViolation) {
			return nil, ErrSubDistrictExists
		}
		return nil, err
	}

	return r.withMedias(ctx, updated)
}

// Delete removes the sub-district with its medias and static map and queues
// their images for deletion, or returns ErrSubDistrictNotFound. A sub-district
// that places reference is not deleted: ErrSubDistrictInUse is returned
// instead.
func (r *postgresRepository) Delete(ctx context.Context, id int64) error {
	var deleted int

	err := r.db.GetContext(
		ctx,
		&deleted,
		`WITH medias AS (
			DELETE FROM MEDIAS
			WHERE id IN (SELECT media_id FROM SUB_DISTRICT_MEDIAS WHERE sub_district_id = $1)
				OR id = (SELECT static_map_media_id FROM SUB_DISTRICTS WHERE id = $1)
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
		if database.IsCode(err, database.ForeignKeyViolation) &&
			database.ConstraintName(err) == "places_sub_district_id_fkey" {
			return ErrSubDistrictInUse
		}
		return err
	}
	if deleted == 0 {
		return ErrSubDistrictNotFound
	}

	return nil
}

// AddMedia stores m at the sub-district's lowest free position and returns the
// sub-district, or ErrSubDistrictNotFound or storage.ErrMediaLimitReached. The
// sub-district row is locked while a free position is chosen, so concurrent
// adds cannot take the same position or exceed the limit.
func (r *postgresRepository) AddMedia(ctx context.Context, id int64, m storage.Media) (*SubDistrict, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var locked int64
	err = tx.GetContext(ctx, &locked, `SELECT id FROM SUB_DISTRICTS WHERE id = $1 FOR NO KEY UPDATE`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSubDistrictNotFound
		}
		return nil, err
	}

	var position *int
	err = tx.GetContext(
		ctx,
		&position,
		`SELECT MIN(free) FROM generate_series(1, $2::int) AS free
		WHERE free NOT IN (SELECT position FROM SUB_DISTRICT_MEDIAS WHERE sub_district_id = $1)`,
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
		INSERT INTO SUB_DISTRICT_MEDIAS (sub_district_id, media_id, position)
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

	return r.GetSubDistrict(ctx, id)
}

// DeleteMedia removes the sub-district's media, queues its image for deletion,
// and shifts the later medias down to close the gap, or returns
// storage.ErrMediaNotFound. The sub-district row is locked so a concurrent add
// cannot pick a position while the others move.
func (r *postgresRepository) DeleteMedia(ctx context.Context, id, mediaID int64) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var locked int64
	err = tx.GetContext(ctx, &locked, `SELECT id FROM SUB_DISTRICTS WHERE id = $1 FOR NO KEY UPDATE`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.ErrMediaNotFound
		}
		return err
	}

	var position *int
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
		`UPDATE SUB_DISTRICT_MEDIAS SET position = position - 1 WHERE sub_district_id = $1 AND position > $2`,
		id,
		*position,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// SetStaticMap stores m as the sub-district's static map and returns the
// sub-district, or ErrSubDistrictNotFound. The previous static map is removed
// and its image queued for deletion. The row is locked while the previous
// static map is read, so a concurrent replace cannot leave an image unqueued.
func (r *postgresRepository) SetStaticMap(ctx context.Context, id int64, m storage.Media) (*SubDistrict, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var previous *int64
	err = tx.GetContext(ctx, &previous, `SELECT static_map_media_id FROM SUB_DISTRICTS WHERE id = $1 FOR UPDATE`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSubDistrictNotFound
		}
		return nil, err
	}

	_, err = tx.ExecContext(
		ctx,
		`WITH media AS (
			INSERT INTO MEDIAS (key, content_type, size_bytes)
			VALUES ($2, $3, $4)
			RETURNING id
		)
		UPDATE SUB_DISTRICTS SET static_map_media_id = media.id
		FROM media
		WHERE SUB_DISTRICTS.id = $1`,
		id,
		m.Key,
		m.ContentType,
		m.Size,
	)
	if err != nil {
		return nil, err
	}

	if previous != nil {
		_, err = tx.ExecContext(
			ctx,
			`WITH deleted AS (
				DELETE FROM MEDIAS WHERE id = $1 RETURNING key
			)
			INSERT INTO STORAGE_DELETIONS (key)
			SELECT key FROM deleted
			ON CONFLICT (key) DO NOTHING`,
			*previous,
		)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return r.GetSubDistrict(ctx, id)
}

// withMedias returns sd with its medias attached.
func (r *postgresRepository) withMedias(ctx context.Context, sd SubDistrict) (*SubDistrict, error) {
	subDistricts := []SubDistrict{sd}
	if err := r.attachMedias(ctx, subDistricts); err != nil {
		return nil, err
	}

	return &subDistricts[0], nil
}

// attachMedias loads the medias of every given sub-district in one query and
// assigns them, in position order, to each sub-district's Medias field.
func (r *postgresRepository) attachMedias(ctx context.Context, subDistricts []SubDistrict) error {
	if len(subDistricts) == 0 {
		return nil
	}

	ids := make([]int64, 0, len(subDistricts))
	for _, sd := range subDistricts {
		ids = append(ids, sd.ID)
	}

	var rows []struct {
		SubDistrictID int64 `db:"sub_district_id"`
		storage.Media
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

	mediasBySubDistrict := make(map[int64][]storage.Media, len(subDistricts))
	for _, row := range rows {
		mediasBySubDistrict[row.SubDistrictID] = append(mediasBySubDistrict[row.SubDistrictID], row.Media)
	}

	for i := range subDistricts {
		subDistricts[i].Medias = mediasBySubDistrict[subDistricts[i].ID]
		if subDistricts[i].Medias == nil {
			subDistricts[i].Medias = make([]storage.Media, 0)
		}
	}

	return nil
}
