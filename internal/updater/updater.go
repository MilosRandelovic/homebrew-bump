package updater

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/MilosRandelovic/homebrew-bump/internal/parser"
	"gopkg.in/yaml.v3"
)

// OutdatedDependency represents a dependency that has a newer version available
type OutdatedDependency struct {
	Name            string
	CurrentVersion  string
	LatestVersion   string
	OriginalVersion string // Original version with prefixes (e.g., "^1.2.3")
}

// CheckResult contains the results of checking dependencies
type CheckResult struct {
	Outdated      []OutdatedDependency
	Errors        []DependencyError
	SemverSkipped []SemverSkipped
}

// DependencyError represents an error that occurred while checking a dependency
type DependencyError struct {
	Name  string
	Error string
}

// SemverSkipped represents a dependency that was skipped due to semver constraints
type SemverSkipped struct {
	Name            string
	CurrentVersion  string
	LatestVersion   string
	OriginalVersion string
	Reason          string
}

// NpmPackageInfo represents the response from NPM registry
type NpmPackageInfo struct {
	DistTags map[string]string `json:"dist-tags"`
	Versions map[string]struct {
		Version string `json:"version"`
	} `json:"versions"`
}

// PubDevPackageInfo represents the response from pub.dev API
type PubDevPackageInfo struct {
	Latest struct {
		Version string `json:"version"`
	} `json:"latest"`
}

// NpmrcConfig holds the parsed .npmrc configuration
type NpmrcConfig struct {
	ScopeRegistries map[string]string // maps scope to registry URL
	AuthTokens      map[string]string // maps registry to auth token
}

// parseNpmrcFiles parses both local and global .npmrc files and merges their configurations
// Local .npmrc takes precedence for scope registries, global .npmrc provides auth tokens
func parseNpmrcFiles(localPath string) (*NpmrcConfig, error) {
	config := &NpmrcConfig{
		ScopeRegistries: make(map[string]string),
		AuthTokens:      make(map[string]string),
	}

	// Parse global .npmrc first (from home directory)
	homeDir, err := os.UserHomeDir()
	if err == nil {
		globalNpmrcPath := filepath.Join(homeDir, ".npmrc")
		globalConfig, err := parseNpmrcFile(globalNpmrcPath)
		if err == nil {
			// Copy global config
			maps.Copy(config.ScopeRegistries, globalConfig.ScopeRegistries)
			maps.Copy(config.AuthTokens, globalConfig.AuthTokens)
		}
	}

	// Parse local .npmrc (overrides global for scope registries)
	localConfig, err := parseNpmrcFile(localPath)
	if err != nil {
		// If local file doesn't exist, that's okay, we still have global config
		if !os.IsNotExist(err) {
			return nil, err
		}
	} else {
		// Local scope registries override global ones
		maps.Copy(config.ScopeRegistries, localConfig.ScopeRegistries)
		// Local auth tokens override global ones
		maps.Copy(config.AuthTokens, localConfig.AuthTokens)
	}

	return config, nil
}

func parseNpmrcFile(filePath string) (*NpmrcConfig, error) {
	config := &NpmrcConfig{
		ScopeRegistries: make(map[string]string),
		AuthTokens:      make(map[string]string),
	}

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty config if file doesn't exist
			return config, nil
		}
		return nil, fmt.Errorf("failed to open .npmrc file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// Parse scope registry: @scope:registry=https://registry.example.com
		if strings.Contains(line, ":registry=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				registry := strings.TrimSpace(parts[1])

				// Extract scope from @scope:registry
				if strings.HasSuffix(key, ":registry") {
					scope := strings.TrimSuffix(key, ":registry")
					config.ScopeRegistries[scope] = registry
				}
			}
		}

		// Parse auth tokens: //registry.example.com/:_authToken=token
		if strings.Contains(line, ":_authToken=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				token := strings.Trim(strings.TrimSpace(parts[1]), "\"'")

				// Extract registry from //registry.example.com/:_authToken
				if strings.HasSuffix(key, ":_authToken") {
					registry := strings.TrimSuffix(key, ":_authToken")
					registry = strings.TrimPrefix(registry, "//")
					registry = strings.TrimSuffix(registry, "/")
					config.AuthTokens[registry] = token
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading .npmrc file: %w", err)
	}

	return config, nil
}

