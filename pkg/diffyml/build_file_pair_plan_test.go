package diffyml

import (
	"path/filepath"
	"testing"
)

func TestBuildFilePairPlan_BothExist(t *testing.T) {
	fromDir := t.TempDir()
	toDir := t.TempDir()

	createFile(t, fromDir, "deploy.yaml", "a: 1")
	createFile(t, toDir, "deploy.yaml", "a: 2")

	pairs, err := BuildFilePairPlan(fromDir, toDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(pairs))
	}
	p := pairs[0]
	if p.Name != "deploy.yaml" {
		t.Errorf("expected Name='deploy.yaml', got %q", p.Name)
	}
	if p.Type != FilePairBothExist {
		t.Errorf("expected Type=FilePairBothExist, got %d", p.Type)
	}
	if p.FromPath == "" || p.ToPath == "" {
		t.Errorf("expected both paths non-empty, got FromPath=%q, ToPath=%q", p.FromPath, p.ToPath)
	}
}

func TestBuildFilePairPlan_OnlyFrom(t *testing.T) {
	fromDir := t.TempDir()
	toDir := t.TempDir()

	createFile(t, fromDir, "removed.yaml", "gone: true")

	pairs, err := BuildFilePairPlan(fromDir, toDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(pairs))
	}
	p := pairs[0]
	if p.Name != "removed.yaml" {
		t.Errorf("expected Name='removed.yaml', got %q", p.Name)
	}
	if p.Type != FilePairOnlyFrom {
		t.Errorf("expected Type=FilePairOnlyFrom, got %d", p.Type)
	}
	if p.FromPath == "" {
		t.Error("expected FromPath non-empty")
	}
	if p.ToPath != "" {
		t.Errorf("expected ToPath empty, got %q", p.ToPath)
	}
}

func TestBuildFilePairPlan_OnlyTo(t *testing.T) {
	fromDir := t.TempDir()
	toDir := t.TempDir()

	createFile(t, toDir, "added.yml", "new: true")

	pairs, err := BuildFilePairPlan(fromDir, toDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(pairs))
	}
	p := pairs[0]
	if p.Name != "added.yml" {
		t.Errorf("expected Name='added.yml', got %q", p.Name)
	}
	if p.Type != FilePairOnlyTo {
		t.Errorf("expected Type=FilePairOnlyTo, got %d", p.Type)
	}
	if p.FromPath != "" {
		t.Errorf("expected FromPath empty, got %q", p.FromPath)
	}
	if p.ToPath == "" {
		t.Error("expected ToPath non-empty")
	}
}

func TestBuildFilePairPlan_MixedScenario(t *testing.T) {
	fromDir := t.TempDir()
	toDir := t.TempDir()

	createFile(t, fromDir, "both.yaml", "a: 1")
	createFile(t, toDir, "both.yaml", "a: 2")
	createFile(t, fromDir, "removed.yaml", "b: 1")
	createFile(t, toDir, "added.yaml", "c: 1")

	pairs, err := BuildFilePairPlan(fromDir, toDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pairs) != 3 {
		t.Fatalf("expected 3 pairs, got %d: %+v", len(pairs), pairs)
	}

	// Should be sorted alphabetically
	expected := []struct {
		name     string
		pairType FilePairType
	}{
		{"added.yaml", FilePairOnlyTo},
		{"both.yaml", FilePairBothExist},
		{"removed.yaml", FilePairOnlyFrom},
	}

	for i, exp := range expected {
		if pairs[i].Name != exp.name {
			t.Errorf("pairs[%d].Name = %q, want %q", i, pairs[i].Name, exp.name)
		}
		if pairs[i].Type != exp.pairType {
			t.Errorf("pairs[%d].Type = %d, want %d", i, pairs[i].Type, exp.pairType)
		}
	}
}

func TestBuildFilePairPlan_AlphabeticalSorting(t *testing.T) {
	fromDir := t.TempDir()
	toDir := t.TempDir()

	createFile(t, fromDir, "z.yaml", "a: 1")
	createFile(t, fromDir, "a.yaml", "b: 2")
	createFile(t, fromDir, "m.yaml", "c: 3")
	createFile(t, toDir, "z.yaml", "a: 1")
	createFile(t, toDir, "a.yaml", "b: 2")
	createFile(t, toDir, "m.yaml", "c: 3")

	pairs, err := BuildFilePairPlan(fromDir, toDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pairs) != 3 {
		t.Fatalf("expected 3 pairs, got %d", len(pairs))
	}

	expectedNames := []string{"a.yaml", "m.yaml", "z.yaml"}
	for i, name := range expectedNames {
		if pairs[i].Name != name {
			t.Errorf("pairs[%d].Name = %q, want %q", i, pairs[i].Name, name)
		}
	}
}

func TestBuildFilePairPlan_BothDirectoriesEmpty(t *testing.T) {
	fromDir := t.TempDir()
	toDir := t.TempDir()

	pairs, err := BuildFilePairPlan(fromDir, toDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pairs) != 0 {
		t.Errorf("expected empty pairs, got %v", pairs)
	}
}

