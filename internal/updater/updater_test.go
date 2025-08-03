package updater

import (
	"fmt"
	"testing"

	"github.com/MilosRandelovic/homebrew-bump/internal/shared"
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
		result := shared.CleanVersion(test.input)
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
		result := shared.GetVersionPrefix(test.input)
		if result != test.expected {
			t.Errorf("GetVersionPrefix(%s) = %s, expected %s", test.input, result, test.expected)
		}
	}
}

func TestParseSemanticVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected *shared.SemanticVersion
		hasError bool
	}{
		{"1.0.0", &shared.SemanticVersion{Major: 1, Minor: 0, Patch: 0}, false},
		{"2.3.4", &shared.SemanticVersion{Major: 2, Minor: 3, Patch: 4}, false},
		{"0.1.2", &shared.SemanticVersion{Major: 0, Minor: 1, Patch: 2}, false},
		{"10.20.30", &shared.SemanticVersion{Major: 10, Minor: 20, Patch: 30}, false},
		{"1.0.0-beta", &shared.SemanticVersion{Major: 1, Minor: 0, Patch: 0}, false},
		{"2.3.4-alpha.1", &shared.SemanticVersion{Major: 2, Minor: 3, Patch: 4}, false},
		{"1.0.0+build.1", &shared.SemanticVersion{Major: 1, Minor: 0, Patch: 0}, false},
		{"1.0.0-beta+build.1", &shared.SemanticVersion{Major: 1, Minor: 0, Patch: 0}, false},
		{"invalid", nil, true},
		{"1.0", nil, true},
		{"1.0.x", nil, true},
	}

	for _, test := range tests {
		result, err := shared.ParseSemanticVersion(test.input)
		if test.hasError {
			if err == nil {
				t.Errorf("ParseSemanticVersion(%s) expected error but got none", test.input)
			}
		} else {
			if err != nil {
				t.Errorf("ParseSemanticVersion(%s) unexpected error: %v", test.input, err)
			} else if result.Major != test.expected.Major || result.Minor != test.expected.Minor || result.Patch != test.expected.Patch {
				t.Errorf("ParseSemanticVersion(%s) = %+v, expected %+v", test.input, result, test.expected)
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
		{"~1.2.3", "1.2.4", true, "tilde allows patch updates"},
		{"~1.2.3", "1.3.0", false, "tilde does not allow minor updates"},
		{"~1.2.3", "2.0.0", false, "tilde does not allow major updates"},

		// Hardcoded versions
		{"1.0.0", "1.0.1", false, "hardcoded versions are not compatible"},
		{"1.0.0", "1.1.0", false, "hardcoded versions are not compatible"},

		// Pre-release versions
		{"^1.0.0", "1.1.0-beta", false, "pre-release versions are skipped"},
		{"^1.0.0", "1.1.0-alpha.1", false, "pre-release versions are skipped"},

		// Other prefixes
		{">=1.0.0", "1.1.0", false, "other prefixes are conservative"},
		{">1.0.0", "1.1.0", false, "other prefixes are conservative"},
		{"<2.0.0", "1.1.0", false, "other prefixes are conservative"},
		{"<=2.0.0", "1.1.0", false, "other prefixes are conservative"},
	}

	for _, test := range tests {
		result := shared.IsSemverCompatible(test.originalVersion, test.latestVersion)
		if result != test.expected {
			t.Errorf("%s: IsSemverCompatible(%s, %s) = %v, expected %v",
				test.description, test.originalVersion, test.latestVersion, result, test.expected)
		}
	}
}

func TestSemverSkippedTracking(t *testing.T) {
	// Test that the SemverSkipped field is properly populated
	// This is a basic test to ensure the struct and tracking work
	result := &shared.CheckResult{
		SemverSkipped: []shared.SemverSkipped{
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

func TestFindBothLatestVersions(t *testing.T) {
	tests := []struct {
		name                  string
		versions              []string
		constraint            string
		expectedAbsolute      string
		expectedConstraint    string
		expectConstraintError bool
		description           string
	}{
		{
			name:               "caret constraint with newer major available",
			versions:           []string{"22.15.0", "22.16.0", "22.17.0", "24.0.0", "24.1.0"},
			constraint:         "^22.16.0",
			expectedAbsolute:   "24.1.0",
			expectedConstraint: "22.17.0",
			description:        "should find 24.1.0 as absolute latest and 22.17.0 as constraint-satisfying latest",
		},
		{
			name:               "tilde constraint",
			versions:           []string{"1.2.0", "1.2.5", "1.3.0", "2.0.0"},
			constraint:         "~1.2.3",
			expectedAbsolute:   "2.0.0",
			expectedConstraint: "1.2.5",
			description:        "should find 2.0.0 as absolute latest and 1.2.5 as constraint-satisfying latest",
		},
		{
			name:                  "no compatible versions",
			versions:              []string{"1.0.0", "1.1.0", "1.2.0"},
			constraint:            "^2.0.0",
			expectedAbsolute:      "1.2.0",
			expectConstraintError: true,
			description:           "should find absolute latest but error for constraint",
		},
		{
			name:               "absolute and constraint latest are same",
			versions:           []string{"1.0.0", "1.1.0", "1.2.0"},
			constraint:         "^1.0.0",
			expectedAbsolute:   "1.2.0",
			expectedConstraint: "1.2.0",
			description:        "should find same version for both when constraint allows latest",
		},
		{
			name:               "pre-release versions should be filtered out",
			versions:           []string{"22.15.0", "22.16.0", "22.17.0", "24.0.0", "24.1.0-alpha", "24.1.0-beta", "24.2.0-rc"},
			constraint:         "^22.16.0",
			expectedAbsolute:   "24.0.0",
			expectedConstraint: "22.17.0",
			description:        "should filter out alpha/beta/rc versions from both absolute and constraint results",
		},
		{
			name:                  "only pre-release versions available should error",
			versions:              []string{"1.0.0-alpha", "1.0.0-beta", "1.1.0-rc"},
			constraint:            "^1.0.0",
			expectedAbsolute:      "",
			expectedConstraint:    "",
			expectConstraintError: true,
			description:           "should error when only pre-release versions are available",
		},
		{
			name:               "mixed stable and pre-release with constraint match",
			versions:           []string{"1.0.0", "1.1.0-alpha", "1.1.0", "1.2.0-beta", "2.0.0-alpha", "2.0.0"},
			constraint:         "^1.0.0",
			expectedAbsolute:   "2.0.0",
			expectedConstraint: "1.1.0",
			description:        "should find stable versions ignoring pre-releases",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			absolute, constraint, err := shared.FindBothLatestVersions(test.versions, test.constraint)

			if test.expectConstraintError {
				if err == nil {
					t.Errorf("%s: expected error for constraint but got none", test.description)
				}
				if test.expectedAbsolute != "" && absolute != test.expectedAbsolute {
					t.Errorf("%s: absolute latest = %s, expected %s", test.description, absolute, test.expectedAbsolute)
				}
				return
			}

			if err != nil {
				t.Errorf("%s: unexpected error: %v", test.description, err)
				return
			}

			if absolute != test.expectedAbsolute {
				t.Errorf("%s: absolute latest = %s, expected %s", test.description, absolute, test.expectedAbsolute)
			}

			if constraint != test.expectedConstraint {
				t.Errorf("%s: constraint latest = %s, expected %s", test.description, constraint, test.expectedConstraint)
			}
		})
	}
}

func TestHasSemanticPrefix(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"^1.0.0", true},
		{"~2.3.4", true},
		{">=3.0.0", true},
		{"1.5.0", false},
		{">1.0.0", false},
		{"<2.0.0", false},
		{"<=2.0.0", false},
		{"", false},
	}

	for _, test := range tests {
		result := shared.HasSemanticPrefix(test.input)
		if result != test.expected {
			t.Errorf("HasSemanticPrefix(%s) = %v, expected %v", test.input, result, test.expected)
		}
	}
}

// MockRegistryClient for testing
type MockRegistryClient struct {
	packageVersions map[string][]string
}

func (mockClient *MockRegistryClient) GetLatestVersionFromRegistry(packageName string) (string, error) {
	versions := mockClient.packageVersions[packageName]
	if len(versions) == 0 {
		return "", fmt.Errorf("package not found")
	}
	return versions[len(versions)-1], nil
}

func (mockClient *MockRegistryClient) GetBothLatestVersions(packageName, constraint string) (string, string, error) {
	versions := mockClient.packageVersions[packageName]
	if len(versions) == 0 {
		return "", "", fmt.Errorf("package not found")
	}
	return shared.FindBothLatestVersions(versions, constraint)
}

func TestCheckForUpdatesIntegration(t *testing.T) {
	// Mock registry client with pre-release versions
	mockRegistry := &MockRegistryClient{
		packageVersions: map[string][]string{
			"@types/node": {"22.15.0", "22.16.0", "22.17.0", "24.0.0-alpha", "24.0.0-beta", "24.1.0"},
			"typescript":  {"5.8.0", "5.8.3", "5.9.0", "5.9.2"},
		},
	}

	// Test that pre-release versions are filtered
	absolute, constraint, err := shared.FindBothLatestVersions(
		mockRegistry.packageVersions["@types/node"],
		"^22.16.0",
	)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if absolute != "24.1.0" {
		t.Errorf("Expected absolute latest 24.1.0, got %s", absolute)
	}

	if constraint != "22.17.0" {
		t.Errorf("Expected constraint latest 22.17.0, got %s", constraint)
	}

	// Test that we would report semver skipped when absolute != constraint
	shouldSkip := absolute != constraint
	if !shouldSkip {
		t.Errorf("Expected to skip major version, but absolute == constraint")
	}
}
