// identifier_node.go - Node-based identifier extraction for list matching.
//
// Mirrors the any-based getIdentifier / canMatchByIdentifier but on raw
// *yaml.Node, so the comparator can decide identifier-based matching without
// materialising every list item. The contract pinned by
// TestGetIdentifierNode_EquivalenceCorpus is:
//
//	getIdentifierNode(n, opts) == getIdentifier(nodeToInterface(n), opts)
//
// for every MappingNode reachable from a parsed (post-resolveMergeKeys) tree.
package diffyml

import (
	"go.yaml.in/yaml/v3"
)

// getIdentifierNode extracts the identifier value from a MappingNode, picking
// the first matching field in order: AdditionalIdentifiers, then "name", then
// "id". Returns nil for non-mapping inputs (matches the any-based path's
// type-assertion-failure semantics). Duplicate keys resolve last-write-wins
// to match nodeToInterface's map-overwrite behaviour.
func getIdentifierNode(n *yaml.Node, opts *Options) any {
	n = resolveAlias(n)
	if n == nil {
		return nil
	}
	if n.Kind != yaml.MappingNode {
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

// canMatchByIdentifierNodes mirrors canMatchByIdentifier for a slice of
// nodes: every item must be a MappingNode (or fail the check), and at least
// one item must yield a usable comparable identifier. The nil and Kind guards
// are kept independent so each mutation target is testable in isolation.
func canMatchByIdentifierNodes(items []*yaml.Node, opts *Options) bool {
	hasIdentifier := false
	for _, item := range items {
		item = resolveAlias(item)
		if item == nil {
			// Cycle-collapsed alias (resolveAlias returns nil) disqualifies
			// outright; nodeToInterface would render it as a nil entry.
			return false
		}
		if item.Kind != yaml.MappingNode {
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
// order occurrence of key — last-write-wins, matching nodeToInterface's map
// behaviour for inputs with duplicate keys (e.g. explicit "name" after a
// merge anchor that already introduced one).
func lookupMappingValueNode(n *yaml.Node, key string) *yaml.Node {
	var found *yaml.Node
	for i := 0; i+1 < len(n.Content); i += 2 {
		if n.Content[i].Value == key {
			found = n.Content[i+1]
		}
	}
	return found
}

// materializeIdentifierValue produces the same Go value as nodeToInterface,
// but short-circuits through resolveAlias + resolveScalar for the common
// scalar-identifier case ("name: alice", "id: 42") to skip the cycle-
// tracking allocation. Composite identifiers fall through to the full walk.
func materializeIdentifierValue(n *yaml.Node) any {
	resolved := resolveAlias(n)
	if resolved == nil {
		return nil
	}
	if resolved.Kind == yaml.ScalarNode {
		return resolveScalar(resolved)
	}
	return nodeToInterface(resolved)
}
