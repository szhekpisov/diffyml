package cli

import (
	"flag"
	"testing"
)

// TestFlagDocsCoverage enforces that FlagDocs() and initFlags() stay in sync.
// Every registered flag must have a matching FlagDoc entry (by long or short name),
// and every FlagDoc entry must correspond to a registered flag.
func TestFlagDocsCoverage(t *testing.T) {
	cfg := NewCLIConfig()

	registered := make(map[string]bool)
	cfg.fs.VisitAll(func(f *flag.Flag) {
		registered[f.Name] = true
	})

	documented := make(map[string]bool)
	for _, d := range FlagDocs() {
		if d.Long == "" {
			t.Errorf("FlagDocs entry has empty Long name: %+v", d)
			continue
		}
		documented[d.Long] = true
		if d.Short != "" {
			documented[d.Short] = true
		}
	}

	for name := range registered {
		if !documented[name] {
			t.Errorf("flag --%s is registered but missing from FlagDocs()", name)
		}
	}
	for name := range documented {
		if !registered[name] {
			t.Errorf("FlagDocs() entry --%s has no corresponding registered flag", name)
		}
	}
}

// TestFlagDocsCategories ensures every FlagDoc entry has a Category set —
// generators rely on this for grouping.
func TestFlagDocsCategories(t *testing.T) {
	for _, d := range FlagDocs() {
		if d.Category == "" {
			t.Errorf("FlagDoc --%s missing Category", d.Long)
		}
		if d.Usage == "" {
			t.Errorf("FlagDoc --%s missing Usage", d.Long)
		}
		if d.Type == "" {
			t.Errorf("FlagDoc --%s missing Type", d.Long)
		}
	}
}
