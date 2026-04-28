---
title: "CI Integration"
weight: 40
---

# CI Integration

## Exit codes

`-s` / `--set-exit-code` is the building block for CI usage:

| Exit code | Meaning |
|-----------|---------|
| `0` | No differences (or success without `-s`) |
| `1` | Differences detected (only with `-s`) |
| `255` | Error occurred |

```bash
diffyml -s before.yaml after.yaml || echo "Config drift detected"
```

## GitHub Actions

The recommended path is the [`diffyml-action`](https://github.com/szhekpisov/diffyml-action) composite action — no manual binary install needed.

```yaml
- uses: szhekpisov/diffyml-action@v1
  with:
    from: old.yaml
    to: new.yaml
```

To inspect drift without failing the job, read the `has-differences` output:

```yaml
- uses: szhekpisov/diffyml-action@v1
  id: diff
  with:
    from: old.yaml
    to: new.yaml
    fail-on-diff: 'false'
    output: github

- if: steps.diff.outputs.has-differences == 'true'
  run: echo "Configuration drift detected"
```

See the [action repo](https://github.com/szhekpisov/diffyml-action) for the full list of inputs and outputs.

If you'd rather call the binary directly, use `-o github` so changes appear as inline PR annotations:

```yaml
- run: |
    curl -L https://github.com/szhekpisov/diffyml/releases/latest/download/diffyml_linux_amd64.tar.gz | tar xz
    ./diffyml -o github -s old.yaml new.yaml
```

## GitLab Code Quality

```yaml
diffyml:
  stage: test
  script:
    - diffyml -o gitlab old.yaml new.yaml > gl-code-quality.json
  artifacts:
    reports:
      codequality: gl-code-quality.json
```

GitLab renders the report inline on merge requests.

## Gitea Actions

```yaml
- run: diffyml -o gitea -s old.yaml new.yaml
```

Gitea consumes the same workflow-command format as GitHub Actions, so annotations appear inline on the diff view.

## kubectl external diff

`kubectl diff` invokes whatever program is in `KUBECTL_EXTERNAL_DIFF`, passing two temporary directories. diffyml auto-detects directory mode and recursively compares the manifests:

```bash
export KUBECTL_EXTERNAL_DIFF="diffyml --omit-header --set-exit-code"
kubectl diff -f manifests/
```

## Git external diff

diffyml can act as `GIT_EXTERNAL_DIFF`. Git passes 7–9 positional arguments per file pair, which diffyml auto-detects. Non-YAML files are skipped with a warning.

One-off:

```bash
GIT_EXTERNAL_DIFF=diffyml git diff
GIT_EXTERNAL_DIFF='diffyml -o compact' git diff
```

Permanent — register a custom driver via `.gitattributes` so other file types still use git's built-in diff:

```gitattributes
*.yaml diff=diffyml
*.yml  diff=diffyml
```

```bash
git config diff.diffyml.command diffyml
```

Notes:

- Color and truecolor are auto-forced (git pipes external-diff output through its pager).
- `--set-exit-code` is silently ignored — git aborts external diff on non-zero exit.
- Parse errors are non-fatal: a warning prints to stderr and git moves on to the next file.

## Pre-commit

```yaml
# .pre-commit-config.yaml
repos:
  - repo: local
    hooks:
      - id: diffyml-staged
        name: diffyml drift check
        language: system
        entry: bash -c 'diffyml -s -o brief HEAD HEAD~1 || true'
        files: '\.ya?ml$'
```

Customize to taste — pre-commit hooks for diffing don't have a one-size-fits-all shape.
