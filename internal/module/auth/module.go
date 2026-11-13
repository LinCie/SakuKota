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

// NewModule builds the auth module's refresh store, service, and handler, linked together.
// Refresh tokens are kept in Redis.
func NewModule(cfg config.Config, userService users.Service, rdb *redis.Client) *Module {
	store := NewRedisRefreshStore(rdb)
	service := NewService(cfg, userService, store)
	handler := NewHandler(cfg, service)

	return &Module{
		Service: service,
		Handler: handler,
	}
}
