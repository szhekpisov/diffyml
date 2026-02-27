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
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) < 1 {
					t.Fatal("expected at least 1 diff")
				}
				if !hasModification(diffs, "foobar", "fOObAr") {
					t.Error("expected modification from 'foobar' to 'fOObAr'")
				}
			},
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
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) < 1 {
					t.Fatal("expected at least 1 diff")
				}
				if !hasDiffType(diffs, diffyml.DiffModified) {
					t.Error("expected a modification diff")
				}
			},
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
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) < 1 {
					t.Fatal("expected at least 1 diff")
				}
			},
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
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) < 1 {
					t.Fatal("expected at least 1 diff")
				}
			},
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
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) < 1 {
					t.Fatal("expected at least 1 diff")
				}
				if !hasDiffType(diffs, diffyml.DiffAdded) {
					t.Error("expected an addition diff")
				}
			},
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
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) < 1 {
					t.Fatal("expected at least 1 diff")
				}
				if !hasDiffType(diffs, diffyml.DiffRemoved) {
					t.Error("expected a removal diff")
				}
			},
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
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 0 {
					t.Errorf("expected 0 diffs, got %d", len(diffs))
				}
			},
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
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) < 1 {
					t.Fatal("expected at least 1 diff")
				}
			},
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
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) < 1 {
					t.Fatal("expected at least 1 diff")
				}
			},
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
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) < 1 {
					t.Fatal("expected at least 1 diff")
				}
			},
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
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) < 1 {
					t.Fatal("expected at least 1 diff")
				}
			},
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
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 0 {
					t.Errorf("expected 0 diffs for hash order change, got %d", len(diffs))
				}
			},
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
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) < 1 {
					t.Fatal("expected at least 1 diff")
				}
				if !hasModification(diffs, "192.168.1.1", "192.168.0.1") {
					t.Error("expected IP address modification")
				}
			},
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
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) < 1 {
					t.Fatal("expected at least 1 diff")
				}
			},
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
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 0 {
					t.Errorf("expected 0 diffs for identical YAMLs, got %d", len(diffs))
				}
			},
		},
		{
			name: "null to value change",
			from: `foo: null`,
			to:   `foo: "bar"`,
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) < 1 {
					t.Fatal("expected at least 1 diff")
				}
			},
		},
		{
			name: "value to null change",
			from: `foo: "bar"`,
			to:   `foo: null`,
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) < 1 {
					t.Fatal("expected at least 1 diff")
				}
				if !hasDiffType(diffs, diffyml.DiffModified) {
					t.Error("expected value to null to be reported as modification")
				}
			},
		},
		{
			name: "type change string to map",
			from: `value: hello`,
			to: `value:
  nested: data`,
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) < 1 {
					t.Fatal("expected at least 1 diff")
				}
				if !hasDiffType(diffs, diffyml.DiffModified) {
					t.Error("expected type change to be reported as modification")
				}
			},
		},
		{
			name: "ignore list order when configured",
			from: `list: [a, b, c]`,
			to:   `list: [c, b, a]`,
			opts: &diffyml.Options{IgnoreOrderChanges: true},
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 0 {
					t.Errorf("expected 0 diffs when ignoring order, got %d", len(diffs))
				}
			},
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
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff, got %d", len(diffs))
				}
				if diffs[0].Path != "level1.level2.level3.level4.value" {
					t.Errorf("expected path 'level1.level2.level3.level4.value', got '%s'", diffs[0].Path)
				}
				if diffs[0].Type != diffyml.DiffModified {
					t.Errorf("expected DiffModified, got %v", diffs[0].Type)
				}
			},
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
			name: "empty map to non-empty map",
			from: `data: {}`,
			to:   `data: {key: value}`,
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff, got %d", len(diffs))
				}
				if diffs[0].Type != diffyml.DiffAdded {
					t.Errorf("expected DiffAdded, got %v", diffs[0].Type)
				}
				if diffs[0].Path != "data.key" {
					t.Errorf("expected path 'data.key', got '%s'", diffs[0].Path)
				}
			},
		},
		{
			name: "non-empty map to empty map",
			from: `data: {key: value}`,
			to:   `data: {}`,
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff, got %d", len(diffs))
				}
				if diffs[0].Type != diffyml.DiffRemoved {
					t.Errorf("expected DiffRemoved, got %v", diffs[0].Type)
				}
				if diffs[0].Path != "data.key" {
					t.Errorf("expected path 'data.key', got '%s'", diffs[0].Path)
				}
			},
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
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff, got %d", len(diffs))
				}
				// Path uses identifier (name: alice) instead of index
				if diffs[0].Path != "users.alice.age" {
					t.Errorf("expected path 'users.alice.age', got '%s'", diffs[0].Path)
				}
				if diffs[0].Type != diffyml.DiffModified {
					t.Errorf("expected DiffModified, got %v", diffs[0].Type)
				}
			},
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
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff, got %d", len(diffs))
				}
				if diffs[0].Path != "config.servers.2" {
					t.Errorf("expected path 'config.servers.2', got '%s'", diffs[0].Path)
				}
				if diffs[0].Type != diffyml.DiffAdded {
					t.Errorf("expected DiffAdded, got %v", diffs[0].Type)
				}
			},
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
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff, got %d", len(diffs))
				}
				if diffs[0].Type != diffyml.DiffAdded {
					t.Errorf("expected DiffAdded, got %v", diffs[0].Type)
				}
			},
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
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff, got %d", len(diffs))
				}
				if diffs[0].Type != diffyml.DiffRemoved {
					t.Errorf("expected DiffRemoved, got %v", diffs[0].Type)
				}
			},
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
			name: "list with mixed types",
			from: `data: [1, "two", true, {key: value}]`,
			to:   `data: [1, "two", false, {key: value}]`,
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff, got %d", len(diffs))
				}
				if diffs[0].Path != "data.2" {
					t.Errorf("expected path 'data.2', got '%s'", diffs[0].Path)
				}
				if diffs[0].Type != diffyml.DiffModified {
					t.Errorf("expected DiffModified, got %v", diffs[0].Type)
				}
			},
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
			opts: &diffyml.Options{Swap: true},
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff, got %d", len(diffs))
				}
				// Addition in to becomes removal when swapped
				if diffs[0].Type != diffyml.DiffRemoved {
					t.Errorf("expected DiffRemoved (swapped addition), got %v", diffs[0].Type)
				}
			},
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
			opts: &diffyml.Options{IgnoreValueChanges: true},
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 0 {
					t.Errorf("expected 0 diffs when ignoring value changes, got %d", len(diffs))
				}
			},
		},
		{
			name: "ignore value changes - still report additions",
			from: `key: value`,
			to: `---
key: value
newkey: added`,
			opts: &diffyml.Options{IgnoreValueChanges: true},
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff (addition), got %d", len(diffs))
				}
				if diffs[0].Type != diffyml.DiffAdded {
					t.Errorf("expected DiffAdded, got %v", diffs[0].Type)
				}
			},
		},
		{
			name: "ignore value changes - still report removals",
			from: `---
key: value
oldkey: removed`,
			to:   `key: value`,
			opts: &diffyml.Options{IgnoreValueChanges: true},
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff (removal), got %d", len(diffs))
				}
				if diffs[0].Type != diffyml.DiffRemoved {
					t.Errorf("expected DiffRemoved, got %v", diffs[0].Type)
				}
			},
		},
		{
			name: "ignore whitespace - leading spaces",
			from: `message: "hello"`,
			to:   `message: "  hello"`,
			opts: &diffyml.Options{IgnoreWhitespaceChanges: true},
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 0 {
					t.Errorf("expected 0 diffs when ignoring whitespace, got %d", len(diffs))
				}
			},
		},
		{
			name: "ignore whitespace - trailing spaces",
			from: `message: "hello"`,
			to:   `message: "hello  "`,
			opts: &diffyml.Options{IgnoreWhitespaceChanges: true},
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 0 {
					t.Errorf("expected 0 diffs when ignoring whitespace, got %d", len(diffs))
				}
			},
		},
		{
			name: "ignore whitespace - both leading and trailing",
			from: `message: "hello"`,
			to:   `message: "  hello  "`,
			opts: &diffyml.Options{IgnoreWhitespaceChanges: true},
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 0 {
					t.Errorf("expected 0 diffs when ignoring whitespace, got %d", len(diffs))
				}
			},
		},
		{
			name: "ignore whitespace - content differs still detected",
			from: `message: "hello"`,
			to:   `message: "  world  "`,
			opts: &diffyml.Options{IgnoreWhitespaceChanges: true},
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff (content differs), got %d", len(diffs))
				}
				if diffs[0].Type != diffyml.DiffModified {
					t.Errorf("expected DiffModified, got %v", diffs[0].Type)
				}
			},
		},
		{
			name: "ignore list order - same elements different order",
			from: `items: [alpha, beta, gamma]`,
			to:   `items: [gamma, alpha, beta]`,
			opts: &diffyml.Options{IgnoreOrderChanges: true},
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 0 {
					t.Errorf("expected 0 diffs when ignoring order, got %d", len(diffs))
				}
			},
		},
		{
			name: "ignore list order - detect actual additions",
			from: `items: [a, b]`,
			to:   `items: [b, a, c]`,
			opts: &diffyml.Options{IgnoreOrderChanges: true},
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff (addition), got %d", len(diffs))
				}
				if diffs[0].Type != diffyml.DiffAdded {
					t.Errorf("expected DiffAdded, got %v", diffs[0].Type)
				}
			},
		},
		{
			name: "ignore list order - detect actual removals",
			from: `items: [a, b, c]`,
			to:   `items: [c, a]`,
			opts: &diffyml.Options{IgnoreOrderChanges: true},
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff (removal), got %d", len(diffs))
				}
				if diffs[0].Type != diffyml.DiffRemoved {
					t.Errorf("expected DiffRemoved, got %v", diffs[0].Type)
				}
			},
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
			opts: &diffyml.Options{IgnoreOrderChanges: true},
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 0 {
					t.Errorf("expected 0 diffs when ignoring order with maps, got %d", len(diffs))
				}
			},
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

