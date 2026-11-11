package subdistrict

import (
	"github.com/jmoiron/sqlx"
)

// Module wires up the subdistrict module's repository, service, and handler.
type Module struct {
	Service Service
	Handler *Handler
}

func NewModule(db *sqlx.DB, images ImageStorage) *Module {
	repo := NewPostgresRepository(db)
	service := NewService(repo, images)
	handler := NewHandler(service, images.PublicURL)

	return &Module{
		Service: service,
		Handler: handler,
	}
}
