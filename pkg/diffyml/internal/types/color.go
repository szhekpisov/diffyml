// color.go - Color mode configuration and terminal detection.
//
// Presentation-layer color constants (ANSI codes, Colorizer, etc.)
// live in internal/format/color.go.
package types

import (
	"fmt"
	"os"
	"strings"
)

// ColorMode represents the color output mode.
type ColorMode int

const (
	ColorModeAuto ColorMode = iota
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
