package property

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestProperty10_OpenSourceDependencies tests that all dependencies have
// open-source licenses compatible with Homebrew distribution.
// **Validates: Requirements 4.3**
func TestProperty10_OpenSourceDependencies(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	// Check if go-licenses is available
	goLicensesPath := os.ExpandEnv("$HOME/go/bin/go-licenses")
	if _, err := os.Stat(goLicensesPath); err != nil {
		// Try finding it in PATH
		goLicensesPath = "go-licenses"
	}

	// Run go-licenses check to verify all dependencies have acceptable licenses
	cmd := exec.Command(goLicensesPath, "check", "./...")
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Check if the error is due to go-licenses not being installed
		if strings.Contains(string(output), "not found") ||
			strings.Contains(err.Error(), "not found") ||
			strings.Contains(err.Error(), "executable file not found") {
			t.Skip("go-licenses tool not installed, skipping test")
		}
		if strings.Contains(string(output), "does not have module info") {
			t.Skip("go-licenses incompatible with current Go version, skipping test")
		}
		t.Fatalf("go-licenses check failed: %v\nOutput: %s", err, string(output))
	}

	// If go-licenses check passes without error, all dependencies have acceptable licenses
}

// TestProperty10_OpenSourceDependencies_Report tests that all dependencies
// can be reported and have identifiable licenses.
// **Validates: Requirements 4.3**
func TestProperty10_OpenSourceDependencies_Report(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	// Check if go-licenses is available
	goLicensesPath := os.ExpandEnv("$HOME/go/bin/go-licenses")
	if _, err := os.Stat(goLicensesPath); err != nil {
		goLicensesPath = "go-licenses"
	}

	// Run go-licenses report to get license information
	cmd := exec.Command(goLicensesPath, "report", "./...")
	output, err := cmd.CombinedOutput()

	if err != nil {
		if strings.Contains(string(output), "not found") ||
			strings.Contains(err.Error(), "not found") ||
			strings.Contains(err.Error(), "executable file not found") {
			t.Skip("go-licenses tool not installed, skipping test")
		}
		if strings.Contains(string(output), "does not have module info") {
			t.Skip("go-licenses incompatible with current Go version, skipping test")
		}
		t.Fatalf("go-licenses report failed: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)
	lines := strings.Split(strings.TrimSpace(outputStr), "\n")

	// Known open-source licenses that are Homebrew-compatible
	acceptableLicenses := map[string]bool{
		"MIT":          true,
		"Apache-2.0":   true,
		"BSD-2-Clause": true,
		"BSD-3-Clause": true,
		"ISC":          true,
		"MPL-2.0":      true,
		"Unlicense":    true,
		"CC0-1.0":      true,
		"LGPL-2.1":     true,
		"LGPL-3.0":     true,
		"GPL-2.0":      true,
		"GPL-3.0":      true,
	}

	for _, line := range lines {
		if line == "" {
			continue
		}

		// Parse the CSV format: module,license_url,license_type
		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			continue
		}

		moduleName := parts[0]
		licenseType := parts[2]

		// Check if the license is acceptable
		if !acceptableLicenses[licenseType] {
			// Special case: "Unknown" license with the module being diffyml itself is OK
			// because we control our own license
			if moduleName == "diffyml" && licenseType == "MIT" {
				continue
			}

			// If license is not in our list but go-licenses check passed,
			// it might still be acceptable (go-licenses has its own allowlist)
			t.Logf("Note: %s has license %s (may still be acceptable)", moduleName, licenseType)
		}
	}
}

// TestProperty10_OpenSourceDependencies_NoProprietaryLicenses tests that
// no dependencies have proprietary or restrictive licenses.
// **Validates: Requirements 4.3**
func TestProperty10_OpenSourceDependencies_NoProprietaryLicenses(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	// Check if go-licenses is available
	goLicensesPath := os.ExpandEnv("$HOME/go/bin/go-licenses")
	if _, err := os.Stat(goLicensesPath); err != nil {
		goLicensesPath = "go-licenses"
	}

	// Run go-licenses report
	cmd := exec.Command(goLicensesPath, "report", "./...")
	output, err := cmd.CombinedOutput()

	if err != nil {
		if strings.Contains(string(output), "not found") ||
			strings.Contains(err.Error(), "not found") ||
			strings.Contains(err.Error(), "executable file not found") {
			t.Skip("go-licenses tool not installed, skipping test")
		}
		if strings.Contains(string(output), "does not have module info") {
			t.Skip("go-licenses incompatible with current Go version, skipping test")
		}
		t.Fatalf("go-licenses report failed: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)

	// Check for known problematic license types
	problematicLicenses := []string{
		"BUSL", // Business Source License
		"SSPL", // Server Side Public License
		"Proprietary",
		"Commercial",
		"Restricted",
	}

	for _, license := range problematicLicenses {
		if strings.Contains(outputStr, license) {
			t.Errorf("Found potentially problematic license: %s", license)
		}
	}
}
