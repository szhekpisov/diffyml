package types

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"
)

// YAMLValue is an alias for interface{} that documents the expected types
// from YAML parsing: string, int, int64, float64, bool, time.Time,
// *OrderedMap, []interface{}, or nil.
type YAMLValue = interface{}

// YAMLKind classifies a parsed YAML value into its runtime type category.
type YAMLKind int

const (
	KindNull      YAMLKind = iota
	KindString
	KindInt
	KindFloat
	KindBool
	KindTimestamp
	KindMap
	KindList
	KindUnknown
)

// YamlKindOf returns the YAMLKind for a parsed YAML value.
func YamlKindOf(v YAMLValue) YAMLKind {
	switch v.(type) {
	case nil:
		return KindNull
	case string:
		return KindString
	case int, int64:
		return KindInt
	case float64:
		return KindFloat
	case bool:
		return KindBool
	case time.Time:
		return KindTimestamp
	case *OrderedMap:
		return KindMap
	case []interface{}:
		return KindList
	default:
		if ToOrderedMap(v) != nil {
			return KindMap
		}
		return KindUnknown
	}
}

// DiffType represents the kind of difference detected between YAML documents.
type DiffType int

const (
	DiffAdded DiffType = iota
	DiffRemoved
	DiffModified
	DiffOrderChanged
)

// Difference represents a single change between two YAML documents.
type Difference struct {
	Path          string
	Type          DiffType
	From          YAMLValue
	To            YAMLValue
	DocumentIndex int
}

// DiffGroup pairs differences from a single file with its path.
type DiffGroup struct {
	FilePath string
	Diffs    []Difference
}

// Options configures the comparison behavior.
type Options struct {
	IgnoreOrderChanges      bool
	IgnoreWhitespaceChanges bool
	IgnoreValueChanges      bool
	DetectKubernetes        bool
	DetectRenames           bool
	IgnoreApiVersion        bool
	AdditionalIdentifiers   []string
	Swap                    bool
	Chroot                  string
	ChrootFrom              string
	ChrootTo                string
	ChrootListToDocuments   bool
}

// IsListEntryDiff checks if a difference represents a list entry.
func IsListEntryDiff(diff Difference) bool {
	path := diff.Path

	if len(path) > 0 && path[len(path)-1] == ']' {
		return true
	}

	lastDot := strings.LastIndex(path, ".")
	if lastDot >= 0 && lastDot < len(path)-1 {
		suffix := path[lastDot+1:]
		isDigit := true
		for _, c := range suffix {
			if c < '0' || c > '9' {
				isDigit = false
				break
			}
		}
		if isDigit {
			return true
		}
	}

	var val interface{}
	if diff.To != nil {
		val = diff.To
	} else {
		val = diff.From
	}

	if om, ok := val.(*OrderedMap); ok {
		if _, hasName := om.Values["name"]; hasName {
			return true
		}
		if _, hasID := om.Values["id"]; hasID {
			return true
		}
	}

	return false
}

// SortDiffsWithOrder sorts differences for consistent output.
func SortDiffsWithOrder(diffs []Difference, pathOrder map[string]int) {
	findParentOrder := func(path string) (int, bool) {
		for {
			if order, ok := pathOrder[path]; ok {
				return order, true
			}
			lastDot := strings.LastIndex(path, ".")
			if lastDot == -1 {
				break
			}
			path = path[:lastDot]
		}
		return 0, false
	}

	slices.SortStableFunc(diffs, func(diffI, diffJ Difference) int {
		pathI := diffI.Path
		pathJ := diffJ.Path

		isRootRemI := diffI.Type == DiffRemoved && !strings.Contains(pathI, ".") && !IsListEntryDiff(diffI)
		isRootRemJ := diffJ.Type == DiffRemoved && !strings.Contains(pathJ, ".") && !IsListEntryDiff(diffJ)

		if isRootRemI && !isRootRemJ {
			return -1
		}
		if !isRootRemI && isRootRemJ {
			return 1
		}

		rootI := pathI
		rootJ := pathJ
		if dotIdx := strings.Index(pathI, "."); dotIdx != -1 {
			rootI = pathI[:dotIdx]
		}
		if dotIdx := strings.Index(pathJ, "."); dotIdx != -1 {
			rootJ = pathJ[:dotIdx]
		}

		if rootI != rootJ {
			orderI, okI := pathOrder[rootI]
			orderJ, okJ := pathOrder[rootJ]
			if okI && okJ {
				return cmp.Compare(orderI, orderJ)
			}
			return cmp.Compare(rootI, rootJ)
		}

		orderI, okI := pathOrder[pathI]
		orderJ, okJ := pathOrder[pathJ]
		if okI && okJ {
			return cmp.Compare(orderI, orderJ)
		}

		if okI && !okJ {
			return -1
		}
		if !okI && okJ {
			return 1
		}

		parentOrderI, okI := findParentOrder(pathI)
		parentOrderJ, okJ := findParentOrder(pathJ)
		if okI && okJ {
			if c := cmp.Compare(parentOrderI, parentOrderJ); c != 0 {
				return c
			}
		}

		if c := cmp.Compare(strings.Count(pathI, "."), strings.Count(pathJ, ".")); c != 0 {
			return c
		}

		return cmp.Compare(pathI, pathJ)
	})
}

// JoinPath joins path segments with a dot.
func JoinPath(base, key string) string {
	if base == "" {
		return key
	}
	return base + "." + key
}

// CleanPath removes leading dots from path.
func CleanPath(path string) string {
	return strings.TrimPrefix(path, ".")
}

// FormatCount returns a human-readable count string. Numbers 1-12 are spelled out.
func FormatCount(n int) string {
	words := []string{"zero", "one", "two", "three", "four", "five",
		"six", "seven", "eight", "nine", "ten", "eleven", "twelve"}
	if n >= 0 && n < len(words) {
		return words[n]
	}
	return fmt.Sprintf("%d", n)
}

// Pluralize returns singular or plural form based on count.
func Pluralize(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}
