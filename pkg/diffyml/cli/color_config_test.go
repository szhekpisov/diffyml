package cli

import (
	"testing"
)

func TestLoadColorPalette_NilConfig(t *testing.T) {
	palette, err := loadColorPalette(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if palette != nil {
		t.Error("expected nil palette when no colors are set")
	}
}

func TestLoadColorPalette_EmptyColors(t *testing.T) {
	fc := &FileConfig{Colors: &ColorOverrides{}}
	palette, err := loadColorPalette(fc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if palette != nil {
		t.Error("expected nil palette when all color overrides are nil")
	}
}

func TestLoadColorPalette_ConfigFile(t *testing.T) {
	added := "#6aa3a5"
	removed := "red"
	fc := &FileConfig{
		Colors: &ColorOverrides{
			Added:   &added,
			Removed: &removed,
		},
	}

	palette, err := loadColorPalette(fc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if palette == nil {
		t.Fatal("expected non-nil palette")
	}
	if !palette.Added.IsCustom {
		t.Error("expected Added.IsCustom=true")
	}
	if palette.Added.R != 106 || palette.Added.G != 163 || palette.Added.B != 165 {
		t.Errorf("Added: got RGB(%d,%d,%d), want (106,163,165)", palette.Added.R, palette.Added.G, palette.Added.B)
	}
	if !palette.Removed.IsCustom {
		t.Error("expected Removed.IsCustom=true")
	}
	if palette.Removed.R != 205 {
		t.Errorf("Removed.R: got %d, want 205", palette.Removed.R)
	}
	// Modified should remain default
	if palette.Modified.IsCustom {
		t.Error("expected Modified.IsCustom=false (not overridden)")
	}
}

func TestLoadColorPalette_EnvVarOverridesConfig(t *testing.T) {
	cfgAdded := "#000000"
	fc := &FileConfig{
		Colors: &ColorOverrides{
			Added: &cfgAdded,
		},
	}

	t.Setenv("DIFFYML_COLOR_ADDED", "#ffffff")

	palette, err := loadColorPalette(fc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if palette == nil {
		t.Fatal("expected non-nil palette")
	}
	// Env var should win
	if palette.Added.R != 255 || palette.Added.G != 255 || palette.Added.B != 255 {
		t.Errorf("Added: got RGB(%d,%d,%d), want (255,255,255)", palette.Added.R, palette.Added.G, palette.Added.B)
	}
}

func TestLoadColorPalette_EnvVarOnly(t *testing.T) {
	t.Setenv("DIFFYML_COLOR_REMOVED", "#702d06")

	palette, err := loadColorPalette(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if palette == nil {
		t.Fatal("expected non-nil palette")
	}
	if palette.Removed.R != 112 || palette.Removed.G != 45 || palette.Removed.B != 6 {
		t.Errorf("Removed: got RGB(%d,%d,%d), want (112,45,6)", palette.Removed.R, palette.Removed.G, palette.Removed.B)
	}
	// Others remain default
	if palette.Added.IsCustom {
		t.Error("expected Added to remain default")
	}
}

func TestLoadColorPalette_AllEnvVars(t *testing.T) {
	t.Setenv("DIFFYML_COLOR_ADDED", "#11ff11")
	t.Setenv("DIFFYML_COLOR_REMOVED", "#ff1111")
	t.Setenv("DIFFYML_COLOR_MODIFIED", "#ffff11")
	t.Setenv("DIFFYML_COLOR_CONTEXT", "#888888")
	t.Setenv("DIFFYML_COLOR_DOC_NAME", "cyan")

	palette, err := loadColorPalette(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if palette == nil {
		t.Fatal("expected non-nil palette")
	}
	if !palette.Added.IsCustom {
		t.Error("expected Added.IsCustom=true")
	}
	if !palette.Removed.IsCustom {
		t.Error("expected Removed.IsCustom=true")
	}
	if !palette.Modified.IsCustom {
		t.Error("expected Modified.IsCustom=true")
	}
	if !palette.Context.IsCustom {
		t.Error("expected Context.IsCustom=true")
	}
	if !palette.DocName.IsCustom {
		t.Error("expected DocName.IsCustom=true")
	}
}

func TestLoadColorPalette_InvalidColor(t *testing.T) {
	invalid := "notacolor"
	fc := &FileConfig{
		Colors: &ColorOverrides{
			Added: &invalid,
		},
	}

	_, err := loadColorPalette(fc)
	if err == nil {
		t.Error("expected error for invalid color")
	}
}

func TestLoadColorPalette_InvalidEnvVar(t *testing.T) {
	t.Setenv("DIFFYML_COLOR_MODIFIED", "#xyz")

	_, err := loadColorPalette(nil)
	if err == nil {
		t.Error("expected error for invalid env var color")
	}
}
