---
title: "diffyml"
type: docs
---

# diffyml

A fast, structural YAML diff tool with built-in Kubernetes intelligence. One dependency, minimal attack surface, native CI annotations for GitHub, GitLab, and Gitea.

![diffyml output](/diffyml/img/demo.png)

```bash
brew install szhekpisov/diffyml/diffyml
diffyml old.yaml new.yaml
```

## Why diffyml?

**Fastest structural YAML diff at scale.** On large (5K lines) and xlarge (50K lines) inputs diffyml is 1.5–1.9× faster than the nearest YAML-aware competitor. On small/medium files it ties within milliseconds — the residual overhead comes from capabilities the simpler tools lack (x509 inspection, remote URL fetching, AI summaries).

**One dependency, zero surprises.** A single module dependency (`yaml.v3`) and pure Go stdlib. Auditable in minutes.

**Gets YAML right.** Dotted keys, type preservation, mixed-type lists, nil values — concrete edge cases other tools get wrong.

## Where to next

- **[Install]({{< relref "/docs/install" >}})** — Homebrew, Docker, `go install`, release binaries
- **[Quick Start]({{< relref "/docs/quick-start" >}})** — three minimal examples that cover 90% of usage
- **[Output Formats]({{< relref "/docs/formats" >}})** — pick the right format for terminal review or CI
- **[CI Integration]({{< relref "/docs/ci" >}})** — GitHub Actions, GitLab Code Quality, Gitea, kubectl, git
- **[Reference]({{< relref "/docs/reference" >}})** — every flag, auto-generated from source

## How it compares

| Feature | diffyml | dyff | plain `diff` |
|---------|---------|------|------|
| YAML-aware (structural) | Yes | Yes | No (line-based) |
| Kubernetes resource matching | apiVersion + kind + name (or generateName) | apiVersion + kind + name | No |
| Rename detection | Yes (content similarity) | Yes (identifier) | No |
| API version migration | Yes (`--ignore-api-version`) | No | No |
| Inverse diff (report unchanged values) | Yes (`--unchanged`) | No | No |
| CI annotation formats | 3 (GitHub, GitLab, Gitea) | 0 | 0 |
| Module dependencies | 1 (yaml.v3) | 23 | 0 |
| Performance (78 KB) | 19 ms | 120 ms (6.4×) | 6 ms |
| Performance (780 KB) | 129 ms | 1,146 ms (8.9×) | 45 ms |

Comparison based on dyff v1.11.3 and diffyml v1.5.23. See the [PERFORMANCE doc](https://github.com/szhekpisov/diffyml/blob/main/doc/PERFORMANCE.md) for full methodology.
