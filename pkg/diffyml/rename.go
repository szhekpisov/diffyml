package diffyml

import (
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"
)

const (
	renameScoreThreshold = 60 // Minimum similarity % for rename match
	renameLimit          = 50 // Max unmatched docs before skipping detection
)

// similarityIndex hashes lines of text for content similarity comparison.
// Uses DJB hash (hash * 33 + byte) on each line, storing counts in a table.
type similarityIndex struct {
	hashes   map[uint32]int
	numLines int
}

// newSimilarityIndex builds a similarity index from raw bytes by hashing each non-empty line.
func newSimilarityIndex(data []byte) *similarityIndex {
	idx := &similarityIndex{
		hashes: make(map[uint32]int),
	}

	start := 0
	for i := 0; i <= len(data); i++ {
		if i == len(data) || data[i] == '\n' {
			line := data[start:i]
			start = i + 1

			// Skip empty/whitespace-only lines
			empty := true
			for _, b := range line {
				if b != ' ' && b != '\t' && b != '\r' {
					empty = false
					break
				}
			}
			if empty {
				continue
			}

			// DJB hash
			var h uint32 = 5381
			for _, b := range line {
				h = h*33 + uint32(b)
			}

			idx.hashes[h]++
			idx.numLines++
		}
	}

	return idx
}

// score computes similarity score (0–100) between two indices.
func (s *similarityIndex) score(other *similarityIndex) int {
	maxLines := s.numLines
	if other.numLines > maxLines {
		maxLines = other.numLines
	}
	if maxLines == 0 {
		return 0
	}

	matching := 0
	for h, count := range other.hashes {
		if selfCount, ok := s.hashes[h]; ok {
			if selfCount < count {
				matching += selfCount
			} else {
				matching += count
			}
		}
	}

	return matching * 100 / maxLines
}

// toYAMLNode converts a parsed YAML value to a yaml.Node tree.
func toYAMLNode(v interface{}) *yaml.Node {
	switch val := v.(type) {
	case *OrderedMap:
		node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		for _, key := range val.Keys {
			keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
			valNode := toYAMLNode(val.Values[key])
			node.Content = append(node.Content, keyNode, valNode)
		}
		return node
	case map[string]interface{}:
		node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, key := range keys {
			keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
			valNode := toYAMLNode(val[key])
			node.Content = append(node.Content, keyNode, valNode)
		}
		return node
	case []interface{}:
		node := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for _, item := range val {
			node.Content = append(node.Content, toYAMLNode(item))
		}
		return node
	case string:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: val}
	case int:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: fmt.Sprintf("%d", val)}
	case float64:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!float", Value: fmt.Sprintf("%g", val)}
	case bool:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: fmt.Sprintf("%t", val)}
	case nil:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}
	default:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: fmt.Sprintf("%v", val)}
	}
}

// serializeDocument converts a parsed YAML document to YAML bytes for similarity comparison.
func serializeDocument(doc interface{}) ([]byte, error) {
	node := toYAMLNode(doc)
	return yaml.Marshal(node)
}

// renamePair holds a scored rename candidate pair.
type renamePair struct {
	fromIdx int
	toIdx   int
	score   int
}

