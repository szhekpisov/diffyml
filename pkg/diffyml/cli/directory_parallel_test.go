package cli

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
)

// testWorkers is the GOMAXPROCS value these tests use to force the parallel
// path. It is a fixed number rather than runtime.NumCPU() on purpose: GOMAXPROCS
// is not capped by the CPU count, so raising it spawns real workers and
// interleaves them even on a single-core runner. The race detector's
// happens-before analysis does not need true parallelism to find a race, so
// pinning it here keeps the ordering guarantees under test everywhere —
// including the one-CPU CI containers where a latent bug is easiest to miss.
const testWorkers = 4

// withGOMAXPROCS pins GOMAXPROCS for the duration of a test so both the
// streaming path (workers < 2) and the parallel path can be exercised
// deterministically. These tests must not call t.Parallel(), since GOMAXPROCS
// is process-global.
func withGOMAXPROCS(t *testing.T, n int) {
	t.Helper()
	prev := runtime.GOMAXPROCS(n)
	t.Cleanup(func() { runtime.GOMAXPROCS(prev) })
}

// buildParallelPairs returns n in-memory file pairs with differing content.
// Every other pair is given multi-key content so pairs vary in comparison cost,
// which makes workers finish out of index order.
func buildParallelPairs(n int) map[string][2][]byte {
	pairs := make(map[string][2][]byte, n)
	for i := range n {
		name := fmt.Sprintf("file%03d.yaml", i)
		if i%2 == 0 {
			pairs[name] = [2][]byte{
				[]byte(fmt.Sprintf("name: svc%d\nreplicas: 1\n", i)),
				[]byte(fmt.Sprintf("name: svc%d\nreplicas: 2\n", i)),
			}
			continue
		}
		var from, to strings.Builder
		for k := range 40 {
			fmt.Fprintf(&from, "key%02d: old-%d-%d\n", k, i, k)
			fmt.Fprintf(&to, "key%02d: new-%d-%d\n", k, i, k)
		}
		pairs[name] = [2][]byte{[]byte(from.String()), []byte(to.String())}
	}
	return pairs
}

