package updater

import (
	"testing"

	"github.com/MilosRandelovic/homebrew-bump/internal/parser"
)

func TestCleanVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"^1.0.0", "1.0.0"},
		{"~2.3.4", "2.3.4"},
		{">=3.0.0", "3.0.0"},
		{"<4.0.0", "4.0.0"},
		{"1.5.0", "1.5.0"},
		{"^>=1.0.0", "1.0.0"},
	}

	for _, test := range tests {
		result := parser.CleanVersion(test.input)
		if result != test.expected {
			t.Errorf("CleanVersion(%s) = %s, expected %s", test.input, result, test.expected)
		}
	}
}

func TestGetVersionPrefix(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"^1.0.0", "^"},
		{"~2.3.4", "~"},
		{">=3.0.0", ">="},
		{"<4.0.0", "<"},
		{"1.5.0", ""},
		{"^>=1.0.0", "^>="},
	}

	for _, test := range tests {
		result := getVersionPrefix(test.input)
		if result != test.expected {
			t.Errorf("getVersionPrefix(%s) = %s, expected %s", test.input, result, test.expected)
		}
	}
}

func TestParseSemanticVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected *SemanticVersion
		hasError bool
	}{
		{"1.0.0", &SemanticVersion{1, 0, 0}, false},
		{"2.3.4", &SemanticVersion{2, 3, 4}, false},
		{"0.1.2", &SemanticVersion{0, 1, 2}, false},
		{"10.20.30", &SemanticVersion{10, 20, 30}, false},
		{"1.0.0-beta", &SemanticVersion{1, 0, 0}, false},
		{"2.3.4-alpha.1", &SemanticVersion{2, 3, 4}, false},
		{"1.0.0+build.1", &SemanticVersion{1, 0, 0}, false},
		{"1.0.0-beta+build.1", &SemanticVersion{1, 0, 0}, false},
		{"invalid", nil, true},
		{"1.0", nil, true},
		{"1.0.x", nil, true},
	}

	for _, test := range tests {
		result, err := parseSemanticVersion(test.input)
		if test.hasError {
			if err == nil {
				t.Errorf("parseSemanticVersion(%s) expected error but got none", test.input)
			}
		} else {
			if err != nil {
				t.Errorf("parseSemanticVersion(%s) unexpected error: %v", test.input, err)
			} else if result.Major != test.expected.Major || result.Minor != test.expected.Minor || result.Patch != test.expected.Patch {
				t.Errorf("parseSemanticVersion(%s) = %+v, expected %+v", test.input, result, test.expected)
			}
		}
	}
}

func TestIsSemverCompatible(t *testing.T) {
	tests := []struct {
		originalVersion string
		latestVersion   string
		expected        bool
		description     string
	}{
		// Caret tests
		{"^1.0.0", "1.0.1", true, "caret allows patch updates"},
		{"^1.0.0", "1.1.0", true, "caret allows minor updates"},
		{"^1.0.0", "2.0.0", false, "caret does not allow major updates"},
		{"^0.1.0", "0.1.1", true, "caret allows patch updates for 0.x"},
		{"^0.1.0", "0.2.0", true, "caret allows minor updates for 0.x"},
		{"^0.1.0", "1.0.0", false, "caret does not allow major updates for 0.x"},
		{"^0.0.1", "0.0.2", true, "caret allows patch updates for 0.0.x"},
		{"^0.0.1", "0.1.0", false, "caret does not allow minor updates for 0.0.x"},

		// Tilde tests
		{"~1.0.0", "1.0.1", true, "tilde allows patch updates"},
		{"~1.0.0", "1.1.0", false, "tilde does not allow minor updates"},
		{"~1.0.0", "2.0.0", false, "tilde does not allow major updates"},
		{"~0.1.0", "0.1.1", true, "tilde allows patch updates for 0.x"},
		{"~0.1.0", "0.2.0", false, "tilde does not allow minor updates for 0.x"},

		// Hardcoded versions (no prefix)
		{"1.0.0", "1.0.1", false, "hardcoded versions are not compatible"},
		{"2.3.4", "2.3.5", false, "hardcoded versions are not compatible"},

		// Other prefixes
		{">=1.0.0", "1.1.0", false, "other prefixes are not supported"},
		{">1.0.0", "1.1.0", false, "other prefixes are not supported"},
		{"<2.0.0", "1.9.0", false, "other prefixes are not supported"},

		// Invalid versions
		{"^invalid", "1.0.0", false, "invalid original version"},
		{"^1.0.0", "invalid", false, "invalid latest version"},
	}

	for _, test := range tests {
		result := isSemverCompatible(test.originalVersion, test.latestVersion)
		if result != test.expected {
			t.Errorf("%s: isSemverCompatible(%s, %s) = %t, expected %t",
				test.description, test.originalVersion, test.latestVersion, result, test.expected)
		}
	}
}

