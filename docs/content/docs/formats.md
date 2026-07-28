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

Every difference becomes exactly one annotation, so multiline values are kept bounded:

- A **changed** multiline value (a ConfigMap block scalar, an embedded `values.yaml`) is rendered as a line diff showing only the changed lines plus `--multi-line-context-lines` of context on each side. Every other run of unchanged lines collapses into a `[N lines unchanged]` marker. Collapsing only removes *unchanged* lines, so the resulting diff is capped at 40 lines to bound a value whose lines all changed.
- Any other value — **added**, **removed**, **unchanged**, or a **changed** pair that is not a multiline string — has nothing to diff against, so each value is truncated to its first 20 lines.

Both caps append a `[N more lines]` marker counting the lines of the value that were dropped, so a hidden `[24 lines unchanged]` run counts as the 24 lines it stands for. The `(N inserts, M deletions)` header always counts the whole change even when the diff below it is truncated.

Individual lines are capped too, at 500 characters with a `[N more characters]` marker. Line counts alone bound the wrong dimension: a minified JSON blob or a `last-applied-configuration` annotation is a single line of any size, so it clears every line cap untouched. The count is in characters rather than bytes, so a multi-byte character is never cut in half.

A value rewritten beyond recognition — more than 80 lines of difference between the two sides — is truncated like any other value instead of being diffed. Producing a line diff costs memory proportional to how different the two values are, and past that point the diff no longer fits in the 40-line cap anyway. Only wholesale rewrites reach it: a long value with a few changed lines is diffed however long it is.

PEM certificate values are replaced by the same one-line `Certificate(CN=…, Issuer=…, Valid=…, Serial=…)` summary the [detailed](#detailed-default) output uses, so a rotation reads as one changed line instead of a diff of base64. Pass `--no-cert-inspection` to see the raw PEM (still subject to the caps above).

Annotation text is percent-encoded per the workflow command spec (`%` → `%25`, `\r` → `%0D`, `\n` → `%0A`). GitHub renders the escapes as line breaks in the annotation; a raw newline would instead terminate the command and spill the rest into the build log. Property values such as `file=` additionally encode `:` → `%3A` and `,` → `%2C`, since those delimit the property list — an unescaped comma in a path would swallow the annotation title.

## gitlab

Emits a [GitLab Code Quality](https://docs.gitlab.com/ee/ci/testing/code_quality.html) JSON report. Surface the report as a Code Quality artifact and GitLab will render diffs in the merge request UI.

```bash
diffyml -o gitlab old.yaml new.yaml > gl-code-quality.json
```

Unlike the GitHub annotations above, descriptions here are **not** truncated. The report is JSON, so an embedded newline is escaped rather than terminating anything, and each entry's fingerprint is a hash of its description — bounding a description would change every fingerprint, making GitLab re-report existing findings as new, and two values sharing a truncated prefix would collide onto one fingerprint.

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
