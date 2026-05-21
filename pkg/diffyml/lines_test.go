package diffyml

import "testing"

// findDiff returns the first diff whose path string matches, or fails.
func findDiff(t *testing.T, diffs []Difference, path string) Difference {
	t.Helper()
	for _, d := range diffs {
		if d.Path.String() == path {
			return d
		}
	}
	t.Fatalf("no diff found for path %q (have %d diffs)", path, len(diffs))
	return Difference{}
}

func TestCaptureLineNumbers_BasicCases(t *testing.T) {
	from := `name: alice
age: 30
config: |
  line one
  line two
nested:
  inner: 5
items:
- name: x
  val: 1
- name: y
  val: 2
`
	to := `name: bob
age: 30
config: |
  line one
  CHANGED
nested:
  inner: 6
  added: true
items:
- name: x
  val: 1
- name: y
  val: 99
`
	diffs, err := Compare([]byte(from), []byte(to), &Options{CaptureLineNumbers: true})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		path             string
		wantFrom, wantTo int
	}{
		{"name", 1, 1},          // inline scalar, same line both files
		{"config", 3, 3},        // block scalar anchored at the "config: |" indicator line
		{"nested.inner", 7, 7},  // nested scalar
		{"items.y.val", 12, 13}, // identifier-matched list item, differing lines
		{"nested", 0, 8},        // added key "added" resolved to its child line (to-only)
	}
	for _, tc := range tests {
		d := findDiff(t, diffs, tc.path)
		if d.LineFrom != tc.wantFrom || d.LineTo != tc.wantTo {
			t.Errorf("path %q: got LineFrom=%d LineTo=%d, want %d/%d", tc.path, d.LineFrom, d.LineTo, tc.wantFrom, tc.wantTo)
		}
	}
}

func TestCaptureLineNumbers_RemovedKey(t *testing.T) {
	from := `a: 1
gone: 2
b: 3
`
	to := `a: 1
b: 3
`
	diffs, err := Compare([]byte(from), []byte(to), &Options{CaptureLineNumbers: true})
	if err != nil {
		t.Fatal(err)
	}
	d := findDiff(t, diffs, "")
	// Removed key "gone" is reported at root; LineFrom resolves to its source line (2),
	// LineTo is unknown (0).
	if d.Type != DiffRemoved {
		t.Fatalf("expected DiffRemoved, got %d", d.Type)
	}
	if d.LineFrom != 2 || d.LineTo != 0 {
		t.Errorf("removed key: got LineFrom=%d LineTo=%d, want 2/0", d.LineFrom, d.LineTo)
	}
}

func TestCaptureLineNumbers_MultiDoc(t *testing.T) {
	from := "x: 1\n---\ny: 2\n"
	to := "x: 9\n---\ny: 8\n"
	diffs, err := Compare([]byte(from), []byte(to), &Options{CaptureLineNumbers: true})
	if err != nil {
		t.Fatal(err)
	}
	d0 := findDiff(t, diffs, "[0].x")
	if d0.LineFrom != 1 || d0.LineTo != 1 {
		t.Errorf("[0].x: got %d/%d, want 1/1", d0.LineFrom, d0.LineTo)
	}
	d1 := findDiff(t, diffs, "[1].y")
	if d1.LineFrom != 3 || d1.LineTo != 3 {
		t.Errorf("[1].y: got %d/%d, want 3/3", d1.LineFrom, d1.LineTo)
	}
}

func TestCaptureLineNumbers_DisabledByDefault(t *testing.T) {
	from := "a: 1\n"
	to := "a: 2\n"
	diffs, err := Compare([]byte(from), []byte(to), nil)
	if err != nil {
		t.Fatal(err)
	}
	d := findDiff(t, diffs, "a")
	if d.LineFrom != 0 || d.LineTo != 0 {
		t.Errorf("line numbers should be 0 when capture disabled, got %d/%d", d.LineFrom, d.LineTo)
	}
}

func TestCaptureLineNumbers_SkippedUnderChroot(t *testing.T) {
	from := "root:\n  a: 1\n"
	to := "root:\n  a: 2\n"
	diffs, err := Compare([]byte(from), []byte(to), &Options{CaptureLineNumbers: true, Chroot: "root"})
	if err != nil {
		t.Fatal(err)
	}
	d := findDiff(t, diffs, "a")
	if d.LineFrom != 0 || d.LineTo != 0 {
		t.Errorf("line numbers should be skipped under chroot, got %d/%d", d.LineFrom, d.LineTo)
	}
}

func TestCaptureLineNumbers_Swap(t *testing.T) {
	// After swap, the original 'to' becomes 'from'. LineFrom should reflect the
	// (post-swap) from file. Both files put the key on different lines.
	a := "x: 1\n"         // key on line 1
	b := "pad: 0\nx: 2\n" // key on line 2
	diffs, err := Compare([]byte(a), []byte(b), &Options{CaptureLineNumbers: true, Swap: true})
	if err != nil {
		t.Fatal(err)
	}
	d := findDiff(t, diffs, "x")
	// Swapped: from=b (line 2), to=a (line 1).
	if d.LineFrom != 2 || d.LineTo != 1 {
		t.Errorf("swap: got LineFrom=%d LineTo=%d, want 2/1", d.LineFrom, d.LineTo)
	}
}
