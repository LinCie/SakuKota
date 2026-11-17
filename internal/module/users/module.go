package users

import (
	"github.com/jmoiron/sqlx"
)

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
