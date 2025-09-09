package pub

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/MilosRandelovic/homebrew-bump/internal/shared"
)

// Updater handles Dart pubspec.yaml updating
type Updater struct{}

// NewUpdater creates a new Dart updater
func NewUpdater() *Updater {
	return &Updater{}
}

// UpdateDependencies updates dependencies in a pubspec.yaml file using line-based updates
func (updater *Updater) UpdateDependencies(filePath string, outdated []shared.OutdatedDependency, verbose bool, semver bool, includePeerDependencies bool) error {
	// Pub ecosystem doesn't support peer dependencies
	if includePeerDependencies {
		return fmt.Errorf("peer dependencies are not supported by pub")
	}

	// If no dependencies to update, return early
	if len(outdated) == 0 {
		return nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Split content into lines
	lines := strings.Split(string(data), "\n")

	// Update each dependency by modifying its specific line
	for _, dependency := range outdated {
		if dependency.LineNumber < 1 || dependency.LineNumber > len(lines) {
			return fmt.Errorf("invalid line number %d for dependency %s", dependency.LineNumber, dependency.Name)
		}

		lineIndex := dependency.LineNumber - 1 // Convert to 0-based index
		line := lines[lineIndex]

		// Simple regex to replace the version on this specific line
		// For hosted packages, this will be the "version: ^x.y.z" line
		// For simple packages, this will be the "package-name: ^x.y.z" line
		var pattern string
		if dependency.HostedURL != "" {
			// For hosted packages, match the version line
			pattern = `(\s*version\s*:\s*)([^\s#]+)`
		} else {
			// For simple packages, match the package name line
			escapedName := regexp.QuoteMeta(dependency.Name)
			pattern = fmt.Sprintf(`(\s*%s\s*:\s*)([^\s#]+)`, escapedName)
		}

		versionRegex := regexp.MustCompile(pattern)
		matches := versionRegex.FindStringSubmatch(line)

		if len(matches) < 3 {
			return fmt.Errorf("could not find %s on line %d for updating", dependency.Name, dependency.LineNumber)
		}

		oldVersion := matches[2]
		prefix := shared.GetVersionPrefix(oldVersion)
		newVersion := prefix + dependency.LatestVersion

		// Replace the version on this line
		newLine := versionRegex.ReplaceAllString(line, fmt.Sprintf(`${1}%s`, newVersion))
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
	return "pub"
}

// Ensure Updater implements the interface
var _ shared.Updater = (*Updater)(nil)
