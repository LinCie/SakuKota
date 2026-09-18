package subdistrict

// SubDistrict represents a row in the SUB_DISTRICTS table.
type SubDistrict struct {
	ID                     int64    `db:"id"`
	Name                   string   `db:"name"`
	Slug                   string   `db:"slug"`
	FloodHistory           *string  `db:"flood_history"`
	SafetyCrimeInformation *string  `db:"safety_crime_information"`
	EstimatedDailySpending *float64 `db:"estimated_daily_spending"`
	StaticMapImageURL      *string  `db:"static_map_image_url"`
}
