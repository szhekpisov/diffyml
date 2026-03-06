package diffyml

import (
	"cmp"
	"hash/crc32"
	"slices"

	"gopkg.in/yaml.v3"
)

const (
	renameScoreThreshold = 60 // Minimum similarity % for rename match
	renameLimit          = 50 // Max unmatched docs before skipping detection
)

// similarityIndex hashes lines of text for content similarity comparison.
// Uses CRC32 on each line, storing counts in a table.
type similarityIndex struct {
	hashes   map[uint32]int
	numLines int
	numBytes int
}

// newSimilarityIndex builds a similarity index from raw bytes by hashing each non-empty line.
func newSimilarityIndex(data []byte) *similarityIndex {
	idx := &similarityIndex{
		hashes:   make(map[uint32]int),
		numBytes: len(data),
	}

	start := 0
	for i := 0; i <= len(data); i++ {
		if i == len(data) || data[i] == '\n' {
			line := data[start:i]
			start = i + 1

			// Skip whitespace-only lines
			hasContent := false
			for _, b := range line {
				if b != ' ' && b != '\t' && b != '\r' {
					hasContent = true
					break
				}
			}
			if !hasContent {
				continue
			}

			idx.hashes[crc32.ChecksumIEEE(line)]++
			idx.numLines++
		}
	}

	return idx
}

// score computes similarity score (0–100) between two indices.
func (s *similarityIndex) score(other *similarityIndex) int {
	maxLines := max(s.numLines, other.numLines)
	if maxLines == 0 {
		return 0
	}

	matching := 0
	for h, count := range other.hashes {
		if selfCount, ok := s.hashes[h]; ok {
			matching += min(selfCount, count)
		}
	}

	return matching * 100 / maxLines
}

// serializeDocument converts a parsed YAML document to YAML bytes for similarity comparison.
// valueToYAMLNode always produces a valid *yaml.Node, so yaml.Marshal cannot fail here.
func serializeDocument(doc any) []byte {
	node := valueToYAMLNode(doc)
	data, _ := yaml.Marshal(node)
	return data
}

// renamePair holds a scored rename candidate pair.
type renamePair struct {
	fromIdx int
	toIdx   int
	score   int
}

// detectRenames finds renamed documents among unmatched K8s resources.
func detectRenames(from, to []any, unmatchedFrom, unmatchedTo []int, opts *Options) (renameMatched map[int]int, remainingFrom, remainingTo []int) {
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
	maxCandidates := max(len(k8sFrom), len(k8sTo))
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
	fromCandidates := make(map[int]*similarityIndex)
	toCandidates := make(map[int]*similarityIndex)

	for _, idx := range k8sFrom {
		fromCandidates[idx] = newSimilarityIndex(serializeDocument(from[idx]))
	}

	for _, idx := range k8sTo {
		toCandidates[idx] = newSimilarityIndex(serializeDocument(to[idx]))
	}

	// Build scored pairs with size-ratio early rejection
	var pairs []renamePair
	for _, fromIdx := range k8sFrom {
		fc := fromCandidates[fromIdx]
		for _, toIdx := range k8sTo {
			tc := toCandidates[toIdx]

			// Size ratio early rejection
			minLen := min(fc.numBytes, tc.numBytes)
			maxLen := max(fc.numBytes, tc.numBytes)
			if maxLen != 0 && minLen*100/maxLen < renameScoreThreshold {
				continue
			}

			s := fc.score(tc)
			if s >= renameScoreThreshold {
				pairs = append(pairs, renamePair{fromIdx: fromIdx, toIdx: toIdx, score: s})
			}
		}
	}

	// Sort descending by score, tiebreak by ascending fromIdx then toIdx
	slices.SortStableFunc(pairs, func(a, b renamePair) int {
		return cmp.Or(
			cmp.Compare(b.score, a.score),     // descending score
			cmp.Compare(a.fromIdx, b.fromIdx), // ascending fromIdx
			cmp.Compare(a.toIdx, b.toIdx),     // ascending toIdx
		)
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
	for _, idx := range k8sFrom {
		if !assignedFrom[idx] {
			remainingFrom = append(remainingFrom, idx)
		}
	}
	for _, idx := range k8sTo {
		if !assignedTo[idx] {
			remainingTo = append(remainingTo, idx)
		}
	}

	return renameMatched, remainingFrom, remainingTo
}
