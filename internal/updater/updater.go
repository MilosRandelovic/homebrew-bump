package updater

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MilosRandelovic/homebrew-bump/internal/npm"
	"github.com/MilosRandelovic/homebrew-bump/internal/pub"
	"github.com/MilosRandelovic/homebrew-bump/internal/shared"
)

// CheckOutdated checks which dependencies have newer versions available
func CheckOutdated(dependencies []shared.Dependency, registryType shared.RegistryType, options shared.Options, progressCallback func(int, int)) (*shared.CheckResult, error) {
	var outdated []shared.OutdatedDependency
	var errors []shared.DependencyError
	var semverSkipped []shared.SemverSkipped

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

	cwd, _ := os.Getwd()

	// Add newline before checking phase in verbose mode
	if options.Verbose {
		fmt.Println()
	}

	for file, dependencies := range grouped {
		displayPath := file
		if relPath, err := filepath.Rel(cwd, file); err == nil {
			displayPath = relPath
		}
		fmt.Printf("Checking %s (%d dependencies)\n", displayPath, len(dependencies))

		for i, dependency := range dependencies {
			if progressCallback != nil {
				progressCallback(i+1, len(dependencies))
			}

			// Skip complex dependencies (git, path, workspace, etc.)
			if strings.HasPrefix(dependency.Version, "git:") || strings.HasPrefix(dependency.Version, "path:") || dependency.Version == "complex" || dependency.Version == "*" {
				if options.Verbose {
					fmt.Printf("Skipping complex dependency: %s (%s)\n", dependency.Name, dependency.Version)
				}
				continue
			}

			// If semver flag is enabled and it's a hardcoded version (no prefix), skip it
			if options.Semver && shared.GetVersionPrefix(dependency.OriginalVersion) == "" {
				if options.Verbose {
					fmt.Printf("Skipping hardcoded version: %s (%s)\n", dependency.Name, dependency.OriginalVersion)
				}
				// We don't need to fetch the latest version for hardcoded versions
				semverSkipped = append(semverSkipped, shared.SemverSkipped{
					OutdatedDependency: shared.OutdatedDependency{
						BaseDependency: shared.BaseDependency{
							Name:            dependency.Name,
							OriginalVersion: dependency.OriginalVersion,
							Type:            dependency.Type,
							FilePath:        dependency.FilePath,
							HostedURL:       dependency.HostedURL,
							LineNumber:      dependency.LineNumber,
						},
						CurrentVersion: dependency.Version,
						LatestVersion:  "", // Not fetched
					},
					Reason: "hardcoded version",
				})
				continue
			}

			var absoluteLatest string
			var constraintLatest string
			var err error

			// If semver flag is enabled and we have a prefixed version, get both versions in one call
			if options.Semver && shared.HasSemanticPrefix(dependency.OriginalVersion) {
				absoluteLatest, constraintLatest, err = registryClient.GetBothLatestVersions(dependency.Name, dependency.OriginalVersion, dependency.HostedURL, options, cache)
				if err != nil {
					// If constraint error, use the absolute latest already returned for semver skipped
					if strings.Contains(err.Error(), "no versions satisfy the constraint") && absoluteLatest != "" {
						// Add to semver skipped since constraint is incompatible but package exists
						semverSkipped = append(semverSkipped, shared.SemverSkipped{
							OutdatedDependency: shared.OutdatedDependency{
								BaseDependency: shared.BaseDependency{
									Name:            dependency.Name,
									OriginalVersion: dependency.OriginalVersion,
									Type:            dependency.Type,
									FilePath:        dependency.FilePath,
									HostedURL:       dependency.HostedURL,
									LineNumber:      dependency.LineNumber,
								},
								CurrentVersion: dependency.Version,
								LatestVersion:  absoluteLatest,
							},
							Reason: "incompatible with constraint",
						})
						continue
					}
					// If we can't get latest version or it's a different error, treat as error
					if options.Verbose {
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
					absoluteLatest, constraintLatest, err = registryClient.GetBothLatestVersions(dependency.Name, dependency.OriginalVersion, dependency.HostedURL, options, cache)
					if err != nil {
						// If constraint error for hardcoded pre-release, treat as up-to-date
						if strings.Contains(err.Error(), "no versions satisfy the constraint") {
							if options.Verbose {
								fmt.Printf("No newer versions found for pre-release: %s (%s)\n", dependency.Name, dependency.OriginalVersion)
							}
							continue
						}
						errors = append(errors, shared.DependencyError{
							Name:  dependency.Name,
							Error: err.Error(),
						})
						if options.Verbose {
							fmt.Printf("Error checking %s: %v\n", dependency.Name, err)
						}
						continue
					}
				} else {
					// Use absolute latest version fetching for stable versions (non-semver cases)
					constraintLatest, err = registryClient.GetLatestVersionFromRegistry(dependency.Name, dependency.HostedURL, options, cache)
					if err != nil {
						errors = append(errors, shared.DependencyError{
							Name:  dependency.Name,
							Error: err.Error(),
						})
						if options.Verbose {
							fmt.Printf("Error checking %s: %v\n", dependency.Name, err)
						}
						continue
					}
				}
				absoluteLatest = constraintLatest // Same as latest when not using semver constraints
			}

			currentVersion := dependency.Version // Already cleaned in parser

			// Check if there's an update available
			if currentVersion != constraintLatest && constraintLatest != "" {
				outdated = append(outdated, shared.OutdatedDependency{
					BaseDependency: shared.BaseDependency{
						Name:            dependency.Name,
						OriginalVersion: dependency.OriginalVersion,
						Type:            dependency.Type,
						FilePath:        dependency.FilePath,
						HostedURL:       dependency.HostedURL,
						LineNumber:      dependency.LineNumber,
					},
					CurrentVersion: currentVersion,
					LatestVersion:  constraintLatest,
				})
				// Add to semverSkipped if the absolute latest is a major version jump
				semverSkipped = append(semverSkipped, shared.SemverSkipped{
					OutdatedDependency: shared.OutdatedDependency{
						BaseDependency: shared.BaseDependency{
							Name:            dependency.Name,
							OriginalVersion: dependency.OriginalVersion,
							Type:            dependency.Type,
							FilePath:        dependency.FilePath,
							HostedURL:       dependency.HostedURL,
							LineNumber:      dependency.LineNumber,
						},
						CurrentVersion: currentVersion,
						LatestVersion:  absoluteLatest,
					},
					Reason: "incompatible with constraint",
				})
			}
		}
	}

	// Save cache if it was used
	if cache != nil {
		// Clean expired entries before saving
		cache.CleanExpiredEntries()
		if err := cache.SaveEntries(); err != nil && options.Verbose {
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

	for path, dependencies := range byFile {
		if err := updater.UpdateDependencies(path, dependencies, options); err != nil {
			return err
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
