// neat.go - Curated noise-filter bundles for Kubernetes/Helm/ArgoCD/Flux diffs.
//
// Provides a stable v1 set of regex patterns that match noise paths injected
// by the K8s API server, kubectl, Helm, ArgoCD, and Flux. CLI exposes these
// via --neat; library consumers call BuildNeatExcludeRegexp directly.
//
// Patterns are written against DiffPath.String() output: dot-separated
// segments, with keys containing dots wrapped in [...] without quoting.
package diffyml

// NeatProfile names a single bundle of noise-filter patterns.
type NeatProfile string

const (
	// NeatProfileK8s covers paths injected by the Kubernetes API server and kubectl.
	NeatProfileK8s NeatProfile = "k8s"
	// NeatProfileStatus covers the status subtree and scheduler-set fields like spec.nodeName.
	NeatProfileStatus NeatProfile = "status"
	// NeatProfileHelm covers paths injected by Helm (release metadata, chart label).
	NeatProfileHelm NeatProfile = "helm"
	// NeatProfileArgoCD covers paths injected by ArgoCD (tracking-id, instance label, sync hooks).
	NeatProfileArgoCD NeatProfile = "argocd"
	// NeatProfileFlux covers paths injected by Flux (kustomize.toolkit, helm.toolkit).
	NeatProfileFlux NeatProfile = "flux"
)

// NeatOptions selects which neat profiles to apply.
type NeatOptions struct {
	K8s    bool
	Status bool
	Helm   bool
	ArgoCD bool
	Flux   bool
}

// DefaultNeatOptions returns the default --neat profile: every bundle enabled.
func DefaultNeatOptions() NeatOptions {
	return NeatOptions{K8s: true, Status: true, Helm: true, ArgoCD: true, Flux: true}
}

// NeatPattern annotates a single regex with its source profile and a
// human-readable label used by --neat-explain.
type NeatPattern struct {
	Profile NeatProfile
	Pattern string
	Label   string
}

// neatProfileK8s contains server- and kubectl-injected paths.
// These are stripped whenever NeatOptions.K8s is true.
var neatProfileK8s = []NeatPattern{
	{NeatProfileK8s, `^metadata\.creationTimestamp$`, "metadata.creationTimestamp"},
	{NeatProfileK8s, `^metadata\.deletionTimestamp$`, "metadata.deletionTimestamp"},
	{NeatProfileK8s, `^metadata\.deletionGracePeriodSeconds$`, "metadata.deletionGracePeriodSeconds"},
	{NeatProfileK8s, `^metadata\.generation$`, "metadata.generation"},
	{NeatProfileK8s, `^metadata\.resourceVersion$`, "metadata.resourceVersion"},
	{NeatProfileK8s, `^metadata\.selfLink$`, "metadata.selfLink"},
	{NeatProfileK8s, `^metadata\.uid$`, "metadata.uid"},
	{NeatProfileK8s, `^metadata\.managedFields(\..*|\[.*)?$`, "metadata.managedFields subtree"},
	{NeatProfileK8s, `^metadata\.annotations\[kubectl\.kubernetes\.io/last-applied-configuration\]$`, "kubectl.kubernetes.io/last-applied-configuration"},
	{NeatProfileK8s, `^metadata\.annotations\[deployment\.kubernetes\.io/revision\]$`, "deployment.kubernetes.io/revision"},
	{NeatProfileK8s, `^metadata\.annotations\[pv\.kubernetes\.io/bind-completed\]$`, "pv.kubernetes.io/bind-completed"},
	{NeatProfileK8s, `^metadata\.annotations\[pv\.kubernetes\.io/bound-by-controller\]$`, "pv.kubernetes.io/bound-by-controller"},
}

// neatProfileStatus contains the status subtree and scheduler-set fields.
// These are conceptually noise for config diffs but are sometimes useful
// when triaging live cluster state, so they live behind a separate gate.
var neatProfileStatus = []NeatPattern{
	{NeatProfileStatus, `^status(\..*|\[.*)?$`, "status subtree"},
	{NeatProfileStatus, `^spec\.nodeName$`, "spec.nodeName (Pod scheduler)"},
}

