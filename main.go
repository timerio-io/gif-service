package main

import (
	"encoding/json"
	"fmt"
	"gif-service/handlers/private"
	"gif-service/internal/database"
	"gif-service/internal/storage"
	"gif-service/routes"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
)

type Response struct {
	Message string `json:"message"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Message: "OK"})
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	db, err := database.New("./data/timerio.db")
	if err != nil {
		log.Fatal(err)
	}
	private.SetDB(db.DB)

	r2Client, err := storage.NewR2Client(
		os.Getenv("R2_BUCKET_ENDPOINT"),
		os.Getenv("R2_BUCKET_ACCESS_KEY_ID"),
		os.Getenv("R2_BUCKET_SECRET_ACCESS_KEY"),
		os.Getenv("R2_BUCKET_NAME"),
	)
	if err != nil {
		log.Fatal("Failed to initialize R2 client:", err)
	}
	private.SetR2Client(r2Client)

	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://127.0.0.1:3000", "http://localhost:5173", "http://127.0.0.1:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "Cookie"},
		AllowCredentials: true,
	}))

	r.Use(middleware.Logger)

	r.Get("/health", healthHandler)
	routes.Setup(r)

	port := ":8080"
	fmt.Printf("\n🚀 Server running at http://localhost%s\n\n", port)

	http.ListenAndServe(port, r)
}