func TestCompare_AdditionalIdentifierReorder_NoDiff(t *testing.T) {
	from := yml(`items:
  - key: a
    value: 1
  - key: b
    value: 2
`)
	to := yml(`items:
  - key: b
    value: 2
  - key: a
    value: 1
`)

	diffs, err := compare(from, to, &diffyml.Options{
		AdditionalIdentifiers: []string{"key"},
	})
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	if len(diffs) != 0 {
		t.Fatalf("expected no diffs with additional identifier, got %d", len(diffs))
	}
}

func TestCompare_NonComparableIdentifier_NoPanicAndDiffs(t *testing.T) {
	from := yml(`items:
  - name: [x]
    value: 1
`)
	to := yml(`items:
  - name: [x]
    value: 2
`)

	diffs, err := compare(from, to, nil)
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].Path != "items.0.value" {
		t.Fatalf("expected diff path items.0.value, got %q", diffs[0].Path)
	}
	if diffs[0].Type != diffyml.DiffModified {
		t.Fatalf("expected DiffModified, got %v", diffs[0].Type)
	}
}

func TestCompare_MixedIdentifierAndNonIdentifier_ReportsRemoval(t *testing.T) {
	from := yml(`items:
  - name: a
    value: 1
  - other: x
`)
	to := yml(`items:
  - name: a
    value: 1
`)

	diffs, err := compare(from, to, nil)
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].Type != diffyml.DiffRemoved {
		t.Fatalf("expected DiffRemoved, got %v", diffs[0].Type)
	}
	if diffs[0].Path != "items.1" {
		t.Fatalf("expected diff path items.1, got %q", diffs[0].Path)
	}
}

