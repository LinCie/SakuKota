package places

import (
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

type Module struct {
	Service Service
	Handler *Handler
}

func NewModule(db *sqlx.DB, rdb *redis.Client, media MediaStorage) *Module {
	repo := NewCachedRepository(NewPostgresRepository(db), rdb)
	service := NewService(repo, media)
	handler := NewHandler(service, media.PublicURL)

	return &Module{
		Service: service,
		Handler: handler,
	}
}
