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
		return nil, fmt.Errorf("failed to fetch %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("failed to fetch %s: HTTP %d", url, resp.StatusCode)
	}

	limited := io.LimitReader(resp.Body, int64(MaxResponseSize)+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("failed to read response from %s: %w", url, err)
	}

	if len(data) > MaxResponseSize {
		return nil, fmt.Errorf("response too large from %s: exceeds 10 MB limit", url)
	}

	return data, nil
}
