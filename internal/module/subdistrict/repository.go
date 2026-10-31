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
	GetMany(ctx context.Context) ([]SubDistrict, error)
	Create(ctx context.Context, sd *SubDistrict) (*SubDistrict, error)
	GetSubDistrict(ctx context.Context, id int64) (*SubDistrict, error)
	Update(ctx context.Context, sd *SubDistrict) (*SubDistrict, error)
	// SetImageKey stores key as the sub-district's image in slot and returns
	// the sub-district.
	SetImageKey(ctx context.Context, id int64, slot ImageSlot, key string) (*SubDistrict, error)
	// Delete removes the sub-district and returns the keys of its images.
	Delete(ctx context.Context, id int64) ([]string, error)
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

func (r *postgresRepository) SetImageKey(ctx context.Context, id int64, slot ImageSlot, key string) (*SubDistrict, error) {
	column, ok := imageKeyColumns[slot]
	if !ok {
		return nil, fmt.Errorf("unknown image slot %q", slot)
	}

	var updated SubDistrict

	err := r.db.GetContext(
		ctx,
		&updated,
		`UPDATE SUB_DISTRICTS SET `+column+` = $2 WHERE id = $1 RETURNING `+subDistrictColumns,
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

// Delete removes the sub-district and returns the keys of its images: the
// image first, then the static map.
func (r *postgresRepository) Delete(ctx context.Context, id int64) ([]string, error) {
	var deleted SubDistrict

	err := r.db.GetContext(
		ctx,
		&deleted,
		`DELETE FROM SUB_DISTRICTS WHERE id = $1 RETURNING image_key, static_map_image_key`,
		id,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSubDistrictNotFound
		}
		if database.IsCode(err, database.ForeignKeyViolation) {
			return nil, ErrSubDistrictInUse
		}
		return nil, err
	}

	var imageKeys []string
	for _, key := range []*string{deleted.ImageKey, deleted.StaticMapImageKey} {
		if key != nil {
			imageKeys = append(imageKeys, *key)
		}
	}

	return imageKeys, nil
}
