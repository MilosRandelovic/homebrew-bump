package shared

import (
	"testing"

	"github.com/Masterminds/semver/v3"
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
		result := CleanVersion(test.input)
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
		result := GetVersionPrefix(test.input)
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

		// Greater than or equal (>=) tests
		{">=1.0.0", "1.0.0", true, ">= allows same version"},
		{">=1.0.0", "1.0.1", true, ">= allows patch updates"},
		{">=1.0.0", "1.1.0", true, ">= allows minor updates"},
		{">=1.0.0", "2.0.0", true, ">= allows major updates"},
		{">=1.2.3", "1.2.2", false, ">= does not allow downgrades"},
		{">=1.2.3", "1.1.9", false, ">= does not allow downgrades"},

		// Greater than (>) tests
		{">1.0.0", "1.0.0", false, "> does not allow same version"},
		{">1.0.0", "1.0.1", true, "> allows patch updates"},
		{">1.0.0", "1.1.0", true, "> allows minor updates"},
		{">1.0.0", "2.0.0", true, "> allows major updates"},
		{">1.2.3", "1.2.3", false, "> does not allow same version"},
		{">1.2.3", "1.2.2", false, "> does not allow downgrades"},

		// Less than or equal (<=) tests
		{"<=2.0.0", "2.0.0", true, "<= allows same version"},
		{"<=2.0.0", "1.9.9", true, "<= allows older versions"},
		{"<=2.0.0", "1.0.0", true, "<= allows much older versions"},
		{"<=2.0.0", "2.0.1", false, "<= does not allow newer versions"},
		{"<=2.0.0", "2.1.0", false, "<= does not allow newer versions"},
		{"<=2.0.0", "3.0.0", false, "<= does not allow newer versions"},

		// Less than (<) tests
		{"<2.0.0", "2.0.0", false, "< does not allow same version"},
		{"<2.0.0", "1.9.9", true, "< allows older versions"},
		{"<2.0.0", "1.0.0", true, "< allows much older versions"},
		{"<2.0.0", "2.0.1", false, "< does not allow newer versions"},
		{"<2.0.0", "2.1.0", false, "< does not allow newer versions"},
		{"<2.0.0", "3.0.0", false, "< does not allow newer versions"},

		// Compound constraint tests
		{">=1.0.0 <2.0.0", "1.0.0", true, "compound allows version at lower bound"},
		{">=1.0.0 <2.0.0", "1.5.0", true, "compound allows version in range"},
		{">=1.0.0 <2.0.0", "1.9.9", true, "compound allows version near upper bound"},
		{">=1.0.0 <2.0.0", "2.0.0", false, "compound excludes upper bound"},
		{">=1.0.0 <2.0.0", "0.9.9", false, "compound excludes version below lower bound"},
		{">=1.0.0 <2.0.0", "2.0.1", false, "compound excludes version above upper bound"},

		{">1.0.0 <=2.0.0", "1.0.0", false, "compound > excludes lower bound"},
		{">1.0.0 <=2.0.0", "1.0.1", true, "compound allows version above lower bound"},
		{">1.0.0 <=2.0.0", "2.0.0", true, "compound <= includes upper bound"},
		{">1.0.0 <=2.0.0", "2.0.1", false, "compound excludes version above upper bound"},

		{">=1.2.3 <1.3.0", "1.2.3", true, "compound patch range allows lower bound"},
		{">=1.2.3 <1.3.0", "1.2.5", true, "compound patch range allows patch updates"},
		{">=1.2.3 <1.3.0", "1.2.99", true, "compound patch range allows high patch"},
		{">=1.2.3 <1.3.0", "1.3.0", false, "compound patch range excludes upper bound"},
		{">=1.2.3 <1.3.0", "1.1.9", false, "compound patch range excludes older versions"},

		// Complex compound constraints
		{">0.0.0 <1.0.0", "0.9.9", true, "compound pre-1.0 range"},
		{">0.0.0 <1.0.0", "1.0.0", false, "compound pre-1.0 excludes 1.0.0"},
		{">=2.1.0 <3.0.0", "2.5.8", true, "compound major range"},
		{">=2.1.0 <3.0.0", "3.0.0", false, "compound major range excludes next major"},
	}

	for _, test := range tests {
		result := IsSemverCompatible(test.originalVersion, test.latestVersion)
		if result != test.expected {
			t.Errorf("%s: IsSemverCompatible(%s, %s) = %v, expected %v",
				test.description, test.originalVersion, test.latestVersion, result, test.expected)
		}
	}
}

func TestGetSemverChange(t *testing.T) {
	tests := []struct {
		currentVer  string
		latestVer   string
		expected    SemverChange
		description string
	}{
		// Patch changes
		{"1.0.0", "1.0.1", PatchChange, "patch update"},
		{"1.0.0", "1.0.5", PatchChange, "multiple patch updates"},
		{"^1.0.0", "1.0.1", PatchChange, "patch update with prefix"},
		{"~1.0.0", "1.0.2", PatchChange, "patch update with tilde"},

		// Minor changes
		{"1.0.0", "1.1.0", MinorChange, "minor update"},
		{"1.0.0", "1.5.0", MinorChange, "multiple minor updates"},
		{"1.2.3", "1.3.0", MinorChange, "minor update with patch reset"},
		{"^1.0.0", "1.2.0", MinorChange, "minor update with prefix"},

		// Major changes
		{"1.0.0", "2.0.0", MajorChange, "major update"},
		{"1.5.3", "3.0.0", MajorChange, "multiple major updates"},
		{"0.9.0", "1.0.0", MajorChange, "major update from 0.x"},
		{"^1.0.0", "2.1.0", MajorChange, "major update with prefix"},

		// Edge cases
		{"1.0.0", "1.0.0", PatchChange, "same version defaults to patch"},
		{"2.1.0", "1.5.0", PatchChange, "downgrade defaults to patch"},

		// With prefixes
		{">=1.0.0", "1.1.0", MinorChange, "minor with >= prefix"},
		{"~2.3.4", "3.0.0", MajorChange, "major with ~ prefix"},

		// Invalid versions - should default to patch
		{"invalid", "1.0.0", PatchChange, "invalid current version defaults to patch"},
		{"1.0.0", "invalid", PatchChange, "invalid latest version defaults to patch"},
		{"invalid", "invalid", PatchChange, "both invalid versions default to patch"},

		// Pre-release versions (should be handled by CleanVersion)
		{"1.0.0-beta", "1.1.0", MinorChange, "current pre-release, latest stable"},
		{"1.0.0", "1.1.0-alpha", MinorChange, "current stable, latest pre-release"},
		{"1.0.0-alpha", "1.1.0-beta", MinorChange, "both pre-release"},
	}

	for _, test := range tests {
		result := GetSemverChange(test.currentVer, test.latestVer)
		if result != test.expected {
			t.Errorf("%s: GetSemverChange(%s, %s) = %v, expected %v",
				test.description, test.currentVer, test.latestVer, result, test.expected)
		}
	}
}
