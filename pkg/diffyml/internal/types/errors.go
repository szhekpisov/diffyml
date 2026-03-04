package types

import "fmt"

// ValidationError represents a CLI configuration validation error.
type ValidationError struct {
	Field   string
	Value   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// ParseError represents a YAML parsing error with location information.
type ParseError struct {
	Line    int
	Column  int
	Message string
	Err     error
}

func (e *ParseError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("yaml: line %d: %s", e.Line, e.Message)
	}
	return fmt.Sprintf("yaml: %s", e.Message)
}

func (e *ParseError) Unwrap() error {
	return e.Err
}

// ChrootError represents an error navigating to a chroot path.
type ChrootError struct {
	Path    string
	Message string
	Err     error
}

func (e *ChrootError) Error() string {
	return fmt.Sprintf("chroot path %q: %s", e.Path, e.Message)
}

func (e *ChrootError) Unwrap() error {
	return e.Err
}

// RemoteError represents an error fetching content from a remote URL.
type RemoteError struct {
	URL        string
	StatusCode int
	Message    string
	Err        error
}

func (e *RemoteError) Error() string {
	return e.Message
}

func (e *RemoteError) Unwrap() error {
	return e.Err
}
