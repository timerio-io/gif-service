package private

import (
	"encoding/json"
	"net/http"

	"gif-service/middleware"
	"gif-service/queries"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

var db *gorm.DB

func SetDB(database *gorm.DB) {
	db = database
}

type CreateCountdownRequest struct {
	Name string `json:"name"`
}

type UpdateCountdownRequest struct {
	Name       *string `json:"name,omitempty"`
	PreviewURL *string `json:"preview_url,omitempty"`
	Views      *int    `json:"views,omitempty"`
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

	countdowns, err := queries.ListCountdowns(db, userID)

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
