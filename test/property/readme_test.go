package property

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestProperty12_READMECompleteness tests that the README.md file exists and
// contains sections for project description, usage instructions, and examples.
// **Validates: Requirements 5.1, 5.5**
func TestProperty12_READMECompleteness(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	// Check if README.md exists
	content, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("README.md not found: %v", err)
	}

	readmeText := string(content)

	// Check for project description (should be near the top)
	hasDescription := strings.Contains(readmeText, "# diffyml") ||
		strings.Contains(readmeText, "## Overview") ||
		strings.Contains(readmeText, "diff tool")
	if !hasDescription {
		t.Error("README.md missing project description")
	}

	// Check for usage instructions
	hasUsage := strings.Contains(readmeText, "## Usage") ||
		strings.Contains(readmeText, "usage") ||
		strings.Contains(readmeText, "diffyml [flags]")
	if !hasUsage {
		t.Error("README.md missing usage instructions")
	}

	// Check for examples
	hasExamples := strings.Contains(readmeText, "## Usage Examples") ||
		strings.Contains(readmeText, "Examples") ||
		strings.Contains(readmeText, "```bash")
	if !hasExamples {
		t.Error("README.md missing examples")
	}
}

// TestProperty13_HomebrewInstallationDocumentation tests that the README.md
// includes installation instructions mentioning Homebrew.
// **Validates: Requirements 5.2**
func TestProperty13_HomebrewInstallationDocumentation(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	content, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("README.md not found: %v", err)
	}

	readmeText := strings.ToLower(string(content))

	// Check for Homebrew mentions
	hasHomebrew := strings.Contains(readmeText, "homebrew") ||
		strings.Contains(readmeText, "brew install")
	if !hasHomebrew {
		t.Error("README.md missing Homebrew mention")
	}

	// Check for installation section
	hasInstallSection := strings.Contains(readmeText, "## installation") ||
		strings.Contains(readmeText, "### homebrew")
	if !hasInstallSection {
		t.Error("README.md missing installation section")
	}
}

// TestProperty14_CLIFlagsDocumentation tests that the README.md documents
// the primary command-line flags.
// **Validates: Requirements 5.3**
func TestProperty14_CLIFlagsDocumentation(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	content, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("README.md not found: %v", err)
	}

	readmeText := string(content)

	// Check for --help flag documentation
	hasHelpFlag := strings.Contains(readmeText, "--help") ||
		strings.Contains(readmeText, "-h")
	if !hasHelpFlag {
		t.Error("README.md missing --help flag documentation")
	}

	// Check for --version flag documentation
	if !strings.Contains(readmeText, "--version") {
		t.Error("README.md missing --version flag documentation")
	}

	// Check for core functionality flags
	hasCoreFlags := strings.Contains(readmeText, "--output") ||
		strings.Contains(readmeText, "--color") ||
		strings.Contains(readmeText, "--ignore-order-changes") ||
		strings.Contains(readmeText, "-o,") ||
		strings.Contains(readmeText, "-c,")
	if !hasCoreFlags {
		t.Error("README.md missing core flag documentation")
	}

	// Check for flags section
	hasFlagsSection := strings.Contains(readmeText, "## Core Flags") ||
		strings.Contains(readmeText, "### Core Flags") ||
		strings.Contains(readmeText, "Flags") ||
		strings.Contains(readmeText, "Options")
	if !hasFlagsSection {
		t.Error("README.md missing flags section")
	}
}

// TestProperty15_HomepageURLPresence tests that either the README.md or go.mod
// contains a valid homepage URL.
// **Validates: Requirements 5.4**
func TestProperty15_HomepageURLPresence(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	// Check README.md for URL
	readmeContent, readmeErr := os.ReadFile("README.md")
	hasURLInReadme := false
	if readmeErr == nil {
		readmeText := string(readmeContent)
		// Look for GitHub URL pattern
		urlPattern := regexp.MustCompile(`https?://github\.com/[a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+`)
		hasURLInReadme = urlPattern.MatchString(readmeText)
	}

	// Check go.mod for URL
	gomodContent, gomodErr := os.ReadFile("go.mod")
	hasURLInGomod := false
	if gomodErr == nil {
		gomodText := string(gomodContent)
		// Look for module path or comments with URLs
		urlPattern := regexp.MustCompile(`github\.com/[a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+`)
		hasURLInGomod = urlPattern.MatchString(gomodText)
	}

	// At least one must contain a URL
	if !hasURLInReadme && !hasURLInGomod {
		t.Error("neither README.md nor go.mod contains a homepage URL")
	}
}

