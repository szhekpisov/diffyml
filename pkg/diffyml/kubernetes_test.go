package diffyml

import (
	"testing"
)

// Tests for Kubernetes resource detection (Task 2.5)

func TestIsKubernetesResource_Deployment(t *testing.T) {
	doc := map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]interface{}{
			"name":      "my-app",
			"namespace": "default",
		},
	}
	if !IsKubernetesResource(doc) {
		t.Error("expected Deployment to be detected as Kubernetes resource")
	}
}

func TestIsKubernetesResource_ConfigMap(t *testing.T) {
	doc := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name": "my-config",
		},
	}
	if !IsKubernetesResource(doc) {
		t.Error("expected ConfigMap to be detected as Kubernetes resource")
	}
}

func TestIsKubernetesResource_NotK8s_MissingApiVersion(t *testing.T) {
	doc := map[string]interface{}{
		"kind": "SomeThing",
		"metadata": map[string]interface{}{
			"name": "test",
		},
	}
	if IsKubernetesResource(doc) {
		t.Error("expected document without apiVersion to NOT be detected as K8s")
	}
}

func TestIsKubernetesResource_NotK8s_MissingKind(t *testing.T) {
	doc := map[string]interface{}{
		"apiVersion": "v1",
		"metadata": map[string]interface{}{
			"name": "test",
		},
	}
	if IsKubernetesResource(doc) {
		t.Error("expected document without kind to NOT be detected as K8s")
	}
}

func TestIsKubernetesResource_GenerateName(t *testing.T) {
	doc := map[string]interface{}{
		"apiVersion": "batch/v1",
		"kind":       "Job",
		"metadata": map[string]interface{}{
			"generateName": "my-job-",
		},
	}
	if !IsKubernetesResource(doc) {
		t.Error("expected resource with generateName to be detected as Kubernetes resource")
	}
}

func TestIsKubernetesResource_GenerateName_OrderedMap(t *testing.T) {
	meta := NewOrderedMap()
	meta.Keys = append(meta.Keys, "generateName")
	meta.Values["generateName"] = "my-job-"

	doc := NewOrderedMap()
	doc.Keys = append(doc.Keys, "apiVersion", "kind", "metadata")
	doc.Values["apiVersion"] = "batch/v1"
	doc.Values["kind"] = "Job"
	doc.Values["metadata"] = meta

	if !IsKubernetesResource(doc) {
		t.Error("expected OrderedMap resource with generateName to be detected as Kubernetes resource")
	}
}

func TestIsKubernetesResource_NoNameNorGenerateName(t *testing.T) {
	doc := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata":   map[string]interface{}{"labels": map[string]interface{}{"app": "test"}},
	}
	if IsKubernetesResource(doc) {
		t.Error("expected resource with neither name nor generateName to NOT be detected as K8s")
	}
}

func TestIsKubernetesResource_NotK8s_MissingMetadata(t *testing.T) {
	doc := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
	}
	if IsKubernetesResource(doc) {
		t.Error("expected document without metadata to NOT be detected as K8s")
	}
}

func TestIsKubernetesResource_NotK8s_MetadataNotMap(t *testing.T) {
	doc := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata":   "not-a-map",
	}
	if IsKubernetesResource(doc) {
		t.Error("expected document with invalid metadata to NOT be detected as K8s")
	}
}

func TestIsKubernetesResource_NotMap(t *testing.T) {
	doc := []interface{}{"item1", "item2"}
	if IsKubernetesResource(doc) {
		t.Error("expected non-map document to NOT be detected as K8s")
	}
}

func TestGetK8sResourceIdentifier_WithNamespace(t *testing.T) {
	doc := map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]interface{}{
			"name":      "my-app",
			"namespace": "production",
		},
	}
	id := GetK8sResourceIdentifier(doc, false)
	expected := "apps/v1:Deployment:production/my-app"
	if id != expected {
		t.Errorf("expected identifier %q, got %q", expected, id)
	}
}

func TestGetK8sResourceIdentifier_WithoutNamespace(t *testing.T) {
	doc := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name": "my-config",
		},
	}
	id := GetK8sResourceIdentifier(doc, false)
	expected := "v1:ConfigMap:my-config"
	if id != expected {
		t.Errorf("expected identifier %q, got %q", expected, id)
	}
}

