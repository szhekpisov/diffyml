# diffyml

Structural diff for YAML files. Understands YAML semantics and detects Kubernetes resources.

[![Tests](https://github.com/szhekpisov/diffyml/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/szhekpisov/diffyml/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/szhekpisov/diffyml/branch/main/graph/badge.svg)](https://codecov.io/gh/szhekpisov/diffyml)
[![Go Report Card](https://goreportcard.com/badge/github.com/szhekpisov/diffyml)](https://goreportcard.com/report/github.com/szhekpisov/diffyml)
[![Security & Static Analysis](https://github.com/szhekpisov/diffyml/actions/workflows/security.yml/badge.svg?branch=main)](https://github.com/szhekpisov/diffyml/actions/workflows/security.yml)
[![Benchmark](https://github.com/szhekpisov/diffyml/actions/workflows/benchmark.yml/badge.svg?branch=main)](https://github.com/szhekpisov/diffyml/actions/workflows/benchmark.yml)

## Features

- **Minimal dependencies** — single runtime dependency ([yaml.v3](https://github.com/yaml/go-yaml)); pure Go stdlib otherwise. Small attack surface, fast by design ([benchmarks](#performance))
- **Kubernetes-aware** — auto-detects and matches resources by apiVersion, kind, and metadata
- **Rename detection** — identifies renamed resources instead of showing remove + add
- **6 output formats** — detailed, compact, brief, GitHub, GitLab, Gitea
- **Path filtering** — include/exclude paths with exact match or regex
- **Remote files** — compare directly from HTTP/HTTPS URLs
- **Certificate inspection** — inspects and compares embedded x509 certificates
- **Chroot navigation** — focus comparison on a specific YAML subtree

## Installation

### Go Install

```bash
go install github.com/szhekpisov/diffyml@latest
```

Make sure `$GOPATH/bin` is in your `PATH`:

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
```

### Homebrew

Coming soon.

### From Source

```bash
git clone https://github.com/szhekpisov/diffyml.git
cd diffyml
go build -o diffyml
```

## Quick Start

```bash
# Compare two local files
diffyml old.yaml new.yaml

# Compare local file against a remote URL
diffyml local.yaml https://example.com/remote.yaml

# Use in CI — exit code 1 when differences found
diffyml -s deployment-old.yaml deployment-new.yaml
```

## Usage

```bash
diffyml [flags] <from> <to>
```

### Output Formats

| Format | Flag | Use case |
|--------|------|----------|
| detailed | `-o detailed` (default) | Human review — full context |
| compact | `-o compact` | Quick scan of changes |
| brief | `-o brief` | Summary only |
| github | `-o github` | GitHub Actions annotations |
| gitlab | `-o gitlab` | GitLab CI annotations |
| gitea | `-o gitea` | Gitea CI annotations |

### Kubernetes Support

Kubernetes resources are automatically detected and matched by `apiVersion`, `kind`, and `metadata.name`. Renames are tracked as moves, not as remove + add.

```bash
# Compare two Kubernetes manifests
diffyml manifests-v1.yaml manifests-v2.yaml

# Disable Kubernetes detection
diffyml --detect-kubernetes=false file1.yaml file2.yaml
```

### Filtering

```bash
# Show only changes under a specific path
diffyml --filter spec.replicas old.yaml new.yaml

# Exclude noisy paths
diffyml --exclude metadata.annotations old.yaml new.yaml

# Regex filtering
diffyml --filter-regexp 'spec\.containers\[.*\]\.image' old.yaml new.yaml
```

### CI Integration

Use `-s` / `--set-exit-code` to set the exit code based on differences:

| Exit code | Meaning |
|-----------|---------|
| `0` | No differences (or success without `-s`) |
| `1` | Differences detected (only with `-s`) |
| `255` | Error occurred |

```bash
diffyml -s before.yaml after.yaml || echo "Config drift detected"
```

### All Flags

<details>
<summary>Complete flag reference</summary>

**Output**

| Flag | Description |
|------|-------------|
| `-o, --output <style>` | Output style: `detailed`, `compact`, `brief`, `github`, `gitlab`, `gitea` (default `detailed`) |
| `-c, --color <mode>` | Color usage: `on`, `off`, `auto` (default `auto`) |
| `-t, --truecolor <mode>` | True color (24-bit): `on`, `off`, `auto` (default `auto`) |
| `-w, --fixed-width <int>` | Fixed terminal width instead of auto-detection |

**Comparison**

| Flag | Description |
|------|-------------|
| `-i, --ignore-order-changes` | Ignore order changes in lists |
| `--ignore-whitespace-changes` | Ignore leading/trailing whitespace differences |
| `-v, --ignore-value-changes` | Show only structural changes, exclude value changes |
| `--detect-kubernetes` | Detect and match Kubernetes resources (default `true`) |
| `--detect-renames` | Detect renamed resources (default `true`) |
| `-x, --no-cert-inspection` | Disable x509 certificate inspection |
| `--swap` | Swap from/to files |

**Filtering**

| Flag | Description |
|------|-------------|
| `--filter <path>` | Include only differences at specified paths (repeatable) |
| `--exclude <path>` | Exclude differences at specified paths (repeatable) |
| `--filter-regexp <pattern>` | Filter using regular expressions (repeatable) |
| `--exclude-regexp <pattern>` | Exclude using regular expressions (repeatable) |
| `--additional-identifier <field>` | Additional field for list item identification |

**Display**

| Flag | Description |
|------|-------------|
| `-b, --omit-header` | Omit summary header |
| `-l, --no-table-style` | Display blocks vertically instead of side-by-side |
| `-g, --use-go-patch-style` | Use Go-Patch style paths |
| `--multi-line-context-lines <int>` | Context lines for multi-line strings (default `4`) |
| `--minor-change-threshold <float>` | Minor change threshold (default `0.1`) |

**Chroot**

| Flag | Description |
|------|-------------|
| `--chroot <path>` | Change root level for both files |
| `--chroot-of-from <path>` | Change root level for the from file only |
| `--chroot-of-to <path>` | Change root level for the to file only |
| `--chroot-list-to-documents` | Treat chroot list as separate documents |

**Other**

| Flag | Description |
|------|-------------|
| `-s, --set-exit-code` | Exit code 1 if differences found |
| `-h, --help` | Show help |
| `-V, --version` | Show version information |

</details>

## Code Quality

Every push and PR is checked by:

- [govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck) — known vulnerability detection
- [golangci-lint](https://golangci-lint.run/) running:
  [errcheck](https://github.com/kisielk/errcheck),
  [gocritic](https://github.com/go-critic/go-critic),
  [gosec](https://github.com/securego/gosec),
  [govet](https://pkg.go.dev/cmd/vet) (with shadow detection),
  [ineffassign](https://github.com/gordonklaus/ineffassign),
  [misspell](https://github.com/client9/misspell),
  [staticcheck](https://staticcheck.dev/) (all checks except style conventions)

Core packages enforce 95–100% test coverage thresholds in CI.

## Performance

Benchmarked against 4 Go-based YAML diff tools using [hyperfine](https://github.com/sharkdp/hyperfine) (20 runs, 5 warmup). Environment: Apple M1 Pro, macOS, Go 1.25.7.

| File size | diffyml | [dyff](https://github.com/homeport/dyff) | [semihbkgr/yamldiff](https://github.com/semihbkgr/yamldiff) | [sters/yaml-diff](https://github.com/sters/yaml-diff) | [sahilm/yamldiff](https://github.com/sahilm/yamldiff) | diff |
|-----------|--------:|-----:|----------:|------:|-------:|-----:|
| ~70 lines | 5.7 ms | 15.3 ms | 3.9 ms | 4.0 ms | **3.7 ms** | 2.2 ms |
| ~530 lines | 6.3 ms | 29.2 ms | **5.2 ms** | 11.5 ms | 16.1 ms | 2.6 ms |
| ~5K lines | **22.3 ms** | 173.8 ms | 27.9 ms | 984 ms | 1,370 ms | 6.2 ms |
| ~50K lines | **152.3 ms** | 3,647 ms | 245.7 ms | — | — | 46.2 ms |

Lowest memory footprint at every size (18.4 MB at 5K lines vs 21–326 MB for alternatives). See [performance.md](performance.md) for full methodology and results.

<details>
<summary>Reproduce benchmarks</summary>

```bash
# Full benchmark (small, medium, large)
make bench-compare

# Include xlarge (sters/yaml-diff and sahilm/yamldiff are auto-excluded at this size)
bash bench/compare/run.sh --sizes small,medium,large,xlarge

# Quick check with fewer runs
bash bench/compare/run.sh --runs 3
```

</details>

## Contributing

Contributions welcome! [Open an issue](https://github.com/szhekpisov/diffyml/issues) for bugs or feature requests.

<details>
<summary>Development setup</summary>

**Prerequisites:** Go 1.25.7+, [pre-commit](https://pre-commit.com/)

```bash
git clone https://github.com/szhekpisov/diffyml.git
cd diffyml
pre-commit install
```

**Pre-commit hooks** run automatically on every commit:

| Hook | What it checks |
|------|---------------|
| `gofmt` | Code formatting |
| `go vet` | Static analysis |
| `check-coverage` | Coverage thresholds (100% parser, 100% ordered_map, 95% kubernetes) |
| `govulncheck` | Known vulnerabilities |
| `golangci-lint` | 7 linters (errcheck, gocritic, gosec, govet, ineffassign, misspell, staticcheck) |

**Useful Make targets:**

```bash
make test           # run all tests
make ci             # full CI pipeline locally (fmt + vet + test + coverage + security)
make bench          # run benchmarks
make bench-compare  # compare against alternative tools
make coverage       # generate HTML coverage report
```

**CI pipelines** (run on every push and PR):
- **Tests** — unit tests + coverage thresholds
- **Security & Static Analysis** — govulncheck + golangci-lint (also runs weekly)
- **Benchmark** — performance regression tracking

</details>

## License

MIT — see [LICENSE](LICENSE).
