package subdistrict

import (
	"net/http"

	"rantau-go/internal/config"
	"rantau-go/internal/httpx"
	"rantau-go/internal/middleware"
)

func RegisterRoutes(mux *http.ServeMux, handler *Handler, cfg config.Config) {
	adminOnly := httpx.NewChain(
		middleware.WithAuth(cfg),
		middleware.Guard([]string{"admin"}),
	)

	mux.HandleFunc("GET /{$}", handler.GetSubDistricts)
	mux.Handle("POST /{$}", adminOnly.ThenFunc(handler.CreateSubDistrict))
}
