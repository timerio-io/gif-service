package gif

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"os"
	"sync"
	"time"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
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

// ---------------------------------------------------------------------------
// Sprite cache (only addition to your original code)
// ---------------------------------------------------------------------------

type spriteCache struct {
	sprites [100]*image.Paletted
	spriteW int
	spriteH int
	palette []color.Color
}

type cacheKey struct {
	BgColor   uint32
	TextColor uint32
}

var (
	spriteCacheMap   = make(map[cacheKey]*spriteCache)
	spriteCacheMapMu sync.RWMutex
)

func packColor(c color.Color) uint32 {
	r, g, b, a := c.RGBA()
	return (r>>8)<<24 | (g>>8)<<16 | (b>>8)<<8 | (a >> 8)
}

func getOrBuildSpriteCache(cfg Config) (*spriteCache, bool) {
	key := cacheKey{
		BgColor:   packColor(cfg.Background),
		TextColor: packColor(cfg.TextColor),
	}

	spriteCacheMapMu.RLock()
	if cached, ok := spriteCacheMap[key]; ok {
		spriteCacheMapMu.RUnlock()
		return cached, true
	}
	spriteCacheMapMu.RUnlock()

	numberFace := truetype.NewFace(parsedFont, &truetype.Options{Size: 60})
	cache := buildSpriteCache(cfg, numberFace)

	spriteCacheMapMu.Lock()
	spriteCacheMap[key] = cache
	spriteCacheMapMu.Unlock()

	return cache, false
}

// ---------------------------------------------------------------------------
// Everything below is your original code, only Generate() changed slightly
// ---------------------------------------------------------------------------

func buildSpriteCache(cfg Config, numberFace font.Face) *spriteCache {
	palette := createPalette(cfg.Background, cfg.TextColor)

	dc := gg.NewContext(1, 1)
	dc.SetFontFace(numberFace)
	tw, th := dc.MeasureString("00")
	pad := 6.0
	spriteW := int(tw + pad*2)
	spriteH := int(th + pad*2)

	cache := &spriteCache{
		spriteW: spriteW,
		spriteH: spriteH,
		palette: palette,
	}

	for v := 0; v < 100; v++ {
		txt := fmt.Sprintf("%02d", v)
		nf := truetype.NewFace(parsedFont, &truetype.Options{Size: 60})

		dc := gg.NewContext(spriteW, spriteH)
		dc.SetColor(cfg.Background)
		dc.Clear()
		dc.SetFontFace(nf)
		dc.SetColor(cfg.TextColor)
		dc.DrawStringAnchored(txt, float64(spriteW)/2, float64(spriteH)/2, 0.5, 0.5)

		cache.sprites[v] = quantizeNearestNeighbor(
			dc.Image(),
			image.Rect(0, 0, spriteW, spriteH),
			palette,
		)
	}

	return cache
}

func buildBaseFrame(cfg Config, palette []color.Color, labelFace font.Face) *image.Paletted {
	dc := gg.NewContext(cfg.Width, cfg.Height)

	dc.SetColor(cfg.Background)
	dc.Clear()

	columnWidth := float64(cfg.Width) / 4
	labels := []string{"Days", "Hours", "Minutes", "Seconds"}

	dc.SetFontFace(labelFace)
	dc.SetColor(cfg.TextColor)
	for i, label := range labels {
		x := columnWidth * (float64(i) + 0.5)
		dc.DrawStringAnchored(label, x, float64(cfg.Height)/2+34, 0.5, 0.5)
	}

	for i := 0; i < 3; i++ {
		separatorX := columnWidth * float64(i+1)
		dc.SetColor(cfg.TextColor)
		dc.SetLineWidth(2.4)
		dc.DrawLine(separatorX, float64(cfg.Height)/2-18, separatorX, float64(cfg.Height)/2+18)
		dc.Stroke()
	}

	return quantizeNearestNeighbor(
		dc.Image(),
		image.Rect(0, 0, cfg.Width, cfg.Height),
		palette,
	)
}

func stampSprite(dst *image.Paletted, sprite *image.Paletted, x, y int) {
	srcBounds := sprite.Bounds()
	dstBounds := dst.Bounds()
	dstStride := dst.Stride
	srcStride := sprite.Stride

	for sy := srcBounds.Min.Y; sy < srcBounds.Max.Y; sy++ {
		dy := y + sy - srcBounds.Min.Y
		if dy < dstBounds.Min.Y || dy >= dstBounds.Max.Y {
			continue
		}
		for sx := srcBounds.Min.X; sx < srcBounds.Max.X; sx++ {
			dx := x + sx - srcBounds.Min.X
			if dx < dstBounds.Min.X || dx >= dstBounds.Max.X {
				continue
			}
			srcIdx := (sy-srcBounds.Min.Y)*srcStride + (sx - srcBounds.Min.X)
			dstIdx := (dy-dstBounds.Min.Y)*dstStride + (dx - dstBounds.Min.X)
			dst.Pix[dstIdx] = sprite.Pix[srcIdx]
		}
	}
}

