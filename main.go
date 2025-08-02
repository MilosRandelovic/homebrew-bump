package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/MilosRandelovic/homebrew-bump/internal/parser"
	"github.com/MilosRandelovic/homebrew-bump/internal/updater"
)

const version = "1.0.0"

func main() {
	var (
		update      = flag.Bool("update", false, "Update dependencies to latest versions")
		updateShort = flag.Bool("u", false, "Update dependencies to latest versions (shorthand)")
		verbose     = flag.Bool("verbose", false, "Enable verbose output")
		verboseShort = flag.Bool("v", false, "Enable verbose output (shorthand)")
		showVersion = flag.Bool("version", false, "Show version information")
		versionShort = flag.Bool("V", false, "Show version information (shorthand)")
		help        = flag.Bool("help", false, "Show help information")
		helpShort   = flag.Bool("h", false, "Show help information (shorthand)")
	)
	flag.Parse()

	// Handle shorthand flags
	if *updateShort {
		*update = true
	}
	if *verboseShort {
		*verbose = true
	}
	if *versionShort {
		*showVersion = true
	}
	if *helpShort {
		*help = true
	}

	if *showVersion {
		fmt.Printf("bump version %s\n", version)
		os.Exit(0)
	}

	if *help {
		fmt.Printf("bump v%s - A utility to check and update dependencies\n\n", version)
		fmt.Println("Usage: bump [options]")
		fmt.Println("\nAuto-detects package.json or pubspec.yaml in current directory")
		fmt.Println("Automatically checks for outdated dependencies")
		fmt.Println("\nSupported files:")
		fmt.Println("  package.json  - NPM dependencies")
		fmt.Println("  pubspec.yaml  - Dart/Flutter dependencies")
		fmt.Println("\nOptions:")
		fmt.Println("  -update, -u     Update dependencies to latest versions")
		fmt.Println("  -verbose, -v    Enable verbose output")
		fmt.Println("  -version, -V    Show version information")
		fmt.Println("  -help, -h       Show this help")
		fmt.Println("\nExamples:")
		fmt.Println("  bump            # Check for outdated dependencies")
		fmt.Println("  bump -update    # Update dependencies to latest versions")
		fmt.Println("  bump -u -v      # Update with verbose output")
		os.Exit(0)
	}

	// Auto-detect dependency file in current directory
	filePath, fileType, err := autoDetectDependencyFile()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	if *verbose {
		fmt.Printf("Found %s file: %s\n", fileType, filePath)
	}

	// Parse the file
	dependencies, err := parser.ParseDependencies(filePath, fileType)
	if err != nil {
		log.Fatalf("Error parsing file: %v", err)
	}

	if *verbose {
		fmt.Printf("Found %d dependencies\n", len(dependencies))
	}

	// Always check for outdated dependencies
	outdated, err := updater.CheckOutdated(dependencies, fileType, *verbose)
	if err != nil {
		log.Fatalf("Error checking for updates: %v", err)
	}

	if len(outdated) == 0 {
		fmt.Println("All dependencies are up to date!")
		return
	}

	fmt.Printf("Found %d outdated dependencies:\n", len(outdated))
	for _, dep := range outdated {
		fmt.Printf("  %s: %s -> %s\n", dep.Name, dep.CurrentVersion, dep.LatestVersion)
	}

	// Update if requested
	if *update {
		err := updater.UpdateDependencies(filePath, outdated, fileType, *verbose)
		if err != nil {
			log.Fatalf("Error updating dependencies: %v", err)
		}
		fmt.Println("Dependencies updated successfully!")
	} else {
		fmt.Printf("\nRun 'bump -update' to update dependencies to latest versions.\n")
	}
}

// autoDetectDependencyFile looks for package.json or pubspec.yaml in the current directory
func autoDetectDependencyFile() (string, string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Check for package.json first
	packageJson := filepath.Join(cwd, "package.json")
	if _, err := os.Stat(packageJson); err == nil {
		return packageJson, "npm", nil
	}

	// Check for pubspec.yaml
	pubspecYaml := filepath.Join(cwd, "pubspec.yaml")
	if _, err := os.Stat(pubspecYaml); err == nil {
		return pubspecYaml, "dart", nil
	}

	return "", "", fmt.Errorf("no package.json or pubspec.yaml found in current directory")
}
