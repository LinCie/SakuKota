package subdistricts

import (
	"github.com/jmoiron/sqlx"
)

// Module wires up the subdistrict module's repository, service, and handler.
type Module struct {
	Service Service
	Handler *Handler
}

// NewModule builds the sub-district module's repository, service, and handler, linked together.
// The handler uses images.PublicURL to build image links.
func NewModule(db *sqlx.DB, images ImageStorage) *Module {
	repo := NewPostgresRepository(db)
	service := NewService(repo, images)
	handler := NewHandler(service, images.PublicURL)

	return &Module{
		Service: service,
		Handler: handler,
	}
}
