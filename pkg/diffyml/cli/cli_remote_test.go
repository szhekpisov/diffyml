package cli

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRun_RemoteFromFile(t *testing.T) {
	fromYAML := "key: remote_value\n"
	toYAML := "key: local_value\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, fromYAML)
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.FromFile = server.URL + "/from.yaml"
	cfg.Output = "compact"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.ToContent = []byte(toYAML)

	result := Run(cfg, rc)
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected exit code %d, got %d; stderr: %s", ExitCodeSuccess, result.Code, stderr.String())
	}
	// Output should contain the diff showing value change
	output := stdout.String()
	if !strings.Contains(output, "remote_value") || !strings.Contains(output, "local_value") {
		t.Errorf("expected output to contain diff values, got: %s", output)
	}
}

func TestRun_BothRemote(t *testing.T) {
	fromYAML := "name: alice\nage: 30\n"
	toYAML := "name: alice\nage: 31\n"

	fromServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, fromYAML)
	}))
	defer fromServer.Close()

	toServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, toYAML)
	}))
	defer toServer.Close()

	cfg := NewCLIConfig()
	cfg.FromFile = fromServer.URL + "/from.yaml"
	cfg.ToFile = toServer.URL + "/to.yaml"
	cfg.Output = "compact"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr

	result := Run(cfg, rc)
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected exit code %d, got %d; stderr: %s", ExitCodeSuccess, result.Code, stderr.String())
	}
	output := stdout.String()
	if !strings.Contains(output, "age") {
		t.Errorf("expected output to contain 'age' diff, got: %s", output)
	}
}

func TestRun_RemoteWithSwap(t *testing.T) {
	fromYAML := "key: from_value\n"
	toYAML := "key: to_value\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, fromYAML)
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.FromFile = server.URL + "/from.yaml"
	cfg.Swap = true
	cfg.Output = "compact"
	cfg.SetExitCode = true

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.ToContent = []byte(toYAML)

	result := Run(cfg, rc)
	// With swap and differences, and -s flag, should return ExitCodeDifferences
	if result.Code != ExitCodeDifferences {
		t.Errorf("expected exit code %d with --swap and -s, got %d; stderr: %s",
			ExitCodeDifferences, result.Code, stderr.String())
	}
	// The swap means from and to are reversed, so from_value should appear as the "to" in the diff
	output := stdout.String()
	if !strings.Contains(output, "from_value") || !strings.Contains(output, "to_value") {
		t.Errorf("expected output to contain swapped diff values, got: %s", output)
	}
}

func TestRun_RemoteWithFilters(t *testing.T) {
	fromYAML := "app:\n  name: myapp\n  version: \"1.0\"\ndb:\n  host: localhost\n"
	toYAML := "app:\n  name: myapp\n  version: \"2.0\"\ndb:\n  host: remotehost\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, fromYAML)
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.FromFile = server.URL + "/config.yaml"
	cfg.Output = "compact"
	cfg.Filter = []string{"app"}

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.ToContent = []byte(toYAML)

	result := Run(cfg, rc)
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected exit code %d, got %d; stderr: %s", ExitCodeSuccess, result.Code, stderr.String())
	}
	output := stdout.String()
	// Should include app.version change but NOT db.host change (filtered out)
	if !strings.Contains(output, "version") {
		t.Errorf("expected output to contain 'version' diff, got: %s", output)
	}
	if strings.Contains(output, "db.host") {
		t.Errorf("expected output to NOT contain 'db.host' (filtered), got: %s", output)
	}
}
