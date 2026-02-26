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

// Default terminal width when auto-detection is not possible.
const defaultTerminalWidth = 80

// Minimum terminal width to enforce.
const minTerminalWidth = 40

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

// GetTerminalWidth returns the terminal width to use.
// If override is positive, it returns that value (with minimum bound).
// If override is 0, it returns a default width.
func GetTerminalWidth(override int) int {
	if override > 0 {
		if override < minTerminalWidth {
			return minTerminalWidth
		}
		return override
	}
	return defaultTerminalWidth
}

// IsTerminal checks if the given file descriptor is a terminal.
func IsTerminal(fd uintptr) bool {
	stat, err := os.Stdout.Stat()
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
	width      int
	isTerminal bool
}

// NewColorConfig creates a new color configuration.
func NewColorConfig(mode ColorMode, trueColor bool, width int) *ColorConfig {
	return &ColorConfig{
		mode:       mode,
		trueColor:  trueColor,
		width:      width,
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
// Only returns true if both truecolor is requested and terminal supports it.
func (c *ColorConfig) ShouldUseTrueColor() bool {
	if !c.trueColor {
		return false
	}
	// Check for true color support via COLORTERM environment variable
	colorTerm := os.Getenv("COLORTERM")
	if colorTerm == "truecolor" || colorTerm == "24bit" {
		return true
	}
	// Also check TERM for common true color terminals
	term := os.Getenv("TERM")
	if strings.Contains(term, "256color") || strings.Contains(term, "truecolor") {
		return true
	}
	// Default to assuming support if explicitly requested
	return c.trueColor && c.isTerminal
}

// GetWidth returns the terminal width to use.
func (c *ColorConfig) GetWidth() int {
	return GetTerminalWidth(c.width)
}

// ToFormatOptions applies the color config to FormatOptions.
func (c *ColorConfig) ToFormatOptions(opts *FormatOptions) {
	if opts == nil {
		return
	}
	opts.Color = c.ShouldUseColor()
	opts.TrueColor = c.ShouldUseTrueColor()
	if c.width > 0 {
		opts.Width = c.width
	}
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

// GetTrueColorCode returns an ANSI escape sequence for 24-bit RGB color.
// RGB values are clamped to the valid range [0, 255].
func GetTrueColorCode(r, g, b int) string {
	r = clamp(r, 0, 255)
	g = clamp(g, 0, 255)
	b = clamp(b, 0, 255)
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
}

// GetDetailedColorCode returns the appropriate color code for a diff type.
// Uses the detailed palette when useTrueColor is true,
// otherwise falls back to standard 8-color ANSI codes.
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

// GetContextColorCode returns gray color for context lines.
// Uses the detailed gray RGB value when useTrueColor is true,
// otherwise uses bright black (gray) ANSI code.
func GetContextColorCode(useTrueColor bool) string {
	if useTrueColor {
		return GetTrueColorCode(DetailedGrayR, DetailedGrayG, DetailedGrayB)
	}
	return colorGray
}

// GetColorReset returns the ANSI reset code to clear all formatting.
func GetColorReset() string {
	return colorReset
}

// clamp restricts a value to the range [min, max].
func clamp(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}
