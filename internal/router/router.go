package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"

	"rantau-go/internal/config"
	"rantau-go/internal/docs"
	"rantau-go/internal/middleware"
	"rantau-go/internal/module/auth"
	"rantau-go/internal/module/place"
	"rantau-go/internal/module/subdistrict"
)

type Router struct {
	mux *http.ServeMux
	cfg config.Config
}

func NewRouter(db *sqlx.DB, cfg config.Config) http.Handler {
	r := chi.NewRouter()

	r.Use(
		chimiddleware.Recoverer,
		chimiddleware.Logger,
		chimiddleware.StripSlashes,
		middleware.WithCORS(cfg),
	)

	r.Get("/docs", docs.Scalar)
	r.Get("/openapi.json", docs.OpenAPI)

	authModule := auth.NewModule(db, cfg)
	placeModule := place.NewModule(db, cfg)
	subdisctrictModule := subdistrict.NewModule(db, cfg)

	r.Route("/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authModule.Handler.Register)
			r.Post("/login", authModule.Handler.Login)
			r.Post("/refresh", authModule.Handler.Refresh)
			r.Post("/logout", authModule.Handler.Logout)
		})

		r.Route("/places", func(r chi.Router) {
			// GET /v1/places
			r.Get("/", placeModule.Handler.GetPlaces)

			r.Group(func(r chi.Router) {
				r.Use(middleware.WithAuth(cfg))
				r.Use(middleware.Guard([]string{"admin"}))

				// POST /v1/places
				r.Post("/", placeModule.Handler.CreatePlace)

				// PATCH /v1/places/{id}
				r.Patch("/{id}", placeModule.Handler.PatchPlace)

				// DELETE /v1/places/{id}
				r.Delete("/{id}", placeModule.Handler.DeletePlace)
			})

			r.Route("/categories", func(r chi.Router) {
				// GET /v1/places/categories
				r.Get("/", placeModule.Handler.GetCategories)

				r.Group(func(r chi.Router) {
					r.Use(middleware.WithAuth(cfg))
					r.Use(middleware.Guard([]string{"admin"}))

					// POST /v1/places/categories
					r.Post("/", placeModule.Handler.CreateCategory)

					// POST /v1/places/categories/{id}/fields
					r.Post("/{id}/fields", placeModule.Handler.CreateCategoryField)
				})
			})
		})

		r.Group(func(r chi.Router) {
			r.Route("/subdistricts", func(r chi.Router) {
				// GET /v1/subdistricts
				r.Get("/", subdisctrictModule.Handler.GetSubDistricts)

				// GET /v1/subdistricts/{id}
				r.Get("/{id}", subdisctrictModule.Handler.GetSubDistrict)

				r.Group(func(r chi.Router) {
					r.Use(middleware.WithAuth(cfg))
					r.Use(middleware.Guard([]string{"admin"}))

					// POST /v1/subdistricts
					r.Post("/", subdisctrictModule.Handler.CreateSubDistrict)

					// PATCH /v1/subdistricts/{id}
					r.Patch("/{id}", subdisctrictModule.Handler.PatchSubDistrict)

					// DELETE /v1/subdistricts/{id}
					r.Delete("/{id}", subdisctrictModule.Handler.DeleteSubDistrict)
				})
			})
		})
	})

	return r
}
