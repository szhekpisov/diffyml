// Copyright © 2019 The Homeport Team
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package diffyml_test

import (
	"strings"
	"testing"

	"github.com/szhekpisov/diffyml/pkg/diffyml"
)

func TestCompare(t *testing.T) {
	tests := []struct {
		name    string
		from    string
		to      string
		opts    *diffyml.Options
		wantErr bool
		check   func(t *testing.T, diffs []diffyml.Difference)
	}{
		{
			name: "string value modified",
			from: `---
some:
  yaml:
    structure:
      name: foobar
      version: v1
`,
			to: `---
some:
  yaml:
    structure:
      name: fOObAr
      version: v1
`,
			check: mod("foobar", "fOObAr"),
		},
		{
			name: "integer modified",
			from: `---
some:
  yaml:
    structure:
      name: 1
      version: v1
`,
			to: `---
some:
  yaml:
    structure:
      name: 2
      version: v1
`,
			check: hasTypes(diffyml.DiffModified),
		},
		{
			name: "float modified",
			from: `---
some:
  yaml:
    structure:
      name: foobar
      level: 3.14159265359
`,
			to: `---
some:
  yaml:
    structure:
      name: foobar
      level: 2.7182818284
`,
			check: hasTypes(),
		},
		{
			name: "boolean modified",
			from: `---
some:
  yaml:
    structure:
      name: foobar
      enabled: false
`,
			to: `---
some:
  yaml:
    structure:
      name: foobar
      enabled: true
`,
			check: hasTypes(),
		},
		{
			name: "value added",
			from: `---
some:
  yaml:
    structure:
      name: foobar
`,
			to: `---
some:
  yaml:
    structure:
      name: foobar
      version: v1
`,
			check: hasTypes(diffyml.DiffAdded),
		},
		{
			name: "value removed",
			from: `---
some:
  yaml:
    structure:
      name: foobar
      version: v1
`,
			to: `---
some:
  yaml:
    structure:
      name: foobar
`,
			check: hasTypes(diffyml.DiffRemoved),
		},
		{
			name: "value removed and another added",
			from: `---
some:
  yaml:
    structure:
      name: foobar
      version: v1
`,
			to: `---
some:
  yaml:
    structure:
      name: foobar
      release: v1
`,
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) < 2 {
					t.Fatal("expected at least 2 diffs")
				}
				if !hasDiffType(diffs, diffyml.DiffRemoved) {
					t.Error("expected a removal diff")
				}
				if !hasDiffType(diffs, diffyml.DiffAdded) {
					t.Error("expected an addition diff")
				}
			},
		},
		{
			name: "ignore whitespace changes",
			from: `{"foo": "bar"}`,
			to:   `{"foo": "bar "}`,
			opts: &diffyml.Options{IgnoreWhitespaceChanges: true},
			check: noDiffs(),
		},
		{
			name: "string list entry added",
			from: `---
some:
  yaml:
    structure:
      list:
      - one
      - two
`,
			to: `---
some:
  yaml:
    structure:
      list:
      - one
      - two
      - three
`,
			check: hasTypes(),
		},
		{
			name: "integer list entry added",
			from: `---
some:
  yaml:
    structure:
      list:
      - 1
      - 2
`,
			to: `---
some:
  yaml:
    structure:
      list:
      - 1
      - 2
      - 3
`,
			check: hasTypes(),
		},
		{
			name: "string list entry removed",
			from: `---
some:
  yaml:
    structure:
      list:
      - one
      - two
      - three
`,
			to: `---
some:
  yaml:
    structure:
      list:
      - one
      - two
`,
			check: hasTypes(),
		},
		{
			name: "integer list entry removed",
			from: `---
some:
  yaml:
    structure:
      list:
      - 1
      - 2
      - 3
`,
			to: `---
some:
  yaml:
    structure:
      list:
      - 1
      - 2
`,
			check: hasTypes(),
		},
		{
			name: "hash order change only - no diff",
			from: `---
list:
- enabled: true
- foo: bar
  version: 1
`,
			to: `---
list:
- enabled: true
- version: 1
  foo: bar
`,
			check: noDiffs(),
		},
		{
			name: "nested structure differences",
			from: `---
instance_groups:
- name: web
  instances: 1
  networks:
  - name: concourse
    static_ips: 192.168.1.1
`,
			to: `---
instance_groups:
- name: web
  instances: 1
  networks:
  - name: concourse
    static_ips: 192.168.0.1
`,
			check: mod("192.168.1.1", "192.168.0.1"),
		},
		{
			name: "list as root",
			from: `---
- name: one
  version: 1
- name: two
  version: 2
- name: three
  version: 4
`,
			to: `---
- name: one
  version: 1
- name: two
  version: 2
- name: three
  version: 3
`,
			check: hasTypes(),
		},
		{
			name: "identical YAMLs - no diff",
			from: `---
foo:
  bar: baz
  list:
  - one
  - two
`,
			to: `---
foo:
  bar: baz
  list:
  - one
  - two
`,
			check: noDiffs(),
		},
		{
			name: "null to value change",
			from: `foo: null`,
			to:   `foo: "bar"`,
			check: hasTypes(),
		},
		{
			name: "value to null change",
			from: `foo: "bar"`,
			to:   `foo: null`,
			check: hasTypes(diffyml.DiffModified),
		},
		{
			name: "type change string to map",
			from: `value: hello`,
			to: `value:
  nested: data`,
			check: hasTypes(diffyml.DiffModified),
		},
		{
			name: "ignore list order when configured",
			from: `list: [a, b, c]`,
			to:   `list: [c, b, a]`,
			opts: &diffyml.Options{IgnoreOrderChanges: true},
			check: noDiffs(),
		},
		{
			name: "map key added with correct path",
			from: `---
root:
  nested:
    existing: value
`,
			to: `---
root:
  nested:
    existing: value
    newkey: newvalue
`,
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff, got %d", len(diffs))
				}
				if diffs[0].Path != "root.nested.newkey" {
					t.Errorf("expected path 'root.nested.newkey', got '%s'", diffs[0].Path)
				}
				if diffs[0].Type != diffyml.DiffAdded {
					t.Errorf("expected DiffAdded, got %v", diffs[0].Type)
				}
				if diffs[0].To != "newvalue" {
					t.Errorf("expected To='newvalue', got %v", diffs[0].To)
				}
			},
		},
		{
			name: "map key removed with correct path",
			from: `---
root:
  nested:
    existing: value
    oldkey: oldvalue
`,
			to: `---
root:
  nested:
    existing: value
`,
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff, got %d", len(diffs))
				}
				if diffs[0].Path != "root.nested.oldkey" {
					t.Errorf("expected path 'root.nested.oldkey', got '%s'", diffs[0].Path)
				}
				if diffs[0].Type != diffyml.DiffRemoved {
					t.Errorf("expected DiffRemoved, got %v", diffs[0].Type)
				}
				if diffs[0].From != "oldvalue" {
					t.Errorf("expected From='oldvalue', got %v", diffs[0].From)
				}
			},
		},
		{
			name: "deeply nested map modifications",
			from: `---
level1:
  level2:
    level3:
      level4:
        value: original
`,
			to: `---
level1:
  level2:
    level3:
      level4:
        value: changed
`,
			check: singleDiff("level1.level2.level3.level4.value", diffyml.DiffModified),
		},
		{
			name: "multiple map changes at same level",
			from: `---
config:
  key1: value1
  key2: value2
  key3: value3
`,
			to: `---
config:
  key1: changed
  key3: value3
  key4: added
`,
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 3 {
					t.Fatalf("expected 3 diffs (modified, removed, added), got %d", len(diffs))
				}
				var hasModified, hasRemoved, hasAdded bool
				for _, d := range diffs {
					switch d.Path {
					case "config.key1":
						hasModified = d.Type == diffyml.DiffModified
					case "config.key2":
						hasRemoved = d.Type == diffyml.DiffRemoved
					case "config.key4":
						hasAdded = d.Type == diffyml.DiffAdded
					}
				}
				if !hasModified {
					t.Error("expected modified diff for config.key1")
				}
				if !hasRemoved {
					t.Error("expected removed diff for config.key2")
				}
				if !hasAdded {
					t.Error("expected added diff for config.key4")
				}
			},
		},
		{
			name:  "empty map to non-empty map",
			from:  `data: {}`,
			to:    `data: {key: value}`,
			check: singleDiff("data.key", diffyml.DiffAdded),
		},
		{
			name:  "non-empty map to empty map",
			from:  `data: {key: value}`,
			to:    `data: {}`,
			check: singleDiff("data.key", diffyml.DiffRemoved),
		},
		{
			name: "list entry added with correct path",
			from: `---
items:
  - first
  - second
`,
			to: `---
items:
  - first
  - second
  - third
`,
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff, got %d", len(diffs))
				}
				if diffs[0].Path != "items.2" {
					t.Errorf("expected path 'items.2', got '%s'", diffs[0].Path)
				}
				if diffs[0].Type != diffyml.DiffAdded {
					t.Errorf("expected DiffAdded, got %v", diffs[0].Type)
				}
				if diffs[0].To != "third" {
					t.Errorf("expected To='third', got %v", diffs[0].To)
				}
			},
		},
		{
			name: "list entry removed with correct path",
			from: `---
items:
  - first
  - second
  - third
`,
			to: `---
items:
  - first
  - second
`,
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff, got %d", len(diffs))
				}
				if diffs[0].Path != "items.2" {
					t.Errorf("expected path 'items.2', got '%s'", diffs[0].Path)
				}
				if diffs[0].Type != diffyml.DiffRemoved {
					t.Errorf("expected DiffRemoved, got %v", diffs[0].Type)
				}
				if diffs[0].From != "third" {
					t.Errorf("expected From='third', got %v", diffs[0].From)
				}
			},
		},
		{
			name: "list of maps with nested changes",
			from: `---
users:
  - name: alice
    age: 30
  - name: bob
    age: 25
`,
			to: `---
users:
  - name: alice
    age: 31
  - name: bob
    age: 25
`,
			check: singleDiff("users.alice.age", diffyml.DiffModified),
		},
		{
			name: "nested list within map",
			from: `---
config:
  servers:
    - host: server1
    - host: server2
`,
			to: `---
config:
  servers:
    - host: server1
    - host: server2
    - host: server3
`,
			check: singleDiff("config.servers.2", diffyml.DiffAdded),
		},
		{
			name: "list of maps - item added by identifier",
			from: `---
services:
  - name: web
    port: 80
  - name: api
    port: 8080
`,
			to: `---
services:
  - name: web
    port: 80
  - name: api
    port: 8080
  - name: db
    port: 5432
`,
			check: singleDiff("", diffyml.DiffAdded),
		},
		{
			name: "list of maps - item removed by identifier",
			from: `---
services:
  - name: web
    port: 80
  - name: api
    port: 8080
  - name: db
    port: 5432
`,
			to: `---
services:
  - name: web
    port: 80
  - name: api
    port: 8080
`,
			check: singleDiff("", diffyml.DiffRemoved),
		},
		{
			name: "deeply nested list modification",
			from: `---
root:
  level1:
    level2:
      items:
        - a
        - b
`,
			to: `---
root:
  level1:
    level2:
      items:
        - a
        - c
`,
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff, got %d", len(diffs))
				}
				if diffs[0].Path != "root.level1.level2.items.1" {
					t.Errorf("expected path 'root.level1.level2.items.1', got '%s'", diffs[0].Path)
				}
			},
		},
		{
			name: "empty list to non-empty list",
			from: `items: []`,
			to:   `items: [a, b, c]`,
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 3 {
					t.Fatalf("expected 3 diffs, got %d", len(diffs))
				}
				for i, d := range diffs {
					if d.Type != diffyml.DiffAdded {
						t.Errorf("diff %d: expected DiffAdded, got %v", i, d.Type)
					}
				}
			},
		},
		{
			name: "non-empty list to empty list",
			from: `items: [a, b, c]`,
			to:   `items: []`,
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 3 {
					t.Fatalf("expected 3 diffs, got %d", len(diffs))
				}
				for i, d := range diffs {
					if d.Type != diffyml.DiffRemoved {
						t.Errorf("diff %d: expected DiffRemoved, got %v", i, d.Type)
					}
				}
			},
		},
		{
			name:  "list with mixed types",
			from:  `data: [1, "two", true, {key: value}]`,
			to:    `data: [1, "two", false, {key: value}]`,
			check: singleDiff("data.2", diffyml.DiffModified),
		},
		// Comparison options tests
		{
			name: "swap reverses from and to",
			from: `value: original`,
			to:   `value: changed`,
			opts: &diffyml.Options{Swap: true},
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff, got %d", len(diffs))
				}
				// With swap, "from" becomes "to" and vice versa
				if diffs[0].From != "changed" {
					t.Errorf("expected From='changed' (swapped), got %v", diffs[0].From)
				}
				if diffs[0].To != "original" {
					t.Errorf("expected To='original' (swapped), got %v", diffs[0].To)
				}
			},
		},
		{
			name: "swap with additions becomes removals",
			from: `key: value`,
			to: `---
key: value
newkey: newvalue`,
			opts:  &diffyml.Options{Swap: true},
			check: singleDiff("", diffyml.DiffRemoved),
		},
		{
			name: "ignore value changes - only report structure",
			from: `---
config:
  name: old_name
  count: 10
  nested:
    value: old_value
`,
			to: `---
config:
  name: new_name
  count: 20
  nested:
    value: new_value
`,
			opts:  &diffyml.Options{IgnoreValueChanges: true},
			check: noDiffs(),
		},
		{
			name: "ignore value changes - still report additions",
			from: `key: value`,
			to: `---
key: value
newkey: added`,
			opts:  &diffyml.Options{IgnoreValueChanges: true},
			check: singleDiff("", diffyml.DiffAdded),
		},
		{
			name: "ignore value changes - still report removals",
			from: `---
key: value
oldkey: removed`,
			to:    `key: value`,
			opts:  &diffyml.Options{IgnoreValueChanges: true},
			check: singleDiff("", diffyml.DiffRemoved),
		},
		{
			name:  "ignore whitespace - leading spaces",
			from:  `message: "hello"`,
			to:    `message: "  hello"`,
			opts:  &diffyml.Options{IgnoreWhitespaceChanges: true},
			check: noDiffs(),
		},
		{
			name:  "ignore whitespace - trailing spaces",
			from:  `message: "hello"`,
			to:    `message: "hello  "`,
			opts:  &diffyml.Options{IgnoreWhitespaceChanges: true},
			check: noDiffs(),
		},
		{
			name:  "ignore whitespace - both leading and trailing",
			from:  `message: "hello"`,
			to:    `message: "  hello  "`,
			opts:  &diffyml.Options{IgnoreWhitespaceChanges: true},
			check: noDiffs(),
		},
		{
			name:  "ignore whitespace - content differs still detected",
			from:  `message: "hello"`,
			to:    `message: "  world  "`,
			opts:  &diffyml.Options{IgnoreWhitespaceChanges: true},
			check: singleDiff("", diffyml.DiffModified),
		},
		{
			name:  "ignore list order - same elements different order",
			from:  `items: [alpha, beta, gamma]`,
			to:    `items: [gamma, alpha, beta]`,
			opts:  &diffyml.Options{IgnoreOrderChanges: true},
			check: noDiffs(),
		},
		{
			name:  "ignore list order - detect actual additions",
			from:  `items: [a, b]`,
			to:    `items: [b, a, c]`,
			opts:  &diffyml.Options{IgnoreOrderChanges: true},
			check: singleDiff("", diffyml.DiffAdded),
		},
		{
			name:  "ignore list order - detect actual removals",
			from:  `items: [a, b, c]`,
			to:    `items: [c, a]`,
			opts:  &diffyml.Options{IgnoreOrderChanges: true},
			check: singleDiff("", diffyml.DiffRemoved),
		},
		{
			name: "ignore list order with maps",
			from: `---
items:
  - name: first
    value: 1
  - name: second
    value: 2
`,
			to: `---
items:
  - name: second
    value: 2
  - name: first
    value: 1
`,
			opts:  &diffyml.Options{IgnoreOrderChanges: true},
			check: noDiffs(),
		},
		{
			name: "combined options - swap and ignore whitespace",
			from: `value: "hello"`,
			to:   `value: "  world  "`,
			opts: &diffyml.Options{Swap: true, IgnoreWhitespaceChanges: true},
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff, got %d", len(diffs))
				}
				// With swap, from and to are reversed
				if diffs[0].From != "  world  " {
					t.Errorf("expected From='  world  ' (swapped), got %v", diffs[0].From)
				}
				if diffs[0].To != "hello" {
					t.Errorf("expected To='hello' (swapped), got %v", diffs[0].To)
				}
			},
		},

		// --- Additional identifier tests ---
		{
			name: "additional identifier reorder - order changed",
			from: `items:
  - key: a
    value: 1
  - key: b
    value: 2
`,
			to: `items:
  - key: b
    value: 2
  - key: a
    value: 1
`,
			opts:  &diffyml.Options{AdditionalIdentifiers: []string{"key"}},
			check: singleDiff("", diffyml.DiffOrderChanged),
		},
		{
			name: "additional identifier reorder - ignored when configured",
			from: `items:
  - key: a
    value: 1
  - key: b
    value: 2
`,
			to: `items:
  - key: b
    value: 2
  - key: a
    value: 1
`,
			opts: &diffyml.Options{
				AdditionalIdentifiers: []string{"key"},
				IgnoreOrderChanges:    true,
			},
			check: noDiffs(),
		},
		{
			name: "non-comparable identifier - no panic and diffs",
			from: `items:
  - name: [x]
    value: 1
`,
			to: `items:
  - name: [x]
    value: 2
`,
			check: singleDiff("items.0.value", diffyml.DiffModified),
		},
		{
			name: "mixed identifier and non-identifier - reports removal",
			from: `items:
  - name: a
    value: 1
  - other: x
`,
			to: `items:
  - name: a
    value: 1
`,
			check: singleDiff("items.1", diffyml.DiffRemoved),
		},
		{
			name: "additional identifier modify",
			from: `items:
  - key: alpha
    value: 1`,
			to: `items:
  - key: alpha
    value: 2`,
			opts: &diffyml.Options{AdditionalIdentifiers: []string{"key"}},
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff, got %d", len(diffs))
				}
				if diffs[0].Type != diffyml.DiffModified {
					t.Errorf("expected DiffModified, got %v", diffs[0].Type)
				}
				if !strings.Contains(diffs[0].Path, "alpha") {
					t.Errorf("expected path to contain identifier 'alpha', got %q", diffs[0].Path)
				}
			},
		},

		// --- Mutation testing: comparator.go ---
		{
			name: "multi-doc from 1 to 2",
			from: `name: single`,
			to: `name: first
---
name: second`,
			check: hasTypes(diffyml.DiffAdded),
		},
		{
			name: "multi-doc from 2 to 1",
			from: `name: first
---
name: second`,
			to: `name: single`,
			check: func(t *testing.T, diffs []diffyml.Difference) {
				hasSecondDocDiff := false
				for _, d := range diffs {
					if strings.HasPrefix(d.Path, "[1]") {
						hasSecondDocDiff = true
					}
				}
				if !hasSecondDocDiff {
					t.Error("expected diff for second document [1] in from")
				}
			},
		},
		{
			name:  "ignore value changes - value to null",
			from:  `key: val`,
			to:    `key: ~`,
			opts:  &diffyml.Options{IgnoreValueChanges: true},
			check: noDiffs(),
		},
		{
			name:  "ignore value changes - type mismatch",
			from:  `port: 80`,
			to:    `port: "80"`,
			opts:  &diffyml.Options{IgnoreValueChanges: true},
			check: noDiffs(),
		},
		{
			name: "heterogeneous single-key list",
			from: `rules:
  - port: 80
  - port: 443`,
			to: `rules:
  - port: 80
  - port: 8443`,
			check: singleDiff("", diffyml.DiffModified),
		},
		{
			name: "unordered list with null",
			from: `items: [~, a]`,
			to:   `items: [a]`,
			opts: &diffyml.Options{IgnoreOrderChanges: true},
			check: func(t *testing.T, diffs []diffyml.Difference) {
				hasRemoved := false
				for _, d := range diffs {
					if d.Type == diffyml.DiffRemoved && d.From == nil {
						hasRemoved = true
					}
				}
				if !hasRemoved {
					t.Error("expected DiffRemoved for null item")
				}
			},
		},

		// --- Mutation testing: diffyml.go sort order ---
		{
			name: "diff order matches document order",
			from: `z: 1
a: 2
m: 3`,
			to: `z: 10
a: 20
m: 30`,
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 3 {
					t.Fatalf("expected 3 diffs, got %d", len(diffs))
				}
				// Diffs should follow source order: z, a, m — NOT alphabetical a, m, z
				if diffs[0].Path != "z" {
					t.Errorf("expected first diff path 'z', got %q", diffs[0].Path)
				}
				if diffs[1].Path != "a" {
					t.Errorf("expected second diff path 'a', got %q", diffs[1].Path)
				}
				if diffs[2].Path != "m" {
					t.Errorf("expected third diff path 'm', got %q", diffs[2].Path)
				}
			},
		},
		{
			name: "list entry at index 9",
			from: `items: [a,b,c,d,e,f,g,h,i,j]`,
			to:   `items: [a,b,c,d,e,f,g,h,i,CHANGED]`,
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff, got %d", len(diffs))
				}
				if diffs[0].Path != "items.9" {
					t.Errorf("expected path 'items.9', got %q", diffs[0].Path)
				}
			},
		},
		{
			name: "root additions vs nested diffs sort",
			from: `existing:
  nested: value`,
			to: `existing:
  nested: changed
newroot: added`,
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) < 2 {
					t.Fatalf("expected at least 2 diffs, got %d", len(diffs))
				}
				// Root-level addition should come first
				if diffs[0].Type != diffyml.DiffAdded || diffs[0].Path != "newroot" {
					t.Errorf("expected first diff to be root-level addition 'newroot', got type=%v path=%q",
						diffs[0].Type, diffs[0].Path)
				}
			},
		},
		{
			name: "sort fallback behaviors",
			from: `root:
  shallow: 1
  deep:
    nested: 2`,
			to: `root:
  shallow: 10
  deep:
    nested: 20`,
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 2 {
					t.Fatalf("expected 2 diffs, got %d", len(diffs))
				}
				// shallow (depth 1) should come before deep.nested (depth 2) in source order
				if diffs[0].Path != "root.shallow" {
					t.Errorf("expected first diff 'root.shallow', got %q", diffs[0].Path)
				}
				if diffs[1].Path != "root.deep.nested" {
					t.Errorf("expected second diff 'root.deep.nested', got %q", diffs[1].Path)
				}
			},
		},
		{
			name: "root add before root modify",
			from: `z_modified: old`,
			to: `a_added: new
z_modified: changed`,
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 2 {
					t.Fatalf("expected 2 diffs, got %d", len(diffs))
				}
				// Root-level addition must come before root-level modification
				if diffs[0].Type != diffyml.DiffAdded || diffs[0].Path != "a_added" {
					t.Errorf("expected first diff to be added 'a_added', got type=%v path=%q",
						diffs[0].Type, diffs[0].Path)
				}
				if diffs[1].Type != diffyml.DiffModified || diffs[1].Path != "z_modified" {
					t.Errorf("expected second diff to be modified 'z_modified', got type=%v path=%q",
						diffs[1].Type, diffs[1].Path)
				}
			},
		},
		{
			name: "heterogeneous list reorder",
			from: `rules:
  - namespaceSelector: ns1
  - ipBlock: 10.0.0.0/8`,
			to: `rules:
  - ipBlock: 10.0.0.0/8
  - namespaceSelector: ns1`,
			check: noDiffs(),
		},
		{
			name: "parent order for added children",
			from: `root:
  zzz:
    child: old
  aaa:
    child: old`,
			to: `root:
  zzz:
    child: old
    newkey: added_z
  aaa:
    child: old
    newkey: added_a`,
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 2 {
					t.Fatalf("expected 2 diffs, got %d", len(diffs))
				}
				// zzz appears before aaa in the document, so zzz.newkey must come first
				if diffs[0].Path != "root.zzz.newkey" {
					t.Errorf("expected first diff 'root.zzz.newkey', got %q", diffs[0].Path)
				}
				if diffs[1].Path != "root.aaa.newkey" {
					t.Errorf("expected second diff 'root.aaa.newkey', got %q", diffs[1].Path)
				}
			},
		},
		{
			name:  "unordered list null vs value",
			from:  `items: [hello]`,
			to:    `items: [null]`,
			opts:  &diffyml.Options{IgnoreOrderChanges: true},
			check: hasTypes(),
		},

		// --- Tests adapted from dyff edge cases ---
		{
			name: "boolean normalization true",
			from: `---
key: true`,
			to: `---
key: True`,
			check: noDiffs(),
		},
		{
			name: "boolean normalization false",
			from: `---
key: false`,
			to: `---
key: False`,
			check: noDiffs(),
		},
		{
			name: "YAML anchors and aliases",
			from: `---
global_defaults: &global_defaults
  - x1
  - x5
cluster-1:
  - *global_defaults`,
			to: `---
global_defaults: &global_defaults
  - x1
  - x5
  - x10
cluster-1:
  - *global_defaults
  - x999`,
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) == 0 {
					t.Fatal("expected diffs for YAML anchor changes, got 0")
				}
				// Should detect the added x10 in global_defaults and x999 in cluster-1
				hasGlobalAdd := false
				hasClusterAdd := false
				for _, d := range diffs {
					if d.Type == diffyml.DiffAdded {
						if strings.HasPrefix(d.Path, "global_defaults") {
							hasGlobalAdd = true
						}
						if strings.HasPrefix(d.Path, "cluster-1") {
							hasClusterAdd = true
						}
					}
				}
				if !hasGlobalAdd {
					t.Error("expected addition in global_defaults")
				}
				if !hasClusterAdd {
					t.Error("expected addition in cluster-1")
				}
			},
		},
		{
			name: "type change map to list",
			from: `---
foo:
  a: 1
  b: 2`,
			to: `---
foo:
  - 1
  - 2`,
			check: hasTypes(diffyml.DiffModified),
		},
		{
			name: "type change map to null",
			from: `---
bar:
  c: 3
  d: 4`,
			to: `---
bar:`,
			check: hasTypes(),
		},
		{
			name: "empty documents ignored",
			from: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: x
---
apiVersion: v1
kind: Service
metadata:
  name: y`,
			to: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: x
---
---
apiVersion: v1
kind: Service
metadata:
  name: y
---`,
		},
		{
			name: "K8s document added",
			from: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: x`,
			to: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: x
---
apiVersion: v1
kind: Service
metadata:
  name: y`,
			check: hasTypes(diffyml.DiffAdded),
		},
		{
			name: "K8s document removed",
			from: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: x
---
apiVersion: v1
kind: Service
metadata:
  name: y`,
			to: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: x`,
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) == 0 {
					t.Fatal("expected diffs for removed K8s document, got 0")
				}
				// The removed document should appear in the diff path as [1]
				found := false
				for _, d := range diffs {
					if strings.HasPrefix(d.Path, "[1]") {
						found = true
					}
				}
				if !found {
					t.Error("expected diff for second document [1]")
				}
			},
		},
		{
			name: "template variables preserved",
			from: `---
example_one: "%{one}"
example_two: "two"`,
			to: `---
example_one: "one"
example_two: "%{two}"`,
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 2 {
					t.Fatalf("expected 2 diffs (both values changed), got %d", len(diffs))
				}
				if !hasModification(diffs, "%{one}", "one") {
					t.Error("expected modification from '%{one}' to 'one'")
				}
				if !hasModification(diffs, "two", "%{two}") {
					t.Error("expected modification from 'two' to '%{two}'")
				}
			},
		},
		{
			name: "duplicate list entries",
			from: `keys:
  - value1
  - value2`,
			to: `keys:
  - value1
  - value2
  - value1`,
			check: singleDiff("", diffyml.DiffAdded),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from := yml(tt.from)
			to := yml(tt.to)

			diffs, err := compare(from, to, tt.opts)
			if (err != nil) != tt.wantErr {
				t.Fatalf("compare() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.check != nil {
				tt.check(t, diffs)
			}
		})
	}
}

