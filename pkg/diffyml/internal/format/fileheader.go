package format

import (
	"fmt"

	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/types"
)

// FilePairType describes the relationship between source and target files.
type FilePairType int

const (
	FilePairBothExist FilePairType = iota
	FilePairOnlyFrom
	FilePairOnlyTo
)

// FormatFileHeader returns a unified-diff-style file header for directory mode.
// Uses "--- a/<filename>" / "+++ b/<filename>" for BothExist,
// "/dev/null" for the absent side on OnlyFrom/OnlyTo.
// Applies yellow/bold color when opts.Color is true.
func FormatFileHeader(filename string, pairType FilePairType, opts *types.FormatOptions) string {
	var fromLine, toLine string

	switch pairType {
	case FilePairBothExist:
		fromLine = "--- a/" + filename
		toLine = "+++ b/" + filename
	case FilePairOnlyFrom:
		fromLine = "--- a/" + filename
		toLine = "+++ /dev/null"
	case FilePairOnlyTo:
		fromLine = "--- /dev/null"
		toLine = "+++ b/" + filename
	}

	if opts != nil && opts.Color {
		return fmt.Sprintf("%s%s%s\n%s%s%s\n",
			StyleBold+ColorWhite, fromLine, ColorReset,
			StyleBold+ColorWhite, toLine, ColorReset)
	}
	return fmt.Sprintf("%s\n%s\n", fromLine, toLine)
}
