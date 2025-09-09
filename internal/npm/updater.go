package npm

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/MilosRandelovic/homebrew-bump/internal/shared"
)

// Updater handles NPM package.json updating
type Updater struct{}

// NewUpdater creates a new NPM updater
func NewUpdater() *Updater {
	return &Updater{}
}

// UpdateDependencies updates dependencies in a package.json file using line-based updates
func (updater *Updater) UpdateDependencies(filePath string, outdated []shared.OutdatedDependency, verbose bool, semver bool, includePeerDependencies bool) error {
	if verbose && len(outdated) > 0 {
		fmt.Printf("\n") // Add space before updates in verbose mode
	}

	// Filter dependencies based on includePeerDependencies flag
	var dependenciesToUpdate []shared.OutdatedDependency
	var skippedPeerDependencies []shared.OutdatedDependency

	for _, dependency := range outdated {
		if dependency.Type == shared.PeerDependencies && !includePeerDependencies {
			skippedPeerDependencies = append(skippedPeerDependencies, dependency)
			continue
		}
		dependenciesToUpdate = append(dependenciesToUpdate, dependency)
	}

	// Inform about skipped peer dependencies
	if verbose && len(skippedPeerDependencies) > 0 {
		fmt.Printf("Skipping peer dependencies (use --include-peer-dependencies to update):\n")
		for _, dep := range skippedPeerDependencies {
			fmt.Printf("  %s: %s -> %s (peer dependency)\n", dep.Name, dep.CurrentVersion, dep.LatestVersion)
		}
		fmt.Printf("\n")
	}

	// If no dependencies to update, return early
	if len(dependenciesToUpdate) == 0 {
		return nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Split content into lines
	lines := strings.Split(string(data), "\n")

	// Update each dependency by modifying its specific line
	for _, dependency := range dependenciesToUpdate {
		if dependency.LineNumber < 1 || dependency.LineNumber > len(lines) {
			return fmt.Errorf("invalid line number %d for dependency %s", dependency.LineNumber, dependency.Name)
		}

		lineIndex := dependency.LineNumber - 1 // Convert to 0-based index
		line := lines[lineIndex]

		// Simple regex to replace the version on this specific line
		// Look for: "package-name": "old-version"
		escapedName := regexp.QuoteMeta(dependency.Name)
		pattern := fmt.Sprintf(`("%s"\s*:\s*)"([^"]*)"`, escapedName)

		versionRegex := regexp.MustCompile(pattern)
		matches := versionRegex.FindStringSubmatch(line)

		if len(matches) < 3 {
			return fmt.Errorf("could not find %s on line %d for updating", dependency.Name, dependency.LineNumber)
		}

		oldVersion := matches[2]
		prefix := shared.GetVersionPrefix(oldVersion)
		newVersion := prefix + dependency.LatestVersion

		// Replace the version on this line
		newLine := versionRegex.ReplaceAllString(line, fmt.Sprintf(`${1}"%s"`, newVersion))
		lines[lineIndex] = newLine

		if verbose {
			fmt.Printf("Updated %s (%s): %s -> %s\n", dependency.Name, dependency.Type.String(), oldVersion, newVersion)
		}
	}

	// Join lines back together and write to file
	content := strings.Join(lines, "\n")
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
