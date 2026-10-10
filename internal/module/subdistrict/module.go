package subdistrict

import (
	"github.com/jmoiron/sqlx"

	"rantau-go/internal/config"
)

// Module wires up the subdistrict module's repository, service, and
// handler, and exposes a mux serving its routes.
type Module struct {
	Service Service
	Handler *Handler
}

func NewModule(db *sqlx.DB, cfg config.Config) *Module {
	repo := NewPostgresRepository(db)
	service := NewService(repo)
	handler := NewHandler(service)

	return &Module{
		Service: service,
		Handler: handler,
	}
}
