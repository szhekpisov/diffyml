package diffyml

import (
	"regexp"
	"strconv"
)

// mappingValuePathAliases appends a mapping key to every equivalent base path.
// A value may have more than one base when it sits below a named list item: the
// numeric index and identifier forms are both retained so filters and masks can
// use either spelling.
func mappingValuePathAliases(bases []DiffPath, key string) []DiffPath {
	paths := make([]DiffPath, len(bases))
	for i, base := range bases {
		paths[i] = base.Append(key)
	}
	return paths
}

// sequenceValuePathAliases appends both the numeric index and, when available,
// the same identifier segment used by the comparator. Supporting both forms
// preserves existing numeric filtering while allowing paths such as
// containers.app.image for collapsed named lists.
func sequenceValuePathAliases(bases []DiffPath, item any, index int, additionalIdentifiers []string) []DiffPath {
	indexSegment := strconv.Itoa(index)
	segments := []string{indexSegment}
	if id := valueIdentifier(item, additionalIdentifiers); isComparableIdentifier(id) {
		if identifierSegment := sprintIdentifier(id); identifierSegment != indexSegment {
			segments = append(segments, identifierSegment)
		}
	}

	var paths []DiffPath
	for _, base := range bases {
		for _, segment := range segments {
			paths = append(paths, base.Append(segment))
		}
	}
	return paths
}

func valueIdentifier(value any, additionalIdentifiers []string) any {
	switch val := value.(type) {
	case *OrderedMap:
		return getIdentifierFromOrderedMap(val, additionalIdentifiers)
	case map[string]any:
		return IdentifierWithAdditional(val, additionalIdentifiers)
	default:
		return nil
	}
}

func appendAliasStrings(paths []string, aliases []DiffPath) []string {
	for _, alias := range aliases {
		paths = append(paths, alias.String())
	}
	return paths
}

func anyAliasMatches(aliases []DiffPath, paths []string, regex []*regexp.Regexp) bool {
	for _, alias := range aliases {
		path := alias.String()
		if matchesAnyPath(path, paths) || matchesAnyRegex(path, regex) {
			return true
		}
	}
	return false
}
