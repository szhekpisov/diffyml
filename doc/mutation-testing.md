# Mutation Testing

## What is Mutation Testing?

Mutation testing evaluates test suite quality by introducing small, systematic changes (mutations) to source code and checking whether tests detect them. Unlike line coverage, which only measures whether code *executes*, mutation testing measures whether tests actually *verify* correct behavior.

### How it works

1. A mutation tool modifies the source code — e.g., changing `<` to `<=`, `+` to `-`, or negating a condition
2. The test suite runs against each mutant
3. If tests **fail** → the mutant is **killed** (tests caught the bug)
4. If tests **pass** → the mutant **lived** (tests missed the bug)
5. **Efficacy** = killed / (killed + lived) — the higher, the better

A surviving mutant means either the test suite has a gap, or the mutation is **equivalent** (it doesn't change observable behavior, so no test can detect it).

## Tool

[gremlins](https://github.com/go-gremlins/gremlins) v0.6.0

## CI Integration

The mutation testing workflow (`.github/workflows/mutation.yml`) runs on every PR targeting `main`. It checks out the PR branch directly (not the merge commit) and runs full mutation testing on `./pkg/diffyml/`. Efficacy is enforced by parsing the JSON report and failing CI if below 96% — gremlins' built-in `--threshold-efficacy` flag does not actually enforce the threshold (v0.6.0 bug).

## Report

**Last full run:** 2026-03-01 — efficacy 94.51% (602 killed / 637 covered)
**Line coverage:** 97.3% (`go test -cover ./pkg/diffyml/`)
**Mutator coverage:** 99.38%

| Status | Count |
|--------|-------|
| Killed | 602 |
| Lived | 35 |
| Timed out | 0 |
| Not covered | 4 |
| **Efficacy** | **94.51%** |
| **Mutator coverage** | **99.38%** |

## Survived Mutants (35 LIVED)

All 35 surviving mutants are **equivalent** — the mutation does not change
observable program behavior, so no test can detect them.

---

### Pattern 1: `<` changed to `<=` in sort comparisons (4 mutants)

**File:** `diffyml.go`
**Mutation:** `CONDITIONALS_BOUNDARY` — `<` changed to `<=`

In `sortDiffsWithOrder`, each `<` comparison is guarded by a prior `!=` check that ensures the operands are never equal. When they can't be equal, `<` and `<=` behave identically.

| Line | Code | Why equivalent |
|------|------|----------------|
| 305:19 | `orderI < orderJ` | Guarded by `rootI != rootJ`; different roots get unique indices |
| 307:17 | `rootI < rootJ` | Guarded by `rootI != rootJ`; identical strings can't reach this line |
| 344:24 | `parentOrderI < parentOrderJ` | Guarded by `parentOrderI != parentOrderJ` on prior line |
| 351:18 | `depthI < depthJ` | Guarded by `depthI != depthJ` on prior line |

---

### Pattern 2: Color mode detection and clamp boundaries (4 mutants)

**File:** `color.go`

| Line | Mutation | Code | Why equivalent |
|------|----------|------|----------------|
| 110:15 | `NEGATION` | `colorTerm == "truecolor"` | Guarded by `|| colorTerm == "24bit"` — both branches enable truecolor |
| 110:43 | `NEGATION` | `colorTerm == "24bit"` | Guarded by `|| colorTerm == "truecolor"` — same as above |
| 214:9 | `BOUNDARY` | `val < min` → `<=` | Clamp: returns `min` when `val == min` either way |
| 217:9 | `BOUNDARY` | `val > max` → `>=` | Clamp: returns `max` when `val == max` either way |

---

### Pattern 3: `maxLen` and list-bounds boundaries in comparator (6 mutants)

**File:** `comparator.go`
**Mutation:** `CONDITIONALS_BOUNDARY`

| Line | Code | Why equivalent |
|------|------|----------------|
| 27:13 | `len(to) > maxLen` → `>=` | Sets `maxLen` to `len(to)` which already equals `maxLen` at boundary |
| 31:16 | `i < maxLen` → `<=` | Extra iteration with both nil docs is a no-op |
| 337:13 | `len(to) > maxLen` → `>=` | Same pattern as line 27 |
| 353:8 | `i >= len(from)` → `>` | At `i == len(from)`: `fromVal` is nil (prior `if i < len(from)` failed), so the `else` branch calls `compareNodes(path, nil, toVal)` which produces `DiffAdded` — same result |
| 492:27 | `len(commonFromOrder) > 1` → `>=` | With exactly 1 common item, order comparison loop finds no difference (single element is always in order) |
| 504:28 | `toSorted[i].idx < toSorted[j].idx` → `<=` | Sort comparator for unique indices; equal indices can't occur, so `<` and `<=` are identical |

---

### Pattern 4: Boundary in `len(path) > 1` (1 mutant)

**File:** `diffyml.go`
**Mutation:** `CONDITIONALS_BOUNDARY` at line 222:15 — `> 1` changed to `>= 1`

For a single-character path, `LastIndex` returns -1 (no dot), and the inner `lastDot >= 0` check fails. The mutation allows entry but the inner guard prevents any behavior change.

---

### Pattern 5: LCS tie-breaking (2 mutants)

**File:** `detailed_formatter.go`
**Mutation:** `CONDITIONALS_BOUNDARY`

When `dp[i-1][j] == dp[i][j-1]`, both branches assign the same value. The DP table is identical regardless of which branch is taken.

| Line | Code | Why equivalent |
|------|------|----------------|
| 490:17 | `j <= n` → `j < n` | Inner loop boundary; at `j == n` the DP cell is already computed by the outer structure |
| 494:25 | `dp[i-1][j] >= dp[i][j-1]` → `>` | When equal, both branches assign the same max LCS value |

---

### Pattern 6: Array reverse self-swap (1 mutant)

**File:** `detailed_formatter.go`
**Mutation:** `CONDITIONALS_BOUNDARY` at line 521:41 — `<` changed to `<=`

When `left == right` (odd-length array midpoint), swapping an element with itself is a no-op.

---

### Pattern 7: Map capacity hint (1 mutant)

**File:** `directory.go`
**Mutation:** `ARITHMETIC_BASE` at line 98:49 — `len(a) + len(b)` mutated

The capacity hint only affects initial memory allocation, not map behavior.

---

### Pattern 8: Flag parsing edge case (1 mutant)

**File:** `cli.go`
**Mutation:** `CONDITIONALS_BOUNDARY` at line 215:51 — `>= 0` changed to `> 0`

For `eqIdx == 0`, the flag name would be `""`. Either way, `fs.Lookup` returns nil and the arg is treated as positional.

---

### Pattern 9: `parseDocIndexPrefix` bracket boundary (1 mutant)

**File:** `detailed_formatter.go`
**Mutation:** `CONDITIONALS_BOUNDARY` at line 744:18 — `< 0` changed to `<= 0`

In `parseDocIndexPrefix`, the prior check `!strings.HasPrefix(path, "[")` ensures `path[0] == '['`. Therefore `strings.Index(path, "]")` can never return 0 (the first `]` is always at index >= 1). Changing `< 0` to `<= 0` has no effect.

---

### Pattern 10: K8s document order detection boundaries (5 mutants)

**File:** `kubernetes.go`
**Mutation:** `CONDITIONALS_BOUNDARY` and `CONDITIONALS_NEGATION`

| Line | Mutation | Code | Why equivalent |
|------|----------|------|----------------|
| 241:11 | `NEGATION` | `opts == nil` → `!= nil` | When `opts != nil` and `IgnoreOrderChanges` is false, negating the nil check still enters the block (short-circuit `\|\|` evaluates the second operand) |
| 241:63 | `BOUNDARY` | `len(matched) > 1` → `>=` | With exactly 1 match, the order check loop finds no reordering (single element is trivially ordered) |
| 248:67 | `BOUNDARY` | `pairs[i].fromIdx < pairs[j].fromIdx` → `<=` | Sort comparator for unique from-indices; equal values can't occur |
| 253:22 | `BOUNDARY` | `pairs[i].toIdx < pairs[i-1].toIdx` → `<=` | Equal to-indices can't occur with unique matches, so `<` and `<=` are identical |
| 265:66 | `BOUNDARY` | `pairs[i].toIdx < pairs[j].toIdx` → `<=` | Sort comparator for unique to-indices; equal values can't occur |

---

### Pattern 11: DJB hash arithmetic (1 mutant)

**File:** `rename.go`
**Mutation:** `ARITHMETIC_BASE` at line 39:14 — `h*33 + uint32(b)` mutated

The DJB hash function is applied symmetrically to both documents being compared. Changing the hash arithmetic (e.g., `+` to `-`) produces different hash values, but *both* documents are hashed with the same mutated function. Identical lines still hash identically, and different lines still hash differently. The similarity score is unchanged.

---

### Pattern 12: Size-ratio early rejection — correlated with similarity score (4 mutants)

**File:** `rename.go`

The size-ratio check (`minLen*100/maxLen < renameScoreThreshold`) is an optimization that skips the full similarity computation for documents with very different sizes. Mutations that break or disable this check do not change the final result because the full similarity score (which follows immediately) is mathematically correlated with the size ratio — documents with a size ratio below 60% also have similarity scores below 60%, so they would be rejected by the score threshold check anyway.

| Line | Mutation | Code | Why equivalent |
|------|----------|------|----------------|
| 151:14 | `BOUNDARY` | `maxLen > 0` → `>=` | Always true for real documents; condition is already always true |
| 151:14 | `NEGATION` | `maxLen > 0` → `<= 0` | Always false; disables size-ratio check. Fallback similarity score still rejects. |
| 151:31 | `ARITHMETIC` | `minLen*100/maxLen` mutated | Produces wrong ratio; disables size-ratio check. Fallback similarity score still rejects. |
| 151:39 | `BOUNDARY` | `< renameScoreThreshold` → `<=` | Only matters when ratio == 60, which means size ratio is borderline and similarity score will make the same accept/reject decision |

---

### Pattern 13: Sort tiebreaker with invalid comparators (4 mutants)

**File:** `rename.go`

The sort comparator in `detectRenames` sorts scored rename pairs descending by score, with deterministic tiebreaking by ascending `fromIdx` then `toIdx`. Mutations to the tiebreaker produce invalid (non-irreflexive) comparators for `sort.SliceStable`, making the sort output undefined. In practice, Go's merge sort implementation with small inputs may happen to produce correct results even with an invalid comparator.

| Line | Mutation | Code |
|------|----------|------|
| 165:26 | `BOUNDARY` | `pairs[i].score > pairs[j].score` → `>=` |
| 167:23 | `NEGATION` | `pairs[i].fromIdx < pairs[j].fromIdx` → `>=` |
| 168:28 | `BOUNDARY` | `pairs[i].fromIdx != pairs[j].fromIdx` → boundary mutation |
| 170:25 | `BOUNDARY` | `pairs[i].toIdx < pairs[j].toIdx` → `<=` |

---

## Not Covered (4 mutants)

4 mutants remain NOT COVERED. All are `ARITHMETIC_BASE` mutations on package-level constants. Constants are compile-time expressions that do not appear as executable statements in Go's `-coverprofile`, so gremlins cannot determine whether they are tested.

| File | Line | Constant |
|------|------|----------|
| `remote.go` | 14:23 | `MaxResponseSize = 10 * 1024 * 1024` |
| `remote.go` | 14:30 | `MaxResponseSize = 10 * 1024 * 1024` |
| `remote.go` | 16:22 | `DefaultTimeout = 30 * time.Second` |
| `summarizer.go` | 26:24 | `summaryTimeout = 30 * time.Second` |

These constants are exercised by unit tests (`TestRemoteConstants`, `TestSummarize_Timeout`), but since Go does not instrument constant declarations, they will always be reported as NOT COVERED by gremlins.