// getRegistryForPackage determines the appropriate registry URL for a package
func getRegistryForPackage(packageName string, npmrcConfig *NpmrcConfig) string {
	// Check if it's a scoped package
	if strings.HasPrefix(packageName, "@") {
		if idx := strings.Index(packageName[1:], "/"); idx != -1 {
			scope := packageName[:idx+1] // Include @ but not the /
			if registry, exists := npmrcConfig.ScopeRegistries[scope]; exists {
				return registry
			}
		}
	}

	// Default to public npm registry
	return "https://registry.npmjs.org"
}

// CheckOutdated checks which dependencies have newer versions available
func CheckOutdated(dependencies []parser.Dependency, fileType string, verbose bool) ([]OutdatedDependency, error) {
	result, err := CheckOutdatedWithProgress(dependencies, fileType, verbose, false, nil)
	if err != nil {
		return nil, err
	}
	return result.Outdated, nil
}

// CheckOutdatedWithProgress checks which dependencies have newer versions available with progress callback
func CheckOutdatedWithProgress(dependencies []parser.Dependency, fileType string, verbose bool, semver bool, progressCallback func(int, int)) (*CheckResult, error) {
	var outdated []OutdatedDependency
	var errors []DependencyError
	var semverSkipped []SemverSkipped
	total := len(dependencies)

	for i, dep := range dependencies {
		// Update progress
		if progressCallback != nil {
			progressCallback(i+1, total)
		}

		// Skip complex dependencies (git, path, etc.)
		if strings.HasPrefix(dep.Version, "git:") || strings.HasPrefix(dep.Version, "path:") || dep.Version == "complex" {
			if verbose {
				fmt.Printf("Skipping complex dependency: %s (%s)\n", dep.Name, dep.Version)
			}
			continue
		}

		// If semver flag is enabled and it's a hardcoded version (no prefix), skip it
		if semver && getVersionPrefix(dep.OriginalVersion) == "" {
			if verbose {
				fmt.Printf("Skipping hardcoded version: %s (%s)\n", dep.Name, dep.OriginalVersion)
			}
			// We don't need to fetch the latest version for hardcoded versions
			semverSkipped = append(semverSkipped, SemverSkipped{
				Name:            dep.Name,
				CurrentVersion:  dep.Version,
				LatestVersion:   "", // Not fetched
				OriginalVersion: dep.OriginalVersion,
				Reason:          "hardcoded version",
			})
			continue
		}

		latestVersion, err := getLatestVersion(dep.Name, fileType, verbose)
		if err != nil {
			errors = append(errors, DependencyError{
				Name:  dep.Name,
				Error: err.Error(),
			})
			if verbose {
				fmt.Printf("Error checking %s: %v\n", dep.Name, err)
			}
			continue
		}

		currentVersion := dep.Version // Already cleaned in parser
		if currentVersion != latestVersion && latestVersion != "" {
			// If semver flag is enabled, check if the latest version is compatible
			if semver && !isSemverCompatible(dep.OriginalVersion, latestVersion) {
				if verbose {
					fmt.Printf("Skipping %s: latest version %s not compatible with constraint %s\n",
						dep.Name, latestVersion, dep.OriginalVersion)
				}
				semverSkipped = append(semverSkipped, SemverSkipped{
					Name:            dep.Name,
					CurrentVersion:  currentVersion,
					LatestVersion:   latestVersion,
					OriginalVersion: dep.OriginalVersion,
					Reason:          "incompatible with constraint",
				})
				continue
			}

			outdated = append(outdated, OutdatedDependency{
				Name:            dep.Name,
				CurrentVersion:  currentVersion,
				LatestVersion:   latestVersion,
				OriginalVersion: dep.OriginalVersion,
			})
		}
	}

	return &CheckResult{
		Outdated:      outdated,
		Errors:        errors,
		SemverSkipped: semverSkipped,
	}, nil
}

// getLatestVersion fetches the latest version of a package
func getLatestVersion(packageName, fileType string, verbose bool) (string, error) {
	switch fileType {
	case "npm":
		return getNpmLatestVersion(packageName, verbose)
	case "dart":
		return getPubDevLatestVersion(packageName, verbose)
	default:
		return "", fmt.Errorf("unsupported file type: %s", fileType)
	}
}