// neatProfileHelm contains paths injected by Helm.
// Note: app.kubernetes.io/managed-by is stripped unconditionally; users running
// a non-Helm chart that uses the same label will need --no-neat-helm.
var neatProfileHelm = []NeatPattern{
	{NeatProfileHelm, `^metadata\.annotations\[meta\.helm\.sh/release-name\]$`, "meta.helm.sh/release-name"},
	{NeatProfileHelm, `^metadata\.annotations\[meta\.helm\.sh/release-namespace\]$`, "meta.helm.sh/release-namespace"},
	{NeatProfileHelm, `^metadata\.labels\[helm\.sh/chart\]$`, "helm.sh/chart"},
	{NeatProfileHelm, `^metadata\.labels\[app\.kubernetes\.io/managed-by\]$`, "app.kubernetes.io/managed-by"},
}

// neatProfileArgoCD contains paths injected by ArgoCD.
var neatProfileArgoCD = []NeatPattern{
	{NeatProfileArgoCD, `^metadata\.labels\[argocd\.argoproj\.io/instance\]$`, "argocd.argoproj.io/instance label"},
	{NeatProfileArgoCD, `^metadata\.labels\[argocd\.argoproj\.io/secret-type\]$`, "argocd.argoproj.io/secret-type label"},
	{NeatProfileArgoCD, `^metadata\.annotations\[argocd\.argoproj\.io/[^\]]+\]$`, "argocd.argoproj.io/* annotations"},
	{NeatProfileArgoCD, `^metadata\.annotations\[link\.argocd\.argoproj\.io/[^\]]+\]$`, "link.argocd.argoproj.io/* annotations"},
	{NeatProfileArgoCD, `^metadata\.annotations\[pref\.argocd\.argoproj\.io/[^\]]+\]$`, "pref.argocd.argoproj.io/* annotations"},
}

// neatProfileFlux contains paths injected by Flux.
var neatProfileFlux = []NeatPattern{
	{NeatProfileFlux, `^metadata\.labels\[kustomize\.toolkit\.fluxcd\.io/name\]$`, "kustomize.toolkit.fluxcd.io/name label"},
	{NeatProfileFlux, `^metadata\.labels\[kustomize\.toolkit\.fluxcd\.io/namespace\]$`, "kustomize.toolkit.fluxcd.io/namespace label"},
	{NeatProfileFlux, `^metadata\.annotations\[kustomize\.toolkit\.fluxcd\.io/[^\]]+\]$`, "kustomize.toolkit.fluxcd.io/* annotations"},
	{NeatProfileFlux, `^metadata\.annotations\[helm\.toolkit\.fluxcd\.io/[^\]]+\]$`, "helm.toolkit.fluxcd.io/* annotations"},
}

// NeatPatterns returns the curated patterns for the enabled profiles.
// Order is stable: K8s, Status, Helm, ArgoCD, Flux. Disabled profiles are omitted.
func NeatPatterns(opts NeatOptions) []NeatPattern {
	total := 0
	if opts.K8s {
		total += len(neatProfileK8s)
	}
	if opts.Status {
		total += len(neatProfileStatus)
	}
	if opts.Helm {
		total += len(neatProfileHelm)
	}
	if opts.ArgoCD {
		total += len(neatProfileArgoCD)
	}
	if opts.Flux {
		total += len(neatProfileFlux)
	}
	out := make([]NeatPattern, 0, total)
	if opts.K8s {
		out = append(out, neatProfileK8s...)
	}
	if opts.Status {
		out = append(out, neatProfileStatus...)
	}
	if opts.Helm {
		out = append(out, neatProfileHelm...)
	}
	if opts.ArgoCD {
		out = append(out, neatProfileArgoCD...)
	}
	if opts.Flux {
		out = append(out, neatProfileFlux...)
	}
	return out
}

// BuildNeatExcludeRegexp returns the curated exclude-regexp list for the
// enabled bundles. Project NeatPatterns(opts) to the .Pattern field;
// suitable for direct use as FilterOptions.ExcludeRegexp.
func BuildNeatExcludeRegexp(opts NeatOptions) []string {
	patterns := NeatPatterns(opts)
	out := make([]string, len(patterns))
	for i, p := range patterns {
		out[i] = p.Pattern
	}
	return out
}
