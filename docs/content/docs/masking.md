---
title: "Sensitive Value Masking"
weight: 65
---

# Sensitive Value Masking

Masking is **opt-in**. When enabled, diffyml replaces matching values with a placeholder (default `***`) before any output is rendered. The redacted value reaches every output format — `detailed`, `compact`, `brief`, `github`, `gitlab`, `gitea`, `json`, `json-patch` — and the AI summarizer prompt. Diffs still show *that* a value changed, just not what the value was.

## Auto-mask Kubernetes Secret data

`--mask-secrets` redacts the `data` and `stringData` fields of any document whose `kind` is `Secret`.

```bash
diffyml --mask-secrets secrets-old.yaml secrets-new.yaml
```

When an entire `Secret` document is added or removed, only its `data` / `stringData` subtrees are masked — `apiVersion`, `kind`, and `metadata` remain visible so the diff is still useful for review.

## Mask additional paths

`--mask-path` adds explicit dot-notation paths. Repeatable. Prefix matches are honored, so `data` masks every leaf under `data.*`.

```bash
diffyml \
  --mask-path 'data.api_key' \
  --mask-path 'data.db_url' \
  configmap-old.yaml configmap-new.yaml
```

The leading document-index prefix for multi-document files (`[0]`, `[1]`) is automatically stripped before matching, so `--mask-path 'data.password'` works whether the input is single- or multi-doc.

## Regex variant

`--mask-path-regexp` matches the same stripped path with a Go regular expression. Repeatable.

```bash
diffyml --mask-path-regexp '(?i)password|token|secret' old.yaml new.yaml
```

## Custom placeholder

```bash
diffyml --mask-secrets --mask-placeholder '<REDACTED>' old.yaml new.yaml
```

## Configure defaults via `.diffyml.yml`

Put masking on for every comparison in a repo:

```yaml
mask-secrets: true
mask-path:
  - "data.api_key"
mask-path-regexp:
  - "(?i)password"
mask-placeholder: "***"
```

## Pipeline order

```
Compare → MaskDifferences → Filter → Format
```

Masking runs **before** filtering so you can still `--exclude` or `--filter` redacted diffs if you want them out of the report entirely.

## What is *not* masked

- **Order-change diffs** — their values are identifier lists for diagnostic purposes, not secrets.
- **Path / key names** — only values are redacted. The path of a changed Secret field is still visible.
- **Length information** — the placeholder is fixed-length regardless of the original value's size.
