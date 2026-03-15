package diffyml

import "testing"

func TestDiffPath_String(t *testing.T) {
	tests := []struct {
		path DiffPath
		want string
	}{
		{nil, ""},
		{DiffPath{}, ""},
		{DiffPath{"root"}, "root"},
		{DiffPath{"a", "b", "c"}, "a.b.c"},
		{DiffPath{"metadata", "labels", "helm.sh/chart"}, "metadata.labels[helm.sh/chart]"},
		{DiffPath{"data", "tls.crt"}, "data[tls.crt]"},
		{DiffPath{"[0]", "spec", "field"}, "[0].spec.field"},
		{DiffPath{"[0]"}, "[0]"},
		{DiffPath{"helm.sh/chart"}, "[helm.sh/chart]"},
		{DiffPath{"a", "app.kubernetes.io/managed-by"}, "a[app.kubernetes.io/managed-by]"},
	}
	for _, tt := range tests {
		got := tt.path.String()
		if got != tt.want {
			t.Errorf("DiffPath%v.String() = %q, want %q", []string(tt.path), got, tt.want)
		}
	}
}

func TestDiffPath_Append(t *testing.T) {
	p := DiffPath{"a", "b"}
	p2 := p.Append("c")

	if got := p2.String(); got != "a.b.c" {
		t.Errorf("Append result = %q, want %q", got, "a.b.c")
	}
	// Original must not be mutated
	if len(p) != 2 {
		t.Errorf("original path was mutated: len = %d, want 2", len(p))
	}

	// Append to nil
	var nilPath DiffPath
	p3 := nilPath.Append("x")
	if got := p3.String(); got != "x" {
		t.Errorf("nil.Append(x) = %q, want %q", got, "x")
	}
}

func TestDiffPath_Last(t *testing.T) {
	if got := (DiffPath{"a", "b"}).Last(); got != "b" {
		t.Errorf("Last() = %q, want %q", got, "b")
	}
	if got := (DiffPath{}).Last(); got != "" {
		t.Errorf("empty.Last() = %q, want %q", got, "")
	}
	var nilPath DiffPath
	if got := nilPath.Last(); got != "" {
		t.Errorf("nil.Last() = %q, want %q", got, "")
	}
}

func TestDiffPath_Root(t *testing.T) {
	if got := (DiffPath{"a", "b"}).Root(); got != "a" {
		t.Errorf("Root() = %q, want %q", got, "a")
	}
	if got := (DiffPath{}).Root(); got != "" {
		t.Errorf("empty.Root() = %q, want %q", got, "")
	}
}

func TestDiffPath_Depth(t *testing.T) {
	tests := []struct {
		path DiffPath
		want int
	}{
		{nil, 0},
		{DiffPath{"a"}, 0},
		{DiffPath{"a", "b"}, 1},
		{DiffPath{"a", "b", "c"}, 2},
	}
	for _, tt := range tests {
		if got := tt.path.Depth(); got != tt.want {
			t.Errorf("DiffPath%v.Depth() = %d, want %d", []string(tt.path), got, tt.want)
		}
	}
}

func TestDiffPath_Parent(t *testing.T) {
	p := DiffPath{"a", "b", "c"}
	parent := p.Parent()
	if got := parent.String(); got != "a.b" {
		t.Errorf("Parent() = %q, want %q", got, "a.b")
	}

	if got := (DiffPath{"a"}).Parent(); got != nil {
		t.Errorf("single.Parent() = %v, want nil", got)
	}
	if got := (DiffPath{}).Parent(); got != nil {
		t.Errorf("empty.Parent() = %v, want nil", got)
	}
}

func TestDiffPath_IsEmpty(t *testing.T) {
	if !(DiffPath{}).IsEmpty() {
		t.Error("DiffPath{}.IsEmpty() = false, want true")
	}
	var nilPath DiffPath
	if !nilPath.IsEmpty() {
		t.Error("nil.IsEmpty() = false, want true")
	}
	if (DiffPath{"a"}).IsEmpty() {
		t.Error("DiffPath{a}.IsEmpty() = true, want false")
	}
}

func TestDiffPath_HasNumericLast(t *testing.T) {
	tests := []struct {
		path DiffPath
		want bool
	}{
		{DiffPath{"items", "0"}, true},
		{DiffPath{"items", "12"}, true},
		{DiffPath{"items", "name"}, false},
		{DiffPath{"items", ""}, false},
		{nil, false},
	}
	for _, tt := range tests {
		if got := tt.path.HasNumericLast(); got != tt.want {
			t.Errorf("DiffPath%v.HasNumericLast() = %v, want %v", []string(tt.path), got, tt.want)
		}
	}
}

func TestDiffPath_DocIndex(t *testing.T) {
	idx, ok := DiffPath{"[0]", "spec"}.DocIndex()
	if !ok || idx != 0 {
		t.Errorf("DocIndex() = (%d, %v), want (0, true)", idx, ok)
	}

	_, ok = DiffPath{"spec"}.DocIndex()
	if ok {
		t.Error("DocIndex() on non-doc-index should return false")
	}

	_, ok = (DiffPath)(nil).DocIndex()
	if ok {
		t.Error("DocIndex() on nil should return false")
	}
}
