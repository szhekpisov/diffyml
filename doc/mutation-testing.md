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

The mutation testing workflow (`.github/workflows/mutation.yml`) runs on every PR targeting `main`. It uses `--diff` to only mutate changed code and enforces a 98% efficacy threshold.

## Report

**Last full run:** 2026-03-02 — efficacy 98.61% (568 killed / 576 covered)
**Line coverage:** 97.2% (`go test -cover ./pkg/diffyml/`)
**Mutator coverage:** 99.31%

| Status | Count |
|--------|-------|
| Killed | 568 |
| Lived | 8 |
| Timed out | 0 |
| Not covered | 4 |
| **Efficacy** | **98.61%** |
| **Mutator coverage** | **99.31%** |

## Survived Mutants (8 LIVED)

All 8 surviving mutants are **equivalent** — the mutation does not change observable program behavior, so no test can detect them.

---

### Pattern 1: Color mode detection (2 mutants)

**File:** `color.go`

| Line | Mutation | Code | Why equivalent |
|------|----------|------|----------------|
| 110:15 | `NEGATION` | `colorTerm == "truecolor"` | Guarded by `\|\| colorTerm == "24bit"` — both branches enable truecolor |
| 110:43 | `NEGATION` | `colorTerm == "24bit"` | Guarded by `\|\| colorTerm == "truecolor"` — same as above |

---

### Pattern 2: Map capacity hint (1 mutant)

**File:** `directory.go`
**Mutation:** `ARITHMETIC_BASE` at line 98:49 — `len(a) + len(b)` mutated

The capacity hint only affects initial memory allocation, not map behavior.

---

### Pattern 3: Flag parsing edge case (1 mutant)

**File:** `cli.go`
**Mutation:** `CONDITIONALS_BOUNDARY` at line 215:51 — `>= 0` changed to `> 0`

For `eqIdx == 0`, the flag name would be `""`. Either way, `fs.Lookup` returns nil and the arg is treated as positional.

---

### Pattern 4: `parseDocIndexPrefix` bracket boundary (2 mutants)

**File:** `detailed_formatter.go`
**Mutations:** `INVERT_NEGATIVES` and `ARITHMETIC_BASE` at line 741:21 — `-1` literal mutated

In `parseDocIndexPrefix`, `strings.Index(path, "]")` returns -1 only when `]` is absent. Mutating `-1` to `1` (INVERT_NEGATIVES) changes the check to `closeBracket == 1`, which would only matter for paths like `[]` where `closeBracket` is 1 — but in that case the subsequent `strconv.Atoi("")` returns an error, producing the same result. ARITHMETIC_BASE similarly mutates the constant without changing observable behavior.

---

### Pattern 5: DJB hash arithmetic (1 mutant)

**File:** `rename.go`
**Mutation:** `ARITHMETIC_BASE` at line 40:14 — `h*33 + uint32(b)` mutated

The DJB hash function is applied symmetrically to both documents being compared. Changing the hash arithmetic (e.g., `+` to `-`) produces different hash values, but *both* documents are hashed with the same mutated function. Identical lines still hash identically, and different lines still hash differently. The similarity score is unchanged.

---

### Pattern 6: Size-ratio threshold boundary (1 mutant)

**File:** `rename.go`
**Mutation:** `CONDITIONALS_BOUNDARY` at line 152:40 — `< renameScoreThreshold` changed to `<=`

Only matters when ratio == 60, which means size ratio is borderline and similarity score will make the same accept/reject decision.

*Previously 2 mutants. The `maxLen > 0` guard was changed to `maxLen != 0`, replacing the boundary mutation with a killable negation mutation.*

---

## Eliminated Mutants (24 former LIVED → removed or killed)

The following equivalent mutants were eliminated via code refactoring:

| Pattern | Mutants | Technique |
|---------|---------|-----------|
| Sort comparisons in `diffyml.go` | 4 | Replaced `sort.SliceStable` (bool less) with `slices.SortStableFunc` (int comparator) using `cmp.Compare`, eliminating `<` boundary targets |
| Clamp boundaries in `color.go` | 2 | Replaced manual `if val < min / if val > max` with `max(lo, min(val, hi))` builtin |
| `maxLen` boundaries in `comparator.go` | 2 | Replaced manual `if len(to) > maxLen` with `max()` builtin |
| Loop boundary in `comparator.go` | 1 | Replaced `for i := 0; i < maxLen; i++` with `for i := range maxLen` |
| `i >= len(from)` boundary in `comparator.go` | 1 | Split single loop into three range-separated loops (shared, added, removed) |
| Array reverse in `detailed_formatter.go` | 1 | Replaced manual reverse loop with `slices.Reverse(ops)` |
| LCS tie-breaking in `detailed_formatter.go` | 1 | Replaced `if-else` branch with `max(dp[i-1][j], dp[i][j-1])` builtin |
| `closeBracket < 0` in `detailed_formatter.go` | 1 | Changed to `closeBracket == -1`; boundary mutation eliminated, replaced by killable INVERT_NEGATIVES (but equivalent for -1 literal) |
| `len(path) > 1` in `diffyml.go` | 1 | Removed redundant outer guard; inner `lastDot >= 0` already handles it |
| Sort tiebreakers in `rename.go` | 4 | Replaced `sort.SliceStable` with `slices.SortStableFunc` + `cmp.Or(cmp.Compare(...), ...)` |
| K8s order detection in `kubernetes.go` | 5 | Removed dead nil guard, replaced `sort.Slice` with `slices.SortFunc` + `cmp.Compare`, used `slices.IsSortedFunc` for monotonicity check |
| List order detection in `comparator.go` | 1 | Replaced `sort.Slice` with `slices.SortFunc` + `cmp.Compare` |

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