func TestGetK8sResourceIdentifier_ClusterScoped(t *testing.T) {
	doc := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Namespace",
		"metadata": map[string]interface{}{
			"name": "my-namespace",
		},
	}
	id := GetK8sResourceIdentifier(doc, false)
	expected := "v1:Namespace:my-namespace"
	if id != expected {
		t.Errorf("expected identifier %q, got %q", expected, id)
	}
}

func TestGetK8sResourceIdentifier_NotK8s(t *testing.T) {
	doc := map[string]interface{}{
		"key": "value",
	}
	id := GetK8sResourceIdentifier(doc, false)
	if id != "" {
		t.Errorf("expected empty identifier for non-K8s doc, got %q", id)
	}
}

func TestGetK8sResourceIdentifier_IgnoreApiVersion_WithNamespace(t *testing.T) {
	doc := map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]interface{}{
			"name":      "my-app",
			"namespace": "production",
		},
	}
	id := GetK8sResourceIdentifier(doc, true)
	expected := "Deployment:production/my-app"
	if id != expected {
		t.Errorf("expected agnostic identifier %q, got %q", expected, id)
	}
}

func TestGetK8sResourceIdentifier_IgnoreApiVersion_WithoutNamespace(t *testing.T) {
	doc := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name": "my-config",
		},
	}
	id := GetK8sResourceIdentifier(doc, true)
	expected := "ConfigMap:my-config"
	if id != expected {
		t.Errorf("expected agnostic identifier %q, got %q", expected, id)
	}
}

func TestGetK8sResourceIdentifier_IgnoreApiVersion_NotK8s(t *testing.T) {
	doc := map[string]interface{}{
		"key": "value",
	}
	id := GetK8sResourceIdentifier(doc, true)
	if id != "" {
		t.Errorf("expected empty identifier for non-K8s doc with ignoreApiVersion, got %q", id)
	}
}

