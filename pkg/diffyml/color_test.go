package diffyml

import (
	"os"
	"testing"
	"time"
)

// --- Mutation testing: color.go ---

func TestShouldUseTrueColor_Requested(t *testing.T) {
	cfg := NewColorConfig(ColorModeAlways, true)
	if !cfg.ShouldUseTrueColor() {
		t.Error("ShouldUseTrueColor() should return true when trueColor is requested")
	}
}

func TestShouldUseTrueColor_NotRequested(t *testing.T) {
	cfg := NewColorConfig(ColorModeAlways, false)
	if cfg.ShouldUseTrueColor() {
		t.Error("ShouldUseTrueColor() should return false when trueColor is not requested")
	}
}

// fakeFileInfo implements os.FileInfo for mocking terminal detection.
type fakeFileInfo struct {
	mode os.FileMode
}

func (f fakeFileInfo) Name() string       { return "stdout" }
func (f fakeFileInfo) Size() int64        { return 0 }
func (f fakeFileInfo) Mode() os.FileMode  { return f.mode }
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool        { return false }
func (f fakeFileInfo) Sys() any           { return nil }

func TestIsTerminal_WithCharDevice(t *testing.T) {
	// Mock stdoutStatFn to simulate a real terminal (character device).
	// Kills both mutants:
	//   color.go:81 (!= 0 → == 0) — mutant returns false for char device
	//   color.go:77 (err != nil → == nil) — mutant returns false on success
	orig := stdoutStatFn
	t.Cleanup(func() { stdoutStatFn = orig })

	stdoutStatFn = func() (os.FileInfo, error) {
		return fakeFileInfo{mode: os.ModeCharDevice}, nil
	}

	if !IsTerminal(0) {
		t.Error("IsTerminal should return true when stdout is a character device")
	}
}

func TestIsTerminal_StatError(t *testing.T) {
	// Mock stdoutStatFn to return an error.
	// Kills mutant at color.go:77 (err != nil → == nil) via the error path.
	orig := stdoutStatFn
	t.Cleanup(func() { stdoutStatFn = orig })

	stdoutStatFn = func() (os.FileInfo, error) {
		return nil, os.ErrPermission
	}

	if IsTerminal(0) {
		t.Error("IsTerminal should return false when Stat() returns an error")
	}
}

func TestIsTerminal_Pipe(t *testing.T) {
	// When piped (e.g. in tests), IsTerminal should return false for a pipe fd
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() failed: %v", err)
	}
	defer func() { _ = r.Close() }()
	defer func() { _ = w.Close() }()

	// Note: IsTerminal actually checks os.Stdout, not the fd parameter.
	// This test just ensures it doesn't panic.
	_ = IsTerminal(r.Fd())
}

func TestResolveColorMode_DefaultBranch(t *testing.T) {
	// Exercise the default case with an invalid ColorMode value.
	if ResolveColorMode(ColorMode(99), true) != true {
		t.Error("default branch should fall back to isTerminal value")
	}
	if ResolveColorMode(ColorMode(99), false) != false {
		t.Error("default branch should fall back to isTerminal value")
	}
}

func TestDetectTerminal(t *testing.T) {
	orig := stdoutStatFn
	t.Cleanup(func() { stdoutStatFn = orig })

	stdoutStatFn = func() (os.FileInfo, error) {
		return fakeFileInfo{mode: os.ModeCharDevice}, nil
	}

	cfg := NewColorConfig(ColorModeAuto, false)
	cfg.DetectTerminal()
	if !cfg.ShouldUseColor() {
		t.Error("after DetectTerminal with char device, ShouldUseColor should be true")
	}
}

func TestToFormatOptions_NilOpts(t *testing.T) {
	cfg := NewColorConfig(ColorModeAlways, true)
	cfg.ToFormatOptions(nil) // should not panic
}

