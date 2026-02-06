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
		r.Patch("/countdowns/{id}", private.UpdateCountdown)
		r.Put("/countdowns/{id}/save", private.SaveExistingCountdown)
		r.Delete("/countdowns/{id}", private.DeleteCountdown)

		// Preview
		r.Post("/preview", private.PreviewGIF)

		// Template routes
		r.Get("/templates/{countdown_id}", private.GetTemplate)
		r.Put("/templates/{id}", private.UpdateTemplate)

		// Palette routes
		r.Get("/palettes", private.ListPalettes)
		r.Post("/palettes", private.CreatePalette)
		r.Put("/palettes/{id}", private.UpdatePalette)
		r.Delete("/palettes/{id}", private.DeletePalette)
	})
}