func TestCompare_K8sMultiDoc_MatchByIdentifier(t *testing.T) {
	from := `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config-a
data:
  key: value1
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config-b
data:
  key: value2
`
	to := `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config-b
data:
  key: value2-modified
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config-a
data:
  key: value1
`
	opts := &Options{
		DetectKubernetes: true,
	}
	diffs, err := Compare([]byte(from), []byte(to), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With K8s detection, config-a should match config-a (even though order changed)
	// and config-b should show modification
	hasConfigBChange := false
	for _, d := range diffs {
		if d.Type == DiffModified && d.From == "value2" && d.To == "value2-modified" {
			hasConfigBChange = true
		}
	}
	if !hasConfigBChange {
		t.Error("expected config-b data.key modification to be detected")
	}
}

func TestCompare_K8sDetectionDisabled_PositionalMatch(t *testing.T) {
	from := `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config-a
data:
  key: value1
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config-b
data:
  key: value2
`
	to := `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config-b
data:
  key: value2
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config-a
data:
  key: value1
`
	opts := &Options{
		DetectKubernetes: false,
	}
	diffs, err := Compare([]byte(from), []byte(to), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Without K8s detection, documents are compared positionally
	// doc[0] (config-a) vs doc[0] (config-b) should show name difference
	hasNameDiff := false
	for _, d := range diffs {
		if d.Type == DiffModified && d.From == "config-a" && d.To == "config-b" {
			hasNameDiff = true
		}
	}
	if !hasNameDiff {
		t.Error("expected name difference when K8s detection disabled")
	}
}

func TestGetIdentifierWithAdditional_DefaultFields(t *testing.T) {
	m := map[string]interface{}{
		"name": "test-name",
		"id":   "test-id",
	}
	id := GetIdentifierWithAdditional(m, nil)
	if id != "test-name" {
		t.Errorf("expected 'test-name', got %q", id)
	}
}

func TestGetIdentifierWithAdditional_CustomField(t *testing.T) {
	m := map[string]interface{}{
		"key":      "my-key",
		"otherKey": "ignored",
	}
	id := GetIdentifierWithAdditional(m, []string{"key"})
	if id != "my-key" {
		t.Errorf("expected 'my-key', got %q", id)
	}
}

func TestGetIdentifierWithAdditional_NoMatch(t *testing.T) {
	m := map[string]interface{}{
		"foo": "bar",
	}
	id := GetIdentifierWithAdditional(m, nil)
	if id != nil {
		t.Errorf("expected nil, got %v", id)
	}
}

func TestCanMatchByIdentifierWithAdditional_CustomFields(t *testing.T) {
	list := []interface{}{
		map[string]interface{}{"key": "a", "value": 1},
		map[string]interface{}{"key": "b", "value": 2},
	}
	if !CanMatchByIdentifierWithAdditional(list, []string{"key"}) {
		t.Error("expected list to be matchable by custom 'key' field")
	}
}

func TestCanMatchByIdentifierWithAdditional_NoMatchingField(t *testing.T) {
	list := []interface{}{
		map[string]interface{}{"foo": "a"},
		map[string]interface{}{"foo": "b"},
	}
	if CanMatchByIdentifierWithAdditional(list, nil) {
		t.Error("expected list without name/id to NOT be matchable")
	}
}

func TestCanMatchByIdentifierWithAdditional_NonComparableIdentifier(t *testing.T) {
	list := []interface{}{
		map[string]interface{}{"name": []interface{}{"x"}},
	}
	if CanMatchByIdentifierWithAdditional(list, nil) {
		t.Error("expected list with non-comparable identifier to NOT be matchable")
	}
}

func TestCompareK8sDocs_RemovedDocument(t *testing.T) {
	from := `apiVersion: v1
kind: ConfigMap
metadata:
  name: config-a
data:
  key: value-a
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config-b
data:
  key: value-b
`
	to := `apiVersion: v1
kind: ConfigMap
metadata:
  name: config-a
data:
  key: value-a
`
	opts := &Options{DetectKubernetes: true}
	diffs, err := Compare([]byte(from), []byte(to), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hasRemoved := false
	for _, d := range diffs {
		if d.Type == DiffRemoved {
			hasRemoved = true
			break
		}
	}
	if !hasRemoved {
		t.Error("expected DiffRemoved for config-b document")
	}
}

func TestCompareK8sDocs_AddedDocument(t *testing.T) {
	from := `apiVersion: v1
kind: ConfigMap
metadata:
  name: config-a
data:
  key: value-a
`
	to := `apiVersion: v1
kind: ConfigMap
metadata:
  name: config-a
data:
  key: value-a
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config-b
data:
  key: value-b
`
	opts := &Options{DetectKubernetes: true}
	diffs, err := Compare([]byte(from), []byte(to), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hasAdded := false
	for _, d := range diffs {
		if d.Type == DiffAdded {
			hasAdded = true
			break
		}
	}
	if !hasAdded {
		t.Error("expected DiffAdded for config-b document")
	}
}

func TestMatchK8sDocuments_EmptyIdentifier(t *testing.T) {
	// Non-K8s docs (no kind/apiVersion) should all end up unmatched
	fromDocs := []interface{}{
		map[string]interface{}{"foo": "bar"},
		map[string]interface{}{"baz": "qux"},
	}
	toDocs := []interface{}{
		map[string]interface{}{"hello": "world"},
	}

	matched, unmatchedFrom, unmatchedTo := matchK8sDocuments(fromDocs, toDocs, &Options{})

	if len(matched) != 0 {
		t.Errorf("expected no matches, got %d", len(matched))
	}
	if len(unmatchedFrom) != 2 {
		t.Errorf("expected 2 unmatched from, got %d", len(unmatchedFrom))
	}
	if len(unmatchedTo) != 1 {
		t.Errorf("expected 1 unmatched to, got %d", len(unmatchedTo))
	}
}

func TestMatchK8sDocuments_PartialMatch(t *testing.T) {
	mkDoc := func(name string) map[string]interface{} {
		return map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata":   map[string]interface{}{"name": name},
		}
	}

	fromDocs := []interface{}{mkDoc("shared"), mkDoc("only-from")}
	toDocs := []interface{}{mkDoc("only-to"), mkDoc("shared")}

	matched, unmatchedFrom, unmatchedTo := matchK8sDocuments(fromDocs, toDocs, &Options{})

	if len(matched) != 1 {
		t.Errorf("expected 1 match, got %d", len(matched))
	}
	if len(unmatchedFrom) != 1 {
		t.Errorf("expected 1 unmatched from, got %d", len(unmatchedFrom))
	}
	if len(unmatchedTo) != 1 {
		t.Errorf("expected 1 unmatched to, got %d", len(unmatchedTo))
	}
}

func TestIsKubernetesResource_OrderedMap(t *testing.T) {
	meta := NewOrderedMap()
	meta.Keys = append(meta.Keys, "name")
	meta.Values["name"] = "test-resource"

	doc := NewOrderedMap()
	doc.Keys = append(doc.Keys, "apiVersion", "kind", "metadata")
	doc.Values["apiVersion"] = "v1"
	doc.Values["kind"] = "ConfigMap"
	doc.Values["metadata"] = meta

	if !IsKubernetesResource(doc) {
		t.Error("expected OrderedMap-based K8s resource to be detected")
	}
}

func TestGetK8sResourceIdentifier_OrderedMap(t *testing.T) {
	meta := NewOrderedMap()
	meta.Keys = append(meta.Keys, "name", "namespace")
	meta.Values["name"] = "my-app"
	meta.Values["namespace"] = "production"

	doc := NewOrderedMap()
	doc.Keys = append(doc.Keys, "apiVersion", "kind", "metadata")
	doc.Values["apiVersion"] = "apps/v1"
	doc.Values["kind"] = "Deployment"
	doc.Values["metadata"] = meta

	id := GetK8sResourceIdentifier(doc, false)
	expected := "apps/v1:Deployment:production/my-app"
	if id != expected {
		t.Errorf("expected %q, got %q", expected, id)
	}
}

func TestIsKubernetesResource_NonStringApiVersion(t *testing.T) {
	doc := map[string]interface{}{
		"apiVersion": 123,
		"kind":       "ConfigMap",
		"metadata":   map[string]interface{}{"name": "test"},
	}
	if IsKubernetesResource(doc) {
		t.Error("expected false for non-string apiVersion")
	}
}

func TestIsKubernetesResource_NonStringKind(t *testing.T) {
	doc := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       42,
		"metadata":   map[string]interface{}{"name": "test"},
	}
	if IsKubernetesResource(doc) {
		t.Error("expected false for non-string kind")
	}
}

func TestGetIdentifierWithAdditional_FallbackToId(t *testing.T) {
	m := map[string]interface{}{
		"id":  "my-id",
		"foo": "bar",
	}
	id := GetIdentifierWithAdditional(m, nil)
	if id != "my-id" {
		t.Errorf("expected 'my-id', got %v", id)
	}
}

func TestCanMatchByIdentifierWithAdditional_OrderedMapWithId(t *testing.T) {
	om := NewOrderedMap()
	om.Keys = append(om.Keys, "id")
	om.Values["id"] = "item-1"

	list := []interface{}{om}
	if !CanMatchByIdentifierWithAdditional(list, nil) {
		t.Error("expected OrderedMap with 'id' field to be matchable")
	}
}

func TestGetK8sResourceIdentifier_GenerateName(t *testing.T) {
	doc := map[string]interface{}{
		"apiVersion": "batch/v1",
		"kind":       "Job",
		"metadata": map[string]interface{}{
			"generateName": "my-job-",
		},
	}
	id := GetK8sResourceIdentifier(doc, false)
	expected := "batch/v1:Job:my-job-"
	if id != expected {
		t.Errorf("expected identifier %q, got %q", expected, id)
	}
}

func TestGetK8sResourceIdentifier_NameOverGenerateName(t *testing.T) {
	doc := map[string]interface{}{
		"apiVersion": "batch/v1",
		"kind":       "Job",
		"metadata": map[string]interface{}{
			"name":         "my-job-abc123",
			"generateName": "my-job-",
		},
	}
	id := GetK8sResourceIdentifier(doc, false)
	expected := "batch/v1:Job:my-job-abc123"
	if id != expected {
		t.Errorf("expected name to take priority, got %q", id)
	}
}

func TestMatchK8sDocuments_GenerateName(t *testing.T) {
	mkDoc := func(genName string) map[string]interface{} {
		return map[string]interface{}{
			"apiVersion": "batch/v1",
			"kind":       "Job",
			"metadata":   map[string]interface{}{"generateName": genName},
		}
	}

	fromDocs := []interface{}{mkDoc("job-a-"), mkDoc("job-b-")}
	toDocs := []interface{}{mkDoc("job-b-"), mkDoc("job-a-")}

	matched, unmatchedFrom, unmatchedTo := matchK8sDocuments(fromDocs, toDocs, &Options{})

	if len(matched) != 2 {
		t.Errorf("expected 2 matches, got %d", len(matched))
	}
	if len(unmatchedFrom) != 0 {
		t.Errorf("expected 0 unmatched from, got %d", len(unmatchedFrom))
	}
	if len(unmatchedTo) != 0 {
		t.Errorf("expected 0 unmatched to, got %d", len(unmatchedTo))
	}
	// job-a- is at index 0 in from and index 1 in to
	if matched[0] != 1 {
		t.Errorf("expected from[0] to match to[1], got to[%d]", matched[0])
	}
	// job-b- is at index 1 in from and index 0 in to
	if matched[1] != 0 {
		t.Errorf("expected from[1] to match to[0], got to[%d]", matched[1])
	}
}

func TestMatchK8sDocuments_FirstOccurrenceWins(t *testing.T) {
	// When two "to" documents produce the same identifier, the first one should be used
	mkDoc := func(name, data string) map[string]interface{} {
		return map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata":   map[string]interface{}{"name": name},
			"data":       data,
		}
	}

	fromDocs := []interface{}{mkDoc("dup", "from-data")}
	// Two "to" documents with the same identifier
	toDocs := []interface{}{mkDoc("dup", "first"), mkDoc("dup", "second")}

	matched, unmatchedFrom, unmatchedTo := matchK8sDocuments(fromDocs, toDocs, &Options{})

	if len(matched) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matched))
	}
	// from[0] should match to[0] (first occurrence), not to[1]
	if matched[0] != 0 {
		t.Errorf("expected from[0] to match to[0] (first occurrence), got to[%d]", matched[0])
	}
	if len(unmatchedFrom) != 0 {
		t.Errorf("expected 0 unmatched from, got %d", len(unmatchedFrom))
	}
	// to[1] should be unmatched
	if len(unmatchedTo) != 1 || unmatchedTo[0] != 1 {
		t.Errorf("expected to[1] unmatched, got unmatchedTo=%v", unmatchedTo)
	}
}

