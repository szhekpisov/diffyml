package format

import (
	"strings"
	"testing"
	"time"

	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/types"
)

func TestConvertToGoPatchPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"config.name", "/config/name"},
		{"items[0].value", "/items/0/value"},
		{"root", "/root"},
		{"a.b.c.d", "/a/b/c/d"},
		{"list[0][1].nested", "/list/0/1/nested"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ConvertToGoPatchPath(tt.input)
			if result != tt.expected {
				t.Errorf("ConvertToGoPatchPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDiffDescription_Modified(t *testing.T) {
	diff := types.Difference{
		Path: "config.host",
		Type: types.DiffModified,
		From: "localhost",
		To:   "production",
	}
	desc := DiffDescription(diff)
	if !strings.Contains(desc, "config.host") {
		t.Errorf("expected path in description, got %q", desc)
	}
	if !strings.Contains(desc, "Modified") {
		t.Errorf("expected 'Modified' in description, got %q", desc)
	}
}

func TestDiffDescription_AllTypes(t *testing.T) {
	diffs := []types.Difference{
		{Path: "config.host", Type: types.DiffModified, From: "localhost", To: "production"},
		{Path: "config.port", Type: types.DiffAdded, To: 8080},
		{Path: "config.old", Type: types.DiffRemoved, From: "value"},
		{Path: "items", Type: types.DiffOrderChanged},
	}
	for _, d := range diffs {
		desc := DiffDescription(d)
		if desc == "" {
			t.Errorf("expected non-empty description for diff type %d", d.Type)
		}
	}
}

func TestFormatValue_OrderedMap(t *testing.T) {
	om := types.NewOrderedMap()
	om.Keys = []string{"name", "port"}
	om.Values["name"] = "my-service"
	om.Values["port"] = 8080
	result := FormatValue(om)
	if strings.Contains(result, "OrderedMap") || strings.Contains(result, "&{") {
		t.Errorf("FormatValue should not produce Go struct repr for *OrderedMap, got: %s", result)
	}
}

func TestFormatValue_ListWithOrderedMaps(t *testing.T) {
	inner := types.NewOrderedMap()
	inner.Keys = []string{"name"}
	inner.Values["name"] = "item1"
	val := []interface{}{inner, "plain-string"}
	result := FormatValue(val)
	if strings.Contains(result, "OrderedMap") || strings.Contains(result, "&{") {
		t.Errorf("FormatValue should not produce Go struct repr for []interface{} with *OrderedMap, got: %s", result)
	}
	if strings.Contains(result, "0x") || strings.Contains(result, "0X") {
		t.Errorf("FormatValue should not produce pointer addresses, got: %s", result)
	}
}

func TestFormatValue_MapStringInterface(t *testing.T) {
	val := map[string]interface{}{"key": "val", "num": 42}
	result := FormatValue(val)
	if strings.Contains(result, "map[") {
		t.Errorf("FormatValue should serialize map[string]interface{} as YAML, not Go repr, got: %s", result)
	}
}

func TestFormatValue_ScalarUnchanged(t *testing.T) {
	if got := FormatValue("hello"); got != "hello" {
		t.Errorf("FormatValue(string) = %q, want %q", got, "hello")
	}
	if got := FormatValue(42); got != "42" {
		t.Errorf("FormatValue(int) = %q, want %q", got, "42")
	}
	if got := FormatValue(nil); got != "<nil>" {
		t.Errorf("FormatValue(nil) = %q, want %q", got, "<nil>")
	}
}

func TestFormatValue_Timestamp(t *testing.T) {
	ts := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	got := FormatValue(ts)
	if got != "2024-01-15" {
		t.Errorf("FormatValue(date) = %q, want %q", got, "2024-01-15")
	}

	ts = time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	got = FormatValue(ts)
	if !strings.Contains(got, "2024-01-15") {
		t.Errorf("FormatValue(datetime) should contain date, got %q", got)
	}
}
