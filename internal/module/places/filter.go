package places

import "math"

// Filter narrows the places List returns. A zero field applies no filter.
type Filter struct {
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

// Location is a place's id, category, and coordinates in degrees.
// The coordinates are nil for a place without a location.
type Location struct {
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

// contains reports whether the point is inside the box, edges included.
// It is false for NaN coordinates, because every NaN comparison is false.
func (b BoundingBox) contains(longitude, latitude float64) bool {
	return longitude >= b.MinLng && longitude <= b.MaxLng && latitude >= b.MinLat && latitude <= b.MaxLat
}

// gridCell returns the side in degrees of the square cells a page of limit places
// is spread over: the smallest power of two that splits the box into at most about
// limit cells. A power of two keeps the grid the same while the box pans at one
// zoom level. It is never below minGridCell, so an empty box still has a grid.
func (b BoundingBox) gridCell(limit int) float64 {
	side := math.Sqrt((b.MaxLng - b.MinLng) * (b.MaxLat - b.MinLat) / float64(limit))

	return math.Exp2(math.Ceil(math.Log2(max(side, minGridCell))))
}
