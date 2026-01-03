package routes

import (
	"gif-service/handlers/private"
	"gif-service/handlers/public"
	"gif-service/middleware"

	"github.com/go-chi/chi/v5"
)

func Setup(r *chi.Mux) {
	r.Post("/generate", public.Generate)

	r.Route("/api", func(r chi.Router) {
		r.Use(middleware.Auth)

		// Countdown routes
		r.Get("/countdowns", private.ListCountdowns)
		r.Post("/countdowns", private.CreateCountdown)
		r.Get("/countdowns/{id}", private.GetCountdown)
		r.Delete("/countdowns/{id}", private.DeleteCountdown)

		// Template routes
		r.Get("/templates/{countdown_id}", private.GetTemplate)
		r.Put("/templates/{id}", private.UpdateTemplate)
	})
}
