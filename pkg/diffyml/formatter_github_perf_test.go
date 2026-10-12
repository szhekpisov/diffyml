package diffyml

import (
	"fmt"
	"math"
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

// allocationSamples is how many times allocatedBytes runs fn before taking the
// smallest reading. fn here is well under a millisecond, so the extra runs are
// free next to what they buy — see allocatedBytes for why the minimum is the
// right summary rather than the mean.
const allocationSamples = 5

// allocatedBytes reports the heap bytes fn allocates. TotalAlloc is cumulative
// and counts memory that fn frees again, so it measures the work done rather
// than the memory still held afterwards — which is the point here: a transient
// gigabyte is just as fatal to a CI runner as a retained one.
//
// TotalAlloc is also process-wide, so anything else allocating between the two
// reads lands in the delta. Today nothing does: no test in this package calls
// t.Parallel, so tests run one at a time and the formatter path spawns no
// goroutines — the readings vary by well under a percent, with or without
// -race. The hazard is that none of that is enforced, and the day someone adds
// a t.Parallel elsewhere in the package these thresholds start failing for a
// reason that has nothing to do with the formatter.
//
// Interference can only ever *add* to the delta, never subtract, so the
// smallest of several readings is the closest estimate of what fn alone costs,
// and one clean run is enough to get it. The mean would instead drag toward
// whatever else was running.
func allocatedBytes(fn func()) uint64 {
	smallest := uint64(math.MaxUint64)
	for range allocationSamples {
		var before, after runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&before)
		fn()
		runtime.ReadMemStats(&after)
		if d := after.TotalAlloc - before.TotalAlloc; d < smallest {
			smallest = d
		}
	}
	return smallest
}

// TestGitHubFormatter_MultilineWorkIsNotQuadratic guards the cost of producing
// a GitHub annotation for a changed multiline value.
//
// The output is capped at gitHubMaxDiffLines regardless of input size, so the
// work behind it should be bounded too. It is not automatically: the Myers diff
// searches an edit script whose length is the whole input for a fully rewritten
// value, and snapshots a row per step of it, so without the ceiling in
// computeLineDiffBounded rendering at most 40 lines costs O((m+n)^2) memory.
//
// Doubling the input must not roughly quadruple the allocations. A linear
// implementation lands near 2x; a quadratic one near 4x. The threshold sits
// between them, wide enough that measurement noise cannot reach it.
// MultilineAllocationCeiling is the sharper guard — see there for why growth
// alone does not catch a trace that snapshots more than it needs.
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
//
// It is also what holds the trace to a band. Snapshotting the whole Myers row
// per step is still linear in the value — the growth test above passes either
// way — but at ~1.3 KB per line, which measures ~46x here against the ~3x a
// banded trace costs. The limit sits between them, far enough from both that
// neither measurement noise nor a modest constant-factor change can reach it.
func TestGitHubFormatter_MultilineAllocationCeiling(t *testing.T) {
	const (
		lines            = 1000
		maxBytesPerInput = 20
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
