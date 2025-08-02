package updater

import (
	"fmt"
	"strings"

	"github.com/MilosRandelovic/homebrew-bump/internal/dart"
	"github.com/MilosRandelovic/homebrew-bump/internal/npm"
	"github.com/MilosRandelovic/homebrew-bump/internal/shared"
)

// CheckOutdated checks which dependencies have newer versions available
func CheckOutdated(dependencies []shared.Dependency, fileType string, verbose bool) ([]shared.OutdatedDependency, error) {
	result, err := CheckOutdatedWithProgress(dependencies, fileType, verbose, false, nil)
	if err != nil {
		return nil, err
	}
	return result.Outdated, nil
}

// CheckOutdatedWithProgress checks which dependencies have newer versions available with progress callback
func CheckOutdatedWithProgress(dependencies []shared.Dependency, fileType string, verbose bool, semver bool, progressCallback func(int, int)) (*shared.CheckResult, error) {
	var outdated []shared.OutdatedDependency
	var errors []shared.DependencyError
	var semverSkipped []shared.SemverSkipped
	total := len(dependencies)

	// Get the appropriate registry client
	registryClient, err := getRegistryClient(fileType)
	if err != nil {
		return nil, err
	}

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
		if semver && shared.GetVersionPrefix(dep.OriginalVersion) == "" {
			if verbose {
				fmt.Printf("Skipping hardcoded version: %s (%s)\n", dep.Name, dep.OriginalVersion)
			}
			// We don't need to fetch the latest version for hardcoded versions
			semverSkipped = append(semverSkipped, shared.SemverSkipped{
				Name:            dep.Name,
				CurrentVersion:  dep.Version,
				LatestVersion:   "", // Not fetched
				OriginalVersion: dep.OriginalVersion,
				Reason:          "hardcoded version",
			})
			continue
		}

		latestVersion, err := registryClient.GetLatestVersion(dep.Name, verbose)
		if err != nil {
			errors = append(errors, shared.DependencyError{
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
			if semver && !shared.IsSemverCompatible(dep.OriginalVersion, latestVersion) {
				if verbose {
					fmt.Printf("Skipping %s: latest version %s not compatible with constraint %s\n",
						dep.Name, latestVersion, dep.OriginalVersion)
				}
				semverSkipped = append(semverSkipped, shared.SemverSkipped{
					Name:            dep.Name,
					CurrentVersion:  currentVersion,
					LatestVersion:   latestVersion,
					OriginalVersion: dep.OriginalVersion,
					Reason:          "incompatible with constraint",
				})
				continue
			}

			outdated = append(outdated, shared.OutdatedDependency{
				Name:            dep.Name,
				CurrentVersion:  currentVersion,
				LatestVersion:   latestVersion,
				OriginalVersion: dep.OriginalVersion,
			})
		}
	}

	return &shared.CheckResult{
		Outdated:      outdated,
		Errors:        errors,
		SemverSkipped: semverSkipped,
	}, nil
}

// UpdateDependencies updates the dependencies in the file
func UpdateDependencies(filePath string, outdated []shared.OutdatedDependency, fileType string, verbose bool, semver bool) error {
	updater, err := getUpdater(fileType)
	if err != nil {
		return err
	}
	return updater.UpdateDependencies(filePath, outdated, verbose, semver)
}

// getRegistryClient returns the appropriate registry client for the given file type
func getRegistryClient(fileType string) (shared.RegistryClient, error) {
	switch fileType {
	case "npm":
		return npm.NewRegistryClient(), nil
	case "dart":
		return dart.NewRegistryClient(), nil
	default:
		return nil, fmt.Errorf("unsupported file type: %s", fileType)
	}
}

// getUpdater returns the appropriate updater for the given file type
func getUpdater(fileType string) (shared.Updater, error) {
	switch fileType {
	case "npm":
		return npm.NewUpdater(), nil
	case "dart":
		return dart.NewUpdater(), nil
	default:
		return nil, fmt.Errorf("unsupported file type: %s", fileType)
	}
}
