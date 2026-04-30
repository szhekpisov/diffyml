package diffyml

import (
	"regexp"
	"strings"
	"testing"
)

// TestNeatPatterns_PerPatternInvariants walks every pattern in the default
// bundle and asserts: it compiles, it is anchored at start (^), and it carries
// a non-empty Label for --neat-explain output.
func TestNeatPatterns_PerPatternInvariants(t *testing.T) {
	all := NeatPatterns(DefaultNeatOptions())
	if len(all) == 0 {
		t.Fatal("DefaultNeatOptions() returned no patterns")
	}
	for _, p := range all {
		if _, err := regexp.Compile(p.Pattern); err != nil {
			t.Errorf("profile %s: pattern %q does not compile: %v", p.Profile, p.Pattern, err)
		}
		if !strings.HasPrefix(p.Pattern, "^") {
			t.Errorf("profile %s: pattern %q must be anchored with ^", p.Profile, p.Pattern)
		}
		if p.Label == "" {
			t.Errorf("profile %s: pattern %q has empty Label", p.Profile, p.Pattern)
		}
	}
}

// TestBuildNeatExcludeRegexp_BundleSelection verifies each bundle gate
// independently controls the resulting pattern count, including the all-on
// (default) and all-off cases.
func TestBuildNeatExcludeRegexp_BundleSelection(t *testing.T) {
	tests := []struct {
		name string
		opts NeatOptions
		want int
	}{
		{
			"default (all profiles)",
			DefaultNeatOptions(),
			len(neatProfileK8s) + len(neatProfileStatus) + len(neatProfileHelm) + len(neatProfileArgoCD) + len(neatProfileFlux),
		},
		{
			"no helm",
			NeatOptions{K8s: true, Status: true, ArgoCD: true, Flux: true},
			len(neatProfileK8s) + len(neatProfileStatus) + len(neatProfileArgoCD) + len(neatProfileFlux),
		},
		{
			"no argocd",
			NeatOptions{K8s: true, Status: true, Helm: true, Flux: true},
			len(neatProfileK8s) + len(neatProfileStatus) + len(neatProfileHelm) + len(neatProfileFlux),
		},
		{
			"no flux",
			NeatOptions{K8s: true, Status: true, Helm: true, ArgoCD: true},
			len(neatProfileK8s) + len(neatProfileStatus) + len(neatProfileHelm) + len(neatProfileArgoCD),
		},
		{
			"no status",
			NeatOptions{K8s: true, Helm: true, ArgoCD: true, Flux: true},
			len(neatProfileK8s) + len(neatProfileHelm) + len(neatProfileArgoCD) + len(neatProfileFlux),
		},
		{"k8s only", NeatOptions{K8s: true}, len(neatProfileK8s)},
		{"none", NeatOptions{}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildNeatExcludeRegexp(tt.opts)
			if len(got) != tt.want {
				t.Errorf("got %d patterns, want %d", len(got), tt.want)
			}
		})
	}
}

// TestNeatRegexes_NoOverlap verifies each pattern lives in exactly one profile.
// A pattern appearing in two profiles would mean disabling one bundle would
// not actually reduce the active regex set, breaking opt-out semantics.
func TestNeatRegexes_NoOverlap(t *testing.T) {
	seen := make(map[string]NeatProfile)
	for _, p := range NeatPatterns(DefaultNeatOptions()) {
		if existing, ok := seen[p.Pattern]; ok {
			t.Errorf("pattern %q appears in profiles %s and %s", p.Pattern, existing, p.Profile)
		}
		seen[p.Pattern] = p.Profile
	}
}

// matchesAny is a test helper: returns true if any pattern in the bundle
// matches the path.
func matchesAny(t *testing.T, patterns []string, path string) bool {
	t.Helper()
	for _, pat := range patterns {
		re, err := regexp.Compile(pat)
		if err != nil {
			t.Fatalf("invalid regex %q: %v", pat, err)
		}
		if re.MatchString(path) {
			return true
		}
	}
	return false
}

