package subdistrict

// SubDistrict represents a row in the SUB_DISTRICTS table.
type SubDistrict struct {
	ID                     int64
	Name                   string
	Slug                   string
	FloodHistory           *string
	SafetyCrimeInformation *string
	EstimatedDailySpending *float64
	StaticMapImageURL      *string
}
