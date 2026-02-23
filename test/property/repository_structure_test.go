package property

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestProperty16_MainPackageLocation tests that the main.go file exists in
// the repository root directory (not in a subdirectory).
// **Validates: Requirements 6.1**
func TestProperty16_MainPackageLocation(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newProperties()

	properties.Property("main.go must exist in repository root", prop.ForAll(
		func(dummyInput int) bool {
			// Check if main.go exists in the current directory (repository root)
			info, err := os.Stat("main.go")
			if err != nil {
				return false
			}

			// Verify it's a regular file
			if !info.Mode().IsRegular() {
				return false
			}

			// Verify the file is in the repository root (not in a subdirectory)
			absPath, err := filepath.Abs("main.go")
			if err != nil {
				return false
			}

			// Get the current working directory (should be repository root)
			cwd, err := os.Getwd()
			if err != nil {
				return false
			}

			// Verify main.go is directly in the repository root
			expectedPath := filepath.Join(cwd, "main.go")
			if absPath != expectedPath {
				return false
			}

			// Verify the file contains a main package declaration
			content, err := os.ReadFile("main.go")
			if err != nil {
				return false
			}

			// Parse the file to verify it's a valid Go file with main package
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "main.go", content, parser.PackageClauseOnly)
			if err != nil {
				return false
			}

			// Verify the package name is "main"
			if file.Name.Name != "main" {
				return false
			}

			return true
		},
		gen.IntRange(1, 100), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty16_MainPackageLocation_WithFunctionCheck tests that main.go
// contains a main function as expected for an executable.
// **Validates: Requirements 6.1**
func TestProperty16_MainPackageLocation_WithFunctionCheck(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newProperties()

	properties.Property("main.go must contain a main function", prop.ForAll(
		func(dummyInput int) bool {
			// Read main.go
			content, err := os.ReadFile("main.go")
			if err != nil {
				return false
			}

			// Parse the file to check for main function
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "main.go", content, parser.ParseComments)
			if err != nil {
				return false
			}

			// Look for main function
			hasMainFunc := false
			for _, decl := range file.Decls {
				if funcDecl, ok := decl.(*ast.FuncDecl); ok {
					if funcDecl.Name.Name == "main" {
						hasMainFunc = true
						break
					}
				}
			}

			// Alternative: check for "func main()" in the content
			if !hasMainFunc {
				hasMainFunc = strings.Contains(string(content), "func main()")
			}

			return hasMainFunc
		},
		gen.IntRange(1, 100), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty17_LibraryCodeOrganization tests that library code is organized
// under the pkg/ directory and contains at least one Go package.
// **Validates: Requirements 6.2**
func TestProperty17_LibraryCodeOrganization(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newProperties()

	properties.Property("library code must be organized under pkg/ directory", prop.ForAll(
		func(dummyInput int) bool {
			// Check if pkg/ directory exists
			pkgInfo, err := os.Stat("pkg")
			if err != nil {
				return false
			}

			// Verify it's a directory
			if !pkgInfo.IsDir() {
				return false
			}

			// Check that pkg/ contains at least one subdirectory (package)
			entries, err := os.ReadDir("pkg")
			if err != nil {
				return false
			}

			hasPackage := false
			for _, entry := range entries {
				if entry.IsDir() {
					// Check if this directory contains Go files
					pkgPath := filepath.Join("pkg", entry.Name())
					pkgEntries, err := os.ReadDir(pkgPath)
					if err != nil {
						continue
					}

					for _, pkgEntry := range pkgEntries {
						if !pkgEntry.IsDir() && strings.HasSuffix(pkgEntry.Name(), ".go") {
							hasPackage = true
							break
						}
					}

					if hasPackage {
						break
					}
				}
			}

			return hasPackage
		},
		gen.IntRange(1, 100), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty17_LibraryCodeOrganization_WithPackageValidation tests that
// packages under pkg/ are valid Go packages.
// **Validates: Requirements 6.2**
func TestProperty17_LibraryCodeOrganization_WithPackageValidation(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newProperties()

	properties.Property("pkg/ directory must contain valid Go packages", prop.ForAll(
		func(dummyInput int) bool {
			// Walk through pkg/ directory
			hasValidPackage := false

			err := filepath.Walk("pkg", func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				// Skip the pkg directory itself
				if path == "pkg" {
					return nil
				}

				// Check if this is a directory
				if info.IsDir() {
					// Check if this directory contains Go files
					entries, err := os.ReadDir(path)
					if err != nil {
						return nil
					}

					for _, entry := range entries {
						if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
							// Found a Go file, verify it's parseable
							goFilePath := filepath.Join(path, entry.Name())
							content, err := os.ReadFile(goFilePath)
							if err != nil {
								continue
							}

							// Parse the file to verify it's valid Go code
							fset := token.NewFileSet()
							_, err = parser.ParseFile(fset, goFilePath, content, parser.PackageClauseOnly)
							if err == nil {
								hasValidPackage = true
								return filepath.SkipDir
							}
						}
					}
				}

				return nil
			})

			if err != nil {
				return false
			}

			return hasValidPackage
		},
		gen.IntRange(1, 100), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty3_GoModuleValidity tests that the go.mod file exists in the
// root directory, is parseable by the Go toolchain, and contains an explicit
// Go version directive.
// **Validates: Requirements 2.2**
func TestProperty3_GoModuleValidity(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newProperties()

	properties.Property("go.mod must exist, be valid, and contain Go version", prop.ForAll(
		func(dummyInput int) bool {
			// Check if go.mod exists in the repository root
			info, err := os.Stat("go.mod")
			if err != nil {
				return false
			}

			// Verify it's a regular file
			if !info.Mode().IsRegular() {
				return false
			}

			// Read go.mod content
			content, err := os.ReadFile("go.mod")
			if err != nil {
				return false
			}

			gomodText := string(content)

			// Check for module declaration
			hasModule := strings.Contains(gomodText, "module ")

			// Check for Go version directive
			hasGoVersion := strings.Contains(gomodText, "go ")

			// Verify go.mod is parseable by running go mod verify
			cmd := exec.Command("go", "mod", "verify")
			err = cmd.Run()
			isValid := err == nil

			return hasModule && hasGoVersion && isValid
		},
		gen.IntRange(1, 100), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty3_GoModuleValidity_WithVersionFormat tests that the Go version
// in go.mod follows the expected format.
// **Validates: Requirements 2.2**
func TestProperty3_GoModuleValidity_WithVersionFormat(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newProperties()

	properties.Property("go.mod must contain properly formatted Go version", prop.ForAll(
		func(dummyInput int) bool {
			content, err := os.ReadFile("go.mod")
			if err != nil {
				return false
			}

			gomodText := string(content)

			// Check for Go version directive with proper format
			// Format: "go X.Y" or "go X.Y.Z"
			goVersionPattern := regexp.MustCompile(`(?m)^go\s+\d+\.\d+(\.\d+)?`)
			hasValidGoVersion := goVersionPattern.MatchString(gomodText)

			return hasValidGoVersion
		},
		gen.IntRange(1, 100), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty19_ModuleNameConsistency tests that the module name in go.mod
// follows Go module naming conventions.
// **Validates: Requirements 6.5**
func TestProperty19_ModuleNameConsistency(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newProperties()

	properties.Property("go.mod module name must follow Go naming conventions", prop.ForAll(
		func(dummyInput int) bool {
			content, err := os.ReadFile("go.mod")
			if err != nil {
				return false
			}

			gomodText := string(content)

			// Extract module name
			modulePattern := regexp.MustCompile(`(?m)^module\s+([^\s]+)`)
			matches := modulePattern.FindStringSubmatch(gomodText)
			if len(matches) < 2 {
				return false
			}

			moduleName := matches[1]

			// Check if module name follows conventions:
			// 1. Domain-based path (e.g., github.com/user/repo)
			// 2. Simple name (e.g., diffyml)

			// Domain-based path pattern
			domainPattern := regexp.MustCompile(`^[a-zA-Z0-9][-a-zA-Z0-9]*\.[a-zA-Z0-9][-a-zA-Z0-9.]*(/[a-zA-Z0-9][-a-zA-Z0-9_]*)+$`)
			isDomainBased := domainPattern.MatchString(moduleName)

			// Simple name pattern (lowercase letters, numbers, hyphens, underscores)
			simplePattern := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)
			isSimpleName := simplePattern.MatchString(moduleName)

			// Module name must be non-empty and follow one of the conventions
			isValid := len(moduleName) > 0 && (isDomainBased || isSimpleName)

			return isValid
		},
		gen.IntRange(1, 100), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty19_ModuleNameConsistency_WithImportPath tests that the module
// name is consistent with import paths used in the codebase.
// **Validates: Requirements 6.5**
func TestProperty19_ModuleNameConsistency_WithImportPath(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newProperties()

	properties.Property("module name must be consistent with import paths", prop.ForAll(
		func(dummyInput int) bool {
			// Read go.mod to get module name
			content, err := os.ReadFile("go.mod")
			if err != nil {
				return false
			}

			gomodText := string(content)

			// Extract module name
			modulePattern := regexp.MustCompile(`(?m)^module\s+([^\s]+)`)
			matches := modulePattern.FindStringSubmatch(gomodText)
			if len(matches) < 2 {
				return false
			}

			moduleName := matches[1]

			// Check if main.go imports packages using the module name
			mainContent, err := os.ReadFile("main.go")
			if err != nil {
				return false
			}

			mainText := string(mainContent)

			// If the module has a domain-based name, check if imports use it
			if strings.Contains(moduleName, "/") {
				// For domain-based modules, imports should use the module name
				// Example: module github.com/user/diffyml, import "github.com/user/diffyml/pkg/diffyml"
				expectedImportPrefix := moduleName + "/"
				hasConsistentImport := strings.Contains(mainText, `"`+expectedImportPrefix) ||
					strings.Contains(mainText, `'`+expectedImportPrefix)

				// If no imports found, that's also valid (might not import internal packages)
				hasImports := strings.Contains(mainText, "import")
				if !hasImports {
					return true
				}

				return hasConsistentImport
			}

			// For simple module names, imports should use the module name as prefix
			// Example: module github.com/szhekpisov/diffyml, import "github.com/szhekpisov/diffyml/pkg/diffyml"
			expectedImportPrefix := moduleName + "/"
			hasConsistentImport := strings.Contains(mainText, `"`+expectedImportPrefix) ||
				strings.Contains(mainText, `'`+expectedImportPrefix)

			// If no imports found, that's also valid
			hasImports := strings.Contains(mainText, "import")
			if !hasImports {
				return true
			}

			return hasConsistentImport
		},
		gen.IntRange(1, 100), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty18_NoPrebuiltBinaries tests that the repository does not contain
// pre-built executable binaries in version control.
// **Validates: Requirements 6.4**
func TestProperty18_NoPrebuiltBinaries(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newProperties()

	properties.Property("repository must not contain pre-built binaries", prop.ForAll(
		func(dummyInput int) bool {
			// List of directories to exclude from the check
			excludeDirs := map[string]bool{
				".git":         true,
				".kiro":        true,
				"node_modules": true,
				"vendor":       true,
			}

			// Walk through the repository
			foundBinary := false

			err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				// Skip excluded directories
				if info.IsDir() {
					for excludeDir := range excludeDirs {
						if strings.Contains(path, excludeDir) {
							return filepath.SkipDir
						}
					}
					return nil
				}

				// Check if the file is executable
				if info.Mode()&0111 != 0 {
					// Exclude script files (shell scripts, etc.)
					if strings.HasSuffix(path, ".sh") ||
						strings.HasSuffix(path, ".bash") ||
						strings.HasSuffix(path, ".py") ||
						strings.HasSuffix(path, ".rb") ||
						strings.HasSuffix(path, ".pl") {
						return nil
					}

					// Check if it's a binary file (not a text file)
					content, err := os.ReadFile(path)
					if err != nil {
						return nil
					}

					// Binary files typically contain null bytes
					// Text files (scripts) typically don't
					if len(content) > 0 {
						// Check for null bytes (indicator of binary file)
						for _, b := range content[:minInt(len(content), 512)] {
							if b == 0 {
								foundBinary = true
								return filepath.SkipAll
							}
						}
					}
				}

				return nil
			})

			if err != nil {
				return false
			}

			// Property passes if no binaries were found
			return !foundBinary
		},
		gen.IntRange(1, 100), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty18_NoPrebuiltBinaries_WithCommonNames tests that common binary
// names are not present in the repository.
// **Validates: Requirements 6.4**
func TestProperty18_NoPrebuiltBinaries_WithCommonNames(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newProperties()

	properties.Property("repository must not contain common binary file names", prop.ForAll(
		func(dummyInput int) bool {
			// Common binary names to check for
			commonBinaryNames := []string{
				"diffyml",
				"diffyml.exe",
				"diffyml_linux",
				"diffyml_darwin",
				"diffyml_windows",
				"diffyml_test",
				"diffyml_test.exe",
			}

			// Check if any of these files exist in the repository root
			for _, binaryName := range commonBinaryNames {
				if _, err := os.Stat(binaryName); err == nil {
					// File exists - this is a violation
					return false
				}
			}

			// Also check in common build directories
			buildDirs := []string{"bin", "build", "dist", "out"}
			for _, dir := range buildDirs {
				if info, err := os.Stat(dir); err == nil && info.IsDir() {
					for _, binaryName := range commonBinaryNames {
						binaryPath := filepath.Join(dir, binaryName)
						if _, err := os.Stat(binaryPath); err == nil {
							// Binary found in build directory
							return false
						}
					}
				}
			}

			return true
		},
		gen.IntRange(1, 100), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Helper function to get minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