func TestMatchK8sDocuments_AgnosticMatching(t *testing.T) {
	// Two documents with same kind/name but different apiVersions should match
	// when IgnoreApiVersion=true
	fromDoc := map[string]interface{}{
		"apiVersion": "apps/v1beta1",
		"kind":       "Deployment",
		"metadata":   map[string]interface{}{"name": "my-app", "namespace": "default"},
	}
	toDoc := map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata":   map[string]interface{}{"name": "my-app", "namespace": "default"},
	}

	matched, unmatchedFrom, unmatchedTo := matchK8sDocuments(
		[]interface{}{fromDoc}, []interface{}{toDoc},
		&Options{IgnoreApiVersion: true},
	)

	if len(matched) != 1 {
		t.Fatalf("expected 1 match with agnostic matching, got %d", len(matched))
	}
	if matched[0] != 0 {
		t.Errorf("expected from[0] to match to[0], got to[%d]", matched[0])
	}
	if len(unmatchedFrom) != 0 {
		t.Errorf("expected 0 unmatched from, got %d", len(unmatchedFrom))
	}
	if len(unmatchedTo) != 0 {
		t.Errorf("expected 0 unmatched to, got %d", len(unmatchedTo))
	}
}

func TestMatchK8sDocuments_AgnosticDuplicateFirstWins(t *testing.T) {
	// When agnostic matching produces duplicate identifiers in "to",
	// first-occurrence-wins
	mkDoc := func(apiVer, name string) map[string]interface{} {
		return map[string]interface{}{
			"apiVersion": apiVer,
			"kind":       "Deployment",
			"metadata":   map[string]interface{}{"name": name, "namespace": "default"},
		}
	}

	fromDocs := []interface{}{mkDoc("apps/v1", "my-app")}
	// Two "to" documents: same kind/name, different apiVersion → same agnostic identifier
	toDocs := []interface{}{mkDoc("apps/v1beta1", "my-app"), mkDoc("apps/v1", "my-app")}

	matched, _, unmatchedTo := matchK8sDocuments(fromDocs, toDocs, &Options{IgnoreApiVersion: true})

	if len(matched) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matched))
	}
	// Should match first occurrence (to[0])
	if matched[0] != 0 {
		t.Errorf("expected from[0] to match to[0] (first occurrence), got to[%d]", matched[0])
	}
	if len(unmatchedTo) != 1 || unmatchedTo[0] != 1 {
		t.Errorf("expected to[1] unmatched, got unmatchedTo=%v", unmatchedTo)
	}
}

