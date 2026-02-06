package gif

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"math"
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

	// Number styling
	NumberFontName string
	NumberFontSize float64

	// Label styling
	ShowLabels    bool
	LabelFontName string
	LabelFontSize float64
	LabelColor    color.Color

	// Separator styling
	ShowSeparators bool
	SeparatorColor color.Color

	// Which units to display
	ShowDays    bool
	ShowHours   bool
	ShowMinutes bool
	ShowSeconds bool

	// Background options
	Transparent    bool
	RoundedCorners bool
	CornerRadius   int

	// Expired state
	Expired        bool
	ExpireBehavior string  // "show_zeros", "hide", "custom_text"
	ExpireText     string  // Custom text to show when expired
	ExpireTextFont string  // Font for expire text
	ExpireTextSize float64 // Font size for expire text
	ExpireTextColor color.Color
}

func init() {
	initFontRegistry()
}

// columnCount returns how many time columns are enabled.
func (c Config) columnCount() int {
	n := 0
	if c.ShowDays {
		n++
	}
	if c.ShowHours {
		n++
	}
	if c.ShowMinutes {
		n++
	}
	if c.ShowSeconds {
		n++
	}
	if n == 0 {
		return 4 // fallback: show all
	}
	return n
}

// enabledColumns returns the ordered list of column indices (0=days,1=hours,2=minutes,3=seconds) that are enabled.
func (c Config) enabledColumns() []int {
	var cols []int
	if c.ShowDays {
		cols = append(cols, 0)
	}
	if c.ShowHours {
		cols = append(cols, 1)
	}
	if c.ShowMinutes {
		cols = append(cols, 2)
	}
	if c.ShowSeconds {
		cols = append(cols, 3)
	}
	if len(cols) == 0 {
		return []int{0, 1, 2, 3}
	}
	return cols
}

// numberFontSizeVal returns the configured number font size or a default.
func (c Config) numberFontSizeVal() float64 {
	if c.NumberFontSize > 0 {
		return c.NumberFontSize
	}
	return 60
}

// labelFontSizeVal returns the configured label font size or a default.
func (c Config) labelFontSizeVal() float64 {
	if c.LabelFontSize > 0 {
		return c.LabelFontSize
	}
	return 14
}

// CalcDimensions computes the ideal Width and Height based on font sizes, columns, labels, etc.
func (c *Config) CalcDimensions() {
	numFont := GetFont(c.NumberFontName)
	numFace := truetype.NewFace(numFont, &truetype.Options{Size: c.numberFontSizeVal()})

	dc := gg.NewContext(1, 1)
	dc.SetFontFace(numFace)
	numW, numH := dc.MeasureString("00")

	numCols := c.columnCount()

	// Column width: number width + horizontal padding on each side
	colPad := c.numberFontSizeVal() * 0.25
	if colPad < 6 {
		colPad = 6
	}
	columnWidth := numW + colPad*2

	// Separator gap between columns
	sepGap := c.numberFontSizeVal() * 0.03
	if sepGap < 1 {
		sepGap = 1
	}

	totalWidth := columnWidth*float64(numCols) + sepGap*float64(numCols-1)

	// Height: top padding + number + gap + label + bottom padding
	topPad := c.numberFontSizeVal() * 0.35
	if topPad < 12 {
		topPad = 12
	}
	bottomPad := topPad

	labelH := 0.0
	labelGap := 0.0
	if c.ShowLabels {
		labelGap = c.numberFontSizeVal() * 0.08
		if labelGap < 3 {
			labelGap = 3
		}
		labelH = c.labelFontSizeVal() * 1.3
	}

	totalHeight := topPad + numH + labelGap + labelH + bottomPad

	c.Width = int(math.Ceil(totalWidth))
	c.Height = int(math.Ceil(totalHeight))
}

// ---------------------------------------------------------------------------
// Sprite cache
// ---------------------------------------------------------------------------

type spriteCache struct {
	sprites [100]*image.Paletted
	spriteW int
	spriteH int
	palette []color.Color
}

type cacheKey struct {
	BgColor        uint32
	TextColor      uint32
	NumberFontName string
	NumberFontSize float64
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
		BgColor:        packColor(cfg.Background),
		TextColor:      packColor(cfg.TextColor),
		NumberFontName: cfg.NumberFontName,
		NumberFontSize: cfg.numberFontSizeVal(),
	}

	spriteCacheMapMu.RLock()
	if cached, ok := spriteCacheMap[key]; ok {
		spriteCacheMapMu.RUnlock()
		return cached, true
	}
	spriteCacheMapMu.RUnlock()

	numberFont := GetFont(cfg.NumberFontName)
	numberFace := truetype.NewFace(numberFont, &truetype.Options{Size: cfg.numberFontSizeVal()})
	cache := buildSpriteCache(cfg, numberFace)

	spriteCacheMapMu.Lock()
	spriteCacheMap[key] = cache
	spriteCacheMapMu.Unlock()

	return cache, false
}

