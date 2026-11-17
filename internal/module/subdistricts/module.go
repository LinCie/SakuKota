package subdistricts

import (
	"github.com/jmoiron/sqlx"
)

type Module struct {
	Service Service
	Handler *Handler
}

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