// TestNeatRegexes_MatchKnownPaths is the canonical match-table: each path is
// expected to be matched (or not) by the default neat bundle. Paths use
// DiffPath.String() format: dotted segments, [...]-wrapped keys with dots.
func TestNeatRegexes_MatchKnownPaths(t *testing.T) {
	patterns := BuildNeatExcludeRegexp(DefaultNeatOptions())

	mustMatch := []string{
		// K8s server
		"metadata.creationTimestamp",
		"metadata.generation",
		"metadata.resourceVersion",
		"metadata.uid",
		"metadata.managedFields",
		"metadata.managedFields[0]",
		"metadata.managedFields[0].fieldsV1",
		"metadata.managedFields[0].fieldsV1.f:spec",
		// kubectl
		"metadata.annotations[kubectl.kubernetes.io/last-applied-configuration]",
		"metadata.annotations[deployment.kubernetes.io/revision]",
		"metadata.annotations[pv.kubernetes.io/bind-completed]",
		// Status
		"status",
		"status.conditions[0].type",
		"status.replicas",
		"spec.nodeName",
		// Helm
		"metadata.annotations[meta.helm.sh/release-name]",
		"metadata.annotations[meta.helm.sh/release-namespace]",
		"metadata.labels[helm.sh/chart]",
		"metadata.labels[app.kubernetes.io/managed-by]",
		// ArgoCD
		"metadata.labels[argocd.argoproj.io/instance]",
		"metadata.labels[argocd.argoproj.io/secret-type]",
		"metadata.annotations[argocd.argoproj.io/tracking-id]",
		"metadata.annotations[argocd.argoproj.io/sync-wave]",
		"metadata.annotations[argocd.argoproj.io/sync-options]",
		"metadata.annotations[argocd.argoproj.io/hook]",
		"metadata.annotations[argocd.argoproj.io/hook-delete-policy]",
		"metadata.annotations[link.argocd.argoproj.io/external-link]",
		"metadata.annotations[pref.argocd.argoproj.io/default-pod-sort]",
		// Flux
		"metadata.labels[kustomize.toolkit.fluxcd.io/name]",
		"metadata.labels[kustomize.toolkit.fluxcd.io/namespace]",
		"metadata.annotations[kustomize.toolkit.fluxcd.io/checksum]",
		"metadata.annotations[helm.toolkit.fluxcd.io/driftDetection]",
	}
	for _, path := range mustMatch {
		if !matchesAny(t, patterns, path) {
			t.Errorf("expected path to be excluded by neat: %q", path)
		}
	}

	mustNotMatch := []string{
		// Real config that must survive
		"spec.replicas",
		"spec.template.spec.containers[0].image",
		"spec.template.spec.containers[0].env[0].value",
		"data",
		"stringData[password]",
		// User labels/annotations outside canonical noise prefixes
		"metadata.labels[app]",
		"metadata.labels[team]",
		"metadata.labels[app.kubernetes.io/name]",
		"metadata.labels[app.kubernetes.io/component]",
		"metadata.labels[app.kubernetes.io/version]",
		"metadata.annotations[my.company.io/owner]",
		"metadata.annotations[external-dns.alpha.kubernetes.io/hostname]",
		// Hypothetical fields named "status" but not the K8s status subtree
		"metadata.status",
		"spec.template.metadata.status",
		// metadata.name and metadata.namespace must NOT be neat-stripped
		"metadata.name",
		"metadata.namespace",
		// Non-Pod scheduler-set field with similar prefix
		"spec.nodeSelector[disktype]",
	}
	for _, path := range mustNotMatch {
		if matchesAny(t, patterns, path) {
			t.Errorf("expected path NOT to be excluded by neat: %q", path)
		}
	}
}

// TestNeatRegexes_DoesNotMatchRestartedAt is an explicit regression guard:
// kubectl rollout restart writes spec.template.metadata.annotations[
// kubectl.kubernetes.io/restartedAt] intentionally to force a pod cycle.
// Stripping it would hide a deliberate user action.
func TestNeatRegexes_DoesNotMatchRestartedAt(t *testing.T) {
	patterns := BuildNeatExcludeRegexp(DefaultNeatOptions())
	path := "spec.template.metadata.annotations[kubectl.kubernetes.io/restartedAt]"
	if matchesAny(t, patterns, path) {
		t.Errorf("kubectl.kubernetes.io/restartedAt MUST NOT be neat-stripped (intentional rollout marker): %q matched", path)
	}
}

