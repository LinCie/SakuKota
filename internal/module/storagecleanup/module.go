package storagecleanup

import (
	"github.com/jmoiron/sqlx"
)

// Module wires up the storage cleanup module's repository and service.
type Module struct {
	Service Service
}

func NewModule(db *sqlx.DB, images ImageStorage) *Module {
	repo := NewPostgresRepository(db)
	service := NewService(repo, images)

	return &Module{
		Service: service,
	}
}
