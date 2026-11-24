package router

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"

	"sakukota-go/internal/config"
	"sakukota-go/internal/docs"
	"sakukota-go/internal/domain"
	"sakukota-go/internal/middleware"
	"sakukota-go/internal/module/auth"
	"sakukota-go/internal/module/places"
	"sakukota-go/internal/module/reports"
	"sakukota-go/internal/module/reviews"
	"sakukota-go/internal/module/subdistricts"
	"sakukota-go/internal/module/suggestions"
	"sakukota-go/internal/module/users"
	"sakukota-go/internal/storage"
)

// requestTimeout bounds each request's context. It stays under the server's
// 30 s WriteTimeout (cmd/api), after which the response could not be sent
// anyway, and leaves room for promoting a 50 MB video in storage.
const requestTimeout = 25 * time.Second

// NewRouter builds the HTTP router with its middleware, docs routes, and every API route.
// Routes live under /v1, and protected ones check the token and role first.
func NewRouter(db *sqlx.DB, rdb *redis.Client, cfg config.Config) http.Handler {
	r := chi.NewRouter()

	// Telemetry and the request log come before Recoverer so a recovered panic
	// is recorded and logged as a 500. Telemetry comes first so each log line
	// carries its request's trace.
	// The web app reaches the API through a same-origin proxy, so no CORS
	// headers are needed.
	r.Use(
		middleware.WithTelemetry(cfg.Port),
		middleware.WithRequestLog(slog.Default()),
		middleware.WithSecurityHeaders(),
		chimiddleware.Recoverer,
		chimiddleware.StripSlashes,
		middleware.WithMaxBody(),
		middleware.WithTimeout(requestTimeout),
	)

	r.Get("/docs", docs.Scalar)
	r.Get("/openapi.json", docs.OpenAPI)

	media := storage.NewMediaStore(storage.NewS3(cfg.Storage), cfg.Storage.PublicURL)

	userModule := users.New(db)
	authModule := auth.New(cfg, userModule.Service, rdb)
	placeModule := places.New(db, rdb, media)
	reviewModule := reviews.New(db, placeModule.Service, media)
	subdistrictModule := subdistricts.New(db, media)
	suggestionModule := suggestions.New(db, placeModule.Service)
	reportModule := reports.New(db, placeModule.Service, reviewModule.Service, media.PublicURL)

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

				// POST /v1/places/{id}/reports
				r.Post("/{id}/reports", reportModule.Handler.CreateForPlace)
			})

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

					// DELETE /v1/places/{id}/reviews/mine
					r.Delete("/mine", reviewModule.Handler.DeleteMine)

					// POST /v1/places/{id}/reviews/mine/medias/upload-url
					r.Post("/mine/medias/upload-url", reviewModule.Handler.CreateMediaUpload)

					// POST /v1/places/{id}/reviews/mine/medias
					r.Post("/mine/medias", reviewModule.Handler.AddMedia)

					// DELETE /v1/places/{id}/reviews/mine/medias/{mediaId}
					r.Delete("/mine/medias/{mediaId}", reviewModule.Handler.DeleteMedia)

					// POST /v1/places/{id}/reviews/{reviewId}/reports
					r.Post("/{reviewId}/reports", reportModule.Handler.CreateForReview)
				})

				r.Group(func(r chi.Router) {
					r.Use(middleware.WithAuth(cfg))
					r.Use(middleware.Guard([]string{domain.RoleAdmin}))

					// DELETE /v1/places/{id}/reviews/{reviewId}
					r.Delete("/{reviewId}", reviewModule.Handler.Delete)
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

		r.Route("/reports", func(r chi.Router) {
			r.Use(middleware.WithAuth(cfg))
			r.Use(middleware.Guard([]string{domain.RoleAdmin}))

			// GET /v1/reports
			r.Get("/", reportModule.Handler.List)

			// GET /v1/reports/{id}
			r.Get("/{id}", reportModule.Handler.Show)

			// POST /v1/reports/{id}/remove
			r.Post("/{id}/remove", reportModule.Handler.Remove)

			// POST /v1/reports/{id}/dismiss
			r.Post("/{id}/dismiss", reportModule.Handler.Dismiss)
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
