package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/MilosRandelovic/homebrew-bump/internal/parser"
	"github.com/MilosRandelovic/homebrew-bump/internal/updater"
)

const version = "1.0.0"

// Color constants
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m" // Major version changes
	ColorYellow = "\033[33m" // Minor version changes
	ColorGreen  = "\033[32m" // Patch version changes
	ColorCyan   = "\033[36m" // Package names
	ColorBlue   = "\033[34m" // Arrows
)

// SemverChange represents the type of version change
type SemverChange int

const (
	PatchChange SemverChange = iota
	MinorChange
	MajorChange
)

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

	// Extract filename from path for display
	fileName := filepath.Base(filePath)

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
	fmt.Printf("Checking %d dependencies from %s...\n", len(dependencies), fileName)

	progressCallback := func(current, total int) {
		printProgressBar(current, total)
	}

	outdated, err := updater.CheckOutdatedWithProgress(dependencies, fileType, *verbose, progressCallback)
	if err != nil {
		log.Fatalf("Error checking for updates: %v", err)
	}

	if len(outdated) == 0 {
		fmt.Println("All dependencies are up to date!")
		return
	}

	fmt.Printf("\n") // Add space after progress bar

	// Sort outdated dependencies alphabetically by name
	sort.Slice(outdated, func(i, j int) bool {
		return outdated[i].Name < outdated[j].Name
	})

	// Display results with colors and proper formatting
	for _, dep := range outdated {
		change := getSemverChange(dep.CurrentVersion, dep.LatestVersion)
		color := getChangeColor(change)

		// Get the original version prefixes
		currentWithPrefix := getOriginalVersionString(dep.Name, filePath, fileType)
		latestWithPrefix := strings.Replace(currentWithPrefix, dep.CurrentVersion, dep.LatestVersion, 1)

		// Format output similar to ncu - only color the latest version
		fmt.Printf(" %s%-30s%s  %15s  →  %s%15s%s\n",
			ColorCyan, dep.Name, ColorReset,
			currentWithPrefix,
			color, latestWithPrefix, ColorReset)
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

// parseVersion extracts major, minor, patch from a version string
func parseVersion(version string) (int, int, int, error) {
	// Remove prefix characters like ^, ~, >=, etc.
	re := regexp.MustCompile(`^[\^~>=<]+`)
	cleanVer := re.ReplaceAllString(version, "")

	// Split by dots
	parts := strings.Split(cleanVer, ".")
	if len(parts) < 3 {
		return 0, 0, 0, fmt.Errorf("invalid version format: %s", version)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, err
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, err
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, err
	}

	return major, minor, patch, nil
}

// getSemverChange determines the type of version change
func getSemverChange(currentVer, latestVer string) SemverChange {
	currMajor, currMinor, currPatch, err1 := parseVersion(currentVer)
	latestMajor, latestMinor, latestPatch, err2 := parseVersion(latestVer)

	if err1 != nil || err2 != nil {
		return PatchChange // Default to patch if we can't parse
	}

	if latestMajor > currMajor {
		return MajorChange
	}
	if latestMinor > currMinor {
		return MinorChange
	}
	if latestPatch > currPatch {
		return PatchChange
	}

	return PatchChange
}

// getChangeColor returns the appropriate color for the version change type
func getChangeColor(change SemverChange) string {
	switch change {
	case MajorChange:
		return ColorRed
	case MinorChange:
		return ColorYellow
	case PatchChange:
		return ColorGreen
	default:
		return ColorReset
	}
}

// getOriginalVersionString retrieves the original version string with prefixes from the file
func getOriginalVersionString(packageName, filePath, fileType string) string {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}

	content := string(data)

	// Escape special regex characters in package name
	escapedName := regexp.QuoteMeta(packageName)

	// Pattern to match the dependency line: "package-name": "version"
	pattern := fmt.Sprintf(`"%s"\s*:\s*"([^"]*)"`, escapedName)
	re := regexp.MustCompile(pattern)

	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}

// printProgressBar prints a progress bar
func printProgressBar(current, total int) {
	const barWidth = 20
	progress := float64(current) / float64(total)
	filled := int(progress * barWidth)

	bar := "["
	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar += "="
		} else {
			bar += " "
		}
	}
	bar += "]"

	fmt.Printf("\r%s %d/%d %d%%", bar, current, total, int(progress*100))
	if current == total {
		fmt.Println() // New line when complete
	}
}
