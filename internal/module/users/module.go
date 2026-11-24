// Package users stores user accounts, serves the logged-in user's profile,
// and exposes users to other modules.
package users

import "github.com/jmoiron/sqlx"

// Module bundles the service and handler of the users feature.
type Module struct {
	Service Service
	Handler *Handler
}

// New builds the users module on db, with profile photos kept in media.
func New(db *sqlx.DB, media MediaStorage) *Module {
	svc := NewService(NewPostgresRepository(db), media)

	return &Module{
		Service: svc,
		Handler: NewHandler(svc, media.PublicURL),
	}
}
