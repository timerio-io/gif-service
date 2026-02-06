package private

import (
	"encoding/json"
	"fmt"
	"image/color"
	"net/http"
	"time"

	"gif-service/gif"
)

type PreviewRequest struct {
	// General
	TimerType string `json:"timer_type"`
	EndTime   string `json:"end_time,omitempty"`
	Duration  int    `json:"duration,omitempty"`

	// Which units to show
	ShowDays    bool `json:"show_days"`
	ShowHours   bool `json:"show_hours"`
	ShowMinutes bool `json:"show_minutes"`
	ShowSeconds bool `json:"show_seconds"`

	// Numbers
	NumberFont     string `json:"number_font"`
	NumberFontSize int    `json:"number_font_size"`
	NumberColor    string `json:"number_color"`

	// Labels
	ShowLabels    bool   `json:"show_labels"`
	LabelFont     string `json:"label_font"`
	LabelFontSize int    `json:"label_font_size"`
	LabelColor    string `json:"label_color"`

	// Separators
	ShowSeparators bool   `json:"show_separators"`
	SeparatorColor string `json:"separator_color"`

	// Background
	BgColor        string `json:"bg_color"`
	Transparent    bool   `json:"transparent"`
	MatteColor     string `json:"matte_color"`
	RoundedCorners bool   `json:"rounded_corners"`
	CornerRadius   int    `json:"corner_radius"`

	// Expire
	ExpireBehavior     string `json:"expire_behavior"`
	ExpireText         string `json:"expire_text"`
	ExpireTextFont     string `json:"expire_text_font"`
	ExpireTextFontSize int    `json:"expire_text_font_size"`
	ExpireTextColor    string `json:"expire_text_color"`

	// Generate expired (static) preview
	Expired bool `json:"expired"`
}

func PreviewGIF(w http.ResponseWriter, r *http.Request) {
	var req PreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Determine end time
	var endTime time.Time
	if req.TimerType == "fixed" && req.EndTime != "" {
		parsed, err := time.Parse(time.RFC3339, req.EndTime)
		if err != nil {
			http.Error(w, "Invalid end_time format", http.StatusBadRequest)
			return
		}
		endTime = parsed
	} else if req.Duration > 0 {
		endTime = time.Now().Add(time.Duration(req.Duration) * time.Second)
	} else {
		endTime = time.Now().Add(24 * time.Hour)
	}

	// Parse colors with defaults
	bgColor := parseColorOrDefault(req.BgColor, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	if req.Transparent && req.MatteColor != "" {
		bgColor = parseColorOrDefault(req.MatteColor, bgColor)
	}

	numberColor := parseColorOrDefault(req.NumberColor, color.RGBA{R: 0, G: 0, B: 0, A: 255})
	labelColor := parseColorOrDefault(req.LabelColor, numberColor)
	separatorColor := parseColorOrDefault(req.SeparatorColor, numberColor)

	// Number font size
	numberFontSize := float64(req.NumberFontSize)
	if numberFontSize <= 0 {
		numberFontSize = 60
	}

	// Label font size
	labelFontSize := float64(req.LabelFontSize)
	if labelFontSize <= 0 {
		labelFontSize = 14
	}

	cfg := gif.Config{
		EndTime:    endTime,
		Background: bgColor,
		TextColor:  numberColor,

		NumberFontName: req.NumberFont,
		NumberFontSize: numberFontSize,

		ShowLabels:    req.ShowLabels,
		LabelFontName: req.LabelFont,
		LabelFontSize: labelFontSize,
		LabelColor:    labelColor,

		ShowSeparators: req.ShowSeparators,
		SeparatorColor: separatorColor,

		ShowDays:    req.ShowDays,
		ShowHours:   req.ShowHours,
		ShowMinutes: req.ShowMinutes,
		ShowSeconds: req.ShowSeconds,

		Transparent:    req.Transparent,
		RoundedCorners: req.RoundedCorners,
		CornerRadius:   req.CornerRadius,

		Expired:         req.Expired,
		ExpireBehavior:  req.ExpireBehavior,
		ExpireText:      req.ExpireText,
		ExpireTextFont:  req.ExpireTextFont,
		ExpireTextSize:  float64(req.ExpireTextFontSize),
		ExpireTextColor: parseColorOrDefault(req.ExpireTextColor, numberColor),
	}

	// Auto-calculate dimensions based on font sizes and enabled columns
	cfg.CalcDimensions()

	gifBytes, err := gif.Generate(cfg)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate preview: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/gif")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(gifBytes)
}

func parseColorOrDefault(hex string, fallback color.Color) color.Color {
	if hex == "" {
		return fallback
	}
	c, err := parseHexColor(hex)
	if err != nil {
		return fallback
	}
	return c
}

func parseHexColor(hex string) (color.Color, error) {
	var r, g, b uint8
	if len(hex) > 0 && hex[0] == '#' {
		hex = hex[1:]
	}
	if len(hex) != 6 {
		return nil, fmt.Errorf("invalid hex color: %s", hex)
	}
	_, err := fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	if err != nil {
		return nil, err
	}
	return color.RGBA{R: r, G: g, B: b, A: 255}, nil
}