// TestProperty11_LicenseDocumentationInREADME tests that the README.md contains
// a reference to the license.
// **Validates: Requirements 4.5**
func TestProperty11_LicenseDocumentationInREADME(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	content, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("README.md not found: %v", err)
	}

	readmeText := strings.ToLower(string(content))

	// Check for license mentions
	hasLicenseMention := strings.Contains(readmeText, "license") ||
		strings.Contains(readmeText, "mit") ||
		strings.Contains(readmeText, "apache") ||
		strings.Contains(readmeText, "gpl")
	if !hasLicenseMention {
		t.Error("README.md missing license mention")
	}

	// Check for LICENSE file reference or license section
	hasLicenseFileRef := strings.Contains(readmeText, "license file") ||
		strings.Contains(readmeText, "[license]") ||
		strings.Contains(readmeText, "see the license")
	hasLicenseSection := strings.Contains(readmeText, "## license")
	if !hasLicenseFileRef && !hasLicenseSection {
		t.Error("README.md missing license file reference or license section")
	}
}

// TestProperty20_ProjectDescriptionPresence tests that the README.md contains
// a concise one-line description of the tool's purpose.
// **Validates: Requirements 8.1**
func TestProperty20_ProjectDescriptionPresence(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	content, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("README.md not found: %v", err)
	}

	readmeText := string(content)
	lines := strings.Split(readmeText, "\n")

	// Check first few lines for a description
	hasDescription := false
	for i := 0; i < len(lines) && i < 10; i++ {
		line := strings.TrimSpace(lines[i])
		if strings.Contains(strings.ToLower(line), "diff") &&
			(strings.Contains(strings.ToLower(line), "yaml") ||
				strings.Contains(strings.ToLower(line), "tool")) {
			hasDescription = true
			break
		}
	}

	if !hasDescription {
		t.Error("README.md missing project description in first 10 lines")
	}
	if !strings.Contains(readmeText, "# diffyml") {
		t.Error("README.md missing title")
	}
}

// TestProperty21_InstallationVerificationInstructions tests that the README.md
// includes instructions for verifying the installation.
// **Validates: Requirements 8.5**
func TestProperty21_InstallationVerificationInstructions(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	content, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("README.md not found: %v", err)
	}

	readmeText := strings.ToLower(string(content))

	// Check for verification section
	hasVerificationSection := strings.Contains(readmeText, "verification") ||
		strings.Contains(readmeText, "verify")

	// Check for verification commands
	hasVersionCheck := strings.Contains(readmeText, "--version") &&
		(strings.Contains(readmeText, "verify") ||
			strings.Contains(readmeText, "check"))

	hasHelpCheck := strings.Contains(readmeText, "--help") &&
		(strings.Contains(readmeText, "verify") ||
			strings.Contains(readmeText, "check") ||
			strings.Contains(readmeText, "display"))

	if !hasVerificationSection && !hasVersionCheck && !hasHelpCheck {
		t.Error("README.md missing installation verification instructions")
	}
}

// TestProperty12_READMECompleteness_DetailedValidation performs more detailed
// validation of README sections to ensure comprehensive documentation.
// **Validates: Requirements 5.1, 5.5**
func TestProperty12_READMECompleteness_DetailedValidation(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	content, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("README.md not found: %v", err)
	}

	readmeText := string(content)

	// Check for multiple required sections
	requiredSections := []string{
		"# diffyml",       // Title
		"## Overview",     // Overview section
		"## Installation", // Installation section
		"## Usage",        // Usage section
		"## License",      // License section
	}

	sectionsFound := 0
	for _, section := range requiredSections {
		if strings.Contains(readmeText, section) {
			sectionsFound++
		}
	}

	if sectionsFound < 4 {
		t.Errorf("README.md has only %d/5 required sections (need at least 4)", sectionsFound)
	}

	// Check for code examples (bash blocks)
	codeBlockPattern := regexp.MustCompile("```bash")
	if !codeBlockPattern.MatchString(readmeText) {
		t.Error("README.md missing bash code examples")
	}

	// Check for feature list or key features
	hasFeatures := strings.Contains(readmeText, "Features") ||
		strings.Contains(readmeText, "Key Features")
	if !hasFeatures {
		t.Error("README.md missing features section")
	}

	// Ensure README has substantial content
	if len(readmeText) <= 1000 {
		t.Errorf("README.md too short (%d bytes, expected > 1000)", len(readmeText))
	}
}
