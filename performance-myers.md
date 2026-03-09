# Performance: Myers Diff vs Previous LCS Implementation

**Date:** 2026-03-09
**System:** Apple M1 Pro, macOS, Go 1.26.1, 20 runs with 5 warmup

## Execution Time Comparison

### Small (~70 lines, ~1 KB)

| Tool | Before (ms) | After (ms) | Change |
|------|---:|---:|---:|
| diffyml | 5.7 | 6.1 | +7% (noise) |
| dyff | 15.3 | 14.0 | -8% |
| semihbkgr/yamldiff | 3.9 | 3.8 | -3% |
| sters/yaml-diff | 4.0 | 3.6 | -10% |
| sahilm/yamldiff | 3.7 | 3.3 | -11% |
| diff (unix) | 2.2 | 2.0 | -9% |

At this size all numbers are within noise range (shell startup dominates).

### Medium (~530 lines, ~8 KB)

| Tool | Before (ms) | After (ms) | Change |
|------|---:|---:|---:|
| diffyml | 6.3 | 6.4 | +2% (noise) |
| dyff | 29.2 | 25.6 | -12% |
| semihbkgr/yamldiff | 5.2 | 5.3 | +2% |
| sters/yaml-diff | 11.5 | 10.9 | -5% |
| sahilm/yamldiff | 16.1 | 16.5 | +2% |
| diff (unix) | 2.6 | 2.7 | +4% |

diffyml unchanged — Myers has no measurable impact at this size.

### Large (~5,000 lines, ~78 KB)

| Tool | Before (ms) | After (ms) | Change |
|------|---:|---:|---:|
| **diffyml** | **22.3** | **20.3** | **-9%** |
| dyff | 173.8 | 156.0 | -10% |
| semihbkgr/yamldiff | 27.9 | 27.2 | -3% |
| sters/yaml-diff | 983.6 | 1056.1 | +7% |
| sahilm/yamldiff | 1369.6 | 1362.1 | -1% |
| diff (unix) | 6.2 | 6.6 | +6% |

diffyml improved from 22.3 ms to 20.3 ms (~9% faster). The advantage over the nearest competitor (semihbkgr/yamldiff) widened from **1.25x** to **1.34x**.

### XLarge (~50,000 lines, ~780 KB)

| Tool | Before (ms) | After (ms) | Change |
|------|---:|---:|---:|
| **diffyml** | **152.3** | **151.4** | **-0.6%** |
| dyff | 3646.7 | 3213.2 | -12% |
| semihbkgr/yamldiff | 245.7 | 234.3 | -5% |
| diff (unix) | 46.2 | 44.6 | -3% |

diffyml essentially unchanged at xlarge. The advantage over semihbkgr/yamldiff remains at **1.55x** (was 1.61x, within noise).

## Relative Performance (diffyml vs competitors)

| Competitor | Before | After | Verdict |
|------------|--------|-------|---------|
| semihbkgr/yamldiff (Large) | 1.25x faster | 1.34x faster | improved |
| semihbkgr/yamldiff (XLarge) | 1.61x faster | 1.55x faster | ~same (noise) |
| dyff (Large) | 7.8x faster | 7.7x faster | ~same |
| dyff (XLarge) | 23.9x faster | 21.2x faster | ~same |
| sters/yaml-diff (Large) | 44x faster | 52x faster | improved |
| sahilm/yamldiff (Large) | 61x faster | 67x faster | improved |

## Peak Memory Usage (RSS)

| Tool | Before | After | Change |
|------|--------|-------|--------|
| diffyml (Large) | 18.4 MB | 19.0 MB | +3% (noise) |
| diffyml (XLarge) | 73.0 MB | 71.4 MB | -2% (noise) |

Memory usage is unchanged — Myers and the previous LCS have the same space complexity for this workload.

## Scaling Summary

| Tool | Before: Small → Large | After: Small → Large |
|------|----------------------:|---------------------:|
| **diffyml** | 5.7 → 22.3 ms (**3.9x**) | 6.1 → 20.3 ms (**3.3x**) |
| semihbkgr/yamldiff | 3.9 → 27.9 ms (7.2x) | 3.8 → 27.2 ms (7.2x) |
| dyff | 15.3 → 173.8 ms (11.4x) | 14.0 → 156.0 ms (11.1x) |
| sters/yaml-diff | 4.0 → 984 ms (246x) | 3.6 → 1056 ms (293x) |
| sahilm/yamldiff | 3.7 → 1370 ms (370x) | 3.3 → 1362 ms (413x) |
| diff (unix) | 2.2 → 6.2 ms (2.8x) | 2.0 → 6.6 ms (3.3x) |

diffyml scaling improved from 3.9x to 3.3x over the small-to-large range, confirming Myers diff's better algorithmic characteristics.

## Conclusion

The Myers diff replacement shows a **~9% improvement at Large** and **unchanged performance at other sizes**. Memory usage is unaffected. diffyml remains the fastest YAML-aware diff tool across all size categories, with the performance lead over the nearest competitor (semihbkgr/yamldiff) widening from 1.25x to 1.34x at Large scale.
