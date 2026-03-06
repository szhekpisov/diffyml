// Package diffyml provides YAML diff functionality for comparing YAML documents.
package diffyml

import (
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/compare"
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/types"
)

// Type aliases for public API
type YAMLValue = types.YAMLValue
type YAMLKind = types.YAMLKind
type DiffType = types.DiffType
type Difference = types.Difference
type DiffGroup = types.DiffGroup
type Options = types.Options

// YAMLKind constants
const (
	KindNull      = types.KindNull
	KindString    = types.KindString
	KindInt       = types.KindInt
	KindFloat     = types.KindFloat
	KindBool      = types.KindBool
	KindTimestamp = types.KindTimestamp
	KindMap       = types.KindMap
	KindList      = types.KindList
	KindUnknown   = types.KindUnknown
)

// DiffType constants
const (
	DiffAdded        = types.DiffAdded
	DiffRemoved      = types.DiffRemoved
	DiffModified     = types.DiffModified
	DiffOrderChanged = types.DiffOrderChanged
)

// Compare compares two YAML documents and returns the differences.
func Compare(from, to []byte, opts *Options) ([]Difference, error) {
	return compare.Compare(from, to, opts)
}
