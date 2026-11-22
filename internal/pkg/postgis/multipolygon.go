package postgis

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/ewkbhex"
)

// MultiPolygon is an area made of one or more polygons, such as a kecamatan's
// boundary. It reads a GeoJSON Polygon or MultiPolygon, writes a GeoJSON
// MultiPolygon, and is a PostGIS geometry(MultiPolygon, 4326) in the database.
type MultiPolygon struct {
	// polygons holds each polygon's rings; a ring is a list of [longitude, latitude].
	polygons [][][]geom.Coord
}

// multiPolygonJSON is a GeoJSON Polygon or MultiPolygon before its coordinates are read.
type multiPolygonJSON struct {
	Type        string          `json:"type"`
	Coordinates json.RawMessage `json:"coordinates"`
}

// MarshalJSON writes the area as a GeoJSON MultiPolygon.
func (m MultiPolygon) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type        string           `json:"type"`
		Coordinates [][][]geom.Coord `json:"coordinates"`
	}{Type: "MultiPolygon", Coordinates: m.polygons})
}

// UnmarshalJSON reads a GeoJSON Polygon or MultiPolygon. Every ring must be
// closed and have at least four positions, and every position must be a
// valid [longitude, latitude].
func (m *MultiPolygon) UnmarshalJSON(data []byte) error {
	var g multiPolygonJSON
	if err := json.Unmarshal(data, &g); err != nil {
		return fmt.Errorf("postgis: decode GeoJSON: %w", err)
	}

	var polygons [][][]geom.Coord
	switch g.Type {
	case "MultiPolygon":
		if err := json.Unmarshal(g.Coordinates, &polygons); err != nil {
			return fmt.Errorf("postgis: decode MultiPolygon coordinates: %w", err)
		}
	case "Polygon":
		var rings [][]geom.Coord
		if err := json.Unmarshal(g.Coordinates, &rings); err != nil {
			return fmt.Errorf("postgis: decode Polygon coordinates: %w", err)
		}
		polygons = [][][]geom.Coord{rings}
	default:
		return errors.New("postgis: expected a GeoJSON Polygon or MultiPolygon")
	}

	if err := checkPolygons(polygons); err != nil {
		return err
	}

	m.polygons = polygons

	return nil
}

// Scan implements database/sql.Scanner for PostGIS geometry values, which
// database/sql receives as hexadecimal EWKB text.
func (m *MultiPolygon) Scan(src any) error {
	var hex string
	switch value := src.(type) {
	case []byte:
		hex = string(value)
	case string:
		hex = value
	default: // NULL lands here too
		return fmt.Errorf("postgis: cannot scan %T into MultiPolygon", src)
	}

	g, err := ewkbhex.Decode(hex)
	if err != nil {
		return fmt.Errorf("postgis: decode EWKB: %w", err)
	}

	mp, ok := g.(*geom.MultiPolygon)
	if !ok || mp.Empty() {
		return fmt.Errorf("postgis: expected a non-empty MultiPolygon, got %T", g)
	}

	m.polygons = mp.Coords()

	return nil
}

// Value implements database/sql/driver.Valuer, encoding the area as hex EWKB.
func (m MultiPolygon) Value() (driver.Value, error) {
	mp, err := geom.NewMultiPolygon(geom.XY).SetCoords(m.polygons)
	if err != nil {
		return nil, fmt.Errorf("postgis: encode MultiPolygon: %w", err)
	}

	return ewkbhex.Encode(mp.SetSRID(SRID), ewkbhex.NDR)
}

// checkPolygons reports whether polygons is a well-formed GeoJSON area: at
// least one polygon, each with at least one closed ring of four or more
// valid [longitude, latitude] positions.
func checkPolygons(polygons [][][]geom.Coord) error {
	if len(polygons) == 0 {
		return errors.New("postgis: expected at least one polygon")
	}

	for _, rings := range polygons {
		if len(rings) == 0 {
			return errors.New("postgis: expected at least one ring in each polygon")
		}

		for _, ring := range rings {
			if len(ring) < 4 || !slices.Equal(ring[0], ring[len(ring)-1]) {
				return errors.New("postgis: expected closed rings of at least four positions")
			}

			for _, c := range ring {
				if len(c) != 2 {
					return errors.New("postgis: expected [longitude, latitude] positions")
				}
				if _, err := NewPoint(c[0], c[1]); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