// getNpmLatestVersion fetches the latest version from NPM registry
func getNpmLatestVersion(packageName string, verbose bool) (string, error) {
	// Parse .npmrc configuration from both local and global files
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	npmrcPath := filepath.Join(cwd, ".npmrc")
	npmrcConfig, err := parseNpmrcFiles(npmrcPath)
	if err != nil {
		return "", fmt.Errorf("failed to parse .npmrc: %w", err)
	}

	// Get the appropriate registry for this package
	registryURL := getRegistryForPackage(packageName, npmrcConfig)
	url := fmt.Sprintf("%s/%s", registryURL, packageName)

	if verbose {
		fmt.Printf("Checking NPM package: %s (registry: %s)\n", packageName, registryURL)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication if available for this registry
	if authToken := getAuthTokenForRegistry(registryURL, npmrcConfig); authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
		if verbose {
			fmt.Printf("Using authentication for registry: %s\n", registryURL)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch package info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("registry returned status %d for %s", resp.StatusCode, packageName)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var packageInfo NpmPackageInfo
	if err := json.Unmarshal(body, &packageInfo); err != nil {
		return "", fmt.Errorf("failed to parse NPM response: %w", err)
	}

	if latest, ok := packageInfo.DistTags["latest"]; ok {
		return latest, nil
	}

	return "", fmt.Errorf("no latest version found for %s", packageName)
}

// getAuthTokenForRegistry finds the appropriate auth token for a registry URL
func getAuthTokenForRegistry(registryURL string, npmrcConfig *NpmrcConfig) string {
	// Extract hostname from registry URL for matching
	if after, ok := strings.CutPrefix(registryURL, "https://"); ok {
		hostname := after
		hostname = strings.TrimSuffix(hostname, "/")

		if token, exists := npmrcConfig.AuthTokens[hostname]; exists {
			return token
		}
	}

	return ""
}

// getPubDevLatestVersion fetches the latest version from pub.dev API
func getPubDevLatestVersion(packageName string, verbose bool) (string, error) {
	url := fmt.Sprintf("https://pub.dev/api/packages/%s", packageName)

	if verbose {
		fmt.Printf("Checking pub.dev package: %s\n", packageName)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch package info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("pub.dev API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var packageInfo PubDevPackageInfo
	if err := json.Unmarshal(body, &packageInfo); err != nil {
		return "", fmt.Errorf("failed to parse pub.dev response: %w", err)
	}

	return packageInfo.Latest.Version, nil
}

// UpdateDependencies updates the dependencies in the file
func UpdateDependencies(filePath string, outdated []OutdatedDependency, fileType string, verbose bool, semver bool) error {
	switch fileType {
	case "npm":
		return updatePackageJson(filePath, outdated, verbose, semver)
	case "dart":
		return updatePubspecYaml(filePath, outdated, verbose, semver)
	default:
		return fmt.Errorf("unsupported file type: %s", fileType)
	}
}

// updatePackageJson updates dependencies in a package.json file
func updatePackageJson(filePath string, outdated []OutdatedDependency, verbose bool, semver bool) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Convert to string for regex replacement
	content := string(data)

	// Update each outdated dependency using regex
	for _, dep := range outdated {
		// Escape special regex characters in package name
		escapedName := regexp.QuoteMeta(dep.Name)

		// Pattern to match the dependency line: "package-name": "version"
		pattern := fmt.Sprintf(`"%s"\s*:\s*"([^"]*)"`, escapedName)
		re := regexp.MustCompile(pattern)

		// Find and replace
		matches := re.FindStringSubmatch(content)
		if len(matches) > 1 {
			oldVersion := matches[1]
			prefix := getVersionPrefix(oldVersion)
			newVersion := prefix + dep.LatestVersion

			// Replace the version while keeping the same structure
			replacement := fmt.Sprintf(`"%s": "%s"`, dep.Name, newVersion)
			content = re.ReplaceAllString(content, replacement)

			if verbose {
				fmt.Printf("Updated %s: %s -> %s\n", dep.Name, oldVersion, newVersion)
			}
		}
	}

	// Write back to file
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// updatePubspecYaml updates dependencies in a pubspec.yaml file
func updatePubspecYaml(filePath string, outdated []OutdatedDependency, verbose bool, semver bool) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var pubspec parser.PubspecYaml
	if err := yaml.Unmarshal(data, &pubspec); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Update dependencies
	for _, dep := range outdated {
		if pubspec.Dependencies != nil {
			if oldVersionInterface, exists := pubspec.Dependencies[dep.Name]; exists {
				if oldVersion, ok := oldVersionInterface.(string); ok {
					prefix := getVersionPrefix(oldVersion)
					pubspec.Dependencies[dep.Name] = prefix + dep.LatestVersion
					if verbose {
						fmt.Printf("Updated %s: %s -> %s\n", dep.Name, oldVersion, pubspec.Dependencies[dep.Name])
					}
				}
			}
		}
		if pubspec.DevDependencies != nil {
			if oldVersionInterface, exists := pubspec.DevDependencies[dep.Name]; exists {
				if oldVersion, ok := oldVersionInterface.(string); ok {
					prefix := getVersionPrefix(oldVersion)
					pubspec.DevDependencies[dep.Name] = prefix + dep.LatestVersion
					if verbose {
						fmt.Printf("Updated %s: %s -> %s\n", dep.Name, oldVersion, pubspec.DevDependencies[dep.Name])
					}
				}
			}
		}
	}

	// Write back to file
	updatedData, err := yaml.Marshal(pubspec)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	if err := os.WriteFile(filePath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// getVersionPrefix extracts the version prefix (^, ~, >=, etc.) from a version string
func getVersionPrefix(version string) string {
	re := regexp.MustCompile(`^([\^~>=<]+)`)
	matches := re.FindStringSubmatch(version)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// isSemverCompatible checks if the latest version is compatible with the original version constraint
func isSemverCompatible(originalVersion, latestVersion string) bool {
	prefix := getVersionPrefix(originalVersion)

	// If no prefix, it's a hardcoded version - not compatible with semver
	if prefix == "" {
		return false
	}

	// If the latest version contains pre-release identifiers, be conservative and skip it
	if strings.Contains(latestVersion, "-") {
		return false
	}

	// Parse current and latest versions
	currentVer, err := parseSemanticVersion(parser.CleanVersion(originalVersion))
	if err != nil {
		return false
	}

	latestVer, err := parseSemanticVersion(latestVersion)
	if err != nil {
		return false
	}

	switch prefix {
	case "^":
		// Caret allows changes that do not modify the left-most non-zero digit
		if currentVer.Major == 0 {
			if currentVer.Minor == 0 {
				// ^0.0.x - only patch-level changes
				return latestVer.Major == 0 && latestVer.Minor == 0 && latestVer.Patch >= currentVer.Patch
			}
			// ^0.x.y - minor and patch-level changes
			return latestVer.Major == 0 && latestVer.Minor >= currentVer.Minor
		}
		// ^x.y.z - minor and patch-level changes
		return latestVer.Major == currentVer.Major &&
			(latestVer.Minor > currentVer.Minor ||
				(latestVer.Minor == currentVer.Minor && latestVer.Patch >= currentVer.Patch))

	case "~":
		// Tilde allows patch-level changes if a minor version is specified
		// ~1.2.3 := >=1.2.3 <1.3.0 (reasonably close to 1.2.3)
		return latestVer.Major == currentVer.Major &&
			latestVer.Minor == currentVer.Minor &&
			latestVer.Patch >= currentVer.Patch

	default:
		// For other prefixes like >=, >, <, <=, we'll be conservative and not update
		return false
	}
}

// SemanticVersion represents a parsed semantic version
type SemanticVersion struct {
	Major int
	Minor int
	Patch int
}

// parseSemanticVersion parses a semantic version string into components
func parseSemanticVersion(version string) (*SemanticVersion, error) {
	// Handle pre-release and build metadata by splitting on '-' and '+'
	parts := strings.Split(version, "-")
	version = parts[0] // Take only the main version part

	parts = strings.Split(version, "+")
	version = parts[0] // Remove build metadata

	parts = strings.Split(version, ".")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid semantic version: %s", version)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid major version: %s", parts[0])
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid minor version: %s", parts[1])
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid patch version: %s", parts[2])
	}

	return &SemanticVersion{
		Major: major,
		Minor: minor,
		Patch: patch,
	}, nil
}