func TestMatchK8sDocuments_DefaultNoAgnosticMatch(t *testing.T) {
	// When IgnoreApiVersion=false (default), different apiVersions produce
	// different identifiers → no match
	fromDoc := map[string]interface{}{
		"apiVersion": "apps/v1beta1",
		"kind":       "Deployment",
		"metadata":   map[string]interface{}{"name": "my-app", "namespace": "default"},
	}
	toDoc := map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata":   map[string]interface{}{"name": "my-app", "namespace": "default"},
	}

	matched, unmatchedFrom, unmatchedTo := matchK8sDocuments(
		[]interface{}{fromDoc}, []interface{}{toDoc},
		&Options{IgnoreApiVersion: false},
	)

	if len(matched) != 0 {
		t.Errorf("expected 0 matches with default mode, got %d", len(matched))
	}
	if len(unmatchedFrom) != 1 {
		t.Errorf("expected 1 unmatched from, got %d", len(unmatchedFrom))
	}
	if len(unmatchedTo) != 1 {
		t.Errorf("expected 1 unmatched to, got %d", len(unmatchedTo))
	}
}

func TestCompareK8sDocs_AgnosticMatch_ReportsApiVersionModified(t *testing.T) {
	// Req 1.3: When agnostic matching pairs resources with different apiVersions,
	// apiVersion should appear as a modified field
	fromDoc := map[string]interface{}{
		"apiVersion": "apps/v1beta1",
		"kind":       "Deployment",
		"metadata":   map[string]interface{}{"name": "my-app", "namespace": "default"},
		"spec":       map[string]interface{}{"replicas": 3},
	}
	toDoc := map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata":   map[string]interface{}{"name": "my-app", "namespace": "default"},
		"spec":       map[string]interface{}{"replicas": 5},
	}

	opts := &Options{
		DetectKubernetes: true,
		IgnoreApiVersion: true,
	}
	diffs := compareK8sDocs([]interface{}{fromDoc}, []interface{}{toDoc}, opts)

	hasApiVersionDiff := false
	hasReplicasDiff := false
	for _, d := range diffs {
		if d.Type == DiffModified && d.From == "apps/v1beta1" && d.To == "apps/v1" {
			hasApiVersionDiff = true
		}
		if d.Type == DiffModified && d.From == 3 && d.To == 5 {
			hasReplicasDiff = true
		}
	}
	if !hasApiVersionDiff {
		t.Error("expected apiVersion to be reported as a modified field (apps/v1beta1 -> apps/v1)")
	}
	if !hasReplicasDiff {
		t.Error("expected spec.replicas to be reported as a modified field (3 -> 5)")
	}
}