// TestNeatRegexes_StatusSubtree exercises the status subtree pattern across
// realistic shapes.
func TestNeatRegexes_StatusSubtree(t *testing.T) {
	patterns := BuildNeatExcludeRegexp(NeatOptions{Status: true})
	must := []string{
		"status",
		"status.replicas",
		"status.conditions[0].type",
		"status.loadBalancer.ingress[0].hostname",
	}
	for _, p := range must {
		if !matchesAny(t, patterns, p) {
			t.Errorf("status pattern should match %q", p)
		}
	}
	mustNot := []string{
		// Not a status subtree — different roots
		"metadata.status",
		"spec.statusCheckPath",
	}
	for _, p := range mustNot {
		if matchesAny(t, patterns, p) {
			t.Errorf("status pattern should NOT match %q", p)
		}
	}
}

// TestNeatRegexes_OptOutTargetsRightProfile verifies disabling exactly one
// profile leaves a representative path of every other profile still matched
// while letting the disabled profile's path through.
func TestNeatRegexes_OptOutTargetsRightProfile(t *testing.T) {
	tests := []struct {
		name             string
		opts             NeatOptions
		stillMatched     []string
		shouldNowSurvive []string
	}{
		{
			name: "disable helm",
			opts: NeatOptions{K8s: true, Status: true, ArgoCD: true, Flux: true},
			stillMatched: []string{
				"metadata.managedFields",
				"metadata.annotations[argocd.argoproj.io/tracking-id]",
				"metadata.annotations[kustomize.toolkit.fluxcd.io/checksum]",
				"status.replicas",
			},
			shouldNowSurvive: []string{
				"metadata.annotations[meta.helm.sh/release-name]",
				"metadata.labels[helm.sh/chart]",
			},
		},
		{
			name: "disable argocd",
			opts: NeatOptions{K8s: true, Status: true, Helm: true, Flux: true},
			stillMatched: []string{
				"metadata.managedFields",
				"metadata.annotations[meta.helm.sh/release-name]",
			},
			shouldNowSurvive: []string{
				"metadata.annotations[argocd.argoproj.io/tracking-id]",
				"metadata.labels[argocd.argoproj.io/instance]",
			},
		},
		{
			name: "disable status",
			opts: NeatOptions{K8s: true, Helm: true, ArgoCD: true, Flux: true},
			stillMatched: []string{
				"metadata.managedFields",
			},
			shouldNowSurvive: []string{
				"status",
				"status.replicas",
				"spec.nodeName",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patterns := BuildNeatExcludeRegexp(tt.opts)
			for _, p := range tt.stillMatched {
				if !matchesAny(t, patterns, p) {
					t.Errorf("%q should still be matched", p)
				}
			}
			for _, p := range tt.shouldNowSurvive {
				if matchesAny(t, patterns, p) {
					t.Errorf("%q should now survive (profile disabled)", p)
				}
			}
		})
	}
}

// TestNeatPatterns_ProfileLabelsConsistent is a defensive check that each
// pattern has the profile its bundle declares (catches future copy-paste bugs).
func TestNeatPatterns_ProfileLabelsConsistent(t *testing.T) {
	check := func(t *testing.T, bundle []NeatPattern, want NeatProfile) {
		t.Helper()
		for _, p := range bundle {
			if p.Profile != want {
				t.Errorf("pattern %q in %s bundle has Profile=%s", p.Pattern, want, p.Profile)
			}
		}
	}
	check(t, neatProfileK8s, NeatProfileK8s)
	check(t, neatProfileStatus, NeatProfileStatus)
	check(t, neatProfileHelm, NeatProfileHelm)
	check(t, neatProfileArgoCD, NeatProfileArgoCD)
	check(t, neatProfileFlux, NeatProfileFlux)
}

// TestNeatPatterns_StableOrder verifies profile order is K8s, Status, Helm,
// ArgoCD, Flux. Tests that depend on neat-explain output order rely on this.
func TestNeatPatterns_StableOrder(t *testing.T) {
	patterns := NeatPatterns(DefaultNeatOptions())
	want := []NeatProfile{NeatProfileK8s, NeatProfileStatus, NeatProfileHelm, NeatProfileArgoCD, NeatProfileFlux}
	seen := make(map[NeatProfile]bool)
	idx := 0
	for _, p := range patterns {
		if !seen[p.Profile] {
			if p.Profile != want[idx] {
				t.Fatalf("profile order: at position %d got %s, want %s", idx, p.Profile, want[idx])
			}
			seen[p.Profile] = true
			idx++
		}
	}
	if idx != len(want) {
		t.Errorf("missing profiles in default bundle: saw only %d/%d", idx, len(want))
	}
}
