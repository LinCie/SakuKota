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
	Categories          []Category     `db:"-"`
}

// PlaceFilter narrows the places returned by GetMany. Zero values apply no filter.
type PlaceFilter struct {
	SubDistrictID int64
	// CategoryIDs matches places having any of the given categories.
	CategoryIDs []int64
	// Query is matched case-insensitively against the name, address,
	// short description, and category names.
	Query string
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