func TestCompareK8sDocs_AgnosticDuplicates_ReportedAsAddedRemoved(t *testing.T) {
	// Req 3.2: When duplicate agnostic identifiers produce unmatched documents,
	// they should be reported as added or removed
	mkDoc := func(apiVer, name string, replicas int) map[string]interface{} {
		return map[string]interface{}{
			"apiVersion": apiVer,
			"kind":       "Deployment",
			"metadata":   map[string]interface{}{"name": name, "namespace": "default"},
			"spec":       map[string]interface{}{"replicas": replicas},
		}
	}

	// "from" has one my-app, "to" has two my-app with different apiVersions
	// → same agnostic identifier → first-occurrence-wins, second is unmatched (added)
	fromDocs := []interface{}{mkDoc("apps/v1", "my-app", 3)}
	toDocs := []interface{}{mkDoc("apps/v1", "my-app", 3), mkDoc("apps/v1beta1", "my-app", 1)}

	opts := &Options{
		DetectKubernetes: true,
		IgnoreApiVersion: true,
	}
	diffs := compareK8sDocs(fromDocs, toDocs, opts)

	hasAdded := false
	for _, d := range diffs {
		if d.Type == DiffAdded {
			hasAdded = true
		}
	}
	if !hasAdded {
		t.Error("expected unmatched duplicate to be reported as DiffAdded")
	}
}

