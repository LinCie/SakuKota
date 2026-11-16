package places

import (
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

type Module struct {
	Service Service
	Handler *Handler
}

func NewModule(db *sqlx.DB, rdb *redis.Client, images ImageStorage) *Module {
	repo := NewCachedRepository(NewPostgresRepository(db), rdb)
	service := NewService(repo, images)
	handler := NewHandler(service, images.PublicURL)

	return &Module{
		Service: service,
		Handler: handler,
	}
}