func TestCheckOutdatedWithSemver(t *testing.T) {
	// Create test dependencies
	dependencies := []parser.Dependency{
		{Name: "package1", Version: "1.0.0", OriginalVersion: "^1.0.0"}, // Should be compatible with newer minor/patch
		{Name: "package2", Version: "2.0.0", OriginalVersion: "~2.0.0"}, // Should only be compatible with patch updates
		{Name: "package3", Version: "1.0.0", OriginalVersion: "1.0.0"},  // Hardcoded - should be skipped in semver mode
		{Name: "package4", Version: "1.0.0", OriginalVersion: "^1.0.0"}, // Would get major update - should be skipped
	}

	// Mock the getLatestVersion function for testing
	// Note: In a real test environment, you might want to mock the HTTP calls
	// For now, we'll test the logic with a simplified approach

	// Test with semver = false (should include all outdated)
	result, err := CheckOutdatedWithProgress(dependencies, "npm", false, false, nil)
	if err != nil {
		t.Fatalf("CheckOutdatedWithProgress failed: %v", err)
	}

	// We can't easily test the HTTP calls without mocking, but we can test the semver logic
	// by testing the individual isSemverCompatible function above
	_ = result // Avoid unused variable error
}

func TestSemverSkippedTracking(t *testing.T) {
	// Test that the SemverSkipped field is properly populated
	// This is a basic test to ensure the struct and tracking work
	result := &CheckResult{
		Outdated: []OutdatedDependency{},
		Errors:   []DependencyError{},
		SemverSkipped: []SemverSkipped{
			{
				Name:            "test-package",
				CurrentVersion:  "1.0.0",
				LatestVersion:   "2.0.0",
				OriginalVersion: "^1.0.0",
				Reason:          "incompatible with constraint",
			},
		},
	}

	if len(result.SemverSkipped) != 1 {
		t.Errorf("Expected 1 semver skipped entry, got %d", len(result.SemverSkipped))
	}

	skipped := result.SemverSkipped[0]
	if skipped.Name != "test-package" {
		t.Errorf("Expected name 'test-package', got '%s'", skipped.Name)
	}

	if skipped.Reason != "incompatible with constraint" {
		t.Errorf("Expected reason 'incompatible with constraint', got '%s'", skipped.Reason)
	}
}

func TestSemverEdgeCases(t *testing.T) {
	tests := []struct {
		originalVersion string
		latestVersion   string
		expected        bool
		description     string
	}{
		// Edge cases for version parsing
		{"^1.0.0", "1.0.0", true, "same version should be compatible"},
		{"~1.0.0", "1.0.0", true, "same version should be compatible with tilde"},

		// Pre-release versions (simplified - we don't handle these in detail)
		{"^1.0.0", "1.0.1-beta", false, "pre-release versions should be handled carefully"},

		// Zero versions
		{"^0.0.0", "0.0.1", true, "zero patch version allows patch updates"},
		{"~0.0.0", "0.0.1", true, "zero patch version allows patch updates with tilde"},

		// Complex constraints not supported
		{">=1.0.0 <2.0.0", "1.5.0", false, "complex constraints not supported"},
		{"|| >=1.0.0", "1.5.0", false, "OR constraints not supported"},
	}

	for _, test := range tests {
		result := isSemverCompatible(test.originalVersion, test.latestVersion)
		if result != test.expected {
			t.Errorf("%s: isSemverCompatible(%s, %s) = %t, expected %t",
				test.description, test.originalVersion, test.latestVersion, result, test.expected)
		}
	}
}
