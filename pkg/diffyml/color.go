// color.go - Terminal color detection and ANSI escape codes.
package diffyml

import (
	"os"

	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/types"
)

// Type aliases
type ColorMode = types.ColorMode
type ColorConfig = types.ColorConfig
type Colorizer = types.Colorizer

// Constants
const (
	ColorModeAuto   = types.ColorModeAuto
	ColorModeAlways = types.ColorModeAlways
	ColorModeNever  = types.ColorModeNever
)

// ANSI color codes (unexported in facade, exported in types)
const (
	colorReset  = types.ColorReset
	colorRed    = types.ColorRed
	colorGreen  = types.ColorGreen
	colorYellow = types.ColorYellow
	colorCyan   = types.ColorCyan
	colorWhite  = types.ColorWhite
	colorGray   = types.ColorGray
)

// ANSI style codes
const (
	styleBold      = types.StyleBold
	styleBoldOff   = types.StyleBoldOff
	styleItalic    = types.StyleItalic
	styleItalicOff = types.StyleItalicOff
)

// Detailed color palette constants
const (
	DetailedYellowR = types.DetailedYellowR
	DetailedYellowG = types.DetailedYellowG
	DetailedYellowB = types.DetailedYellowB
	DetailedRedR    = types.DetailedRedR
	DetailedRedG    = types.DetailedRedG
	DetailedRedB    = types.DetailedRedB
	DetailedGreenR  = types.DetailedGreenR
	DetailedGreenG  = types.DetailedGreenG
	DetailedGreenB  = types.DetailedGreenB
	DetailedGrayR   = types.DetailedGrayR
	DetailedGrayG   = types.DetailedGrayG
	DetailedGrayB   = types.DetailedGrayB
)

// stdoutStatFn is injectable for testing. Must be a local var so tests in
// package diffyml can reassign it. IsTerminal below reads this var.
var stdoutStatFn = func() (os.FileInfo, error) {
	return os.Stdout.Stat()
}

// IsTerminal checks if the given file descriptor is a terminal.
// Uses the local stdoutStatFn so that tests can override it.
func IsTerminal(fd uintptr) bool {
	stat, err := stdoutStatFn()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

func ParseColorMode(s string) (ColorMode, error)                               { return types.ParseColorMode(s) }
func ResolveColorMode(mode ColorMode, isTerminal bool) bool                     { return types.ResolveColorMode(mode, isTerminal) }
func NewColorConfig(mode ColorMode, trueColor bool) *ColorConfig                { return types.NewColorConfig(mode, trueColor) }
func GetTrueColorCode(r, g, b int) string                                       { return types.GetTrueColorCode(r, g, b) }
func GetDetailedColorCode(diffType DiffType, useTrueColor bool) string          { return types.GetDetailedColorCode(diffType, useTrueColor) }
func GetContextColorCode(useTrueColor bool) string                              { return types.GetContextColorCode(useTrueColor) }
func GetColorReset() string                                                     { return types.GetColorReset() }
func CompactColor(dt DiffType) string                                           { return types.CompactColor(dt) }
func clamp(val, lo, hi int) int                                                 { return types.Clamp(val, lo, hi) }
