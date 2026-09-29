package auth

import (
	"net/http"

	"github.com/jmoiron/sqlx"

	"rantau-go/internal/config"
)

// Module wires up the auth module's repository, service, and handler, and
// exposes a mux serving its routes.
type Module struct {
	Service Service
	Mux     *http.ServeMux
}

func NewModule(db *sqlx.DB, cfg config.Config) *Module {
	repo := NewPostgresRepository(db)
	hasher := NewArgon2Hasher()
	store := NewMemoryRefreshStore()
	service := NewService(cfg, repo, hasher, store)
	handler := NewHandler(service)

	mux := http.NewServeMux()
	RegisterRoutes(mux, handler)

	return &Module{
		Service: service,
		Mux:     mux,
	}
}
