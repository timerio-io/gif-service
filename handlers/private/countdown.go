package private

import (
	"encoding/json"
	"fmt"
	"image/color"
	"log"
	"net/http"
	"os"
	"time"

	"gif-service/gif"
	"gif-service/internal/storage"
	"gif-service/middleware"
	"gif-service/queries"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

var db *gorm.DB
var r2Client *storage.R2Client

func SetDB(database *gorm.DB) {
	db = database
}

func SetR2Client(client *storage.R2Client) {
	r2Client = client
}

type CreateCountdownRequest struct {
	Name string `json:"name"`
}

type UpdateCountdownRequest struct {
	Name          *string `json:"name,omitempty"`
	PreviewURL    *string `json:"preview_url,omitempty"`
	Views         *int    `json:"views,omitempty"`
	IsSoftDeleted *bool   `json:"is_soft_deleted,omitempty"`
}

func CreateCountdown(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(string)

	var req CreateCountdownRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	countdown, err := queries.CreateCountdown(db, userID, req.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = queries.CreateTemplate(db, countdown.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	gifBytes, err := gif.Generate(gif.Config{
		EndTime:    time.Now().Add(24 * time.Hour),
		Background: color.RGBA{R: 255, G: 255, B: 255, A: 255},
		TextColor:  color.RGBA{R: 0, G: 0, B: 0, A: 255},
		Width:      350,
		Height:     150,
	})
	if err != nil {
		http.Error(w, "Failed to generate GIF", http.StatusInternalServerError)
		return
	}

	key := fmt.Sprintf("previews/%s.gif", countdown.ID)
	if err := r2Client.UploadGIF(key, gifBytes); err != nil {
		log.Printf("R2 Upload Error: %v", err)
		http.Error(w, fmt.Sprintf("Failed to upload GIF: %v", err), http.StatusInternalServerError)
		return
	}

	previewURL := fmt.Sprintf("%s/%s", os.Getenv("R2_PUBLIC_URL"), key)

	queries.UpdateCountdown(db, countdown.ID, map[string]interface{}{
		"preview_url": previewURL,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(countdown)
}

func GetCountdown(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	countdown, err := queries.GetCountdownById(db, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "countdown not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(countdown)
}

func ListCountdowns(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(string)

	filters := make(map[string]interface{})

	if countdownType := r.URL.Query().Get("type"); countdownType != "" && countdownType != "all" {
		filters["type"] = countdownType
	}

	if archived := r.URL.Query().Get("archived"); archived == "true" {
		filters["is_soft_deleted"] = true
	} else if archived == "all" {
		// Fetch all (active + archived)
	} else {
		filters["is_soft_deleted"] = false
	}

	countdowns, err := queries.ListCountdowns(db, userID, filters)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(countdowns)
}

func DeleteCountdown(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	err := queries.DeleteCountdown(db, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "countdown not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	key := fmt.Sprintf("previews/%s.gif", id)
	if err := r2Client.DeleteObject(key); err != nil {
		log.Printf("R2 Delete Error for %s: %v", id, err)
		// We don't return error here because the DB record is already deleted
	}

	w.WriteHeader(http.StatusNoContent)
}

func UpdateCountdown(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req UpdateCountdownRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Invalid request body: unknown fields or malformed JSON", http.StatusBadRequest)
		return
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.PreviewURL != nil {
		updates["preview_url"] = *req.PreviewURL
	}
	if req.Views != nil {
		updates["views"] = *req.Views
	}
	if req.IsSoftDeleted != nil {
		updates["is_soft_deleted"] = *req.IsSoftDeleted
	}

	if len(updates) == 0 {
		http.Error(w, "No fields to update", http.StatusBadRequest)
		return
	}

	err := queries.UpdateCountdown(db, id, updates)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "countdown not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
