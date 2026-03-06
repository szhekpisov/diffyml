// chroot.go - Path navigation to focus comparison on YAML subsections.
package diffyml

import (
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/compare"
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/types"
)

type ChrootError = types.ChrootError

func navigateToPath(doc interface{}, path string) (interface{}, error) {
	return compare.NavigateToPath(doc, path)
}

func applyChroot(doc interface{}, path string, listToDocuments bool) ([]interface{}, error) {
	return compare.ApplyChroot(doc, path, listToDocuments)
}

type pathSegment = compare.PathSegment

func parsePath(path string) ([]pathSegment, error) {
	return compare.ParsePath(path)
}

func splitPath(path string) ([]string, error) {
	return compare.SplitPath(path)
}
