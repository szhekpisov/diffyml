package compare

import (
	"cmp"
	"hash/crc32"
	"slices"

	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/parse"
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/types"
	"gopkg.in/yaml.v3"
)

const (
	RenameScoreThreshold = 60 // Minimum similarity % for rename match
	RenameLimit          = 50 // Max unmatched docs before skipping detection
)

// SimilarityIndex hashes lines of text for content similarity comparison.
// Uses CRC32 on each line, storing counts in a table.
type SimilarityIndex struct {
	Hashes   map[uint32]int
	NumLines int
	NumBytes int
}

// NewSimilarityIndex builds a similarity index from raw bytes by hashing each non-empty line.
func NewSimilarityIndex(data []byte) *SimilarityIndex {
	idx := &SimilarityIndex{
		Hashes:   make(map[uint32]int),
		NumBytes: len(data),
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

			idx.Hashes[crc32.ChecksumIEEE(line)]++
			idx.NumLines++
		}
	}

	return idx
}

// Score computes similarity score (0–100) between two indices.
func (s *SimilarityIndex) Score(other *SimilarityIndex) int {
	maxLines := max(s.NumLines, other.NumLines)
	if maxLines == 0 {
		return 0
	}

	matching := 0
	for h, count := range other.Hashes {
		if selfCount, ok := s.Hashes[h]; ok {
			matching += min(selfCount, count)
		}
	}

	return matching * 100 / maxLines
}

// SerializeDocument converts a parsed YAML document to YAML bytes for similarity comparison.
// parse.ValueToYAMLNode always produces a valid *yaml.Node, so yaml.Marshal cannot fail here.
func SerializeDocument(doc interface{}) []byte {
	node := parse.ValueToYAMLNode(doc)
	data, _ := yaml.Marshal(node)
	return data
}

// RenamePair holds a scored rename candidate pair.
type RenamePair struct {
	FromIdx int
	ToIdx   int
	Score   int
}

// DetectRenames finds renamed documents among unmatched K8s resources.
func DetectRenames(from, to []interface{}, unmatchedFrom, unmatchedTo []int, opts *types.Options) (renameMatched map[int]int, remainingFrom, remainingTo []int) {
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
	if maxCandidates > RenameLimit {
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
	fromCandidates := make(map[int]*SimilarityIndex)
	toCandidates := make(map[int]*SimilarityIndex)

	for _, idx := range k8sFrom {
		fromCandidates[idx] = NewSimilarityIndex(SerializeDocument(from[idx]))
	}

	for _, idx := range k8sTo {
		toCandidates[idx] = NewSimilarityIndex(SerializeDocument(to[idx]))
	}

	// Build scored pairs with size-ratio early rejection
	var pairs []RenamePair
	for _, fromIdx := range k8sFrom {
		fc := fromCandidates[fromIdx]
		for _, toIdx := range k8sTo {
			tc := toCandidates[toIdx]

			// Size ratio early rejection
			minLen := min(fc.NumBytes, tc.NumBytes)
			maxLen := max(fc.NumBytes, tc.NumBytes)
			if maxLen != 0 && minLen*100/maxLen < RenameScoreThreshold {
				continue
			}

			s := fc.Score(tc)
			if s >= RenameScoreThreshold {
				pairs = append(pairs, RenamePair{FromIdx: fromIdx, ToIdx: toIdx, Score: s})
			}
		}
	}

	// Sort descending by score, tiebreak by ascending fromIdx then toIdx
	slices.SortStableFunc(pairs, func(a, b RenamePair) int {
		return cmp.Or(
			cmp.Compare(b.Score, a.Score),     // descending score
			cmp.Compare(a.FromIdx, b.FromIdx), // ascending fromIdx
			cmp.Compare(a.ToIdx, b.ToIdx),     // ascending toIdx
		)
	})

	// Greedy assignment
	assignedFrom := make(map[int]bool)
	assignedTo := make(map[int]bool)
	for _, pair := range pairs {
		if assignedFrom[pair.FromIdx] || assignedTo[pair.ToIdx] {
			continue
		}
		renameMatched[pair.FromIdx] = pair.ToIdx
		assignedFrom[pair.FromIdx] = true
		assignedTo[pair.ToIdx] = true
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
