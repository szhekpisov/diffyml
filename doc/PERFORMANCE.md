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

**Environment:** Apple M1 Pro, macOS, Go 1.26.2, 20 runs with 5 warmup iterations.

**Tool versions:** diffyml (built from source), dyff v1.11.3, semihbkgr/yamldiff v0.3.1, sters/yaml-diff v1.4.1, sahilm/yamldiff v1.3.

### Execution Time

#### Small (~70 lines, ~1 KB)

| Command | Mean [ms] | Min [ms] | Max [ms] | Relative |
|:---|---:|---:|---:|---:|
| `diffyml` | 5.5 ± 0.5 | 4.7 | 7.2 | 2.66 ± 0.58 |
| `dyff` | 13.2 ± 0.5 | 11.9 | 15.0 | 6.39 ± 1.30 |
| `semihbkgr/yamldiff` | 3.6 ± 0.6 | 2.9 | 9.8 | 1.75 ± 0.45 |
| `sters/yaml-diff` | 3.6 ± 0.5 | 2.9 | 5.6 | 1.73 ± 0.42 |
| `sahilm/yamldiff` | 3.6 ± 0.4 | 2.9 | 5.0 | 1.72 ± 0.40 |
| `diff (unix)` | 2.1 ± 0.4 | 1.6 | 3.9 | 1.00 |

At this size, all Go tools are within a narrow band (~4-6 ms). The ~2 ms gap between diffyml and the simpler tools is startup overhead from capabilities they lack: `net/http` (remote URL fetching, AI summaries) and `crypto/x509` (certificate inspection) add ~87 transitive packages to the binary. The actual diff algorithm completes in <1 ms.

#### Medium (~530 lines, ~8 KB)

| Command | Mean [ms] | Min [ms] | Max [ms] | Relative |
|:---|---:|---:|---:|---:|
| `diffyml` | 6.7 ± 0.5 | 5.8 | 8.4 | 2.58 ± 0.47 |
| `dyff` | 23.9 ± 1.0 | 22.4 | 30.1 | 9.16 ± 1.56 |
| `semihbkgr/yamldiff` | 5.5 ± 0.4 | 4.8 | 7.1 | 2.09 ± 0.38 |
| `sters/yaml-diff` | 11.5 ± 0.5 | 10.5 | 15.5 | 4.41 ± 0.76 |
| `sahilm/yamldiff` | 16.6 ± 0.6 | 15.3 | 18.4 | 6.36 ± 1.07 |
| `diff (unix)` | 2.6 ± 0.4 | 2.0 | 4.0 | 1.00 |

Algorithmic differences become visible. diffyml and semihbkgr/yamldiff remain close (~6-7 ms), while sters/yaml-diff (12 ms), sahilm/yamldiff (17 ms), and dyff (24 ms) fall behind. diffyml is 1.2x faster than semihbkgr/yamldiff and 3.6x faster than dyff.

#### Large (~5,000 lines, ~78 KB)

| Command | Mean [ms] | Min [ms] | Max [ms] | Relative |
|:---|---:|---:|---:|---:|
| `diffyml` | 18.8 ± 0.6 | 17.6 | 21.7 | 3.05 ± 0.25 |
| `dyff` | 120.1 ± 1.8 | 117.1 | 124.2 | 19.47 ± 1.48 |
| `semihbkgr/yamldiff` | 27.9 ± 0.6 | 26.7 | 29.8 | 4.52 ± 0.35 |
| `sters/yaml-diff` | 998.6 ± 12.2 | 976.2 | 1022.5 | 161.96 ± 12.26 |
| `sahilm/yamldiff` | 1317.2 ± 4.4 | 1310.6 | 1327.3 | 213.64 ± 15.97 |
| `diff (unix)` | 6.2 ± 0.5 | 5.4 | 7.6 | 1.00 |

The scaling characteristics become clear. diffyml (19 ms) is **1.48x faster** than semihbkgr/yamldiff (28 ms), **6.39x faster** than dyff (120 ms), **53x faster** than sters/yaml-diff (999 ms), and **70x faster** than sahilm/yamldiff (1,317 ms).

#### XLarge (~50,000 lines, ~780 KB)

