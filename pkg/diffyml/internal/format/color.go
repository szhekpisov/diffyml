// color.go - ANSI color codes and presentation-layer color helpers.
//
// These are rendering/formatting concerns used only by formatters.
// Configuration-level color types (ColorMode, ColorConfig) remain in types.
package format

import (
	"fmt"

	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/types"
)

// Detailed color palette constants (24-bit RGB values)
const (
	DetailedYellowR, DetailedYellowG, DetailedYellowB = 199, 196, 63
	DetailedRedR, DetailedRedG, DetailedRedB          = 185, 49, 27
	DetailedGreenR, DetailedGreenG, DetailedGreenB    = 88, 191, 56
	DetailedGrayR, DetailedGrayG, DetailedGrayB       = 105, 105, 105
)

// ANSI color codes (8-color fallback)
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorGray   = "\033[90m"
)

// ANSI style codes
const (
	StyleBold      = "\033[1m"
	StyleBoldOff   = "\033[22m"
	StyleItalic    = "\033[3m"
	StyleItalicOff = "\033[23m"
)

// GetTrueColorCode returns an ANSI escape sequence for 24-bit RGB color.
func GetTrueColorCode(r, g, b int) string {
	r = Clamp(r, 0, 255)
	g = Clamp(g, 0, 255)
	b = Clamp(b, 0, 255)
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
}

// GetDetailedColorCode returns the appropriate color code for a diff type.
func GetDetailedColorCode(diffType types.DiffType, useTrueColor bool) string {
	if useTrueColor {
		switch diffType {
		case types.DiffAdded:
			return GetTrueColorCode(DetailedGreenR, DetailedGreenG, DetailedGreenB)
		case types.DiffRemoved:
			return GetTrueColorCode(DetailedRedR, DetailedRedG, DetailedRedB)
		case types.DiffModified, types.DiffOrderChanged:
			return GetTrueColorCode(DetailedYellowR, DetailedYellowG, DetailedYellowB)
		}
	}
	switch diffType {
	case types.DiffAdded:
		return ColorGreen
	case types.DiffRemoved:
		return ColorRed
	case types.DiffModified, types.DiffOrderChanged:
		return ColorYellow
	}
	return ""
}

// GetContextColorCode returns gray color for context lines.
func GetContextColorCode(useTrueColor bool) string {
	if useTrueColor {
		return GetTrueColorCode(DetailedGrayR, DetailedGrayG, DetailedGrayB)
	}
	return ColorGray
}

// GetColorReset returns the ANSI reset code.
func GetColorReset() string {
	return ColorReset
}

// Clamp restricts a value to the range [lo, hi].
func Clamp(val, lo, hi int) int {
	return max(lo, min(val, hi))
}

// Colorizer provides diff-type-aware color codes for formatters.
type Colorizer struct {
	TrueColor bool
}

func (c Colorizer) Added() string    { return GetDetailedColorCode(types.DiffAdded, c.TrueColor) }
func (c Colorizer) Removed() string  { return GetDetailedColorCode(types.DiffRemoved, c.TrueColor) }
func (c Colorizer) Modified() string { return GetDetailedColorCode(types.DiffModified, c.TrueColor) }
func (c Colorizer) Context() string  { return GetContextColorCode(c.TrueColor) }
func (c Colorizer) Reset() string    { return ColorReset }
func (c Colorizer) Bold() string     { return StyleBold }

// CompactColor returns the 8-color ANSI code for a diff type.
func CompactColor(dt types.DiffType) string {
	switch dt {
	case types.DiffAdded:
		return ColorGreen
	case types.DiffRemoved:
		return ColorRed
	case types.DiffModified, types.DiffOrderChanged:
		return ColorYellow
	default:
		return ""
	}
}
