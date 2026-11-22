// Package reviews lets users rate and comment on places and lists a place's
// reviews.
package reviews

import (
	"github.com/jmoiron/sqlx"

	"rantau-go/internal/database"
)

// Module bundles the service and handler of the reviews feature.
type Module struct {
	Service *Service
	Handler *Handler
}

// New builds the reviews module on db. Ratings are added to place totals
// through places, in the same transaction as the review.
func New(db *sqlx.DB, places PlaceService) *Module {
	service := NewService(NewPostgresRepository(db), database.NewTransactor(db), places)

	return &Module{
		Service: service,
		Handler: NewHandler(service),
	}
}
