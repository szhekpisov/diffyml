package cli

import (
	"testing"

	"github.com/szhekpisov/diffyml/pkg/diffyml"
)

func TestCLIConfig_ToCompareOptions(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.IgnoreOrderChanges = true
	cfg.IgnoreWhitespaceChanges = true
	cfg.FormatStrings = true
	cfg.Swap = true
	cfg.Chroot = "data"

	opts := cfg.ToCompareOptions()

	if !opts.IgnoreOrderChanges {
		t.Error("expected IgnoreOrderChanges=true in Options")
	}
	if !opts.IgnoreWhitespaceChanges {
		t.Error("expected IgnoreWhitespaceChanges=true in Options")
	}
	if !opts.FormatStrings {
		t.Error("expected FormatStrings=true in Options")
	}
	if !opts.Swap {
		t.Error("expected Swap=true in Options")
	}
	if opts.Chroot != "data" {
		t.Errorf("expected Chroot='data', got %q", opts.Chroot)
	}
}

func TestCLIConfig_LineNumbersPropagation(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.LineNumbers = true

	if !cfg.ToCompareOptions().CaptureLineNumbers {
		t.Error("expected CaptureLineNumbers=true in Options")
	}
	if !cfg.ToFormatOptions().LineNumbers {
		t.Error("expected LineNumbers=true in FormatOptions")
	}
}

func TestCLIConfig_ToFilterOptions(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.Filter = []string{"config"}
	cfg.Exclude = []string{"secret"}
	cfg.FilterRegexp = []string{`^test\.`}
	cfg.ExcludeRegexp = []string{`password`}

	opts := cfg.ToFilterOptions()

	if len(opts.IncludePaths) != 1 || opts.IncludePaths[0] != "config" {
		t.Errorf("expected IncludePaths=['config'], got %v", opts.IncludePaths)
	}
	if len(opts.ExcludePaths) != 1 || opts.ExcludePaths[0] != "secret" {
		t.Errorf("expected ExcludePaths=['secret'], got %v", opts.ExcludePaths)
	}
	if len(opts.IncludeRegexp) != 1 {
		t.Errorf("expected IncludeRegexp length 1, got %d", len(opts.IncludeRegexp))
	}
	if len(opts.ExcludeRegexp) != 1 {
		t.Errorf("expected ExcludeRegexp length 1, got %d", len(opts.ExcludeRegexp))
	}
}

func TestCLIConfig_ToFormatOptions(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.OmitHeader = true
	cfg.UseGoPatchStyle = true
	cfg.MultiLineContextLines = 10

	opts := cfg.ToFormatOptions()

	if !opts.OmitHeader {
		t.Error("expected OmitHeader=true")
	}
	if !opts.UseGoPatchStyle {
		t.Error("expected UseGoPatchStyle=true")
	}
	if opts.ContextLines != 10 {
		t.Errorf("expected ContextLines=10, got %d", opts.ContextLines)
	}
}

func TestCLIConfig_ToFormatOptions_NoCertInspection(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.NoCertInspection = true

	opts := cfg.ToFormatOptions()

	if !opts.NoCertInspection {
		t.Error("expected NoCertInspection=true in FormatOptions when set in CLI config")
	}
}

func TestCLIConfig_ToFormatOptions_NoCertInspection_Default(t *testing.T) {
	cfg := NewCLIConfig()

	opts := cfg.ToFormatOptions()

	if opts.NoCertInspection {
		t.Error("expected NoCertInspection=false by default in FormatOptions")
	}
}

func TestCLIConfig_ToCompareOptions_IgnoreApiVersion(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.IgnoreApiVersion = true

	opts := cfg.ToCompareOptions()
	if !opts.IgnoreApiVersion {
		t.Error("expected Options.IgnoreApiVersion=true when CLIConfig.IgnoreApiVersion=true")
	}
}

func TestCLIConfig_ToCompareOptions_IgnoreApiVersion_Default(t *testing.T) {
	cfg := NewCLIConfig()

	opts := cfg.ToCompareOptions()
	if opts.IgnoreApiVersion {
		t.Error("expected Options.IgnoreApiVersion=false by default")
	}
}

func TestCLIConfig_ToNeatOptions_DefaultsAllOnExceptK8sAlwaysOn(t *testing.T) {
	cfg := NewCLIConfig()
	// All NoNeat* fields default to false; ToNeatOptions inverts them.
	opts := cfg.ToNeatOptions()
	if !opts.K8s || !opts.Status || !opts.Helm || !opts.ArgoCD || !opts.Flux {
		t.Errorf("expected every profile gate true by default, got %+v", opts)
	}
}

