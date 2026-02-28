# Mutation Testing

## What is Mutation Testing?

Mutation testing evaluates test suite quality by introducing small, systematic
changes (mutations) to source code and checking whether tests detect them.
Unlike line coverage, which only measures whether code *executes*, mutation
testing measures whether tests actually *verify* correct behavior.

### How it works

1. A mutation tool modifies the source code — e.g., changing `<` to `<=`,
   `+` to `-`, or negating a condition
2. The test suite runs against each mutant
3. If tests **fail** → the mutant is **killed** (tests caught the bug)
4. If tests **pass** → the mutant **lived** (tests missed the bug)
5. **Efficacy** = killed / (killed + lived) — the higher, the better

A surviving mutant means either the test suite has a gap, or the mutation is
**equivalent** (it doesn't change observable behavior, so no test can detect it).

## Tool

[gremlins](https://github.com/go-gremlins/gremlins) v0.6.0

## CI Integration

The mutation testing workflow (`.github/workflows/mutation.yml`) runs on every
PR targeting `main`. It uses `--diff` to only mutate changed code and enforces
a 96% efficacy threshold via `--threshold-efficacy`.

## Report

**Last full run:** 2026-02-28 — efficacy 96.79% (542 killed / 560 covered)
**Line coverage:** 96.6% (`go test -cover ./pkg/diffyml/`)
**Mutator coverage:** 99.29%

| Status | Count |
|--------|-------|
| Killed | 542 |
| Lived | 18 |
| Timed out | 0 |
| Not covered | 4 |
| **Efficacy** | **96.79%** |
| **Mutator coverage** | **99.29%** |

## Survived Mutants (18 LIVED)

All 18 surviving mutants are **equivalent** — the mutation does not change
observable program behavior, so no test can detect them.

---

### Pattern 1: `<` changed to `<=` in sort comparisons (4 mutants)

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

---

### Pattern 2: Clamp boundary values (3 mutants)

**File:** `color.go`
**Mutation:** `CONDITIONALS_BOUNDARY`

| Line | Code | Why equivalent |
|------|------|----------------|
| 66:15 | `override < minTerminalWidth` → `<=` | When `override == min`, returns `min` either way |
| 243:9 | `val < min` → `<=` | Clamp: returns `min` when `val == min` either way |
| 246:9 | `val > max` → `>=` | Clamp: returns `max` when `val == max` either way |

---

### Pattern 3: `maxLen` and list-bounds boundaries in comparator (4 mutants)

**File:** `comparator.go`
**Mutation:** `CONDITIONALS_BOUNDARY`

| Line | Code | Why equivalent |
|------|------|----------------|
| 27:13 | `len(to) > maxLen` → `>=` | Sets `maxLen` to `len(to)` which already equals `maxLen` at boundary |
| 31:16 | `i < maxLen` → `<=` | Extra iteration with both nil docs is a no-op |
| 337:13 | `len(to) > maxLen` → `>=` | Same pattern as line 27 |
| 352:8 | `i >= len(from)` → `>` | At `i == len(from)`: `fromVal` is nil (prior `if i < len(from)` failed), so the `else` branch calls `compareNodes(path, nil, toVal)` which produces `DiffAdded` — same result |

---

### Pattern 4: Boundary in `len(path) > 1` (1 mutant)

**File:** `diffyml.go`
**Mutation:** `CONDITIONALS_BOUNDARY` at line 222:15 — `> 1` changed to `>= 1`

For a single-character path, `LastIndex` returns -1 (no dot), and the inner
`lastDot >= 0` check fails. The mutation allows entry but the inner guard
prevents any behavior change.

---

### Pattern 5: LCS tie-breaking (2 mutants)

**File:** `detailed_formatter.go`
**Mutation:** `CONDITIONALS_BOUNDARY`

When `dp[i-1][j] == dp[i][j-1]`, both branches assign the same value. The DP
table is identical regardless of which branch is taken.

| Line | Code | Why equivalent |
|------|------|----------------|
| 472:17 | `j <= n` → `j < n` | Inner loop boundary; at `j == n` the DP cell is already computed by the outer structure |
| 475:25 | `dp[i-1][j] >= dp[i][j-1]` → `>` | When equal, both branches assign the same max LCS value |

---

### Pattern 6: Array reverse self-swap (1 mutant)

**File:** `detailed_formatter.go`
**Mutation:** `CONDITIONALS_BOUNDARY` at line 501:41 — `<` changed to `<=`

When `left == right` (odd-length array midpoint), swapping an element with
itself is a no-op.

---

### Pattern 7: Map capacity hint (1 mutant)

**File:** `directory.go`
**Mutation:** `ARITHMETIC_BASE` at line 98:49 — `len(a) + len(b)` mutated

The capacity hint only affects initial memory allocation, not map behavior.

---

### Pattern 8: Flag parsing edge case (1 mutant)

**File:** `cli.go`
**Mutation:** `CONDITIONALS_BOUNDARY` at line 225:51 — `>= 0` changed to `> 0`

For `eqIdx == 0`, the flag name would be `""`. Either way, `fs.Lookup` returns
nil and the arg is treated as positional.

---

### Pattern 9: `parseDocIndexPrefix` bracket boundary (1 mutant)

**File:** `detailed_formatter.go`
**Mutation:** `CONDITIONALS_BOUNDARY` at line 646:18 — `< 0` changed to `<= 0`

In `parseDocIndexPrefix`, the prior check `!strings.HasPrefix(path, "[")` ensures
`path[0] == '['`. Therefore `strings.Index(path, "]")` can never return 0 (the
first `]` is always at index >= 1). Changing `< 0` to `<= 0` has no effect.

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
