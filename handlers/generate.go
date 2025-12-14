package handlers

import (
	"encoding/json"
	"fmt"
	"image/color"
	"net/http"
	"time"

	"gif-service/gif"
)

type GenerateRequest struct {
	EndTime  string         `json:"endTime"`
	Template TemplateConfig `json:"template"`
	Colors   ColorConfig    `json:"colors"`
}

type TemplateConfig struct {
	FontSize int    `json:"fontSize"`
	Layout   string `json:"layout"`
}

type ColorConfig struct {
	Background string `json:"background"`
	Text       string `json:"text"`
}

func Generate(w http.ResponseWriter, r *http.Request) {
	var req GenerateRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		http.Error(w, "Invalid endTime format", http.StatusBadRequest)
		return
	}

	bgColor, err := parseHexColor(req.Colors.Background)
	if err != nil {
		http.Error(w, "Invalid background color", http.StatusBadRequest)
		return
	}

	textColor, err := parseHexColor(req.Colors.Text)
	if err != nil {
		http.Error(w, "Invalid text color", http.StatusBadRequest)
		return
	}

	gifBytes, err := gif.Generate(gif.Config{
		EndTime:    endTime,
		Background: bgColor,
		TextColor:  textColor,
		Width:      534,
		Height:     143,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate GIF: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/gif")
	w.Write(gifBytes)
}

func parseHexColor(hex string) (color.Color, error) {
	var r, g, b uint8

	if hex[0] == '#' {
		hex = hex[1:]
	}

	_, err := fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	if err != nil {
		return nil, err
	}

	return color.RGBA{R: r, G: g, B: b, A: 255}, nil
}
