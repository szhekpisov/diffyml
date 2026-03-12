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

**Last full run:** 2026-03-12 — efficacy 100.00% (575 killed / 575 covered), 0 lived
**Mutator coverage:** 99.31%

| Status | Count |
|--------|-------|
| Killed | 575 |
| Lived | 0 |
| Timed out | 0 |
| Not covered | 4 |
| **Efficacy** | **100.00%** |
| **Mutator coverage** | **99.31%** |

## Not Covered (4 mutants)

4 mutants remain NOT COVERED. All are `ARITHMETIC_BASE` mutations on package-level constants. Constants are compile-time expressions that do not appear as executable statements in Go's `-coverprofile`, so gremlins cannot determine whether they are tested.

| File | Line | Constant |
|------|------|----------|
| `remote.go` | 14:23 | `MaxResponseSize = 10 * 1024 * 1024` |
| `remote.go` | 14:30 | `MaxResponseSize = 10 * 1024 * 1024` |
| `remote.go` | 16:22 | `DefaultTimeout = 30 * time.Second` |
| `cli/summarizer.go` | 26:24 | `summaryTimeout = 30 * time.Second` |

These constants are exercised by unit tests (`TestRemoteConstants`, `TestSummarize_Timeout`), but since Go does not instrument constant declarations, they will always be reported as NOT COVERED by gremlins.
