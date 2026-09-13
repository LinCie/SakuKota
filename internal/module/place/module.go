package place

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Module wires up the place module's repository, service, and handler,
// and exposes a mux serving its routes.
type Module struct {
	Service *Service
	Mux     *http.ServeMux
}

func NewModule(pool *pgxpool.Pool) *Module {
	repo := NewPostgresRepository(pool)
	service := NewService(repo)
	handler := NewHandler(service)

	mux := http.NewServeMux()
	RegisterRoutes(mux, handler)

	return &Module{
		Service: service,
		Mux:     mux,
	}
}
