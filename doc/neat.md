# Neat mode (`--neat`)

`--neat` excludes paths injected by the Kubernetes API server, kubectl, Helm, ArgoCD, and Flux — the noise that dominates output of `kubectl diff`, `helm diff upgrade`, and ArgoCD-rendered manifest comparisons.

It is post-diff filtering, not manifest rewriting: the comparison runs unchanged, and the filter drops diffs whose path matches a curated regex bundle before output.

## Quick reference

```bash
diffyml --neat old.yaml new.yaml                # full bundle
diffyml --neat --no-neat-helm old.yaml new.yaml # keep Helm-injected diffs
diffyml --neat --neat-explain old.yaml new.yaml # report which patterns fired
```

`--neat-explain` reports hits **only for patterns in the curated neat bundle**. Hits for `--neat-strip-path` entries and user-supplied `--exclude-regexp` patterns are intentionally **not** included in the report — if you need to audit those, run a separate pass with `--exclude-regexp` removed.

## Profile bundles

`--neat` is the union of five profiles. Each can be opted out individually.

### `k8s` (server- and kubectl-injected) — always on with `--neat`

| Path pattern | Source |
|---|---|
| `metadata.creationTimestamp` | API server |
| `metadata.deletionTimestamp` | API server |
| `metadata.deletionGracePeriodSeconds` | API server |
| `metadata.generation` | API server |
| `metadata.resourceVersion` | API server |
| `metadata.selfLink` | API server (deprecated) |
| `metadata.uid` | API server |
| `metadata.managedFields` (entire subtree) | API server (server-side apply) |
| `metadata.annotations[kubectl.kubernetes.io/last-applied-configuration]` | kubectl |
| `metadata.annotations[deployment.kubernetes.io/revision]` | Deployment controller |
| `metadata.annotations[pv.kubernetes.io/bind-completed]` | PVC binding |
| `metadata.annotations[pv.kubernetes.io/bound-by-controller]` | PVC binding |

### `status` (server-set runtime state) — gated by `--no-neat-status`

| Path pattern | Source |
|---|---|
| `status` (entire subtree) | API server |
| `spec.nodeName` | Scheduler (Pods only) |

Disable with `--no-neat-status` when triaging HPA, Job completion, or Certificate readiness.

### `helm` — gated by `--no-neat-helm`

| Path pattern | Source |
|---|---|
| `metadata.annotations[meta.helm.sh/release-name]` | Helm |
| `metadata.annotations[meta.helm.sh/release-namespace]` | Helm |
| `metadata.labels[helm.sh/chart]` | Helm (cause of phantom diffs on chart bumps) |
| `metadata.labels[app.kubernetes.io/managed-by]` | Helm convention |

The `app.kubernetes.io/managed-by` label is stripped unconditionally — `--neat` is a path-based filter and does not inspect values. Even if your resource sets `managed-by: Kustomize` or `managed-by: my-tool`, the diff on that label will be suppressed under `--neat`. If you rely on this label for non-Helm semantics, disable the bundle with `--no-neat-helm`.

### `argocd` — gated by `--no-neat-argocd`

| Path pattern | Source |
|---|---|
| `metadata.labels[argocd.argoproj.io/instance]` | ArgoCD app tracking |
| `metadata.labels[argocd.argoproj.io/secret-type]` | ArgoCD secrets |
| `metadata.annotations[argocd.argoproj.io/*]` (any annotation in this namespace) | ArgoCD (tracking-id, sync-wave, sync-options, hook, hook-delete-policy, compare-options, manifest-generate-paths, refresh, etc.) |
| `metadata.annotations[link.argocd.argoproj.io/*]` | ArgoCD UI links |
| `metadata.annotations[pref.argocd.argoproj.io/*]` | ArgoCD UI preferences |

### `flux` — gated by `--no-neat-flux`

| Path pattern | Source |
|---|---|
| `metadata.labels[kustomize.toolkit.fluxcd.io/name]` | Flux Kustomize controller |
| `metadata.labels[kustomize.toolkit.fluxcd.io/namespace]` | Flux Kustomize controller |
| `metadata.annotations[kustomize.toolkit.fluxcd.io/*]` | Flux (checksum, prune, ssa, force, reconcile, substitute) |
| `metadata.annotations[helm.toolkit.fluxcd.io/*]` | Flux Helm controller |

## What `--neat` deliberately does NOT strip

- **`spec.template.metadata.annotations[kubectl.kubernetes.io/restartedAt]`** — `kubectl rollout restart` writes this *intentionally* to force a pod cycle. Stripping it would hide a deliberate user action.
- **`spec.template.metadata.creationTimestamp`** — the `null` PodTemplateSpec leak that kubectl emits. Covering it requires a `.*`-prefix regex that's a maintenance hazard. Add `--neat-strip-path '\.metadata\.creationTimestamp$'` if you need it gone.
- **`spec.replicas`** — real config.
- **`data` / `stringData`** on Secrets — real values; use `--mask-secrets` instead.
- **User-defined labels and annotations** outside the canonical noise prefixes (`app.kubernetes.io/name`, `app.kubernetes.io/component`, `external-dns.alpha.kubernetes.io/*`, your-company-prefixed annotations).
- **`metadata.name`, `metadata.namespace`** — identity.
- **`metadata.ownerReferences`** — controller relationships (the `uid` is noise but `kind`/`name`/`controller` are real).

## Combining with other flags

```bash
# Neat + Secret masking
diffyml --neat --mask-secrets manifest1.yaml manifest2.yaml

# Neat + custom additional exclusions
diffyml --neat --neat-strip-path '\.metadata\.creationTimestamp$' a.yaml b.yaml

# Neat + brief format for CI summaries
diffyml --neat -o brief --set-exit-code a.yaml b.yaml

# Neat in git-external-diff mode
GIT_EXTERNAL_DIFF='diffyml --neat' git diff
```

## Configuration file

```yaml
# .diffyml.yml
neat:
  enabled: true
  helm: true        # set false to drop the Helm bundle (== --no-neat-helm)
  argocd: true
  flux: true
  status: true
  strip-path:
    - '\.metadata\.creationTimestamp$'
  explain: false
```

CLI flags override config file values. The config file uses positive truth-tables (`helm: false` ⇒ drop helm); the CLI uses opt-out flags (`--no-neat-helm`).

## Profile stability

The bundle is `v1`. Future versions may add patterns (e.g. for new ArgoCD or Flux conventions). Rare regressions where a previously-shown diff is suddenly suppressed will be called out in the release notes; if you depend on a specific path being shown, add it to your `--filter` list explicitly.
