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

// resolveMergeKeys rewrites every YAML merge key under n in-place. Safe to
// call on any Kind (non-containers are no-ops). The cycle-seen map breaks
// recursion on pathological self-referential anchors like `&a {<<: *a}`.
func resolveMergeKeys(n *yaml.Node) {
	resolveMergeKeysWithCycles(n, make(map[*yaml.Node]bool))
}

func resolveMergeKeysWithCycles(n *yaml.Node, cycles map[*yaml.Node]bool) {
	if n == nil {
		return
	}
	switch n.Kind {
	case yaml.DocumentNode, yaml.SequenceNode:
		for _, c := range n.Content {
			resolveMergeKeysWithCycles(c, cycles)
		}
	case yaml.MappingNode:
		resolveMappingMergeKeys(n, cycles)
	}
}

// resolveMappingMergeKeys rewrites n.Content to inline any "<<: *anchor"
// entries. seen-set membership is host-mapping-local (matching the host's
// om.Values precedence). cycles tracks MappingNodes currently mid-resolution
// so a self-referential anchor terminates instead of recursing forever.
func resolveMappingMergeKeys(n *yaml.Node, cycles map[*yaml.Node]bool) {
	if cycles[n] {
		// Already resolving this mapping — a deeper "<<" pointing back to it
		// must not recurse. Matches nodeToInterfaceImpl's cycle break which
		// returns nil for the alias and contributes no merged keys.
		return
	}
	cycles[n] = true
	defer delete(cycles, n)

	seen := make(map[string]bool)
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
			if cycles[source] {
				// Self/back reference: skip the merge to terminate the cycle.
				continue
			}
			// Flatten nested merges in the source first so its Content holds
			// only direct pairs. Idempotent: second reference is a no-op.
			// Without this call, a source only reachable via alias (and thus
			// skipped by the outer recursion) would leak its own "<<" entries
			// into the host.
			resolveMappingMergeKeys(source, cycles)
			for j := 0; j+1 < len(source.Content); j += 2 {
				mk := source.Content[j]
				if seen[mk.Value] {
					continue
				}
				seen[mk.Value] = true
				newContent = append(newContent, mk, source.Content[j+1])
			}
			continue
		}

		// Explicit pair: recurse into the value to resolve any merges nested
		// inside it (mapping values, sequence items containing merges, etc.),
		// then preserve the pair in source order.
		resolveMergeKeysWithCycles(valNode, cycles)
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
