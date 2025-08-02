package npm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MilosRandelovic/homebrew-bump/internal/shared"
)

func TestParsePackageJson(t *testing.T) {
	// Create a temporary package.json file
	tempDir := t.TempDir()
	packageJsonPath := filepath.Join(tempDir, "package.json")

	packageJsonContent := `{
		"dependencies": {
			"react": "^18.0.0",
			"lodash": "~4.17.20"
		},
		"devDependencies": {
			"typescript": ">=4.9.0"
		}
	}`

	err := os.WriteFile(packageJsonPath, []byte(packageJsonContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	parser := NewParser()
	dependencies, err := parser.ParseDependencies(packageJsonPath)
	if err != nil {
		t.Fatalf("Failed to parse package.json: %v", err)
	}

	if len(dependencies) != 3 {
		t.Errorf("Expected 3 dependencies, got %d", len(dependencies))
	}

	// Check specific dependencies - create maps for both clean and original versions
	cleanVersionMap := make(map[string]string)
	originalVersionMap := make(map[string]string)
	for _, dep := range dependencies {
		cleanVersionMap[dep.Name] = dep.Version
		originalVersionMap[dep.Name] = dep.OriginalVersion
	}

	// Check clean versions (without prefixes)
	if cleanVersionMap["react"] != "18.0.0" {
		t.Errorf("Expected react clean version '18.0.0', got '%s'", cleanVersionMap["react"])
	}

	if cleanVersionMap["lodash"] != "4.17.20" {
		t.Errorf("Expected lodash clean version '4.17.20', got '%s'", cleanVersionMap["lodash"])
	}

	if cleanVersionMap["typescript"] != "4.9.0" {
		t.Errorf("Expected typescript clean version '4.9.0', got '%s'", cleanVersionMap["typescript"])
	}

	// Check original versions (with prefixes)
	if originalVersionMap["react"] != "^18.0.0" {
		t.Errorf("Expected react original version '^18.0.0', got '%s'", originalVersionMap["react"])
	}

	if originalVersionMap["lodash"] != "~4.17.20" {
		t.Errorf("Expected lodash original version '~4.17.20', got '%s'", originalVersionMap["lodash"])
	}

	if originalVersionMap["typescript"] != ">=4.9.0" {
		t.Errorf("Expected typescript original version '>=4.9.0', got '%s'", originalVersionMap["typescript"])
	}
}

func TestUpdatePackageJson(t *testing.T) {
	// Create a temporary package.json file
	tempDir := t.TempDir()
	packageJsonPath := filepath.Join(tempDir, "package.json")

	packageJsonContent := `{
  "dependencies": {
    "react": "^18.0.0",
    "lodash": "~4.17.20"
  },
  "devDependencies": {
    "typescript": ">=4.9.0"
  }
}`

	err := os.WriteFile(packageJsonPath, []byte(packageJsonContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Mock outdated dependencies
	outdated := []shared.OutdatedDependency{
		{
			Name:            "react",
			CurrentVersion:  "18.0.0",
			LatestVersion:   "18.2.0",
			OriginalVersion: "^18.0.0",
		},
		{
			Name:            "lodash",
			CurrentVersion:  "4.17.20",
			LatestVersion:   "4.17.21",
			OriginalVersion: "~4.17.20",
		},
	}

	updater := NewUpdater()
	err = updater.UpdateDependencies(packageJsonPath, outdated, false, false)
	if err != nil {
		t.Fatalf("Failed to update package.json: %v", err)
	}

	// Read and verify the updated file
	updatedContent, err := os.ReadFile(packageJsonPath)
	if err != nil {
		t.Fatalf("Failed to read updated file: %v", err)
	}

	updatedStr := string(updatedContent)

	// Check that versions were updated correctly with prefixes preserved
	if !strings.Contains(updatedStr, `"react": "^18.2.0"`) {
		t.Errorf("React version not updated correctly, content: %s", updatedStr)
	}

	if !strings.Contains(updatedStr, `"lodash": "~4.17.21"`) {
		t.Errorf("Lodash version not updated correctly, content: %s", updatedStr)
	}

	// TypeScript should remain unchanged
	if !strings.Contains(updatedStr, `"typescript": ">=4.9.0"`) {
		t.Errorf("TypeScript version should not have changed, content: %s", updatedStr)
	}
}

func TestGetFileType(t *testing.T) {
	parser := NewParser()
	if parser.GetFileType() != "npm" {
		t.Errorf("Expected file type 'npm', got '%s'", parser.GetFileType())
	}

	updater := NewUpdater()
	if updater.GetFileType() != "npm" {
		t.Errorf("Expected file type 'npm', got '%s'", updater.GetFileType())
	}

	registry := NewRegistryClient()
	if registry.GetFileType() != "npm" {
		t.Errorf("Expected file type 'npm', got '%s'", registry.GetFileType())
	}
}
