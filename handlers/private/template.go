package private

import (
	"encoding/json"
	"gif-service/internal/models"
	"gif-service/queries"
	"net/http"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

func GetTemplate(w http.ResponseWriter, r *http.Request) {
	countdownID := chi.URLParam(r, "countdown_id")

	template, err := queries.GetTemplate(db, countdownID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "template not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(template)
}

func UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var updates models.Template

	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := queries.UpdateTemplate(db, id, &updates)

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "template not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
