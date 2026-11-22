// Package places manages places and their categories: listing, CRUD, photos,
// and the Redis-cached place index behind the map.
package places

import (
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

// Module is the places service and its HTTP handler.
type Module struct {
	Service *Service
	Handler *Handler
}

// New wires places on db, with Redis caching of the place index and photos kept in media.
func New(db *sqlx.DB, rdb *redis.Client, media MediaStorage) *Module {
	repo := NewCachedRepository(NewPostgresRepository(db), rdb)
	service := NewService(repo, media)
	handler := NewHandler(service, media.PublicURL)

	return &Module{
		Service: service,
		Handler: handler,
	}
}
