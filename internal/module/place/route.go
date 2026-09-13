package place

import "net/http"

func RegisterRoutes(mux *http.ServeMux, handler *Handler) {
	mux.HandleFunc("GET /{$}", handler.GetMany)
	mux.HandleFunc("GET /categories/", handler.GetCategories)
}
