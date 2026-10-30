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

// Indonesia's bounding box in degrees: west and east longitude, south and north latitude.
const (
	indonesiaMinLongitude = 95
	indonesiaMaxLongitude = 141
	indonesiaMinLatitude  = -11
	indonesiaMaxLatitude  = 6
)

// maxNameLength is the size of the VARCHAR(100) columns holding category names
// and slugs and field labels. VARCHAR counts characters, not bytes.
const maxNameLength = 100

// maxPlaceNameLength is the size of the VARCHAR(150) column holding place names.
const maxPlaceNameLength = 150
