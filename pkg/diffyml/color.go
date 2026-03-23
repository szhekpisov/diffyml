// color.go - Terminal color detection and ANSI escape codes.
//
// Detects terminal capabilities (color, true color, width) from environment.
// Provides ANSI color codes for diff output highlighting.
// Key types: ColorMode, ColorConfig.
package diffyml

import (
	"fmt"
	"os"
	"strconv"
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

// ColorRole represents a semantic color role in diff output.
type ColorRole int

const (
	ColorRoleAdded ColorRole = iota
	ColorRoleRemoved
	ColorRoleModified
	ColorRoleContext
	ColorRoleDocName
)

// CustomColor holds a parsed custom color specification.
type CustomColor struct {
	R, G, B  int
	ANSICode string
	IsCustom bool
}

// CustomColorPalette holds user-customizable colors for each semantic role.
type CustomColorPalette struct {
	Added    *CustomColor
	Removed  *CustomColor
	Modified *CustomColor
	Context  *CustomColor
	DocName  *CustomColor
}

// DefaultCustomColorPalette returns a palette with the built-in default colors.
func DefaultCustomColorPalette() *CustomColorPalette {
	return &CustomColorPalette{
		Added:    &CustomColor{R: DetailedGreenR, G: DetailedGreenG, B: DetailedGreenB, ANSICode: colorGreen},
		Removed:  &CustomColor{R: DetailedRedR, G: DetailedRedG, B: DetailedRedB, ANSICode: colorRed},
		Modified: &CustomColor{R: DetailedYellowR, G: DetailedYellowG, B: DetailedYellowB, ANSICode: colorYellow},
		Context:  &CustomColor{R: DetailedGrayR, G: DetailedGrayG, B: DetailedGrayB, ANSICode: colorGray},
		DocName:  &CustomColor{R: DetailedDocNameR, G: DetailedDocNameG, B: DetailedDocNameB, ANSICode: colorCyan},
	}
}

var defaultPalette = DefaultCustomColorPalette()

func (p *CustomColorPalette) colorForRole(role ColorRole) *CustomColor {
	switch role {
	case ColorRoleAdded:
		return p.Added
	case ColorRoleRemoved:
		return p.Removed
	case ColorRoleModified:
		return p.Modified
	case ColorRoleContext:
		return p.Context
	case ColorRoleDocName:
		return p.DocName
	default:
		return p.Modified
	}
}

// ColorCode returns the ANSI escape string for a color role.
// When the color is a default (not custom), returns the existing hardcoded value.
func (p *CustomColorPalette) ColorCode(role ColorRole, useTrueColor bool) string {
	c := p.colorForRole(role)
	if !c.IsCustom {
		switch role {
		case ColorRoleAdded:
			return DetailedColorCode(DiffAdded, useTrueColor)
		case ColorRoleRemoved:
			return DetailedColorCode(DiffRemoved, useTrueColor)
		case ColorRoleModified:
			return DetailedColorCode(DiffModified, useTrueColor)
		case ColorRoleContext:
			return ContextColorCode(useTrueColor)
		case ColorRoleDocName:
			return DocNameColorCode(useTrueColor)
		}
	}
	if useTrueColor {
		return TrueColorCode(c.R, c.G, c.B)
	}
	return c.ANSICode
}

// EntryPalette returns the YAML color palette for rendering diff entry values.
// For default colors, returns the existing cached palettes (zero allocation).
// For custom colors, builds a flat palette using the custom color.
func (p *CustomColorPalette) EntryPalette(diffType DiffType, useTrueColor bool) *YAMLColorPalette {
	var c *CustomColor
	switch diffType {
	case DiffAdded:
		c = p.Added
	case DiffRemoved:
		c = p.Removed
	default:
		return entryPalette(diffType, useTrueColor)
	}

	if !c.IsCustom {
		return entryPalette(diffType, useTrueColor)
	}

	if useTrueColor {
		code := TrueColorCode(c.R, c.G, c.B)
		return &YAMLColorPalette{
			Key:            styleBold + code,
			Scalar:         code,
			MultilineText:  code,
			Null:           code,
			EmptyStructure: code,
		}
	}
	return flatPalette(c.ANSICode)
}

// resolvedPalette returns the custom palette from opts, or the default palette if nil.
func resolvedPalette(opts *FormatOptions) *CustomColorPalette {
	if opts != nil && opts.Palette != nil {
		return opts.Palette
	}
	return defaultPalette
}

// ParseColor parses a color specification string.
// Supports hex (#rrggbb, #rgb) and named ANSI colors (red, green, yellow, cyan, gray, white).
func ParseColor(s string) (*CustomColor, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty color specification")
	}

	switch strings.ToLower(s) {
	case "red":
		return &CustomColor{R: 205, G: 0, B: 0, ANSICode: colorRed, IsCustom: true}, nil
	case "green":
		return &CustomColor{R: 0, G: 205, B: 0, ANSICode: colorGreen, IsCustom: true}, nil
	case "yellow":
		return &CustomColor{R: 205, G: 205, B: 0, ANSICode: colorYellow, IsCustom: true}, nil
	case "cyan":
		return &CustomColor{R: 0, G: 205, B: 205, ANSICode: colorCyan, IsCustom: true}, nil
	case "gray", "grey":
		return &CustomColor{R: 127, G: 127, B: 127, ANSICode: colorGray, IsCustom: true}, nil
	case "white":
		return &CustomColor{R: 229, G: 229, B: 229, ANSICode: colorWhite, IsCustom: true}, nil
	}

	if !strings.HasPrefix(s, "#") {
		return nil, fmt.Errorf("invalid color %q: use hex (#rrggbb, #rgb) or named (red, green, yellow, cyan, gray, white)", s)
	}

	hex := s[1:]
	var r, g, b uint64
	var err error
	switch len(hex) {
	case 3:
		r, err = strconv.ParseUint(string([]byte{hex[0], hex[0]}), 16, 8)
		if err != nil {
			return nil, fmt.Errorf("invalid hex color %q", s)
		}
		g, err = strconv.ParseUint(string([]byte{hex[1], hex[1]}), 16, 8)
		if err != nil {
			return nil, fmt.Errorf("invalid hex color %q", s)
		}
		b, err = strconv.ParseUint(string([]byte{hex[2], hex[2]}), 16, 8)
		if err != nil {
			return nil, fmt.Errorf("invalid hex color %q", s)
		}
	case 6:
		r, err = strconv.ParseUint(hex[0:2], 16, 8)
		if err != nil {
			return nil, fmt.Errorf("invalid hex color %q", s)
		}
		g, err = strconv.ParseUint(hex[2:4], 16, 8)
		if err != nil {
			return nil, fmt.Errorf("invalid hex color %q", s)
		}
		b, err = strconv.ParseUint(hex[4:6], 16, 8)
		if err != nil {
			return nil, fmt.Errorf("invalid hex color %q", s)
		}
	default:
		return nil, fmt.Errorf("invalid hex color %q: expected #rrggbb or #rgb", s)
	}

	ri, gi, bi := int(r), int(g), int(b)
	return &CustomColor{
		R: ri, G: gi, B: bi,
		ANSICode: nearestANSI(ri, gi, bi),
		IsCustom: true,
	}, nil
}

// ansiColorRef maps an ANSI code to approximate RGB values for nearest-match.
type ansiColorRef struct {
	code    string
	r, g, b int
}

var ansiColorRefs = []ansiColorRef{
	{colorRed, 205, 0, 0},
	{colorGreen, 0, 205, 0},
	{colorYellow, 205, 205, 0},
	{colorCyan, 0, 205, 205},
	{colorWhite, 229, 229, 229},
	{colorGray, 127, 127, 127},
}

// nearestANSI returns the nearest 8-color ANSI code for an RGB value
// using squared Euclidean distance.
func nearestANSI(r, g, b int) string {
	best := ansiColorRefs[0].code
	bestDist := colorDistSq(r, g, b, ansiColorRefs[0])
	for _, c := range ansiColorRefs[1:] {
		if dist := colorDistSq(r, g, b, c); dist < bestDist {
			bestDist = dist
			best = c.code
		}
	}
	return best
}

func colorDistSq(r, g, b int, c ansiColorRef) int {
	dr := r - c.r
	dg := g - c.g
	db := b - c.b
	return dr*dr + dg*dg + db*db
}
