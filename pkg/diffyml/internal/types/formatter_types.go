package types

// Formatter formats differences for output.
type Formatter interface {
	Format(diffs []Difference, opts *FormatOptions) string
}

// FormatOptions configures output formatting.
type FormatOptions struct {
	Color            bool
	TrueColor        bool
	OmitHeader       bool
	UseGoPatchStyle  bool
	ContextLines     int
	NoCertInspection bool
	FilePath         string
}

// StructuredFormatter is an opt-in interface for formatters that need
// aggregated output across all files in directory mode.
type StructuredFormatter interface {
	FormatAll(groups []DiffGroup, opts *FormatOptions) string
}

// DefaultFormatOptions returns FormatOptions with default values.
func DefaultFormatOptions() *FormatOptions {
	return &FormatOptions{
		Color:           false,
		TrueColor:       false,
		OmitHeader:      false,
		UseGoPatchStyle: false,
		ContextLines:    4,
	}
}
