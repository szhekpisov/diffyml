package diffyml

import (
	"os"
	"testing"
	"time"
)

// --- Mutation testing: color.go ---

func TestGetTerminalWidth_Zero(t *testing.T) {
	// GetTerminalWidth(0) should return default (80), not min (40)
	got := GetTerminalWidth(0)
	if got != defaultTerminalWidth {
		t.Errorf("GetTerminalWidth(0) = %d, want %d (default)", got, defaultTerminalWidth)
	}
	if got == minTerminalWidth {
		t.Errorf("GetTerminalWidth(0) should not return minTerminalWidth (%d)", minTerminalWidth)
	}
}

func TestShouldUseTrueColor_COLORTERM(t *testing.T) {
	// When COLORTERM=truecolor, ShouldUseTrueColor should return true
	t.Setenv("COLORTERM", "truecolor")
	t.Setenv("TERM", "") // clear TERM to isolate

	cfg := NewColorConfig(ColorModeAlways, true, 0)
	cfg.SetIsTerminal(false) // not a terminal, but trueColor requested

	if !cfg.ShouldUseTrueColor() {
		t.Error("ShouldUseTrueColor() should return true when COLORTERM=truecolor")
	}
}

func TestShouldUseTrueColor_24bit(t *testing.T) {
	t.Setenv("COLORTERM", "24bit")
	t.Setenv("TERM", "")

	cfg := NewColorConfig(ColorModeAlways, true, 0)
	cfg.SetIsTerminal(false)

	if !cfg.ShouldUseTrueColor() {
		t.Error("ShouldUseTrueColor() should return true when COLORTERM=24bit")
	}
}

func TestToFormatOptions_ZeroWidth(t *testing.T) {
	// When color config width is 0, it should NOT overwrite existing opts.Width
	cfg := NewColorConfig(ColorModeNever, false, 0)
	opts := &FormatOptions{Width: 120}

	cfg.ToFormatOptions(opts)

	if opts.Width != 120 {
		t.Errorf("ToFormatOptions with width=0 should not overwrite existing Width, got %d", opts.Width)
	}
}

func TestToFormatOptions_PositiveWidth(t *testing.T) {
	// When color config width is positive, it should set opts.Width
	cfg := NewColorConfig(ColorModeNever, false, 200)
	opts := &FormatOptions{Width: 120}

	cfg.ToFormatOptions(opts)

	if opts.Width != 200 {
		t.Errorf("ToFormatOptions with width=200 should set Width to 200, got %d", opts.Width)
	}
}

func TestShouldUseTrueColor_NotRequested(t *testing.T) {
	// When trueColor is false, ShouldUseTrueColor should return false
	// regardless of environment
	t.Setenv("COLORTERM", "truecolor")

	cfg := NewColorConfig(ColorModeAlways, false, 0)
	cfg.SetIsTerminal(true)

	if cfg.ShouldUseTrueColor() {
		t.Error("ShouldUseTrueColor() should return false when trueColor is not requested")
	}
}

func TestShouldUseTrueColor_TERM256color(t *testing.T) {
	t.Setenv("COLORTERM", "")
	t.Setenv("TERM", "xterm-256color")

	cfg := NewColorConfig(ColorModeAlways, true, 0)
	cfg.SetIsTerminal(false)

	if !cfg.ShouldUseTrueColor() {
		t.Error("ShouldUseTrueColor() should return true when TERM contains 256color")
	}
}

func TestGetTerminalWidth_BelowMin(t *testing.T) {
	got := GetTerminalWidth(10)
	if got != minTerminalWidth {
		t.Errorf("GetTerminalWidth(10) = %d, want %d (min)", got, minTerminalWidth)
	}
}

func TestGetTerminalWidth_AboveMin(t *testing.T) {
	got := GetTerminalWidth(100)
	if got != 100 {
		t.Errorf("GetTerminalWidth(100) = %d, want 100", got)
	}
}

// fakeFileInfo implements os.FileInfo for mocking terminal detection.
type fakeFileInfo struct {
	mode os.FileMode
}

func (f fakeFileInfo) Name() string      { return "stdout" }
func (f fakeFileInfo) Size() int64       { return 0 }
func (f fakeFileInfo) Mode() os.FileMode { return f.mode }
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool       { return false }
func (f fakeFileInfo) Sys() interface{}  { return nil }

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
