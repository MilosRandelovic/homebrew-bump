package updater

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/MilosRandelovic/homebrew-bump/internal/npm"
	"github.com/MilosRandelovic/homebrew-bump/internal/output"
	"github.com/MilosRandelovic/homebrew-bump/internal/pub"
	"github.com/MilosRandelovic/homebrew-bump/internal/shared"
)

// checkResult accumulates results from checking individual dependencies
type checkResult struct {
	outdated      []shared.OutdatedDependency
	errors        []shared.DependencyError
	semverSkipped []shared.SemverSkipped
}

// CheckOutdated checks which dependencies have newer versions available
func CheckOutdated(dependencies []shared.Dependency, registryType shared.RegistryType, options shared.Options, progressCallback func(int, int)) (*shared.CheckResult, error) {
	// Initialize cache if not disabled
	var cache *shared.Cache
	if !options.NoCache {
		cache = shared.NewCache()
	}

	// Get the appropriate registry client
	registryClient, err := getRegistryClient(registryType)
	if err != nil {
		return nil, err
	}

	grouped := make(map[string][]shared.Dependency)
	for _, dependency := range dependencies {
		grouped[dependency.FilePath] = append(grouped[dependency.FilePath], dependency)
	}

	// Sort file paths for deterministic ordering
	filePaths := make([]string, 0, len(grouped))
	for file := range grouped {
		filePaths = append(filePaths, file)
	}
	shared.SortFilesByDepth(filePaths)

	cwd, _ := os.Getwd()

	// Add newline before checking phase in verbose mode
	output.VerbosePrintf(options, "\n")

	var result checkResult

	for _, file := range filePaths {
		fileDependencies := grouped[file]
		displayPath := file
		if relPath, err := filepath.Rel(cwd, file); err == nil {
			displayPath = relPath
		}
		dependencyWord := "dependencies"
		if len(fileDependencies) == 1 {
			dependencyWord = "dependency"
		}
		fmt.Printf("Checking %s (%d %s)\n", displayPath, len(fileDependencies), dependencyWord)

		for i, dependency := range fileDependencies {
			if progressCallback != nil {
				progressCallback(i+1, len(fileDependencies))
			}
			checkSingleDependency(dependency, registryClient, options, cache, &result)
		}
	}

	// Save cache if it was used
	if cache != nil {
		cache.CleanExpiredEntries()
		if err := cache.SaveEntries(); err != nil {
			output.VerbosePrintf(options, "Warning: Could not save cache: %v\n", err)
		}
	}

	return &shared.CheckResult{
		Outdated:      result.outdated,
		Errors:        result.errors,
		SemverSkipped: result.semverSkipped,
	}, nil
}

// checkSingleDependency checks a single dependency for updates and appends results
func checkSingleDependency(dependency shared.Dependency, registryClient shared.RegistryClient, options shared.Options, cache *shared.Cache, result *checkResult) {
	// Skip complex dependencies (git, path, workspace, etc.)
	if strings.HasPrefix(dependency.Version, "git:") || strings.HasPrefix(dependency.Version, "path:") || dependency.Version == "complex" || dependency.Version == "*" {
		output.VerbosePrintf(options, "Skipping complex dependency: %s (%s)\n", dependency.Name, dependency.Version)
		return
	}

	// If semver flag is enabled and it's a hardcoded version (no prefix), skip it
	if options.Semver && shared.GetVersionPrefix(dependency.OriginalVersion) == "" {
		output.VerbosePrintf(options, "Skipping hardcoded version: %s (%s)\n", dependency.Name, dependency.OriginalVersion)
		result.semverSkipped = append(result.semverSkipped, shared.SemverSkipped{
			OutdatedDependency: shared.OutdatedDependency{
				BaseDependency: dependency.BaseDependency,
				CurrentVersion: dependency.Version,
				LatestVersion:  "",
			},
			Reason: shared.HardcodedVersion,
		})
		return
	}

	absoluteLatest, constraintLatest, err := fetchLatestVersions(dependency, registryClient, options, cache)
	if err != nil {
		// If constraint error, use the absolute latest already returned for semver skipped
		if errors.Is(err, shared.ErrNoVersionsSatisfyConstraint) && absoluteLatest != "" {
			result.semverSkipped = append(result.semverSkipped, shared.SemverSkipped{
				OutdatedDependency: shared.OutdatedDependency{
					BaseDependency: dependency.BaseDependency,
					CurrentVersion: dependency.Version,
					LatestVersion:  absoluteLatest,
				},
				Reason: shared.IncompatibleWithConstraint,
			})
			return
		}
		// If constraint error for hardcoded pre-release, treat as up-to-date
		if errors.Is(err, shared.ErrNoVersionsSatisfyConstraint) {
			output.VerbosePrintf(options, "No newer versions found for pre-release: %s (%s)\n", dependency.Name, dependency.OriginalVersion)
			return
		}
		output.VerbosePrintf(options, "Error checking %s: %v\n", dependency.Name, err)
		result.errors = append(result.errors, shared.DependencyError{
			Name:  dependency.Name,
			Error: err.Error(),
		})
		return
	}

	currentVersion := dependency.Version

	// Check if there's an update available
	if currentVersion != constraintLatest && constraintLatest != "" {
		result.outdated = append(result.outdated, shared.OutdatedDependency{
			BaseDependency: dependency.BaseDependency,
			CurrentVersion: currentVersion,
			LatestVersion:  constraintLatest,
		})
	}

	// Add to semverSkipped if the absolute latest differs from the constraint-compatible latest
	if absoluteLatest != constraintLatest && absoluteLatest != "" {
		result.semverSkipped = append(result.semverSkipped, shared.SemverSkipped{
			OutdatedDependency: shared.OutdatedDependency{
				BaseDependency: dependency.BaseDependency,
				CurrentVersion: currentVersion,
				LatestVersion:  absoluteLatest,
			},
			Reason: shared.IncompatibleWithConstraint,
		})
	}
}

