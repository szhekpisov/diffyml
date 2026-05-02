// color_custom.go - User-customizable color overrides.
//
// CustomColorPalette layers user-supplied hex colors on top of the built-in
// ANSI/true-color defaults defined in color.go. Includes hex parsing
// (ParseColor) and 8-color ANSI fallback (nearestANSI) for terminals
// without true-color support.
package diffyml

import (
	"fmt"
	"strconv"
	"strings"
)

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
// DefaultCustomColorPalette is constructed to mirror the built-in
// DetailedColorCode / ContextColorCode / DocNameColorCode values exactly,
// so for default roles we render straight from the palette's R/G/B/ANSICode
// fields. TestCustomColorPalette_ColorCode_Default pins this equivalence.
func (p *CustomColorPalette) ColorCode(role ColorRole, useTrueColor bool) string {
	c := p.colorForRole(role)
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
// Supports hex formats: #rrggbb and #rgb.
func ParseColor(s string) (*CustomColor, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty color specification")
	}

	if !strings.HasPrefix(s, "#") {
		return nil, fmt.Errorf("invalid color %q: use hex (#rrggbb or #rgb)", s)
	}

	hex := s[1:]
	var ri, gi, bi int
	switch len(hex) {
	case 3:
		val, err := strconv.ParseUint(string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]}), 16, 24)
		if err != nil {
			return nil, fmt.Errorf("invalid hex color %q", s)
		}
		ri, gi, bi = int(val>>16)&0xFF, int(val>>8)&0xFF, int(val)&0xFF
	case 6:
		val, err := strconv.ParseUint(hex, 16, 24)
		if err != nil {
			return nil, fmt.Errorf("invalid hex color %q", s)
		}
		ri, gi, bi = int(val>>16)&0xFF, int(val>>8)&0xFF, int(val)&0xFF
	default:
		return nil, fmt.Errorf("invalid hex color %q: expected #rrggbb or #rgb", s)
	}

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
	{"\033[34m", 0, 0, 238},
	{"\033[35m", 205, 0, 205},
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
