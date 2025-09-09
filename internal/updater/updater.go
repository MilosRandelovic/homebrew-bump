package updater

import (
	"fmt"
	"strings"

	"github.com/MilosRandelovic/homebrew-bump/internal/npm"
	"github.com/MilosRandelovic/homebrew-bump/internal/pub"
	"github.com/MilosRandelovic/homebrew-bump/internal/shared"
)

// CheckOutdated checks which dependencies have newer versions available
func CheckOutdated(dependencies []shared.Dependency, fileType string, verbose bool) ([]shared.OutdatedDependency, error) {
	result, err := CheckOutdatedWithProgress(dependencies, fileType, verbose, false, false, nil)
	if err != nil {
		return nil, err
	}
	return result.Outdated, nil
}

// CheckOutdatedWithProgress checks which dependencies have newer versions available with progress callback
func CheckOutdatedWithProgress(dependencies []shared.Dependency, fileType string, verbose bool, semver bool, noCache bool, progressCallback func(int, int)) (*shared.CheckResult, error) {
	var outdated []shared.OutdatedDependency
	var errors []shared.DependencyError
	var semverSkipped []shared.SemverSkipped
	total := len(dependencies)

	// Initialize cache if not disabled
	var cache *shared.Cache
	if !noCache {
		cache = shared.NewCache()
	}

	// Get the appropriate registry client
	registryClient, err := getRegistryClient(fileType)
	if err != nil {
		return nil, err
	}

	for i, dependency := range dependencies {
		// Update progress
		if progressCallback != nil {
			progressCallback(i+1, total)
		}

		// Skip complex dependencies (git, path, etc.)
		if strings.HasPrefix(dependency.Version, "git:") || strings.HasPrefix(dependency.Version, "path:") || dependency.Version == "complex" {
			if verbose {
				fmt.Printf("Skipping complex dependency: %s (%s)\n", dependency.Name, dependency.Version)
			}
			continue
		}

		// If semver flag is enabled and it's a hardcoded version (no prefix), skip it
		if semver && shared.GetVersionPrefix(dependency.OriginalVersion) == "" {
			if verbose {
				fmt.Printf("Skipping hardcoded version: %s (%s)\n", dependency.Name, dependency.OriginalVersion)
			}
			// We don't need to fetch the latest version for hardcoded versions
			semverSkipped = append(semverSkipped, shared.SemverSkipped{
				Name:            dependency.Name,
				CurrentVersion:  dependency.Version,
				LatestVersion:   "", // Not fetched
				OriginalVersion: dependency.OriginalVersion,
				Reason:          "hardcoded version",
			})
			continue
		}

		var latestVersion string
		var absoluteLatest string
		var err error

		// If semver flag is enabled and we have a prefixed version, get both versions in one call
		if semver && shared.HasSemanticPrefix(dependency.OriginalVersion) {
			absoluteLatest, latestVersion, err = registryClient.GetBothLatestVersions(dependency.Name, dependency.OriginalVersion, dependency.HostedURL, verbose, cache)
			if err != nil {
				// If constraint error, use the absolute latest already returned for semver skipped
				if strings.Contains(err.Error(), "no versions satisfy the constraint") && absoluteLatest != "" {
					// Add to semver skipped since constraint is incompatible but package exists
					semverSkipped = append(semverSkipped, shared.SemverSkipped{
						Name:            dependency.Name,
						CurrentVersion:  dependency.Version,
						LatestVersion:   absoluteLatest,
						OriginalVersion: dependency.OriginalVersion,
						Reason:          "incompatible with constraint",
					})
					continue
				}
				// If we can't get latest version or it's a different error, treat as error
				if verbose {
					fmt.Printf("Error checking %s: %v\n", dependency.Name, err)
				}
				errors = append(errors, shared.DependencyError{
					Name:  dependency.Name,
					Error: err.Error(),
				})
				continue
			}
		} else {
			// Check if current version is pre-release to determine which method to use
			if strings.Contains(dependency.Version, "-") {
				// Current version is pre-release, so we need to check all versions including pre-releases
				// Use the original version as constraint (even if it has no prefix)
				absoluteLatest, latestVersion, err = registryClient.GetBothLatestVersions(dependency.Name, dependency.OriginalVersion, dependency.HostedURL, verbose, cache)
				if err != nil {
					// If constraint error for hardcoded pre-release, treat as up-to-date
					if strings.Contains(err.Error(), "no versions satisfy the constraint") {
						if verbose {
							fmt.Printf("No newer versions found for pre-release: %s (%s)\n", dependency.Name, dependency.OriginalVersion)
						}
						continue
					}
					errors = append(errors, shared.DependencyError{
						Name:  dependency.Name,
						Error: err.Error(),
					})
					if verbose {
						fmt.Printf("Error checking %s: %v\n", dependency.Name, err)
					}
					continue
				}
			} else {
				// Use absolute latest version fetching for stable versions (non-semver cases)
				latestVersion, err = registryClient.GetLatestVersionFromRegistry(dependency.Name, dependency.HostedURL, verbose, cache)
				if err != nil {
					errors = append(errors, shared.DependencyError{
						Name:  dependency.Name,
						Error: err.Error(),
					})
					if verbose {
						fmt.Printf("Error checking %s: %v\n", dependency.Name, err)
					}
					continue
				}
			}
			absoluteLatest = latestVersion // Same as latest when not using semver constraints
		}

		currentVersion := dependency.Version // Already cleaned in parser

		// Check if there's an update available
		if currentVersion != latestVersion && latestVersion != "" {
			outdated = append(outdated, shared.OutdatedDependency{
				Name:            dependency.Name,
				CurrentVersion:  currentVersion,
				LatestVersion:   latestVersion,
				OriginalVersion: dependency.OriginalVersion,
				HostedURL:       dependency.HostedURL,
				Type:            dependency.Type,
				LineNumber:      dependency.LineNumber,
			})
		}

		// If semver is enabled and we have both versions, check if there's a newer incompatible version
		if semver && shared.HasSemanticPrefix(dependency.OriginalVersion) && absoluteLatest != latestVersion {
			if verbose {
				fmt.Printf("Note: %s has newer version %s available, but it's incompatible with constraint %s (using %s)\n",
					dependency.Name, absoluteLatest, dependency.OriginalVersion, latestVersion)
			}
			// Add to semverSkipped if the absolute latest is a major version jump
			semverSkipped = append(semverSkipped, shared.SemverSkipped{
				Name:            dependency.Name,
				CurrentVersion:  currentVersion,
				LatestVersion:   absoluteLatest,
				OriginalVersion: dependency.OriginalVersion,
				Reason:          "incompatible with constraint",
			})
		}
	}

	// Save cache if it was used
	if cache != nil {
		// Clean expired entries before saving
		cache.CleanExpiredEntries()
		if err := cache.SaveEntries(); err != nil && verbose {
			fmt.Printf("Warning: Could not save cache: %v\n", err)
		}
	}

	return &shared.CheckResult{
		Outdated:      outdated,
		Errors:        errors,
		SemverSkipped: semverSkipped,
	}, nil
}

// UpdateDependencies updates the dependencies in the file
func UpdateDependencies(filePath string, outdated []shared.OutdatedDependency, fileType string, verbose bool, semver bool, includePeerDependencies bool) error {
	updater, err := getUpdater(fileType)
	if err != nil {
		return err
	}
	return updater.UpdateDependencies(filePath, outdated, verbose, semver, includePeerDependencies)
}

// getRegistryClient returns the appropriate registry client for the given file type
func getRegistryClient(fileType string) (shared.RegistryClient, error) {
	switch fileType {
	case "npm":
		return npm.NewRegistryClient(), nil
	case "pub":
		return pub.NewRegistryClient(), nil
	default:
		return nil, fmt.Errorf("unsupported file type: %s", fileType)
	}
}

// getUpdater returns the appropriate updater for the given file type
func getUpdater(fileType string) (shared.Updater, error) {
	switch fileType {
	case "npm":
		return npm.NewUpdater(), nil
	case "pub":
		return pub.NewUpdater(), nil
	default:
		return nil, fmt.Errorf("unsupported file type: %s", fileType)
	}
}
