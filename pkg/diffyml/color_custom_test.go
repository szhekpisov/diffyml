package diffyml

import (
	"testing"
)

func TestParseColor_Hex6(t *testing.T) {
	c, err := ParseColor("#6aa3a5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.R != 106 || c.G != 163 || c.B != 165 {
		t.Errorf("got RGB(%d,%d,%d), want (106,163,165)", c.R, c.G, c.B)
	}
	if !c.IsCustom {
		t.Error("expected IsCustom=true")
	}
	if c.ANSICode == "" {
		t.Error("expected non-empty ANSICode")
	}
}

func TestParseColor_Hex3(t *testing.T) {
	c, err := ParseColor("#f00")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.R != 255 || c.G != 0 || c.B != 0 {
		t.Errorf("got RGB(%d,%d,%d), want (255,0,0)", c.R, c.G, c.B)
	}
	if !c.IsCustom {
		t.Error("expected IsCustom=true")
	}
}

func TestParseColor_Invalid(t *testing.T) {
	tests := []string{
		"",
		"red",
		"blue",
		"#gg0000",
		"#00gg00",
		"#0000gg",
		"#12345",
		"#1234567",
		"invalid",
		"#xyz",
		"#0x0",
		"#00x",
	}
	for _, s := range tests {
		t.Run(s, func(t *testing.T) {
			_, err := ParseColor(s)
			if err == nil {
				t.Errorf("expected error for %q", s)
			}
		})
	}
}

func TestParseColor_HexCaseInsensitive(t *testing.T) {
	c, err := ParseColor("#FF00FF")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.R != 255 || c.G != 0 || c.B != 255 {
		t.Errorf("got RGB(%d,%d,%d), want (255,0,255)", c.R, c.G, c.B)
	}
}

func TestParseColor_Whitespace(t *testing.T) {
	c, err := ParseColor("  #ff0000  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.R != 255 || c.G != 0 || c.B != 0 {
		t.Errorf("got RGB(%d,%d,%d), want (255,0,0)", c.R, c.G, c.B)
	}
}

func TestNearestANSI(t *testing.T) {
	tests := []struct {
		r, g, b int
		want    string
	}{
		{255, 0, 0, colorRed},
		{0, 255, 0, colorGreen},
		{200, 200, 0, colorYellow},
		{0, 200, 200, colorCyan},
		{100, 100, 100, colorGray},
		{240, 240, 240, colorWhite},
		// Blue and magenta
		{0, 0, 200, "\033[34m"},
		{200, 0, 200, "\033[35m"},
		// Teal (#6aa3a5) is closer to gray in RGB distance
		{106, 163, 165, colorGray},
	}
	for _, tt := range tests {
		got := nearestANSI(tt.r, tt.g, tt.b)
		if got != tt.want {
			t.Errorf("nearestANSI(%d,%d,%d) = %q, want %q", tt.r, tt.g, tt.b, got, tt.want)
		}
	}
}

func TestCustomColorPalette_ColorCode_Default(t *testing.T) {
	p := DefaultCustomColorPalette()

	// Default palette should produce same results as existing functions
	if got, want := p.ColorCode(ColorRoleAdded, false), DetailedColorCode(DiffAdded, false); got != want {
		t.Errorf("Added 8-color: got %q, want %q", got, want)
	}
	if got, want := p.ColorCode(ColorRoleAdded, true), DetailedColorCode(DiffAdded, true); got != want {
		t.Errorf("Added TrueColor: got %q, want %q", got, want)
	}
	if got, want := p.ColorCode(ColorRoleRemoved, false), DetailedColorCode(DiffRemoved, false); got != want {
		t.Errorf("Removed 8-color: got %q, want %q", got, want)
	}
	if got, want := p.ColorCode(ColorRoleModified, true), DetailedColorCode(DiffModified, true); got != want {
		t.Errorf("Modified TrueColor: got %q, want %q", got, want)
	}
	if got, want := p.ColorCode(ColorRoleContext, false), ContextColorCode(false); got != want {
		t.Errorf("Context 8-color: got %q, want %q", got, want)
	}
	if got, want := p.ColorCode(ColorRoleDocName, true), DocNameColorCode(true); got != want {
		t.Errorf("DocName TrueColor: got %q, want %q", got, want)
	}
}

func TestCustomColorPalette_ColorCode_Custom(t *testing.T) {
	p := DefaultCustomColorPalette()
	p.Added = &CustomColor{R: 106, G: 163, B: 165, ANSICode: colorCyan, IsCustom: true}

	// Custom color in TrueColor mode
	got := p.ColorCode(ColorRoleAdded, true)
	want := TrueColorCode(106, 163, 165)
	if got != want {
		t.Errorf("Custom Added TrueColor: got %q, want %q", got, want)
	}

	// Custom color in 8-color mode
	got = p.ColorCode(ColorRoleAdded, false)
	if got != colorCyan {
		t.Errorf("Custom Added 8-color: got %q, want %q", got, colorCyan)
	}

	// Non-custom role still uses defaults
	got = p.ColorCode(ColorRoleRemoved, false)
	want = DetailedColorCode(DiffRemoved, false)
	if got != want {
		t.Errorf("Default Removed: got %q, want %q", got, want)
	}
}

