package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
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
	Outdated []OutdatedDependency
	Errors   []DependencyError
}

// DependencyError represents an error that occurred while checking a dependency
type DependencyError struct {
	Name  string
	Error string
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

// CheckOutdated checks which dependencies have newer versions available
func CheckOutdated(dependencies []parser.Dependency, fileType string, verbose bool) ([]OutdatedDependency, error) {
	result, err := CheckOutdatedWithProgress(dependencies, fileType, verbose, nil)
	return result.Outdated, err
}

// CheckOutdatedWithProgress checks which dependencies have newer versions available with progress callback
func CheckOutdatedWithProgress(dependencies []parser.Dependency, fileType string, verbose bool, progressCallback func(int, int)) (*CheckResult, error) {
	var outdated []OutdatedDependency
	var errors []DependencyError
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
			outdated = append(outdated, OutdatedDependency{
				Name:            dep.Name,
				CurrentVersion:  currentVersion,
				LatestVersion:   latestVersion,
				OriginalVersion: dep.OriginalVersion,
			})
		}
	}

	return &CheckResult{
		Outdated: outdated,
		Errors:   errors,
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
	url := fmt.Sprintf("https://registry.npmjs.org/%s", packageName)

	if verbose {
		fmt.Printf("Checking NPM package: %s\n", packageName)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch package info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("NPM registry returned status %d", resp.StatusCode)
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
func UpdateDependencies(filePath string, outdated []OutdatedDependency, fileType string, verbose bool) error {
	switch fileType {
	case "npm":
		return updatePackageJson(filePath, outdated, verbose)
	case "dart":
		return updatePubspecYaml(filePath, outdated, verbose)
	default:
		return fmt.Errorf("unsupported file type: %s", fileType)
	}
}

// updatePackageJson updates dependencies in a package.json file
func updatePackageJson(filePath string, outdated []OutdatedDependency, verbose bool) error {
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
func updatePubspecYaml(filePath string, outdated []OutdatedDependency, verbose bool) error {
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
