package pub

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

	// Should exclude flutter SDK dependency and 'any' versions
	// Should include: http, path, pubdev_hosted, private_package, test = 5 dependencies
	if len(dependencies) != 5 {
		t.Errorf("Expected 5 dependencies, got %d", len(dependencies))
		for _, dependency := range dependencies {
			t.Logf("Found dependency: %s - %s (hosted: %s)", dependency.Name, dependency.OriginalVersion, dependency.HostedURL)
		}
	}

	// Check specific dependencies
	cleanVersionMap := make(map[string]string)
	originalVersionMap := make(map[string]string)
	hostedURLMap := make(map[string]string)
	for _, dependency := range dependencies {
		cleanVersionMap[dependency.Name] = dependency.Version
		originalVersionMap[dependency.Name] = dependency.OriginalVersion
		hostedURLMap[dependency.Name] = dependency.HostedURL
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

	// Check that private hosted packages are included with correct hosted URL
	if originalVersionMap["private_package"] != "^0.0.1" {
		t.Errorf("Expected private_package original version '^0.0.1', got '%s'", originalVersionMap["private_package"])
	}
	if hostedURLMap["private_package"] != "https://private.registry.com/pub" {
		t.Errorf("Expected private_package hosted URL 'https://private.registry.com/pub', got '%s'", hostedURLMap["private_package"])
	}

	// Check that pub.dev packages have empty hosted URL
	if hostedURLMap["http"] != "" {
		t.Errorf("Expected http to have empty hosted URL, got '%s'", hostedURLMap["http"])
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
	if parser.GetFileType() != "pub" {
		t.Errorf("Expected file type 'pub', got '%s'", parser.GetFileType())
	}

	updater := NewUpdater()
	if updater.GetFileType() != "pub" {
		t.Errorf("Expected file type 'pub', got '%s'", updater.GetFileType())
	}

	registry := NewRegistryClient()
	if registry.GetFileType() != "pub" {
		t.Errorf("Expected file type 'pub', got '%s'", registry.GetFileType())
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

	// Should include: http, pubdev_hosted, private_pkg = 3 dependencies ('any' versions are filtered out)
	expectedDeps := []string{"http", "pubdev_hosted", "private_pkg"}
	if len(dependencies) != len(expectedDeps) {
		t.Errorf("Expected %d dependencies, got %d", len(expectedDeps), len(dependencies))
		for _, dependency := range dependencies {
			t.Logf("Found dependency: %s - %s (hosted: %s)", dependency.Name, dependency.OriginalVersion, dependency.HostedURL)
		}
	}

	// Create map for easier testing
	dependencyMap := make(map[string]shared.Dependency)
	for _, dependency := range dependencies {
		dependencyMap[dependency.Name] = dependency
	}

	// Verify each expected dependency
	for _, expectedName := range expectedDeps {
		if _, exists := dependencyMap[expectedName]; !exists {
			t.Errorf("Expected dependency '%s' not found", expectedName)
		}
	}

	// Check specific version handling for pubdev_hosted
	if dependency, exists := dependencyMap["pubdev_hosted"]; exists {
		if dependency.OriginalVersion != "^1.0.0" {
			t.Errorf("Expected pubdev_hosted original version '^1.0.0', got '%s'", dependency.OriginalVersion)
		}
		if dependency.Version != "1.0.0" {
			t.Errorf("Expected pubdev_hosted cleaned version '1.0.0', got '%s'", dependency.Version)
		}
		if dependency.HostedURL != "" {
			t.Errorf("Expected pubdev_hosted to have empty hosted URL, got '%s'", dependency.HostedURL)
		}
	}

	// Check private hosted package
	if dependency, exists := dependencyMap["private_pkg"]; exists {
		if dependency.OriginalVersion != "1.0.0" {
			t.Errorf("Expected private_pkg original version '1.0.0', got '%s'", dependency.OriginalVersion)
		}
		if dependency.HostedURL != "https://private-registry.example.com" {
			t.Errorf("Expected private_pkg hosted URL 'https://private-registry.example.com', got '%s'", dependency.HostedURL)
		}
	}

	// Verify excluded dependencies (including 'any' versions)
	excludedDeps := []string{"flutter", "flutter_localizations", "git_pkg", "local_pkg", "intl"}
	for _, excludedName := range excludedDeps {
		if _, exists := dependencyMap[excludedName]; exists {
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

	for _, dependency := range unchangedDeps {
		if !strings.Contains(updatedStr, dependency) {
			t.Errorf("Unchanged dependency missing: %s", dependency)
		}
	}
}

func TestUpdateHostedPackages(t *testing.T) {
	// Create a pubspec.yaml with hosted packages
	tempDir := t.TempDir()
	pubspecPath := filepath.Join(tempDir, "pubspec.yaml")

	pubspecContent := `name: test_app
dependencies:
  flutter:
    sdk: flutter

  # Regular pub.dev dependency
  http: ^0.13.0

  # Private hosted package
  company_core:
    hosted: https://packages.company.com/pub
    version: ^1.0.0

  # Another private hosted package
  internal_tools:
    hosted: https://internal-registry.example.com/pub/
    version: ~2.5.0

dev_dependencies:
  flutter_test:
    sdk: flutter

  # Private dev dependency
  company_test_utils:
    hosted: https://packages.company.com/pub
    version: ^0.3.0`

	err := os.WriteFile(pubspecPath, []byte(pubspecContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Mock outdated hosted dependencies
	outdated := []shared.OutdatedDependency{
		{
			Name:            "company_core",
			CurrentVersion:  "1.0.0",
			LatestVersion:   "1.2.0",
			OriginalVersion: "^1.0.0",
			HostedURL:       "https://packages.company.com/pub",
		},
		{
			Name:            "internal_tools",
			CurrentVersion:  "2.5.0",
			LatestVersion:   "2.6.1",
			OriginalVersion: "~2.5.0",
			HostedURL:       "https://internal-registry.example.com/pub/",
		},
		{
			Name:            "http",
			CurrentVersion:  "0.13.0",
			LatestVersion:   "0.13.5",
			OriginalVersion: "^0.13.0",
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

	// Verify that versions were updated correctly
	if !strings.Contains(updatedStr, "http: ^0.13.5") {
		t.Errorf("http version not updated correctly")
	}

	// Check hosted package updates - these should update the version field, not the hosted field
	if !strings.Contains(updatedStr, "version: ^1.2.0") {
		t.Errorf("company_core version not updated correctly")
	}
	if !strings.Contains(updatedStr, "version: ~2.6.1") {
		t.Errorf("internal_tools version not updated correctly")
	}

	// Verify hosted URLs are preserved
	if !strings.Contains(updatedStr, "hosted: https://packages.company.com/pub") {
		t.Errorf("company_core hosted URL not preserved")
	}
	if !strings.Contains(updatedStr, "hosted: https://internal-registry.example.com/pub/") {
		t.Errorf("internal_tools hosted URL not preserved")
	}

	// Verify unchanged dependency
	if !strings.Contains(updatedStr, "company_test_utils:") {
		t.Errorf("company_test_utils should remain unchanged")
	}
}

func TestParsePubTokensFile(t *testing.T) {
	// Create a temporary pub-tokens.json file
	tempDir := t.TempDir()
	pubTokensPath := filepath.Join(tempDir, "pub-tokens.json")

	pubTokensContent := `{
  "version": 1,
  "hosted": [
    {
      "url": "https://packages.company.com/pub/",
      "token": "company_token_123"
    },
    {
      "url": "https://internal-registry.example.com/pub",
      "token": "internal_token_456"
    }
  ]
}`

	err := os.WriteFile(pubTokensPath, []byte(pubTokensContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test pub-tokens.json file: %v", err)
	}

	// Create config to parse into
	config := &PubConfig{
		Registries: make(map[string]RegistryConfig),
	}

	// Add default pub.dev registry
	config.Registries["pub.dev"] = RegistryConfig{
		URL: "https://pub.dev",
	}

	err = parsePubTokensFile(pubTokensPath, config)
	if err != nil {
		t.Fatalf("Failed to parse pub-tokens.json file: %v", err)
	}

	// Test that registries were added correctly
	expectedRegistries := map[string]struct {
		url   string
		token string
	}{
		"packages.company.com": {
			url:   "https://packages.company.com/pub/",
			token: "company_token_123",
		},
		"internal-registry.example.com": {
			url:   "https://internal-registry.example.com/pub",
			token: "internal_token_456",
		},
		"pub.dev": {
			url:   "https://pub.dev",
			token: "",
		},
	}

	if len(config.Registries) != len(expectedRegistries) {
		t.Errorf("Expected %d registries, got %d", len(expectedRegistries), len(config.Registries))
	}

	for hostname, expected := range expectedRegistries {
		if registry, exists := config.Registries[hostname]; !exists {
			t.Errorf("Expected registry for hostname '%s' not found", hostname)
		} else {
			if registry.URL != expected.url {
				t.Errorf("Expected URL for %s to be '%s', got '%s'", hostname, expected.url, registry.URL)
			}
			if registry.AuthToken != expected.token {
				t.Errorf("Expected token for %s to be '%s', got '%s'", hostname, expected.token, registry.AuthToken)
			}
		}
	}
}

func TestPubConfigIntegration(t *testing.T) {
	// This test verifies the full integration of parsing pub configuration
	// Create temporary directories to simulate the real environment
	tempDir := t.TempDir()

	// Create fake home directory structure
	homeDir := filepath.Join(tempDir, "home")
	dartDir := filepath.Join(homeDir, "Library", "Application Support", "dart")
	err := os.MkdirAll(dartDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create dart directory: %v", err)
	}

	// Create pub-tokens.json
	pubTokensPath := filepath.Join(dartDir, "pub-tokens.json")
	pubTokensContent := `{
  "version": 1,
  "hosted": [
    {
      "url": "https://registry.api.hectre.com/pub/",
      "token": "hectre_token_xyz"
    },
    {
      "url": "https://packages.company.com/pub",
      "token": "company_token_abc"
    }
  ]
}`

	err = os.WriteFile(pubTokensPath, []byte(pubTokensContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create pub-tokens.json: %v", err)
	}

	// Save and set temporary HOME
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", homeDir)

	// Test the full configuration parsing
	config, err := parsePubConfig()
	if err != nil {
		t.Fatalf("Failed to parse pub config: %v", err)
	}

	// Verify default pub.dev registry is present
	if registry, exists := config.Registries["pub.dev"]; !exists {
		t.Errorf("Default pub.dev registry not found")
	} else if registry.URL != "https://pub.dev" {
		t.Errorf("Expected pub.dev URL to be 'https://pub.dev', got '%s'", registry.URL)
	}

	// Verify pub-tokens.json registries
	expectedRegistries := map[string]struct {
		url   string
		token string
	}{
		"registry.api.hectre.com": {
			url:   "https://registry.api.hectre.com/pub/",
			token: "hectre_token_xyz",
		},
		"packages.company.com": {
			url:   "https://packages.company.com/pub",
			token: "company_token_abc",
		},
	}

	for hostname, expected := range expectedRegistries {
		if registry, exists := config.Registries[hostname]; !exists {
			t.Errorf("Registry for hostname '%s' from pub-tokens.json not found", hostname)
		} else {
			if registry.URL != expected.url {
				t.Errorf("Expected URL for %s to be '%s', got '%s'", hostname, expected.url, registry.URL)
			}
			if registry.AuthToken != expected.token {
				t.Errorf("Expected token for %s to be '%s', got '%s'", hostname, expected.token, registry.AuthToken)
			}
		}
	}
}