func TestCompare_K8sMultiDoc_GenerateNameMatch(t *testing.T) {
	from := `---
apiVersion: batch/v1
kind: Job
metadata:
  generateName: hook-a-
spec:
  template:
    spec:
      containers:
      - name: hook
        image: alpine:3.18
---
apiVersion: batch/v1
kind: Job
metadata:
  generateName: hook-b-
spec:
  template:
    spec:
      containers:
      - name: hook
        image: alpine:3.19
`
	to := `---
apiVersion: batch/v1
kind: Job
metadata:
  generateName: hook-b-
spec:
  template:
    spec:
      containers:
      - name: hook
        image: alpine:3.20
---
apiVersion: batch/v1
kind: Job
metadata:
  generateName: hook-a-
spec:
  template:
    spec:
      containers:
      - name: hook
        image: alpine:3.18
`
	opts := &Options{DetectKubernetes: true}
	diffs, err := Compare([]byte(from), []byte(to), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// hook-a- should match across reorder (no diffs)
	// hook-b- should show image change: alpine:3.19 -> alpine:3.20
	hasImageChange := false
	for _, d := range diffs {
		if d.Type == DiffModified && d.From == "alpine:3.19" && d.To == "alpine:3.20" {
			hasImageChange = true
		}
	}
	if !hasImageChange {
		t.Error("expected hook-b- image modification to be detected")
	}
	if len(diffs) != 1 {
		t.Errorf("expected exactly 1 diff (image change), got %d", len(diffs))
	}
}

// Integration tests for --ignore-api-version (Task 3.2)

func TestCompare_IgnoreApiVersion_FieldLevelDiff(t *testing.T) {
	// Req 1.3: Full Compare() with IgnoreApiVersion shows apiVersion as modified
	// field alongside other field-level diffs
	from := `apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: my-app
  namespace: default
spec:
  replicas: 3
`
	to := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  namespace: default
spec:
  replicas: 5
`
	opts := &Options{
		DetectKubernetes: true,
		IgnoreApiVersion: true,
	}
	diffs, err := Compare([]byte(from), []byte(to), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hasApiVersionDiff := false
	hasReplicasDiff := false
	for _, d := range diffs {
		if d.Type == DiffModified && d.From == "apps/v1beta1" && d.To == "apps/v1" {
			hasApiVersionDiff = true
		}
		if d.Type == DiffModified && d.From == 3 && d.To == 5 {
			hasReplicasDiff = true
		}
	}
	if !hasApiVersionDiff {
		t.Error("expected apiVersion to appear as a modified field")
	}
	if !hasReplicasDiff {
		t.Error("expected spec.replicas to appear as a modified field")
	}
	if len(diffs) != 2 {
		t.Errorf("expected exactly 2 diffs (apiVersion + replicas), got %d", len(diffs))
	}
}

func TestCompare_IgnoreApiVersion_WithFilter(t *testing.T) {
	// Req 4.1: --filter combined with --ignore-api-version works correctly
	from := `apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: my-app
  namespace: default
spec:
  replicas: 3
`
	to := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  namespace: default
spec:
  replicas: 5
`
	opts := &Options{
		DetectKubernetes: true,
		IgnoreApiVersion: true,
	}
	diffs, err := Compare([]byte(from), []byte(to), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Filter to only spec.replicas
	filterOpts := &FilterOptions{
		IncludePaths: []string{"spec"},
	}
	filtered, err := FilterDiffsWithRegexp(diffs, filterOpts)
	if err != nil {
		t.Fatalf("filter error: %v", err)
	}

	if len(filtered) != 1 {
		t.Fatalf("expected 1 diff after filter, got %d", len(filtered))
	}
	if filtered[0].From != 3 || filtered[0].To != 5 {
		t.Errorf("expected replicas diff (3->5), got %v->%v", filtered[0].From, filtered[0].To)
	}
}

func TestCompare_IgnoreApiVersion_WithExclude(t *testing.T) {
	// Req 4.1: --exclude combined with --ignore-api-version works correctly
	from := `apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: my-app
  namespace: default
spec:
  replicas: 3
`
	to := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  namespace: default
spec:
  replicas: 5
`
	opts := &Options{
		DetectKubernetes: true,
		IgnoreApiVersion: true,
	}
	diffs, err := Compare([]byte(from), []byte(to), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Exclude apiVersion
	filterOpts := &FilterOptions{
		ExcludePaths: []string{"apiVersion"},
	}
	filtered, err := FilterDiffsWithRegexp(diffs, filterOpts)
	if err != nil {
		t.Fatalf("filter error: %v", err)
	}

	if len(filtered) != 1 {
		t.Fatalf("expected 1 diff after exclude, got %d", len(filtered))
	}
	if filtered[0].From != 3 || filtered[0].To != 5 {
		t.Errorf("expected replicas diff (3->5), got %v->%v", filtered[0].From, filtered[0].To)
	}
}

func TestCompare_IgnoreApiVersion_KubernetesDisabled(t *testing.T) {
	// Req 2.2, 4.4: Flag has no effect when K8s detection is disabled
	from := `apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: my-app
  namespace: default
spec:
  replicas: 3
`
	to := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  namespace: default
spec:
  replicas: 5
`
	// With K8s disabled + IgnoreApiVersion: positional matching is used
	optsNoK8s := &Options{
		DetectKubernetes: false,
		IgnoreApiVersion: true,
	}
	diffsNoK8s, err := Compare([]byte(from), []byte(to), optsNoK8s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Without IgnoreApiVersion (same K8s disabled): should produce identical result
	optsDefault := &Options{
		DetectKubernetes: false,
		IgnoreApiVersion: false,
	}
	diffsDefault, err := Compare([]byte(from), []byte(to), optsDefault)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(diffsNoK8s) != len(diffsDefault) {
		t.Errorf("expected same diff count with K8s disabled, got %d vs %d",
			len(diffsNoK8s), len(diffsDefault))
	}
}

func TestCompare_IgnoreApiVersion_NonK8sUnaffected(t *testing.T) {
	// Req 4.4: Non-K8s documents use positional matching regardless of flag
	from := `name: config-a
value: old
---
name: config-b
value: stable
`
	to := `name: config-b
value: stable
---
name: config-a
value: new
`
	opts := &Options{
		DetectKubernetes: true,
		IgnoreApiVersion: true,
	}
	diffs, err := Compare([]byte(from), []byte(to), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Non-K8s docs → positional matching:
	// doc[0] config-a vs config-b → name change + value change
	// doc[1] config-b vs config-a → name change + value change
	hasNameDiff := false
	for _, d := range diffs {
		if d.Type == DiffModified && d.From == "config-a" && d.To == "config-b" {
			hasNameDiff = true
		}
	}
	if !hasNameDiff {
		t.Error("expected positional matching for non-K8s documents (name diff config-a→config-b)")
	}
}

func TestCompare_IgnoreApiVersion_WithSwap(t *testing.T) {
	// Req 4.2: --swap is applied before apiVersion-agnostic matching
	from := `apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: my-app
  namespace: default
spec:
  replicas: 3
`
	to := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  namespace: default
spec:
  replicas: 5
`
	opts := &Options{
		DetectKubernetes: true,
		IgnoreApiVersion: true,
		Swap:             true,
	}
	diffs, err := Compare([]byte(from), []byte(to), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With swap, the direction is reversed: from=to, to=from
	hasSwappedApiVersion := false
	hasSwappedReplicas := false
	for _, d := range diffs {
		if d.Type == DiffModified && d.From == "apps/v1" && d.To == "apps/v1beta1" {
			hasSwappedApiVersion = true
		}
		if d.Type == DiffModified && d.From == 5 && d.To == 3 {
			hasSwappedReplicas = true
		}
	}
	if !hasSwappedApiVersion {
		t.Error("expected swapped apiVersion diff (apps/v1 -> apps/v1beta1)")
	}
	if !hasSwappedReplicas {
		t.Error("expected swapped replicas diff (5 -> 3)")
	}
}

func TestCompare_IgnoreApiVersion_WithChroot(t *testing.T) {
	// Req 4.3: --chroot is applied before K8s detection and matching
	from := `wrapper:
  apiVersion: apps/v1beta1
  kind: Deployment
  metadata:
    name: my-app
    namespace: default
  spec:
    replicas: 3
`
	to := `wrapper:
  apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: my-app
    namespace: default
  spec:
    replicas: 5
`
	opts := &Options{
		DetectKubernetes: true,
		IgnoreApiVersion: true,
		Chroot:           "wrapper",
	}
	diffs, err := Compare([]byte(from), []byte(to), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// After chroot to "wrapper", the docs are K8s resources
	// They should match via agnostic identifiers
	hasApiVersionDiff := false
	hasReplicasDiff := false
	for _, d := range diffs {
		if d.Type == DiffModified && d.From == "apps/v1beta1" && d.To == "apps/v1" {
			hasApiVersionDiff = true
		}
		if d.Type == DiffModified && d.From == 3 && d.To == 5 {
			hasReplicasDiff = true
		}
	}
	if !hasApiVersionDiff {
		t.Error("expected apiVersion diff after chroot")
	}
	if !hasReplicasDiff {
		t.Error("expected replicas diff after chroot")
	}
}
