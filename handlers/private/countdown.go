package private

import (
	"encoding/json"
	"net/http"

	"gif-service/queries"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

var db *gorm.DB

func SetDB(database *gorm.DB) {
	db = database
}

func CreateCountdown(w http.ResponseWriter, r *http.Request) {
	userID := "test-user"

	countdown, err := queries.CreateCountdown(db, userID)
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
	userID := "test-user"

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