// fetchLatestVersions determines the appropriate strategy and fetches version info
func fetchLatestVersions(dependency shared.Dependency, registryClient shared.RegistryClient, options shared.Options, cache *shared.Cache) (absoluteLatest, constraintLatest string, err error) {
	// If semver flag is enabled and we have a prefixed version, get both versions in one call
	if options.Semver && shared.HasSemanticPrefix(dependency.OriginalVersion) {
		return registryClient.GetBothLatestVersions(dependency.Name, dependency.OriginalVersion, dependency.HostedURL, options, cache)
	}

	// Check if current version is pre-release to determine which method to use
	currentSemver, parseErr := semver.NewVersion(dependency.Version)
	if parseErr == nil && currentSemver.Prerelease() != "" {
		return registryClient.GetBothLatestVersions(dependency.Name, dependency.OriginalVersion, dependency.HostedURL, options, cache)
	}

	// Use absolute latest version fetching for stable versions (non-semver cases)
	latest, err := registryClient.GetLatestVersionFromRegistry(dependency.Name, dependency.HostedURL, options, cache)
	if err != nil {
		return "", "", err
	}
	return latest, latest, nil
}

// UpdateDependencies updates the dependencies in the file
func UpdateDependencies(filePath string, outdated []shared.OutdatedDependency, registryType shared.RegistryType, options shared.Options) error {
	byFile := make(map[string][]shared.OutdatedDependency)
	for _, dependency := range outdated {
		path := dependency.FilePath
		if path == "" {
			path = filePath
		}
		byFile[path] = append(byFile[path], dependency)
	}

	updater, err := getUpdater(registryType)
	if err != nil {
		return err
	}

	// Validate options before updating
	if err := updater.ValidateOptions(options); err != nil {
		return err
	}

	patternProvider := updater.GetPatternProvider()

	// Sort file paths for deterministic ordering
	filePaths := make([]string, 0, len(byFile))
	for path := range byFile {
		filePaths = append(filePaths, path)
	}
	shared.SortFilesByDepth(filePaths)

	showFilenames := len(filePaths) > 1

	for _, path := range filePaths {
		dependencies := byFile[path]
		if err := shared.UpdateDependenciesInFile(path, dependencies, patternProvider, options); err != nil {
			return err
		}
		if options.Verbose {
			output.VerbosePrintf(options, "\n")
			if showFilenames {
				displayPath := path
				if cwd, err := os.Getwd(); err == nil {
					if relPath, err := filepath.Rel(cwd, path); err == nil {
						displayPath = relPath
					}
				}
				output.VerbosePrintf(options, "%s:\n", displayPath)
			}
			for _, dependency := range dependencies {
				output.VerbosePrintf(options, "  Updated %s (%s): %s -> %s\n",
					dependency.Name,
					dependency.Type.String(),
					dependency.CurrentVersion,
					dependency.LatestVersion)
			}
		}
	}

	return nil
}

// getRegistryClient returns the appropriate registry client for the given registry type
func getRegistryClient(registryType shared.RegistryType) (shared.RegistryClient, error) {
	switch registryType {
	case shared.Npm:
		return npm.NewRegistryClient(), nil
	case shared.Pub:
		return pub.NewRegistryClient(), nil
	default:
		return nil, fmt.Errorf("unsupported registry type: %s", registryType)
	}
}

// getUpdater returns the appropriate updater for the given registry type
func getUpdater(registryType shared.RegistryType) (shared.Updater, error) {
	switch registryType {
	case shared.Npm:
		return npm.NewUpdater(), nil
	case shared.Pub:
		return pub.NewUpdater(), nil
	default:
		return nil, fmt.Errorf("unsupported registry type: %s", registryType)
	}
}
