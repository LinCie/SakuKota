package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"

	"rantau-go/internal/config"
	"rantau-go/internal/docs"
	"rantau-go/internal/domain"
	"rantau-go/internal/middleware"
	"rantau-go/internal/module/auth"
	"rantau-go/internal/module/places"
	"rantau-go/internal/module/reviews"
	"rantau-go/internal/module/subdistricts"
	"rantau-go/internal/module/suggestions"
	"rantau-go/internal/module/users"
	"rantau-go/internal/storage"
)

// NewRouter builds the HTTP router with its middleware, docs routes, and every API route.
// Routes live under /v1, and protected ones check the token and role first.
func NewRouter(db *sqlx.DB, rdb *redis.Client, cfg config.Config) http.Handler {
	r := chi.NewRouter()

	// Logger comes before Recoverer so a recovered panic is logged as a 500.
	// The web app reaches the API through a same-origin proxy, so no CORS
	// headers are needed.
	r.Use(
		chimiddleware.Logger,
		chimiddleware.Recoverer,
		chimiddleware.StripSlashes,
		middleware.WithMaxBody(),
	)

	r.Get("/docs", docs.Scalar)
	r.Get("/openapi.json", docs.OpenAPI)

	media := storage.NewMediaStore(storage.NewS3(cfg.Storage), cfg.Storage.PublicURL)

	userModule := users.New(db)
	authModule := auth.New(cfg, userModule.Service, rdb)
	placeModule := places.New(db, rdb, media)
	reviewModule := reviews.New(db)
	subdistrictModule := subdistricts.New(db, media)
	suggestionModule := suggestions.New(db, placeModule.Service)

	r.Route("/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authModule.Handler.Register)
			r.Post("/login", authModule.Handler.Login)
			r.Post("/refresh", authModule.Handler.Refresh)
			r.Post("/logout", authModule.Handler.Logout)
		})

		r.Route("/places", func(r chi.Router) {
			// GET /v1/places
			r.Get("/", placeModule.Handler.List)

			// GET /v1/places/{id}
			r.Get("/{id}", placeModule.Handler.Show)

			r.Group(func(r chi.Router) {
				r.Use(middleware.WithAuth(cfg))
				r.Use(middleware.Guard([]string{domain.RoleAdmin}))

				// POST /v1/places
				r.Post("/", placeModule.Handler.Create)

				// PUT /v1/places/{id}
				r.Put("/{id}", placeModule.Handler.Replace)

				// DELETE /v1/places/{id}
				r.Delete("/{id}", placeModule.Handler.Delete)

				// POST /v1/places/{id}/medias/upload-url
				r.Post("/{id}/medias/upload-url", placeModule.Handler.CreateMediaUpload)

				// POST /v1/places/{id}/medias
				r.Post("/{id}/medias", placeModule.Handler.AddMedia)

				// DELETE /v1/places/{id}/medias/{mediaId}
				r.Delete("/{id}/medias/{mediaId}", placeModule.Handler.DeleteMedia)
			})

			r.Route("/{id}/reviews", func(r chi.Router) {
				// GET /v1/places/{id}/reviews
				r.Get("/", reviewModule.Handler.List)

				r.Group(func(r chi.Router) {
					r.Use(middleware.WithAuth(cfg))

					// POST /v1/places/{id}/reviews
					r.Post("/", reviewModule.Handler.Create)
				})
			})

			r.Route("/categories", func(r chi.Router) {
				// GET /v1/places/categories
				r.Get("/", placeModule.Handler.ListCategories)

				// GET /v1/places/categories/{id}
				r.Get("/{id}", placeModule.Handler.ShowCategory)
			})
		})

		r.Route("/suggestions", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(middleware.WithAuth(cfg))

				// POST /v1/suggestions
				r.Post("/", suggestionModule.Handler.Create)

				// GET /v1/suggestions/mine
				r.Get("/mine", suggestionModule.Handler.ListMine)
			})

			r.Group(func(r chi.Router) {
				r.Use(middleware.WithAuth(cfg))
				r.Use(middleware.Guard([]string{domain.RoleAdmin}))

				// GET /v1/suggestions
				r.Get("/", suggestionModule.Handler.List)

				// GET /v1/suggestions/{id}
				r.Get("/{id}", suggestionModule.Handler.Show)

				// POST /v1/suggestions/{id}/approve
				r.Post("/{id}/approve", suggestionModule.Handler.Approve)

				// POST /v1/suggestions/{id}/reject
				r.Post("/{id}/reject", suggestionModule.Handler.Reject)
			})
		})

		r.Route("/subdistricts", func(r chi.Router) {
			// GET /v1/subdistricts
			r.Get("/", subdistrictModule.Handler.List)

			// GET /v1/subdistricts/{id}
			r.Get("/{id}", subdistrictModule.Handler.Show)

			r.Group(func(r chi.Router) {
				r.Use(middleware.WithAuth(cfg))
				r.Use(middleware.Guard([]string{domain.RoleAdmin}))

				// POST /v1/subdistricts
				r.Post("/", subdistrictModule.Handler.Create)

				// PUT /v1/subdistricts/{id}
				r.Put("/{id}", subdistrictModule.Handler.Replace)

				// DELETE /v1/subdistricts/{id}
				r.Delete("/{id}", subdistrictModule.Handler.Delete)

				// POST /v1/subdistricts/{id}/medias/upload-url
				r.Post("/{id}/medias/upload-url", subdistrictModule.Handler.CreateMediaUpload)

				// POST /v1/subdistricts/{id}/medias
				r.Post("/{id}/medias", subdistrictModule.Handler.AddMedia)

				// DELETE /v1/subdistricts/{id}/medias/{mediaId}
				r.Delete("/{id}/medias/{mediaId}", subdistrictModule.Handler.DeleteMedia)

				// PUT /v1/subdistricts/{id}/boundary
				r.Put("/{id}/boundary", subdistrictModule.Handler.SetBoundary)
			})
		})
	})

	return r
}
