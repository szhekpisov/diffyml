// color.go - Terminal color detection and ANSI escape codes.
//
// Detects terminal capabilities (color, true color, width) from environment.
// Provides ANSI color codes for diff output highlighting.
// Key types: ColorMode, ColorConfig.
package diffyml

import (
	"fmt"
	"os"
	"strings"
)

// ColorMode represents the color output mode.
type ColorMode int

const (
	// ColorModeAuto automatically detects terminal capability.
	ColorModeAuto ColorMode = iota
	// ColorModeAlways always enables color output.
	ColorModeAlways
	// ColorModeNever always disables color output.
	ColorModeNever
)

// ParseColorMode parses a color mode string (always, never, auto).
// Empty string defaults to auto.
func ParseColorMode(s string) (ColorMode, error) {
	switch strings.ToLower(s) {
	case "", "auto":
		return ColorModeAuto, nil
	case "always":
		return ColorModeAlways, nil
	case "never":
		return ColorModeNever, nil
	default:
		return ColorModeAuto, fmt.Errorf("invalid color mode %q, valid modes: always, never, auto", s)
	}
}

// ResolveColorMode determines if color should be enabled based on mode and terminal state.
func ResolveColorMode(mode ColorMode, isTerminal bool) bool {
	switch mode {
	case ColorModeAlways:
		return true
	case ColorModeNever:
		return false
	case ColorModeAuto:
		return isTerminal
	default:
		return isTerminal
	}
}

// stdoutStatFn is an injectable function for os.Stdout.Stat(), enabling
// terminal-mode mocking in tests without a real TTY.
var stdoutStatFn = func() (os.FileInfo, error) {
	return os.Stdout.Stat()
}

// IsTerminal checks if the given file descriptor is a terminal.
func IsTerminal(fd uintptr) bool {
	stat, err := stdoutStatFn()
	if err != nil {
		return false
	}
	// Check if it's a character device (terminal)
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// ColorConfig holds color and terminal configuration.
type ColorConfig struct {
	mode       ColorMode
	trueColor  bool
	isTerminal bool
}

// NewColorConfig creates a new color configuration.
func NewColorConfig(mode ColorMode, trueColor bool) *ColorConfig {
	return &ColorConfig{
		mode:       mode,
		trueColor:  trueColor,
		isTerminal: false, // Default, can be set via SetIsTerminal
	}
}

// SetIsTerminal sets whether output is to a terminal.
func (c *ColorConfig) SetIsTerminal(isTerminal bool) {
	c.isTerminal = isTerminal
}

// DetectTerminal automatically detects if stdout is a terminal.
func (c *ColorConfig) DetectTerminal() {
	c.isTerminal = IsTerminal(os.Stdout.Fd())
}

// ShouldUseColor returns whether color output should be used.
func (c *ColorConfig) ShouldUseColor() bool {
	return ResolveColorMode(c.mode, c.isTerminal)
}

// ShouldUseTrueColor returns whether 24-bit true color should be used.
// trueColor is only set for "always" mode, so the explicit request is honored.
func (c *ColorConfig) ShouldUseTrueColor() bool {
	return c.trueColor
}

// ToFormatOptions applies the color config to FormatOptions.
func (c *ColorConfig) ToFormatOptions(opts *FormatOptions) {
	if opts == nil {
		return
	}
	opts.Color = c.ShouldUseColor()
	opts.TrueColor = c.ShouldUseTrueColor()
}

// Detailed color palette constants (24-bit RGB values)
const (
	// DetailedYellowR, DetailedYellowG, DetailedYellowB - Yellow for change indicators
	DetailedYellowR, DetailedYellowG, DetailedYellowB = 199, 196, 63
	// DetailedRedR, DetailedRedG, DetailedRedB - Red for removed values
	DetailedRedR, DetailedRedG, DetailedRedB = 185, 49, 27
	// DetailedGreenR, DetailedGreenG, DetailedGreenB - Green for added values
	DetailedGreenR, DetailedGreenG, DetailedGreenB = 88, 191, 56
	// DetailedGrayR, DetailedGrayG, DetailedGrayB - Gray for context lines
	DetailedGrayR, DetailedGrayG, DetailedGrayB = 105, 105, 105
	// DetailedDocNameR, DetailedDocNameG, DetailedDocNameB - Light steel blue for document identifiers
	DetailedDocNameR, DetailedDocNameG, DetailedDocNameB = 176, 196, 222
)

// ANSI color codes (8-color fallback)
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
	colorGray   = "\033[90m" // Bright black
)

// ANSI style codes (bold, italic)
const (
	styleBold      = "\033[1m"
	styleBoldOff   = "\033[22m"
	styleItalic    = "\033[3m"
	styleItalicOff = "\033[23m"
)

// TrueColorCode returns an ANSI escape sequence for 24-bit RGB color.
// RGB values are clamped to the valid range [0, 255].
func TrueColorCode(r, g, b int) string {
	r = clamp(r, 0, 255)
	g = clamp(g, 0, 255)
	b = clamp(b, 0, 255)
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
}

