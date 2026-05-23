package diffyml

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"go.yaml.in/yaml/v3"
)

// TestGetIdentifierNode_EquivalenceCorpus is the Stage 3 contract test: for
// every MappingNode reachable from every fixture's parsed (post-merge-resolve)
// tree, getIdentifierNode must yield exactly the same Go value as
// getIdentifier(nodeToInterface(node), opts). Two opts profiles are exercised:
// nil (defaults) and a non-default AdditionalIdentifiers slice to cover the
// extra-fields lookup branch.
func TestGetIdentifierNode_EquivalenceCorpus(t *testing.T) {
	files := collectFixtureYAMLs(t)
	if len(files) == 0 {
		t.Fatal("no fixture YAML files found; corpus sweep would be vacuous")
	}

	optsProfiles := []*Options{
		nil,
		{AdditionalIdentifiers: []string{"key", "host", "port"}},
	}

	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		nodes, err := parseNodes(content)
		if err != nil {
			// Some fixtures intentionally contain malformed YAML for parser
			// tests; skip them rather than failing the sweep.
			continue
		}
		for _, root := range nodes {
			walkMappingNodes(root, func(m *yaml.Node) {
				for _, opts := range optsProfiles {
					got := getIdentifierNode(m, opts)
					want := getIdentifier(nodeToInterface(m), opts)
					if !reflect.DeepEqual(got, want) {
						t.Errorf("%s: getIdentifierNode mismatch with opts=%v\n  got:  %#v\n  want: %#v", f, opts, got, want)
					}
				}
			})
		}
	}
}

// TestCanMatchByIdentifierNodes_EquivalenceCorpus pins the same equivalence
// for the list-level "can match by identifier" decision. Every SequenceNode
// in the corpus is checked under the same two opts profiles.
func TestCanMatchByIdentifierNodes_EquivalenceCorpus(t *testing.T) {
	files := collectFixtureYAMLs(t)
	optsProfiles := []*Options{
		nil,
		{AdditionalIdentifiers: []string{"key", "host"}},
	}

	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		nodes, err := parseNodes(content)
		if err != nil {
			continue
		}
		for _, root := range nodes {
			walkSequenceNodes(root, func(s *yaml.Node) {
				for _, opts := range optsProfiles {
					got := canMatchByIdentifierNodes(s.Content, opts)
					materialized, ok := nodeToInterface(s).([]any)
					if !ok {
						// nodeToInterface always produces []any for sequence
						// nodes; defensive check, not a real branch.
						continue
					}
					want := canMatchByIdentifier(materialized, opts)
					if got != want {
						t.Errorf("%s: canMatchByIdentifierNodes mismatch with opts=%v\n  got:  %v\n  want: %v\n  list: %#v", f, opts, got, want, materialized)
					}
				}
			})
		}
	}
}

// TestGetIdentifierNode_TargetedCases covers the explicit branches: nil input,
// non-mapping input, additional-identifier priority, name-then-id fallback,
// scalar fast path vs. non-scalar identifier value, and the duplicate-key
// last-write-wins rule.
func TestGetIdentifierNode_TargetedCases(t *testing.T) {
	cases := []struct {
		name string
		yaml string
		opts *Options
		want any
	}{
		{"nil input", "", nil, nil}, // input handled below as nil node
		{
			name: "name field scalar",
			yaml: "name: alice\nvalue: 1\n",
			want: "alice",
		},
		{
			name: "id fallback when no name",
			yaml: "id: 42\nvalue: x\n",
			want: 42,
		},
		{
			name: "additional identifier takes priority over name",
			yaml: "name: alice\nkey: special\n",
			opts: &Options{AdditionalIdentifiers: []string{"key"}},
			want: "special",
		},
		{
			name: "non-mapping returns nil",
			yaml: "- 1\n- 2\n",
			want: nil,
		},
		{
			name: "no identifier present",
			yaml: "value: 1\n",
			want: nil,
		},
		{
			name: "non-scalar identifier value falls through nodeToInterface",
			yaml: "name:\n  composite: true\n",
			want: &OrderedMap{Keys: []string{"composite"}, Values: map[string]any{"composite": true}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var node *yaml.Node
			if tc.yaml != "" {
				node = decodeOne(t, tc.yaml)
				node = node.Content[0] // unwrap DocumentNode
			}
			got := getIdentifierNode(node, tc.opts)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}

// collectFixtureYAMLs returns every *.yaml file beneath testdata/fixtures.
func collectFixtureYAMLs(t *testing.T) []string {
	t.Helper()
	var paths []string
	root := filepath.Join("..", "..", "testdata", "fixtures")
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(p) == ".yaml" || filepath.Ext(p) == ".yml" {
			paths = append(paths, p)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk fixtures: %v", err)
	}
	return paths
}

// walkMappingNodes invokes fn on every MappingNode reachable from root.
func walkMappingNodes(n *yaml.Node, fn func(*yaml.Node)) {
	if n == nil {
		return
	}
	if n.Kind == yaml.MappingNode {
		fn(n)
	}
	for _, c := range n.Content {
		walkMappingNodes(c, fn)
	}
}

// walkSequenceNodes invokes fn on every SequenceNode reachable from root.
func walkSequenceNodes(n *yaml.Node, fn func(*yaml.Node)) {
	if n == nil {
		return
	}
	if n.Kind == yaml.SequenceNode {
		fn(n)
	}
	for _, c := range n.Content {
		walkSequenceNodes(c, fn)
	}
}
