package gif

import (
	"fmt"
	"os"
	"sync"

	"github.com/golang/freetype/truetype"
)

var (
	fontRegistry   = make(map[string]*truetype.Font)
	fontRegistryMu sync.RWMutex
	fallbackFont   *truetype.Font
)

// fontFileMap maps frontend font names to local font file paths.
// For now only Arial is available; more fonts can be added later.
var fontFileMap = map[string]string{
	"Arial": "fonts/Arial.ttf",
}

func initFontRegistry() {
	fontBytes, err := os.ReadFile("fonts/Arial.ttf")
	if err != nil {
		panic(fmt.Sprintf("failed to read fallback font: %v", err))
	}
	f, err := truetype.Parse(fontBytes)
	if err != nil {
		panic(fmt.Sprintf("failed to parse fallback font: %v", err))
	}
	fallbackFont = f
	fontRegistry["Arial"] = f
}

// GetFont returns the font for the given name, or the Arial fallback.
func GetFont(name string) *truetype.Font {
	if name == "" {
		return fallbackFont
	}

	fontRegistryMu.RLock()
	if f, ok := fontRegistry[name]; ok {
		fontRegistryMu.RUnlock()
		return f
	}
	fontRegistryMu.RUnlock()

	// Try to load from fontFileMap
	if path, ok := fontFileMap[name]; ok {
		fontRegistryMu.Lock()
		defer fontRegistryMu.Unlock()

		// Double-check after acquiring write lock
		if f, ok := fontRegistry[name]; ok {
			return f
		}

		fontBytes, err := os.ReadFile(path)
		if err != nil {
			return fallbackFont
		}
		f, err := truetype.Parse(fontBytes)
		if err != nil {
			return fallbackFont
		}
		fontRegistry[name] = f
		return f
	}

	return fallbackFont
}
