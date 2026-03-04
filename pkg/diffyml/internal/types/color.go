package types

import (
	"fmt"
	"os"
	"strings"
)

// ColorMode represents the color output mode.
type ColorMode int

const (
	ColorModeAuto   ColorMode = iota
	ColorModeAlways
	ColorModeNever
)

// ParseColorMode parses a color mode string.
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

// ResolveColorMode determines if color should be enabled.
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

// StdoutStatFn is an injectable function for os.Stdout.Stat().
var StdoutStatFn = func() (os.FileInfo, error) {
	return os.Stdout.Stat()
}

// IsTerminal checks if the given file descriptor is a terminal.
func IsTerminal(fd uintptr) bool {
	stat, err := StdoutStatFn()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// ColorConfig holds color and terminal configuration.
type ColorConfig struct {
	Mode       ColorMode
	TrueColor  bool
	IsTerminal bool
}

// NewColorConfig creates a new color configuration.
func NewColorConfig(mode ColorMode, trueColor bool) *ColorConfig {
	return &ColorConfig{
		Mode:       mode,
		TrueColor:  trueColor,
		IsTerminal: false,
	}
}

// SetIsTerminal sets whether output is to a terminal.
func (c *ColorConfig) SetIsTerminal(isTerminal bool) {
	c.IsTerminal = isTerminal
}

// DetectTerminal automatically detects if stdout is a terminal.
func (c *ColorConfig) DetectTerminal() {
	c.IsTerminal = IsTerminal(os.Stdout.Fd())
}

// ShouldUseColor returns whether color output should be used.
func (c *ColorConfig) ShouldUseColor() bool {
	return ResolveColorMode(c.Mode, c.IsTerminal)
}

// ShouldUseTrueColor returns whether 24-bit true color should be used.
func (c *ColorConfig) ShouldUseTrueColor() bool {
	return c.TrueColor
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
	DetailedYellowR, DetailedYellowG, DetailedYellowB = 199, 196, 63
	DetailedRedR, DetailedRedG, DetailedRedB           = 185, 49, 27
	DetailedGreenR, DetailedGreenG, DetailedGreenB     = 88, 191, 56
	DetailedGrayR, DetailedGrayG, DetailedGrayB        = 105, 105, 105
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
func GetDetailedColorCode(diffType DiffType, useTrueColor bool) string {
	if useTrueColor {
		switch diffType {
		case DiffAdded:
			return GetTrueColorCode(DetailedGreenR, DetailedGreenG, DetailedGreenB)
		case DiffRemoved:
			return GetTrueColorCode(DetailedRedR, DetailedRedG, DetailedRedB)
		case DiffModified, DiffOrderChanged:
			return GetTrueColorCode(DetailedYellowR, DetailedYellowG, DetailedYellowB)
		}
	}
	switch diffType {
	case DiffAdded:
		return ColorGreen
	case DiffRemoved:
		return ColorRed
	case DiffModified, DiffOrderChanged:
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

func (c Colorizer) Added() string    { return GetDetailedColorCode(DiffAdded, c.TrueColor) }
func (c Colorizer) Removed() string  { return GetDetailedColorCode(DiffRemoved, c.TrueColor) }
func (c Colorizer) Modified() string { return GetDetailedColorCode(DiffModified, c.TrueColor) }
func (c Colorizer) Context() string  { return GetContextColorCode(c.TrueColor) }
func (c Colorizer) Reset() string    { return ColorReset }
func (c Colorizer) Bold() string     { return StyleBold }

// CompactColor returns the 8-color ANSI code for a diff type.
func CompactColor(dt DiffType) string {
	switch dt {
	case DiffAdded:
		return ColorGreen
	case DiffRemoved:
		return ColorRed
	case DiffModified, DiffOrderChanged:
		return ColorYellow
	default:
		return ""
	}
}
