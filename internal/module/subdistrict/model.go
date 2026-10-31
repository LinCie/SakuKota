package subdistrict

import "rantau-go/internal/httpx"

// SubDistrict represents a row in the SUB_DISTRICTS table.
type SubDistrict struct {
	ID                     int64    `db:"id"`
	Name                   string   `db:"name"`
	Slug                   string   `db:"slug"`
	FloodHistory           *string  `db:"flood_history"`
	SafetyCrimeInformation *string  `db:"safety_crime_information"`
	EstimatedDailySpending *float64 `db:"estimated_daily_spending"`
	// ImageKey and StaticMapImageKey are the storage keys of the sub-district's
	// images, or nil for an image it does not have.
	ImageKey          *string `db:"image_key"`
	StaticMapImageKey *string `db:"static_map_image_key"`
}

// SubDistrictPatch holds the fields a client may change on a sub-district. A
// field that is not Set keeps its current value.
type SubDistrictPatch struct {
	Name                   httpx.Optional[string]
	Slug                   httpx.Optional[string]
	FloodHistory           httpx.Optional[string]
	SafetyCrimeInformation httpx.Optional[string]
	EstimatedDailySpending httpx.Optional[float64]
}

// ImageSlot names one of a sub-district's images. It is also the image's path
// segment in the API and in storage.
type ImageSlot string

const (
	SlotImage     ImageSlot = "image"
	SlotStaticMap ImageSlot = "static-map"
)
