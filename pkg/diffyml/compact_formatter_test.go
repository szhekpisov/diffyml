package diffyml

import (
	"strings"
	"testing"
)

func TestCompactFormatter_SingleLineFormat(t *testing.T) {
	f, _ := GetFormatter("compact")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "config.timeout", Type: DiffModified, From: "30", To: "60"},
	}

	output := f.Format(diffs, opts)

	for _, want := range []string{"±", "config.timeout", "30", "60", "→"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in output, got: %s", want, output)
		}
	}
}

func TestCompactFormatter_ChangeTypeIndicators(t *testing.T) {
	f, _ := GetFormatter("compact")
	opts := DefaultFormatOptions()

	tests := []struct {
		name      string
		diffType  DiffType
		indicator string
	}{
		{"added", DiffAdded, "+"},
		{"removed", DiffRemoved, "-"},
		{"modified", DiffModified, "±"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diffs := []Difference{
				{Path: "test.path", Type: tt.diffType, From: "old", To: "new"},
			}
			output := f.Format(diffs, opts)
			if !strings.Contains(output, tt.indicator) {
				t.Errorf("expected indicator %q in output, got: %s", tt.indicator, output)
			}
		})
	}
}

func TestCompactFormatter_ColorCodes(t *testing.T) {
	f, _ := GetFormatter("compact")
	diffs := []Difference{
		{Path: "test", Type: DiffAdded, From: nil, To: "new"},
	}
	opts := DefaultFormatOptions()
	opts.Color = true

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "\033[32m") {
		t.Errorf("expected green color code for additions, got: %s", output)
	}
}

func TestCompactFormatter_InlineColor(t *testing.T) {
	f := &CompactFormatter{}
	opts := DefaultFormatOptions()
	opts.Color = true

	diffs := []Difference{
		{Path: "key.a", Type: DiffAdded, To: "new"},
		{Path: "key.r", Type: DiffRemoved, From: "old"},
		{Path: "key.m", Type: DiffModified, From: "old", To: "new"},
		{Path: "key.o", Type: DiffOrderChanged},
	}

	output := f.Format(diffs, opts)
	for _, color := range []string{colorGreen, colorRed, colorYellow} {
		if !strings.Contains(output, color) {
			t.Errorf("expected color %q in output, got: %s", color, output)
		}
	}
}

func TestCompactFormatter_OrderChangedIndicator(t *testing.T) {
	f, _ := GetFormatter("compact")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "list.items", Type: DiffOrderChanged},
	}
	output := f.Format(diffs, opts)
	if !strings.Contains(output, "⇆") {
		t.Errorf("expected '⇆' indicator for order changed, got: %s", output)
	}
	if !strings.Contains(output, "(order changed)") {
		t.Errorf("expected '(order changed)' in output, got: %s", output)
	}
}

func TestCompactFormatter_GoPatchStylePath(t *testing.T) {
	f := &CompactFormatter{}
	opts := DefaultFormatOptions()
	opts.UseGoPatchStyle = true

	diff := Difference{Path: "config.items[0].name", Type: DiffModified, From: "old", To: "new"}
	output := f.FormatSingle(diff, opts)
	if !strings.Contains(output, "/config/items/0/name") {
		t.Errorf("expected Go-Patch style path, got: %s", output)
	}
}

func TestCompactFormatter_FormatSingle_NilOpts(t *testing.T) {
	f := &CompactFormatter{}
	diff := Difference{Path: "key", Type: DiffAdded, To: "value"}

	output := f.FormatSingle(diff, nil)
	if output == "" {
		t.Error("FormatSingle with nil opts should produce output")
	}
}

func TestCompactFormatter_FormatSingle_NilValue(t *testing.T) {
	f := &CompactFormatter{}
	opts := DefaultFormatOptions()

	diff := Difference{Path: "key", Type: DiffModified, From: nil, To: "new"}
	output := f.FormatSingle(diff, opts)
	if !strings.Contains(output, "<nil>") {
		t.Errorf("expected <nil> for nil value, got: %s", output)
	}
}

func TestCompactFormatter_HeaderCounts(t *testing.T) {
	diffs := []Difference{
		{Path: "a", Type: DiffAdded, To: "x"},
		{Path: "b", Type: DiffAdded, To: "y"},
		{Path: "c", Type: DiffRemoved, From: "z"},
		{Path: "d", Type: DiffModified, From: "old", To: "new"},
	}

	f := &CompactFormatter{}
	opts := &FormatOptions{Color: false}
	output := f.Format(diffs, opts)

	for _, want := range []string{"(2 added,", " 1 removed,", " 1 modified)"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in header, got: %s", want, output)
		}
	}
}

func TestStyleConstants_BoldAndItalic(t *testing.T) {
	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"styleBold", styleBold, "\033[1m"},
		{"styleBoldOff", styleBoldOff, "\033[22m"},
		{"styleItalic", styleItalic, "\033[3m"},
		{"styleItalicOff", styleItalicOff, "\033[23m"},
	}
	for _, tt := range tests {
		if tt.got != tt.expected {
			t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.expected)
		}
	}
}

func TestStyleConstants_CombiningWithColor(t *testing.T) {
	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"bold+green", styleBold + colorGreen, "\033[1m\033[32m"},
		{"italic+yellow", styleItalic + colorYellow, "\033[3m\033[33m"},
	}
	for _, tt := range tests {
		if tt.got != tt.expected {
			t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.expected)
		}
	}
}
