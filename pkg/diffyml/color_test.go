package diffyml

import (
	"os"
	"testing"
	"time"
)

func TestResolveColorMode(t *testing.T) {
	tests := []struct {
		name       string
		mode       ColorMode
		isTerminal bool
		expected   bool
	}{
		{"always/terminal", ColorModeAlways, true, true},
		{"always/not terminal", ColorModeAlways, false, true},
		{"never/terminal", ColorModeNever, true, false},
		{"never/not terminal", ColorModeNever, false, false},
		{"auto/terminal", ColorModeAuto, true, true},
		{"auto/not terminal", ColorModeAuto, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveColorMode(tt.mode, tt.isTerminal)
			if got != tt.expected {
				t.Errorf("ResolveColorMode(%v, %v) = %v, want %v", tt.mode, tt.isTerminal, got, tt.expected)
			}
		})
	}
}

func TestParseColorMode_Valid(t *testing.T) {
	tests := []struct {
		input    string
		expected ColorMode
	}{
		{"always", ColorModeAlways},
		{"ALWAYS", ColorModeAlways},
		{"Always", ColorModeAlways},
		{"never", ColorModeNever},
		{"NEVER", ColorModeNever},
		{"auto", ColorModeAuto},
		{"AUTO", ColorModeAuto},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			mode, err := ParseColorMode(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if mode != tt.expected {
				t.Errorf("ParseColorMode(%q) = %v, want %v", tt.input, mode, tt.expected)
			}
		})
	}
}

func TestParseColorMode_Invalid(t *testing.T) {
	_, err := ParseColorMode("invalid")
	if err == nil {
		t.Error("expected error for invalid color mode")
	}
}

func TestParseColorMode_Empty(t *testing.T) {
	mode, err := ParseColorMode("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != ColorModeAuto {
		t.Errorf("empty string should default to ColorModeAuto, got %v", mode)
	}
}

func TestColorConfig(t *testing.T) {
	tests := []struct {
		name           string
		mode           ColorMode
		trueColor      bool
		isTerminal     bool
		wantColor      bool
		wantTrueColor  bool
	}{
		{"new is not nil", ColorModeAuto, false, false, false, false},
		{"auto+terminal", ColorModeAuto, false, true, true, false},
		{"auto+no terminal", ColorModeAuto, false, false, false, false},
		{"always+truecolor", ColorModeAlways, true, true, true, true},
		{"always+no truecolor", ColorModeAlways, false, true, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewColorConfig(tt.mode, tt.trueColor)
			if cfg == nil {
				t.Fatal("NewColorConfig should not return nil")
			}
			cfg.SetIsTerminal(tt.isTerminal)
			if got := cfg.ShouldUseColor(); got != tt.wantColor {
				t.Errorf("ShouldUseColor() = %v, want %v", got, tt.wantColor)
			}
			if got := cfg.ShouldUseTrueColor(); got != tt.wantTrueColor {
				t.Errorf("ShouldUseTrueColor() = %v, want %v", got, tt.wantTrueColor)
			}
		})
	}
}

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
func (f fakeFileInfo) Sys() interface{}   { return nil }

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
