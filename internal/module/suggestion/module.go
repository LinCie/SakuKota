package suggestion

import (
	"github.com/jmoiron/sqlx"
)

// Module wires up the suggestion module's repository, service, and handler.
type Module struct {
	Service Service
	Handler *Handler
}

// NewModule builds the suggestion module's repository, service, and handler, linked together.
// The place service is used to create and delete places when a suggestion is approved.
func NewModule(db *sqlx.DB, places PlaceService) *Module {
	repo := NewPostgresRepository(db)
	service := NewService(repo, places)
	handler := NewHandler(service)

	return &Module{
		Service: service,
		Handler: handler,
	}
}