// buildK8sPairs returns n in-memory manifest pairs shaped to exercise the
// option-driven parts of the compare phase: Kubernetes entity detection and
// rename detection, Secret masking, x509 inspection, and the neat bundle.
func buildK8sPairs(n int) map[string][2][]byte {
	pairs := make(map[string][2][]byte, n)
	for i := range n {
		from := fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: svc%d
  namespace: default
  annotations:
    meta.helm.sh/release-name: r%d
    kubectl.kubernetes.io/last-applied-configuration: '{"a":1}'
  labels:
    app.kubernetes.io/managed-by: Helm
spec:
  replicas: 1
  template:
    spec:
      containers:
        - name: c
          image: nginx:1.1
          env:
            - name: PASSWORD
              value: hunter2-%d
status:
  readyReplicas: 1
---
apiVersion: v1
kind: Secret
metadata:
  name: s%d
data:
  password: aHVudGVyMg==
`, i, i, i, i)
		to := strings.NewReplacer(
			"replicas: 1", "replicas: 3",
			"nginx:1.1", "nginx:1.2",
			"hunter2-", "hunter3-",
			"readyReplicas: 1", "readyReplicas: 3",
			"aHVudGVyMg==", "aHVudGVyMw==",
		).Replace(from)
		pairs[fmt.Sprintf("manifest%03d.yaml", i)] = [2][]byte{[]byte(from), []byte(to)}
	}
	return pairs
}

// dirRun is the observable outcome of one directory-mode run.
type dirRun struct {
	stdout, stderr string
	code           int
}

// runDirectoryCapture runs directory mode over the given pairs and returns what
// it wrote and its exit code. Any tweak functions are applied to the config
// before the run, for tests that need options beyond the defaults.
func runDirectoryCapture(t *testing.T, output string, pairs map[string][2][]byte, tweak ...func(*CLIConfig)) dirRun {
	t.Helper()
	cfg := NewCLIConfig()
	cfg.Color = "never"
	cfg.SetExitCode = true
	cfg.Output = output
	for _, fn := range tweak {
		fn(cfg)
	}

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = pairs

	result := runDirectory(cfg, rc, "", "")
	return dirRun{stdout.String(), stderr.String(), result.Code}
}

// assertSameRun fails the test if two runs differ in output or exit code.
func assertSameRun(t *testing.T, label string, want, got dirRun) {
	t.Helper()
	if got.stdout != want.stdout {
		t.Errorf("%s: stdout differs\nwant (%d bytes):\n%s\ngot (%d bytes):\n%s",
			label, len(want.stdout), want.stdout, len(got.stdout), got.stdout)
	}
	if got.stderr != want.stderr {
		t.Errorf("%s: stderr differs\nwant:\n%s\ngot:\n%s", label, want.stderr, got.stderr)
	}
	if got.code != want.code {
		t.Errorf("%s: exit code %d, want %d", label, got.code, want.code)
	}
}

// TestRunDirectory_ParallelMatchesSequential is the core guarantee of the
// parallel compare phase: results are replayed in pair order, so output must be
// byte-identical to the single-worker streaming path for every output format.
func TestRunDirectory_ParallelMatchesSequential(t *testing.T) {
	formats := []string{"detailed", "compact", "brief", "json", "json-patch", "github", "gitlab", "gitea"}
	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			withGOMAXPROCS(t, 1)
			seq := runDirectoryCapture(t, format, buildParallelPairs(60))

			withGOMAXPROCS(t, testWorkers)
			par := runDirectoryCapture(t, format, buildParallelPairs(60))

			assertSameRun(t, "parallel vs sequential", seq, par)
		})
	}
}

// TestRunDirectory_ParallelMatchesSequentialWithOptions extends the equivalence
// guarantee to the option structs the workers share. Those are what the
// safety argument for sharing rests on — Compare, MaskDifferences and
// FilterDiffsWithRegexp treating their options as immutable — and the
// default-config tests exercise none of them.
func TestRunDirectory_ParallelMatchesSequentialWithOptions(t *testing.T) {
	heavy := func(cfg *CLIConfig) {
		cfg.DetectKubernetes = true
		cfg.DetectRenames = true
		cfg.MaskSecrets = true
		cfg.MaskPathRegexp = []string{".*[Pp]assword.*"}
		cfg.Neat = true
		cfg.FilterRegexp = []string{".*"}
		cfg.ExcludeRegexp = []string{"nothing-matches-this"}
		cfg.MultiLineContextLines = 2
	}

	withGOMAXPROCS(t, 1)
	seq := runDirectoryCapture(t, "detailed", buildK8sPairs(40), heavy)

	withGOMAXPROCS(t, testWorkers)
	par := runDirectoryCapture(t, "detailed", buildK8sPairs(40), heavy)

	if seq.stdout == "" {
		t.Fatal("no output produced; the comparison is not exercising anything")
	}
	assertSameRun(t, "parallel vs sequential with kubernetes/mask/neat/filter options", seq, par)
}

// TestRunDirectory_ParallelIsDeterministic guards against completion order
// leaking into the output: repeated parallel runs over identical input must
// produce identical bytes.
func TestRunDirectory_ParallelIsDeterministic(t *testing.T) {
	withGOMAXPROCS(t, testWorkers)

	want := runDirectoryCapture(t, "detailed", buildParallelPairs(60))
	for i := range 12 {
		assertSameRun(t, fmt.Sprintf("run %d vs run 0", i+1), want, runDirectoryCapture(t, "detailed", buildParallelPairs(60)))
		if t.Failed() {
			return
		}
	}
}

// TestRunDirectory_ParallelPreservesErrorOrder verifies that per-pair errors are
// reported in pair order rather than completion order, and that a failing pair
// does not suppress the diffs of the pairs around it.
func TestRunDirectory_ParallelPreservesErrorOrder(t *testing.T) {
	pairs := buildParallelPairs(30)
	badNames := []string{"file005.yaml", "file015.yaml", "file025.yaml"}
	for _, name := range badNames {
		pairs[name] = [2][]byte{[]byte("key: old\n"), []byte(":\nbad yaml [[[")}
	}

	withGOMAXPROCS(t, 1)
	seq := runDirectoryCapture(t, "detailed", pairs)

	withGOMAXPROCS(t, testWorkers)
	par := runDirectoryCapture(t, "detailed", pairs)

	assertSameRun(t, "parallel vs sequential with failing pairs", seq, par)

	// Errors must appear in pair order, matching the sorted pair plan.
	lastIdx := -1
	for _, name := range badNames {
		idx := strings.Index(par.stderr, name)
		if idx < 0 {
			t.Fatalf("expected an error mentioning %s, got:\n%s", name, par.stderr)
		}
		if idx < lastIdx {
			t.Errorf("error for %s reported out of pair order", name)
		}
		lastIdx = idx
	}

	// A healthy pair on either side of a failing one still produces output.
	for _, name := range []string{"file004.yaml", "file006.yaml"} {
		if !strings.Contains(par.stdout, name) {
			t.Errorf("expected output for %s alongside failing pairs", name)
		}
	}
}

// TestDirCompareWorkers checks the streaming/parallel decision: the count is
// capped by the pair count so a single pair never spawns workers, defaults to
// GOMAXPROCS, and honors an explicit --jobs either way.
func TestDirCompareWorkers(t *testing.T) {
	tests := []struct {
		gomaxprocs int
		pairCount  int
		jobs       int
		want       int
	}{
		{testWorkers, 0, 0, 0},
		{testWorkers, 1, 0, 1}, // a single pair has nothing to overlap with
		{testWorkers, 2, 0, 2},
		{testWorkers, testWorkers, 0, testWorkers},
		{testWorkers, 100, 0, testWorkers},                   // capped at GOMAXPROCS
		{testWorkers, 100, 1, 1},                             // --jobs 1: the caller asked for streaming
		{testWorkers, 100, 2, 2},                             // --jobs below GOMAXPROCS wins
		{testWorkers, 100, testWorkers * 4, testWorkers * 4}, // --jobs above GOMAXPROCS is honored
		{testWorkers, 3, testWorkers * 4, 3},                 // ...but never exceeds the pair count
		{testWorkers, 1, 8, 1},                               // one pair stays sequential whatever --jobs says
		{1, 100, 0, 1},                                       // one usable CPU: nothing to gain
		{1, 100, 4, 4},                                       // ...unless --jobs oversubscribes on purpose
	}
	for _, tt := range tests {
		withGOMAXPROCS(t, tt.gomaxprocs)
		if got := dirCompareWorkers(tt.pairCount, tt.jobs); got != tt.want {
			t.Errorf("GOMAXPROCS=%d: dirCompareWorkers(%d, jobs=%d) = %d, want %d",
				tt.gomaxprocs, tt.pairCount, tt.jobs, got, tt.want)
		}
	}
}

// TestRunDirectory_JobsCapsWorkers exercises --jobs end to end against the
// --jobs 1 streaming path: the default (one per CPU), fewer workers than CPUs,
// and more workers than CPUs must all cover every pair, in order.
func TestRunDirectory_JobsCapsWorkers(t *testing.T) {
	withGOMAXPROCS(t, testWorkers)

	want := runDirectoryCapture(t, "detailed", buildParallelPairs(60),
		func(cfg *CLIConfig) { cfg.Jobs = 1 })

	for _, jobs := range []int{0, 2, 3, testWorkers * 4} {
		got := runDirectoryCapture(t, "detailed", buildParallelPairs(60),
			func(cfg *CLIConfig) { cfg.Jobs = jobs })
		assertSameRun(t, fmt.Sprintf("--jobs %d vs --jobs 1", jobs), want, got)
	}
}
