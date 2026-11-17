package places

import "time"

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
	MaxNameLength                = 150
	MaxAddressLength             = 255
	MaxOperatingHoursLength      = 255
	MaxEstimatedPriceRangeLength = 100
)

// maxLimit is the most places one page returns. The total still counts every match.
const maxLimit = 40

// listCacheKeyPrefix namespaces cached place indexes in Redis.
const listCacheKeyPrefix = "places:index:"

// listCacheTTL is how long a cached index is served. Nothing clears it sooner,
// so it is also how stale a list can be.
const listCacheTTL = 30 * time.Second

// indexEntrySize is the bytes one place takes in an index: id, longitude and
// latitude as 8 little-endian bytes each. A place without a location has NaN coordinates.
const indexEntrySize = 24
