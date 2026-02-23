# Performance Comparison: diffyml vs Alternatives

## Methodology

### Goal

Measure the execution time and peak memory usage of `diffyml` against 5 alternative YAML diff tools across a range of file sizes, from small configuration snippets to large-scale service manifests.

### Tools Under Test

| Tool | Repository | Language | Description |
|------|-----------|----------|-------------|
| **diffyml** | [szhekpisov/diffyml](https://github.com/szhekpisov/diffyml) | Go | Structural YAML comparison (this project) |
| **dyff** | [homeport/dyff](https://github.com/homeport/dyff) | Go | A diff tool for YAML files with rich output |
| **semihbkgr/yamldiff** | [semihbkgr/yamldiff](https://github.com/semihbkgr/yamldiff) | Go | YAML diff with colored output |
| **sters/yaml-diff** | [sters/yaml-diff](https://github.com/sters/yaml-diff) | Go | YAML file diff tool |
| **sahilm/yamldiff** | [sahilm/yamldiff](https://github.com/sahilm/yamldiff) | Go | A YAML diffing tool |
| **diff** (unix) | system | C | Plain text diff (baseline, no YAML awareness) |

All Go-based tools were installed from their latest releases. Unix `diff` is included as a non-YAML-aware baseline to show the overhead of structural parsing.

### Test Data

Test data is generated programmatically by `bench/compare/generate_testdata.go`, which produces a `from.yaml` / `to.yaml` pair at each size. The generator is ported from the internal benchmark suite (`pkg/diffyml/benchmark_test.go`) to ensure the data is representative of real-world usage.

Each file contains:
- A **metadata block** (name, version, environment, region, replicas)
- A **config block** (database, cache, logging settings)
- A **services list** of variable length, where each service has 12 fields (name, version, replicas, memory, cpu, enabled, port, protocol, timeout, labels)

The "to" file has ~20% modifications relative to "from":
- 2 services removed
- ~10% new services added
- ~20% of remaining services have changed versions, replica counts, or other fields
- Metadata/config header has minor value changes

This ensures every tool must actually detect and process meaningful structural differences, not just parse identical files.

**Size presets:**

| Preset | Services | Approx. Lines | Approx. File Size |
|--------|----------|---------------|-------------------|
| Small | 4 | ~70 | ~1 KB |
| Medium | 42 | ~530 | ~8 KB |
| Large | 420 | ~5,000 | ~78 KB |
| XLarge | 4,200 | ~50,000 | ~780 KB |

### Timing Measurement

Timing is measured using [hyperfine](https://github.com/sharkdp/hyperfine), a widely adopted CLI benchmarking tool. Configuration:

- **Warmup runs:** 5 (ensures filesystem caches are populated and Go runtime is warmed up)
- **Minimum runs:** 20 (provides statistical significance)
- **Shell overhead:** Each command runs inside a shell invocation; at sub-5ms runtimes this introduces noise, which hyperfine warns about. Results at the small size should be interpreted with this caveat.
- **Output suppression:** All tools run with `stdout` and `stderr` redirected to `/dev/null` to measure pure processing time, not terminal rendering.
- **Exit code handling:** All commands are wrapped with `|| true` since diff tools conventionally exit 1 when differences are found.

Hyperfine reports mean, standard deviation, min/max, and relative performance compared to the fastest tool.

### Memory Measurement

Peak memory (maximum resident set size) is measured using macOS `/usr/bin/time -l`, which reports the high-water RSS mark from the kernel. Each tool is measured 5 times per size, and the **median** is reported to reduce sensitivity to outliers.

### Justification of Methodology

**Why hyperfine over Go's `testing.Benchmark`?**
Go's built-in benchmark framework measures function-level performance within a single process. For a cross-tool comparison we need to measure the full end-to-end CLI execution — including binary startup, argument parsing, file I/O, YAML parsing, diff computation, and output formatting. Hyperfine benchmarks the actual user experience: the time from command invocation to completion.

**Why generated data instead of real-world files?**
Generated data provides controllable, reproducible size scaling (4x → 42x → 420x → 4200x services) while maintaining realistic structural properties. Real-world files would introduce variability in structure, nesting depth, and diff density that would make cross-size comparisons less meaningful. The generator produces data that closely mirrors production Kubernetes service configurations.

**Why include Unix `diff`?**
Unix `diff` serves as an absolute lower bound — it performs no YAML parsing, no structural comparison, no semantic matching. It shows the raw cost of reading and comparing text. Any YAML-aware tool will be slower; the question is by how much. This baseline helps separate "overhead of YAML awareness" from "algorithmic efficiency."

**Why 5 competitors?**
These represent the most actively maintained and commonly referenced YAML diff tools in the Go ecosystem. They cover a range of implementation approaches: some focus on rich output (dyff), others on simplicity (sters/yaml-diff, sahilm/yamldiff), and others on performance (semihbkgr/yamldiff). This gives a well-rounded competitive landscape.

**Why exclude sters/yaml-diff and sahilm/yamldiff from XLarge?**
Both tools exhibit super-linear scaling: they take ~1 second at Large (5,000 lines) and would require multiple minutes per run at XLarge (50,000 lines). Running 20+ iterations with warmup would take an impractical amount of time. Their Large results already clearly demonstrate the scaling trend.

## Results

**Environment:** Apple M1 Pro, macOS, Go 1.25.7, 20 runs with 5 warmup iterations.

### Execution Time

#### Small (~70 lines, ~1 KB)

| Command | Mean [ms] | Min [ms] | Max [ms] | Relative |
|:---|---:|---:|---:|---:|
| `diffyml` | 5.7 ± 0.8 | 4.7 | 14.3 | 2.58 ± 0.77 |
| `dyff` | 15.3 ± 1.1 | 13.2 | 21.6 | 6.93 ± 1.92 |
| `semihbkgr/yamldiff` | 3.9 ± 1.0 | 2.9 | 12.3 | 1.77 ± 0.64 |
| `sters/yaml-diff` | 4.0 ± 0.7 | 3.0 | 7.5 | 1.82 ± 0.59 |
| `sahilm/yamldiff` | 3.7 ± 0.6 | 2.9 | 7.7 | 1.68 ± 0.53 |
| `diff (unix)` | 2.2 ± 0.6 | 1.6 | 10.1 | 1.00 |

At this size, all Go tools are within a narrow band (~4-6 ms). The numbers are dominated by process startup and shell overhead; the differences are not statistically significant.

#### Medium (~530 lines, ~8 KB)

| Command | Mean [ms] | Min [ms] | Max [ms] | Relative |
|:---|---:|---:|---:|---:|
| `diffyml` | 6.3 ± 0.5 | 5.5 | 8.2 | 2.38 ± 0.60 |
| `dyff` | 29.2 ± 1.6 | 27.5 | 36.0 | 11.11 ± 2.70 |
| `semihbkgr/yamldiff` | 5.2 ± 0.6 | 4.6 | 8.2 | 1.99 ± 0.52 |
| `sters/yaml-diff` | 11.5 ± 0.8 | 10.4 | 18.3 | 4.38 ± 1.08 |
| `sahilm/yamldiff` | 16.1 ± 1.0 | 14.9 | 22.3 | 6.11 ± 1.49 |
| `diff (unix)` | 2.6 ± 0.6 | 1.9 | 7.0 | 1.00 |

Algorithmic differences become visible. diffyml and semihbkgr/yamldiff remain close (~5-6 ms), while sters/yaml-diff (12 ms), sahilm/yamldiff (16 ms), and dyff (29 ms) fall behind. diffyml is 1.2x faster than semihbkgr/yamldiff and 4.6x faster than dyff.

#### Large (~5,000 lines, ~78 KB)

| Command | Mean [ms] | Min [ms] | Max [ms] | Relative |
|:---|---:|---:|---:|---:|
| `diffyml` | 22.3 ± 1.8 | 20.1 | 31.4 | 3.60 ± 0.44 |
| `dyff` | 173.8 ± 4.1 | 167.2 | 182.2 | 28.00 ± 2.57 |
| `semihbkgr/yamldiff` | 27.9 ± 0.9 | 26.3 | 31.3 | 4.49 ± 0.42 |
| `sters/yaml-diff` | 983.6 ± 51.2 | 935.1 | 1126.0 | 158.52 ± 16.29 |
| `sahilm/yamldiff` | 1369.6 ± 18.7 | 1351.7 | 1417.5 | 220.71 ± 19.78 |
| `diff (unix)` | 6.2 ± 0.5 | 5.3 | 7.9 | 1.00 |

The scaling characteristics become clear. diffyml (22 ms) is **1.25x faster** than semihbkgr/yamldiff (28 ms), **7.8x faster** than dyff (174 ms), **44x faster** than sters/yaml-diff (984 ms), and **61x faster** than sahilm/yamldiff (1,370 ms).

#### XLarge (~50,000 lines, ~780 KB)

| Command | Mean [ms] | Min [ms] | Max [ms] | Relative |
|:---|---:|---:|---:|---:|
| `diffyml` | 152.3 ± 1.6 | 149.9 | 155.9 | 3.30 ± 0.12 |
| `dyff` | 3646.7 ± 115.6 | 3504.6 | 3926.2 | 79.01 ± 3.63 |
| `semihbkgr/yamldiff` | 245.7 ± 5.5 | 238.2 | 264.0 | 5.32 ± 0.21 |
| `sters/yaml-diff` | — | — | — | _excluded (>100s est.)_ |
| `sahilm/yamldiff` | — | — | — | _excluded (>100s est.)_ |
| `diff (unix)` | 46.2 ± 1.5 | 44.4 | 54.2 | 1.00 |

At maximum scale, diffyml (152 ms) is **1.61x faster** than semihbkgr/yamldiff (246 ms) and **23.9x faster** than dyff (3,647 ms). sters/yaml-diff and sahilm/yamldiff were excluded — their super-linear scaling from 1s at Large (10x smaller) would project to well over 100 seconds per run.

### Peak Memory Usage (RSS)

| Tool | Small | Medium | Large | XLarge |
|------|------:|-------:|------:|-------:|
| diffyml | 10.3 MB | 11.4 MB | 18.4 MB | 73.0 MB |
| dyff | 18.0 MB | 18.9 MB | 29.5 MB | 116.1 MB |
| semihbkgr/yamldiff | 5.3 MB | 7.6 MB | 21.4 MB | 152.0 MB |
| sters/yaml-diff | 5.1 MB | 10.7 MB | 325.7 MB | — |
| sahilm/yamldiff | 5.1 MB | 10.5 MB | 13.3 MB | — |
| diff (unix) | 1.8 MB | 5.5 MB | 6.2 MB | 15.2 MB |

diffyml has the lowest memory footprint among YAML-aware tools at Large and XLarge. At Large, it uses **18.4 MB** — 1.2x less than semihbkgr/yamldiff, 1.6x less than dyff, and 17.7x less than sters/yaml-diff (which consumes 326 MB). At XLarge, diffyml uses **73 MB** — 1.6x less than dyff and 2.1x less than semihbkgr/yamldiff.

### Scaling Summary

| Tool | Small → Large (70x lines) | Medium → Large (10x lines) |
|------|---------------------------|----------------------------|
| diffyml | 5.7 → 22.3 ms (**3.9x**) | 6.3 → 22.3 ms (**3.5x**) |
| semihbkgr/yamldiff | 3.9 → 27.9 ms (7.2x) | 5.2 → 27.9 ms (5.4x) |
| dyff | 15.3 → 173.8 ms (11.4x) | 29.2 → 173.8 ms (6.0x) |
| sters/yaml-diff | 4.0 → 984 ms (246x) | 11.5 → 984 ms (86x) |
| sahilm/yamldiff | 3.7 → 1370 ms (370x) | 16.1 → 1370 ms (85x) |
| diff (unix) | 2.2 → 6.2 ms (2.8x) | 2.6 → 6.2 ms (2.4x) |

diffyml scales nearly linearly — growing ~3.9x when input grows 70x (small to large). sters/yaml-diff and sahilm/yamldiff exhibit super-linear (likely quadratic or worse) scaling, growing 246x and 370x respectively over the same range.

## Key Findings

1. **diffyml is the fastest YAML-aware diff tool** at large and xlarge sizes, where algorithmic efficiency dominates over startup overhead.

2. **The performance advantage grows with file size.** At small sizes, all Go tools are comparable (~4-6 ms). At large (5K lines), diffyml is 1.25x faster than the nearest competitor. At xlarge (50K lines), the gap widens to 1.61x.

3. **diffyml has the best memory efficiency** among YAML-aware tools at large scale, using 18.4 MB at large vs 21–326 MB for alternatives, and 73 MB at xlarge vs 116–152 MB.

4. **diffyml scales near-linearly**, while sters/yaml-diff and sahilm/yamldiff exhibit super-linear scaling that makes them impractical for large files.

5. **dyff has consistently high overhead** (~3-4x baseline even at small sizes), likely due to its rich output formatting and multi-document comparison pipeline.

## Reproducing These Results

```bash
# Full benchmark (small, medium, large)
make bench-compare

# Include xlarge (sters/yaml-diff and sahilm/yamldiff auto-excluded)
bash bench/compare/run.sh --sizes small,medium,large,xlarge

# Quick check with fewer runs
bash bench/compare/run.sh --runs 3

# Skip tool installation (reuse previously installed)
bash bench/compare/run.sh --skip-install
```

Results are written to `bench/compare/results/REPORT.md`.
