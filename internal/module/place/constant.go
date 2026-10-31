package place

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

// Lengths in characters, counted after trimming spaces. They match the VARCHAR
// columns of PLACES.
const (
	MaxNameLength                = 150
	MaxAddressLength             = 255
	MaxOperatingHoursLength      = 255
	MaxEstimatedPriceRangeLength = 100
)
