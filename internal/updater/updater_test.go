package updater

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
		result := cleanVersion(test.input)
		if result != test.expected {
			t.Errorf("cleanVersion(%s) = %s, expected %s", test.input, result, test.expected)
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
