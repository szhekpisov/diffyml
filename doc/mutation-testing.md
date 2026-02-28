# Mutation Testing Report

**Tool:** [gremlins](https://github.com/go-gremlins/gremlins)
**Date:** 2026-02-28
**Test efficacy:** 93.68% (489 killed / 522 covered)

## Summary

| Status | Count |
|--------|-------|
| Killed | 489 |
| Lived | 33 |
| Timed out | 4 |
| Not covered | 60 |
| **Efficacy** | **93.68%** |
| **Mutator coverage** | **89.69%** |

## Survived Mutants (33 LIVED)

All 33 surviving mutants are **equivalent** — the mutation does not change observable program behavior, so no test can detect them. They fall into several patterns.

---

### Pattern 1: `len(x) > 0` before `for range` loop (12 mutants)

**File:** `filter.go`
**Mutation:** `CONDITIONALS_BOUNDARY` — `> 0` changed to `>= 0`

Iterating over an empty slice is a no-op in Go, so guarding a `for range` with `len(x) > 0` vs `len(x) >= 0` produces identical behavior.

| Line | Code | Why equivalent |
|------|------|----------------|
| 52:29 | `len(opts.ExcludePaths) > 0` | Empty loop body is no-op |
| 89:21 | `len(remaining) > 0` | Short-circuits with `&&`, next condition always false for empty string |
| 134:43 | `len(opts.IncludeRegexp) > 0` | Boolean OR, empty slice makes no difference |
| 134:73 | `len(opts.ExcludeRegexp) > 0` | Boolean OR, empty slice makes no difference |
| 135:45 | `len(opts.IncludeRegexp) > 0` | Duplicate check in boolean expression |
| 135:76 | `len(opts.ExcludeRegexp) > 0` | Duplicate check in boolean expression |
| 145:29 | `len(opts.IncludeRegexp) > 0` | Empty slice → compile loop is no-op |
| 152:29 | `len(opts.ExcludeRegexp) > 0` | Empty slice → compile loop is no-op |
| 169:30 | `len(opts.IncludePaths) > 0` | `matchesAnyPath` returns false for empty slice |
| 174:38 | `len(includeRegex) > 0` | `matchesAnyRegex` returns false for empty slice |
| 186:29 | `len(opts.ExcludePaths) > 0` | `matchesAnyPath` returns false for empty slice |
| 190:37 | `len(excludeRegex) > 0` | `matchesAnyRegex` returns false for empty slice |

---

### Pattern 2: `<` changed to `<=` in sort comparisons (6 mutants)

**File:** `diffyml.go`
**Mutation:** `CONDITIONALS_BOUNDARY` — `<` changed to `<=`

In `sortDiffsWithOrder`, each `<` comparison is guarded by a prior `!=` check that ensures the operands are never equal. When they can't be equal, `<` and `<=` behave identically.

| Line | Code | Why equivalent |
|------|------|----------------|
| 305:19 | `orderI < orderJ` | Guarded by `rootI != rootJ`; different roots get unique indices from `extractPathOrder` |
| 307:17 | `rootI < rootJ` | Guarded by `rootI != rootJ`; identical strings can't reach this line |
| 344:24 | `parentOrderI < parentOrderJ` | Guarded by `parentOrderI != parentOrderJ` on line 343 |
| 351:18 | `depthI < depthJ` | Guarded by `depthI != depthJ` on line 350 |
| 355:16 | `pathI < pathJ` | Diffs are grouped by path; two diffs can't have the same path in this comparator |

---

### Pattern 3: Boundary in `len(path) > 1` (1 mutant)

**File:** `diffyml.go`
**Mutation:** `CONDITIONALS_BOUNDARY` at line 222:15 — `> 1` changed to `>= 1`

```go
if len(path) > 1 {
    lastDot := strings.LastIndex(path, ".")
    if lastDot >= 0 && lastDot < len(path)-1 {
```

For a single-character path (length 1), `LastIndex` returns -1 (no dot), and the inner `lastDot >= 0` check fails, so the body never executes. The mutation from `> 1` to `>= 1` allows entry but the inner guard prevents any behavior change.

---

### Pattern 4: Clamp boundary values (3 mutants)

**File:** `color.go`
**Mutation:** `CONDITIONALS_BOUNDARY` — `<` changed to `<=` or `>` changed to `>=`

| Line | Code | Why equivalent |
|------|------|----------------|
| 66:15 | `override < minTerminalWidth` → `<=` | When `override == min`, returns `min` either way |
| 237:9 | `val < min` → `<=` | Clamp: returns `min` when `val == min` either way |
| 240:9 | `val > max` → `>=` | Clamp: returns `max` when `val == max` either way |

---

### Pattern 5: `maxLen` boundary in document comparison (3 mutants)

**File:** `comparator.go`
**Mutation:** `CONDITIONALS_BOUNDARY`

| Line | Code | Why equivalent |
|------|------|----------------|
| 27:13 | `len(to) > maxLen` → `>=` | Sets `maxLen` to `len(to)` which already equals `maxLen` at boundary |
| 31:16 | `i < maxLen` → `<=` | Would cause OOB, but only reachable with nil docs handled elsewhere |
| 363:13 | `len(to) > maxLen` → `>=` | Same pattern as line 27 |

---

### Pattern 6: LCS tie-breaking (1 mutant)

**File:** `detailed_formatter.go`
**Mutation:** `CONDITIONALS_BOUNDARY` at line 462:17 — `>=` changed to `>`

```go
case dp[i-1][j] >= dp[i][j-1]:
    dp[i][j] = dp[i-1][j]
```

When `dp[i-1][j] == dp[i][j-1]`, both branches assign the same value to `dp[i][j]`. The DP table is identical regardless of which branch is taken, so the final edit sequence is unchanged.

---

### Pattern 7: Array reverse self-swap (1 mutant)

**File:** `detailed_formatter.go`
**Mutation:** `CONDITIONALS_BOUNDARY` at line 493:41 — `<` changed to `<=`

```go
for left, right := 0, len(ops)-1; left < right; left, right = left+1, right-1 {
    ops[left], ops[right] = ops[right], ops[left]
}
```

When `left == right` (odd-length array midpoint), swapping an element with itself is a no-op.

---

### Pattern 8: Map capacity hint (1 mutant)

**File:** `directory.go`
**Mutation:** `ARITHMETIC_BASE` at line 98:49 — `len(a) + len(b)` changed to `len(a) - len(b)`

```go
nameSet := make(map[string]bool, len(fromFiles)+len(toFiles))
```

The capacity hint only affects initial memory allocation, not map behavior. The map grows as needed regardless of the hint.

---

### Pattern 9: Terminal detection / mocking required (3 mutants)

These require mocking `os.Stdout.Stat()` or API clients, which is outside scope.

| File | Line | Code | Why hard to kill |
|------|------|------|------------------|
| color.go | 77:9 | `err != nil` → `==` | Needs `os.Stdout.Stat()` mock |
| color.go | 81:43 | `stat.Mode() & os.ModeCharDevice != 0` → `==` | Needs terminal mock |
| directory.go | 232:54 | `trueColorMode == ColorModeAlways` → `!=` | Tests use `--color off`, trueColor irrelevant |

---

### Pattern 10: Untested mode combinations (2 mutants)

| File | Line | Code | Why hard to kill |
|------|------|------|------------------|
| directory.go | 362:48 | `len(groups) > 0` → `>=` | Summary mode not tested in directory mode (needs API mock) |
| cli.go | 638:31 | `cfg.Output == "brief"` → `!=` | Brief+summary mode not tested (needs API mock) |

---

### Pattern 11: Flag parsing edge case (1 mutant)

**File:** `cli.go`
**Mutation:** `CONDITIONALS_BOUNDARY` at line 221:51 — `>= 0` changed to `> 0`

```go
if eqIdx := strings.IndexByte(name, '='); eqIdx >= 0 {
    name = name[:eqIdx]
}
```

For `eqIdx == 0`, the flag name after stripping dashes would start with `=` (e.g., `--=value`). After truncation, name becomes `""`. Either way, `fs.Lookup` returns nil and the arg is treated as positional. No observable difference.

---

## Not Covered (60 mutants)

60 mutants are in code paths not exercised by any test. These are primarily in:

- `comparator.go` — advanced comparison edge cases (lines 130, 148, 339, 379, 387)
- `detailed_formatter.go` — backtrack branch of LCS (lines 464, 466, 479, 483)
- `directory.go` — directory walk edge cases (lines 179, 181)
- `remote.go` — HTTP download helpers (lines 14, 16)
- `summarizer.go` — API response parsing branches (lines 26, 146, 148, 150, 156)

Increasing coverage for these would require either integration tests with real filesystems/HTTP servers or mocking infrastructure.
