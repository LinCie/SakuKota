package places

import (
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

// Module wires up the place module's repository, service, and handler.
type Module struct {
	Service Service
	Handler *Handler
}

// NewModule builds the place module's repository, service, and handler, linked together.
// Pages of places are cached in Redis, and the handler uses images.PublicURL to build photo links.
func NewModule(db *sqlx.DB, rdb *redis.Client, images ImageStorage) *Module {
	repo := NewCachedRepository(NewPostgresRepository(db), rdb)
	// repo := NewPostgresRepository(db)
	service := NewService(repo, images)
	handler := NewHandler(service, images.PublicURL)

	return &Module{
		Service: service,
		Handler: handler,
	}
}
