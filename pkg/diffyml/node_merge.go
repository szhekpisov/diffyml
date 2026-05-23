// node_merge.go - YAML merge-key resolution at parse time.
//
// YAML merge keys ("<<: *anchor") let a mapping inherit entries from another
// mapping. Legacy nodeToInterfaceImpl resolved them on the fly while
// converting nodes to any. With the node pipeline carrying *yaml.Node trees
// through Compare/extractPathOrder/chroot/compareNodes, every downstream
// walker would have to special-case "<<" to remain semantically equivalent.
//
// Instead, resolveMergeKeys rewrites each MappingNode's Content slice once,
// at parse time, replacing "<<" entries with the synthesized key/value pairs
// from the merged map. After resolution no "<<" keys remain in the tree and
// downstream walkers see a flat, regular mapping.
//
// Semantics match nodeToInterfaceImpl byte-for-byte:
//   - Keys already present in the host mapping (from explicit pairs encountered
//     earlier in source order, or from earlier merges) take precedence over
//     later "<<" expansions of the same key.
//   - Merge sources that resolve to anything other than a MappingNode (missing
//     anchor, scalar/sequence target) are silently dropped.
//   - Nested merges in the merge source are flattened first, so the same source
//     referenced from multiple "<<" sites stays idempotent.
//   - Alias cycles terminate at a nil target rather than recursing forever.
package diffyml

import (
	"go.yaml.in/yaml/v3"
)

// resolveMergeKeys walks a parsed yaml.Node tree (a DocumentNode or any
// content) and rewrites every YAML merge key in-place. Safe to call on any
// Kind; non-container nodes are no-ops.
func resolveMergeKeys(n *yaml.Node) {
	if n == nil {
		return
	}
	switch n.Kind {
	case yaml.DocumentNode, yaml.SequenceNode:
		for _, c := range n.Content {
			resolveMergeKeys(c)
		}
	case yaml.MappingNode:
		resolveMappingMergeKeys(n)
	}
}

// resolveMappingMergeKeys rewrites n.Content to inline any "<<: *anchor"
// entries. seen-set membership is host-mapping-local (matching nodeToInterface
// where the precedence check is against om.Values of the host).
func resolveMappingMergeKeys(n *yaml.Node) {
	seen := make(map[string]bool, len(n.Content)/2)
	newContent := make([]*yaml.Node, 0, len(n.Content))

	for i := 0; i+1 < len(n.Content); i += 2 {
		keyNode := n.Content[i]
		valNode := n.Content[i+1]

		if keyNode.Value == "<<" {
			source := resolveAlias(valNode)
			// Silently drop sources that resolve to anything other than a
			// MappingNode — matches the legacy `merged.(*OrderedMap)` assertion
			// failing and the merge being skipped.
			if source == nil || source.Kind != yaml.MappingNode {
				continue
			}
			// Flatten nested merges in the source first so its Content holds
			// only direct pairs. Idempotent: second reference is a no-op.
			resolveMappingMergeKeys(source)
			for j := 0; j+1 < len(source.Content); j += 2 {
				mk := source.Content[j]
				mv := source.Content[j+1]
				if seen[mk.Value] {
					continue
				}
				seen[mk.Value] = true
				newContent = append(newContent, mk, mv)
			}
			continue
		}

		// Explicit pair: recurse into the value to resolve any merges nested
		// inside it (mapping values, sequence items containing merges, etc.),
		// then preserve the pair in source order.
		resolveMergeKeys(valNode)
		seen[keyNode.Value] = true
		newContent = append(newContent, keyNode, valNode)
	}

	n.Content = newContent
}

// resolveAlias follows an AliasNode chain to its target, returning nil if the
// chain is broken (missing anchor) or self-referential. Non-alias inputs are
// returned unchanged.
func resolveAlias(n *yaml.Node) *yaml.Node {
	seen := make(map[*yaml.Node]bool)
	for n != nil && n.Kind == yaml.AliasNode {
		if seen[n] {
			return nil
		}
		seen[n] = true
		n = n.Alias
	}
	return n
}
