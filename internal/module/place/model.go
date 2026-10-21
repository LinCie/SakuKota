package place

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"rantau-go/internal/httpx"
	"rantau-go/internal/postgis"
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
	Category   Category        `db:"category"`
	// IdempotencyKey and RequestHash identify a create request that the client
	// may retry. They are empty when the request carries no Idempotency-Key.
	IdempotencyKey string `db:"-"`
	RequestHash    string `db:"-"`
}

// PlaceFilter narrows the places returned by GetMany. Zero values apply no filter.
type PlaceFilter struct {
	SubDistrictID int64
	// CategoryIDs matches places in any of the given categories.
	CategoryIDs []int64
	// Query is matched case-insensitively against the name, address,
	// short description, and category name.
	Query string
}

// PlacePatch holds the fields a client may change on a place. A field that is
// not Set keeps its current value.
type PlacePatch struct {
	SubDistrictID       httpx.Optional[int64]
	CategoryID          httpx.Optional[int64]
	Name                httpx.Optional[string]
	Address             httpx.Optional[string]
	Location            httpx.Optional[postgis.Point]
	OperatingHours      httpx.Optional[string]
	EstimatedPriceRange httpx.Optional[string]
	ShortDescription    httpx.Optional[string]
	Attributes          httpx.Optional[json.RawMessage]
}

// Category represents a row in the CATEGORIES table.
type Category struct {
	ID     int64           `db:"id"`
	Name   string          `db:"name"`
	Slug   string          `db:"slug"`
	Fields []CategoryField `db:"-"`
}

// Data types a CategoryField can have. They are validated by the service, not
// the database, so adding one needs no migration.
const (
	FieldTypeText    = "text"
	FieldTypeNumber  = "number"
	FieldTypeBoolean = "boolean"
	FieldTypeSelect  = "select"
	// FieldTypeRange values are objects of the form {"min": 1, "max": 2}.
	FieldTypeRange = "range"
	// FieldTypeDate values are strings in YYYY-MM-DD format.
	FieldTypeDate = "date"
)

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
	// On create, nil adds the field last.
	SortOrder *int32 `db:"sort_order"`
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