func TestCompare_DeterministicResults(t *testing.T) {
	// dyff issue-525: comparison must produce deterministic results regardless
	// of Go map iteration order.
	from := yml(`name: a-type-of-file
allowed:
  - digest: sha256:1111111111111111111111111111111111111111111111111111111111111111
    image: name/container
    registry: ghcr.io
    tag: 1.2.3
    field:
      - test
  - digest: sha256:22222222222222222222222222222222222222222222222222222222222222222
    image: yes/i-am-an-image
    registry: docker.io
    tag: 1.2.3-test_with.symbols
  - digest: sha256:33333333333333333333333333333333333333333333333333333333333333333
    image: another/image
    registry: gcr.io
    tag: 3.2.1
  - digest: sha256:4444444444444444444444444444444444444444444444444444444444444444
    image: oh-look/another-image
    registry: quay.io
    tag: 3.1.2-test-with-dashes
  - digest: sha256:5555555555555555555555555555555555555555555555555555555555555555
    image: you-would-not/guess
    registry: docker.io
    tag: 1.3.2
  - digest: sha256:6666666666666666666666666666666666666666666666666666666666666666
    image: no-way/this-is-an-image
    registry: guess.io
    tag: latest`)
	to := yml(`name: a-type-of-file
allowed:
  - digest: sha256:1111111111111111111111111111111111111111111111111111111111111111
    image: name/container
    registry: ghcr.io
    tag: 1.2.4
    field:
      - test
  - digest: sha256:22222222222222222222222222222222222222222222222222222222222222222
    image: yes/i-am-an-image
    registry: docker.io
    tag: 1.2.4-test_with.symbols
  - digest: sha256:33333333333333333333333333333333333333333333333333333333333333333
    image: another/image
    registry: gcr.io
    tag: 3.2.1
  - digest: sha256:4444444444444444444444444444444444444444444444444444444444444444
    image: oh-look/another-flaky
    registry: quay.io
    tag: 3.1.2-test-with-dashes
  - digest: sha256:0000000000000000000000000000000000000000000000000000000000000000
    image: you-would-not/guess
    registry: docker.io
    tag: 1.3.2
  - digest: sha256:6666666666666666666666666666666666666666666666666666666666666666
    image: no-way/this-is-an-image
    registry: guess.io
    tag: latest
  - digest: sha256:6666666666666666666666666666666666666666666666666666666666666666
    image: additional/image
    registry: new.io
    tag: 9.8.7`)

	// Run 100 times to catch non-deterministic map iteration order.
	var expectedCount int
	for i := range 100 {
		diffs, err := compare(from, to, nil)
		if err != nil {
			t.Fatalf("compare() failed on iteration %d: %v", i, err)
		}
		if i == 0 {
			expectedCount = len(diffs)
			if expectedCount == 0 {
				t.Fatal("expected at least 1 diff")
			}
		} else if len(diffs) != expectedCount {
			t.Fatalf("non-deterministic: iteration %d got %d diffs, expected %d",
				i, len(diffs), expectedCount)
		}
	}
}

