// color.go - Terminal color detection and ANSI escape codes.
package diffyml

import (
	"os"

	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/format"
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/types"
)

// Type aliases
type ColorMode = types.ColorMode
type ColorConfig = types.ColorConfig
type Colorizer = format.Colorizer

// Constants
const (
	ColorModeAuto   = types.ColorModeAuto
	ColorModeAlways = types.ColorModeAlways
	ColorModeNever  = types.ColorModeNever
)

// ANSI color codes (unexported in facade, exported in format)
const (
	colorReset  = format.ColorReset
	colorRed    = format.ColorRed
	colorGreen  = format.ColorGreen
	colorYellow = format.ColorYellow
	colorCyan   = format.ColorCyan
	colorWhite  = format.ColorWhite
	colorGray   = format.ColorGray
)

// ANSI style codes
const (
	styleBold      = format.StyleBold
	styleBoldOff   = format.StyleBoldOff
	styleItalic    = format.StyleItalic
	styleItalicOff = format.StyleItalicOff
)

// Detailed color palette constants
const (
	DetailedYellowR = format.DetailedYellowR
	DetailedYellowG = format.DetailedYellowG
	DetailedYellowB = format.DetailedYellowB
	DetailedRedR    = format.DetailedRedR
	DetailedRedG    = format.DetailedRedG
	DetailedRedB    = format.DetailedRedB
	DetailedGreenR  = format.DetailedGreenR
	DetailedGreenG  = format.DetailedGreenG
	DetailedGreenB  = format.DetailedGreenB
	DetailedGrayR   = format.DetailedGrayR
	DetailedGrayG   = format.DetailedGrayG
	DetailedGrayB   = format.DetailedGrayB
)

// stdoutStatFn is injectable for testing. Tests can reassign this variable
// and IsTerminal will sync it to the internal package before delegating.
var stdoutStatFn = func() (os.FileInfo, error) {
	return os.Stdout.Stat()
}

// IsTerminal checks if the given file descriptor is a terminal.
// Delegates to the internal implementation, syncing the test-overridable
// stdoutStatFn so that mutations in the internal code are exercised.
func IsTerminal(fd uintptr) bool {
	types.StdoutStatFn = stdoutStatFn
	return types.IsTerminal(fd)
}

func ParseColorMode(s string) (ColorMode, error) { return types.ParseColorMode(s) }
func ResolveColorMode(mode ColorMode, isTerminal bool) bool {
	return types.ResolveColorMode(mode, isTerminal)
}
func NewColorConfig(mode ColorMode, trueColor bool) *ColorConfig {
	return types.NewColorConfig(mode, trueColor)
}
func GetTrueColorCode(r, g, b int) string { return format.GetTrueColorCode(r, g, b) }
func GetDetailedColorCode(diffType DiffType, useTrueColor bool) string {
	return format.GetDetailedColorCode(diffType, useTrueColor)
}
func GetContextColorCode(useTrueColor bool) string { return format.GetContextColorCode(useTrueColor) }
func GetColorReset() string                        { return format.GetColorReset() }
func CompactColor(dt DiffType) string              { return format.CompactColor(dt) }
func clamp(val, lo, hi int) int                    { return format.Clamp(val, lo, hi) }