func buildSpriteCache(cfg Config, numberFace font.Face) *spriteCache {
	palette := createPalette(cfg.Background, cfg.TextColor, cfg.LabelColor, cfg.SeparatorColor)

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

	numberFont := GetFont(cfg.NumberFontName)
	for v := 0; v < 100; v++ {
		txt := fmt.Sprintf("%02d", v)
		nf := truetype.NewFace(numberFont, &truetype.Options{Size: cfg.numberFontSizeVal()})

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

func buildBaseFrame(cfg Config, palette []color.Color, labelFace font.Face, spriteH int) *image.Paletted {
	dc := gg.NewContext(cfg.Width, cfg.Height)

	// Draw rounded rectangle background or plain fill
	if cfg.RoundedCorners && cfg.CornerRadius > 0 {
		dc.SetColor(color.Transparent)
		dc.Clear()
		radius := float64(cfg.CornerRadius)
		dc.DrawRoundedRectangle(0, 0, float64(cfg.Width), float64(cfg.Height), radius)
		dc.SetColor(cfg.Background)
		dc.Fill()
	} else {
		dc.SetColor(cfg.Background)
		dc.Clear()
	}

	allLabels := []string{"Days", "Hours", "Minutes", "Seconds"}
	enabledCols := cfg.enabledColumns()
	numCols := len(enabledCols)

	// Calculate dynamic spacing based on number font size
	fontSize := cfg.numberFontSizeVal()
	colPad := fontSize * 0.25
	if colPad < 4 {
		colPad = 4
	}
	topPad := fontSize * 0.35
	if topPad < 12 {
		topPad = 12
	}

	// Number center Y position
	numCenterY := topPad + float64(spriteH)/2

	// Separator dimensions scale with number font size
	sepHeight := fontSize * 0.6
	sepLineWidth := math.Max(1.5, fontSize*0.04)
	sepOffsetY := 10.0 // push separator down for alignment

	// Column width
	sepGap := fontSize * 0.03
	if sepGap < 1 {
		sepGap = 1
	}

	numFont := GetFont(cfg.NumberFontName)
	numFace := truetype.NewFace(numFont, &truetype.Options{Size: fontSize})
	dcMeasure := gg.NewContext(1, 1)
	dcMeasure.SetFontFace(numFace)
	numW, _ := dcMeasure.MeasureString("00")
	columnWidth := numW + colPad*2

	// Draw labels
	if cfg.ShowLabels {
		dc.SetFontFace(labelFace)
		labelColor := cfg.LabelColor
		if labelColor == nil {
			labelColor = cfg.TextColor
		}
		dc.SetColor(labelColor)

		labelGap := fontSize * 0.08
		if labelGap < 3 {
			labelGap = 3
		}
		labelY := topPad + float64(spriteH) + labelGap + cfg.labelFontSizeVal()*0.5

		for i, colIdx := range enabledCols {
			x := float64(i)*(columnWidth+sepGap) + columnWidth/2
			dc.DrawStringAnchored(allLabels[colIdx], x, labelY, 0.5, 0.5)
		}
	}

	// Draw separators
	if cfg.ShowSeparators {
		sepColor := cfg.SeparatorColor
		if sepColor == nil {
			sepColor = cfg.TextColor
		}
		dc.SetColor(sepColor)
		dc.SetLineWidth(sepLineWidth)
		for i := 0; i < numCols-1; i++ {
			separatorX := float64(i+1)*columnWidth + float64(i)*sepGap + sepGap/2
			dc.DrawLine(separatorX, numCenterY-sepHeight/2+sepOffsetY, separatorX, numCenterY+sepHeight/2+sepOffsetY)
			dc.Stroke()
		}
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

	// Handle expired "hide" — return a 1x1 transparent GIF
	if cfg.Expired && cfg.ExpireBehavior == "hide" {
		return generateHideGIF()
	}

	// Handle expired "custom_text" — single frame with centered text
	if cfg.Expired && cfg.ExpireBehavior == "custom_text" && cfg.ExpireText != "" {
		return generateCustomTextGIF(cfg)
	}

	frames := 60
	delay := 100

	// Expired mode (show_zeros): single frame with all zeros
	if cfg.Expired {
		frames = 1
	}

	// Auto-calculate dimensions if not explicitly set or if set to 0
	if cfg.Width == 0 || cfg.Height == 0 {
		cfg.CalcDimensions()
	}

	// Use cached sprites instead of building every time
	cache, cacheHit := getOrBuildSpriteCache(cfg)
	if cacheHit {
		fmt.Println("Sprite cache HIT")
	} else {
		fmt.Printf("Sprite cache MISS — built in: %v\n", time.Since(start))
	}

	labelFont := GetFont(cfg.LabelFontName)
	labelFace := truetype.NewFace(labelFont, &truetype.Options{Size: cfg.labelFontSizeVal()})
	baseFrame := buildBaseFrame(cfg, cache.palette, labelFace, cache.spriteH)

	stampStart := time.Now()
	enabledCols := cfg.enabledColumns()

	// Calculate column positions using same logic as buildBaseFrame
	fontSize := cfg.numberFontSizeVal()
	colPad := fontSize * 0.25
	if colPad < 4 {
		colPad = 4
	}
	topPad := fontSize * 0.35
	if topPad < 12 {
		topPad = 12
	}
	sepGap := fontSize * 0.03
	if sepGap < 1 {
		sepGap = 1
	}

	numFont := GetFont(cfg.NumberFontName)
	numFace := truetype.NewFace(numFont, &truetype.Options{Size: fontSize})
	dcMeasure := gg.NewContext(1, 1)
	dcMeasure.SetFontFace(numFace)
	numW, _ := dcMeasure.MeasureString("00")
	columnWidth := numW + colPad*2

	numY := int(topPad)

	fullFrames := make([]*image.Paletted, frames)

	for i := 0; i < frames; i++ {
		var days, hours, minutes, seconds int
		if cfg.Expired {
			// All zeros for expired state
			days, hours, minutes, seconds = 0, 0, 0, 0
		} else {
			remaining := time.Until(cfg.EndTime) - time.Duration(i)*time.Second
			days, hours, minutes, seconds = splitDuration(remaining)
		}

		frame := image.NewPaletted(baseFrame.Bounds(), cache.palette)
		copy(frame.Pix, baseFrame.Pix)

		allValues := []int{days, hours, minutes, seconds}
		for col, colIdx := range enabledCols {
			val := allValues[colIdx]
			cx := int(float64(col)*(columnWidth+sepGap) + columnWidth/2)
			pasteX := cx - cache.spriteW/2
			stampSprite(frame, cache.sprites[val], pasteX, numY)
		}

		fullFrames[i] = frame
	}
	fmt.Printf("%d frames stamped in: %v\n", frames, time.Since(stampStart))

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

func generateHideGIF() ([]byte, error) {
	palette := []color.Color{color.Transparent}
	frame := image.NewPaletted(image.Rect(0, 0, 1, 1), palette)
	frame.SetColorIndex(0, 0, 0)

	anim := gif.GIF{
		Image:     []*image.Paletted{frame},
		Delay:     []int{0},
		Disposal:  []byte{gif.DisposalNone},
		LoopCount: 0,
	}

	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, &anim); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func generateCustomTextGIF(cfg Config) ([]byte, error) {
	textSize := cfg.ExpireTextSize
	if textSize <= 0 {
		textSize = 24
	}
	textColor := cfg.ExpireTextColor
	if textColor == nil {
		textColor = cfg.TextColor
	}

	textFont := GetFont(cfg.ExpireTextFont)
	textFace := truetype.NewFace(textFont, &truetype.Options{Size: textSize})

	// Measure text to determine dimensions
	dc := gg.NewContext(1, 1)
	dc.SetFontFace(textFace)
	tw, th := dc.MeasureString(cfg.ExpireText)

	padX := textSize * 0.5
	padY := textSize * 0.5
	width := int(math.Ceil(tw + padX*2))
	height := int(math.Ceil(th + padY*2))

	// If the normal countdown dimensions are larger, use those
	cfg.CalcDimensions()
	if cfg.Width > width {
		width = cfg.Width
	}
	if cfg.Height > height {
		height = cfg.Height
	}

	// Draw frame
	dc = gg.NewContext(width, height)
	if cfg.RoundedCorners && cfg.CornerRadius > 0 {
		dc.SetColor(color.Transparent)
		dc.Clear()
		dc.DrawRoundedRectangle(0, 0, float64(width), float64(height), float64(cfg.CornerRadius))
		dc.SetColor(cfg.Background)
		dc.Fill()
	} else {
		dc.SetColor(cfg.Background)
		dc.Clear()
	}

	dc.SetFontFace(textFace)
	dc.SetColor(textColor)
	dc.DrawStringAnchored(cfg.ExpireText, float64(width)/2, float64(height)/2, 0.5, 0.5)

	palette := createPalette(cfg.Background, textColor)
	frame := quantizeNearestNeighbor(dc.Image(), image.Rect(0, 0, width, height), palette)

	anim := gif.GIF{
		Image:     []*image.Paletted{frame},
		Delay:     []int{0},
		Disposal:  []byte{gif.DisposalNone},
		LoopCount: 0,
	}

	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, &anim); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func createPalette(bg, text color.Color, extra ...color.Color) []color.Color {
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

	// Add extra colors (label, separator) with interpolation steps to bg
	for _, ec := range extra {
		if ec == nil {
			continue
		}
		ecR, ecG, ecB, _ := ec.RGBA()
		for i := 1; i <= 3; i++ {
			t := float64(i) / 4.0
			palette = append(palette, color.RGBA{
				R: uint8(float64(bgR>>8)*(1-t) + float64(ecR>>8)*t),
				G: uint8(float64(bgG>>8)*(1-t) + float64(ecG>>8)*t),
				B: uint8(float64(bgB>>8)*(1-t) + float64(ecB>>8)*t),
				A: 255,
			})
		}
		palette = append(palette, ec)
	}

	return palette
}
