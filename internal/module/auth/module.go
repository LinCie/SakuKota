// Package auth registers users and signs them in with an access token and a
// rotating refresh token.
package auth

import (
	"github.com/redis/go-redis/v9"

	"sakukota-go/internal/config"
)

// Module bundles the service and handler of the auth feature.
type Module struct {
	Service Service
	Handler *Handler
}

// New builds the auth module. Refresh tokens are kept in rdb.
func New(cfg config.Config, userService UserService, rdb *redis.Client) *Module {
	svc := NewService(cfg, userService, NewRedisRefreshStore(rdb))

	return &Module{
		Service: svc,
		Handler: NewHandler(cfg, svc),
	}
}