// detectRenames finds renamed documents among unmatched K8s resources.
func detectRenames(from, to []interface{}, unmatchedFrom, unmatchedTo []int, opts *Options) (renameMatched map[int]int, remainingFrom, remainingTo []int) {
	renameMatched = make(map[int]int)

	// Early return if disabled or either list is empty
	if !opts.DetectRenames || len(unmatchedFrom) == 0 || len(unmatchedTo) == 0 {
		return renameMatched, unmatchedFrom, unmatchedTo
	}

	// Filter to K8s documents only; non-K8s pass straight through to remaining
	var k8sFrom, k8sTo []int
	for _, idx := range unmatchedFrom {
		if from[idx] != nil && IsKubernetesResource(from[idx]) {
			k8sFrom = append(k8sFrom, idx)
		} else {
			remainingFrom = append(remainingFrom, idx)
		}
	}
	for _, idx := range unmatchedTo {
		if to[idx] != nil && IsKubernetesResource(to[idx]) {
			k8sTo = append(k8sTo, idx)
		} else {
			remainingTo = append(remainingTo, idx)
		}
	}

	// Check rename limit
	maxCandidates := len(k8sFrom)
	if len(k8sTo) > maxCandidates {
		maxCandidates = len(k8sTo)
	}
	if maxCandidates > renameLimit {
		remainingFrom = append(remainingFrom, k8sFrom...)
		remainingTo = append(remainingTo, k8sTo...)
		return renameMatched, remainingFrom, remainingTo
	}

	// If either K8s list is empty after filtering, no renames possible
	if len(k8sFrom) == 0 || len(k8sTo) == 0 {
		remainingFrom = append(remainingFrom, k8sFrom...)
		remainingTo = append(remainingTo, k8sTo...)
		return renameMatched, remainingFrom, remainingTo
	}

	// Serialize candidates and build similarity indices
	type candidateInfo struct {
		idx     *similarityIndex
		byteLen int
	}

	fromCandidates := make(map[int]*candidateInfo)
	toCandidates := make(map[int]*candidateInfo)

	var validK8sFrom []int
	for _, idx := range k8sFrom {
		data, err := serializeDocument(from[idx])
		if err != nil {
			remainingFrom = append(remainingFrom, idx)
			continue
		}
		fromCandidates[idx] = &candidateInfo{
			idx:     newSimilarityIndex(data),
			byteLen: len(data),
		}
		validK8sFrom = append(validK8sFrom, idx)
	}

	var validK8sTo []int
	for _, idx := range k8sTo {
		data, err := serializeDocument(to[idx])
		if err != nil {
			remainingTo = append(remainingTo, idx)
			continue
		}
		toCandidates[idx] = &candidateInfo{
			idx:     newSimilarityIndex(data),
			byteLen: len(data),
		}
		validK8sTo = append(validK8sTo, idx)
	}

	// Build scored pairs with size-ratio early rejection
	var pairs []renamePair
	for _, fromIdx := range validK8sFrom {
		fc := fromCandidates[fromIdx]
		for _, toIdx := range validK8sTo {
			tc := toCandidates[toIdx]

			// Size ratio early rejection
			minLen := fc.byteLen
			maxLen := tc.byteLen
			if minLen > maxLen {
				minLen, maxLen = maxLen, minLen
			}
			if maxLen > 0 && minLen*100/maxLen < renameScoreThreshold {
				continue
			}

			s := fc.idx.score(tc.idx)
			if s >= renameScoreThreshold {
				pairs = append(pairs, renamePair{fromIdx: fromIdx, toIdx: toIdx, score: s})
			}
		}
	}

	// Sort descending by score, tiebreak by ascending fromIdx then toIdx
	sort.SliceStable(pairs, func(i, j int) bool {
		if pairs[i].score != pairs[j].score {
			return pairs[i].score > pairs[j].score
		}
		if pairs[i].fromIdx != pairs[j].fromIdx {
			return pairs[i].fromIdx < pairs[j].fromIdx
		}
		return pairs[i].toIdx < pairs[j].toIdx
	})

	// Greedy assignment
	assignedFrom := make(map[int]bool)
	assignedTo := make(map[int]bool)
	for _, pair := range pairs {
		if assignedFrom[pair.fromIdx] || assignedTo[pair.toIdx] {
			continue
		}
		renameMatched[pair.fromIdx] = pair.toIdx
		assignedFrom[pair.fromIdx] = true
		assignedTo[pair.toIdx] = true
	}

	// Remaining = non-K8s passthrough (already added) + unassigned K8s candidates
	for _, idx := range validK8sFrom {
		if !assignedFrom[idx] {
			remainingFrom = append(remainingFrom, idx)
		}
	}
	for _, idx := range validK8sTo {
		if !assignedTo[idx] {
			remainingTo = append(remainingTo, idx)
		}
	}

	return renameMatched, remainingFrom, remainingTo
}
