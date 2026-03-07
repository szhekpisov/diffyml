package diffyml

import (
	"strings"
	"testing"
)

func TestFormatFileHeader_BothExist_NoColor(t *testing.T) {
	opts := &FormatOptions{Color: false}
	header := FormatFileHeader("deploy.yaml", FilePairBothExist, opts)

	if !strings.Contains(header, "--- a/deploy.yaml") {
		t.Errorf("expected '--- a/deploy.yaml' in header, got: %q", header)
	}
	if !strings.Contains(header, "+++ b/deploy.yaml") {
		t.Errorf("expected '+++ b/deploy.yaml' in header, got: %q", header)
	}
}

func TestFormatFileHeader_OnlyFrom_NoColor(t *testing.T) {
	opts := &FormatOptions{Color: false}
	header := FormatFileHeader("removed.yaml", FilePairOnlyFrom, opts)

	if !strings.Contains(header, "--- a/removed.yaml") {
		t.Errorf("expected '--- a/removed.yaml' in header, got: %q", header)
	}
	if !strings.Contains(header, "+++ /dev/null") {
		t.Errorf("expected '+++ /dev/null' in header, got: %q", header)
	}
}

func TestFormatFileHeader_OnlyTo_NoColor(t *testing.T) {
	opts := &FormatOptions{Color: false}
	header := FormatFileHeader("added.yaml", FilePairOnlyTo, opts)

	if !strings.Contains(header, "--- /dev/null") {
		t.Errorf("expected '--- /dev/null' in header, got: %q", header)
	}
	if !strings.Contains(header, "+++ b/added.yaml") {
		t.Errorf("expected '+++ b/added.yaml' in header, got: %q", header)
	}
}

func TestFormatFileHeader_BothExist_WithColor(t *testing.T) {
	opts := &FormatOptions{Color: true}
	header := FormatFileHeader("deploy.yaml", FilePairBothExist, opts)

	// Should contain ANSI bold+white for both "---" and "+++"
	if !strings.Contains(header, "\033[1m") {
		t.Errorf("expected bold ANSI code in colored header, got: %q", header)
	}
	if !strings.Contains(header, "\033[37m") {
		t.Errorf("expected white ANSI code in colored header, got: %q", header)
	}
	if !strings.Contains(header, "--- a/deploy.yaml") {
		t.Errorf("expected '--- a/deploy.yaml' in header, got: %q", header)
	}
	if !strings.Contains(header, "\033[0m") {
		t.Errorf("expected reset ANSI code in colored header, got: %q", header)
	}
}

func TestFormatFileHeader_OnlyFrom_WithColor(t *testing.T) {
	opts := &FormatOptions{Color: true}
	header := FormatFileHeader("removed.yaml", FilePairOnlyFrom, opts)

	if !strings.Contains(header, "+++ /dev/null") {
		t.Errorf("expected '+++ /dev/null' in header, got: %q", header)
	}
	if !strings.Contains(header, "\033[1m") {
		t.Errorf("expected bold ANSI code, got: %q", header)
	}
}

func TestFormatFileHeader_OnlyTo_WithColor(t *testing.T) {
	opts := &FormatOptions{Color: true}
	header := FormatFileHeader("added.yaml", FilePairOnlyTo, opts)

	if !strings.Contains(header, "--- /dev/null") {
		t.Errorf("expected '--- /dev/null' in header, got: %q", header)
	}
	if !strings.Contains(header, "\033[1m") {
		t.Errorf("expected bold ANSI code, got: %q", header)
	}
}

func TestFormatFileHeader_EndsWithNewline(t *testing.T) {
	opts := &FormatOptions{Color: false}
	header := FormatFileHeader("test.yaml", FilePairBothExist, opts)

	if !strings.HasSuffix(header, "\n") {
		t.Errorf("expected header to end with newline, got: %q", header)
	}
}

func TestFormatFileHeader_NilOpts(t *testing.T) {
	header := FormatFileHeader("test.yaml", FilePairBothExist, nil)
	expected := "--- a/test.yaml\n+++ b/test.yaml\n"
	if header != expected {
		t.Errorf("expected %q, got %q", expected, header)
	}
}
