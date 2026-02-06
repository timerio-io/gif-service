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
	"gif-service/internal/models"
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

	// Timer config
	TimerType string `json:"timer_type,omitempty"`
	EndTime   string `json:"end_time,omitempty"`
	Duration  *int   `json:"duration,omitempty"`

	// Style config (full JSON for GIF generator)
	StyleConfig map[string]interface{} `json:"style_config,omitempty"`
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

	// Determine countdown type
	countdownType := models.CountdownTypeEvent
	if req.TimerType == "on_send" {
		countdownType = models.CountdownTypeBirthday
	} else if req.TimerType == "on_open" {
		countdownType = models.CountdownTypeHoliday
	}

	// Parse end time
	var endTime *time.Time
	if req.EndTime != "" {
		parsed, err := time.Parse(time.RFC3339, req.EndTime)
		if err == nil {
			endTime = &parsed
		}
	}

	countdown, err := queries.CreateCountdownFull(db, userID, req.Name, countdownType, endTime, req.Duration)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Serialize style config to JSON
	styleConfigJSON := ""
	if req.StyleConfig != nil {
		scBytes, err := json.Marshal(req.StyleConfig)
		if err == nil {
			styleConfigJSON = string(scBytes)
		}
	}

	_, err = queries.CreateTemplateWithStyle(db, countdown.ID, styleConfigJSON)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Build GIF config from style config or use defaults
	gifCfg := buildGIFConfigFromStyle(req.StyleConfig)

	// Set end time for GIF generation
	if endTime != nil {
		gifCfg.EndTime = *endTime
	} else if req.Duration != nil {
		gifCfg.EndTime = time.Now().Add(time.Duration(*req.Duration) * time.Second)
	} else {
		gifCfg.EndTime = time.Now().Add(24 * time.Hour)
	}

	gifCfg.CalcDimensions()

	gifBytes, err := gif.Generate(gifCfg)
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

	countdown.PreviewURL = previewURL

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(countdown)
}

func buildGIFConfigFromStyle(style map[string]interface{}) gif.Config {
	cfg := gif.Config{
		Background:     color.RGBA{R: 255, G: 255, B: 255, A: 255},
		TextColor:      color.RGBA{R: 0, G: 0, B: 0, A: 255},
		NumberFontSize: 60,
		ShowLabels:     true,
		LabelFontSize:  14,
		LabelColor:     color.RGBA{R: 0, G: 0, B: 0, A: 255},
		ShowSeparators: true,
		SeparatorColor: color.RGBA{R: 0, G: 0, B: 0, A: 255},
		ShowDays:       true,
		ShowHours:      true,
		ShowMinutes:    true,
		ShowSeconds:    true,
	}

	if style == nil {
		return cfg
	}

	// Parse colors
	if v, ok := style["number_color"].(string); ok && v != "" {
		cfg.TextColor = parseColorFallback(v, cfg.TextColor)
	}
	if v, ok := style["bg_color"].(string); ok && v != "" {
		cfg.Background = parseColorFallback(v, cfg.Background)
	}
	if v, ok := style["label_color"].(string); ok && v != "" {
		cfg.LabelColor = parseColorFallback(v, cfg.LabelColor)
	}
	if v, ok := style["separator_color"].(string); ok && v != "" {
		cfg.SeparatorColor = parseColorFallback(v, cfg.SeparatorColor)
	}

	// Parse fonts
	if v, ok := style["number_font"].(string); ok {
		cfg.NumberFontName = v
	}
	if v, ok := style["label_font"].(string); ok {
		cfg.LabelFontName = v
	}

	// Parse sizes
	if v, ok := style["number_font_size"].(float64); ok && v > 0 {
		cfg.NumberFontSize = v
	}
	if v, ok := style["label_font_size"].(float64); ok && v > 0 {
		cfg.LabelFontSize = v
	}

	// Parse booleans
	if v, ok := style["show_labels"].(bool); ok {
		cfg.ShowLabels = v
	}
	if v, ok := style["show_separators"].(bool); ok {
		cfg.ShowSeparators = v
	}
	if v, ok := style["show_days"].(bool); ok {
		cfg.ShowDays = v
	}
	if v, ok := style["show_hours"].(bool); ok {
		cfg.ShowHours = v
	}
	if v, ok := style["show_minutes"].(bool); ok {
		cfg.ShowMinutes = v
	}
	if v, ok := style["show_seconds"].(bool); ok {
		cfg.ShowSeconds = v
	}
	if v, ok := style["transparent"].(bool); ok {
		cfg.Transparent = v
	}
	if v, ok := style["rounded_corners"].(bool); ok {
		cfg.RoundedCorners = v
	}
	if v, ok := style["corner_radius"].(float64); ok {
		cfg.CornerRadius = int(v)
	}

	return cfg
}

