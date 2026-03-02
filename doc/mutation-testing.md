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

**Last full run:** 2026-03-02 — efficacy 100.00% (562 killed / 562 covered)
**Line coverage:** 97.2% (`go test -cover ./pkg/diffyml/`)
**Mutator coverage:** 99.29%

| Status | Count |
|--------|-------|
| Killed | 562 |
| Lived | 0 |
| Timed out | 0 |
| Not covered | 4 |
| **Efficacy** | **100.00%** |
| **Mutator coverage** | **99.29%** |

## Survived Mutants (0 LIVED)

No surviving mutants.

---

## Eliminated Mutants (35 former LIVED → removed or killed)

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
| `ShouldUseTrueColor` dead code in `color.go` | 2 | Removed redundant COLORTERM/TERM env checks — function always returned `c.trueColor` regardless; simplified to `return c.trueColor` |
| `renderListItems` map bullet in `detailed_formatter.go` | 1 | Strengthened test to verify `- ` bullet appears on first key only, not just key presence |
| Size-ratio threshold boundary in `rename.go` | 1 | Added boundary test with byte-size ratio exactly at threshold (60%), distinguishing `<` from `<=` |
| Map/slice capacity hints in `directory.go`, `summarizer.go` | 3 | Removed capacity hints that produced unkillable `ARITHMETIC_BASE` mutants on allocation-only expressions |
| Flag parsing `>= 0` in `cli.go` | 1 | Replaced `IndexByte` + boundary check with `strings.Cut`, removing the mutation target |
| `parseDocIndexPrefix` `-1` literal in `detailed_formatter.go` | 2 | Replaced `strings.Index` + `-1` check with `strings.Cut`, removing the mutation target |
| DJB hash arithmetic in `rename.go` | 1 | Replaced hand-rolled DJB hash with `crc32.ChecksumIEEE`, removing the `h*33 + uint32(b)` mutation target |

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
