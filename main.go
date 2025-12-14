package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"gif-service/handlers"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Response struct {
	Message string `json:"message"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Message: "OK"})
}

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Get("/health", healthHandler)
	r.Post("/generate", handlers.Generate)
	port := ":8080"
	fmt.Printf("\n🚀 Server running at http://localhost%s\n\n", port)

	http.ListenAndServe(port, r)
}
