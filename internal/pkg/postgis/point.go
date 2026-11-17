package postgis

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/ewkbhex"
)

// SRID is the spatial reference ID stored with every geometry. It tells PostGIS
// what the coordinates mean: 4326 is WGS 84, the longitude/latitude in degrees
// that GPS and web maps use.
const SRID = 4326

// Point is a longitude/latitude pair. It reads and writes GeoJSON in the API
// and a PostGIS geometry(Point, 4326) in the database.
type Point struct {
	lng, lat float64
}

// NewPoint makes a point from longitude and latitude.
// It returns an error if either value is outside its valid range.
func NewPoint(longitude, latitude float64) (*Point, error) {
	if longitude < -180 || longitude > 180 {
		return nil, fmt.Errorf("postgis: longitude out of range: %f", longitude)
	}

	if latitude < -90 || latitude > 90 {
		return nil, fmt.Errorf("postgis: latitude out of range: %f", latitude)
	}

	return &Point{lng: longitude, lat: latitude}, nil
}

// MustPoint is NewPoint for values known to be valid. It panics on bad input.
func MustPoint(longitude, latitude float64) *Point {
	p, err := NewPoint(longitude, latitude)
	if err != nil {
		panic(err)
	}

	return p
}

// X returns the longitude.
func (p Point) X() float64 { return p.lng }

// Y returns the latitude.
func (p Point) Y() float64 { return p.lat }

// geoJSON is a GeoJSON Point: {"type": "Point", "coordinates": [longitude, latitude]}.
type geoJSON struct {
	Type        string    `json:"type"`
	Coordinates []float64 `json:"coordinates"`
}

// MarshalJSON writes the point as a GeoJSON Point.
func (p Point) MarshalJSON() ([]byte, error) {
	return json.Marshal(geoJSON{Type: "Point", Coordinates: []float64{p.lng, p.lat}})
}

// UnmarshalJSON reads a GeoJSON Point.
func (p *Point) UnmarshalJSON(data []byte) error {
	var g geoJSON
	if err := json.Unmarshal(data, &g); err != nil {
		return fmt.Errorf("postgis: decode GeoJSON: %w", err)
	}

	if g.Type != "Point" || len(g.Coordinates) != 2 {
		return errors.New("postgis: expected a GeoJSON Point with [longitude, latitude]")
	}

	decoded, err := NewPoint(g.Coordinates[0], g.Coordinates[1])
	if err != nil {
		return err
	}

	*p = *decoded

	return nil
}

// Scan implements database/sql.Scanner for PostGIS geometry values, which
// database/sql receives as hexadecimal EWKB text.
func (p *Point) Scan(src any) error {
	var hex string
	switch value := src.(type) {
	case []byte:
		hex = string(value)
	case string:
		hex = value
	default: // NULL lands here too
		return fmt.Errorf("postgis: cannot scan %T into Point", src)
	}

	g, err := ewkbhex.Decode(hex)
	if err != nil {
		return fmt.Errorf("postgis: decode EWKB: %w", err)
	}

	point, ok := g.(*geom.Point)
	if !ok || point.Empty() {
		return fmt.Errorf("postgis: expected a non-empty Point, got %T", g)
	}

	*p = Point{lng: point.X(), lat: point.Y()}

	return nil
}

// Value implements database/sql/driver.Valuer, encoding the point as hex EWKB,
// which PostGIS accepts as input for geometry and geography columns.
func (p Point) Value() (driver.Value, error) {
	return ewkbhex.Encode(geom.NewPointFlat(geom.XY, []float64{p.lng, p.lat}).SetSRID(SRID), ewkbhex.NDR)
}
