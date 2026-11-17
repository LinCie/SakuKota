package auth

import (
	"github.com/redis/go-redis/v9"

	"rantau-go/internal/config"
	"rantau-go/internal/module/users"
)

type Module struct {
	Service Service
	Handler *Handler
}

func NewModule(cfg config.Config, userService users.Service, rdb *redis.Client) *Module {
	store := NewRedisRefreshStore(rdb)
	service := NewService(cfg, userService, store)
	handler := NewHandler(cfg, service)

	return &Module{
		Service: service,
		Handler: handler,
	}
}
