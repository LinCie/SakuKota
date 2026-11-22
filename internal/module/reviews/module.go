// Package reviews lets users rate and comment on places and lists a place's
// reviews.
package reviews

import "github.com/jmoiron/sqlx"

// Module bundles the service and handler of the reviews feature.
type Module struct {
	Service *Service
	Handler *Handler
}

// New builds the reviews module on db.
func New(db *sqlx.DB) *Module {
	service := NewService(NewPostgresRepository(db))

	return &Module{
		Service: service,
		Handler: NewHandler(service),
	}
}
