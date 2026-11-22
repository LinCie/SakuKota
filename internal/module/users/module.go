// Package users stores user accounts and exposes them to other modules.
package users

import "github.com/jmoiron/sqlx"

// Module wires up the users module's repository and service.
type Module struct {
	Service *Service
}

// New builds the users module's repository and service, linked together.
func New(db *sqlx.DB) *Module {
	return &Module{Service: NewService(NewPostgresRepository(db))}
}
