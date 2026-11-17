package storagecleanup

import (
	"github.com/jmoiron/sqlx"
)

// Module wires up the storage cleanup module's repository and service.
type Module struct {
	Service Service
}

// NewModule builds the storage cleanup module's repository and service, linked together.
func NewModule(db *sqlx.DB, media MediaStorage) *Module {
	repo := NewPostgresRepository(db)
	service := NewService(repo, media)

	return &Module{
		Service: service,
	}
}
