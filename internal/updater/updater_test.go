package updater

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Masterminds/semver/v3"
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

func TestSemverVersionParsing(t *testing.T) {
	tests := []struct {
		input       string
		expectedMaj uint64
		expectedMin uint64
		expectedPat uint64
		hasError    bool
	}{
		{"1.0.0", 1, 0, 0, false},
		{"2.3.4", 2, 3, 4, false},
		{"0.1.2", 0, 1, 2, false},
		{"10.20.30", 10, 20, 30, false},
		{"1.0.0-beta", 1, 0, 0, false},
		{"2.3.4-alpha.1", 2, 3, 4, false},
		{"1.0.0+build.1", 1, 0, 0, false},
		{"1.0.0-beta+build.1", 1, 0, 0, false},
		{"invalid", 0, 0, 0, true},
		{"1.0.x", 0, 0, 0, true},
	}

	for _, test := range tests {
		result, err := semver.NewVersion(test.input)
		if test.hasError {
			if err == nil {
				t.Errorf("semver.NewVersion(%s) expected error but got none", test.input)
			}
		} else {
			if err != nil {
				t.Errorf("semver.NewVersion(%s) unexpected error: %v", test.input, err)
			} else if result.Major() != test.expectedMaj || result.Minor() != test.expectedMin || result.Patch() != test.expectedPat {
				t.Errorf("semver.NewVersion(%s) = %d.%d.%d, expected %d.%d.%d", test.input, result.Major(), result.Minor(), result.Patch(), test.expectedMaj, test.expectedMin, test.expectedPat)
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
		{"^0.1.0", "0.2.0", false, "caret does not allow minor updates for 0.x in strict semver"},
		{"^0.1.0", "1.0.0", false, "caret does not allow major updates for 0.x"},
		{"^0.0.1", "0.0.2", false, "caret does not allow patch updates for 0.0.x in strict semver"},
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

		// Comparison operator tests
		{">=1.0.0", "1.1.0", true, ">= allows newer versions"},
		{">1.0.0", "1.1.0", true, "> allows newer versions"},
		{"<2.0.0", "1.1.0", true, "< allows older versions"},
		{"<=2.0.0", "1.1.0", true, "<= allows older/same versions"},
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
		{">1.0.0", true},
		{"<2.0.0", true},
		{"<=2.0.0", true},
		{">=1.0.0 <2.0.0", true},
		{">1.0.0 <=2.0.0", true},
		{">=1.2.3 <1.3.0", true},
		{"1.5.0", false},
		{"", false},
		{">=1.0.0 1.5.0", false}, // Mix of semantic and non-semantic
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

func (mockClient *MockRegistryClient) GetLatestVersionFromRegistry(packageName, registryURL string, verbose bool) (string, error) {
	versions := mockClient.packageVersions[packageName]
	if len(versions) == 0 {
		return "", fmt.Errorf("package not found")
	}
	return versions[len(versions)-1], nil
}

func (mockClient *MockRegistryClient) GetBothLatestVersions(packageName, constraint, registryURL string, verbose bool) (string, string, error) {
	versions := mockClient.packageVersions[packageName]
	if len(versions) == 0 {
		return "", "", fmt.Errorf("package not found")
	}
	return shared.FindBothLatestVersions(versions, constraint)
}

func (mockClient *MockRegistryClient) GetFileType() string {
	return "mock"
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

func TestConstraintMatchesNoVersions(t *testing.T) {
	// Test that when constraint matches no available versions,
	// it goes to semverSkipped instead of errors
	mockRegistry := &MockRegistryClient{
		packageVersions: map[string][]string{
			"core": {"1.0.0", "1.1.0", "1.7.0"}, // Available versions: all 1.x
		},
	}

	// Test the scenario directly using the shared function
	absoluteLatest, constraintLatest, err := mockRegistry.GetBothLatestVersions("core", "^0.0.1", "", false)
	if err == nil {
		t.Fatal("Expected error for incompatible constraint, got nil")
	}

	if !strings.Contains(err.Error(), "no versions satisfy the constraint") {
		t.Errorf("Expected 'no versions satisfy the constraint' error, got: %v", err)
	}

	// Verify that even with the error, absoluteLatest is still returned
	if absoluteLatest != "1.7.0" {
		t.Errorf("Expected absolute latest '1.7.0' even with constraint error, got '%s'", absoluteLatest)
	}

	// Verify constraintLatest is empty when no versions satisfy constraint
	if constraintLatest != "" {
		t.Errorf("Expected empty constraint latest when no versions satisfy, got '%s'", constraintLatest)
	}
}
