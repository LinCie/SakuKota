package domain

import "rantau-go/internal/pkg/postgis"

const (
	// MaxSubDistrictNameLength is the size of the VARCHAR(100) name and slug
	// columns. VARCHAR counts characters, not bytes.
	MaxSubDistrictNameLength = 100

	// MaxEstimatedDailySpending is the largest value a DECIMAL(10,2) column holds.
	MaxEstimatedDailySpending = 99999999.99
)

// SubDistrict represents a row in the SUB_DISTRICTS table.
type SubDistrict struct {
	ID                     int64    `db:"id"`
	Name                   string   `db:"name"`
	Slug                   string   `db:"slug"`
	FloodHistory           *string  `db:"flood_history"`
	SafetyCrimeInformation *string  `db:"safety_crime_information"`
	EstimatedDailySpending *float64 `db:"estimated_daily_spending"`
	// Boundary is the sub-district's area, or nil if it has none.
	Boundary *postgis.MultiPolygon `db:"boundary"`
	// Medias are the sub-district's images in position order. The repository
	// loads them separately, so they are never nil once read.
	Medias []Media `db:"-"`
}
