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
  intl: any
  private_package:
    hosted: https://private.registry.com/pub
    version: ^0.0.1
  pubdev_hosted:
    hosted: https://pub.dev
    version: ^1.0.0
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

	// Should exclude flutter SDK dependency, private hosted packages, and 'any' versions
	// Should include: http, path, pubdev_hosted, test = 4 dependencies
	if len(dependencies) != 4 {
		t.Errorf("Expected 4 dependencies, got %d", len(dependencies))
		for _, dep := range dependencies {
			t.Logf("Found dependency: %s - %s", dep.Name, dep.OriginalVersion)
		}
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

	// Check that pub.dev hosted packages are included
	if originalVersionMap["pubdev_hosted"] != "^1.0.0" {
		t.Errorf("Expected pubdev_hosted original version '^1.0.0', got '%s'", originalVersionMap["pubdev_hosted"])
	}

	// Check that 'any' versions are excluded
	if _, exists := originalVersionMap["intl"]; exists {
		t.Errorf("Expected intl ('any' version) to be excluded, but it was found with version '%s'", originalVersionMap["intl"])
	}

	// Check that private hosted packages are excluded
	if _, exists := originalVersionMap["private_package"]; exists {
		t.Errorf("Expected private_package to be excluded, but it was found with version '%s'", originalVersionMap["private_package"])
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

func TestParsePubspecYamlEdgeCases(t *testing.T) {
	// Test various edge cases for pubspec parsing
	tempDir := t.TempDir()
	pubspecPath := filepath.Join(tempDir, "pubspec.yaml")

	pubspecContent := `name: test_package
dependencies:
  # Regular dependencies
  http: ^0.13.5
  intl: any

  # SDK dependencies (should be skipped)
  flutter:
    sdk: flutter
  flutter_localizations:
    sdk: flutter

  # Private hosted packages (should be skipped)
  private_pkg:
    hosted: "https://private-registry.example.com"
    version: "1.0.0"

  # Public hosted packages (should be included)
  pubdev_hosted:
    hosted: https://pub.dev
    version: ^1.0.0

  # Git dependencies (should be skipped)
  git_pkg:
    git:
      url: https://github.com/example/repo.git
      ref: main

  # Path dependencies (should be skipped)
  local_pkg:
    path: ../local_package
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

	// Should only include: http, pubdev_hosted = 2 dependencies ('any' versions are filtered out)
	expectedDeps := []string{"http", "pubdev_hosted"}
	if len(dependencies) != len(expectedDeps) {
		t.Errorf("Expected %d dependencies, got %d", len(expectedDeps), len(dependencies))
		for _, dep := range dependencies {
			t.Logf("Found dependency: %s - %s", dep.Name, dep.OriginalVersion)
		}
	}

	// Create map for easier testing
	depMap := make(map[string]shared.Dependency)
	for _, dep := range dependencies {
		depMap[dep.Name] = dep
	}

	// Verify each expected dependency
	for _, expectedName := range expectedDeps {
		if _, exists := depMap[expectedName]; !exists {
			t.Errorf("Expected dependency '%s' not found", expectedName)
		}
	}

	// Check specific version handling for pubdev_hosted
	if dep, exists := depMap["pubdev_hosted"]; exists {
		if dep.OriginalVersion != "^1.0.0" {
			t.Errorf("Expected pubdev_hosted original version '^1.0.0', got '%s'", dep.OriginalVersion)
		}
		if dep.Version != "1.0.0" {
			t.Errorf("Expected pubdev_hosted cleaned version '1.0.0', got '%s'", dep.Version)
		}
	}

	// Verify excluded dependencies (including 'any' versions)
	excludedDeps := []string{"flutter", "flutter_localizations", "private_pkg1", "private_pkg2", "git_pkg", "local_pkg", "intl"}
	for _, excludedName := range excludedDeps {
		if _, exists := depMap[excludedName]; exists {
			t.Errorf("Dependency '%s' should have been excluded but was found", excludedName)
		}
	}
}

func TestUpdatePreservesAllContent(t *testing.T) {
	// Realistic pubspec.yaml content with comments, metadata, assets, and dependencies
	originalContent := `name: my_flutter_app
description: A new Flutter application.
version: 1.0.0+1

environment:
  sdk: '>=3.1.0 <4.0.0'
  flutter: ">=3.13.0"

dependencies:
  flutter:
    sdk: flutter

  # HTTP client
  http: ^0.13.0

  # State management
  provider: ^6.0.0

  # Utilities
  collection: ^1.17.0

dev_dependencies:
  flutter_test:
    sdk: flutter

  # Code generation
  build_runner: ^2.4.0

  # Linting
  flutter_lints: ^2.0.0

flutter:
  uses-material-design: true

  # Assets
  assets:
    - assets/images/
    - assets/icons/

  # Fonts
  fonts:
    - family: CustomFont
      fonts:
        - asset: fonts/CustomFont-Regular.ttf
        - asset: fonts/CustomFont-Bold.ttf
          weight: 700

# Custom configuration
custom_config:
  feature_flags:
    new_ui: true
    analytics: false`

	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "pubspec.yaml")

	// Write the original content
	err := os.WriteFile(testFile, []byte(originalContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Mock dependencies for update
	deps := []shared.OutdatedDependency{
		{Name: "http", CurrentVersion: "0.13.0", LatestVersion: "0.13.5"},
		{Name: "provider", CurrentVersion: "6.0.0", LatestVersion: "6.1.2"},
		{Name: "build_runner", CurrentVersion: "2.4.0", LatestVersion: "2.4.7"},
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
		"name: my_flutter_app",
		"description: A new Flutter application.",
		"version: 1.0.0+1",
		"environment:",
		"sdk: '>=3.1.0 <4.0.0'",
		"flutter: \">=3.13.0\"",
		"# HTTP client",
		"# State management",
		"# Utilities",
		"# Code generation",
		"# Linting",
		"flutter:",
		"uses-material-design: true",
		"# Assets",
		"assets:",
		"- assets/images/",
		"- assets/icons/",
		"# Fonts",
		"fonts:",
		"- family: CustomFont",
		"fonts:",
		"- asset: fonts/CustomFont-Regular.ttf",
		"- asset: fonts/CustomFont-Bold.ttf",
		"weight: 700",
		"# Custom configuration",
		"custom_config:",
		"feature_flags:",
		"new_ui: true",
		"analytics: false",
	}

	for _, content := range criticalContent {
		if !strings.Contains(updatedStr, content) {
			t.Errorf("Critical content missing after update: %s", content)
		}
	}

	// Verify that dependencies were actually updated
	expectedUpdates := map[string]string{
		"http: ^0.13.5":        "http version should be updated to 0.13.5",
		"provider: ^6.1.2":     "provider version should be updated to 6.1.2",
		"build_runner: ^2.4.7": "build_runner version should be updated to 2.4.7",
	}

	for expectedText, errorMsg := range expectedUpdates {
		if !strings.Contains(updatedStr, expectedText) {
			t.Errorf("%s, but found:\n%s", errorMsg, updatedStr)
		}
	}

	// Verify that unchanged dependencies remain unchanged
	unchangedDeps := []string{
		"collection: ^1.17.0",
		"flutter_lints: ^2.0.0",
	}

	for _, dep := range unchangedDeps {
		if !strings.Contains(updatedStr, dep) {
			t.Errorf("Unchanged dependency missing: %s", dep)
		}
	}
}
