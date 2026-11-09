package auth

import (
	"github.com/redis/go-redis/v9"

	"rantau-go/internal/config"
	"rantau-go/internal/module/users"
)

// Module wires up the auth module's service and handler.
type Module struct {
	Service Service
	Handler *Handler
}

func NewModule(cfg config.Config, userService users.Service, cache *redis.Client) *Module {
	store := NewRedisRefreshStore(cache)
	service := NewService(cfg, userService, store)
	handler := NewHandler(cfg, service)

	return &Module{
		Service: service,
		Handler: handler,
	}
}
