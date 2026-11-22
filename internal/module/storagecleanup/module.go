// Package storagecleanup deletes storage objects whose database rows are gone,
// using a queue of keys kept in PostgreSQL.
package storagecleanup

import "github.com/jmoiron/sqlx"

// Module wires up the storage cleanup module's repository and service.
type Module struct {
	Service *Service
}

// New builds the storage cleanup module's repository and service, linked together.
func New(db *sqlx.DB, media MediaStorage) *Module {
	return &Module{Service: NewService(NewPostgresRepository(db), media)}
}
