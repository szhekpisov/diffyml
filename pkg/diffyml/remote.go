package diffyml

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	// MaxResponseSize is the maximum allowed response body size (10 MB).
	MaxResponseSize = 10 * 1024 * 1024
	// DefaultTimeout is the HTTP client timeout for remote fetches.
	DefaultTimeout = 30 * time.Second
)

// RemoteError represents an error fetching content from a remote URL.
type RemoteError struct {
	URL        string
	StatusCode int
	Message    string
	Err        error
}

// Error implements the error interface.
func (e *RemoteError) Error() string {
	return e.Message
}

// Unwrap returns the underlying error, if any.
func (e *RemoteError) Unwrap() error {
	return e.Err
}

// ValidateFileExists checks if a file exists and is not a directory.
// Returns an error with the file path if validation fails.
func ValidateFileExists(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", path)
		}
		return fmt.Errorf("cannot access file %s: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file: %s", path)
	}
	return nil
}

// IsRemoteSource returns true if the source string is an HTTP/HTTPS URL.
// Uses strict lowercase prefix matching.
func IsRemoteSource(source string) bool {
	return strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://")
}

// LoadContent loads content from a source, which can be a local file path
// or an HTTP/HTTPS URL. Returns the content bytes or an error.
func LoadContent(source string) ([]byte, error) {
	if IsRemoteSource(source) {
		return fetchURL(source)
	}

	if err := ValidateFileExists(source); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(source) // #nosec G304 -- source is a user-provided CLI argument
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", source, err)
	}
	return data, nil
}

// fetchURL fetches content from an HTTP/HTTPS URL with timeout and size limits.
func fetchURL(url string) ([]byte, error) {
	client := &http.Client{Timeout: DefaultTimeout}

	resp, err := client.Get(url)
	if err != nil {
		return nil, &RemoteError{
			URL:     url,
			Message: fmt.Sprintf("failed to fetch %s: %v", url, err),
			Err:     err,
		}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &RemoteError{
			URL:        url,
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("failed to fetch %s: HTTP %d", url, resp.StatusCode),
		}
	}

	limited := io.LimitReader(resp.Body, int64(MaxResponseSize)+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, &RemoteError{
			URL:     url,
			Message: fmt.Sprintf("failed to read response from %s: %v", url, err),
			Err:     err,
		}
	}

	if len(data) > MaxResponseSize {
		return nil, &RemoteError{
			URL:     url,
			Message: fmt.Sprintf("response too large from %s: exceeds 10 MB limit", url),
		}
	}

	return data, nil
}
