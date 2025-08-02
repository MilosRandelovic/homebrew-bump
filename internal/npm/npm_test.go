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

func TestUpdatePreservesAllContent(t *testing.T) {
	// Realistic package.json content with scripts, metadata, config, and dependencies
	originalContent := `{
  "name": "my-react-app",
  "version": "0.1.0",
  "private": true,
  "description": "A comprehensive React application",
  "author": "John Doe <john@example.com>",
  "license": "MIT",
  "keywords": ["react", "frontend", "web"],
  "homepage": "https://example.com",
  "repository": {
    "type": "git",
    "url": "https://github.com/user/my-react-app.git"
  },
  "bugs": {
    "url": "https://github.com/user/my-react-app/issues"
  },
  "engines": {
    "node": ">=16.0.0",
    "npm": ">=8.0.0"
  },
  "scripts": {
    "start": "react-scripts start",
    "build": "react-scripts build",
    "test": "react-scripts test",
    "eject": "react-scripts eject",
    "lint": "eslint src/",
    "format": "prettier --write src/"
  },
  "dependencies": {
    "react": "^18.0.0",
    "react-dom": "^18.0.0",
    "axios": "^1.4.0",
    "lodash": "~4.17.20",
    "moment": ">=2.29.0"
  },
  "devDependencies": {
    "react-scripts": "5.0.1",
    "typescript": ">=4.9.0",
    "eslint": "^8.45.0",
    "prettier": "^2.8.0",
    "@types/react": "^18.0.0"
  },
  "peerDependencies": {
    "react": ">=16.8.0"
  },
  "browserslist": {
    "production": [
      ">0.2%",
      "not dead",
      "not op_mini all"
    ],
    "development": [
      "last 1 chrome version",
      "last 1 firefox version",
      "last 1 safari version"
    ]
  },
  "eslintConfig": {
    "extends": [
      "react-app",
      "react-app/jest"
    ]
  },
  "jest": {
    "collectCoverageFrom": [
      "src/**/*.{js,jsx,ts,tsx}",
      "!src/index.js"
    ]
  },
  "proxy": "http://localhost:3001",
  "custom": {
    "feature_flags": {
      "new_ui": true,
      "analytics": false
    },
    "build_config": {
      "optimization": "advanced",
      "source_maps": true
    }
  }
}`

	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "package.json")

	// Write the original content
	err := os.WriteFile(testFile, []byte(originalContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Mock dependencies for update
	deps := []shared.OutdatedDependency{
		{Name: "react", CurrentVersion: "18.0.0", LatestVersion: "18.2.0", OriginalVersion: "^18.0.0"},
		{Name: "axios", CurrentVersion: "1.4.0", LatestVersion: "1.5.0", OriginalVersion: "^1.4.0"},
		{Name: "eslint", CurrentVersion: "8.45.0", LatestVersion: "8.47.0", OriginalVersion: "^8.45.0"},
	}

	// Update the dependencies
	updater := NewUpdater()
	err = updater.UpdateDependencies(testFile, deps, false, false)
	if err != nil {
		t.Fatal(err)
	}

	// Read the updated content
	updatedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	updatedStr := string(updatedContent)

	// Verify that critical non-dependency content is preserved
	criticalContent := []string{
		`"name": "my-react-app"`,
		`"version": "0.1.0"`,
		`"private": true`,
		`"description": "A comprehensive React application"`,
		`"author": "John Doe <john@example.com>"`,
		`"license": "MIT"`,
		`"keywords": ["react", "frontend", "web"]`,
		`"homepage": "https://example.com"`,
		`"repository":`,
		`"type": "git"`,
		`"url": "https://github.com/user/my-react-app.git"`,
		`"bugs":`,
		`"engines":`,
		`"node": ">=16.0.0"`,
		`"npm": ">=8.0.0"`,
		`"scripts":`,
		`"start": "react-scripts start"`,
		`"build": "react-scripts build"`,
		`"test": "react-scripts test"`,
		`"eject": "react-scripts eject"`,
		`"lint": "eslint src/"`,
		`"format": "prettier --write src/"`,
		`"peerDependencies":`,
		`"browserslist":`,
		`"production":`,
		`">0.2%"`,
		`"not dead"`,
		`"not op_mini all"`,
		`"development":`,
		`"last 1 chrome version"`,
		`"eslintConfig":`,
		`"extends":`,
		`"react-app"`,
		`"react-app/jest"`,
		`"jest":`,
		`"collectCoverageFrom":`,
		`"src/**/*.{js,jsx,ts,tsx}"`,
		`"!src/index.js"`,
		`"proxy": "http://localhost:3001"`,
		`"custom":`,
		`"feature_flags":`,
		`"new_ui": true`,
		`"analytics": false`,
		`"build_config":`,
		`"optimization": "advanced"`,
		`"source_maps": true`,
	}

	for _, content := range criticalContent {
		if !strings.Contains(updatedStr, content) {
			t.Errorf("Critical content missing after update: %s", content)
		}
	}

	// Verify that dependencies were actually updated
	expectedUpdates := map[string]string{
		`"react": "^18.2.0"`:  "react version should be updated to 18.2.0",
		`"axios": "^1.5.0"`:   "axios version should be updated to 1.5.0",
		`"eslint": "^8.47.0"`: "eslint version should be updated to 8.47.0",
	}

	for expectedText, errorMsg := range expectedUpdates {
		if !strings.Contains(updatedStr, expectedText) {
			t.Errorf("%s, but found:\n%s", errorMsg, updatedStr)
		}
	}

	// Verify that unchanged dependencies remain unchanged
	unchangedDeps := []string{
		`"react-dom": "^18.0.0"`,
		`"lodash": "~4.17.20"`,
		`"moment": ">=2.29.0"`,
		`"typescript": ">=4.9.0"`,
		`"prettier": "^2.8.0"`,
	}

	for _, dep := range unchangedDeps {
		if !strings.Contains(updatedStr, dep) {
			t.Errorf("Unchanged dependency missing: %s", dep)
		}
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
