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

[gomutants](https://github.com/szhekpisov/gomutants) v0.1.0

## CI Integration

`.github/workflows/mutation.yml` runs in two modes:

- **Pull requests** — `gomutants --changed-since origin/main` scopes mutation to lines touched by the PR. Gate: any LIVED mutant on changed lines fails the job. This makes mutation testing a per-PR quality bar rather than a periodic audit.
- **Push to main** — full-tree run after merge. Gate: `test_efficacy ≥ 85.65%` (the calibrated floor at gomutants migration; ratchet up as tests improve). Catches coverage rot in code no PR happens to touch.

## Report

**Last full run:** 2026-04-29 — efficacy 85.66% on 2325 mutants (gomutants v0.1.0)
**Mutations coverage:** 98.54%

| Status | Count |
|--------|-------|
| Killed | 1666 |
| Lived | 279 |
| Not viable | 324 |
| Not covered | 34 |
| Timed out | 22 |
| **Efficacy** | **85.66%** |
| **Mutations coverage** | **98.54%** |

The drop from the legacy 100% figure reflects gomutants's larger mutator set (~16 active mutators vs. gremlins' 11). New mutators — chiefly `STATEMENT_REMOVE`, `EXPRESSION_REMOVE`, `BRANCH_CASE`, and `INVERT_LOOP_CTRL` — probe locations the gremlins-era suite did not exercise. The 279 lived mutants represent a mix of real test gaps and equivalent mutants; the PR-scoped gate prevents new ones from sneaking in while the floor is ratcheted up.

## Lived Mutants by File (top 10)

| File | Lived | Total |
|------|------:|------:|
| `comparator.go` | 44 | 332 |
| `formatter.go` | 36 | 295 |
| `kubernetes.go` | 32 | 126 |
| `diffyml.go` | 23 | 155 |
| `detailed_formatter.go` | 22 | 181 |
| `rename.go` | 18 | 102 |
| `color.go` | 16 | 114 |
| `chroot.go` | 15 | 90 |
| `diffpath.go` | 14 | 90 |
| `inline_diff.go` | 10 | 76 |

By mutator: `STATEMENT_REMOVE` (77), `EXPRESSION_REMOVE` (74), `BRANCH_IF` (61), `INVERT_LOGICAL` (28), `BRANCH_CASE` (18), `INVERT_LOOP_CTRL` (17), `INVERT_BITWISE` (3), `REMOVE_SELF_ASSIGNMENTS` (1).

## Not Covered

34 mutants are NOT COVERED — predominantly `ARITHMETIC_BASE` mutations on package-level constants. Go does not instrument constant declarations in `-coverprofile`, so gomutants cannot determine whether they are tested even when they are exercised by unit tests (e.g., `TestRemoteConstants`, `TestSummarize_Timeout`). These will always report as NOT COVERED regardless of test additions; full enumeration is in `mutation-report.json`.
