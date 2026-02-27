package diffyml

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestIsRemoteSource(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   bool
	}{
		{"http URL", "http://example.com/file.yaml", true},
		{"https URL", "https://example.com/file.yaml", true},
		{"http with path", "http://example.com/path/to/file.yaml", true},
		{"https with path", "https://raw.githubusercontent.com/user/repo/main/config.yaml", true},
		{"local file path", "/path/to/file.yaml", false},
		{"relative path", "file.yaml", false},
		{"empty string", "", false},
		{"uppercase HTTP treated as local", "HTTP://example.com/file.yaml", false},
		{"uppercase HTTPS treated as local", "HTTPS://example.com/file.yaml", false},
		{"mixed case Http treated as local", "Http://example.com/file.yaml", false},
		{"ftp not supported", "ftp://example.com/file.yaml", false},
		{"just http prefix no colon-slash", "http-something", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRemoteSource(tt.source)
			if got != tt.want {
				t.Errorf("IsRemoteSource(%q) = %v, want %v", tt.source, got, tt.want)
			}
		})
	}
}

func TestLoadContent_LocalFile(t *testing.T) {
	// Create a temporary file with known content
	tmpDir := t.TempDir()
	tmpFile := tmpDir + "/test.yaml"
	content := []byte("key: value\n")
	if err := writeTestFile(tmpFile, content); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	got, err := LoadContent(tmpFile)
	if err != nil {
		t.Fatalf("LoadContent(%q) returned error: %v", tmpFile, err)
	}
	if string(got) != string(content) {
		t.Errorf("LoadContent(%q) = %q, want %q", tmpFile, got, content)
	}
}

func TestLoadContent_NonExistentFile(t *testing.T) {
	_, err := LoadContent("/nonexistent/path/file.yaml")
	if err == nil {
		t.Error("LoadContent for non-existent file should return error")
	}
}

func TestFetchURL_Success(t *testing.T) {
	yamlContent := "key: value\nlist:\n  - item1\n  - item2\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/yaml")
		fmt.Fprint(w, yamlContent)
	}))
	defer server.Close()

	got, err := fetchURL(server.URL)
	if err != nil {
		t.Fatalf("fetchURL(%q) returned error: %v", server.URL, err)
	}
	if string(got) != yamlContent {
		t.Errorf("fetchURL(%q) = %q, want %q", server.URL, got, yamlContent)
	}
}

func TestFetchURL_Non2xxStatus(t *testing.T) {
	statusCodes := []int{400, 403, 404, 500, 502}

	for _, code := range statusCodes {
		t.Run(fmt.Sprintf("status_%d", code), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
			}))
			defer server.Close()

			_, err := fetchURL(server.URL)
			if err == nil {
				t.Fatalf("fetchURL with status %d should return error", code)
			}
			// Error should include the status code
			if !strings.Contains(err.Error(), fmt.Sprintf("%d", code)) {
				t.Errorf("error %q should contain status code %d", err.Error(), code)
			}
			// Error should include the URL
			if !strings.Contains(err.Error(), server.URL) {
				t.Errorf("error %q should contain URL %q", err.Error(), server.URL)
			}
		})
	}
}

func TestFetchURL_NetworkError(t *testing.T) {
	// Use a server that is immediately closed to produce a connection refused error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	serverURL := server.URL
	server.Close() // Close immediately so connection is refused

	_, err := fetchURL(serverURL)
	if err == nil {
		t.Error("fetchURL to closed server should return error")
	}
	// Error should contain the URL
	if !strings.Contains(err.Error(), serverURL) {
		t.Errorf("error %q should contain URL %q", err.Error(), serverURL)
	}
}

func TestFetchURL_TooLarge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Write more than MaxResponseSize (10 MB + 1 byte)
		data := make([]byte, MaxResponseSize+1)
		for i := range data {
			data[i] = 'x'
		}
		w.Write(data)
	}))
	defer server.Close()

	_, err := fetchURL(server.URL)
	if err == nil {
		t.Error("fetchURL with oversized response should return error")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Errorf("error %q should mention 'too large'", err.Error())
	}
}

func TestFetchURL_ExactlyMaxSize(t *testing.T) {
	// A response of exactly MaxResponseSize should be allowed
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := make([]byte, MaxResponseSize)
		for i := range data {
			data[i] = 'y'
		}
		w.Write(data)
	}))
	defer server.Close()

	got, err := fetchURL(server.URL)
	if err != nil {
		t.Fatalf("fetchURL with exactly max size should succeed, got error: %v", err)
	}
	if len(got) != MaxResponseSize {
		t.Errorf("expected %d bytes, got %d", MaxResponseSize, len(got))
	}
}

func TestFetchURL_InvalidURL(t *testing.T) {
	_, err := fetchURL("http://[invalid-url")
	if err == nil {
		t.Error("fetchURL with invalid URL should return error")
	}
}

func TestLoadContent_RemoteURL(t *testing.T) {
	yamlContent := "remote: true\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, yamlContent)
	}))
	defer server.Close()

	got, err := LoadContent(server.URL)
	if err != nil {
		t.Fatalf("LoadContent(%q) returned error: %v", server.URL, err)
	}
	if string(got) != yamlContent {
		t.Errorf("LoadContent(%q) = %q, want %q", server.URL, got, yamlContent)
	}
}

func TestLoadContent_RemoteError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := LoadContent(server.URL)
	if err == nil {
		t.Error("LoadContent for remote 404 should return error")
	}
}

// --- Mutation testing: remote.go ---

func TestFetchURL_HTTP300Rejected(t *testing.T) {
	// remote.go:52 — HTTP 300 (not in 200-299 range) should be rejected
	// If >= 300 mutated to > 300, status 300 would slip through
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(300)
	}))
	defer server.Close()

	_, err := fetchURL(server.URL)
	if err == nil {
		t.Fatal("fetchURL with HTTP 300 should return error")
	}
	if !strings.Contains(err.Error(), "300") {
		t.Errorf("error should contain status code 300, got: %v", err)
	}
}

// writeTestFile is a helper to create test files.
func writeTestFile(path string, content []byte) error {
	return os.WriteFile(path, content, 0600)
}
