package review

import (
	"github.com/jmoiron/sqlx"
)

// Module wires up the review module's repository, service, and handler.
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
