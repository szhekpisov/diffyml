// Package compare provides the core YAML comparison logic for diffyml.
//
// It exposes the Compare function as the main entry point, which parses,
// optionally chroots, and compares two YAML documents, returning structured differences.
package compare

import (
	"fmt"

	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/parse"
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/types"
)

// Compare compares two YAML documents and returns the differences.
// The from and to parameters should contain valid YAML content.
// If opts is nil, default options are used.
func Compare(from, to []byte, opts *types.Options) ([]types.Difference, error) {
	if opts == nil {
		opts = &types.Options{}
	}

	// Parse both YAML documents
	fromDocs, err := parse.Parse(from)
	if err != nil {
		return nil, err
	}
	toDocs, err := parse.Parse(to)
	if err != nil {
		return nil, err
	}

	// Apply swap if requested
	if opts.Swap {
		fromDocs, toDocs = toDocs, fromDocs
	}

	// Apply chroot if specified
	if opts.Chroot != "" {
		fromDocs, err = ApplyChrootToDocs(fromDocs, opts.Chroot, opts.ChrootListToDocuments)
		if err != nil {
			return nil, err
		}
		toDocs, err = ApplyChrootToDocs(toDocs, opts.Chroot, opts.ChrootListToDocuments)
		if err != nil {
			return nil, err
		}
	} else {
		if opts.ChrootFrom != "" {
			fromDocs, err = ApplyChrootToDocs(fromDocs, opts.ChrootFrom, opts.ChrootListToDocuments)
			if err != nil {
				return nil, err
			}
		}
		if opts.ChrootTo != "" {
			toDocs, err = ApplyChrootToDocs(toDocs, opts.ChrootTo, opts.ChrootListToDocuments)
			if err != nil {
				return nil, err
			}
		}
	}

	// Compare documents and sort results
	pathOrder := ExtractPathOrder(fromDocs, toDocs, opts)
	diffs := CompareDocs(fromDocs, toDocs, opts)
	types.SortDiffsWithOrder(diffs, pathOrder)

	return diffs, nil
}

// ExtractPathOrder extracts the order of all paths from parsed documents.
func ExtractPathOrder(fromDocs, toDocs []interface{}, opts *types.Options) map[string]int {
	pathOrder := make(map[string]int)
	index := 0

	registerPrefix := func(prefix string) {
		if prefix != "" {
			if _, exists := pathOrder[prefix]; !exists {
				pathOrder[prefix] = index
				index++
			}
		}
	}

	var extractFromValue func(prefix string, val interface{})
	extractFromValue = func(prefix string, val interface{}) {
		switch v := val.(type) {
		case *types.OrderedMap:
			registerPrefix(prefix)
			for _, key := range v.Keys {
				childPath := key
				if prefix != "" {
					childPath = prefix + "." + key
				}
				extractFromValue(childPath, v.Values[key])
			}
		case []interface{}:
			registerPrefix(prefix)
			for i, item := range v {
				var childPath string
				if id := GetIdentifier(item, opts); IsComparableIdentifier(id) {
					childPath = fmt.Sprintf("%s.%v", prefix, id)
				} else {
					childPath = fmt.Sprintf("%s.%d", prefix, i)
				}
				extractFromValue(childPath, item)
			}
		default:
			registerPrefix(prefix)
		}
	}

	// Extract from both documents to cover all keys
	for _, doc := range fromDocs {
		extractFromValue("", doc)
	}
	for _, doc := range toDocs {
		extractFromValue("", doc)
	}

	return pathOrder
}
