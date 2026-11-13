package places

// Data types a CategoryField can have. A CHECK on CATEGORY_FIELDS.data_type
// allows exactly these, so adding one needs a migration as well.
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

// maxLimit is the most places a list returns. A request without a limit, or
// with a larger one, gets this many; the total still counts every match.
const maxLimit = 1000
