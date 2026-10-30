package users

import (
	"github.com/jmoiron/sqlx"
)

// Module wires up the users module's repository and service.
type Module struct {
	Service Service
}

func NewModule(db *sqlx.DB) *Module {
	repo := NewPostgresRepository(db)
	service := NewService(repo)

	return &Module{
		Service: service,
	}
}
