package postgis

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Register registers the PostGIS geometry type with a pgx connection.
//
// PostGIS extension OIDs are installation-specific, so we ask PostgreSQL
// for the geometry OID instead of hard-coding it.
func Register(
	ctx context.Context,
	conn *pgx.Conn,
) error {
	var geometryOID uint32

	err := conn.QueryRow(
		ctx,
		`SELECT 'geometry'::regtype::oid`,
	).Scan(&geometryOID)
	if err != nil {
		return fmt.Errorf(
			"postgis: get geometry OID: %w",
			err,
		)
	}

	typeMap := conn.TypeMap()

	typeMap.RegisterType(&pgtype.Type{
		Name:  "geometry",
		OID:   geometryOID,
		Codec: PointCodec{},
	})

	// These mappings matter when pgx needs to determine the PostgreSQL
	// type from the Go value rather than from SQL context.
	typeMap.RegisterDefaultPgType(
		Point{},
		"geometry",
	)

	typeMap.RegisterDefaultPgType(
		(*Point)(nil),
		"geometry",
	)

	return nil
}