// DetailedColorCode returns the appropriate color code for a diff type.
// Uses the detailed palette when useTrueColor is true,
// otherwise falls back to standard 8-color ANSI codes.
func DetailedColorCode(diffType DiffType, useTrueColor bool) string {
	if useTrueColor {
		switch diffType {
		case DiffAdded:
			return TrueColorCode(DetailedGreenR, DetailedGreenG, DetailedGreenB)
		case DiffRemoved:
			return TrueColorCode(DetailedRedR, DetailedRedG, DetailedRedB)
		case DiffModified, DiffOrderChanged:
			return TrueColorCode(DetailedYellowR, DetailedYellowG, DetailedYellowB)
		}
	}
	// Fallback to 8-color ANSI
	switch diffType {
	case DiffAdded:
		return colorGreen
	case DiffRemoved:
		return colorRed
	case DiffModified, DiffOrderChanged:
		return colorYellow
	}
	return ""
}

// DocNameColorCode returns the color code for document identifier labels.
// Uses light steel blue when useTrueColor is true,
// otherwise falls back to cyan ANSI code.
func DocNameColorCode(useTrueColor bool) string {
	if useTrueColor {
		return TrueColorCode(DetailedDocNameR, DetailedDocNameG, DetailedDocNameB)
	}
	return colorCyan
}

// ContextColorCode returns gray color for context lines.
// Uses the detailed gray RGB value when useTrueColor is true,
// otherwise uses bright black (gray) ANSI code.
func ContextColorCode(useTrueColor bool) string {
	if useTrueColor {
		return TrueColorCode(DetailedGrayR, DetailedGrayG, DetailedGrayB)
	}
	return colorGray
}

// ColorReset returns the ANSI reset code to clear all formatting.
func ColorReset() string {
	return colorReset
}

// colorStart returns the ANSI color code if color is enabled, empty string otherwise.
func colorStart(opts *FormatOptions, code string) string {
	if opts.Color {
		return code
	}
	return ""
}

// colorEnd returns the ANSI reset code if color is enabled, empty string otherwise.
func colorEnd(opts *FormatOptions) string {
	if opts.Color {
		return colorReset
	}
	return ""
}

// YAMLColorPalette holds per-element color codes for YAML syntax highlighting.
// Each field is a pre-computed ANSI escape string. In TrueColor mode, fields
// have distinct shades; in 8-color mode, all fields share the same ANSI code.
type YAMLColorPalette struct {
	Key            string // Map keys, key-only lines, and structured list "- " prefixes
	Scalar         string // String, bool, int, float values
	MultilineText  string // Block literal content lines
	Null           string // nil/null values
	EmptyStructure string // Empty maps {} and empty lists []
}

// ScalarColor returns the color for a scalar value, using Null for nil values.
func (p *YAMLColorPalette) ScalarColor(val any) string {
	if val == nil {
		return p.Null
	}
	return p.Scalar
}

// Cached palettes (computed once). TrueColor palettes use per-element shading;
// flat palettes use a single ANSI code for all elements.
var (
	cachedGreenPalette = &YAMLColorPalette{
		Key:            styleBold + TrueColorCode(50, 170, 100),
		Scalar:         TrueColorCode(130, 230, 100),
		MultilineText:  TrueColorCode(140, 200, 95),
		Null:           TrueColorCode(100, 155, 115),
		EmptyStructure: TrueColorCode(80, 135, 95),
	}
	cachedRedPalette = &YAMLColorPalette{
		Key:            styleBold + TrueColorCode(210, 80, 70),
		Scalar:         TrueColorCode(245, 140, 110),
		MultilineText:  TrueColorCode(230, 155, 120),
		Null:           TrueColorCode(195, 140, 130),
		EmptyStructure: TrueColorCode(170, 120, 110),
	}
	cachedFlatGreen = flatPalette(colorGreen)
	cachedFlatRed   = flatPalette(colorRed)
	// Neutral palette for unexpected DiffTypes — renders as white so it's
	// visually distinct from additions (green) and removals (red).
	cachedNeutralPalette = &YAMLColorPalette{
		Key:            styleBold + TrueColorCode(200, 200, 200),
		Scalar:         TrueColorCode(220, 220, 220),
		MultilineText:  TrueColorCode(210, 210, 210),
		Null:           TrueColorCode(180, 180, 180),
		EmptyStructure: TrueColorCode(170, 170, 170),
	}
	cachedFlatNeutral = flatPalette(colorWhite)
)

func flatPalette(code string) *YAMLColorPalette {
	return &YAMLColorPalette{
		Key: code, Scalar: code, MultilineText: code,
		Null: code, EmptyStructure: code,
	}
}

// entryPalette returns the color palette for rendering diff entries.
// Always returns a non-nil palette: TrueColor mode gets per-element shading,
// 8-color mode gets a flat palette where all fields share the same ANSI code.
func entryPalette(diffType DiffType, useTrueColor bool) *YAMLColorPalette {
	switch diffType {
	case DiffAdded:
		if useTrueColor {
			return cachedGreenPalette
		}
		return cachedFlatGreen
	case DiffRemoved:
		if useTrueColor {
			return cachedRedPalette
		}
		return cachedFlatRed
	default:
		// Neutral palette — currently unreachable in production (DiffModified and
		// DiffOrderChanged route through formatChangeDescriptor, not renderEntryValue).
		// Kept as a safety net for future DiffTypes.
		if useTrueColor {
			return cachedNeutralPalette
		}
		return cachedFlatNeutral
	}
}

// DetectTrueColorSupport checks if the terminal supports 24-bit color
// via the COLORTERM environment variable (standard detection method).
func DetectTrueColorSupport() bool {
	ct := os.Getenv("COLORTERM")
	return ct == "truecolor" || ct == "24bit"
}

// clamp restricts a value to the range [lo, hi].
func clamp(val, lo, hi int) int {
	return max(lo, min(val, hi))
}
