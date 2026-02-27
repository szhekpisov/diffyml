package e2e

import (
	"fmt"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func skipIfNoDocker(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not installed, skipping E2E test")
	}
	// Check that Docker daemon is actually running
	cmd := exec.Command("docker", "info")
	if err := cmd.Run(); err != nil {
		t.Skip("docker daemon not running, skipping E2E test")
	}
}

func skipIfNoKind(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("kind"); err != nil {
		t.Skip("kind not installed, skipping E2E test")
	}
}

func skipIfNoKubectl(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("kubectl"); err != nil {
		t.Skip("kubectl not installed, skipping E2E test")
	}
}

const baseManifest = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: diffyml-e2e-test
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: diffyml-e2e-test
  template:
    metadata:
      labels:
        app: diffyml-e2e-test
    spec:
      containers:
        - name: nginx
          image: nginx
`

const modifiedManifest = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: diffyml-e2e-test
  namespace: default
spec:
  replicas: 3
  selector:
    matchLabels:
      app: diffyml-e2e-test
  template:
    metadata:
      labels:
        app: diffyml-e2e-test
    spec:
      containers:
        - name: nginx
          image: nginx:latest
`

func TestKubectlDiffWithDiffyml(t *testing.T) {
	skipIfNoDocker(t)
	skipIfNoKind(t)
	skipIfNoKubectl(t)

	// Build diffyml binary
	tmpDir := t.TempDir()
	diffymlBin := filepath.Join(tmpDir, "diffyml")

	// Find project root (two levels up from test/e2e/)
	projectRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("failed to resolve project root: %v", err)
	}

	buildCmd := exec.Command("go", "build", "-o", diffymlBin, ".")
	buildCmd.Dir = projectRoot
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build diffyml: %v\n%s", err, out)
	}

	// Create kind cluster with unique name
	clusterName := fmt.Sprintf("diffyml-e2e-%d", rand.IntN(100000))
	t.Logf("Creating kind cluster %s...", clusterName)

	createCmd := exec.Command("kind", "create", "cluster", "--name", clusterName, "--wait", "60s")
	if out, err := createCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to create kind cluster: %v\n%s", err, out)
	}
	t.Cleanup(func() {
		t.Logf("Deleting kind cluster %s...", clusterName)
		deleteCmd := exec.Command("kind", "delete", "cluster", "--name", clusterName)
		if out, err := deleteCmd.CombinedOutput(); err != nil {
			t.Logf("warning: failed to delete kind cluster: %v\n%s", err, out)
		}
	})

	// Get kubeconfig for this cluster
	kubeconfigPath := filepath.Join(tmpDir, "kubeconfig")
	kcCmd := exec.Command("kind", "get", "kubeconfig", "--name", clusterName)
	kcOut, err := kcCmd.Output()
	if err != nil {
		t.Fatalf("failed to get kubeconfig: %v", err)
	}
	if err := os.WriteFile(kubeconfigPath, kcOut, 0600); err != nil {
		t.Fatalf("failed to write kubeconfig: %v", err)
	}

	env := append(os.Environ(),
		"KUBECONFIG="+kubeconfigPath,
		fmt.Sprintf("KUBECTL_EXTERNAL_DIFF=%s --omit-header --set-exit-code --color never", diffymlBin),
	)

	// Apply base manifest
	applyCmd := exec.Command("kubectl", "apply", "-f", "-")
	applyCmd.Env = env
	applyCmd.Stdin = strings.NewReader(baseManifest)
	if out, err := applyCmd.CombinedOutput(); err != nil {
		t.Fatalf("kubectl apply failed: %v\n%s", err, out)
	}

	// Wait for the resource to exist in the API server
	waitForResource(t, env)

	// Run kubectl diff with modified manifest
	diffCmd := exec.Command("kubectl", "diff", "-f", "-")
	diffCmd.Env = env
	diffCmd.Stdin = strings.NewReader(modifiedManifest)
	var stdout, stderr strings.Builder
	diffCmd.Stdout = &stdout
	diffCmd.Stderr = &stderr

	err = diffCmd.Run()

	// kubectl diff exits 1 when differences are found (propagated from external diff tool)
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("kubectl diff failed unexpectedly: %v\nstderr: %s", err, stderr.String())
		}
	}

	t.Logf("kubectl diff exit code: %d", exitCode)
	t.Logf("stdout:\n%s", stdout.String())
	if stderr.Len() > 0 {
		t.Logf("stderr:\n%s", stderr.String())
	}

	// Assert: exit code is 0 or 1
	// kubectl diff returns 1 when differences are found with the built-in diff,
	// but may return 0 when using KUBECTL_EXTERNAL_DIFF depending on kubectl version.
	// Exit code >= 2 indicates an error.
	if exitCode >= 2 {
		t.Errorf("expected exit code 0 or 1, got %d (indicates an error)", exitCode)
	}

	// Assert: stdout is non-empty
	output := stdout.String()
	if output == "" {
		t.Fatal("expected non-empty stdout from kubectl diff")
	}

	// Assert: output contains expected diff fragments
	for _, fragment := range []string{"spec.replicas", "image", "nginx:latest"} {
		if !strings.Contains(output, fragment) {
			t.Errorf("expected output to contain %q, got:\n%s", fragment, output)
		}
	}
}

// waitForResource polls until the Deployment exists in the API server.
func waitForResource(t *testing.T, env []string) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		cmd := exec.Command("kubectl", "get", "deployment", "diffyml-e2e-test", "-n", "default")
		cmd.Env = env
		if err := cmd.Run(); err == nil {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatal("timed out waiting for deployment to exist")
}
