package diffyml

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
)

// perfSink keeps formatter results reachable so the compiler cannot elide the
// work being measured.
var perfSink string

// generateRewrittenPair returns two n-line values that share no line at all.
// A full rewrite is the worst case for the underlying Myers diff — no snake to
// follow, so the edit distance equals the whole input — and it is also the case
// where collapsing cannot help, since collapsing only ever removes *unchanged*
// lines. That makes it the shape where the cost of producing a bounded
// annotation is most visible.
func generateRewrittenPair(n int) (from, to string) {
	fromLines := make([]string, n)
	toLines := make([]string, n)
	for i := range n {
		fromLines[i] = fmt.Sprintf("old setting-%04d: value-%04d", i, i)
		toLines[i] = fmt.Sprintf("new setting-%04d: value-%04d", i, i)
	}
	return strings.Join(fromLines, "\n"), strings.Join(toLines, "\n")
}

// generateSingleChangePair returns two n-line values differing in one line:
// the common case, where a long ConfigMap value has one setting changed.
func generateSingleChangePair(n int) (from, to string) {
	lines := make([]string, n)
	for i := range n {
		lines[i] = fmt.Sprintf("setting-%04d: value-%04d", i, i)
	}
	from = strings.Join(lines, "\n")
	lines[n/2] = "setting-changed: new-value"
	return from, strings.Join(lines, "\n")
}

// formatGitHubMultiline renders one modified multiline difference through the
// full GitHub formatter, the path an annotation actually takes.
func formatGitHubMultiline(from, to string) string {
	f := &GitHubFormatter{}
	return f.Format([]Difference{{
		Path: DiffPath{"data", "values.yaml"},
		Type: DiffModified,
		From: from,
		To:   to,
	}}, DefaultFormatOptions())
}

// allocatedBytes reports the heap bytes fn allocates. TotalAlloc is cumulative
// and counts memory that fn frees again, so it measures the work done rather
// than the memory still held afterwards — which is the point here: a transient
// gigabyte is just as fatal to a CI runner as a retained one.
func allocatedBytes(fn func()) uint64 {
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	fn()
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}

// TestGitHubFormatter_MultilineWorkIsNotQuadratic guards the cost of producing
// a GitHub annotation for a changed multiline value.
//
// The output is capped at gitHubMaxDiffLines regardless of input size, so the
// work behind it should be bounded too. It is not automatically: the Myers
// diff in computeLineDiff snapshots a (2*(m+n)+1)-int row per edit-script step,
// so a fully rewritten value costs O((m+n)^2) memory to render at most 40 lines.
//
// Doubling the input must not roughly quadruple the allocations. A linear
// implementation lands near 2x; a quadratic one near 4x. The threshold sits
// between them, wide enough that measurement noise cannot reach it.
func TestGitHubFormatter_MultilineWorkIsNotQuadratic(t *testing.T) {
	const (
		baseLines    = 500
		growthFactor = 2
		// Linear ~2.0, quadratic ~4.0.
		maxGrowth = 3.0
	)

	smallFrom, smallTo := generateRewrittenPair(baseLines)
	largeFrom, largeTo := generateRewrittenPair(baseLines * growthFactor)

	small := allocatedBytes(func() { perfSink = formatGitHubMultiline(smallFrom, smallTo) })
	large := allocatedBytes(func() { perfSink = formatGitHubMultiline(largeFrom, largeTo) })

	if small == 0 {
		t.Fatal("measured zero allocations for the smaller input; the measurement is broken")
	}

	growth := float64(large) / float64(small)
	t.Logf("%d lines: %s, %d lines: %s, growth %.2fx",
		baseLines, byteCount(small), baseLines*growthFactor, byteCount(large), growth)

	if growth > maxGrowth {
		t.Errorf("allocations grew %.2fx when the input doubled (limit %.1fx): %s -> %s.\n"+
			"Rendering a %d-line annotation should not cost more than linear work in the value size.",
			growth, maxGrowth, byteCount(small), byteCount(large), gitHubMaxDiffLines)
	}
}

// TestGitHubFormatter_MultilineAllocationCeiling backstops the growth test with
// an absolute bound. Growth alone cannot catch an implementation that is linear
// but with a ruinous constant, and the ratio is only meaningful next to a scale.
// The limit is deliberately loose — it is a blowup detector, not a budget.
func TestGitHubFormatter_MultilineAllocationCeiling(t *testing.T) {
	const (
		lines            = 1000
		maxBytesPerInput = 100
	)

	from, to := generateRewrittenPair(lines)
	inputSize := uint64(len(from) + len(to))

	allocated := allocatedBytes(func() { perfSink = formatGitHubMultiline(from, to) })

	limit := inputSize * maxBytesPerInput
	t.Logf("input %s, allocated %s (%.1fx)", byteCount(inputSize), byteCount(allocated),
		float64(allocated)/float64(inputSize))

	if allocated > limit {
		t.Errorf("formatting a %d-line value allocated %s, over the %s limit (%dx the %s input)",
			lines, byteCount(allocated), byteCount(limit), maxBytesPerInput, byteCount(inputSize))
	}
}

// TestGitHubFormatter_UnchangedLinesAreCheap guards the case the collapsing was
// written for: a single change inside a long value. Here the edit distance stays
// at two no matter how long the value gets, so cost should track the value size
// and nothing worse.
func TestGitHubFormatter_UnchangedLinesAreCheap(t *testing.T) {
	const (
		baseLines    = 2000
		growthFactor = 2
		maxGrowth    = 3.0
	)

	smallFrom, smallTo := generateSingleChangePair(baseLines)
	largeFrom, largeTo := generateSingleChangePair(baseLines * growthFactor)

	small := allocatedBytes(func() { perfSink = formatGitHubMultiline(smallFrom, smallTo) })
	large := allocatedBytes(func() { perfSink = formatGitHubMultiline(largeFrom, largeTo) })

	if small == 0 {
		t.Fatal("measured zero allocations for the smaller input; the measurement is broken")
	}

	growth := float64(large) / float64(small)
	t.Logf("%d lines: %s, %d lines: %s, growth %.2fx",
		baseLines, byteCount(small), baseLines*growthFactor, byteCount(large), growth)

	if growth > maxGrowth {
		t.Errorf("allocations grew %.2fx when the input doubled (limit %.1fx): %s -> %s.\n"+
			"One changed line in a long value should cost linear work, not more.",
			growth, maxGrowth, byteCount(small), byteCount(large))
	}
}

// byteCount renders a byte count in the largest unit that keeps it readable.
func byteCount(n uint64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := uint64(unit), 0
	for n/div >= unit && exp < 2 {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMG"[exp])
}
