package dart

import (
	"fmt"
	"os"

	"github.com/MilosRandelovic/homebrew-bump/internal/shared"
	"gopkg.in/yaml.v3"
)

// Updater handles Dart pubspec.yaml updating
type Updater struct{}

// NewUpdater creates a new Dart updater
func NewUpdater() *Updater {
	return &Updater{}
}

// UpdateDependencies updates dependencies in a pubspec.yaml file
func (u *Updater) UpdateDependencies(filePath string, outdated []shared.OutdatedDependency, verbose bool, semver bool) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var pubspec PubspecYaml
	if err := yaml.Unmarshal(data, &pubspec); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Update dependencies
	for _, dep := range outdated {
		if pubspec.Dependencies != nil {
			if oldVersionInterface, exists := pubspec.Dependencies[dep.Name]; exists {
				if oldVersion, ok := oldVersionInterface.(string); ok {
					prefix := shared.GetVersionPrefix(oldVersion)
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
					prefix := shared.GetVersionPrefix(oldVersion)
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

// GetFileType returns the file type this updater handles
func (u *Updater) GetFileType() string {
	return "dart"
}

// Ensure Updater implements the interface
var _ shared.Updater = (*Updater)(nil)
