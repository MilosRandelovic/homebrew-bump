package dart

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MilosRandelovic/homebrew-bump/internal/shared"
)

func TestParsePubspecYaml(t *testing.T) {
	// Create a temporary pubspec.yaml file
	tempDir := t.TempDir()
	pubspecPath := filepath.Join(tempDir, "pubspec.yaml")

	pubspecContent := `name: test_package
dependencies:
  flutter:
    sdk: flutter
  http: ^0.13.5
  path: ^1.8.0
dev_dependencies:
  flutter_test:
    sdk: flutter
  test: ">=1.21.0 <2.0.0"
`

	err := os.WriteFile(pubspecPath, []byte(pubspecContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	parser := NewParser()
	dependencies, err := parser.ParseDependencies(pubspecPath)
	if err != nil {
		t.Fatalf("Failed to parse pubspec.yaml: %v", err)
	}

	// Should exclude flutter SDK dependency but include others
	if len(dependencies) != 3 {
		t.Errorf("Expected 3 dependencies, got %d", len(dependencies))
	}

	// Check specific dependencies
	cleanVersionMap := make(map[string]string)
	originalVersionMap := make(map[string]string)
	for _, dep := range dependencies {
		cleanVersionMap[dep.Name] = dep.Version
		originalVersionMap[dep.Name] = dep.OriginalVersion
	}

	// Check clean versions (without prefixes)
	if cleanVersionMap["http"] != "0.13.5" {
		t.Errorf("Expected http clean version '0.13.5', got '%s'", cleanVersionMap["http"])
	}

	if cleanVersionMap["path"] != "1.8.0" {
		t.Errorf("Expected path clean version '1.8.0', got '%s'", cleanVersionMap["path"])
	}

	// Check original versions (with prefixes)
	if originalVersionMap["http"] != "^0.13.5" {
		t.Errorf("Expected http original version '^0.13.5', got '%s'", originalVersionMap["http"])
	}

	if originalVersionMap["path"] != "^1.8.0" {
		t.Errorf("Expected path original version '^1.8.0', got '%s'", originalVersionMap["path"])
	}

	if originalVersionMap["test"] != ">=1.21.0 <2.0.0" {
		t.Errorf("Expected test original version '>=1.21.0 <2.0.0', got '%s'", originalVersionMap["test"])
	}
}

func TestUpdatePubspecYaml(t *testing.T) {
	// Create a temporary pubspec.yaml file
	tempDir := t.TempDir()
	pubspecPath := filepath.Join(tempDir, "pubspec.yaml")

	pubspecContent := `name: test_package
dependencies:
  flutter:
    sdk: flutter
  http: ^0.13.5
  path: ^1.8.0
dev_dependencies:
  flutter_test:
    sdk: flutter
  test: ">=1.21.0 <2.0.0"
`

	err := os.WriteFile(pubspecPath, []byte(pubspecContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Mock outdated dependencies
	outdated := []shared.OutdatedDependency{
		{
			Name:            "http",
			CurrentVersion:  "0.13.5",
			LatestVersion:   "0.13.6",
			OriginalVersion: "^0.13.5",
		},
		{
			Name:            "path",
			CurrentVersion:  "1.8.0",
			LatestVersion:   "1.8.3",
			OriginalVersion: "^1.8.0",
		},
	}

	updater := NewUpdater()
	err = updater.UpdateDependencies(pubspecPath, outdated, false, false)
	if err != nil {
		t.Fatalf("Failed to update pubspec.yaml: %v", err)
	}

	// Read and verify the updated file
	updatedContent, err := os.ReadFile(pubspecPath)
	if err != nil {
		t.Fatalf("Failed to read updated file: %v", err)
	}

	updatedStr := string(updatedContent)

	// Check that versions were updated correctly with prefixes preserved
	if !strings.Contains(updatedStr, "http: ^0.13.6") {
		t.Errorf("http version not updated correctly, content: %s", updatedStr)
	}

	if !strings.Contains(updatedStr, "path: ^1.8.3") {
		t.Errorf("path version not updated correctly, content: %s", updatedStr)
	}

	// test should remain unchanged
	if !strings.Contains(updatedStr, `test: '>=1.21.0 <2.0.0'`) && !strings.Contains(updatedStr, `test: ">=1.21.0 <2.0.0"`) {
		t.Errorf("test version should not have changed, content: %s", updatedStr)
	}
}

func TestGetFileType(t *testing.T) {
	parser := NewParser()
	if parser.GetFileType() != "dart" {
		t.Errorf("Expected file type 'dart', got '%s'", parser.GetFileType())
	}

	updater := NewUpdater()
	if updater.GetFileType() != "dart" {
		t.Errorf("Expected file type 'dart', got '%s'", updater.GetFileType())
	}

	registry := NewRegistryClient()
	if registry.GetFileType() != "dart" {
		t.Errorf("Expected file type 'dart', got '%s'", registry.GetFileType())
	}
}
