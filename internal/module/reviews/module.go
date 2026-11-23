// Package reviews lets users rate and comment on places and lists a place's
// reviews.
package reviews

import (
	"github.com/jmoiron/sqlx"

	"rantau-go/internal/database"
)

// Module bundles the service and handler of the reviews feature.
type Module struct {
	Service Service
	Handler *Handler
}

// New builds the reviews module on db, with review photos kept in media.
// Ratings are added to place totals through places, in the same transaction
// as the review.
func New(db *sqlx.DB, places PlaceService, media MediaStorage) *Module {
	svc := NewService(NewPostgresRepository(db), database.NewTransactor(db), places, media)

	return &Module{
		Service: svc,
		Handler: NewHandler(svc, media.PublicURL),
	}
}