// --- Mutation testing: comparator.go ---

func TestCompare_MultiDoc_From1To2(t *testing.T) {
	from := yml(`name: single`)
	to := yml(`name: first
---
name: second`)

	diffs, err := compare(from, to, nil)
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	// Should have at least 1 diff (the added second document)
	hasAdded := false
	for _, d := range diffs {
		if d.Type == diffyml.DiffAdded {
			hasAdded = true
		}
	}
	if !hasAdded {
		t.Error("expected DiffAdded for second document in to")
	}
}

func TestCompare_MultiDoc_From2To1(t *testing.T) {
	from := yml(`name: first
---
name: second`)
	to := yml(`name: single`)

	diffs, err := compare(from, to, nil)
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	// Second document in from has no counterpart in to → DiffModified (from=doc, to=nil)
	hasSecondDocDiff := false
	for _, d := range diffs {
		if strings.HasPrefix(d.Path, "[1]") {
			hasSecondDocDiff = true
		}
	}
	if !hasSecondDocDiff {
		t.Error("expected diff for second document [1] in from")
	}
}

func TestCompare_IgnoreValueChanges_ValueToNull(t *testing.T) {
	from := yml(`key: val`)
	to := yml(`key: ~`)

	diffs, err := compare(from, to, &diffyml.Options{IgnoreValueChanges: true})
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	if len(diffs) != 0 {
		t.Errorf("expected 0 diffs with IgnoreValueChanges when value → null, got %d", len(diffs))
	}
}

