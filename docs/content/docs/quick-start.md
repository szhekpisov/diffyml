---
title: "Quick Start"
weight: 20
---

# Quick Start

Five minimal examples that cover most of what you'll do day-to-day.

## Compare two local files

```bash
diffyml old.yaml new.yaml
```

The default `detailed` format renders a colored, contextual diff in the terminal.

## Compare two local directories

```bash
diffyml dir-old/ dir-new/
```

diffyml walks both directories recursively, matches files by relative path, and shows aggregated differences. Non-YAML files are silently skipped.

## Compare a local file against a remote URL

```bash
diffyml local.yaml https://example.com/remote.yaml
```

`from` and `to` arguments accept HTTP/HTTPS URLs in addition to local paths.

## Use in CI with an exit code

```bash
diffyml -s before.yaml after.yaml || echo "Config drift detected"
```

The `-s` / `--set-exit-code` flag makes diffyml exit `1` when differences are found, `0` when files are identical, and `255` on errors. Without `-s` the exit code is always `0` (drift is reported but not failed).

## Use as the kubectl external diff provider

```bash
export KUBECTL_EXTERNAL_DIFF="diffyml --omit-header --set-exit-code"
kubectl diff -f manifests/
```

![kubectl diff with diffyml](/diffyml/img/kubectl-demo.png)

`kubectl` passes two temporary directories of resource manifests; diffyml compares them recursively.

## What's next

- [Output Formats]({{< relref "/docs/formats" >}}) — pick `compact`, `brief`, `json`, or a CI-specific format
- [Filtering]({{< relref "/docs/filtering" >}}) — narrow the diff to paths you care about
- [CI Integration]({{< relref "/docs/ci" >}}) — wire diffyml into GitHub Actions, GitLab, Gitea, or git
- [Reference]({{< relref "/docs/reference" >}}) — every flag, generated from source
