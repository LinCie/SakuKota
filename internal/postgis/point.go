package postgis

import (
	"fmt"

	"github.com/twpayne/go-geom"
)

const SRID = 4326

// Point is the go-geom Point type.
//
// This is an alias, not a wrapper, so all geom.Point methods remain available.
type Point = geom.Point

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

	return geom.NewPointFlat(
		geom.XY,
		[]float64{
			longitude,
			latitude,
		},
	).SetSRID(SRID), nil
}

func MustPoint(longitude, latitude float64) *Point {
	p, err := NewPoint(longitude, latitude)
	if err != nil {
		panic(err)
	}

	return p
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
