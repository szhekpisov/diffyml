---
title: "Output Formats"
weight: 30
---

# Output Formats

diffyml supports eight output formats. Pick one with `-o` / `--output`.

| Format | Flag | Use case |
|--------|------|----------|
| [detailed]({{< relref "#detailed-default" >}}) | `-o detailed` (default) | Human review — full context |
| [compact]({{< relref "#compact" >}}) | `-o compact` | Quick scan of changes |
| [brief]({{< relref "#brief" >}}) | `-o brief` | Counts only |
| [github]({{< relref "#github" >}}) | `-o github` | GitHub Actions annotations |
| [gitlab]({{< relref "#gitlab" >}}) | `-o gitlab` | GitLab Code Quality JSON |
| [gitea]({{< relref "#gitea" >}}) | `-o gitea` | Gitea CI annotations |
| [json]({{< relref "#json" >}}) | `-o json` | Machine-readable, scriptable |
| [json-patch]({{< relref "#json-patch" >}}) | `-o json-patch` | RFC 6902 JSON Patch |

## detailed (default)

Human-readable terminal output with colors, paths, and surrounding context. Best for interactive use.

```bash
diffyml old.yaml new.yaml
```

## compact

One-line-per-change format. Good when you want a quick scan and don't need surrounding YAML context.

```bash
diffyml -o compact old.yaml new.yaml
```

## brief

Just the change counts. Useful when you only care whether something changed, not what.

```bash
diffyml -o brief old.yaml new.yaml
```

Pair with `--summary` to swap the bare counts for an AI-generated description (see [AI Summaries]({{< relref "/docs/ai-summary" >}})).

## github

Emits [GitHub Actions workflow commands](https://docs.github.com/en/actions/using-workflows/workflow-commands-for-github-actions) so changes show up as inline annotations on the PR diff.

```bash
diffyml -o github old.yaml new.yaml
```

To avoid spamming the UI, output is capped at 10 annotations per type. Combine with `-s` to fail the workflow when drift is detected.

## gitlab

Emits a [GitLab Code Quality](https://docs.gitlab.com/ee/ci/testing/code_quality.html) JSON report. Surface the report as a Code Quality artifact and GitLab will render diffs in the merge request UI.

```bash
diffyml -o gitlab old.yaml new.yaml > gl-code-quality.json
```

## gitea

Emits annotations in Gitea's GitHub-Actions-compatible format.

```bash
diffyml -o gitea old.yaml new.yaml
```

## json

Machine-readable JSON: a top-level array of `{path, type, from, to, document_index}` objects (with `file` added in directory mode). `type` is one of `added`, `removed`, `modified`, `order_changed`. Pipe into `jq` for scripted processing.

```bash
diffyml -o json old.yaml new.yaml | jq '.[] | select(.type == "modified")'
```

## json-patch

[RFC 6902 JSON Patch](https://datatracker.ietf.org/doc/html/rfc6902) — a sequence of `add`/`remove`/`replace` operations that, when applied to `from`, produce `to`. Useful for replaying changes programmatically.

```bash
diffyml -o json-patch old.yaml new.yaml
```
