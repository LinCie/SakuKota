// Package suggestions lets users suggest places and lets admins approve or reject them.
package suggestions

import (
	"github.com/jmoiron/sqlx"
)

// Module bundles the suggestion service and its HTTP handler.
type Module struct {
	Service *Service
	Handler *Handler
}

// New wires the module over db. The place service is used to create and delete
// places when a suggestion is approved.
func New(db *sqlx.DB, places PlaceService) *Module {
	repo := NewPostgresRepository(db)
	service := NewService(repo, places)
	handler := NewHandler(service)

	return &Module{
		Service: service,
		Handler: handler,
	}
}
