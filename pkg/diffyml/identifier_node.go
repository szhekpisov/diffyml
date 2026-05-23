// identifier_node.go - Node-based identifier extraction for list matching.
//
// The comparator matches list items by an identifier field ("name"/"id" by
// default, configurable via Options.AdditionalIdentifiers) so that re-ordered
// or partially-modified lists diff cleanly. The legacy any-based path goes
// through getIdentifier -> getIdentifierFromOrderedMap / IdentifierWithAdditional
// (which require materialized *OrderedMap or map[string]any).
//
// getIdentifierNode does the same job on a raw *yaml.Node, so the comparator
// can decide identifier-based matching without materializing every list item
// up front. The contract pinned by TestGetIdentifierNode_EquivalenceCorpus:
//
//   getIdentifierNode(n, opts) == getIdentifier(nodeToInterface(n), opts)
//
// for every MappingNode reachable from a parsed (post-resolveMergeKeys) tree.
// canMatchByIdentifierNodes mirrors canMatchByIdentifier under the same
// contract on []*yaml.Node.
package diffyml

import (
	"go.yaml.in/yaml/v3"
)

// getIdentifierNode extracts the identifier value from a MappingNode. Returns
// nil for non-mapping inputs (matching the any-based getIdentifier's
// type-assertion-failure path). The lookup order — additional identifiers (in
// configured order), then "name", then "id" — matches IdentifierWithAdditional
// /getIdentifierFromOrderedMap exactly. For duplicate keys within a mapping
// (possible via the legacy explicit-key-after-merge quirk preserved by
// resolveMergeKeys), the last source-order occurrence wins, matching
// nodeToInterface's map-overwrite semantics.
func getIdentifierNode(n *yaml.Node, opts *Options) any {
	if n == nil {
		return nil
	}
	n = resolveAlias(n)
	if n == nil || n.Kind != yaml.MappingNode {
		return nil
	}

	var additional []string
	if opts != nil {
		additional = opts.AdditionalIdentifiers
	}

	for _, field := range additional {
		if v := lookupMappingValueNode(n, field); v != nil {
			return materializeIdentifierValue(v)
		}
	}
	if v := lookupMappingValueNode(n, "name"); v != nil {
		return materializeIdentifierValue(v)
	}
	if v := lookupMappingValueNode(n, "id"); v != nil {
		return materializeIdentifierValue(v)
	}
	return nil
}

// canMatchByIdentifierNodes mirrors canMatchByIdentifier for a slice of nodes:
// every item must be a MappingNode (or fail the check), and at least one item
// must yield a usable comparable identifier.
func canMatchByIdentifierNodes(items []*yaml.Node, opts *Options) bool {
	hasIdentifier := false
	for _, item := range items {
		item = resolveAlias(item)
		if item == nil || item.Kind != yaml.MappingNode {
			// Non-mapping item disqualifies identifier matching outright,
			// matching CanMatchByIdentifierWithAdditional's behavior.
			return false
		}
		id := getIdentifierNode(item, opts)
		if isComparableIdentifier(id) {
			hasIdentifier = true
		}
	}
	return hasIdentifier
}

// lookupMappingValueNode returns the value node paired with the LAST source-
// order occurrence of key in a MappingNode. Matching the last-write-wins of
// nodeToInterface's Values map is important for inputs with duplicate keys
// (e.g. an explicit "name" after a "<<:" merge that already introduced one).
func lookupMappingValueNode(n *yaml.Node, key string) *yaml.Node {
	var found *yaml.Node
	for i := 0; i+1 < len(n.Content); i += 2 {
		if n.Content[i].Value == key {
			found = n.Content[i+1]
		}
	}
	return found
}

// materializeIdentifierValue converts an identifier value node into the Go-
// typed value nodeToInterface would have produced. Scalars take the fast path
// through resolveScalar; non-scalar identifier values (rare) defer to
// nodeToInterface so deep-equality against the legacy materialized form holds.
func materializeIdentifierValue(n *yaml.Node) any {
	n = resolveAlias(n)
	if n == nil {
		return nil
	}
	if n.Kind == yaml.ScalarNode {
		return resolveScalar(n)
	}
	return nodeToInterface(n)
}
