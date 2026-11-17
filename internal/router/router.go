package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"

	"rantau-go/internal/config"
	"rantau-go/internal/docs"
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

	userModule := users.NewModule(db)
	authModule := auth.NewModule(cfg, userModule.Service, rdb)
	placeModule := places.NewModule(db, rdb, media)
	reviewModule := reviews.NewModule(db)
	subdistrictModule := subdistricts.NewModule(db, media)
	suggestionModule := suggestions.NewModule(db, placeModule.Service)

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

			// GET /v1/places/{id}
			r.Get("/{id}", placeModule.Handler.GetPlace)

			r.Group(func(r chi.Router) {
				r.Use(middleware.WithAuth(cfg))
				r.Use(middleware.Guard([]string{users.RoleAdmin}))

				// POST /v1/places
				r.Post("/", placeModule.Handler.CreatePlace)

				// PUT /v1/places/{id}
				r.Put("/{id}", placeModule.Handler.PutPlace)

				// DELETE /v1/places/{id}
				r.Delete("/{id}", placeModule.Handler.DeletePlace)

				// POST /v1/places/{id}/medias/upload-url
				r.Post("/{id}/medias/upload-url", placeModule.Handler.CreateMediaUpload)

				// POST /v1/places/{id}/medias
				r.Post("/{id}/medias", placeModule.Handler.AddMedia)

				// DELETE /v1/places/{id}/medias/{mediaId}
				r.Delete("/{id}/medias/{mediaId}", placeModule.Handler.DeleteMedia)
			})

			r.Route("/{id}/reviews", func(r chi.Router) {
				// GET /v1/places/{id}/reviews
				r.Get("/", reviewModule.Handler.GetReviews)

				r.Group(func(r chi.Router) {
					r.Use(middleware.WithAuth(cfg))

					// POST /v1/places/{id}/reviews
					r.Post("/", reviewModule.Handler.CreateReview)
				})
			})

			r.Route("/categories", func(r chi.Router) {
				// GET /v1/places/categories
				r.Get("/", placeModule.Handler.GetCategories)

				// GET /v1/places/categories/{id}
				r.Get("/{id}", placeModule.Handler.GetCategory)
			})
		})

		r.Route("/suggestions", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(middleware.WithAuth(cfg))

				// POST /v1/suggestions
				r.Post("/", suggestionModule.Handler.CreateSuggestion)

				// GET /v1/suggestions/mine
				r.Get("/mine", suggestionModule.Handler.GetMySuggestions)
			})

			r.Group(func(r chi.Router) {
				r.Use(middleware.WithAuth(cfg))
				r.Use(middleware.Guard([]string{users.RoleAdmin}))

				// GET /v1/suggestions
				r.Get("/", suggestionModule.Handler.GetSuggestions)

				// GET /v1/suggestions/{id}
				r.Get("/{id}", suggestionModule.Handler.GetSuggestion)

				// POST /v1/suggestions/{id}/approve
				r.Post("/{id}/approve", suggestionModule.Handler.ApproveSuggestion)

				// POST /v1/suggestions/{id}/reject
				r.Post("/{id}/reject", suggestionModule.Handler.RejectSuggestion)
			})
		})

		r.Route("/subdistricts", func(r chi.Router) {
			// GET /v1/subdistricts
			r.Get("/", subdistrictModule.Handler.GetSubDistricts)

			// GET /v1/subdistricts/{id}
			r.Get("/{id}", subdistrictModule.Handler.GetSubDistrict)

			r.Group(func(r chi.Router) {
				r.Use(middleware.WithAuth(cfg))
				r.Use(middleware.Guard([]string{users.RoleAdmin}))

				// POST /v1/subdistricts
				r.Post("/", subdistrictModule.Handler.CreateSubDistrict)

				// PUT /v1/subdistricts/{id}
				r.Put("/{id}", subdistrictModule.Handler.PutSubDistrict)

				// DELETE /v1/subdistricts/{id}
				r.Delete("/{id}", subdistrictModule.Handler.DeleteSubDistrict)

				// POST /v1/subdistricts/{id}/medias/upload-url
				r.Post("/{id}/medias/upload-url", subdistrictModule.Handler.CreateMediaUpload)

				// POST /v1/subdistricts/{id}/medias
				r.Post("/{id}/medias", subdistrictModule.Handler.AddMedia)

				// DELETE /v1/subdistricts/{id}/medias/{mediaId}
				r.Delete("/{id}/medias/{mediaId}", subdistrictModule.Handler.DeleteMedia)

				// POST /v1/subdistricts/{id}/static-map/upload-url
				r.Post("/{id}/static-map/upload-url", subdistrictModule.Handler.CreateStaticMapUpload)

				// PUT /v1/subdistricts/{id}/static-map
				r.Put("/{id}/static-map", subdistrictModule.Handler.SetStaticMap)
			})
		})
	})

	return r
}
