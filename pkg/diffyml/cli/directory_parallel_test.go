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

// runDirectoryCapture runs directory mode over the given pairs and returns
// stdout, stderr and the exit code. Any tweak functions are applied to the
// config before the run, for tests that need options beyond the defaults.
func runDirectoryCapture(t *testing.T, output string, pairs map[string][2][]byte, tweak ...func(*CLIConfig)) (string, string, int) {
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
	return stdout.String(), stderr.String(), result.Code
}

// TestRunDirectory_ParallelMatchesSequential is the core guarantee of the
// parallel compare phase: results are replayed in pair order, so output must be
// byte-identical to the single-worker streaming path for every output format.
func TestRunDirectory_ParallelMatchesSequential(t *testing.T) {
	formats := []string{"detailed", "compact", "brief", "json", "json-patch", "github", "gitlab", "gitea"}
	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			withGOMAXPROCS(t, 1)
			seqOut, seqErr, seqCode := runDirectoryCapture(t, format, buildParallelPairs(60))

			withGOMAXPROCS(t, testWorkers)
			parOut, parErr, parCode := runDirectoryCapture(t, format, buildParallelPairs(60))

			if seqOut != parOut {
				t.Errorf("stdout differs between sequential and parallel paths\nsequential (%d bytes):\n%s\nparallel (%d bytes):\n%s",
					len(seqOut), seqOut, len(parOut), parOut)
			}
			if seqErr != parErr {
				t.Errorf("stderr differs: sequential %q, parallel %q", seqErr, parErr)
			}
			if seqCode != parCode {
				t.Errorf("exit code differs: sequential %d, parallel %d", seqCode, parCode)
			}
		})
	}
}

// TestRunDirectory_ParallelMatchesSequentialWithOptions extends the equivalence
// guarantee to the option structs the workers share. Those are what the
// safety argument for sharing rests on — Compare, MaskDifferences and
// FilterDiffsWithRegexp treating their options as immutable and compiling
// regexes per call — and the default-config tests exercise none of them.
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
	seqOut, seqErr, seqCode := runDirectoryCapture(t, "detailed", buildK8sPairs(40), heavy)

	withGOMAXPROCS(t, testWorkers)
	parOut, parErr, parCode := runDirectoryCapture(t, "detailed", buildK8sPairs(40), heavy)

	if seqOut == "" {
		t.Fatal("no output produced; the comparison is not exercising anything")
	}
	if seqOut != parOut {
		t.Errorf("stdout differs with kubernetes/mask/neat/filter options set (%d vs %d bytes)",
			len(seqOut), len(parOut))
	}
	if seqErr != parErr {
		t.Errorf("stderr differs: sequential %q, parallel %q", seqErr, parErr)
	}
	if seqCode != parCode {
		t.Errorf("exit code differs: sequential %d, parallel %d", seqCode, parCode)
	}
}

