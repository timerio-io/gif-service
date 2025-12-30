package main

import (
	"encoding/json"
	"fmt"
	"gif-service/handlers/private"
	"gif-service/routes"

	"gif-service/internal/database"
	"log"
	"net/http"

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
	private.SetDB(db.DB)
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Get("/health", healthHandler)
	routes.Setup(r)

	port := ":8080"
	fmt.Printf("\n🚀 Server running at http://localhost%s\n\n", port)

	http.ListenAndServe(port, r)
}
