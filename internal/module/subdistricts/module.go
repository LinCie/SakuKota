package subdistricts

import (
	"github.com/jmoiron/sqlx"
)

type Module struct {
	Service Service
	Handler *Handler
}

// The handler uses media.PublicURL to build image links.
func NewModule(db *sqlx.DB, media MediaStorage) *Module {
	repo := NewPostgresRepository(db)
	service := NewService(repo, media)
	handler := NewHandler(service, media.PublicURL)

	return &Module{
		Service: service,
		Handler: handler,
	}
}