func TestCustomColorPalette_EntryPalette_Default(t *testing.T) {
	p := DefaultCustomColorPalette()

	// Default palette should return cached palettes
	got := p.EntryPalette(DiffAdded, true)
	want := entryPalette(DiffAdded, true)
	if got != want {
		t.Error("Default Added TrueColor should return cached palette")
	}

	got = p.EntryPalette(DiffRemoved, false)
	want = entryPalette(DiffRemoved, false)
	if got != want {
		t.Error("Default Removed 8-color should return cached palette")
	}
}

func TestCustomColorPalette_EntryPalette_Custom(t *testing.T) {
	p := DefaultCustomColorPalette()
	p.Added = &CustomColor{R: 106, G: 163, B: 165, ANSICode: colorCyan, IsCustom: true}

	// Custom TrueColor palette should be flat with bold Key
	palette := p.EntryPalette(DiffAdded, true)
	expectedCode := TrueColorCode(106, 163, 165)
	if palette.Scalar != expectedCode {
		t.Errorf("Custom palette Scalar: got %q, want %q", palette.Scalar, expectedCode)
	}
	if palette.Key != styleBold+expectedCode {
		t.Errorf("Custom palette Key: got %q, want %q", palette.Key, styleBold+expectedCode)
	}
	if palette.MultilineText != expectedCode {
		t.Errorf("Custom palette MultilineText: got %q, want %q", palette.MultilineText, expectedCode)
	}
	if palette.Null != expectedCode {
		t.Errorf("Custom palette Null: got %q, want %q", palette.Null, expectedCode)
	}
	if palette.EmptyStructure != expectedCode {
		t.Errorf("Custom palette EmptyStructure: got %q, want %q", palette.EmptyStructure, expectedCode)
	}

	// Custom 8-color palette should be flat
	palette = p.EntryPalette(DiffAdded, false)
	if palette.Scalar != colorCyan {
		t.Errorf("Custom 8-color Scalar: got %q, want %q", palette.Scalar, colorCyan)
	}
	if palette.Key != colorCyan {
		t.Errorf("Custom 8-color Key: got %q, want %q", palette.Key, colorCyan)
	}
}

func TestCustomColorPalette_EntryPalette_CustomRemoved(t *testing.T) {
	p := DefaultCustomColorPalette()
	p.Removed = &CustomColor{R: 112, G: 45, B: 6, ANSICode: colorRed, IsCustom: true}

	// Custom Removed TrueColor
	palette := p.EntryPalette(DiffRemoved, true)
	expectedCode := TrueColorCode(112, 45, 6)
	if palette.Scalar != expectedCode {
		t.Errorf("Custom Removed TrueColor Scalar: got %q, want %q", palette.Scalar, expectedCode)
	}

	// Custom Removed 8-color
	palette = p.EntryPalette(DiffRemoved, false)
	if palette.Scalar != colorRed {
		t.Errorf("Custom Removed 8-color Scalar: got %q, want %q", palette.Scalar, colorRed)
	}
}

func TestCustomColorPalette_ColorForRole_Default(t *testing.T) {
	p := DefaultCustomColorPalette()
	// Invalid role should fall back to Modified
	got := p.ColorCode(ColorRole(99), false)
	want := p.ColorCode(ColorRoleModified, false)
	if got != want {
		t.Errorf("invalid role: got %q, want %q (Modified default)", got, want)
	}
}

func TestCustomColorPalette_EntryPalette_NeutralDefault(t *testing.T) {
	p := DefaultCustomColorPalette()

	// Neutral (non-added, non-removed) should use cached palettes
	got := p.EntryPalette(DiffModified, true)
	want := entryPalette(DiffModified, true)
	if got != want {
		t.Error("Neutral TrueColor should return cached palette")
	}
}

func TestResolvedPalette_Nil(t *testing.T) {
	got := resolvedPalette(nil)
	if got != defaultPalette {
		t.Error("nil opts should return defaultPalette")
	}

	opts := &FormatOptions{}
	got = resolvedPalette(opts)
	if got != defaultPalette {
		t.Error("opts with nil Palette should return defaultPalette")
	}
}

func TestResolvedPalette_Custom(t *testing.T) {
	custom := DefaultCustomColorPalette()
	custom.Added = &CustomColor{R: 1, G: 2, B: 3, ANSICode: colorRed, IsCustom: true}

	opts := &FormatOptions{Palette: custom}
	got := resolvedPalette(opts)
	if got != custom {
		t.Error("opts with custom Palette should return it")
	}
}
