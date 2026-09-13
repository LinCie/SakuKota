package place

import "rantau-go/internal/postgis"

// Place represents a row in the PLACES table.
type Place struct {
	ID                  int64
	SubDistrictID       int64
	Name                string
	Address             *string
	Location            *postgis.Point
	OperatingHours      *string
	EstimatedPriceRange *string
	ShortDescription    *string
}

// PlaceCategory represents a row in the PLACE_CATEGORIES table.
type PlaceCategory struct {
	PlaceID    int64
	CategoryID int64
}

// Category represents a row in the CATEGORIES table.
type Category struct {
	ID   int64
	Name string
	Slug string
}
