package pub

import (
	"fmt"
	"os"
	"regexp"

	"github.com/MilosRandelovic/homebrew-bump/internal/shared"
)

// Updater handles Dart pubspec.yaml updating
type Updater struct{}

// NewUpdater creates a new Dart updater
func NewUpdater() *Updater {
	return &Updater{}
}

// UpdateDependencies updates dependencies in a pubspec.yaml file using string replacement
func (u *Updater) UpdateDependencies(filePath string, outdated []shared.OutdatedDependency, verbose bool, semver bool) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	content := string(data)

	// Update each outdated dependency
	for _, dep := range outdated {
		var updated bool

		// For hosted packages, look for the version field within the hosted package block
		if dep.HostedURL != "" {
			// Pattern for hosted packages: look for version field after the package name
			hostedPattern := fmt.Sprintf(`(\s+%s:\s*\n(?:\s+hosted:[^\n]+\n)?\s+version:\s*)([^\s\n]+)`, regexp.QuoteMeta(dep.Name))
			hostedRe := regexp.MustCompile(hostedPattern)

			matches := hostedRe.FindStringSubmatch(content)
			if len(matches) >= 3 {
				currentVersionInFile := matches[2]
				prefix := shared.GetVersionPrefix(currentVersionInFile)
				newVersion := prefix + dep.LatestVersion

				replacement := matches[1] + newVersion
				content = hostedRe.ReplaceAllString(content, replacement)
				updated = true

				if verbose {
					fmt.Printf("Updated %s: %s -> %s\n", dep.Name, currentVersionInFile, newVersion)
				}
			}
		}

		// If not updated yet, try the simple pattern for regular dependencies
		if !updated {
			oldVersionPattern := fmt.Sprintf(`(\s+%s:\s*)([^\s\n]+)`, regexp.QuoteMeta(dep.Name))
			re := regexp.MustCompile(oldVersionPattern)

			matches := re.FindStringSubmatch(content)
			if len(matches) >= 3 {
				currentVersionInFile := matches[2]
				prefix := shared.GetVersionPrefix(currentVersionInFile)
				newVersion := prefix + dep.LatestVersion

				replacement := matches[1] + newVersion
				content = re.ReplaceAllString(content, replacement)
				updated = true

				if verbose {
					fmt.Printf("Updated %s: %s -> %s\n", dep.Name, currentVersionInFile, newVersion)
				}
			}
		}

		if !updated && verbose {
			fmt.Printf("Warning: Could not find %s in file for updating\n", dep.Name)
		}
	}

	// Write the updated content back to file
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// GetFileType returns the file type this updater handles
func (u *Updater) GetFileType() string {
	return "pub"
}

// Ensure Updater implements the interface
var _ shared.Updater = (*Updater)(nil)
