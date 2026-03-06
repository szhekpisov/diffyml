package diffyml

import (
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/compare"
)

const (
	renameScoreThreshold = compare.RenameScoreThreshold
	renameLimit          = compare.RenameLimit
)

type similarityIndex = compare.SimilarityIndex

func newSimilarityIndex(data []byte) *similarityIndex {
	return compare.NewSimilarityIndex(data)
}

func serializeDocument(doc interface{}) []byte {
	return compare.SerializeDocument(doc)
}

func detectRenames(from, to []interface{}, unmatchedFrom, unmatchedTo []int, opts *Options) (renameMatched map[int]int, remainingFrom, remainingTo []int) {
	return compare.DetectRenames(from, to, unmatchedFrom, unmatchedTo, opts)
}
