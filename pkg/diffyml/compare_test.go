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

func TestCompare_ScalarModifications(t *testing.T) {
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

func TestCompare_ValueAddRemove(t *testing.T) {
	tests := []struct {
		name    string
		from    string
		to      string
		opts    *diffyml.Options
		wantErr bool
		check   func(t *testing.T, diffs []diffyml.Difference)
	}{
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

func TestCompare_NullAndTypeChanges(t *testing.T) {
	tests := []struct {
		name    string
		from    string
		to      string
		opts    *diffyml.Options
		wantErr bool
		check   func(t *testing.T, diffs []diffyml.Difference)
	}{
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

func TestCompare_MapPaths(t *testing.T) {
	tests := []struct {
		name    string
		from    string
		to      string
		opts    *diffyml.Options
		wantErr bool
		check   func(t *testing.T, diffs []diffyml.Difference)
	}{
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
				if diffs[0].Path.String() != "root.nested" {
					t.Errorf("expected path 'root.nested', got '%s'", diffs[0].Path)
				}
				if diffs[0].Type != diffyml.DiffAdded {
					t.Errorf("expected DiffAdded, got %v", diffs[0].Type)
				}
				om, ok := diffs[0].To.(*diffyml.OrderedMap)
				if !ok {
					t.Fatalf("expected To to be *OrderedMap, got %T", diffs[0].To)
				}
				if len(om.Keys) != 1 || om.Keys[0] != "newkey" {
					t.Errorf("expected OrderedMap with key 'newkey', got keys %v", om.Keys)
				}
				if om.Values["newkey"] != "newvalue" {
					t.Errorf("expected OrderedMap value 'newvalue', got %v", om.Values["newkey"])
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
				if diffs[0].Path.String() != "root.nested" {
					t.Errorf("expected path 'root.nested', got '%s'", diffs[0].Path)
				}
				if diffs[0].Type != diffyml.DiffRemoved {
					t.Errorf("expected DiffRemoved, got %v", diffs[0].Type)
				}
				om, ok := diffs[0].From.(*diffyml.OrderedMap)
				if !ok {
					t.Fatalf("expected From to be *OrderedMap, got %T", diffs[0].From)
				}
				if len(om.Keys) != 1 || om.Keys[0] != "oldkey" {
					t.Errorf("expected OrderedMap with key 'oldkey', got keys %v", om.Keys)
				}
				if om.Values["oldkey"] != "oldvalue" {
					t.Errorf("expected OrderedMap value 'oldvalue', got %v", om.Values["oldkey"])
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
				if diffs[0].Path.String() != "level1.level2.level3.level4.value" {
					t.Errorf("expected path 'level1.level2.level3.level4.value', got '%s'", diffs[0].Path)
				}
				if diffs[0].Type != diffyml.DiffModified {
					t.Errorf("expected DiffModified, got %v", diffs[0].Type)
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

func TestCompare_MultipleMapChanges(t *testing.T) {
	from := yml(`---
config:
  key1: value1
  key2: value2
  key3: value3
`)
	to := yml(`---
config:
  key1: changed
  key3: value3
  key4: added
`)

	diffs, err := compare(from, to, nil)
	if err != nil {
		t.Fatalf("compare() error = %v", err)
	}
	if len(diffs) != 3 {
		t.Fatalf("expected 3 diffs (modified, removed, added), got %d", len(diffs))
	}
	var hasModified, hasRemoved, hasAdded bool
	for _, d := range diffs {
		switch {
		case d.Path.String() == "config.key1" && d.Type == diffyml.DiffModified:
			hasModified = true
		case d.Path.String() == "config" && d.Type == diffyml.DiffRemoved:
			if om, ok := d.From.(*diffyml.OrderedMap); ok {
				if _, exists := om.Values["key2"]; exists {
					hasRemoved = true
				}
			}
		case d.Path.String() == "config" && d.Type == diffyml.DiffAdded:
			if om, ok := d.To.(*diffyml.OrderedMap); ok {
				if _, exists := om.Values["key4"]; exists {
					hasAdded = true
				}
			}
		}
	}
	if !hasModified {
		t.Error("expected modified diff for config.key1")
	}
	if !hasRemoved {
		t.Error("expected removed diff for config (key2 wrapped in OrderedMap)")
	}
	if !hasAdded {
		t.Error("expected added diff for config (key4 wrapped in OrderedMap)")
	}
}

func TestCompare_MapEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		from    string
		to      string
		opts    *diffyml.Options
		wantErr bool
		check   func(t *testing.T, diffs []diffyml.Difference)
	}{
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
				if diffs[0].Path.String() != "data" {
					t.Errorf("expected path 'data', got '%s'", diffs[0].Path)
				}
				om, ok := diffs[0].To.(*diffyml.OrderedMap)
				if !ok {
					t.Fatalf("expected To to be *OrderedMap, got %T", diffs[0].To)
				}
				if len(om.Keys) != 1 || om.Keys[0] != "key" {
					t.Errorf("expected OrderedMap with key 'key', got keys %v", om.Keys)
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
				if diffs[0].Path.String() != "data" {
					t.Errorf("expected path 'data', got '%s'", diffs[0].Path)
				}
				om, ok := diffs[0].From.(*diffyml.OrderedMap)
				if !ok {
					t.Fatalf("expected From to be *OrderedMap, got %T", diffs[0].From)
				}
				if len(om.Keys) != 1 || om.Keys[0] != "key" {
					t.Errorf("expected OrderedMap with key 'key', got keys %v", om.Keys)
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

func TestCompare_ListScalarOps(t *testing.T) {
	tests := []struct {
		name    string
		from    string
		to      string
		opts    *diffyml.Options
		wantErr bool
		check   func(t *testing.T, diffs []diffyml.Difference)
	}{
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

func TestCompare_ListEntryPaths(t *testing.T) {
	tests := []struct {
		name    string
		from    string
		to      string
		opts    *diffyml.Options
		wantErr bool
		check   func(t *testing.T, diffs []diffyml.Difference)
	}{
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
				if diffs[0].Path.String() != "items.2" {
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
				if diffs[0].Path.String() != "items.2" {
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

func TestCompare_ListNestedPaths(t *testing.T) {
	tests := []struct {
		name    string
		from    string
		to      string
		opts    *diffyml.Options
		wantErr bool
		check   func(t *testing.T, diffs []diffyml.Difference)
	}{
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
				if diffs[0].Path.String() != "users.alice.age" {
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
				if diffs[0].Path.String() != "config.servers.2" {
					t.Errorf("expected path 'config.servers.2', got '%s'", diffs[0].Path)
				}
				if diffs[0].Type != diffyml.DiffAdded {
					t.Errorf("expected DiffAdded, got %v", diffs[0].Type)
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

func TestCompare_ListByIdentifier(t *testing.T) {
	tests := []struct {
		name    string
		from    string
		to      string
		opts    *diffyml.Options
		wantErr bool
		check   func(t *testing.T, diffs []diffyml.Difference)
	}{
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
				if diffs[0].Path.String() != "root.level1.level2.items.1" {
					t.Errorf("expected path 'root.level1.level2.items.1', got '%s'", diffs[0].Path)
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

func TestCompare_ListEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		from    string
		to      string
		opts    *diffyml.Options
		wantErr bool
		check   func(t *testing.T, diffs []diffyml.Difference)
	}{
		{
			name: "list with mixed types",
			from: `data: [1, "two", true, {key: value}]`,
			to:   `data: [1, "two", false, {key: value}]`,
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff, got %d", len(diffs))
				}
				if diffs[0].Path.String() != "data.2" {
					t.Errorf("expected path 'data.2', got '%s'", diffs[0].Path)
				}
				if diffs[0].Type != diffyml.DiffModified {
					t.Errorf("expected DiffModified, got %v", diffs[0].Type)
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

func TestCompare_ListEmptyNonEmpty(t *testing.T) {
	tests := []struct {
		name    string
		from    string
		to      string
		opts    *diffyml.Options
		wantErr bool
		check   func(t *testing.T, diffs []diffyml.Difference)
	}{
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

func TestCompare_SwapOption(t *testing.T) {
	tests := []struct {
		name    string
		from    string
		to      string
		opts    *diffyml.Options
		wantErr bool
		check   func(t *testing.T, diffs []diffyml.Difference)
	}{
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

func TestCompare_IgnoreValues(t *testing.T) {
	tests := []struct {
		name    string
		from    string
		to      string
		opts    *diffyml.Options
		wantErr bool
		check   func(t *testing.T, diffs []diffyml.Difference)
	}{
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

func TestCompare_IgnoreWhitespace(t *testing.T) {
	tests := []struct {
		name    string
		from    string
		to      string
		opts    *diffyml.Options
		wantErr bool
		check   func(t *testing.T, diffs []diffyml.Difference)
	}{
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

func TestCompare_FormatStrings(t *testing.T) {
	tests := []struct {
		name  string
		from  string
		to    string
		opts  *diffyml.Options
		check func(t *testing.T, diffs []diffyml.Difference)
	}{
		{
			name: "formatting-only JSON diff suppressed",
			from: `data: '{"key":"value","nested":{"foo":"bar"}}'`,
			to:   "data: |\n  {\n    \"key\": \"value\",\n    \"nested\": {\n      \"foo\": \"bar\"\n    }\n  }",
			opts: &diffyml.Options{FormatStrings: true},
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 0 {
					t.Errorf("expected 0 diffs, got %d", len(diffs))
				}
			},
		},
		{
			name: "semantic JSON diff still detected",
			from: `data: '{"key":"old"}'`,
			to:   `data: '{"key":"new"}'`,
			opts: &diffyml.Options{FormatStrings: true},
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff, got %d", len(diffs))
				}
				if diffs[0].Type != diffyml.DiffModified {
					t.Errorf("expected DiffModified, got %v", diffs[0].Type)
				}
			},
		},
		{
			name: "non-JSON strings compared normally",
			from: `data: "hello world"`,
			to:   `data: "hello mars"`,
			opts: &diffyml.Options{FormatStrings: true},
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff, got %d", len(diffs))
				}
			},
		},
		{
			name: "one side JSON other not - falls through",
			from: `data: '{"key":"value"}'`,
			to:   `data: "not json"`,
			opts: &diffyml.Options{FormatStrings: true},
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff, got %d", len(diffs))
				}
			},
		},
		{
			name: "first string looks like JSON but is invalid - falls through",
			from: `data: '{invalid json}'`,
			to:   `data: '{"key":"value"}'`,
			opts: &diffyml.Options{FormatStrings: true},
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff, got %d", len(diffs))
				}
			},
		},
		{
			name: "second string looks like JSON but is invalid - falls through",
			from: `data: '{"key":"value"}'`,
			to:   `data: '{not valid}'`,
			opts: &diffyml.Options{FormatStrings: true},
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff, got %d", len(diffs))
				}
			},
		},
		{
			name: "JSON key order differences ignored",
			from: `data: '{"a":1,"b":2}'`,
			to:   `data: '{"b":2,"a":1}'`,
			opts: &diffyml.Options{FormatStrings: true},
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 0 {
					t.Errorf("expected 0 diffs, got %d", len(diffs))
				}
			},
		},
		{
			name: "feature disabled - formatting diff reported",
			from: `data: '{"key":"value"}'`,
			to:   "data: |\n  {\n    \"key\": \"value\"\n  }",
			opts: &diffyml.Options{FormatStrings: false},
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff when format-strings disabled, got %d", len(diffs))
				}
			},
		},
		{
			name: "JSON arrays preserve order",
			from: `data: '[1,2,3]'`,
			to:   `data: '[3,2,1]'`,
			opts: &diffyml.Options{FormatStrings: true},
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff for reordered array, got %d", len(diffs))
				}
			},
		},
		{
			name: "identical strings - fast path",
			from: `data: '{"key":"value"}'`,
			to:   `data: '{"key":"value"}'`,
			opts: &diffyml.Options{FormatStrings: true},
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 0 {
					t.Errorf("expected 0 diffs, got %d", len(diffs))
				}
			},
		},
		{
			name: "format-strings with ignore-whitespace - JSON formatting ignored",
			from: `data: '{"key":"value"}'`,
			to:   "data: |\n  {\n    \"key\": \"value\"\n  }",
			opts: &diffyml.Options{FormatStrings: true, IgnoreWhitespaceChanges: true},
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 0 {
					t.Errorf("expected 0 diffs, got %d", len(diffs))
				}
			},
		},
		{
			name: "JSON array formatting-only diff suppressed",
			from: `data: '[1, 2, 3]'`,
			to:   "data: |\n  [\n    1,\n    2,\n    3\n  ]",
			opts: &diffyml.Options{FormatStrings: true},
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 0 {
					t.Errorf("expected 0 diffs, got %d", len(diffs))
				}
			},
		},
		{
			name: "empty JSON object with whitespace formatting suppressed",
			from: `data: '{}'`,
			to:   "data: '{ }'",
			opts: &diffyml.Options{FormatStrings: true},
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 0 {
					t.Errorf("expected 0 diffs, got %d", len(diffs))
				}
			},
		},
		{
			name: "format-strings with ignore-whitespace - non-JSON whitespace ignored",
			from: `data: "hello"`,
			to:   `data: "  hello  "`,
			opts: &diffyml.Options{FormatStrings: true, IgnoreWhitespaceChanges: true},
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 0 {
					t.Errorf("expected 0 diffs, got %d", len(diffs))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from := yml(tt.from)
			to := yml(tt.to)

			diffs, err := compare(from, to, tt.opts)
			if err != nil {
				t.Fatalf("compare() error = %v", err)
			}

			if tt.check != nil {
				tt.check(t, diffs)
			}
		})
	}
}

func TestCompare_IgnoreOrder(t *testing.T) {
	tests := []struct {
		name    string
		from    string
		to      string
		opts    *diffyml.Options
		wantErr bool
		check   func(t *testing.T, diffs []diffyml.Difference)
	}{
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
			name: "ignore order - nested modification produces precise diff",
			from: `---
items:
  - host: example.com
    port: 80
`,
			to: `---
items:
  - host: example.com
    port: 8080
`,
			opts: &diffyml.Options{IgnoreOrderChanges: true},
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff, got %d: %v", len(diffs), diffs)
				}
				if diffs[0].Type != diffyml.DiffModified {
					t.Errorf("expected DiffModified, got %v", diffs[0].Type)
				}
				if diffs[0].Path.String() != "items.0.port" {
					t.Errorf("expected path items.0.port, got %s", diffs[0].Path.String())
				}
			},
		},
		{
			name: "ignore order - reordered list with nested modification",
			from: `---
items:
  - host: a.com
    port: 80
  - host: b.com
    port: 443
`,
			to: `---
items:
  - host: b.com
    port: 443
  - host: a.com
    port: 8080
`,
			opts: &diffyml.Options{IgnoreOrderChanges: true},
			check: func(t *testing.T, diffs []diffyml.Difference) {
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff, got %d: %v", len(diffs), diffs)
				}
				if diffs[0].Type != diffyml.DiffModified {
					t.Errorf("expected DiffModified, got %v", diffs[0].Type)
				}
			},
		},
		{
			name: "ignore order - mixed exact and modified items",
			from: `---
items:
  - host: a.com
    port: 80
  - host: b.com
    port: 443
  - host: c.com
    port: 9090
`,
			to: `---
items:
  - host: b.com
    port: 443
  - host: c.com
    port: 9090
  - host: a.com
    port: 8080
`,
			opts: &diffyml.Options{IgnoreOrderChanges: true},
			check: func(t *testing.T, diffs []diffyml.Difference) {
				// b.com and c.com match exactly; a.com has port change
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff, got %d: %v", len(diffs), diffs)
				}
				if diffs[0].Type != diffyml.DiffModified {
					t.Errorf("expected DiffModified, got %v", diffs[0].Type)
				}
			},
		},
		{
			name: "ignore order - addition and removal with modification",
			from: `---
items:
  - host: a.com
    port: 80
  - host: removed.com
    port: 999
`,
			to: `---
items:
  - host: a.com
    port: 8080
  - host: added.com
    port: 111
`,
			opts: &diffyml.Options{IgnoreOrderChanges: true},
			check: func(t *testing.T, diffs []diffyml.Difference) {
				// No exact matches. Positional comparison:
				// a.com(80) vs a.com(8080) → modified port
				// removed.com vs added.com → modified host + port
				if len(diffs) == 0 {
					t.Fatal("expected diffs, got 0")
				}
				// All diffs should be modifications (positional comparison),
				// not remove+add
				for _, d := range diffs {
					if d.Type == diffyml.DiffRemoved || d.Type == diffyml.DiffAdded {
						t.Errorf("expected modifications not remove/add, got %v at %s", d.Type, d.Path)
					}
				}
			},
		},
		{
			name: "ignore order - scalar list with actual value change",
			from: `items: [a, b, c]`,
			to:   `items: [c, b, d]`,
			opts: &diffyml.Options{IgnoreOrderChanges: true},
			check: func(t *testing.T, diffs []diffyml.Difference) {
				// a and d don't match anything exactly.
				// "c" and "b" match exactly. Remaining: a vs d → modified
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff, got %d: %v", len(diffs), diffs)
				}
				if diffs[0].Type != diffyml.DiffModified {
					t.Errorf("expected DiffModified, got %v", diffs[0].Type)
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

func TestCompare_CombinedOptions(t *testing.T) {
	tests := []struct {
		name    string
		from    string
		to      string
		opts    *diffyml.Options
		wantErr bool
		check   func(t *testing.T, diffs []diffyml.Difference)
	}{
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

func TestCompare_AdditionalIdentifierReorder_OrderChanged(t *testing.T) {
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
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff (order change), got %d", len(diffs))
	}
	if diffs[0].Type != diffyml.DiffOrderChanged {
		t.Errorf("expected DiffOrderChanged, got %v", diffs[0].Type)
	}
}

func TestCompare_AdditionalIdentifierReorder_IgnoredWhenConfigured(t *testing.T) {
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
		IgnoreOrderChanges:    true,
	})
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	if len(diffs) != 0 {
		t.Fatalf("expected no diffs with IgnoreOrderChanges, got %d", len(diffs))
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
	if diffs[0].Path.String() != "items.0.value" {
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
	if diffs[0].Path.String() != "items.1" {
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
		if strings.HasPrefix(d.Path.String(), "[1]") {
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
	if diffs[0].Path.String() != "z" {
		t.Errorf("expected first diff path 'z', got %q", diffs[0].Path)
	}
	if diffs[1].Path.String() != "a" {
		t.Errorf("expected second diff path 'a', got %q", diffs[1].Path)
	}
	if diffs[2].Path.String() != "m" {
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
	if diffs[0].Path.String() != "items.9" {
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
	// Root-level addition (empty parent path) comes before nested modification
	if diffs[0].Type != diffyml.DiffAdded || diffs[0].Path.String() != "" {
		t.Errorf("expected first diff to be addition at root (empty path), got type=%v path=%q",
			diffs[0].Type, diffs[0].Path)
	}
	om, ok := diffs[0].To.(*diffyml.OrderedMap)
	if !ok {
		t.Fatalf("expected To to be *OrderedMap, got %T", diffs[0].To)
	}
	if _, exists := om.Values["newroot"]; !exists {
		t.Errorf("expected OrderedMap to contain key 'newroot', got keys %v", om.Keys)
	}
	if diffs[1].Type != diffyml.DiffModified || diffs[1].Path.String() != "existing.nested" {
		t.Errorf("expected second diff to be modification 'existing.nested', got type=%v path=%q",
			diffs[1].Type, diffs[1].Path)
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
	if diffs[0].Path.String() != "root.shallow" {
		t.Errorf("expected first diff 'root.shallow', got %q", diffs[0].Path)
	}
	if diffs[1].Path.String() != "root.deep.nested" {
		t.Errorf("expected second diff 'root.deep.nested', got %q", diffs[1].Path)
	}
}

func TestCompare_RootModifyBeforeRootAdd(t *testing.T) {
	// A root-level modification (present in source) must be sorted before
	// a root-level addition (target-only), because source paths get lower
	// pathOrder numbers.
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
	// Root-level addition (empty parent path) comes before root-level modification
	if diffs[0].Type != diffyml.DiffAdded || diffs[0].Path.String() != "" {
		t.Errorf("expected first diff to be addition at root (empty path), got type=%v path=%q",
			diffs[0].Type, diffs[0].Path)
	}
	om, ok := diffs[0].To.(*diffyml.OrderedMap)
	if !ok {
		t.Fatalf("expected To to be *OrderedMap, got %T", diffs[0].To)
	}
	if _, exists := om.Values["a_added"]; !exists {
		t.Errorf("expected OrderedMap to contain key 'a_added', got keys %v", om.Keys)
	}
	if diffs[1].Type != diffyml.DiffModified || diffs[1].Path.String() != "z_modified" {
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

func TestCompare_AdditionalIdentifierModify(t *testing.T) {
	// When using AdditionalIdentifiers, modifying an identified item must produce
	// DiffModified (not remove+add) and use the identifier in the path.
	from := yml(`items:
  - key: alpha
    value: 1`)
	to := yml(`items:
  - key: alpha
    value: 2`)

	diffs, err := compare(from, to, &diffyml.Options{
		AdditionalIdentifiers: []string{"key"},
	})
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].Type != diffyml.DiffModified {
		t.Errorf("expected DiffModified, got %v", diffs[0].Type)
	}
	if !strings.Contains(diffs[0].Path.String(), "alpha") {
		t.Errorf("expected path to contain identifier 'alpha', got %q", diffs[0].Path)
	}
}

func TestCompare_ParentOrderForAddedChildren(t *testing.T) {
	// Added children under different parents (same root) must sort
	// by the parents' document order, not alphabetically.
	from := yml(`root:
  zzz:
    child: old
  aaa:
    child: old`)
	to := yml(`root:
  zzz:
    child: old
    newkey: added_z
  aaa:
    child: old
    newkey: added_a`)

	diffs, err := compare(from, to, nil)
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	if len(diffs) != 2 {
		t.Fatalf("expected 2 diffs, got %d", len(diffs))
	}
	// zzz appears before aaa in the document, so zzz addition must come first
	if diffs[0].Path.String() != "root.zzz" {
		t.Errorf("expected first diff path 'root.zzz', got %q", diffs[0].Path)
	}
	if om, ok := diffs[0].To.(*diffyml.OrderedMap); !ok {
		t.Errorf("expected first diff To to be *OrderedMap, got %T", diffs[0].To)
	} else if _, exists := om.Values["newkey"]; !exists {
		t.Errorf("expected first diff OrderedMap to contain key 'newkey', got keys %v", om.Keys)
	}
	if diffs[1].Path.String() != "root.aaa" {
		t.Errorf("expected second diff path 'root.aaa', got %q", diffs[1].Path)
	}
	if om, ok := diffs[1].To.(*diffyml.OrderedMap); !ok {
		t.Errorf("expected second diff To to be *OrderedMap, got %T", diffs[1].To)
	} else if _, exists := om.Values["newkey"]; !exists {
		t.Errorf("expected second diff OrderedMap to contain key 'newkey', got keys %v", om.Keys)
	}
}

func TestCompare_UnorderedListNullVsValue(t *testing.T) {
	// In unordered comparison, a non-null item must not match a null item.
	from := yml(`items: [hello]`)
	to := yml(`items: [null]`)

	diffs, err := compare(from, to, &diffyml.Options{IgnoreOrderChanges: true})
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	// "hello" removed and null added — at least 1 diff expected
	if len(diffs) == 0 {
		t.Fatal("expected diffs when replacing a value with null, got 0")
	}
}

// --- Tests adapted from dyff edge cases ---

func TestCompare_BooleanNormalization(t *testing.T) {
	// dyff issue: different representations of true (true vs True) should be identical.
	from := yml(`---
key: true`)
	to := yml(`---
key: True`)

	diffs, err := compare(from, to, nil)
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	if len(diffs) != 0 {
		t.Errorf("expected 0 diffs for boolean normalization (true vs True), got %d", len(diffs))
	}
}

func TestCompare_BooleanNormalizationFalse(t *testing.T) {
	from := yml(`---
key: false`)
	to := yml(`---
key: False`)

	diffs, err := compare(from, to, nil)
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	if len(diffs) != 0 {
		t.Errorf("expected 0 diffs for boolean normalization (false vs False), got %d", len(diffs))
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

func TestCompare_YAMLAnchorsAndAliases(t *testing.T) {
	// dyff issue-76: YAML anchors and aliases must be resolved before comparison.
	from := yml(`---
global_defaults: &global_defaults
  - x1
  - x5
cluster-1:
  - *global_defaults`)
	to := yml(`---
global_defaults: &global_defaults
  - x1
  - x5
  - x10
cluster-1:
  - *global_defaults
  - x999`)

	diffs, err := compare(from, to, nil)
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	if len(diffs) == 0 {
		t.Fatal("expected diffs for YAML anchor changes, got 0")
	}
	// Should detect the added x10 in global_defaults and x999 in cluster-1
	hasGlobalAdd := false
	hasClusterAdd := false
	for _, d := range diffs {
		if d.Type == diffyml.DiffAdded {
			if strings.HasPrefix(d.Path.String(), "global_defaults") {
				hasGlobalAdd = true
			}
			if strings.HasPrefix(d.Path.String(), "cluster-1") {
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
}

func TestCompare_TypeChangeMapToList(t *testing.T) {
	// dyff issue-89: type change from map to list must be reported.
	from := yml(`---
foo:
  a: 1
  b: 2`)
	to := yml(`---
foo:
  - 1
  - 2`)

	diffs, err := compare(from, to, nil)
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	if len(diffs) == 0 {
		t.Fatal("expected diffs for map-to-list type change, got 0")
	}
	if !hasDiffType(diffs, diffyml.DiffModified) {
		t.Error("expected DiffModified for type change from map to list")
	}
}

func TestCompare_TypeChangeMapToNull(t *testing.T) {
	// dyff issue-89: type change from map to null must be reported.
	from := yml(`---
bar:
  c: 3
  d: 4`)
	to := yml(`---
bar:`)

	diffs, err := compare(from, to, nil)
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	if len(diffs) == 0 {
		t.Fatal("expected diffs for map-to-null type change, got 0")
	}
}

func TestCompare_EmptyDocumentsIgnored(t *testing.T) {
	// Empty documents (extra --- separators) are filtered by the CLI pipeline.
	// At the Compare level, they cause positional mismatches.
	// This test verifies Compare doesn't crash on empty documents.
	// The fixture test 044-empty-documents-ignored verifies the full CLI behavior.
	from := yml(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: x
---
apiVersion: v1
kind: Service
metadata:
  name: y`)
	to := yml(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: x
---
---
apiVersion: v1
kind: Service
metadata:
  name: y
---`)

	_, err := compare(from, to, nil)
	if err != nil {
		t.Fatalf("compare() should not error on empty documents: %v", err)
	}
}

func TestCompare_K8sDocumentAdded(t *testing.T) {
	// dyff compare_test: adding a new K8s document should be reported.
	from := yml(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: x`)
	to := yml(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: x
---
apiVersion: v1
kind: Service
metadata:
  name: y`)

	diffs, err := compare(from, to, nil)
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	if !hasDiffType(diffs, diffyml.DiffAdded) {
		t.Error("expected DiffAdded for new K8s document")
	}
}

func TestCompare_K8sDocumentRemoved(t *testing.T) {
	// Removing a K8s document should be reported as a diff.
	// At the Compare level, a document present in from but not in to is
	// reported as DiffModified (from=doc, to=nil).
	from := yml(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: x
---
apiVersion: v1
kind: Service
metadata:
  name: y`)
	to := yml(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: x`)

	diffs, err := compare(from, to, nil)
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	if len(diffs) == 0 {
		t.Fatal("expected diffs for removed K8s document, got 0")
	}
	// The removed document should appear in the diff path as [1]
	found := false
	for _, d := range diffs {
		if strings.HasPrefix(d.Path.String(), "[1]") {
			found = true
		}
	}
	if !found {
		t.Error("expected diff for second document [1]")
	}
}

func TestCompare_TemplateVariablesPreserved(t *testing.T) {
	// dyff issue-132: template variables must not be evaluated.
	from := yml(`---
example_one: "%{one}"
example_two: "two"`)
	to := yml(`---
example_one: "one"
example_two: "%{two}"`)

	diffs, err := compare(from, to, nil)
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	if len(diffs) != 2 {
		t.Fatalf("expected 2 diffs (both values changed), got %d", len(diffs))
	}
	if !hasModification(diffs, "%{one}", "one") {
		t.Error("expected modification from '%{one}' to 'one'")
	}
	if !hasModification(diffs, "two", "%{two}") {
		t.Error("expected modification from 'two' to '%{two}'")
	}
}

func TestCompare_DuplicateListEntries(t *testing.T) {
	// dyff issue-143: lists with duplicate entries must be handled correctly.
	from := yml(`keys:
  - value1
  - value2`)
	to := yml(`keys:
  - value1
  - value2
  - value1`)

	diffs, err := compare(from, to, nil)
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff (added duplicate), got %d", len(diffs))
	}
	if diffs[0].Type != diffyml.DiffAdded {
		t.Errorf("expected DiffAdded, got %v", diffs[0].Type)
	}
}

// --- Mutation testing: comparator.go ---

// hasK8sDocuments must short-circuit on the first K8s resource found in `from`,
// even when `to` contains no K8s resources. Without the early return, the from
// loop completes and the to loop returns false, taking the generic path and
// producing key-level diffs instead of the K8s doc-level remove/add.
func TestCompare_K8sDetection_OnlyFromHasResources(t *testing.T) {
	from := yml(`apiVersion: v1
kind: ConfigMap
metadata:
  name: cm-a
data:
  key: value`)
	to := yml(`unrelated: scalar`)

	diffs, err := compare(from, to, &diffyml.Options{DetectKubernetes: true})
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	// Only the K8s path stamps DocumentKind on resource-level diffs.
	foundCMKind := false
	for _, d := range diffs {
		if d.DocumentKind == "ConfigMap" {
			foundCMKind = true
		}
	}
	if !foundCMKind {
		t.Errorf("expected a diff carrying DocumentKind=ConfigMap (K8s compare path), got %+v", diffs)
	}
}

// Both root documents parse to nil — compareNodeNils must short-circuit, not
// fall through to the from==nil branch and emit a bogus DiffAdded.
func TestCompare_BothNullDocuments_NoDiffs(t *testing.T) {
	from := yml(`~`)
	to := yml(`~`)

	diffs, err := compare(from, to, nil)
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	if len(diffs) != 0 {
		t.Fatalf("expected 0 diffs for null vs null, got %d: %+v", len(diffs), diffs)
	}
}

// canMatchByIdentifier must return false on either side independently. With
// `from` carrying name keys but `to` carrying none (both all-map lists),
// taking the ID-matching path produces list-level Remove+Add at the parent
// "items" path. Positional compare instead drills into a Modified at
// "items.0.value", which is the assertion below.
func TestCompare_IdentifierMatching_AsymmetricSides(t *testing.T) {
	tests := []struct {
		name string
		from string
		to   string
	}{
		{
			name: "from has name, to does not",
			from: `items:
  - name: foo
    value: 1`,
			to: `items:
  - other: bar
    value: 2`,
		},
		{
			name: "from has no name, to has name",
			from: `items:
  - other: bar
    value: 1`,
			to: `items:
  - name: foo
    value: 2`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diffs, err := compare(yml(tt.from), yml(tt.to), nil)
			if err != nil {
				t.Fatalf("compare() failed: %v", err)
			}
			foundValuePath := false
			for _, d := range diffs {
				if strings.Contains(d.Path.String(), "value") {
					foundValuePath = true
				}
			}
			if !foundValuePath {
				t.Errorf("expected positional Modified under '.value' path, got %+v", diffs)
			}
		})
	}
}

// areListItemsHeterogeneous: OrderedMap items with multiple keys must NOT be
// treated as heterogeneous (the single-key heuristic doesn't apply). Swapping
// them should produce diffs under positional comparison; the mutated case
// body would silently exact-match the swap under unordered comparison.
func TestCompare_HeterogeneityCheck_MultiKeyOrderedMapIsHomogeneous(t *testing.T) {
	from := yml(`rules:
  - a: 1
    b: 2
  - c: 3
    d: 4`)
	to := yml(`rules:
  - c: 3
    d: 4
  - a: 1
    b: 2`)

	diffs, err := compare(from, to, nil)
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	if len(diffs) == 0 {
		t.Fatal("expected diffs from positional compare of swapped multi-key items, got 0")
	}
}

// areListItemsHeterogeneous: a non-map (scalar) item must break the
// single-key heuristic via the default case. With allKeys spanning multiple
// distinct map keys, the original returns false (homogeneous → positional);
// the mutated default returns true (heterogeneous → unordered) and matches
// the swapped maps exactly, hiding the scalar Modified at index 0.
func TestCompare_HeterogeneityCheck_ScalarItemBreaksHomogeneity(t *testing.T) {
	from := yml(`items:
  - 1
  - a: 1
  - b: 1`)
	to := yml(`items:
  - 2
  - b: 1
  - a: 1`)

	diffs, err := compare(from, to, nil)
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	// Positional compare yields a diff at items.1 (a:1 vs b:1) AND at items.2.
	// Heterogeneous unordered would exact-match those swaps and only diff items.0.
	foundIndex1Or2 := false
	for _, d := range diffs {
		p := d.Path.String()
		if strings.HasPrefix(p, "items.1") || strings.HasPrefix(p, "items.2") {
			foundIndex1Or2 = true
		}
	}
	if !foundIndex1Or2 {
		t.Errorf("expected positional diff under items.1 or items.2, got %+v", diffs)
	}
}

// areListItemsHeterogeneous: checkSingleDistinctKeys(to) must veto the
// heterogeneous classification when `to` carries a multi-key item. The
// mutant forcing it to true would re-route to unordered comparison and
// hide swap-induced diffs.
func TestCompare_HeterogeneityCheck_AsymmetricSingleKeyShape(t *testing.T) {
	from := yml(`items:
  - a: 1
  - b: 2
  - c: 3`)
	to := yml(`items:
  - x: 1
    y: 2
  - a: 1
  - b: 2`)

	diffs, err := compare(from, to, nil)
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	// Positional compare emits per-key Removed+Added at items.0 plus
	// shape-changes at items.1, items.2. Unordered heterogeneous would
	// exact-match {a:1} and {b:2} and only diff items.0 (or items.2).
	pathSet := map[string]bool{}
	for _, d := range diffs {
		pathSet[d.Path.String()] = true
	}
	// Look for diffs at multiple item indexes to confirm positional path.
	count := 0
	for p := range pathSet {
		if strings.HasPrefix(p, "items.0") || strings.HasPrefix(p, "items.1") || strings.HasPrefix(p, "items.2") {
			count++
		}
	}
	if count < 2 {
		t.Errorf("expected diffs at multiple items.N paths (positional), got %+v", diffs)
	}
}

// compareListsUnordered must skip already-matched to-items in the exact-match
// scan. With duplicate "a" in `from` and a single "a" in `to`, the second
// from-"a" must remain unmatched and surface as DiffRemoved.
func TestCompare_IgnoreOrder_DuplicatesFromSide(t *testing.T) {
	from := yml(`items: [a, a]`)
	to := yml(`items: [a]`)

	diffs, err := compare(from, to, &diffyml.Options{IgnoreOrderChanges: true})
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	if !hasDiffType(diffs, diffyml.DiffRemoved) {
		t.Errorf("expected DiffRemoved for the second duplicate, got %+v", diffs)
	}
}

// compareListsUnordered must `break` after pairing fromItem[i] with one
// toItem[j]. If `break` becomes `continue`, the same fromItem keeps matching
// further toItems and hides the second-duplicate DiffAdded.
func TestCompare_IgnoreOrder_DuplicatesToSide(t *testing.T) {
	from := yml(`items: [a]`)
	to := yml(`items: [a, a]`)

	diffs, err := compare(from, to, &diffyml.Options{IgnoreOrderChanges: true})
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	if !hasDiffType(diffs, diffyml.DiffAdded) {
		t.Errorf("expected DiffAdded for the second duplicate, got %+v", diffs)
	}
}

// compareListsUnordered trailing fi cleanup must `continue` past matched
// indices, not `break`. With fromMatched=[F,T,F], breaking on index 1 would
// drop the Removed for index 2 ("c").
func TestCompare_IgnoreOrder_UnmatchedAfterMatchedFromIdx(t *testing.T) {
	from := yml(`items: [a, b, c]`)
	to := yml(`items: [b]`)

	diffs, err := compare(from, to, &diffyml.Options{IgnoreOrderChanges: true})
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	removed := 0
	for _, d := range diffs {
		if d.Type == diffyml.DiffRemoved {
			removed++
		}
	}
	if removed != 2 {
		t.Errorf("expected 2 DiffRemoved for unmatched a and c, got %d (%+v)", removed, diffs)
	}
}

// Mirror of the previous test for the `to` cleanup loop.
func TestCompare_IgnoreOrder_UnmatchedAfterMatchedToIdx(t *testing.T) {
	from := yml(`items: [b]`)
	to := yml(`items: [a, b, c]`)

	diffs, err := compare(from, to, &diffyml.Options{IgnoreOrderChanges: true})
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	added := 0
	for _, d := range diffs {
		if d.Type == diffyml.DiffAdded {
			added++
		}
	}
	if added != 2 {
		t.Errorf("expected 2 DiffAdded for unmatched a and c, got %d (%+v)", added, diffs)
	}
}

// detectListOrderChanges must skip its order-change report when identifiers
// are not unique on either side. Two from-items share name=a, so the
// duplicate-detection guard `hasUniqueIDs` is false and no DiffOrderChanged
// should be emitted (regardless of from/to ordering).
func TestCompare_OrderChange_SkippedWhenDuplicateIdentifiers(t *testing.T) {
	from := yml(`items:
  - name: a
    v: 1
  - name: a
    v: 2
  - name: b
    v: 3`)
	to := yml(`items:
  - name: b
    v: 3
  - name: a
    v: 1
  - name: a
    v: 2`)

	diffs, err := compare(from, to, nil)
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	for _, d := range diffs {
		if d.Type == diffyml.DiffOrderChanged {
			t.Errorf("did not expect DiffOrderChanged with duplicate identifiers, got %+v", d)
		}
	}
}

// compareUnidentifiedItems must skip already-matched to-items in the
// exact-match scan (mirror of the compareListsUnordered case). Reached via a
// mixed-identifier list where {foo:1} entries fall into the unidentified
// fallback.
func TestCompare_MixedIdentifierList_DuplicateUnidentifiedItems(t *testing.T) {
	from := yml(`items:
  - name: keep
    v: 1
  - foo: 1
  - foo: 1`)
	to := yml(`items:
  - name: keep
    v: 1
  - foo: 1`)

	diffs, err := compare(from, to, nil)
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	if !hasDiffType(diffs, diffyml.DiffRemoved) {
		t.Errorf("expected DiffRemoved for the second {foo:1}, got %+v", diffs)
	}
}

// Mirror for the to-side cleanup loop in compareUnidentifiedItems.
func TestCompare_MixedIdentifierList_ToSideHasExtraUnidentified(t *testing.T) {
	from := yml(`items:
  - name: keep
    v: 1
  - foo: 1`)
	to := yml(`items:
  - name: keep
    v: 1
  - foo: 1
  - foo: 1`)

	diffs, err := compare(from, to, nil)
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	if !hasDiffType(diffs, diffyml.DiffAdded) {
		t.Errorf("expected DiffAdded for the extra {foo:1}, got %+v", diffs)
	}
}

// Drives the fromNoIDMatched-skip in the compareUnidentifiedItems cleanup
// loop: a matched unidentified item is sandwiched between two unmatched
// ones, so `continue` (not `break`) is required to surface both.
func TestCompare_MixedIdentifierList_UnmatchedAroundMatched(t *testing.T) {
	from := yml(`items:
  - name: keep
    v: 1
  - x: 1
  - y: 2
  - z: 3`)
	to := yml(`items:
  - name: keep
    v: 1
  - y: 2`)

	diffs, err := compare(from, to, nil)
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	removed := 0
	for _, d := range diffs {
		if d.Type == diffyml.DiffRemoved {
			removed++
		}
	}
	if removed < 2 {
		t.Errorf("expected at least 2 DiffRemoved (x:1 and z:3), got %d (%+v)", removed, diffs)
	}
}

// couldBeJSON must require a leading '{' or '[' — not just len>=2.
// "123" and "123 " both unmarshal as JSON numbers and canonicalize equal;
// with the start-char check mutated to `true`, they'd be reported equal
// and the diff would disappear.
func TestCompare_FormatStrings_NonJSONShapedStringsStillDiff(t *testing.T) {
	from := yml(`data: "123"`)
	to := yml(`data: "123 "`)

	diffs, err := compare(from, to, &diffyml.Options{FormatStrings: true})
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff for non-JSON-shaped strings, got %d: %+v", len(diffs), diffs)
	}
}

// compareListsUnordered walk loop must `continue` past an already-matched
// fromMatched[fi]. With break, a leading exact match would stop the
// pair-walk entirely and convert the remaining "modified" item into a
// spurious Remove+Add at the same path.
func TestCompare_IgnoreOrder_WalkSkipsMatchedFromPrefix(t *testing.T) {
	from := yml(`items: [a, b]`)
	to := yml(`items: [a, c]`)

	diffs, err := compare(from, to, &diffyml.Options{IgnoreOrderChanges: true})
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff (b→c modified), got %d: %+v", len(diffs), diffs)
	}
	if diffs[0].Type != diffyml.DiffModified {
		t.Errorf("expected DiffModified (walk should pair b and c), got %v", diffs[0].Type)
	}
}

// compareUnidentifiedItems walk loop must `continue` past an already-matched
// fromNoIDMatched[fi]. Same shape as the previous test but routed through
// the mixed-identifier fallback.
func TestCompare_MixedIdentifierList_WalkSkipsMatchedFromPrefix(t *testing.T) {
	from := yml(`items:
  - name: keep
  - match: 1
  - value: 1`)
	to := yml(`items:
  - name: keep
  - match: 1
  - value: 2`)

	diffs, err := compare(from, to, nil)
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	for _, d := range diffs {
		if d.Type == diffyml.DiffRemoved || d.Type == diffyml.DiffAdded {
			t.Errorf("expected pure DiffModified (walk should pair value:1 and value:2), got %v at %s", d.Type, d.Path)
		}
	}
	if !hasDiffType(diffs, diffyml.DiffModified) {
		t.Errorf("expected DiffModified at items.2.value, got %+v", diffs)
	}
}

// compareUnidentifiedItems inner scan must `continue` past an already-matched
// toNoIDMatched[tj], not `break`. Breaking would prevent a later fromItem
// from finding its real match further down the toNoID slice.
func TestCompare_MixedIdentifierList_InnerScanSkipsMatchedToPrefix(t *testing.T) {
	from := yml(`items:
  - name: keep
  - a: 1
  - b: 1
  - c: 1`)
	to := yml(`items:
  - name: keep
  - a: 1
  - c: 1
  - b: 1`)

	diffs, err := compare(from, to, nil)
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	if len(diffs) != 0 {
		t.Fatalf("expected 0 diffs (all items match exactly across order), got %d: %+v", len(diffs), diffs)
	}
}

// detectListOrderChanges' hasUniqueIDs guard combines two checks with &&.
// Asymmetric tests (one side has duplicate IDs, the other unique) prevent
// either operand from being silently mutated to `true`, and prevent
// `&&` → `||` from passing the guard.
func TestCompare_OrderChange_DuplicateFromUniqueTo(t *testing.T) {
	from := yml(`items:
  - name: a
  - name: a
  - name: b`)
	to := yml(`items:
  - name: b
  - name: a
  - name: c`)

	diffs, err := compare(from, to, nil)
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	for _, d := range diffs {
		if d.Type == diffyml.DiffOrderChanged {
			t.Errorf("did not expect DiffOrderChanged when from has duplicate IDs, got %+v", d)
		}
	}
}

func TestCompare_OrderChange_UniqueFromDuplicateTo(t *testing.T) {
	from := yml(`items:
  - name: a
  - name: b
  - name: c`)
	to := yml(`items:
  - name: c
  - name: a
  - name: a`)

	diffs, err := compare(from, to, nil)
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	for _, d := range diffs {
		if d.Type == diffyml.DiffOrderChanged {
			t.Errorf("did not expect DiffOrderChanged when to has duplicate IDs, got %+v", d)
		}
	}
}

// couldBeJSON's length guard must short-circuit before indexing s[0].
// Mutating `len(s) >= 2` to `true` (or `&&` to `||`) would access s[0]
// on an empty string and panic.
func TestCompare_FormatStrings_EmptyStringNoPanic(t *testing.T) {
	from := yml(`data: ""`)
	to := yml(`data: "x"`)

	diffs, err := compare(from, to, &diffyml.Options{FormatStrings: true})
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff for empty vs 'x', got %d: %+v", len(diffs), diffs)
	}
}

// deepEqualOrderedMaps must reject a size mismatch up front. With the length
// guard removed, a subset map would be wrongly considered equal to a
// superset in the exact-match scan of unordered list comparison.
func TestCompare_IgnoreOrder_SupersetMapMustNotMatchExactly(t *testing.T) {
	from := yml(`items:
  - a: 1`)
	to := yml(`items:
  - a: 1
    b: 2`)

	diffs, err := compare(from, to, &diffyml.Options{IgnoreOrderChanges: true})
	if err != nil {
		t.Fatalf("compare() failed: %v", err)
	}
	if len(diffs) == 0 {
		t.Fatal("expected at least 1 diff for the added key b, got 0")
	}
	if !hasDiffType(diffs, diffyml.DiffAdded) {
		t.Errorf("expected DiffAdded for new key b, got %+v", diffs)
	}
}
