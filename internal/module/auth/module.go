package auth

import (
	"rantau-go/internal/config"
)

// Module wires up the auth module's service and handler.
type Module struct {
	Service Service
	Handler *Handler
}

func NewModule(cfg config.Config, users UserService) *Module {
	store := NewMemoryRefreshStore()
	service := NewService(cfg, users, store)
	handler := NewHandler(cfg, service)

	return &Module{
		Service: service,
		Handler: handler,
	}
}
