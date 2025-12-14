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

			face := truetype.NewFace(parsedFont, &truetype.Options{Size: 70})

			remaining := time.Until(cfg.EndTime) - time.Duration(i)*time.Second
			timeStr := formatDuration(remaining)

			dc := gg.NewContext(cfg.Width, cfg.Height)
			dc.SetFontFace(face)

			dc.SetColor(cfg.Background)
			dc.Clear()

			dc.SetColor(cfg.TextColor)
			dc.DrawStringAnchored(timeStr, float64(cfg.Width)/2, float64(cfg.Height)/2, 0.5, 0.5)

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

func formatDuration(d time.Duration) string {
	if d < 0 {
		return "00:00:00"
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
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
