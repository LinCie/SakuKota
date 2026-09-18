package router

import (
	"net/http"

	"github.com/jmoiron/sqlx"

	"rantau-go/internal/httpx"
	"rantau-go/internal/middleware"
	"rantau-go/internal/module/place"
	"rantau-go/internal/module/subdistrict"
)

type Router struct {
	mux *http.ServeMux
}

func NewRouter(db *sqlx.DB) Router {
	mux := http.NewServeMux()

	placeModule := place.NewModule(db)
	mux.Handle("/places/", http.StripPrefix("/places", placeModule.Mux))

	subdistrictModule := subdistrict.NewModule(db)
	mux.Handle("/subdistricts/", http.StripPrefix("/subdistricts", subdistrictModule.Mux))

	return Router{
		mux: mux,
	}
}

func (r Router) Handler() http.Handler {
	return httpx.NewChain(
		middleware.Logger,
		middleware.Recovery,
	).Then(r.mux)
}
