package property

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestProperty12_READMECompleteness tests that the README.md file exists and
// contains sections for project description, usage instructions, and examples.
// **Validates: Requirements 5.1, 5.5**
func TestProperty12_READMECompleteness(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newProperties()

	properties.Property("README.md must exist and contain required sections", prop.ForAll(
		func(dummyInput int) bool {
			// Check if README.md exists
			content, err := os.ReadFile("README.md")
			if err != nil {
				return false
			}

			readmeText := string(content)

			// Check for project description (should be near the top)
			// Look for a header or introductory text
			hasDescription := strings.Contains(readmeText, "# diffyml") ||
				strings.Contains(readmeText, "## Overview") ||
				strings.Contains(readmeText, "diff tool")

			// Check for usage instructions
			// Look for usage section or command examples
			hasUsage := strings.Contains(readmeText, "## Usage") ||
				strings.Contains(readmeText, "usage") ||
				strings.Contains(readmeText, "diffyml [flags]")

			// Check for examples
			// Look for examples section or code blocks with examples
			hasExamples := strings.Contains(readmeText, "## Usage Examples") ||
				strings.Contains(readmeText, "Examples") ||
				strings.Contains(readmeText, "```bash")

			return hasDescription && hasUsage && hasExamples
		},
		gen.IntRange(1, 100), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty13_HomebrewInstallationDocumentation tests that the README.md
// includes installation instructions mentioning Homebrew.
// **Validates: Requirements 5.2**
func TestProperty13_HomebrewInstallationDocumentation(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newProperties()

	properties.Property("README.md must include Homebrew installation instructions", prop.ForAll(
		func(dummyInput int) bool {
			content, err := os.ReadFile("README.md")
			if err != nil {
				return false
			}

			readmeText := strings.ToLower(string(content))

			// Check for Homebrew mentions
			hasHomebrew := strings.Contains(readmeText, "homebrew") ||
				strings.Contains(readmeText, "brew install")

			// Check for installation section
			hasInstallSection := strings.Contains(readmeText, "## installation") ||
				strings.Contains(readmeText, "### homebrew")

			return hasHomebrew && hasInstallSection
		},
		gen.IntRange(1, 100), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty14_CLIFlagsDocumentation tests that the README.md documents
// the primary command-line flags.
// **Validates: Requirements 5.3**
func TestProperty14_CLIFlagsDocumentation(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newProperties()

	properties.Property("README.md must document primary CLI flags", prop.ForAll(
		func(dummyInput int) bool {
			content, err := os.ReadFile("README.md")
			if err != nil {
				return false
			}

			readmeText := string(content)

			// Check for --help flag documentation
			hasHelpFlag := strings.Contains(readmeText, "--help") ||
				strings.Contains(readmeText, "-h")

			// Check for --version flag documentation
			hasVersionFlag := strings.Contains(readmeText, "--version")

			// Check for core functionality flags
			// Look for common flags like output, color, ignore-order-changes, etc.
			hasCoreFlags := strings.Contains(readmeText, "--output") ||
				strings.Contains(readmeText, "--color") ||
				strings.Contains(readmeText, "--ignore-order-changes") ||
				strings.Contains(readmeText, "-o,") ||
				strings.Contains(readmeText, "-c,")

			// Check for flags section
			hasFlagsSection := strings.Contains(readmeText, "## Core Flags") ||
				strings.Contains(readmeText, "### Core Flags") ||
				strings.Contains(readmeText, "Flags") ||
				strings.Contains(readmeText, "Options")

			return hasHelpFlag && hasVersionFlag && hasCoreFlags && hasFlagsSection
		},
		gen.IntRange(1, 100), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty15_HomepageURLPresence tests that either the README.md or go.mod
// contains a valid homepage URL.
// **Validates: Requirements 5.4**
func TestProperty15_HomepageURLPresence(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newProperties()

	properties.Property("README.md or go.mod must contain a valid homepage URL", prop.ForAll(
		func(dummyInput int) bool {
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
			return hasURLInReadme || hasURLInGomod
		},
		gen.IntRange(1, 100), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty11_LicenseDocumentationInREADME tests that the README.md contains
// a reference to the license.
// **Validates: Requirements 4.5**
func TestProperty11_LicenseDocumentationInREADME(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newProperties()

	properties.Property("README.md must contain license reference", prop.ForAll(
		func(dummyInput int) bool {
			content, err := os.ReadFile("README.md")
			if err != nil {
				return false
			}

			readmeText := strings.ToLower(string(content))

			// Check for license mentions
			hasLicenseMention := strings.Contains(readmeText, "license") ||
				strings.Contains(readmeText, "mit") ||
				strings.Contains(readmeText, "apache") ||
				strings.Contains(readmeText, "gpl")

			// Check for LICENSE file reference
			hasLicenseFileRef := strings.Contains(readmeText, "license file") ||
				strings.Contains(readmeText, "[license]") ||
				strings.Contains(readmeText, "see the license")

			// Check for license section
			hasLicenseSection := strings.Contains(readmeText, "## license")

			return hasLicenseMention && (hasLicenseFileRef || hasLicenseSection)
		},
		gen.IntRange(1, 100), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty20_ProjectDescriptionPresence tests that the README.md contains
// a concise one-line description of the tool's purpose.
// **Validates: Requirements 8.1**
func TestProperty20_ProjectDescriptionPresence(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newProperties()

	properties.Property("README.md must contain a concise project description", prop.ForAll(
		func(dummyInput int) bool {
			content, err := os.ReadFile("README.md")
			if err != nil {
				return false
			}

			readmeText := string(content)
			lines := strings.Split(readmeText, "\n")

			// Check first few lines for a description
			// Typically, the description is in the first 10 lines
			hasDescription := false
			for i := 0; i < len(lines) && i < 10; i++ {
				line := strings.TrimSpace(lines[i])
				// Look for a description that mentions what the tool does
				if strings.Contains(strings.ToLower(line), "diff") &&
					(strings.Contains(strings.ToLower(line), "yaml") ||
						strings.Contains(strings.ToLower(line), "tool")) {
					hasDescription = true
					break
				}
			}

			// Also check for title and subtitle pattern
			hasTitle := strings.Contains(readmeText, "# diffyml")

			return hasDescription && hasTitle
		},
		gen.IntRange(1, 100), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty21_InstallationVerificationInstructions tests that the README.md
// includes instructions for verifying the installation.
// **Validates: Requirements 8.5**
func TestProperty21_InstallationVerificationInstructions(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newProperties()

	properties.Property("README.md must include installation verification instructions", prop.ForAll(
		func(dummyInput int) bool {
			content, err := os.ReadFile("README.md")
			if err != nil {
				return false
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

			// Check for installation verification instructions
			hasVerificationInstructions := hasVerificationSection ||
				(hasVersionCheck || hasHelpCheck)

			return hasVerificationInstructions
		},
		gen.IntRange(1, 100), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty12_READMECompleteness_DetailedValidation performs more detailed
// validation of README sections to ensure comprehensive documentation.
// **Validates: Requirements 5.1, 5.5**
func TestProperty12_READMECompleteness_DetailedValidation(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newProperties()

	properties.Property("README.md must have comprehensive documentation structure", prop.ForAll(
		func(dummyInput int) bool {
			content, err := os.ReadFile("README.md")
			if err != nil {
				return false
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

			// At least 4 out of 5 sections should be present
			hasSufficientSections := sectionsFound >= 4

			// Check for code examples (bash blocks)
			codeBlockPattern := regexp.MustCompile("```bash")
			hasCodeExamples := codeBlockPattern.MatchString(readmeText)

			// Check for feature list or key features
			hasFeatures := strings.Contains(readmeText, "Features") ||
				strings.Contains(readmeText, "Key Features")

			// Ensure README is not empty and has substantial content
			hasSubstantialContent := len(readmeText) > 1000

			return hasSufficientSections && hasCodeExamples && hasFeatures && hasSubstantialContent
		},
		gen.IntRange(1, 100), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