func computeDiff(prev, curr *image.Paletted) *image.Paletted {
	bounds := curr.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	stride := curr.Stride

	minX, minY := w, h
	maxX, maxY := 0, 0

	for y := 0; y < h; y++ {
		rowOffset := y * stride
		for x := 0; x < w; x++ {
			if prev.Pix[rowOffset+x] != curr.Pix[rowOffset+x] {
				if x < minX {
					minX = x
				}
				if x > maxX {
					maxX = x
				}
				if y < minY {
					minY = y
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}

	if maxX < minX {
		tiny := image.NewPaletted(image.Rect(0, 0, 1, 1), curr.Palette)
		tiny.Pix[0] = 0
		return tiny
	}

	diffRect := image.Rect(minX, minY, maxX+1, maxY+1)
	diff := image.NewPaletted(diffRect, curr.Palette)

	diffW := diffRect.Dx()
	for y := diffRect.Min.Y; y < diffRect.Max.Y; y++ {
		srcOffset := y*stride + diffRect.Min.X
		dstOffset := (y - diffRect.Min.Y) * diff.Stride
		copy(diff.Pix[dstOffset:dstOffset+diffW], curr.Pix[srcOffset:srcOffset+diffW])
	}

	return diff
}

func Generate(cfg Config) ([]byte, error) {
	start := time.Now()

	frames := 60
	delay := 100

	// Only change: use cached sprites instead of building every time
	cache, cacheHit := getOrBuildSpriteCache(cfg)
	if cacheHit {
		fmt.Println("Sprite cache HIT")
	} else {
		fmt.Printf("Sprite cache MISS — built in: %v\n", time.Since(start))
	}

	labelFace := truetype.NewFace(parsedFont, &truetype.Options{Size: 14})
	baseFrame := buildBaseFrame(cfg, cache.palette, labelFace)

	stampStart := time.Now()
	columnWidth := float64(cfg.Width) / 4
	numY := cfg.Height/2 - 10 - cache.spriteH/2

	fullFrames := make([]*image.Paletted, frames)

	for i := 0; i < frames; i++ {
		remaining := time.Until(cfg.EndTime) - time.Duration(i)*time.Second
		days, hours, minutes, seconds := splitDuration(remaining)

		frame := image.NewPaletted(baseFrame.Bounds(), cache.palette)
		copy(frame.Pix, baseFrame.Pix)

		values := []int{days, hours, minutes, seconds}
		for col, val := range values {
			cx := int(columnWidth * (float64(col) + 0.5))
			pasteX := cx - cache.spriteW/2
			stampSprite(frame, cache.sprites[val], pasteX, numY)
		}

		fullFrames[i] = frame
	}
	fmt.Printf("60 frames stamped in: %v\n", time.Since(stampStart))

	diffStart := time.Now()
	anim := gif.GIF{
		Image:     make([]*image.Paletted, frames),
		Delay:     make([]int, frames),
		Disposal:  make([]byte, frames),
		LoopCount: 0,
	}

	anim.Image[0] = fullFrames[0]
	anim.Delay[0] = delay
	anim.Disposal[0] = gif.DisposalNone

	for i := 1; i < frames; i++ {
		anim.Image[i] = computeDiff(fullFrames[i-1], fullFrames[i])
		anim.Delay[i] = delay
		anim.Disposal[i] = gif.DisposalNone
	}
	fmt.Printf("Diffs computed in: %v\n", time.Since(diffStart))

	encodeStart := time.Now()
	var buf bytes.Buffer
	err := gif.EncodeAll(&buf, &anim)
	fmt.Printf("GIF encoded in: %v\n", time.Since(encodeStart))
	fmt.Printf("GIF size: %d bytes (%d KB)\n", buf.Len(), buf.Len()/1024)
	fmt.Printf("Total: %v\n", time.Since(start))

	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func quantizeNearestNeighbor(src image.Image, bounds image.Rectangle, palette []color.Color) *image.Paletted {
	dst := image.NewPaletted(bounds, palette)

	type rgbaColor struct{ r, g, b uint32 }
	palRGBA := make([]rgbaColor, len(palette))
	for i, c := range palette {
		r, g, b, _ := c.RGBA()
		palRGBA[i] = rgbaColor{r >> 8, g >> 8, b >> 8}
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := src.At(x, y).RGBA()
			sr, sg, sb := r>>8, g>>8, b>>8

			bestIdx := 0
			bestDist := uint32(1<<32 - 1)
			for i, pc := range palRGBA {
				dr := sr - pc.r
				dg := sg - pc.g
				db := sb - pc.b
				dist := dr*dr + dg*dg + db*db
				if dist < bestDist {
					bestDist = dist
					bestIdx = i
				}
			}

			dst.SetColorIndex(x, y, uint8(bestIdx))
		}
	}

	return dst
}

func splitDuration(remaining time.Duration) (int, int, int, int) {
	if remaining < 0 {
		return 0, 0, 0, 0
	}
	days := int(remaining.Hours() / 24)
	hours := int(remaining.Hours()) % 24
	minutes := int(remaining.Minutes()) % 60
	seconds := int(remaining.Seconds()) % 60
	return days, hours, minutes, seconds
}

func createPalette(bg, text color.Color) []color.Color {
	bgR, bgG, bgB, _ := bg.RGBA()
	textR, textG, textB, _ := text.RGBA()

	palette := []color.Color{bg}
	steps := 6
	for i := 1; i <= steps; i++ {
		t := float64(i) / float64(steps+1)
		palette = append(palette, color.RGBA{
			R: uint8(float64(bgR>>8)*(1-t) + float64(textR>>8)*t),
			G: uint8(float64(bgG>>8)*(1-t) + float64(textG>>8)*t),
			B: uint8(float64(bgB>>8)*(1-t) + float64(textB>>8)*t),
			A: 255,
		})
	}
	palette = append(palette, text)

	return palette
}