// TestRunDirectory_ParallelIsDeterministic guards against completion order
// leaking into the output: repeated parallel runs over identical input must
// produce identical bytes.
func TestRunDirectory_ParallelIsDeterministic(t *testing.T) {
	withGOMAXPROCS(t, testWorkers)

	wantOut, wantErr, wantCode := runDirectoryCapture(t, "detailed", buildParallelPairs(60))
	for i := range 12 {
		gotOut, gotErr, gotCode := runDirectoryCapture(t, "detailed", buildParallelPairs(60))
		if gotOut != wantOut {
			t.Fatalf("run %d produced different stdout than run 0", i+1)
		}
		if gotErr != wantErr {
			t.Fatalf("run %d produced different stderr than run 0: %q vs %q", i+1, gotErr, wantErr)
		}
		if gotCode != wantCode {
			t.Fatalf("run %d produced exit code %d, want %d", i+1, gotCode, wantCode)
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
	seqOut, seqErr, seqCode := runDirectoryCapture(t, "detailed", pairs)

	withGOMAXPROCS(t, testWorkers)
	parOut, parErr, parCode := runDirectoryCapture(t, "detailed", pairs)

	if seqErr != parErr {
		t.Errorf("stderr order differs between paths\nsequential:\n%s\nparallel:\n%s", seqErr, parErr)
	}
	if seqOut != parOut {
		t.Error("stdout differs between sequential and parallel paths when some pairs fail")
	}
	if seqCode != parCode {
		t.Errorf("exit code differs: sequential %d, parallel %d", seqCode, parCode)
	}

	// Errors must appear in pair order, matching the sorted pair plan.
	lastIdx := -1
	for _, name := range badNames {
		idx := strings.Index(parErr, name)
		if idx < 0 {
			t.Fatalf("expected an error mentioning %s, got:\n%s", name, parErr)
		}
		if idx < lastIdx {
			t.Errorf("error for %s reported out of pair order", name)
		}
		lastIdx = idx
	}

	// A healthy pair on either side of a failing one still produces output.
	for _, name := range []string{"file004.yaml", "file006.yaml"} {
		if !strings.Contains(parOut, name) {
			t.Errorf("expected output for %s alongside failing pairs", name)
		}
	}
}

// TestDirCompareWorkers checks both halves of the streaming/parallel decision:
// the count is capped by the pair count so a single pair never spawns workers
// and never exceeds GOMAXPROCS, and the parallel flag agrees with it.
func TestDirCompareWorkers(t *testing.T) {
	withGOMAXPROCS(t, testWorkers)

	tests := []struct {
		pairCount    int
		jobs         int
		want         int
		wantParallel bool
	}{
		{0, 0, 0, false},
		{1, 0, 1, false}, // a single pair has nothing to overlap with
		{2, 0, 2, true},
		{testWorkers, 0, testWorkers, true},
		{100, 0, testWorkers, true},                   // capped at GOMAXPROCS
		{100, 1, 1, false},                            // --jobs 1: the caller asked for streaming
		{100, 2, 2, true},                             // --jobs below GOMAXPROCS wins
		{100, testWorkers * 4, testWorkers * 4, true}, // --jobs above GOMAXPROCS is honored
		{3, testWorkers * 4, 3, true},                 // ...but never exceeds the pair count
		{1, 8, 1, false},                              // one pair stays sequential whatever --jobs says
	}
	for _, tt := range tests {
		got, parallel := dirCompareWorkers(tt.pairCount, tt.jobs)
		if got != tt.want || parallel != tt.wantParallel {
			t.Errorf("dirCompareWorkers(%d, jobs=%d) = (%d, %t), want (%d, %t)",
				tt.pairCount, tt.jobs, got, parallel, tt.want, tt.wantParallel)
		}
	}

	// One usable CPU: plenty of pairs, but still nothing to gain.
	withGOMAXPROCS(t, 1)
	if got, parallel := dirCompareWorkers(100, 0); got != 1 || parallel {
		t.Errorf("with GOMAXPROCS=1, dirCompareWorkers(100, 0) = (%d, %t), want (1, false)", got, parallel)
	}
	// ...unless --jobs asks for more anyway: worker count is not capped by the
	// CPU count, and oversubscribing is the caller's call to make.
	if got, parallel := dirCompareWorkers(100, 4); got != 4 || !parallel {
		t.Errorf("with GOMAXPROCS=1, dirCompareWorkers(100, 4) = (%d, %t), want (4, true)", got, parallel)
	}
}

// TestRunDirectory_JobsOneForcesStreaming covers --jobs as the memory escape
// hatch: with CPUs available it must still take the sequential path, and produce
// exactly what the parallel path produces.
func TestRunDirectory_JobsOneForcesStreaming(t *testing.T) {
	withGOMAXPROCS(t, testWorkers)

	parOut, parErr, parCode := runDirectoryCapture(t, "detailed", buildParallelPairs(60))
	seqOut, seqErr, seqCode := runDirectoryCapture(t, "detailed", buildParallelPairs(60),
		func(cfg *CLIConfig) { cfg.Jobs = 1 })

	if seqOut != parOut {
		t.Errorf("--jobs 1 stdout differs from the parallel path (%d vs %d bytes)", len(seqOut), len(parOut))
	}
	if seqErr != parErr {
		t.Errorf("--jobs 1 stderr differs: %q vs %q", seqErr, parErr)
	}
	if seqCode != parCode {
		t.Errorf("--jobs 1 exit code differs: %d vs %d", seqCode, parCode)
	}
}

// TestRunDirectory_JobsCapsWorkers exercises a bounded worker count end to end,
// since dirCompareWorkers is only half the contract: processDirPairs has to
// cover all pairs with fewer workers than pairs, in order.
func TestRunDirectory_JobsCapsWorkers(t *testing.T) {
	withGOMAXPROCS(t, testWorkers)

	wantOut, wantErr, wantCode := runDirectoryCapture(t, "detailed", buildParallelPairs(60),
		func(cfg *CLIConfig) { cfg.Jobs = 1 })

	for _, jobs := range []int{2, 3, testWorkers * 4} {
		gotOut, gotErr, gotCode := runDirectoryCapture(t, "detailed", buildParallelPairs(60),
			func(cfg *CLIConfig) { cfg.Jobs = jobs })
		if gotOut != wantOut {
			t.Errorf("--jobs %d stdout differs from --jobs 1", jobs)
		}
		if gotErr != wantErr {
			t.Errorf("--jobs %d stderr differs from --jobs 1: %q vs %q", jobs, gotErr, wantErr)
		}
		if gotCode != wantCode {
			t.Errorf("--jobs %d exit code %d, want %d", jobs, gotCode, wantCode)
		}
	}
}

// TestRunDirectory_SinglePairUsesStreamingPath confirms a one-pair run still
// works when dirCompareWorkers sends it down the streaming path.
func TestRunDirectory_SinglePairUsesStreamingPath(t *testing.T) {
	// GOMAXPROCS is high, so only the pair-count cap can force streaming here.
	withGOMAXPROCS(t, testWorkers)

	if _, parallel := dirCompareWorkers(1, 0); parallel {
		t.Fatal("expected a single pair to take the streaming path")
	}

	stdout, _, code := runDirectoryCapture(t, "detailed", map[string][2][]byte{
		"only.yaml": {[]byte("a: 1\n"), []byte("a: 2\n")},
	})
	if code != ExitCodeDifferences {
		t.Errorf("expected exit %d, got %d", ExitCodeDifferences, code)
	}
	if !strings.Contains(stdout, "only.yaml") {
		t.Errorf("expected output for only.yaml, got: %q", stdout)
	}
}
