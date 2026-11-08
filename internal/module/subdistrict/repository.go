package subdistrict

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"rantau-go/internal/database"
)

// Repository defines the persistence operations required by the service.
//
//go:generate mockgen -source=repository.go -destination=mock/repository.go -package=mock
type Repository interface {
	Create(ctx context.Context, sd *SubDistrict) (*SubDistrict, error)
	GetMany(ctx context.Context) ([]SubDistrict, error)
	GetSubDistrict(ctx context.Context, id int64) (*SubDistrict, error)
	Update(ctx context.Context, sd *SubDistrict) (*SubDistrict, error)
	// Delete removes the sub-district and queues its images for deletion. It
	// returns ErrSubDistrictNotFound, or ErrSubDistrictInUse if places still
	// reference it.
	Delete(ctx context.Context, id int64) error
	// SetImageKey stores key as the sub-district's image in slot and returns
	// the sub-district.
	SetImageKey(ctx context.Context, id int64, slot ImageSlot, key string) (*SubDistrict, error)
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

// subDistrictColumns lists the columns scanned into a SubDistrict, in order.
const subDistrictColumns = `id, name, slug, flood_history, safety_crime_information,
	estimated_daily_spending, image_key, static_map_image_key`

// imageKeyColumns maps each image slot to the column holding its key.
var imageKeyColumns = map[ImageSlot]string{
	SlotImage:     "image_key",
	SlotStaticMap: "static_map_image_key",
}

// Create inserts the sub-district and returns it, or ErrSubDistrictExists if the
// name or slug is taken.
func (r *postgresRepository) Create(ctx context.Context, sd *SubDistrict) (*SubDistrict, error) {
	var created SubDistrict

	err := r.db.GetContext(
		ctx,
		&created,
		`INSERT INTO SUB_DISTRICTS (name, slug, flood_history,
			safety_crime_information, estimated_daily_spending)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+subDistrictColumns,
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

	return &created, nil
}

// GetMany returns every sub-district in id order.
func (r *postgresRepository) GetMany(ctx context.Context) ([]SubDistrict, error) {
	subDistricts := make([]SubDistrict, 0)

	err := r.db.SelectContext(
		ctx,
		&subDistricts,
		`SELECT `+subDistrictColumns+` FROM SUB_DISTRICTS ORDER BY id`,
	)
	if err != nil {
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
		`SELECT `+subDistrictColumns+` FROM SUB_DISTRICTS WHERE id = $1`,
		id,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSubDistrictNotFound
		}
		return nil, err
	}

	return &sd, nil
}

// Update stores the sub-district's fields and returns it, or
// ErrSubDistrictNotFound or ErrSubDistrictExists.
func (r *postgresRepository) Update(ctx context.Context, sd *SubDistrict) (*SubDistrict, error) {
	var updated SubDistrict

	err := r.db.GetContext(
		ctx,
		&updated,
		`UPDATE SUB_DISTRICTS
		SET name = $2, slug = $3, flood_history = $4, safety_crime_information = $5,
			estimated_daily_spending = $6
		WHERE id = $1
		RETURNING `+subDistrictColumns,
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

	return &updated, nil
}

// Delete removes the sub-district and queues its images for deletion, or
// returns ErrSubDistrictNotFound. A sub-district that places reference is not
// deleted: ErrSubDistrictInUse is returned instead.
func (r *postgresRepository) Delete(ctx context.Context, id int64) error {
	var deleted int

	err := r.db.GetContext(
		ctx,
		&deleted,
		`WITH deleted AS (
			DELETE FROM SUB_DISTRICTS WHERE id = $1 RETURNING image_key, static_map_image_key
		), queued AS (
			INSERT INTO STORAGE_DELETIONS (key)
			SELECT image_key FROM deleted WHERE image_key IS NOT NULL
			UNION
			SELECT static_map_image_key FROM deleted WHERE static_map_image_key IS NOT NULL
			ON CONFLICT (key) DO NOTHING
		)
		SELECT COUNT(*) FROM deleted`,
		id,
	)
	if err != nil {
		if database.IsCode(err, database.ForeignKeyViolation) {
			return ErrSubDistrictInUse
		}
		return err
	}
	if deleted == 0 {
		return ErrSubDistrictNotFound
	}

	return nil
}

// SetImageKey stores key as the sub-district's image in slot and returns the
// sub-district. The previous image in that slot is queued for deletion. The row
// is locked while its previous key is read, so a concurrent replace cannot
// leave an image unqueued.
func (r *postgresRepository) SetImageKey(ctx context.Context, id int64, slot ImageSlot, key string) (*SubDistrict, error) {
	column, ok := imageKeyColumns[slot]
	if !ok {
		return nil, fmt.Errorf("unknown image slot %q", slot)
	}

	var updated SubDistrict

	err := r.db.GetContext(
		ctx,
		&updated,
		`WITH previous AS (
			SELECT id AS sub_district_id, `+column+` AS previous_key
			FROM SUB_DISTRICTS
			WHERE id = $1
			FOR UPDATE
		), updated AS (
			UPDATE SUB_DISTRICTS
			SET `+column+` = $2
			FROM previous
			WHERE SUB_DISTRICTS.id = previous.sub_district_id
			RETURNING `+subDistrictColumns+`
		), queued AS (
			INSERT INTO STORAGE_DELETIONS (key)
			SELECT previous_key FROM previous
			WHERE previous_key IS NOT NULL AND previous_key <> $2
			ON CONFLICT (key) DO NOTHING
		)
		SELECT `+subDistrictColumns+` FROM updated`,
		id,
		key,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSubDistrictNotFound
		}
		return nil, err
	}

	return &updated, nil
}
