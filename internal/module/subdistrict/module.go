package subdistrict

import (
	"github.com/jmoiron/sqlx"
)

// Module wires up the subdistrict module's repository, service, and
// handler, and exposes a mux serving its routes.
type Module struct {
	Service Service
	Handler *Handler
}

func NewModule(db *sqlx.DB) *Module {
	repo := NewPostgresRepository(db)
	service := NewService(repo)
	handler := NewHandler(service)

	return &Module{
		Service: service,
		Handler: handler,
	}
}
