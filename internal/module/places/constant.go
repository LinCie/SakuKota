package places

import "time"

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
const maxLimit = 40

// listCacheKeyPrefix namespaces cached place indexes in Redis.
const listCacheKeyPrefix = "places:index:"

// listCacheTTL is how long a cached index is served before it is read again.
// Nothing clears indexes sooner, so it is also how stale a list can be.
const listCacheTTL = 30 * time.Second

// cacheStatusKey is the context key under which a request carries its *cacheStatus.
const cacheStatusKey = "places.cache_status"

// cacheHeader is the response header that reports the cacheStatus of a list request.
const cacheHeader = "X-Cache"

// How a list request used the index cache.
const (
	// cacheHit means the index came from Redis.
	cacheHit cacheStatus = "HIT"
	// cacheMiss means the index was read from the repository, including after a Redis failure.
	cacheMiss cacheStatus = "MISS"
	// cacheBypass means the filter is not cached, so the repository answered directly.
	cacheBypass cacheStatus = "BYPASS"
)
