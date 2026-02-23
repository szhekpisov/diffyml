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
)

// TestProperty16_MainPackageLocation tests that the main.go file exists in
// the repository root directory (not in a subdirectory).
// **Validates: Requirements 6.1**
func TestProperty16_MainPackageLocation(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	info, err := os.Stat("main.go")
	if err != nil {
		t.Fatalf("main.go not found: %v", err)
	}

	if !info.Mode().IsRegular() {
		t.Fatal("main.go is not a regular file")
	}

	absPath, err := filepath.Abs("main.go")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	expectedPath := filepath.Join(cwd, "main.go")
	if absPath != expectedPath {
		t.Fatalf("main.go not in repository root: got %s, expected %s", absPath, expectedPath)
	}

	content, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("Failed to read main.go: %v", err)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", content, parser.PackageClauseOnly)
	if err != nil {
		t.Fatalf("Failed to parse main.go: %v", err)
	}

	if file.Name.Name != "main" {
		t.Fatalf("main.go has package %q, expected \"main\"", file.Name.Name)
	}
}

// TestProperty16_MainPackageLocation_WithFunctionCheck tests that main.go
// contains a main function as expected for an executable.
// **Validates: Requirements 6.1**
func TestProperty16_MainPackageLocation_WithFunctionCheck(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	content, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("Failed to read main.go: %v", err)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", content, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse main.go: %v", err)
	}

	hasMainFunc := false
	for _, decl := range file.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			if funcDecl.Name.Name == "main" {
				hasMainFunc = true
				break
			}
		}
	}

	if !hasMainFunc {
		// Fallback: check string content
		if !strings.Contains(string(content), "func main()") {
			t.Fatal("main.go does not contain a main function")
		}
	}
}

// TestProperty17_LibraryCodeOrganization tests that library code is organized
// under the pkg/ directory and contains at least one Go package.
// **Validates: Requirements 6.2**
func TestProperty17_LibraryCodeOrganization(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	pkgInfo, err := os.Stat("pkg")
	if err != nil {
		t.Fatalf("pkg/ directory not found: %v", err)
	}

	if !pkgInfo.IsDir() {
		t.Fatal("pkg is not a directory")
	}

	entries, err := os.ReadDir("pkg")
	if err != nil {
		t.Fatalf("Failed to read pkg/ directory: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			pkgPath := filepath.Join("pkg", entry.Name())
			pkgEntries, err := os.ReadDir(pkgPath)
			if err != nil {
				continue
			}

			for _, pkgEntry := range pkgEntries {
				if !pkgEntry.IsDir() && strings.HasSuffix(pkgEntry.Name(), ".go") {
					return // Found at least one Go package
				}
			}
		}
	}

	t.Fatal("pkg/ directory does not contain any Go packages")
}

// TestProperty17_LibraryCodeOrganization_WithPackageValidation tests that
// packages under pkg/ are valid Go packages.
// **Validates: Requirements 6.2**
func TestProperty17_LibraryCodeOrganization_WithPackageValidation(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	hasValidPackage := false

	err := filepath.Walk("pkg", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path == "pkg" {
			return nil
		}

		if info.IsDir() {
			entries, err := os.ReadDir(path)
			if err != nil {
				return nil
			}

			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
					goFilePath := filepath.Join(path, entry.Name())
					content, err := os.ReadFile(goFilePath)
					if err != nil {
						continue
					}

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
		t.Fatalf("Failed to walk pkg/ directory: %v", err)
	}

	if !hasValidPackage {
		t.Fatal("pkg/ directory does not contain any valid Go packages")
	}
}

// TestProperty3_GoModuleValidity tests that the go.mod file exists in the
// root directory, is parseable by the Go toolchain, and contains an explicit
// Go version directive.
// **Validates: Requirements 2.2**
func TestProperty3_GoModuleValidity(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	info, err := os.Stat("go.mod")
	if err != nil {
		t.Fatalf("go.mod not found: %v", err)
	}

	if !info.Mode().IsRegular() {
		t.Fatal("go.mod is not a regular file")
	}

	content, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatalf("Failed to read go.mod: %v", err)
	}

	gomodText := string(content)

	if !strings.Contains(gomodText, "module ") {
		t.Fatal("go.mod missing module declaration")
	}

	if !strings.Contains(gomodText, "go ") {
		t.Fatal("go.mod missing Go version directive")
	}

	cmd := exec.Command("go", "mod", "verify")
	if err := cmd.Run(); err != nil {
		t.Fatalf("go mod verify failed: %v", err)
	}
}

