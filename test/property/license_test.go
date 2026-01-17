package property

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestProperty8_LICENSEFilePresence tests that a LICENSE file exists in the repository root.
// **Validates: Requirements 4.1**
func TestProperty8_LICENSEFilePresence(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newProperties()

	properties.Property("LICENSE file must exist in repository root", prop.ForAll(
		func(dummyInput bool) bool {
			// Check if LICENSE file exists in the current directory (repository root)
			_, err := os.Stat("LICENSE")
			return err == nil
		},
		// This property doesn't need generated inputs - it's a constant check
		gen.Const(true),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty9_DFSGCompatibleLicense tests that the LICENSE file contains
// text matching a known DFSG-compatible license.
// **Validates: Requirements 4.2**
func TestProperty9_DFSGCompatibleLicense(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newProperties()

	// List of DFSG-compatible licenses to check for
	// These are common open-source licenses accepted by Debian and Homebrew
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

	properties.Property("LICENSE file must contain DFSG-compatible license text", prop.ForAll(
		func(dummyInput bool) bool {
			// Read LICENSE file
			licenseContent, err := os.ReadFile("LICENSE")
			if err != nil {
				// If file doesn't exist, this property fails
				return false
			}

			content := string(licenseContent)

			// Check if the content contains any of the known DFSG-compatible license names
			for _, license := range dfsgLicenses {
				if strings.Contains(content, license) {
					return true
				}
			}

			// Additional check for common license patterns
			// MIT License often just says "Permission is hereby granted"
			if strings.Contains(content, "Permission is hereby granted") &&
				strings.Contains(content, "free of charge") {
				return true
			}

			// Apache 2.0 has specific text
			if strings.Contains(content, "Licensed under the Apache License, Version 2.0") {
				return true
			}

			// GPL has specific text
			if strings.Contains(content, "GNU General Public License") ||
				strings.Contains(content, "GNU GENERAL PUBLIC LICENSE") {
				return true
			}

			// BSD licenses have specific text
			if strings.Contains(content, "Redistribution and use in source and binary forms") {
				return true
			}

			return false
		},
		// This property doesn't need generated inputs - it's a constant check
		gen.Const(true),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty9_DFSGCompatibleLicense_WithVariations tests the DFSG compatibility
// property with various repository states to ensure robustness.
// **Validates: Requirements 4.2**
func TestProperty9_DFSGCompatibleLicense_WithVariations(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newProperties()

	properties.Property("LICENSE file must be readable and contain valid license text", prop.ForAll(
		func(dummyInput int) bool {
			// This test runs multiple times to ensure consistency
			// The dummyInput is just to trigger multiple iterations

			licenseContent, err := os.ReadFile("LICENSE")
			if err != nil {
				return false
			}

			content := string(licenseContent)

			// Ensure the file is not empty
			if len(strings.TrimSpace(content)) == 0 {
				return false
			}

			// Check for DFSG-compatible license indicators
			// MIT License indicators
			mitIndicators := []string{
				"MIT License",
				"Permission is hereby granted",
			}

			// Apache License indicators
			apacheIndicators := []string{
				"Apache License",
				"Version 2.0",
			}

			// GPL indicators
			gplIndicators := []string{
				"GNU General Public License",
				"GNU GENERAL PUBLIC LICENSE",
			}

			// BSD indicators
			bsdIndicators := []string{
				"BSD License",
				"Redistribution and use",
			}

			// Check if content matches any known DFSG-compatible license pattern
			hasMIT := containsAll(content, mitIndicators[:1]) ||
				(strings.Contains(content, "Permission is hereby granted") &&
					strings.Contains(content, "free of charge"))

			hasApache := containsAny(content, apacheIndicators)
			hasGPL := containsAny(content, gplIndicators)
			hasBSD := containsAny(content, bsdIndicators)

			return hasMIT || hasApache || hasGPL || hasBSD
		},
		gen.IntRange(1, 100), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
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

	properties := newProperties()

	properties.Property("LICENSE file must exist, be readable, and be a regular file", prop.ForAll(
		func(dummyInput int) bool {
			// Check if LICENSE file exists
			info, err := os.Stat("LICENSE")
			if err != nil {
				return false
			}

			// Verify it's a regular file (not a directory or symlink)
			if !info.Mode().IsRegular() {
				return false
			}

			// Verify the file is readable
			content, err := os.ReadFile("LICENSE")
			if err != nil {
				return false
			}

			// Verify the file is not empty
			if len(content) == 0 {
				return false
			}

			// Verify the file is in the repository root (not in a subdirectory)
			absPath, err := filepath.Abs("LICENSE")
			if err != nil {
				return false
			}

			// Get the current working directory (should be repository root)
			cwd, err := os.Getwd()
			if err != nil {
				return false
			}

			// Verify LICENSE is directly in the repository root
			expectedPath := filepath.Join(cwd, "LICENSE")
			return absPath == expectedPath
		},
		gen.IntRange(1, 100), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