func TestBuildFilePairPlan_OneDirectoryEmpty(t *testing.T) {
	fromDir := t.TempDir()
	toDir := t.TempDir()

	createFile(t, fromDir, "deploy.yaml", "a: 1")
	createFile(t, fromDir, "service.yaml", "b: 2")

	pairs, err := BuildFilePairPlan(fromDir, toDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(pairs))
	}
	for _, p := range pairs {
		if p.Type != FilePairOnlyFrom {
			t.Errorf("expected FilePairOnlyFrom for %q, got %d", p.Name, p.Type)
		}
	}
}

func TestBuildFilePairPlan_ErrorOnInvalidFromDir(t *testing.T) {
	toDir := t.TempDir()
	_, err := BuildFilePairPlan("/nonexistent/dir", toDir)
	if err == nil {
		t.Error("expected error for non-existent from directory")
	}
}

func TestBuildFilePairPlan_ErrorOnInvalidToDir(t *testing.T) {
	fromDir := t.TempDir()
	_, err := BuildFilePairPlan(fromDir, "/nonexistent/dir")
	if err == nil {
		t.Error("expected error for non-existent to directory")
	}
}

func TestBuildFilePairPlan_FullPaths(t *testing.T) {
	fromDir := t.TempDir()
	toDir := t.TempDir()

	createFile(t, fromDir, "deploy.yaml", "a: 1")
	createFile(t, toDir, "deploy.yaml", "a: 2")

	pairs, err := BuildFilePairPlan(fromDir, toDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(pairs))
	}

	expectedFrom := fromDir + "/deploy.yaml"
	expectedTo := toDir + "/deploy.yaml"
	if pairs[0].FromPath != expectedFrom {
		t.Errorf("FromPath = %q, want %q", pairs[0].FromPath, expectedFrom)
	}
	if pairs[0].ToPath != expectedTo {
		t.Errorf("ToPath = %q, want %q", pairs[0].ToPath, expectedTo)
	}
}

func TestBuildFilePairPlan_NestedFiles(t *testing.T) {
	fromDir := t.TempDir()
	toDir := t.TempDir()

	createFile(t, fromDir, "ns-a/deploy.yaml", "a: 1")
	createFile(t, toDir, "ns-a/deploy.yaml", "a: 2")
	createFile(t, fromDir, "ns-a/removed.yaml", "b: 1")
	createFile(t, toDir, "ns-b/added.yaml", "c: 1")

	pairs, err := BuildFilePairPlan(fromDir, toDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pairs) != 3 {
		t.Fatalf("expected 3 pairs, got %d: %+v", len(pairs), pairs)
	}

	expected := []struct {
		name     string
		pairType FilePairType
	}{
		{"ns-a/deploy.yaml", FilePairBothExist},
		{"ns-a/removed.yaml", FilePairOnlyFrom},
		{"ns-b/added.yaml", FilePairOnlyTo},
	}

	for i, exp := range expected {
		if pairs[i].Name != exp.name {
			t.Errorf("pairs[%d].Name = %q, want %q", i, pairs[i].Name, exp.name)
		}
		if pairs[i].Type != exp.pairType {
			t.Errorf("pairs[%d].Type = %d, want %d", i, pairs[i].Type, exp.pairType)
		}
	}

	// Verify full paths are constructed correctly
	p := pairs[0]
	wantFrom := filepath.Join(fromDir, "ns-a/deploy.yaml")
	wantTo := filepath.Join(toDir, "ns-a/deploy.yaml")
	if p.FromPath != wantFrom {
		t.Errorf("FromPath = %q, want %q", p.FromPath, wantFrom)
	}
	if p.ToPath != wantTo {
		t.Errorf("ToPath = %q, want %q", p.ToPath, wantTo)
	}
}

func TestBuildFilePairPlan_SameBaseNameDifferentSubdirs(t *testing.T) {
	fromDir := t.TempDir()
	toDir := t.TempDir()

	// Same base name "deploy.yaml" in different subdirs — must NOT be matched
	createFile(t, fromDir, "ns-a/deploy.yaml", "a: 1")
	createFile(t, toDir, "ns-b/deploy.yaml", "a: 2")

	pairs, err := BuildFilePairPlan(fromDir, toDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d: %+v", len(pairs), pairs)
	}

	expected := []struct {
		name     string
		pairType FilePairType
	}{
		{"ns-a/deploy.yaml", FilePairOnlyFrom},
		{"ns-b/deploy.yaml", FilePairOnlyTo},
	}

	for i, exp := range expected {
		if pairs[i].Name != exp.name {
			t.Errorf("pairs[%d].Name = %q, want %q", i, pairs[i].Name, exp.name)
		}
		if pairs[i].Type != exp.pairType {
			t.Errorf("pairs[%d].Type = %d, want %d (%s)", i, pairs[i].Type, exp.pairType, exp.name)
		}
	}
}
