---
title: "Filtering & Chroot"
weight: 60
---

# Filtering & Chroot

Two complementary mechanisms: **filters** narrow the *output* (which differences are shown), **chroot** narrows the *input* (which subtree is compared).

## Filtering by path

```bash
# Only changes under spec.replicas
diffyml --filter spec.replicas old.yaml new.yaml

# Exclude noisy paths
diffyml --exclude metadata.annotations old.yaml new.yaml

# Both flags are repeatable
diffyml --exclude status --exclude metadata.managedFields old.yaml new.yaml
```

## Bracket syntax for keys with dots

When a key itself contains `.`, wrap it in brackets:

```bash
diffyml --exclude 'metadata.annotations[argocd.argoproj.io/tracking-id]' old.yaml new.yaml
```

Without the brackets diffyml would interpret `argocd.argoproj.io/tracking-id` as a four-level path, which isn't what you want.

## Regex filtering

```bash
# Keep only image-tag changes
diffyml --filter-regexp 'spec\.containers\[.*\]\.image' old.yaml new.yaml

# Drop anything matching "password" or "secret"
diffyml --exclude-regexp '(?i)password|secret' old.yaml new.yaml
```

`--filter-regexp` and `--exclude-regexp` are repeatable. Patterns must compile as Go [`regexp`](https://pkg.go.dev/regexp/syntax).

## Combining include and exclude

When both `--filter` and `--exclude` are given, `--exclude` wins. Same with the regex variants. Mixed include/exclude is fine — useful for `--filter spec --exclude spec.template.metadata.annotations` patterns.

With `--unchanged`, fully equal subtrees collapse to one atomic entry. Filters may still target descendants using numeric list indices (`containers.0.image`) or identifiers (`containers.app.image`); a descendant match keeps or removes the entire collapsed entry.

## Chroot — narrow the input

`--chroot <path>` re-roots **both** input documents at the given subtree before comparing.

```bash
# Compare only the spec subtree
diffyml --chroot spec deployment-v1.yaml deployment-v2.yaml
```

Apply different roots to each side with `--chroot-of-from` and `--chroot-of-to`:

```bash
diffyml \
  --chroot-of-from spec.template \
  --chroot-of-to spec \
  pod-spec.yaml deployment.yaml
```

`--chroot-list-to-documents` treats a list at the chroot as a sequence of documents (one per element), useful when comparing array-shaped manifests.

## Additional list identifiers

When matching list items between `from` and `to`, diffyml uses common identifier fields (`name`, `id`, etc.) by default. Add custom ones with `--additional-identifier`:

```bash
diffyml --additional-identifier email users-v1.yaml users-v2.yaml
```

The flag is repeatable. Useful when your list items are keyed by domain-specific fields (`uuid`, `slug`, `email`).
