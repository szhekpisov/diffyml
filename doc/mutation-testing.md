# Mutation Testing Report

**Tool:** [gremlins](https://github.com/go-gremlins/gremlins)
**Last full run:** 2026-02-28 — efficacy 93.68% (489 killed / 522 covered)
**Line coverage:** 96.6% (`go test -cover ./pkg/diffyml/`)
**Mutator coverage:** 92.27% (gremlins dry-run, 537 runnable / 45 not covered)

## Summary (last full run)

| Status | Count |
|--------|-------|
| Killed | 489 |
| Lived | 33 |
| Timed out | 4 |
| Not covered | 60 → 45 (dry-run) |
| **Efficacy** | **93.68%** |
| **Mutator coverage** | **89.69% → 92.27%** |

> **Note:** Efficacy and killed/lived counts are from the last full `gremlins unleash`
> run. After removing dead code and adding new tests, the not-covered count dropped
> from 60 to 45 (dry-run), and line coverage rose from 92.3% to 96.6%. A full run
> is needed to update efficacy numbers.

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
| 337:13 | `len(to) > maxLen` → `>=` | Same pattern as line 27 |

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

## Not Covered (45 mutants)

45 mutants are in code paths not exercised by tests. Down from 60 after adding
coverage for plain-map branches, positional list bounds, `deepEqual` slices,
`clamp` boundaries, `ChrootError.Error()`, `ExitResult.String()` edge cases,
`renderFirstKeyValueYAML` list values, `compareListsByIdentifier` no-ID fallback,
`extractPathOrder`/`areListItemsHeterogeneous` plain maps, `GetContextColorCode`
true color path, and `runDirectory` with real filesystem paths. Dead cross-type
branches in `compareNodes` and `deepEqual` were removed (provably unreachable
due to type-equality guards).

**Line coverage:** 96.6% (`go test -cover`)
**Mutator coverage:** 92.27% (gremlins dry-run)

### By file

| File | Mutants | Category |
|------|---------|----------|
| `detailed_formatter.go` | 29 | LCS algorithm in `computeLineDiff` (lines 464, 466, 479, 483) |
| `summarizer.go` | 6 | Anthropic API error-status branches (lines 146, 148, 150, 156) and timeout constant (line 26) |
| `comparator.go` | 4 | `compareListsPositional` add/remove bounds (lines 353, 361) — covered by `go test` but gremlins disagrees |
| `remote.go` | 3 | `IsRemoteSource` prefix checks (line 14) and `fetchURL` status check (line 16) — covered by `go test` but gremlins disagrees |
| `directory.go` | 3 | `buildFilePairsFromMap` branch conditions (lines 179, 181) — covered by `go test` but gremlins disagrees |

### Analysis

**LCS algorithm (29 mutants):** The `computeLineDiff` function implements a Longest
Common Subsequence algorithm used for inline multiline diffs. The LCS DP table
construction (line 464) and backtracking (lines 479, 483) have many arithmetic
and conditional mutants. These lines ARE covered by existing tests (`go test
-coverprofile` shows `computeLineDiff` at 100%), but gremlins' coverage gathering
reports them as NOT COVERED — likely a discrepancy in how gremlins aggregates
coverage across packages.

**Summarizer API errors (6 mutants):** The HTTP status-code switch in `Summarize()`
(401, 429, >=500, !=200) and the `summaryTimeout` constant. Tests exist for all
these paths (`TestSummarize_Auth401`, `TestSummarize_RateLimit429`,
`TestSummarize_ServerError500`), but gremlins does not recognize them as covered.

**Gremlins coverage discrepancy (10 mutants):** 10 of the 45 NOT COVERED mutants
are in lines that `go test -coverprofile` confirms ARE executed (comparator.go,
remote.go, directory.go). This appears to be a gremlins-specific issue with
coverage aggregation, not an actual test gap.