func TestCompare_DateStringEdgeCases(t *testing.T) {
	// dyff issue-217: various date-like strings must not be modified or misinterpreted.
	from := yml(`---
Datestring: 2033-12-20
ThirteenthMonth: 2033-13-20
FortyDays: 2033-13-40
TheYear9999: 9999-11-20
OneShortDatestring: 999-99-99
ExtDatestring: 2021-01-01-04-05
DatestringFake: 9999-99-99
DatestringNonHyphenated: 99999999
DatestringOneHyphen: 9999-9999
DatestringSlashes: 2022/01/01`)
	to := yml(`---
Datestring: 2033-12-20
ThirteenthMonth: 2033-13-20
FortyDays: 2033-13-40
TheYear9999: 9999-11-20
OneShortDatestring: 999-99-99
ExtDatestring: 2021-01-01-04-05
DatestringFake: 9999-99-99
DatestringNonHyphenated: 99999999
DatestringOneHyphen: 9999-9999
DatestringSlashes: 2022/01/01`)

	diffs, err := compare(from, to, nil)
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	if len(diffs) != 0 {
		t.Errorf("expected 0 diffs for identical date strings, got %d", len(diffs))
		for _, d := range diffs {
			t.Logf("  diff: path=%q type=%v from=%v to=%v", d.Path, d.Type, d.From, d.To)
		}
	}
}
