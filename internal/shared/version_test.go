package shared

import (
	"testing"
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
		result, err := ParseSemanticVersion(test.input)
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
		result := IsSemverCompatible(test.originalVersion, test.latestVersion)
		if result != test.expected {
			t.Errorf("%s: IsSemverCompatible(%s, %s) = %v, expected %v",
				test.description, test.originalVersion, test.latestVersion, result, test.expected)
		}
	}
}
