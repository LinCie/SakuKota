package postgis

import (
	"database/sql/driver"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/ewkb"
)

// PointCodec teaches pgx how to encode/decode PostGIS geometry values
// using *Point.
//
// It deliberately supports binary format only.
type PointCodec struct{}

var _ pgtype.Codec = PointCodec{}

func (PointCodec) FormatSupported(format int16) bool {
	return format == pgtype.BinaryFormatCode
}

func (PointCodec) PreferredFormat() int16 {
	return pgtype.BinaryFormatCode
}

func (PointCodec) PlanEncode(
	_ *pgtype.Map,
	_ uint32,
	format int16,
	value any,
) pgtype.EncodePlan {
	if format != pgtype.BinaryFormatCode {
		return nil
	}

	switch value.(type) {
	case *Point, Point:
		return encodePlanPoint{}
	default:
		return nil
	}
}

func (PointCodec) PlanScan(
	_ *pgtype.Map,
	_ uint32,
	format int16,
	target any,
) pgtype.ScanPlan {
	if format != pgtype.BinaryFormatCode {
		return nil
	}

	switch target.(type) {
	case *Point:
		return scanPlanPoint{}
	default:
		return nil
	}
}

// DecodeDatabaseSQLValue is required by pgtype.Codec.
//
// pgx's database/sql adapter uses this method for values returned through
// sqlx. Returning the raw EWKB as []byte is a valid database/sql driver.Value.
func (PointCodec) DecodeDatabaseSQLValue(
	_ *pgtype.Map,
	_ uint32,
	format int16,
	src []byte,
) (driver.Value, error) {
	if src == nil {
		return nil, nil
	}

	if format != pgtype.BinaryFormatCode {
		return nil, errors.ErrUnsupported
	}

	// pgx owns src, so don't return it directly.
	dst := make([]byte, len(src))
	copy(dst, src)

	return dst, nil
}

func (PointCodec) DecodeValue(
	_ *pgtype.Map,
	_ uint32,
	format int16,
	src []byte,
) (any, error) {
	if src == nil {
		return nil, nil
	}

	if format != pgtype.BinaryFormatCode {
		return nil, errors.ErrUnsupported
	}

	return decodePoint(src)
}

// -----------------------------------------------------------------------------
// Encode
// -----------------------------------------------------------------------------

type encodePlanPoint struct{}

var _ pgtype.EncodePlan = encodePlanPoint{}

func (encodePlanPoint) Encode(
	value any,
	buf []byte,
) ([]byte, error) {
	var point *Point

	switch v := value.(type) {
	case *Point:
		if v == nil {
			// nil means SQL NULL.
			return nil, nil
		}

		point = v

	case Point:
		// Support a non-pointer Point as well.
		point = &v

	default:
		return nil, fmt.Errorf(
			"postgis: cannot encode %T as Point",
			value,
		)
	}

	if err := validatePoint(point); err != nil {
		return nil, fmt.Errorf(
			"postgis: invalid point: %w",
			err,
		)
	}

	data, err := ewkb.Marshal(
		&point.Point,
		binary.LittleEndian,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"postgis: encode point as EWKB: %w",
			err,
		)
	}

	return append(buf, data...), nil
}

// -----------------------------------------------------------------------------
// Scan
// -----------------------------------------------------------------------------

type scanPlanPoint struct{}

var _ pgtype.ScanPlan = scanPlanPoint{}

func (scanPlanPoint) Scan(
	src []byte,
	target any,
) error {
	point, ok := target.(*Point)
	if !ok {
		return fmt.Errorf(
			"postgis: cannot scan Point into %T",
			target,
		)
	}

	if src == nil {
		return fmt.Errorf(
			"postgis: cannot scan NULL into Point",
		)
	}

	decoded, err := decodePoint(src)
	if err != nil {
		return err
	}

	*point = *decoded

	return nil
}

func decodePoint(src []byte) (*Point, error) {
	// database/sql receives PostGIS geometry as hexadecimal EWKB text because
	// custom types are not included in pgx's binary result format list.
	// Accept both that representation and native binary EWKB.
	if encoded := src; len(encoded)%2 == 0 {
		if len(encoded) >= 2 && encoded[0] == '\\' && encoded[1] == 'x' {
			encoded = encoded[2:]
		}

		decoded := make([]byte, hex.DecodedLen(len(encoded)))
		if _, err := hex.Decode(decoded, encoded); err == nil {
			src = decoded
		}
	}

	g, err := ewkb.Unmarshal(src)
	if err != nil {
		return nil, fmt.Errorf(
			"postgis: decode EWKB: %w",
			err,
		)
	}

	point, ok := g.(*geom.Point)
	if !ok {
		return nil, fmt.Errorf(
			"postgis: expected Point, got %T",
			g,
		)
	}

	decoded := &Point{Point: *point}

	if err := validatePoint(decoded); err != nil {
		return nil, fmt.Errorf(
			"postgis: invalid point from database: %w",
			err,
		)
	}

	return decoded, nil
}