// TestProperty3_GoModuleValidity_WithVersionFormat tests that the Go version
// in go.mod follows the expected format.
// **Validates: Requirements 2.2**
func TestProperty3_GoModuleValidity_WithVersionFormat(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	content, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatalf("Failed to read go.mod: %v", err)
	}

	goVersionPattern := regexp.MustCompile(`(?m)^go\s+\d+\.\d+(\.\d+)?`)
	if !goVersionPattern.MatchString(string(content)) {
		t.Fatal("go.mod does not contain properly formatted Go version")
	}
}

// TestProperty19_ModuleNameConsistency tests that the module name in go.mod
// follows Go module naming conventions.
// **Validates: Requirements 6.5**
func TestProperty19_ModuleNameConsistency(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	content, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatalf("Failed to read go.mod: %v", err)
	}

	modulePattern := regexp.MustCompile(`(?m)^module\s+([^\s]+)`)
	matches := modulePattern.FindStringSubmatch(string(content))
	if len(matches) < 2 {
		t.Fatal("go.mod missing module declaration")
	}

	moduleName := matches[1]

	// Domain-based path pattern
	domainPattern := regexp.MustCompile(`^[a-zA-Z0-9][-a-zA-Z0-9]*\.[a-zA-Z0-9][-a-zA-Z0-9.]*(/[a-zA-Z0-9][-a-zA-Z0-9_]*)+$`)
	isDomainBased := domainPattern.MatchString(moduleName)

	// Simple name pattern
	simplePattern := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)
	isSimpleName := simplePattern.MatchString(moduleName)

	if !isDomainBased && !isSimpleName {
		t.Fatalf("Module name %q does not follow Go naming conventions", moduleName)
	}
}

// TestProperty19_ModuleNameConsistency_WithImportPath tests that the module
// name is consistent with import paths used in the codebase.
// **Validates: Requirements 6.5**
func TestProperty19_ModuleNameConsistency_WithImportPath(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	content, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatalf("Failed to read go.mod: %v", err)
	}

	modulePattern := regexp.MustCompile(`(?m)^module\s+([^\s]+)`)
	matches := modulePattern.FindStringSubmatch(string(content))
	if len(matches) < 2 {
		t.Fatal("go.mod missing module declaration")
	}

	moduleName := matches[1]

	mainContent, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("Failed to read main.go: %v", err)
	}

	mainText := string(mainContent)

	if !strings.Contains(mainText, "import") {
		return // No imports, that's valid
	}

	expectedImportPrefix := moduleName + "/"
	if !strings.Contains(mainText, `"`+expectedImportPrefix) &&
		!strings.Contains(mainText, `'`+expectedImportPrefix) {
		t.Fatalf("main.go does not import packages using module name %q", moduleName)
	}
}

// TestProperty18_NoPrebuiltBinaries tests that the repository does not contain
// pre-built executable binaries in version control.
// **Validates: Requirements 6.4**
func TestProperty18_NoPrebuiltBinaries(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	excludeDirs := map[string]bool{
		".git":         true,
		".kiro":        true,
		".claude":      true,
		"node_modules": true,
		"vendor":       true,
	}

	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			for excludeDir := range excludeDirs {
				if strings.Contains(path, excludeDir) {
					return filepath.SkipDir
				}
			}
			return nil
		}

		if info.Mode()&0111 != 0 {
			// Exclude script files
			if strings.HasSuffix(path, ".sh") ||
				strings.HasSuffix(path, ".bash") ||
				strings.HasSuffix(path, ".py") ||
				strings.HasSuffix(path, ".rb") ||
				strings.HasSuffix(path, ".pl") {
				return nil
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			if len(content) > 0 {
				for _, b := range content[:minInt(len(content), 512)] {
					if b == 0 {
						t.Fatalf("Found pre-built binary: %s", path)
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		t.Fatalf("Failed to walk repository: %v", err)
	}
}

// TestProperty18_NoPrebuiltBinaries_WithCommonNames tests that common binary
// names are not present in the repository.
// **Validates: Requirements 6.4**
func TestProperty18_NoPrebuiltBinaries_WithCommonNames(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	commonBinaryNames := []string{
		"diffyml",
		"diffyml.exe",
		"diffyml_linux",
		"diffyml_darwin",
		"diffyml_windows",
		"diffyml_test",
		"diffyml_test.exe",
	}

	for _, binaryName := range commonBinaryNames {
		if _, err := os.Stat(binaryName); err == nil {
			t.Fatalf("Found pre-built binary in repository root: %s", binaryName)
		}
	}

	buildDirs := []string{"bin", "build", "dist", "out"}
	for _, dir := range buildDirs {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			for _, binaryName := range commonBinaryNames {
				binaryPath := filepath.Join(dir, binaryName)
				if _, err := os.Stat(binaryPath); err == nil {
					t.Fatalf("Found pre-built binary in %s: %s", dir, binaryName)
				}
			}
		}
	}
}

// Helper function to get minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
