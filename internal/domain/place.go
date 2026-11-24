package domain

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"sakukota-go/internal/pkg/postgis"
	"sakukota-go/internal/pkg/text"
)

// Data types of a CategoryField. A CHECK on CATEGORY_FIELDS.data_type allows
// exactly these, so a new one needs a migration too.
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

// Maximum lengths in characters, counted after trimming. They match the VARCHAR columns of PLACES.
const (
	MaxPlaceNameLength                = 150
	MaxPlaceAddressLength             = 255
	MaxPlaceOperatingHoursLength      = 255
	MaxPlaceEstimatedPriceRangeLength = 100
)

// MaxPlaceBatchSize is the most places one bulk create may hold.
const MaxPlaceBatchSize = 500

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
	Medias []Media `db:"-"`
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

// NormalizePlaceName trims name and rejects it if it is empty or too long.
func NormalizePlaceName(name string) (string, error) {
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		return "", ErrPlaceNameRequired
	case utf8.RuneCountInString(name) > MaxPlaceNameLength:
		return "", ErrPlaceNameTooLong
	}

	return name, nil
}

// NormalizePlaceFields trims the optional text fields of p and turns blank ones into nil.
// It rejects any that exceed their column length.
func NormalizePlaceFields(p *Place) error {
	var ok bool

	if p.Address, ok = text.TrimOptionalMax(p.Address, MaxPlaceAddressLength); !ok {
		return ErrPlaceAddressTooLong
	}

	if p.OperatingHours, ok = text.TrimOptionalMax(p.OperatingHours, MaxPlaceOperatingHoursLength); !ok {
		return ErrPlaceOperatingHoursTooLong
	}

	if p.EstimatedPriceRange, ok = text.TrimOptionalMax(p.EstimatedPriceRange, MaxPlaceEstimatedPriceRangeLength); !ok {
		return ErrPlaceEstimatedPriceRangeTooLong
	}

	p.ShortDescription = text.TrimOptional(p.ShortDescription)

	return nil
}
