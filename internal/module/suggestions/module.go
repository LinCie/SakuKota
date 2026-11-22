// Package suggestions lets users suggest places and lets admins approve or reject them.
package suggestions

import (
	"github.com/jmoiron/sqlx"

	"rantau-go/internal/database"
)

// Module bundles the suggestion service and its HTTP handler.
type Module struct {
	Service *Service
	Handler *Handler
}

// New wires the module over db. The place service creates the place when a
// suggestion is approved, in the same transaction as the approval.
func New(db *sqlx.DB, places PlaceService) *Module {
	repo := NewPostgresRepository(db)
	service := NewService(repo, database.NewTransactor(db), places)
	handler := NewHandler(service)

	return &Module{
		Service: service,
		Handler: handler,
	}
}
