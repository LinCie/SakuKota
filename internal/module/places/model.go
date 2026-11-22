package places

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"math"

	"rantau-go/internal/pkg/postgis"
	"rantau-go/internal/storage"
)

// Place is a row of PLACES, joined with its category.
type Place struct {
	ID                  int64          `db:"id"`
	SubDistrictID       int64          `db:"sub_district_id"`
	CategoryID          int64          `db:"category_id"`
	Name                string         `db:"name"`
	Address             *string        `db:"address"`
	Location            *postgis.Point `db:"location"`
	OperatingHours      *string        `db:"operating_hours"`
	EstimatedPriceRange *string        `db:"estimated_price_range"`
	ShortDescription    *string        `db:"short_description"`
	// Attributes holds the category's custom field values, keyed by CategoryField.Key.
	Attributes json.RawMessage `db:"attributes"`
	// RatingCount and RatingSum total the place's review ratings.
	// The review module maintains them; place updates leave them alone.
	RatingCount int      `db:"rating_count"`
	RatingSum   int      `db:"rating_sum"`
	Category    Category `db:"category"`
	// Medias are the place's images in position order, loaded separately by
	// the repository. They are never nil once read.
	Medias []storage.Media `db:"-"`
}

// PlaceFilter narrows the places GetMany returns. A zero field applies no filter.
type PlaceFilter struct {
	SubDistrictID int64
	// CategoryIDs matches places in any of the given categories.
	CategoryIDs []int64
	// Query matches the name, address, short description, or category name, ignoring case.
	Query string
	// Bounds keeps places located inside the box, edges included.
	Bounds *BoundingBox
	// Limit is the page size. The service replaces zero or an oversized value
	// with its maximum, so below the service it is always positive.
	Limit int
	// Offset skips this many places before the page starts. Without Bounds,
	// places are ordered by id; with Bounds, they are spread over the box (see
	// spreadPlaces).
	Offset int
}

// PlaceLocation is a place's id, category, and coordinates in degrees.
// The coordinates are nil for a place without a location.
type PlaceLocation struct {
	ID         int64    `db:"id"`
	CategoryID int64    `db:"category_id"`
	Longitude  *float64 `db:"longitude"`
	Latitude   *float64 `db:"latitude"`
}

// BoundingBox is an area in degrees.
type BoundingBox struct {
	MinLng float64
	MinLat float64
	MaxLng float64
	MaxLat float64
}

// Center returns the longitude and latitude of the middle of the box.
func (b BoundingBox) Center() (float64, float64) {
	return (b.MinLng + b.MaxLng) / 2, (b.MinLat + b.MaxLat) / 2
}

// gridCell returns the side in degrees of the square cells a page of limit places
// is spread over: the smallest power of two that splits the box into at most about
// limit cells. A power of two keeps the grid the same while the box pans at one
// zoom level. It is never below minGridCell, so an empty box still has a grid.
func (b BoundingBox) gridCell(limit int) float64 {
	side := math.Sqrt((b.MaxLng - b.MinLng) * (b.MaxLat - b.MinLat) / float64(limit))

	return math.Exp2(math.Ceil(math.Log2(max(side, minGridCell))))
}

// Category is a row of CATEGORIES. Migrations define categories; the API cannot create them.
type Category struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
	Slug string `db:"slug"`
	// Code is the single character that marks the category on the map.
	Code string `db:"code"`
	// Label is the short name clients display.
	Label string `db:"label"`
	// Color is the palette key clients style the category with.
	Color  string          `db:"color"`
	Fields []CategoryField `db:"-"`
}

// CategoryField is a row of CATEGORY_FIELDS: a custom field that places of the
// category carry in their Attributes.
type CategoryField struct {
	ID         int64        `db:"id"`
	CategoryID int64        `db:"category_id"`
	Key        string       `db:"key"`
	Label      string       `db:"label"`
	DataType   string       `db:"data_type"`
	Required   bool         `db:"required"`
	Options    FieldOptions `db:"options"`
	// SortOrder is the field's 1-based position among its category's fields.
	SortOrder int32 `db:"sort_order"`
}

// FieldOptions are the allowed values of a select field, stored as a JSONB array.
// It is nil for other data types.
type FieldOptions []string

// Scan reads a JSON array from the database (database/sql's Scanner).
func (o *FieldOptions) Scan(src any) error {
	switch value := src.(type) {
	case nil:
		*o = nil
		return nil
	case []byte:
		return json.Unmarshal(value, o)
	default:
		return fmt.Errorf("place: cannot scan %T into FieldOptions", src)
	}
}

// Value writes the options as a JSON array, or NULL if nil (database/sql's Valuer).
func (o FieldOptions) Value() (driver.Value, error) {
	if o == nil {
		return nil, nil
	}

	return json.Marshal(o)
}
