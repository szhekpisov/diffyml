# Mutation Testing Report

**Tool:** [gremlins](https://github.com/go-gremlins/gremlins)
**Last full run:** 2026-02-28 — efficacy 96.22% (535 killed / 556 covered)
**Line coverage:** 96.6% (`go test -cover ./pkg/diffyml/`)
**Mutator coverage:** 99.29%

## Summary

| Status | Count |
|--------|-------|
| Killed | 535 |
| Lived | 21 |
| Timed out | 4 |
| Not covered | 4 |
| **Efficacy** | **96.22%** |
| **Mutator coverage** | **99.29%** |

### Change log

- **Round 6** (2026-02-28): Killed 2 `IsTerminal` mutants (color.go:77,81) by
  introducing injectable `stdoutStatFn` and mock tests that simulate a real
  terminal (character device). LIVED 23 → 21, efficacy 95.86% → 96.22%.
- **Round 5** (2026-02-28): Fixed gremlins coverage discrepancy — converted 5
  `switch/case` statements to `if/else` chains in `detailed_formatter.go`,
  `summarizer.go`, `comparator.go`, and `directory.go`. Root cause: Go's
  `-coverprofile` instruments case bodies but not case conditions; gremlins
  checks mutation positions against coverage block boundaries and classified
  case-condition mutations as NOT COVERED. The `if/else` form places conditions
  inside coverage blocks. NOT COVERED 45 → 4 (remaining 4 are package-level
  constants), mutator coverage 91.96% → 99.29%. Two previously hidden equivalent
  mutants were exposed (LIVED 21 → 23).
- **Round 4** (2026-02-28): Killed 4 surviving mutants by fixing imprecise test
  assertions. Tests now exercise the correct type branches (nested maps for
  `extractPathOrder`, map values for `renderFirstKeyValueYAML`) and assert exact
  indentation and output exclusion. LIVED 25 → 21, efficacy 95.15% → 95.92%.
- **Round 3** (2026-02-28): Refactored `filter.go` to remove 12 redundant `len > 0`
  guards (equivalent mutants). Added 17 new mutation-targeted tests in
  `coverage_gaps_test.go`. Net result: LIVED 36 → 25, efficacy 93.23% → 95.15%.
- **Round 2**: Added coverage for plain-map branches, positional list bounds,
  `deepEqual` slices, `clamp` boundaries, error formatting edge cases, and more.
  Removed provably dead cross-type branches in `compareNodes`/`deepEqual`.
  Not-covered dropped from 60 → 45, line coverage rose from 92.3% → 96.6%.
- **Round 1**: Initial mutation testing setup with gremlins.

## Survived Mutants (21 LIVED)

All 23 surviving mutants are **equivalent** — the mutation does not change
observable program behavior, so no test can detect them.

---

### Pattern 1: `<` changed to `<=` in sort comparisons (5 mutants)

**File:** `diffyml.go`
**Mutation:** `CONDITIONALS_BOUNDARY` — `<` changed to `<=`

In `sortDiffsWithOrder`, each `<` comparison is guarded by a prior `!=` check
that ensures the operands are never equal. When they can't be equal, `<` and
`<=` behave identically.

| Line | Code | Why equivalent |
|------|------|----------------|
| 305:19 | `orderI < orderJ` | Guarded by `rootI != rootJ`; different roots get unique indices |
| 307:17 | `rootI < rootJ` | Guarded by `rootI != rootJ`; identical strings can't reach this line |
| 344:24 | `parentOrderI < parentOrderJ` | Guarded by `parentOrderI != parentOrderJ` on line 343 |
| 351:18 | `depthI < depthJ` | Guarded by `depthI != depthJ` on line 350 |
| 355:16 | `pathI < pathJ` | Diffs are grouped by path; two diffs can't have the same path |

---

### Pattern 2: `len(x) > 0` guards (3 mutants)

**Files:** `filter.go`
**Mutation:** `CONDITIONALS_BOUNDARY` — `> 0` changed to `>= 0`

| Line | Code | Why equivalent |
|------|------|----------------|
| 87:21 | `len(remaining) > 0` | `HasPrefix` guarantees `remaining` is non-empty when reached |
| 143:46 | `len(opts.IncludePaths) > 0` | Boolean OR; empty slice makes `hasIncludeFilters` true but filtering loop handles empties |
| 143:71 | `len(includeRegex) > 0` | Same — empty compiled slice handled by match functions |

---

### Pattern 3: Clamp boundary values (3 mutants)

**File:** `color.go`
**Mutation:** `CONDITIONALS_BOUNDARY`

| Line | Code | Why equivalent |
|------|------|----------------|
| 66:15 | `override < minTerminalWidth` → `<=` | When `override == min`, returns `min` either way |
| 237:9 | `val < min` → `<=` | Clamp: returns `min` when `val == min` either way |
| 240:9 | `val > max` → `>=` | Clamp: returns `max` when `val == max` either way |

---

### Pattern 4: `maxLen` and list-bounds boundaries in comparator (4 mutants)

**File:** `comparator.go`
**Mutation:** `CONDITIONALS_BOUNDARY`

| Line | Code | Why equivalent |
|------|------|----------------|
| 27:13 | `len(to) > maxLen` → `>=` | Sets `maxLen` to `len(to)` which already equals `maxLen` at boundary |
| 31:16 | `i < maxLen` → `<=` | Extra iteration with both nil docs is a no-op |
| 337:13 | `len(to) > maxLen` → `>=` | Same pattern as line 27 |
| 352:8 | `i >= len(from)` → `>` | At `i == len(from)`: `fromVal` is nil (prior `if i < len(from)` failed), so the `else` branch calls `compareNodes(path, nil, toVal)` which produces `DiffAdded` — same result |

---

### Pattern 5: Boundary in `len(path) > 1` (1 mutant)

**File:** `diffyml.go`
**Mutation:** `CONDITIONALS_BOUNDARY` at line 222:15 — `> 1` changed to `>= 1`

For a single-character path, `LastIndex` returns -1 (no dot), and the inner
`lastDot >= 0` check fails. The mutation allows entry but the inner guard
prevents any behavior change.

---

### Pattern 6: LCS tie-breaking (2 mutants)

**File:** `detailed_formatter.go`
**Mutation:** `CONDITIONALS_BOUNDARY`

When `dp[i-1][j] == dp[i][j-1]`, both branches assign the same value. The DP
table is identical regardless of which branch is taken.

| Line | Code | Why equivalent |
|------|------|----------------|
| 462:17 | `j <= n` → `j < n` | Inner loop boundary; at `j == n` the DP cell is already computed by the outer structure |
| 465:25 | `dp[i-1][j] >= dp[i][j-1]` → `>` | When equal, both branches assign the same max LCS value |

---

### Pattern 7: Array reverse self-swap (1 mutant)

**File:** `detailed_formatter.go`
**Mutation:** `CONDITIONALS_BOUNDARY` at line 491:41 — `<` changed to `<=`

When `left == right` (odd-length array midpoint), swapping an element with
itself is a no-op.

---

### Pattern 8: Map capacity hint (1 mutant)

**File:** `directory.go`
**Mutation:** `ARITHMETIC_BASE` at line 98:49 — `len(a) + len(b)` mutated

The capacity hint only affects initial memory allocation, not map behavior.

---

### Pattern 9: Flag parsing edge case (1 mutant)

**File:** `cli.go`
**Mutation:** `CONDITIONALS_BOUNDARY` at line 221:51 — `>= 0` changed to `> 0`

For `eqIdx == 0`, the flag name would be `""`. Either way, `fs.Lookup` returns
nil and the arg is treated as positional.

---

## Not Covered (4 mutants)

4 mutants remain NOT COVERED. All are `ARITHMETIC_BASE` mutations on
package-level constants. Constants are compile-time expressions that do not
appear as executable statements in Go's `-coverprofile`, so gremlins cannot
determine whether they are tested.

| File | Line | Constant |
|------|------|----------|
| `remote.go` | 14:23 | `MaxResponseSize = 10 * 1024 * 1024` |
| `remote.go` | 14:30 | `MaxResponseSize = 10 * 1024 * 1024` |
| `remote.go` | 16:22 | `DefaultTimeout = 30 * time.Second` |
| `summarizer.go` | 26:24 | `summaryTimeout = 30 * time.Second` |

These constants are exercised by unit tests (`TestRemoteConstants`,
`TestSummarize_Timeout`), but since Go does not instrument constant
declarations, they will always be reported as NOT COVERED by gremlins.

### Previously NOT COVERED (resolved in Round 5)

41 mutants were previously reported as NOT COVERED due to a Go coverage
instrumentation limitation: `switch/case` condition expressions fall outside
Go's coverage blocks (which only instrument case *bodies*, starting after the
`:`). Gremlins checks if a mutation's `line:column` falls within a covered
block, so mutations on case conditions were classified as NOT COVERED even
though the code was fully tested.

**Fix:** Converted 5 `switch` statements to `if/else` chains, which places
conditions inside coverage blocks. Of the 41 newly covered mutants, 39 were
KILLED and 2 became LIVED (newly exposed equivalent boundary mutants).