func TestCLIConfig_ToNeatOptions_PolarityInversion(t *testing.T) {
	tests := []struct {
		name       string
		setNoFlag  func(*CLIConfig)
		wantField  func(diffyml.NeatOptions) bool
		fieldLabel string
	}{
		{"NoNeatHelm flips Helm", func(c *CLIConfig) { c.NoNeatHelm = true }, func(o diffyml.NeatOptions) bool { return o.Helm }, "Helm"},
		{"NoNeatArgoCD flips ArgoCD", func(c *CLIConfig) { c.NoNeatArgoCD = true }, func(o diffyml.NeatOptions) bool { return o.ArgoCD }, "ArgoCD"},
		{"NoNeatFlux flips Flux", func(c *CLIConfig) { c.NoNeatFlux = true }, func(o diffyml.NeatOptions) bool { return o.Flux }, "Flux"},
		{"NoNeatStatus flips Status", func(c *CLIConfig) { c.NoNeatStatus = true }, func(o diffyml.NeatOptions) bool { return o.Status }, "Status"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewCLIConfig()
			tt.setNoFlag(cfg)
			opts := cfg.ToNeatOptions()
			if tt.wantField(opts) {
				t.Errorf("expected %s=false after NoNeat* set, got true", tt.fieldLabel)
			}
			// K8s is always on regardless.
			if !opts.K8s {
				t.Error("K8s gate should remain true regardless of opt-outs")
			}
		})
	}
}

func TestCLIConfig_ToFilterOptions_NeatOff_DoesNotAddPatterns(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.ExcludeRegexp = []string{`user-pattern`}
	// cfg.Neat is false

	opts := cfg.ToFilterOptions()
	if len(opts.ExcludeRegexp) != 1 || opts.ExcludeRegexp[0] != `user-pattern` {
		t.Errorf("expected only user-pattern when --neat off, got %v", opts.ExcludeRegexp)
	}
}

func TestCLIConfig_ToFilterOptions_NeatOn_PrependsBundle(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.Neat = true
	cfg.ExcludeRegexp = []string{`user-pattern`}

	opts := cfg.ToFilterOptions()
	neatBundle := diffyml.BuildNeatExcludeRegexp(diffyml.DefaultNeatOptions())
	wantLen := len(neatBundle) + 1
	if len(opts.ExcludeRegexp) != wantLen {
		t.Fatalf("expected %d patterns (%d neat + 1 user), got %d", wantLen, len(neatBundle), len(opts.ExcludeRegexp))
	}
	// Neat must come first; user pattern last.
	if opts.ExcludeRegexp[0] != neatBundle[0] {
		t.Errorf("expected neat bundle to lead: got %q at index 0, want %q", opts.ExcludeRegexp[0], neatBundle[0])
	}
	if opts.ExcludeRegexp[wantLen-1] != `user-pattern` {
		t.Errorf("expected user pattern last: got %q", opts.ExcludeRegexp[wantLen-1])
	}
}

func TestCLIConfig_ToFilterOptions_NeatOn_StripPathBetweenBundleAndUser(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.Neat = true
	cfg.NeatStripPath = []string{`extra-strip`}
	cfg.ExcludeRegexp = []string{`user-pattern`}

	opts := cfg.ToFilterOptions()
	neatBundle := diffyml.BuildNeatExcludeRegexp(diffyml.DefaultNeatOptions())

	// Layout: neat... ++ NeatStripPath ++ user ExcludeRegexp
	if got := opts.ExcludeRegexp[len(neatBundle)]; got != `extra-strip` {
		t.Errorf("expected NeatStripPath right after neat bundle at index %d, got %q", len(neatBundle), got)
	}
	if got := opts.ExcludeRegexp[len(neatBundle)+1]; got != `user-pattern` {
		t.Errorf("expected user pattern after NeatStripPath, got %q", got)
	}
}

func TestCLIConfig_ToFilterOptions_NeatOn_RespectsOptOuts(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.Neat = true
	cfg.NoNeatHelm = true

	opts := cfg.ToFilterOptions()
	withoutHelm := diffyml.BuildNeatExcludeRegexp(diffyml.NeatOptions{
		K8s: true, Status: true, ArgoCD: true, Flux: true,
	})
	if len(opts.ExcludeRegexp) != len(withoutHelm) {
		t.Errorf("expected %d patterns when --no-neat-helm set, got %d", len(withoutHelm), len(opts.ExcludeRegexp))
	}
}