func TestCompare_IgnoreValueChanges_TypeMismatch(t *testing.T) {
	from := yml(`port: 80`)
	to := yml(`port: "80"`)

	diffs, err := compare(from, to, &diffyml.Options{IgnoreValueChanges: true})
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	if len(diffs) != 0 {
		t.Errorf("expected 0 diffs with IgnoreValueChanges on type mismatch, got %d", len(diffs))
	}
}

func TestCompare_HeterogeneousSingleKeyList(t *testing.T) {
	// Items with same single-key structure should use positional compare (homogeneous)
	from := yml(`rules:
  - port: 80
  - port: 443`)
	to := yml(`rules:
  - port: 80
  - port: 8443`)

	diffs, err := compare(from, to, nil)
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff for positional compare, got %d", len(diffs))
	}
	if diffs[0].Type != diffyml.DiffModified {
		t.Errorf("expected DiffModified, got %v", diffs[0].Type)
	}
}

func TestCompare_UnorderedListWithNull(t *testing.T) {
	from := yml(`items: [~, a]`)
	to := yml(`items: [a]`)

	diffs, err := compare(from, to, &diffyml.Options{IgnoreOrderChanges: true})
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	// null was removed
	hasRemoved := false
	for _, d := range diffs {
		if d.Type == diffyml.DiffRemoved && d.From == nil {
			hasRemoved = true
		}
	}
	if !hasRemoved {
		t.Error("expected DiffRemoved for null item")
	}
}

// --- Mutation testing: diffyml.go sort order ---

func TestCompare_DiffOrderMatchesDocumentOrder(t *testing.T) {
	from := yml(`z: 1
a: 2
m: 3`)
	to := yml(`z: 10
a: 20
m: 30`)

	diffs, err := compare(from, to, nil)
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
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
}

func TestCompare_ListEntryAtIndex9(t *testing.T) {
	from := yml(`items: [a,b,c,d,e,f,g,h,i,j]`)
	to := yml(`items: [a,b,c,d,e,f,g,h,i,CHANGED]`)

	diffs, err := compare(from, to, nil)
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].Path != "items.9" {
		t.Errorf("expected path 'items.9', got %q", diffs[0].Path)
	}
}

func TestCompare_RootAdditionsVsNestedDiffsSort(t *testing.T) {
	from := yml(`existing:
  nested: value`)
	to := yml(`existing:
  nested: changed
newroot: added`)

	diffs, err := compare(from, to, nil)
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	if len(diffs) < 2 {
		t.Fatalf("expected at least 2 diffs, got %d", len(diffs))
	}
	// Root-level addition should come first
	if diffs[0].Type != diffyml.DiffAdded || diffs[0].Path != "newroot" {
		t.Errorf("expected first diff to be root-level addition 'newroot', got type=%v path=%q",
			diffs[0].Type, diffs[0].Path)
	}
}

func TestCompare_SortFallbackBehaviors(t *testing.T) {
	// Test depth sort and alphabetical tiebreak within same root component
	from := yml(`root:
  shallow: 1
  deep:
    nested: 2`)
	to := yml(`root:
  shallow: 10
  deep:
    nested: 20`)

	diffs, err := compare(from, to, nil)
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
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
}

func TestCompare_RootAddBeforeRootModify(t *testing.T) {
	// A root-level addition must be sorted before a root-level modification,
	// even when document order would place the modification first.
	from := yml(`z_modified: old`)
	to := yml(`a_added: new
z_modified: changed`)

	diffs, err := compare(from, to, nil)
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
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
}

func TestCompare_HeterogeneousListReorder(t *testing.T) {
	// Heterogeneous list items (single distinct keys) should be compared unordered,
	// so reordering them should produce no diffs.
	from := yml(`rules:
  - namespaceSelector: ns1
  - ipBlock: 10.0.0.0/8`)
	to := yml(`rules:
  - ipBlock: 10.0.0.0/8
  - namespaceSelector: ns1`)

	diffs, err := compare(from, to, nil)
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	if len(diffs) != 0 {
		t.Errorf("expected 0 diffs for reordered heterogeneous list, got %d", len(diffs))
		for _, d := range diffs {
			t.Logf("  diff: type=%v path=%q", d.Type, d.Path)
		}
	}
}
