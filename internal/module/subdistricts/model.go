package subdistricts

import "rantau-go/internal/storage"

// SubDistrict represents a row in the SUB_DISTRICTS table.
type SubDistrict struct {
	ID                     int64    `db:"id"`
	Name                   string   `db:"name"`
	Slug                   string   `db:"slug"`
	FloodHistory           *string  `db:"flood_history"`
	SafetyCrimeInformation *string  `db:"safety_crime_information"`
	EstimatedDailySpending *float64 `db:"estimated_daily_spending"`
	// StaticMapImageKey is the storage key of the sub-district's static map,
	// or nil if it has none.
	StaticMapImageKey *string `db:"static_map_image_key"`
	// Medias are the sub-district's images in position order. The repository
	// loads them separately, so they are never nil once read.
	Medias []storage.Media `db:"-"`
}

// ImageSlot names where a sub-district's image is uploaded. It is also the
// image's path segment in the API and in storage.
type ImageSlot string

const (
	SlotMedias    ImageSlot = "medias"
	SlotStaticMap ImageSlot = "static-map"
)
