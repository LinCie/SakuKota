package suggestions

import (
	"github.com/jmoiron/sqlx"
)

type Module struct {
	Service Service
	Handler *Handler
}

// NewModule takes the place service is used to create and delete places when a suggestion is approved.
func NewModule(db *sqlx.DB, places PlaceService) *Module {
	repo := NewPostgresRepository(db)
	service := NewService(repo, places)
	handler := NewHandler(service)

	return &Module{
		Service: service,
		Handler: handler,
	}
}
