---
title: "AI Summaries"
weight: 80
---

# AI Summaries

Generate a natural-language summary of changes via the [Anthropic API](https://docs.anthropic.com/).

## Setup

Set your API key:

```bash
export ANTHROPIC_API_KEY="sk-ant-..."
```

Then run with `--summary` (or `-S`):

```bash
diffyml --summary old.yaml new.yaml
```

The summary is appended after the standard diff output. If the API call fails, a warning is printed to stderr and the diff output is preserved. **The exit code is never affected by summary success or failure.**

## With other formats

`--summary` composes with every output format. The notable variant: `brief + summary` swaps the bare counts for the AI description.

```bash
# Append AI summary after detailed diff
diffyml --summary old.yaml new.yaml

# Replace brief counts with AI summary
diffyml --summary -o brief old.yaml new.yaml

# Pipe-friendly: summary alongside JSON output
diffyml --summary -o json old.yaml new.yaml
```

## Choosing a model

The default is `claude-haiku-4-5-20251001` (Haiku 4.5) — fast and cheap, suitable for most diff-summarization workloads. Override with `--summary-model`:

```bash
diffyml --summary --summary-model claude-sonnet-4-5-20250514 old.yaml new.yaml
```

Use any model ID supported by the [Anthropic Messages API](https://docs.anthropic.com/en/docs/about-claude/models). Sonnet is a good upgrade for very large or nuanced diffs; Opus is overkill for almost all summarization.

## Persistent config

Set the model in `.diffyml.yml` to avoid passing it every time:

```yaml
summary: true
summary-model: claude-sonnet-4-5-20250514
```

## Cost and latency

The summary call is one HTTP request per `diffyml` invocation. With Haiku 4.5 and a typical diff (a few hundred lines), expect ~1 second of added latency and a fraction of a cent per call. Track usage in the [Anthropic Console](https://console.anthropic.com/).

## Privacy

Diff content is sent to Anthropic for summarization. **Do not enable `--summary` on inputs containing secrets you wouldn't share with a third-party LLM provider.** Combine with `--exclude-regexp '(?i)password|secret|token'` if you need a safety net.