func parseColorFallback(hex string, fallback color.Color) color.Color {
	if hex == "" {
		return fallback
	}
	c, err := parseHexColor(hex)
	if err != nil {
		return fallback
	}
	return c
}

type SaveCountdownRequest struct {
	Name        string                 `json:"name"`
	TimerType   string                 `json:"timer_type,omitempty"`
	EndTime     string                 `json:"end_time,omitempty"`
	Duration    *int                   `json:"duration,omitempty"`
	StyleConfig map[string]interface{} `json:"style_config,omitempty"`
}

func SaveExistingCountdown(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req SaveCountdownRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update countdown fields
	countdownType := models.CountdownTypeEvent
	if req.TimerType == "on_send" {
		countdownType = models.CountdownTypeBirthday
	} else if req.TimerType == "on_open" {
		countdownType = models.CountdownTypeHoliday
	}

	var endTime *time.Time
	if req.EndTime != "" {
		parsed, err := time.Parse(time.RFC3339, req.EndTime)
		if err == nil {
			endTime = &parsed
		}
	}

	updates := map[string]interface{}{
		"type": countdownType,
	}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if endTime != nil {
		updates["end_time"] = endTime
	}
	if req.Duration != nil {
		updates["duration"] = *req.Duration
	}

	if err := queries.UpdateCountdown(db, id, updates); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update template style config
	styleConfigJSON := ""
	if req.StyleConfig != nil {
		scBytes, err := json.Marshal(req.StyleConfig)
		if err == nil {
			styleConfigJSON = string(scBytes)
		}
	}

	template, err := queries.GetTemplate(db, id)
	if err != nil {
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}

	if err := queries.UpdateTemplate(db, template.ID, &models.Template{StyleConfig: styleConfigJSON}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Regenerate GIF
	gifCfg := buildGIFConfigFromStyle(req.StyleConfig)
	if endTime != nil {
		gifCfg.EndTime = *endTime
	} else if req.Duration != nil {
		gifCfg.EndTime = time.Now().Add(time.Duration(*req.Duration) * time.Second)
	} else {
		gifCfg.EndTime = time.Now().Add(24 * time.Hour)
	}
	gifCfg.CalcDimensions()

	gifBytes, err := gif.Generate(gifCfg)
	if err != nil {
		http.Error(w, "Failed to generate GIF", http.StatusInternalServerError)
		return
	}

	key := fmt.Sprintf("previews/%s.gif", id)
	if err := r2Client.UploadGIF(key, gifBytes); err != nil {
		log.Printf("R2 Upload Error: %v", err)
		http.Error(w, fmt.Sprintf("Failed to upload GIF: %v", err), http.StatusInternalServerError)
		return
	}

	previewURL := fmt.Sprintf("%s/%s", os.Getenv("R2_PUBLIC_URL"), key)
	queries.UpdateCountdown(db, id, map[string]interface{}{"preview_url": previewURL})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"id": id, "preview_url": previewURL})
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
