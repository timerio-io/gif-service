package private

import (
	"encoding/json"
	"net/http"
	"regexp"

	"gif-service/middleware"
	"gif-service/queries"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

var hexColorRegex = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

type CreatePaletteRequest struct {
	Name           string `json:"name"`
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
	AccentColor    string `json:"accent_color"`
}

type UpdatePaletteRequest struct {
	Name           *string `json:"name,omitempty"`
	PrimaryColor   *string `json:"primary_color,omitempty"`
	SecondaryColor *string `json:"secondary_color,omitempty"`
	AccentColor    *string `json:"accent_color,omitempty"`
}

func isValidHexColor(c string) bool {
	return hexColorRegex.MatchString(c)
}

func CreatePalette(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(string)

	var req CreatePaletteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}
	if !isValidHexColor(req.PrimaryColor) || !isValidHexColor(req.SecondaryColor) || !isValidHexColor(req.AccentColor) {
		http.Error(w, "Colors must be valid hex format like #FF5733", http.StatusBadRequest)
		return
	}

	count, err := queries.CountPalettes(db, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if count >= 20 {
		http.Error(w, "Maximum of 20 palettes reached", http.StatusConflict)
		return
	}

	palette, err := queries.CreatePalette(db, userID, req.Name, req.PrimaryColor, req.SecondaryColor, req.AccentColor)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(palette)
}

func ListPalettes(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(string)

	palettes, err := queries.ListPalettes(db, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(palettes)
}

func UpdatePalette(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(string)
	id := chi.URLParam(r, "id")

	var req UpdatePaletteRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Invalid request body: unknown fields or malformed JSON", http.StatusBadRequest)
		return
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		if *req.Name == "" {
			http.Error(w, "Name cannot be empty", http.StatusBadRequest)
			return
		}
		updates["name"] = *req.Name
	}
	if req.PrimaryColor != nil {
		if !isValidHexColor(*req.PrimaryColor) {
			http.Error(w, "primary_color must be valid hex format like #FF5733", http.StatusBadRequest)
			return
		}
		updates["primary_color"] = *req.PrimaryColor
	}
	if req.SecondaryColor != nil {
		if !isValidHexColor(*req.SecondaryColor) {
			http.Error(w, "secondary_color must be valid hex format like #FF5733", http.StatusBadRequest)
			return
		}
		updates["secondary_color"] = *req.SecondaryColor
	}
	if req.AccentColor != nil {
		if !isValidHexColor(*req.AccentColor) {
			http.Error(w, "accent_color must be valid hex format like #FF5733", http.StatusBadRequest)
			return
		}
		updates["accent_color"] = *req.AccentColor
	}

	if len(updates) == 0 {
		http.Error(w, "No fields to update", http.StatusBadRequest)
		return
	}

	err := queries.UpdatePalette(db, id, userID, updates)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "palette not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func DeletePalette(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(string)
	id := chi.URLParam(r, "id")

	err := queries.DeletePalette(db, id, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "palette not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
