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
		},
		"peerDependencies": {
			"react-dom": "^18.0.0"
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

	if len(dependencies) != 4 {
		t.Errorf("Expected 4 dependencies, got %d", len(dependencies))
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

	if cleanVersionMap["react-dom"] != "18.0.0" {
		t.Errorf("Expected react-dom clean version '18.0.0', got '%s'", cleanVersionMap["react-dom"])
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

	if originalVersionMap["react-dom"] != "^18.0.0" {
		t.Errorf("Expected react-dom original version '^18.0.0', got '%s'", originalVersionMap["react-dom"])
	}
}

func TestParsePeerDependencies(t *testing.T) {
	// Create a temporary package.json file with only peer dependencies
	tempDir := t.TempDir()
	packageJsonPath := filepath.Join(tempDir, "package.json")

	packageJsonContent := `{
		"name": "test-package",
		"version": "1.0.0",
		"peerDependencies": {
			"react": "^18.0.0",
			"react-dom": "^18.0.0",
			"@types/react": ">=18.0.0",
			"lodash": "~4.17.20"
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

	if len(dependencies) != 4 {
		t.Errorf("Expected 4 peer dependencies, got %d", len(dependencies))
		for _, dep := range dependencies {
			t.Logf("Found dependency: %s - %s", dep.Name, dep.OriginalVersion)
		}
	}

	// Create maps for easier testing
	cleanVersionMap := make(map[string]string)
	originalVersionMap := make(map[string]string)
	for _, dep := range dependencies {
		cleanVersionMap[dep.Name] = dep.Version
		originalVersionMap[dep.Name] = dep.OriginalVersion
	}

	// Test peer dependency parsing
	expectedDeps := map[string]struct {
		cleanVersion    string
		originalVersion string
	}{
		"react":        {"18.0.0", "^18.0.0"},
		"react-dom":    {"18.0.0", "^18.0.0"},
		"@types/react": {"18.0.0", ">=18.0.0"},
		"lodash":       {"4.17.20", "~4.17.20"},
	}

	for name, expected := range expectedDeps {
		if cleanVersionMap[name] != expected.cleanVersion {
			t.Errorf("Expected %s clean version '%s', got '%s'", name, expected.cleanVersion, cleanVersionMap[name])
		}
		if originalVersionMap[name] != expected.originalVersion {
			t.Errorf("Expected %s original version '%s', got '%s'", name, expected.originalVersion, originalVersionMap[name])
		}
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

	// First, parse dependencies to get line numbers
	parser := NewParser()
	dependencies, err := parser.ParseDependencies(packageJsonPath)
	if err != nil {
		t.Fatalf("Failed to parse package.json: %v", err)
	}

	// Create a map to look up line numbers
	lineNumbers := make(map[string]int)
	for _, dep := range dependencies {
		lineNumbers[dep.Name] = dep.LineNumber
	}

	// Mock outdated dependencies
	outdated := []shared.OutdatedDependency{
		{
			Name:            "react",
			CurrentVersion:  "18.0.0",
			LatestVersion:   "18.2.0",
			OriginalVersion: "^18.0.0",
			Type:            shared.Dependencies,
			LineNumber:      lineNumbers["react"],
		},
		{
			Name:            "lodash",
			CurrentVersion:  "4.17.20",
			LatestVersion:   "4.17.21",
			OriginalVersion: "~4.17.20",
			Type:            shared.Dependencies,
			LineNumber:      lineNumbers["lodash"],
		},
	}

	updater := NewUpdater()
	err = updater.UpdateDependencies(packageJsonPath, outdated, false, false, false)
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
		{
			Name:            "react",
			CurrentVersion:  "18.0.0",
			LatestVersion:   "18.2.0",
			OriginalVersion: "^18.0.0",
			Type:            shared.Dependencies,
			LineNumber:      30, // "react": "^18.0.0",
		},
		{
			Name:            "axios",
			CurrentVersion:  "1.4.0",
			LatestVersion:   "1.5.0",
			OriginalVersion: "^1.4.0",
			Type:            shared.Dependencies,
			LineNumber:      32, // "axios": "^1.4.0",
		},
		{
			Name:            "eslint",
			CurrentVersion:  "8.45.0",
			LatestVersion:   "8.47.0",
			OriginalVersion: "^8.45.0",
			Type:            shared.DevDependencies,
			LineNumber:      39, // "eslint": "^8.45.0",
		},
	}

	// Update the dependencies
	updater := NewUpdater()
	err = updater.UpdateDependencies(testFile, deps, false, false, false)
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

	// Note: PeerDependencies should remain unchanged by default.
	// Only dependencies and devDependencies should be updated.
	if !strings.Contains(updatedStr, `"react": "^18.2.0"`) {
		t.Errorf("React dependency should be updated to ^18.2.0")
	}

	// Verify that peerDependencies remain unchanged
	if !strings.Contains(updatedStr, `"react": ">=16.8.0"`) {
		t.Errorf("PeerDependencies should remain unchanged when includePeerDependencies is false")
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

func TestParseScopedPackages(t *testing.T) {
	// Create a temporary package.json file with scoped packages
	tempDir := t.TempDir()
	packageJsonPath := filepath.Join(tempDir, "package.json")

	packageJsonContent := `{
		"dependencies": {
			"react": "^18.0.0",
			"@company/private-pkg": "^1.2.3",
			"@angular/core": "^16.0.0",
			"@types/node": "^20.0.0"
		},
		"devDependencies": {
			"@company/dev-tools": "~2.1.0",
			"@babel/core": ">=7.22.0"
		},
		"peerDependencies": {
			"@angular/common": "^16.0.0"
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

	// Should include all 7 dependencies including scoped ones
	if len(dependencies) != 7 {
		t.Errorf("Expected 7 dependencies, got %d", len(dependencies))
		for _, dep := range dependencies {
			t.Logf("Found dependency: %s - %s", dep.Name, dep.OriginalVersion)
		}
	}

	// Create maps for easier testing
	cleanVersionMap := make(map[string]string)
	originalVersionMap := make(map[string]string)
	for _, dep := range dependencies {
		cleanVersionMap[dep.Name] = dep.Version
		originalVersionMap[dep.Name] = dep.OriginalVersion
	}

	// Test scoped package parsing
	expectedDeps := map[string]struct {
		cleanVersion    string
		originalVersion string
	}{
		"react":                {"18.0.0", "^18.0.0"},
		"@company/private-pkg": {"1.2.3", "^1.2.3"},
		"@angular/core":        {"16.0.0", "^16.0.0"},
		"@types/node":          {"20.0.0", "^20.0.0"},
		"@company/dev-tools":   {"2.1.0", "~2.1.0"},
		"@babel/core":          {"7.22.0", ">=7.22.0"},
		"@angular/common":      {"16.0.0", "^16.0.0"},
	}

	for name, expected := range expectedDeps {
		if cleanVersionMap[name] != expected.cleanVersion {
			t.Errorf("Expected %s clean version '%s', got '%s'", name, expected.cleanVersion, cleanVersionMap[name])
		}
		if originalVersionMap[name] != expected.originalVersion {
			t.Errorf("Expected %s original version '%s', got '%s'", name, expected.originalVersion, originalVersionMap[name])
		}
	}
}

func TestUpdateScopedPackages(t *testing.T) {
	// Create a temporary package.json file with scoped packages
	tempDir := t.TempDir()
	packageJsonPath := filepath.Join(tempDir, "package.json")

	packageJsonContent := `{
  "dependencies": {
    "react": "^18.0.0",
    "@company/private-pkg": "^1.2.3",
    "@angular/core": "^16.0.0"
  },
  "devDependencies": {
    "@company/dev-tools": "~2.1.0",
    "@babel/core": ">=7.22.0"
  }
}`

	err := os.WriteFile(packageJsonPath, []byte(packageJsonContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Mock outdated scoped dependencies
	outdated := []shared.OutdatedDependency{
		{
			Name:            "@company/private-pkg",
			CurrentVersion:  "1.2.3",
			LatestVersion:   "1.3.0",
			OriginalVersion: "^1.2.3",
			Type:            shared.Dependencies,
			LineNumber:      4,
		},
		{
			Name:            "@angular/core",
			CurrentVersion:  "16.0.0",
			LatestVersion:   "16.2.0",
			OriginalVersion: "^16.0.0",
			Type:            shared.Dependencies,
			LineNumber:      5,
		},
		{
			Name:            "@babel/core",
			CurrentVersion:  "7.22.0",
			LatestVersion:   "7.22.5",
			OriginalVersion: ">=7.22.0",
			Type:            shared.DevDependencies,
			LineNumber:      9,
		},
	}

	updater := NewUpdater()
	err = updater.UpdateDependencies(packageJsonPath, outdated, false, false, false)
	if err != nil {
		t.Fatalf("Failed to update package.json: %v", err)
	}

	// Read and verify the updated file
	updatedContent, err := os.ReadFile(packageJsonPath)
	if err != nil {
		t.Fatalf("Failed to read updated file: %v", err)
	}

	updatedStr := string(updatedContent)

	// Check that scoped packages were updated correctly with prefixes preserved
	expectedUpdates := map[string]string{
		`"@company/private-pkg": "^1.3.0"`: "@company/private-pkg should be updated to ^1.3.0",
		`"@angular/core": "^16.2.0"`:       "@angular/core should be updated to ^16.2.0",
		`"@babel/core": ">=7.22.5"`:        "@babel/core should be updated to >=7.22.5",
	}

	for expectedText, errorMsg := range expectedUpdates {
		if !strings.Contains(updatedStr, expectedText) {
			t.Errorf("%s, but got: %s", errorMsg, updatedStr)
		}
	}

	// Verify unchanged dependencies
	if !strings.Contains(updatedStr, `"react": "^18.0.0"`) {
		t.Errorf("React version should not have changed")
	}
	if !strings.Contains(updatedStr, `"@company/dev-tools": "~2.1.0"`) {
		t.Errorf("@company/dev-tools version should not have changed")
	}
}

func TestParseNpmrcFile(t *testing.T) {
	// Create a temporary .npmrc file
	tempDir := t.TempDir()
	npmrcPath := filepath.Join(tempDir, ".npmrc")

	npmrcContent := `# NPM configuration
registry=https://registry.npmjs.org/
@company:registry=https://npm.company.com
@internal:registry=https://internal-registry.example.com/

# Authentication tokens
//npm.company.com/:_authToken=company_token_123
//internal-registry.example.com/:_authToken="internal_token_456"
//registry.example.com/:_authToken='quoted_token_789'

# Comments and empty lines should be ignored

; Semicolon comments too
`

	err := os.WriteFile(npmrcPath, []byte(npmrcContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test .npmrc file: %v", err)
	}

	config, err := parseNpmrcFile(npmrcPath)
	if err != nil {
		t.Fatalf("Failed to parse .npmrc file: %v", err)
	}

	// Test scope registries
	expectedScopeRegistries := map[string]string{
		"@company":  "https://npm.company.com",
		"@internal": "https://internal-registry.example.com/",
	}

	for scope, expectedRegistry := range expectedScopeRegistries {
		if actualRegistry, exists := config.ScopeRegistries[scope]; !exists {
			t.Errorf("Expected scope registry for %s not found", scope)
		} else if actualRegistry != expectedRegistry {
			t.Errorf("Expected scope registry for %s to be '%s', got '%s'", scope, expectedRegistry, actualRegistry)
		}
	}

	// Test auth tokens (should strip quotes)
	expectedAuthTokens := map[string]string{
		"npm.company.com":               "company_token_123",
		"internal-registry.example.com": "internal_token_456",
		"registry.example.com":          "quoted_token_789",
	}

	for registry, expectedToken := range expectedAuthTokens {
		if actualToken, exists := config.AuthTokens[registry]; !exists {
			t.Errorf("Expected auth token for %s not found", registry)
		} else if actualToken != expectedToken {
			t.Errorf("Expected auth token for %s to be '%s', got '%s'", registry, expectedToken, actualToken)
		}
	}
}

func TestParseNpmrcFilesWithGlobalAndLocal(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	homeDir := filepath.Join(tempDir, "home")
	projectDir := filepath.Join(tempDir, "project")

	err := os.MkdirAll(homeDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create home directory: %v", err)
	}
	err = os.MkdirAll(projectDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}

	// Create global .npmrc (in home directory)
	globalNpmrcPath := filepath.Join(homeDir, ".npmrc")
	globalNpmrcContent := `@company:registry=https://global.company.com
//global.company.com/:_authToken=global_token
//shared-registry.com/:_authToken=global_shared_token`

	err = os.WriteFile(globalNpmrcPath, []byte(globalNpmrcContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create global .npmrc file: %v", err)
	}

	// Create local .npmrc (in project directory)
	localNpmrcPath := filepath.Join(projectDir, ".npmrc")
	localNpmrcContent := `@company:registry=https://local.company.com
@internal:registry=https://internal.example.com
//local.company.com/:_authToken=local_token
//shared-registry.com/:_authToken=local_shared_token`

	err = os.WriteFile(localNpmrcPath, []byte(localNpmrcContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create local .npmrc file: %v", err)
	}

	// Set HOME environment variable temporarily
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", homeDir)

	// Test parseNpmrcFiles function
	config, err := parseNpmrcFiles(localNpmrcPath)
	if err != nil {
		t.Fatalf("Failed to parse .npmrc files: %v", err)
	}

	// Local scope registries should override global ones
	if config.ScopeRegistries["@company"] != "https://local.company.com" {
		t.Errorf("Expected local @company registry to override global, got '%s'", config.ScopeRegistries["@company"])
	}

	// Local-only scope registries should be present
	if config.ScopeRegistries["@internal"] != "https://internal.example.com" {
		t.Errorf("Expected @internal registry to be '%s', got '%s'", "https://internal.example.com", config.ScopeRegistries["@internal"])
	}

	// Local auth tokens should override global ones
	if config.AuthTokens["shared-registry.com"] != "local_shared_token" {
		t.Errorf("Expected local shared token to override global, got '%s'", config.AuthTokens["shared-registry.com"])
	}

	// Local-only auth tokens should be present
	if config.AuthTokens["local.company.com"] != "local_token" {
		t.Errorf("Expected local.company.com token to be 'local_token', got '%s'", config.AuthTokens["local.company.com"])
	}

	// Global-only auth tokens should be present
	if config.AuthTokens["global.company.com"] != "global_token" {
		t.Errorf("Expected global.company.com token to be 'global_token', got '%s'", config.AuthTokens["global.company.com"])
	}
}

func TestGetRegistryForPackage(t *testing.T) {
	config := &NpmConfig{
		ScopeRegistries: map[string]string{
			"@company":  "https://npm.company.com",
			"@internal": "https://internal-registry.example.com",
		},
		AuthTokens: map[string]string{
			"npm.company.com": "company_token",
		},
	}

	tests := []struct {
		packageName      string
		expectedRegistry string
	}{
		{"@company/package", "https://npm.company.com"},
		{"@internal/tool", "https://internal-registry.example.com"},
		{"@unknown/package", "https://registry.npmjs.org"},
		{"regular-package", "https://registry.npmjs.org"},
		{"@malformed", "https://registry.npmjs.org"},
	}

	for _, test := range tests {
		actualRegistry := getRegistryForPackage(test.packageName, config)
		if actualRegistry != test.expectedRegistry {
			t.Errorf("For package '%s', expected registry '%s', got '%s'",
				test.packageName, test.expectedRegistry, actualRegistry)
		}
	}
}

func TestGetAuthTokenForRegistry(t *testing.T) {
	config := &NpmConfig{
		ScopeRegistries: map[string]string{},
		AuthTokens: map[string]string{
			"npm.company.com":      "company_token",
			"registry.example.com": "example_token",
			"internal.corp.com":    "internal_token",
		},
	}

	tests := []struct {
		registryURL   string
		expectedToken string
	}{
		{"https://npm.company.com", "company_token"},
		{"https://npm.company.com/", "company_token"},
		{"https://registry.example.com", "example_token"},
		{"https://internal.corp.com/npm", "internal_token"},
		{"https://unknown-registry.com", ""},
		{"http://npm.company.com", "company_token"}, // Same hostname, same token regardless of protocol
	}

	for _, test := range tests {
		actualToken := getAuthTokenForRegistry(test.registryURL, config)
		if actualToken != test.expectedToken {
			t.Errorf("For registry '%s', expected token '%s', got '%s'",
				test.registryURL, test.expectedToken, actualToken)
		}
	}
}

// TestUpdateDuplicateDependenciesWithDifferentConstraints tests that when the same
// dependency appears in multiple sections with different semver constraints,
// each section preserves its own constraint prefix
func TestUpdateDuplicateDependenciesWithDifferentConstraints(t *testing.T) {
	packageJsonContent := `{
  "name": "test-package",
  "version": "1.0.0",
  "dependencies": {
    "react": "^18.0.0"
  },
  "devDependencies": {
    "eslint": "^8.45.0"
  },
  "peerDependencies": {
    "react": ">=16.0.0"
  }
}`

	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "package.json")

	// Write the original content
	err := os.WriteFile(testFile, []byte(packageJsonContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Update both react dependencies (one in dependencies, one in peerDependencies)
	deps := []shared.OutdatedDependency{
		{
			Name:            "react",
			CurrentVersion:  "18.0.0",
			LatestVersion:   "18.2.0",
			OriginalVersion: "^18.0.0",
			Type:            shared.Dependencies,
			LineNumber:      5,
		},
		{
			Name:            "react",
			CurrentVersion:  "16.0.0",
			LatestVersion:   "18.2.0",
			OriginalVersion: ">=16.0.0",
			Type:            shared.PeerDependencies,
			LineNumber:      11,
		},
	}

	updater := NewUpdater()
	err = updater.UpdateDependencies(testFile, deps, false, false, true) // includePeerDependencies = true
	if err != nil {
		t.Fatal(err)
	}

	// Read the updated content
	updatedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	updatedStr := string(updatedContent)

	// Verify that dependencies section has caret constraint preserved
	if !strings.Contains(updatedStr, `"react": "^18.2.0"`) {
		t.Errorf("Expected react in dependencies to be updated to '^18.2.0' with caret constraint preserved")
	}

	// Verify that peerDependencies section has >= constraint preserved
	if !strings.Contains(updatedStr, `"react": ">=18.2.0"`) {
		t.Errorf("Expected react in peerDependencies to be updated to '>=18.2.0' with >= constraint preserved")
	}

	// Verify both sections exist and have different versions
	dependenciesMatch := strings.Contains(updatedStr, `"dependencies": {
    "react": "^18.2.0"
  }`)
	peerDependenciesMatch := strings.Contains(updatedStr, `"peerDependencies": {
    "react": ">=18.2.0"
  }`)

	if !dependenciesMatch {
		t.Errorf("Dependencies section not correctly updated")
	}
	if !peerDependenciesMatch {
		t.Errorf("PeerDependencies section not correctly updated")
	}
}
