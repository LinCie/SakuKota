package router

import "net/http"

type Router struct {
	mux *http.ServeMux
}

func NewRouter() Router {
	mux := http.NewServeMux()
	return Router{
		mux: mux,
	}
}
