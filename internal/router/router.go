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
	"rantau-go/internal/module/review"
	"rantau-go/internal/module/subdistrict"
	"rantau-go/internal/module/users"
	"rantau-go/internal/storage"
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

	images := storage.NewImages(storage.NewS3(cfg.Storage), cfg.Storage.PublicURL)

	usersModule := users.NewModule(db)
	authModule := auth.NewModule(cfg, usersModule.Service)
	placeModule := place.NewModule(db, images)
	reviewModule := review.NewModule(db)
	subdisctrictModule := subdistrict.NewModule(db, images)

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
				r.Use(middleware.Guard([]string{"admin"}))

				// POST /v1/places
				r.Post("/", placeModule.Handler.CreatePlace)

				// PATCH /v1/places/{id}
				r.Patch("/{id}", placeModule.Handler.PatchPlace)

				// DELETE /v1/places/{id}
				r.Delete("/{id}", placeModule.Handler.DeletePlace)

				// POST /v1/places/{id}/image/upload-url
				r.Post("/{id}/image/upload-url", placeModule.Handler.CreateImageUpload)

				// PUT /v1/places/{id}/image
				r.Put("/{id}/image", placeModule.Handler.SetImage)
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

					// POST /v1/subdistricts/{id}/image/upload-url
					r.Post("/{id}/image/upload-url", subdisctrictModule.Handler.CreateImageUpload(subdistrict.SlotImage))

					// PUT /v1/subdistricts/{id}/image
					r.Put("/{id}/image", subdisctrictModule.Handler.SetImage(subdistrict.SlotImage))

					// POST /v1/subdistricts/{id}/static-map/upload-url
					r.Post("/{id}/static-map/upload-url", subdisctrictModule.Handler.CreateImageUpload(subdistrict.SlotStaticMap))

					// PUT /v1/subdistricts/{id}/static-map
					r.Put("/{id}/static-map", subdisctrictModule.Handler.SetImage(subdistrict.SlotStaticMap))
				})
			})
		})
	})

	return r
}
