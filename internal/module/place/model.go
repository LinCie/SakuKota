package place

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"rantau-go/internal/pkg/postgis"
	"rantau-go/internal/storage"
)

// Place represents a row in the PLACES table, joined with its category.
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
	// Attributes holds the values of the category's custom fields as a JSON
	// object keyed by CategoryField.Key.
	Attributes json.RawMessage `db:"attributes"`
	// RatingCount and RatingSum total the ratings of the place's reviews. The
	// review module keeps them up to date; place updates never change them.
	RatingCount int      `db:"rating_count"`
	RatingSum   int      `db:"rating_sum"`
	Category    Category `db:"category"`
	// Medias are the place's images in position order. The repository loads
	// them separately, so they are never nil once read.
	Medias []storage.Media `db:"-"`
}

// PlaceFilter narrows the places returned by GetMany. Zero values apply no filter.
type PlaceFilter struct {
	SubDistrictID int64
	// CategoryIDs matches places in any of the given categories.
	CategoryIDs []int64
	// Query is matched case-insensitively against the name, address,
	// short description, and category name.
	Query string
	// Bounds keeps places located inside the box, edges included.
	Bounds *BoundingBox
	// Limit caps the number of places returned, lowest ids first. The service
	// replaces zero, or a value above its maximum, with the maximum.
	Limit int
	// Offset skips this many places, lowest ids first, before Limit applies.
	Offset int
}

// BoundingBox is an area in degrees, from its west and south edges to its
// east and north edges.
type BoundingBox struct {
	MinLng float64
	MinLat float64
	MaxLng float64
	MaxLat float64
}

// Category represents a row in the CATEGORIES table. Categories are defined by
// the migrations, not created through the API.
type Category struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
	Slug string `db:"slug"`
	// Code is the single character that marks the category on a map sign.
	Code string `db:"code"`
	// Label is the short name clients display.
	Label string `db:"label"`
	// Color is the palette key clients style the category with.
	Color  string          `db:"color"`
	Fields []CategoryField `db:"-"`
}

// CategoryField represents a row in the CATEGORY_FIELDS table: a custom field
// that places in the category carry in their Attributes.
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

// FieldOptions are the allowed values of a select field, stored as a JSONB
// array. It is nil for other data types.
type FieldOptions []string

// Scan implements database/sql.Scanner.
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

// Value implements database/sql/driver.Valuer.
func (o FieldOptions) Value() (driver.Value, error) {
	if o == nil {
		return nil, nil
	}

	return json.Marshal(o)
}
