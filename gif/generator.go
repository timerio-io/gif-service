package gif

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"os"
	"sync"
	"time"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
)

type Config struct {
	EndTime    time.Time
	Background color.Color
	TextColor  color.Color
	Width      int
	Height     int
}

var parsedFont *truetype.Font

func init() {
	fontBytes, err := os.ReadFile("fonts/Arial.ttf")
	if err != nil {
		panic(fmt.Sprintf("failed to read font: %v", err))
	}

	parsedFont, err = truetype.Parse(fontBytes)
	if err != nil {
		panic(fmt.Sprintf("failed to parse font: %v", err))
	}
}

func Generate(cfg Config) ([]byte, error) {
	start := time.Now()

	frames := 60
	delay := 100

	anim := gif.GIF{
		Image:     make([]*image.Paletted, frames),
		Delay:     make([]int, frames),
		LoopCount: 0,
	}

	palette := createPalette(cfg.Background, cfg.TextColor)

	var wg sync.WaitGroup

	for i := range frames {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			remaining := time.Until(cfg.EndTime) - time.Duration(i)*time.Second
			days := int(remaining.Hours() / 24)
			hours := int(remaining.Hours()) % 24
			minutes := int(remaining.Minutes()) % 60
			seconds := int(remaining.Seconds()) % 60

			dc := gg.NewContext(cfg.Width, cfg.Height)

			// Background
			dc.SetColor(cfg.Background)
			dc.Clear()

			// Number font (larger)
			numberFace := truetype.NewFace(parsedFont, &truetype.Options{Size: 60})
			dc.SetFontFace(numberFace)

			// Label font
			labelFace := truetype.NewFace(parsedFont, &truetype.Options{Size: 14})

			// Calculate positions for 4 columns
			columnWidth := float64(cfg.Width) / 4

			values := []struct {
				num   string
				label string
			}{
				{fmt.Sprintf("%02d", days), "Days"},
				{fmt.Sprintf("%02d", hours), "Hours"},
				{fmt.Sprintf("%02d", minutes), "Minutes"},
				{fmt.Sprintf("%02d", seconds), "Seconds"},
			}

			for i, v := range values {
				x := columnWidth * (float64(i) + 0.5)

				// Draw number
				dc.SetFontFace(numberFace)
				dc.SetColor(cfg.TextColor)
				dc.DrawStringAnchored(v.num, x, float64(cfg.Height)/2-10, 0.5, 0.5)

				// Draw label below number
				dc.SetFontFace(labelFace)
				dc.DrawStringAnchored(v.label, x, float64(cfg.Height)/2+34, 0.5, 0.5)
			}

			// Draw all separators after text to avoid anti-aliasing issues
			for i := 0; i < 3; i++ {
				separatorX := columnWidth * float64(i+1)
				dc.SetColor(cfg.TextColor)
				dc.SetLineWidth(2.4)
				dc.DrawLine(separatorX, float64(cfg.Height)/2-18, separatorX, float64(cfg.Height)/2+18)
				dc.Stroke()
			}

			bounds := image.Rect(0, 0, cfg.Width, cfg.Height)
			palettedImg := image.NewPaletted(bounds, palette)
			draw.Draw(palettedImg, bounds, dc.Image(), image.Point{}, draw.Src)

			anim.Image[i] = palettedImg
			anim.Delay[i] = delay
		}(i)
	}

	wg.Wait()
	fmt.Printf("Frames generated in: %v\n", time.Since(start))

	encodeStart := time.Now()
	var buf bytes.Buffer
	err := gif.EncodeAll(&buf, &anim)
	fmt.Printf("GIF encoded in: %v\n", time.Since(encodeStart))

	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func createPalette(bg, text color.Color) []color.Color {
	// Just background, text, and smooth gradient between for anti-aliasing
	palette := make([]color.Color, 32)

	bgR, bgG, bgB, _ := bg.RGBA()
	textR, textG, textB, _ := text.RGBA()

	for i := 0; i < 32; i++ {
		t := float64(i) / 31.0
		palette[i] = color.RGBA{
			R: uint8((float64(bgR>>8)*(1-t) + float64(textR>>8)*t)),
			G: uint8((float64(bgG>>8)*(1-t) + float64(textG>>8)*t)),
			B: uint8((float64(bgB>>8)*(1-t) + float64(textB>>8)*t)),
			A: 255,
		}
	}

	return palette
}
