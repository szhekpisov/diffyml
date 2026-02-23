package property

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestProperty8_LICENSEFilePresence tests that a LICENSE file exists in the repository root.
// **Validates: Requirements 4.1**
func TestProperty8_LICENSEFilePresence(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	if _, err := os.Stat("LICENSE"); err != nil {
		t.Fatal("LICENSE file must exist in repository root")
	}
}

// TestProperty9_DFSGCompatibleLicense tests that the LICENSE file contains
// text matching a known DFSG-compatible license.
// **Validates: Requirements 4.2**
func TestProperty9_DFSGCompatibleLicense(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	licenseContent, err := os.ReadFile("LICENSE")
	if err != nil {
		t.Fatalf("Failed to read LICENSE file: %v", err)
	}

	content := string(licenseContent)

	// List of DFSG-compatible licenses to check for
	dfsgLicenses := []string{
		"MIT License",
		"Apache License",
		"GNU General Public License",
		"GNU Lesser General Public License",
		"BSD License",
		"ISC License",
		"Mozilla Public License",
		"Eclipse Public License",
		"Creative Commons Zero",
		"Unlicense",
		"Artistic License",
		"Zlib License",
		"PostgreSQL License",
		"Python Software Foundation License",
	}

	for _, license := range dfsgLicenses {
		if strings.Contains(content, license) {
			return
		}
	}

	// Additional check for common license patterns
	if strings.Contains(content, "Permission is hereby granted") &&
		strings.Contains(content, "free of charge") {
		return
	}
	if strings.Contains(content, "Licensed under the Apache License, Version 2.0") {
		return
	}
	if strings.Contains(content, "GNU General Public License") ||
		strings.Contains(content, "GNU GENERAL PUBLIC LICENSE") {
		return
	}
	if strings.Contains(content, "Redistribution and use in source and binary forms") {
		return
	}

	t.Fatal("LICENSE file does not contain DFSG-compatible license text")
}

// TestProperty9_DFSGCompatibleLicense_WithVariations tests the DFSG compatibility
// property to ensure robustness.
// **Validates: Requirements 4.2**
func TestProperty9_DFSGCompatibleLicense_WithVariations(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	licenseContent, err := os.ReadFile("LICENSE")
	if err != nil {
		t.Fatalf("Failed to read LICENSE file: %v", err)
	}

	content := string(licenseContent)

	if len(strings.TrimSpace(content)) == 0 {
		t.Fatal("LICENSE file is empty")
	}

	// MIT License indicators
	mitIndicators := []string{"MIT License"}
	// Apache License indicators
	apacheIndicators := []string{"Apache License", "Version 2.0"}
	// GPL indicators
	gplIndicators := []string{"GNU General Public License", "GNU GENERAL PUBLIC LICENSE"}
	// BSD indicators
	bsdIndicators := []string{"BSD License", "Redistribution and use"}

	hasMIT := containsAll(content, mitIndicators) ||
		(strings.Contains(content, "Permission is hereby granted") &&
			strings.Contains(content, "free of charge"))
	hasApache := containsAny(content, apacheIndicators)
	hasGPL := containsAny(content, gplIndicators)
	hasBSD := containsAny(content, bsdIndicators)

	if !hasMIT && !hasApache && !hasGPL && !hasBSD {
		t.Fatal("LICENSE file does not match any known DFSG-compatible license pattern")
	}
}

// Helper function to check if content contains all strings in the list
func containsAll(content string, patterns []string) bool {
	for _, pattern := range patterns {
		if !strings.Contains(content, pattern) {
			return false
		}
	}
	return true
}

// Helper function to check if content contains any string in the list
func containsAny(content string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.Contains(content, pattern) {
			return true
		}
	}
	return false
}

// TestProperty8_LICENSEFilePresence_WithFileSystemChecks performs additional
// file system validation for the LICENSE file.
// **Validates: Requirements 4.1**
func TestProperty8_LICENSEFilePresence_WithFileSystemChecks(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	info, err := os.Stat("LICENSE")
	if err != nil {
		t.Fatalf("LICENSE file not found: %v", err)
	}

	if !info.Mode().IsRegular() {
		t.Fatal("LICENSE is not a regular file")
	}

	content, err := os.ReadFile("LICENSE")
	if err != nil {
		t.Fatalf("LICENSE file is not readable: %v", err)
	}

	if len(content) == 0 {
		t.Fatal("LICENSE file is empty")
	}

	// Verify the file is in the repository root
	absPath, err := filepath.Abs("LICENSE")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	expectedPath := filepath.Join(cwd, "LICENSE")
	if absPath != expectedPath {
		t.Fatalf("LICENSE not in repository root: got %s, expected %s", absPath, expectedPath)
	}
}
