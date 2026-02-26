package diffyml

import (
	"io"
	"testing"
)

// --- Corpus helpers ---

// yamlCorpus returns representative YAML snippets for seeding fuzz targets.
func yamlCorpus() [][]byte {
	return [][]byte{
		// simple key-value
		[]byte("key: value\n"),
		// empty document
		[]byte(""),
		// multi-document
		[]byte("---\na: 1\n---\nb: 2\n"),
		// list
		[]byte("items:\n  - one\n  - two\n  - three\n"),
		// anchors and merge keys
		[]byte("defaults: &defaults\n  adapter: postgres\n  host: localhost\ndev:\n  <<: *defaults\n  database: dev_db\n"),
		// nested maps
		[]byte("root:\n  child:\n    grandchild: value\n    list:\n      - a\n      - b\n"),
		// Kubernetes resource
		[]byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: my-config\n  namespace: default\ndata:\n  key1: value1\n  key2: value2\n"),
		// JSON-compatible YAML
		[]byte("{\"name\": \"test\", \"items\": [1, 2, 3]}\n"),
		// scalars: booleans, nulls, numbers
		[]byte("bool_true: true\nbool_false: false\nnull_val: null\nint_val: 42\nfloat_val: 3.14\n"),
		// multiline strings
		[]byte("literal: |\n  line1\n  line2\nfolded: >\n  line1\n  line2\n"),
	}
}

// yamlPairs returns pairs of YAML snippets showing various diff scenarios.
func yamlPairs() [][2][]byte {
	return [][2][]byte{
		// modification
		{[]byte("key: old\n"), []byte("key: new\n")},
		// addition
		{[]byte("a: 1\n"), []byte("a: 1\nb: 2\n")},
		// removal
		{[]byte("a: 1\nb: 2\n"), []byte("a: 1\n")},
		// multi-doc diff
		{[]byte("---\na: 1\n---\nb: 2\n"), []byte("---\na: 1\n---\nb: 3\n")},
		// list reorder
		{[]byte("items:\n  - a\n  - b\n  - c\n"), []byte("items:\n  - c\n  - a\n  - b\n")},
		// Kubernetes resource change
		{
			[]byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: cfg\ndata:\n  k: v1\n"),
			[]byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: cfg\ndata:\n  k: v2\n"),
		},
		// empty to content
		{[]byte(""), []byte("key: value\n")},
		// identical
		{[]byte("same: same\n"), []byte("same: same\n")},
	}
}

// --- Fuzz targets ---

// FuzzCompare exercises the full Compare pipeline: parse, compare, diff.
func FuzzCompare(f *testing.F) {
	// Seed with single-input corpus
	for _, b := range yamlCorpus() {
		f.Add(b, b)
	}
	// Seed with diff pairs
	for _, p := range yamlPairs() {
		f.Add(p[0], p[1])
	}

	f.Fuzz(func(t *testing.T, from, to []byte) {
		// Without options
		Compare(from, to, nil) //nolint:errcheck

		// With Kubernetes detection
		Compare(from, to, &Options{DetectKubernetes: true}) //nolint:errcheck
	})
}

// FuzzCompareWithOptions exercises Compare with all Options flag combinations.
func FuzzCompareWithOptions(f *testing.F) {
	for _, p := range yamlPairs() {
		f.Add(p[0], p[1], uint8(0), "")
	}
	f.Add([]byte("a: 1\n"), []byte("a: 2\n"), uint8(0xFF), "data")

	f.Fuzz(func(t *testing.T, from, to []byte, flags uint8, chroot string) {
		opts := &Options{
			IgnoreOrderChanges:      flags&(1<<0) != 0,
			IgnoreWhitespaceChanges: flags&(1<<1) != 0,
			IgnoreValueChanges:      flags&(1<<2) != 0,
			DetectKubernetes:        flags&(1<<3) != 0,
			DetectRenames:           flags&(1<<4) != 0,
			NoCertInspection:        flags&(1<<5) != 0,
			Swap:                    flags&(1<<6) != 0,
			ChrootListToDocuments:   flags&(1<<7) != 0,
			Chroot:                  chroot,
		}
		Compare(from, to, opts) //nolint:errcheck
	})
}

// FuzzParseWithOrder exercises the core YAML parser in isolation.
func FuzzParseWithOrder(f *testing.F) {
	for _, b := range yamlCorpus() {
		f.Add(b)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		ParseWithOrder(data) //nolint:errcheck
	})
}

// FuzzDocumentParser exercises the streaming document parser state machine.
func FuzzDocumentParser(f *testing.F) {
	for _, b := range yamlCorpus() {
		f.Add(b)
	}
	// Seed with many document separators
	f.Add([]byte("---\n---\n---\n---\n---\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		p := NewDocumentParser(data)
		// Cap iterations to prevent DoS on inputs with many "---" separators.
		for i := 0; i < 10_000; i++ {
			_, err := p.Next()
			if err == io.EOF {
				break
			}
		}
	})
}
