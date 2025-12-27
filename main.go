package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"gif-service/handlers"
	"gif-service/internal/database"
	"gif-service/queries"

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
	db, err := database.New("./data/timerio.db")
	if err != nil {
		log.Fatal(err)
	}

	countdown, err := queries.CreateCountdown(db.DB, "test-user")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created countdown: %+v\n", countdown)

	found, err := queries.GetCountdownById(db.DB, countdown.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found countdown: %+v\n", found)

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Get("/health", healthHandler)
	r.Post("/generate", handlers.Generate)
	port := ":8080"
	fmt.Printf("\n🚀 Server running at http://localhost%s\n\n", port)

	http.ListenAndServe(port, r)
}
