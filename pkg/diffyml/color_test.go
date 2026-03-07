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
