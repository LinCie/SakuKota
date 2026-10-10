package auth

import (
	"github.com/jmoiron/sqlx"

	"rantau-go/internal/config"
)

// Module wires up the auth module's repository, service, and handler.
type Module struct {
	Service Service
	Handler *Handler
}

func NewModule(db *sqlx.DB, cfg config.Config) *Module {
	repo := NewPostgresRepository(db)
	hasher := NewArgon2Hasher()
	store := NewMemoryRefreshStore()
	service := NewService(cfg, repo, hasher, store)
	handler := NewHandler(cfg, service)

	return &Module{
		Service: service,
		Handler: handler,
	}
}
