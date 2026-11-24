// Package reports lets users report abusive places and reviews, and lets
// admins remove the reported content or dismiss the report.
package reports

import (
	"github.com/jmoiron/sqlx"

	"sakukota-go/internal/database"
)

// Module bundles the report service and its HTTP handler.
type Module struct {
	Service Service
	Handler *Handler
}

// New wires the module over db. Reported content is read and removed through
// places and reviews, removal in the same transaction as resolving the report.
// mediaURL turns the content's media keys into public URLs.
func New(db *sqlx.DB, places PlaceService, reviews ReviewService, mediaURL func(key string) string) *Module {
	svc := NewService(NewPostgresRepository(db), database.NewTransactor(db), places, reviews)

	return &Module{
		Service: svc,
		Handler: NewHandler(svc, mediaURL),
	}
}
