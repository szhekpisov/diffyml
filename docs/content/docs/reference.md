---
title: "Reference"
description: "Complete reference for every diffyml command-line flag, auto-generated from the Go source."
weight: 90
---

This page is auto-generated from `pkg/diffyml/cli/flag_metadata.go` by `make docs-gen`.
Do not edit by hand — your changes will be overwritten on the next build.

```
diffyml [flags] <from> <to>
```

## Output

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-o`, `--output` | `string` | `detailed` | specify output style: compact, brief, github, gitlab, gitea, json, json-patch, detailed |
| `-c`, `--color` | `string` | `auto` | specify color usage: always, never, or auto |
| `-t`, `--truecolor` | `string` | `auto` | specify true color usage: always, never, or auto |

## Comparison

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-i`, `--ignore-order-changes` | `bool` | — | ignore order changes in lists |
| `--ignore-whitespace-changes` | `bool` | — | ignore leading or trailing whitespace changes |
| `--format-strings` | `bool` | — | canonicalize embedded JSON strings before comparison |
| `-v`, `--ignore-value-changes` | `bool` | — | exclude changes in values |
| `--detect-kubernetes` | `bool` | `true` | detect kubernetes entities |
| `--detect-renames` | `bool` | `true` | enable detection for renames |
| `--ignore-api-version` | `bool` | — | ignore apiVersion when matching Kubernetes resources |
| `-x`, `--no-cert-inspection` | `bool` | — | disable x509 certificate inspection |
| `--swap` | `bool` | — | swap 'from' and 'to' for comparison |
| `-u`, `--unchanged` | `bool` | — | report keys equal between both files (inverse diff) |

## Filtering

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--filter` | `list` | — | filter reports to a subset of differences (repeatable) |
| `--exclude` | `list` | — | exclude reports from a set of differences (repeatable) |
| `--filter-regexp` | `list` | — | filter reports using regular expressions (repeatable) |
| `--exclude-regexp` | `list` | — | exclude reports using regular expressions (repeatable) |
| `--additional-identifier` | `list` | — | use additional identifier in named entry lists (repeatable) |

## Neat

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--neat` | `bool` | — | exclude well-known noisy K8s/Helm/ArgoCD/Flux paths |
| `--no-neat-helm` | `bool` | — | with --neat: keep Helm-injected paths |
| `--no-neat-argocd` | `bool` | — | with --neat: keep ArgoCD-injected paths |
| `--no-neat-flux` | `bool` | — | with --neat: keep Flux-injected paths |
| `--no-neat-status` | `bool` | — | with --neat: keep .status subtree and spec.nodeName |
| `--neat-explain` | `bool` | — | print neat exclude regexes that fired (to stderr) |
| `--neat-strip-path` | `list` | — | additional regex appended to the neat bundle (requires --neat; repeatable) |

## Masking

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--mask-secrets` | `bool` | — | auto-mask data/stringData of Kubernetes Secret resources |
| `--mask-path` | `list` | — | additional path to mask (dot-notation, prefix match; repeatable) |
| `--mask-path-regexp` | `list` | — | additional path to mask (regex; repeatable) |
| `--mask-placeholder` | `string` | `***` | placeholder for masked values |

## Display

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-b`, `--omit-header` | `bool` | — | omit the diffyml summary header |
| `-g`, `--use-go-patch-style` | `bool` | — | use Go-Patch style paths in outputs |
| `--multi-line-context-lines` | `int` | `4` | context lines for multi-line strings |

## Chroot

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--chroot` | `string` | — | change the root level of the input file |
| `--chroot-of-from` | `string` | — | only change the root level of the from input file |
| `--chroot-of-to` | `string` | — | only change the root level of the to input file |
| `--chroot-list-to-documents` | `bool` | — | treat chroot list as set of documents |

## AI Summary

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-S`, `--summary` | `bool` | — | enable AI-powered summary of differences (requires ANTHROPIC_API_KEY) |
| `--summary-model` | `string` | `claude-haiku-4-5-20251001` | specify Anthropic model for summary |

## Configuration

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | `string` | `.diffyml.yml` | path to config file |

## Other

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-s`, `--set-exit-code` | `bool` | — | set program exit code based on differences |
| `-h`, `--help` | `bool` | — | show help |

## Version

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-V`, `--version` | `bool` | — | show version information |

`--version` is handled in `main.go` before flag parsing, so it works even
when other arguments are missing or invalid.
