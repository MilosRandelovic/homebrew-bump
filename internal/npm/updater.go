package npm

import (
	"fmt"
	"os"
	"regexp"

	"github.com/MilosRandelovic/homebrew-bump/internal/shared"
)

// Updater handles NPM package.json updating
type Updater struct{}

// NewUpdater creates a new NPM updater
func NewUpdater() *Updater {
	return &Updater{}
}

// UpdateDependencies updates dependencies in a package.json file
func (updater *Updater) UpdateDependencies(filePath string, outdated []shared.OutdatedDependency, verbose bool, semver bool) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Convert to string for regex replacement
	content := string(data)

	// Update each outdated dependency using regex
	for _, dependency := range outdated {
		// Escape special regex characters in package name
		escapedName := regexp.QuoteMeta(dependency.Name)

		// Pattern to match the dependency line: "package-name": "version"
		pattern := fmt.Sprintf(`"%s"\s*:\s*"([^"]*)"`, escapedName)
		versionRegex := regexp.MustCompile(pattern)

		// Find and replace
		matches := versionRegex.FindStringSubmatch(content)
		if len(matches) > 1 {
			oldVersion := matches[1]
			prefix := shared.GetVersionPrefix(oldVersion)
			newVersion := prefix + dependency.LatestVersion

			// Replace the version while keeping the same structure
			replacement := fmt.Sprintf(`"%s": "%s"`, dependency.Name, newVersion)
			content = versionRegex.ReplaceAllString(content, replacement)

			if verbose {
				fmt.Printf("Updated %s: %s -> %s\n", dependency.Name, oldVersion, newVersion)
			}
		} else if verbose {
			fmt.Printf("Warning: Could not find %s in file for updating\n", dependency.Name)
		}
	}

	// Write back to file
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// GetFileType returns the file type this updater handles
func (updater *Updater) GetFileType() string {
	return "npm"
}

// Ensure Updater implements the interface
var _ shared.Updater = (*Updater)(nil)
