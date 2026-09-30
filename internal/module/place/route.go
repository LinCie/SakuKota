package place

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

	mux.HandleFunc("GET /{$}", handler.GetPlaces)
	mux.Handle("POST /{$}", adminOnly.ThenFunc(handler.CreatePlace))
	mux.HandleFunc("GET /categories/{$}", handler.GetCategories)
	mux.Handle("POST /categories/{$}", adminOnly.ThenFunc(handler.CreateCategory))
	mux.Handle("POST /categories/{id}/fields/{$}", adminOnly.ThenFunc(handler.CreateCategoryField))
}
