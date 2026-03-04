// remote.go - Remote and local file loading.
package diffyml

import (
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/run"
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/types"
)

const (
	// MaxResponseSize is the maximum allowed response body size (10 MB).
	MaxResponseSize = run.MaxResponseSize
	// DefaultTimeout is the HTTP client timeout for remote fetches.
	DefaultTimeout = run.DefaultTimeout
)

// RemoteError represents an error fetching content from a remote URL.
type RemoteError = types.RemoteError

// ValidateFileExists checks if a file exists and is not a directory.
func ValidateFileExists(path string) error { return run.ValidateFileExists(path) }

// IsRemoteSource returns true if the source string is an HTTP/HTTPS URL.
func IsRemoteSource(source string) bool { return run.IsRemoteSource(source) }

// LoadContent loads content from a source, which can be a local file path or HTTP/HTTPS URL.
func LoadContent(source string) ([]byte, error) { return run.LoadContent(source) }

// fetchURL fetches content from an HTTP/HTTPS URL with timeout and size limits.
// Unexported; required by tests in package diffyml.
func fetchURL(url string) ([]byte, error) { return run.FetchURL(url) }
