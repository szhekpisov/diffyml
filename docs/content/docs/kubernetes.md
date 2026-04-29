---
title: "Kubernetes Support"
weight: 50
---

# Kubernetes Support

diffyml understands Kubernetes manifests by default. Resources are auto-detected and matched by `apiVersion` + `kind` + `metadata.name` (or `metadata.generateName`), so diffs stay meaningful even when document order changes.

```bash
diffyml manifests-v1.yaml manifests-v2.yaml
```

## Rename detection

When resources can't be matched by identifier — for example, `kustomize` `configMapGenerator` hash-suffix changes like `app-config-abc123` → `app-config-def456` — diffyml pairs unmatched documents by content similarity (60% threshold) and shows field-level diffs instead of bulk add/remove.

Disable with:

```bash
diffyml --detect-renames=false old.yaml new.yaml
```

## API version migration

`--ignore-api-version` drops `apiVersion` from the matching key, so an upgrade from `apps/v1beta1` to `apps/v1` produces field-level diffs rather than a remove + add.

```bash
diffyml --ignore-api-version manifests-v1.yaml manifests-v2.yaml
```

## Opting out

`--detect-kubernetes=false` disables Kubernetes-aware matching and compares documents by position only.

```bash
diffyml --detect-kubernetes=false file1.yaml file2.yaml
```

## Hiding Secret values

`--mask-secrets` redacts the `data` / `stringData` fields of `Secret` resources before any output is produced — useful when diffs land in CI logs or PR comments. See [Sensitive Value Masking]({{< relref "/docs/masking" >}}).

## kubectl external diff

See [CI Integration → kubectl external diff]({{< relref "/docs/ci#kubectl-external-diff" >}}) for the full setup.

```bash
export KUBECTL_EXTERNAL_DIFF="diffyml --omit-header --set-exit-code"
kubectl diff -f manifests/
```

## Directory comparison

diffyml accepts two directories as positional arguments. It recursively discovers regular files (regardless of extension), matches them by relative path, and shows aggregated differences. Non-YAML files are silently skipped.

```bash
diffyml dir-old/ dir-new/
```

This is what makes diffyml a drop-in `KUBECTL_EXTERNAL_DIFF` provider — kubectl passes extensionless temp files like `apps.v1.Deployment.default.nginx`, and diffyml discovers them automatically.
