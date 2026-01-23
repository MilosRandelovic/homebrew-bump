package parser

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/MilosRandelovic/homebrew-bump/internal/shared"
)

// AutoDetectDependencyFile looks for package.json or pubspec.yaml in the current directory
func AutoDetectDependencyFile(options shared.Options) (string, shared.RegistryType, error) {
	currentWorkingDir, err := os.Getwd()
	if err != nil {
		return "", 0, fmt.Errorf("failed to get current directory: %w", err)
	}

	// Check for package.json first
	packageJson := filepath.Join(currentWorkingDir, "package.json")
	if _, err := os.Stat(packageJson); err == nil {
		if options.Verbose {
			relPath, err := filepath.Rel(currentWorkingDir, packageJson)
			if err != nil {
				relPath = packageJson
			}
			fmt.Printf("Found npm file: %s\n", relPath)
		}
		return packageJson, shared.Npm, nil
	}

	// Check for pubspec.yaml
	pubspecYaml := filepath.Join(currentWorkingDir, "pubspec.yaml")
	if _, err := os.Stat(pubspecYaml); err == nil {
		if options.Verbose {
			relPath, err := filepath.Rel(currentWorkingDir, pubspecYaml)
			if err != nil {
				relPath = pubspecYaml
			}
			fmt.Printf("Found pub file: %s\n", relPath)
		}
		return pubspecYaml, shared.Pub, nil
	}

	return "", 0, fmt.Errorf("no package.json or pubspec.yaml found in current directory")
}
