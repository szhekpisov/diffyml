---
title: "Configuration File"
weight: 70
---

# Configuration File

diffyml loads project-level defaults from `.diffyml.yml` (or `.diffyml.yaml`) in the current directory. CLI flags override config file values. Use `--config` to specify a custom path.

```bash
diffyml --config /etc/diffyml/global.yml old.yaml new.yaml
```

## Example

```yaml
# .diffyml.yml
output: compact
ignore-order-changes: true
detect-kubernetes: false
filter:
  - spec.containers
exclude:
  - status
```

All CLI flags map to config keys in **kebab-case** (matching the long flag name). Unknown keys are rejected to catch typos.

## Full reference

```yaml
# Output
output: detailed        # detailed, compact, brief, github, gitlab, gitea, json, json-patch
color: auto             # always, never, auto
truecolor: auto         # always, never, auto

# Comparison
ignore-order-changes: false
ignore-whitespace-changes: false
format-strings: false
ignore-value-changes: false
detect-kubernetes: true
detect-renames: true
ignore-api-version: false
no-cert-inspection: false
swap: false

# Filtering (lists — CLI replaces these entirely if specified)
filter: []
exclude: []
filter-regexp: []
exclude-regexp: []
additional-identifier: []

# Display
omit-header: false
use-go-patch-style: false
multi-line-context-lines: 4
line-numbers: false

# Chroot
chroot: ""
chroot-of-from: ""
chroot-of-to: ""
chroot-list-to-documents: false

# Sensitive value masking (opt-in)
mask-secrets: false
mask-path: []
mask-path-regexp: []
mask-placeholder: "***"

# AI Summary (requires ANTHROPIC_API_KEY environment variable)
summary: false
summary-model: "claude-haiku-4-5-20251001"

# Exit code
set-exit-code: false
```

See [Sensitive Value Masking]({{< relref "/docs/masking" >}}) for usage.

## Custom colors

Diff colors can be customized for accessibility (e.g., colorblind-friendly palettes). Five color roles are configurable: `added`, `removed`, `modified`, `context`, and `doc-name`. Hex format (`#rrggbb`, `#rgb`).

```yaml
colors:
  added: "#58bf38"        # default: green
  removed: "#b9311b"      # default: red
  modified: "#c7c43f"     # default: yellow
  context: "#696969"      # default: gray
  doc-name: "#b0c4de"     # default: cyan
```

Example — Fabio Crameri Glasgow palette for colorblind accessibility:

```yaml
colors:
  added: "#6aa3a5"
  removed: "#702d06"
```

Environment variables override config file values:

```bash
export DIFFYML_COLOR_ADDED="#6aa3a5"
export DIFFYML_COLOR_REMOVED="#702d06"
diffyml old.yaml new.yaml
```

Available variables: `DIFFYML_COLOR_ADDED`, `DIFFYML_COLOR_REMOVED`, `DIFFYML_COLOR_MODIFIED`, `DIFFYML_COLOR_CONTEXT`, `DIFFYML_COLOR_DOC_NAME`.

## Precedence

Three layers, highest first:

1. **CLI flags** (e.g. `-o github`)
2. **Environment variables** (color overrides only)
3. **Config file** (`.diffyml.yml`)
