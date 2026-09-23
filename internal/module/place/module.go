package place

import (
	"net/http"

	"github.com/jmoiron/sqlx"
)

// Module wires up the place module's repository, service, and handler,
// and exposes a mux serving its routes.
type Module struct {
	Service Service
	Mux     *http.ServeMux
}

func NewModule(db *sqlx.DB) *Module {
	repo := NewPostgresRepository(db)
	service := NewService(repo)
	handler := NewHandler(service)

	mux := http.NewServeMux()
	RegisterRoutes(mux, handler)

	return &Module{
		Service: service,
		Mux:     mux,
	}
}
