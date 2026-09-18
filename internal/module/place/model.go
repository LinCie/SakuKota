package place

import "rantau-go/internal/postgis"

// Place represents a row in the PLACES table.
type Place struct {
	ID                  int64          `db:"id"`
	SubDistrictID       int64          `db:"sub_district_id"`
	Name                string         `db:"name"`
	Address             *string        `db:"address"`
	Location            *postgis.Point `db:"location"`
	OperatingHours      *string        `db:"operating_hours"`
	EstimatedPriceRange *string        `db:"estimated_price_range"`
	ShortDescription    *string        `db:"short_description"`
}

// PlaceCategory represents a row in the PLACE_CATEGORIES table.
type PlaceCategory struct {
	PlaceID    int64 `db:"place_id"`
	CategoryID int64 `db:"category_id"`
}

// Category represents a row in the CATEGORIES table.
type Category struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
	Slug string `db:"slug"`
}
