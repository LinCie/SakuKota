package places

import "time"

// maxLimit is the most places one page returns. The total still counts every match.
const maxLimit = 100

// listCacheKeyPrefix namespaces cached place indexes in Redis. The version
// changes with the entry layout, so an old entry is never read as a new one.
const listCacheKeyPrefix = "places:index:v2:"

// listCacheTTL is how long a cached index is served. Nothing clears it sooner,
// so it is also how stale a list can be.
const listCacheTTL = 30 * time.Second

// indexEntrySize is the bytes one place takes in an index: id, category id,
// longitude, and latitude as 8 little-endian bytes each. A place without a
// location has NaN coordinates.
const indexEntrySize = 32

// minGridCell is the smallest grid cell, in degrees (about a metre), that a
// page with bounds is spread over.
const minGridCell = 1e-5
