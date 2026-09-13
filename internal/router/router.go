package router

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"rantau-go/internal/httpx"
	"rantau-go/internal/middleware"
	"rantau-go/internal/module/place"
	"rantau-go/internal/module/subdistrict"
)

type Router struct {
	mux *http.ServeMux
}

func NewRouter(pool *pgxpool.Pool) Router {
	mux := http.NewServeMux()

	placeModule := place.NewModule(pool)
	mux.Handle("/places/", http.StripPrefix("/places", placeModule.Mux))

	subdistrictModule := subdistrict.NewModule(pool)
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
