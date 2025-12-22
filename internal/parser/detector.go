package parser

import (
	"fmt"
	"os"
	"path/filepath"
)

// AutoDetectDependencyFile looks for package.json or pubspec.yaml in the current directory
func AutoDetectDependencyFile() (string, string, error) {
	currentWorkingDir, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Check for package.json first
	packageJson := filepath.Join(currentWorkingDir, "package.json")
	if _, err := os.Stat(packageJson); err == nil {
		return packageJson, "npm", nil
	}

	// Check for pubspec.yaml
	pubspecYaml := filepath.Join(currentWorkingDir, "pubspec.yaml")
	if _, err := os.Stat(pubspecYaml); err == nil {
		return pubspecYaml, "pub", nil
	}

	return "", "", fmt.Errorf("no package.json or pubspec.yaml found in current directory")
}
