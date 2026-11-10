package postgis

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/ewkbhex"
	"github.com/twpayne/go-geom/encoding/geojson"
)

const SRID = 4326

// Point is a PostGIS point backed by a go-geom Point.
//
// It embeds geom.Point so its geometry methods remain available while allowing
// the API response to use a JSON representation of the point.
type Point struct {
	geom.Point
}

func NewPoint(longitude, latitude float64) (*Point, error) {
	if longitude < -180 || longitude > 180 {
		return nil, fmt.Errorf(
			"postgis: longitude out of range: %f",
			longitude,
		)
	}

	if latitude < -90 || latitude > 90 {
		return nil, fmt.Errorf(
			"postgis: latitude out of range: %f",
			latitude,
		)
	}

	return &Point{
		Point: *geom.NewPointFlat(
			geom.XY,
			[]float64{
				longitude,
				latitude,
			},
		).SetSRID(SRID),
	}, nil
}

func MustPoint(longitude, latitude float64) *Point {
	p, err := NewPoint(longitude, latitude)
	if err != nil {
		panic(err)
	}

	return p
}

// MarshalJSON encodes a point as a GeoJSON Point instead of exposing the
// unexported fields used internally by go-geom. It marshals once, unlike
// geojson.Marshal, which marshals the coordinates and then the geometry.
func (p Point) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type        string     `json:"type"`
		Coordinates [2]float64 `json:"coordinates"`
	}{"Point", [2]float64{p.X(), p.Y()}})
}

// UnmarshalJSON decodes a GeoJSON Point whose coordinates are
// [longitude, latitude].
func (p *Point) UnmarshalJSON(data []byte) error {
	var g geom.T
	if err := geojson.Unmarshal(data, &g); err != nil {
		return fmt.Errorf("postgis: decode GeoJSON: %w", err)
	}

	point, ok := g.(*geom.Point)
	if !ok {
		return fmt.Errorf("postgis: expected Point, got %T", g)
	}

	if point.Layout() != geom.XY || point.Empty() {
		return fmt.Errorf("postgis: expected non-empty XY point")
	}

	decoded, err := NewPoint(point.X(), point.Y())
	if err != nil {
		return err
	}

	*p = *decoded

	return nil
}

// Scan implements database/sql.Scanner for PostGIS geometry values, which
// database/sql receives as hexadecimal EWKB text.
func (p *Point) Scan(src any) error {
	if p == nil {
		return fmt.Errorf("postgis: cannot scan Point into nil receiver")
	}

	if src == nil {
		return fmt.Errorf("postgis: cannot scan NULL into Point")
	}

	var data string
	switch value := src.(type) {
	case []byte:
		data = string(value)
	case string:
		data = value
	default:
		return fmt.Errorf("postgis: cannot scan %T into Point", src)
	}

	g, err := ewkbhex.Decode(data)
	if err != nil {
		return fmt.Errorf("postgis: decode EWKB: %w", err)
	}

	point, ok := g.(*geom.Point)
	if !ok {
		return fmt.Errorf("postgis: expected Point, got %T", g)
	}

	decoded := Point{Point: *point}
	if err := validatePoint(&decoded); err != nil {
		return fmt.Errorf("postgis: invalid point from database: %w", err)
	}

	*p = decoded

	return nil
}

// Value implements database/sql/driver.Valuer, encoding the point as hex EWKB,
// which PostGIS accepts as input for geometry and geography columns.
func (p Point) Value() (driver.Value, error) {
	if err := validatePoint(&p); err != nil {
		return nil, fmt.Errorf("postgis: invalid point: %w", err)
	}

	data, err := ewkbhex.Encode(&p.Point, ewkbhex.NDR)
	if err != nil {
		return nil, fmt.Errorf("postgis: encode EWKB: %w", err)
	}

	return data, nil
}

func validatePoint(p *Point) error {
	if p == nil {
		return fmt.Errorf("point is nil")
	}

	if p.Layout() != geom.XY {
		return fmt.Errorf(
			"expected XY layout, got %s",
			p.Layout(),
		)
	}

	if p.SRID() != SRID {
		return fmt.Errorf(
			"expected SRID %d, got %d",
			SRID,
			p.SRID(),
		)
	}

	if p.Empty() {
		return fmt.Errorf("empty point is not supported")
	}

	return nil
}
