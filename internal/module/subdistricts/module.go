// Package subdistricts serves the sub-districts (kecamatan) API: their details,
// photos, and GeoJSON boundaries.
package subdistricts

import "github.com/jmoiron/sqlx"

// Module wires the service and handler of sub-districts.
type Module struct {
	Service *Service
	Handler *Handler
}

// New builds a Module over db and media. The handler uses media.PublicURL to
// build image links.
func New(db *sqlx.DB, media MediaStorage) *Module {
	service := NewService(NewPostgresRepository(db), media)

	return &Module{
		Service: service,
		Handler: NewHandler(service, media.PublicURL),
	}
}
