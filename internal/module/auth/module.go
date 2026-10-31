package auth

import (
	"rantau-go/internal/config"
	"rantau-go/internal/module/users"
)

// Module wires up the auth module's service and handler.
type Module struct {
	Service Service
	Handler *Handler
}

func NewModule(cfg config.Config, userService users.Service) *Module {
	store := NewMemoryRefreshStore()
	service := NewService(cfg, userService, store)
	handler := NewHandler(cfg, service)

	return &Module{
		Service: service,
		Handler: handler,
	}
}