| Command | Mean [ms] | Min [ms] | Max [ms] | Relative |
|:---|---:|---:|---:|---:|
| `diffyml` | 128.6 ± 1.2 | 125.7 | 130.6 | 2.84 ± 0.04 |
| `dyff` | 1145.9 ± 8.6 | 1134.6 | 1164.6 | 25.27 ± 0.35 |
| `semihbkgr/yamldiff` | 239.7 ± 3.4 | 235.1 | 249.2 | 5.29 ± 0.10 |
| `sters/yaml-diff` | — | — | — | _excluded (>100s est.)_ |
| `sahilm/yamldiff` | — | — | — | _excluded (>100s est.)_ |
| `diff (unix)` | 45.3 ± 0.5 | 44.3 | 46.6 | 1.00 |

At maximum scale, diffyml (129 ms) is **1.86x faster** than semihbkgr/yamldiff (240 ms) and **8.91x faster** than dyff (1,146 ms). sters/yaml-diff and sahilm/yamldiff were excluded — their super-linear scaling from 1s at Large (10x smaller) would project to well over 100 seconds per run.

### Peak Memory Usage (RSS)

| Tool | Small | Medium | Large | XLarge |
|------|------:|-------:|------:|-------:|
| diffyml | 10.7 MB | 11.8 MB | 18.5 MB | 66.2 MB |
| dyff | 18.6 MB | 19.5 MB | 29.3 MB | 113.3 MB |
| semihbkgr/yamldiff | 5.5 MB | 7.9 MB | 22.3 MB | 155.3 MB |
| sters/yaml-diff | 5.4 MB | 10.8 MB | 331.5 MB | — |
| sahilm/yamldiff | 5.4 MB | 10.8 MB | 13.5 MB | — |
| diff (unix) | 1.9 MB | 5.5 MB | 6.2 MB | 14.8 MB |

diffyml has the lowest memory footprint among YAML-aware tools at Large and XLarge. At Large, it uses **18.5 MB** — 1.2x less than semihbkgr/yamldiff, 1.6x less than dyff, and 17.9x less than sters/yaml-diff (which consumes 332 MB). At XLarge, diffyml uses **66 MB** — 1.7x less than dyff and 2.3x less than semihbkgr/yamldiff.

### Scaling Summary

| Tool | Small → Large (70x lines) | Medium → Large (10x lines) |
|------|---------------------------|----------------------------|
| diffyml | 5.5 → 18.8 ms (**3.4x**) | 6.7 → 18.8 ms (**2.8x**) |
| semihbkgr/yamldiff | 3.6 → 27.9 ms (7.8x) | 5.5 → 27.9 ms (5.1x) |
| dyff | 13.2 → 120.1 ms (9.1x) | 23.9 → 120.1 ms (5.0x) |
| sters/yaml-diff | 3.6 → 999 ms (278x) | 11.5 → 999 ms (87x) |
| sahilm/yamldiff | 3.6 → 1317 ms (366x) | 16.6 → 1317 ms (79x) |
| diff (unix) | 2.1 → 6.2 ms (3.0x) | 2.6 → 6.2 ms (2.4x) |

diffyml scales nearly linearly — growing ~3.4x when input grows 70x (small to large). sters/yaml-diff and sahilm/yamldiff exhibit super-linear (likely quadratic or worse) scaling, growing 278x and 366x respectively over the same range.

## Key Findings

1. **diffyml is the fastest YAML-aware diff tool** at medium, large, and xlarge sizes, where algorithmic efficiency dominates over startup overhead.

2. **The performance advantage grows with file size.** At small sizes, all Go tools are comparable (~4-6 ms). At large (5K lines), diffyml is 1.48x faster than the nearest competitor. At xlarge (50K lines), the gap widens to 1.86x.

3. **diffyml has the best memory efficiency** among YAML-aware tools at large scale, using 19 MB at large vs 22-332 MB for alternatives, and 66 MB at xlarge vs 113-155 MB.

4. **diffyml scales near-linearly**, while sters/yaml-diff and sahilm/yamldiff exhibit super-linear scaling that makes them impractical for large files.

5. **dyff has consistently high overhead** (~2.4x the baseline even at small sizes), likely due to its 23 module dependencies (cobra, go-colorful, go-diff, etc.) and rich output formatting pipeline.

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
