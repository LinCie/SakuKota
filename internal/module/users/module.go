package users

import (
	"github.com/jmoiron/sqlx"
)

// Module wires up the user module's repository and service.
type Module struct {
	Service Service
}

// NewModule builds the user module's repository and service, linked together.
func NewModule(db *sqlx.DB) *Module {
	repo := NewPostgresRepository(db)
	service := NewService(repo)

	return &Module{
		Service: service,
	}
}