func TestToFormatOptions_AppliesColorAndTrueColor(t *testing.T) {
	// Pin the two assignments in ToFormatOptions: opts.Color must reflect
	// ShouldUseColor and opts.TrueColor must reflect ShouldUseTrueColor.
	cfg := NewColorConfig(ColorModeAlways, true)
	opts := &FormatOptions{Color: false, TrueColor: false}
	cfg.ToFormatOptions(opts)
	if !opts.Color {
		t.Error("expected opts.Color=true after ToFormatOptions with ColorModeAlways")
	}
	if !opts.TrueColor {
		t.Error("expected opts.TrueColor=true after ToFormatOptions with trueColor=true")
	}

	// Inverse: never mode + no true color → both false
	cfg2 := NewColorConfig(ColorModeNever, false)
	opts2 := &FormatOptions{Color: true, TrueColor: true}
	cfg2.ToFormatOptions(opts2)
	if opts2.Color {
		t.Error("expected opts.Color=false after ToFormatOptions with ColorModeNever")
	}
	if opts2.TrueColor {
		t.Error("expected opts.TrueColor=false after ToFormatOptions with trueColor=false")
	}
}

func TestIsTerminal_NonCharDevice(t *testing.T) {
	// Mock stdoutStatFn to return a non-char-device mode (named pipe).
	// Kills the INVERT_BITWISE mutant at color.go:69 (& → |): with `|`,
	// (Mode | ModeCharDevice) is always non-zero, so the mutant returns
	// true even for non-terminals.
	orig := stdoutStatFn
	t.Cleanup(func() { stdoutStatFn = orig })

	stdoutStatFn = func() (os.FileInfo, error) {
		return fakeFileInfo{mode: os.ModeNamedPipe}, nil
	}

	if IsTerminal(0) {
		t.Error("IsTerminal should return false for a non-char-device (named pipe)")
	}
}

func TestTrueColorCode_ClampsBlueAbove255(t *testing.T) {
	// Pins the `b = clamp(b, 0, 255)` assignment in TrueColorCode at color.go:156.
	// Without clamping, the formatted string would contain "300" for b.
	got := TrueColorCode(0, 0, 300)
	want := TrueColorCode(0, 0, 255)
	if got != want {
		t.Errorf("TrueColorCode blue=300 must clamp to 255: got %q want %q", got, want)
	}
}

func TestDetailedColorCode_UnknownType(t *testing.T) {
	// Exercise the fallback return "" for an unknown diff type in 8-color mode.
	code := DetailedColorCode(DiffType(99), false)
	if code != "" {
		t.Errorf("expected empty string for unknown diff type, got %q", code)
	}
}

func TestColorReset(t *testing.T) {
	if ColorReset() != "\033[0m" {
		t.Errorf("expected ANSI reset code, got %q", ColorReset())
	}
}

func TestDocNameColorCode(t *testing.T) {
	tc := DocNameColorCode(true)
	if tc != TrueColorCode(DetailedDocNameR, DetailedDocNameG, DetailedDocNameB) {
		t.Errorf("true color mismatch: got %q", tc)
	}
	fc := DocNameColorCode(false)
	if fc != "\033[36m" {
		t.Errorf("8-color fallback mismatch: got %q", fc)
	}
}

func TestDetectTrueColorSupport(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{"truecolor", "truecolor", true},
		{"24bit", "24bit", true},
		{"empty", "", false},
		{"256color", "256color", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("COLORTERM", tt.value)
			if got := DetectTrueColorSupport(); got != tt.expected {
				t.Errorf("DetectTrueColorSupport() with COLORTERM=%q = %v, want %v", tt.value, got, tt.expected)
			}
		})
	}
}

func TestEntryPalette_NeutralFallback(t *testing.T) {
	// DiffModified is not DiffAdded or DiffRemoved, so entryPalette returns the neutral palette.
	p := entryPalette(DiffModified, true)
	if p != cachedNeutralPalette {
		t.Errorf("expected neutral TrueColor palette for DiffModified, got: %+v", p)
	}
	p = entryPalette(DiffModified, false)
	if p != cachedFlatNeutral {
		t.Errorf("expected neutral flat palette for DiffModified, got: %+v", p)
	}
	p = entryPalette(DiffOrderChanged, true)
	if p != cachedNeutralPalette {
		t.Errorf("expected neutral TrueColor palette for DiffOrderChanged, got: %+v", p)
	}
	p = entryPalette(DiffOrderChanged, false)
	if p != cachedFlatNeutral {
		t.Errorf("expected neutral flat palette for DiffOrderChanged, got: %+v", p)
	}
}
