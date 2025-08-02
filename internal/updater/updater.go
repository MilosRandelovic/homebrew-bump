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
	Name           string
	CurrentVersion string
	LatestVersion  string
}

// NPMPackageInfo represents the response from NPM registry
type NPMPackageInfo struct {
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
	var outdated []OutdatedDependency

	for _, dep := range dependencies {
		// Skip complex dependencies (git, path, etc.)
		if strings.HasPrefix(dep.Version, "git:") || strings.HasPrefix(dep.Version, "path:") || dep.Version == "complex" {
			if verbose {
				fmt.Printf("Skipping %s (complex dependency: %s)\n", dep.Name, dep.Version)
			}
			continue
		}

		latestVersion, err := getLatestVersion(dep.Name, fileType, verbose)
		if err != nil {
			if verbose {
				fmt.Printf("Error checking %s: %v\n", dep.Name, err)
			}
			continue
		}

		currentVersion := cleanVersion(dep.Version)
		if currentVersion != latestVersion && latestVersion != "" {
			outdated = append(outdated, OutdatedDependency{
				Name:           dep.Name,
				CurrentVersion: currentVersion,
				LatestVersion:  latestVersion,
			})
		}
	}

	return outdated, nil
}

// getLatestVersion fetches the latest version of a package
func getLatestVersion(packageName, fileType string, verbose bool) (string, error) {
	switch fileType {
	case "npm":
		return getNPMLatestVersion(packageName, verbose)
	case "dart":
		return getPubDevLatestVersion(packageName, verbose)
	default:
		return "", fmt.Errorf("unsupported file type: %s", fileType)
	}
}

// getNPMLatestVersion fetches the latest version from NPM registry
func getNPMLatestVersion(packageName string, verbose bool) (string, error) {
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

	var packageInfo NPMPackageInfo
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

// cleanVersion removes version prefixes like ^, ~, >=, etc.
func cleanVersion(version string) string {
	// Remove common version prefixes
	re := regexp.MustCompile(`^[\^~>=<]+`)
	return re.ReplaceAllString(version, "")
}

// UpdateDependencies updates the dependencies in the file
func UpdateDependencies(filePath string, outdated []OutdatedDependency, fileType string, verbose bool) error {
	switch fileType {
	case "npm":
		return updatePackageJSON(filePath, outdated, verbose)
	case "dart":
		return updatePubspecYAML(filePath, outdated, verbose)
	default:
		return fmt.Errorf("unsupported file type: %s", fileType)
	}
}

// updatePackageJSON updates dependencies in a package.json file
func updatePackageJSON(filePath string, outdated []OutdatedDependency, verbose bool) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var pkg parser.PackageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Update dependencies
	for _, dep := range outdated {
		if pkg.Dependencies != nil {
			if oldVersion, exists := pkg.Dependencies[dep.Name]; exists {
				prefix := getVersionPrefix(oldVersion)
				pkg.Dependencies[dep.Name] = prefix + dep.LatestVersion
				if verbose {
					fmt.Printf("Updated %s: %s -> %s\n", dep.Name, oldVersion, pkg.Dependencies[dep.Name])
				}
			}
		}
		if pkg.DevDependencies != nil {
			if oldVersion, exists := pkg.DevDependencies[dep.Name]; exists {
				prefix := getVersionPrefix(oldVersion)
				pkg.DevDependencies[dep.Name] = prefix + dep.LatestVersion
				if verbose {
					fmt.Printf("Updated %s: %s -> %s\n", dep.Name, oldVersion, pkg.DevDependencies[dep.Name])
				}
			}
		}
	}

	// Write back to file with proper formatting
	updatedData, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(filePath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// updatePubspecYAML updates dependencies in a pubspec.yaml file
func updatePubspecYAML(filePath string, outdated []OutdatedDependency, verbose bool) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var pubspec parser.PubspecYAML
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
