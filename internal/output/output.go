package output

import (
	"fmt"

	"github.com/MilosRandelovic/homebrew-bump/internal/shared"
)

// Color constants for terminal output
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m" // Major version changes
	ColorYellow = "\033[33m" // Minor version changes
	ColorGreen  = "\033[32m" // Patch version changes
	ColorCyan   = "\033[36m" // Package names
)

// GetChangeColor returns the appropriate color for the version change type
func GetChangeColor(change shared.SemverChange) string {
	switch change {
	case shared.MajorChange:
		return ColorRed
	case shared.MinorChange:
		return ColorYellow
	case shared.PatchChange:
		return ColorGreen
	default:
		return ColorReset
	}
}

// PrintProgressBar prints a progress bar to stdout
func PrintProgressBar(current, total int) {
	const barWidth = 20
	progress := float64(current) / float64(total)
	filled := int(progress * barWidth)

	bar := "["
	for i := range barWidth {
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

// PrintOutdatedDependencies displays the list of outdated dependencies with color-coded changes
func PrintOutdatedDependencies(outdated []shared.OutdatedDependency, verbose bool) {
	if verbose && len(outdated) > 0 {
		fmt.Printf("\nFound %d outdated dependencies:\n", len(outdated))
	}

	if len(outdated) == 0 {
		return
	}

	// Calculate maximum widths for proper alignment
	maxNameWidth := 0
	maxCurrentVersionWidth := 0
	for _, dependency := range outdated {
		if len(dependency.Name) > maxNameWidth {
			maxNameWidth = len(dependency.Name)
		}
		if len(dependency.OriginalVersion) > maxCurrentVersionWidth {
			maxCurrentVersionWidth = len(dependency.OriginalVersion)
		}
	}

	// Add some padding
	maxNameWidth += 2
	maxCurrentVersionWidth += 2

	for _, dependency := range outdated {
		change := shared.GetSemverChange(dependency.CurrentVersion, dependency.LatestVersion)
		color := GetChangeColor(change)

		// Use the original version from the dependency struct
		currentVersion := dependency.OriginalVersion
		// Replace only the version number, preserving any prefix
		latestVersion := currentVersion
		if dependency.CurrentVersion != "" {
			latestVersion = currentVersion[:len(currentVersion)-len(dependency.CurrentVersion)] + dependency.LatestVersion
		}

		// Apply color to output for better visibility
		fmt.Printf("%s%-*s%s  %*s  →  %s%s%s\n",
			ColorCyan, maxNameWidth, dependency.Name, ColorReset,
			maxCurrentVersionWidth, currentVersion,
			color, latestVersion, ColorReset)
	}
}

// PrintSemverSkipped displays packages that were skipped due to semver constraints
func PrintSemverSkipped(semverSkipped []shared.SemverSkipped, verbose bool) {
	if len(semverSkipped) == 0 {
		return
	}

	if verbose {
		fmt.Printf("\nPackages skipped due to semver constraints:\n")
		for _, skipped := range semverSkipped {
			if skipped.LatestVersion != "" {
				fmt.Printf("  %s%s%s: %s → %s (%s)\n", ColorCyan, skipped.Name, ColorReset, skipped.OriginalVersion, skipped.LatestVersion, skipped.Reason)
			} else {
				fmt.Printf("  %s%s%s: %s (%s)\n", ColorCyan, skipped.Name, ColorReset, skipped.OriginalVersion, skipped.Reason)
			}
		}
	} else {
		fmt.Printf("\n%d packages were skipped due to updates not meeting semver constraints. Run 'bump --semver --verbose' to see the full output.\n", len(semverSkipped))
	}
}

// PrintErrors displays errors encountered during dependency checking
func PrintErrors(errors []shared.DependencyError, verbose, semver bool) {
	if len(errors) == 0 {
		return
	}

	if verbose {
		fmt.Printf("\nErrors encountered:\n")
		for _, dependencyError := range errors {
			fmt.Printf("  %s%s%s: %s\n", ColorCyan, dependencyError.Name, ColorReset, dependencyError.Error)
		}
	} else {
		if semver {
			fmt.Printf("\n%d packages could not be checked due to errors. Run 'bump --semver --verbose' to see the full output.\n", len(errors))
		} else {
			fmt.Printf("\n%d packages could not be checked due to errors. Run 'bump --verbose' to see the full output.\n", len(errors))
		}
	}
}

// PrintUpdatePrompt displays a message prompting the user to run the update command
func PrintUpdatePrompt(hasOutdated, semver bool) {
	if !hasOutdated {
		return
	}

	if semver {
		fmt.Printf("\nRun 'bump --update --semver' to update dependencies while respecting semver constraints.\n")
	} else {
		fmt.Printf("\nRun 'bump --update' to update dependencies to latest versions.\n")
	}
}

// PrintHelp displays the help message for the bump CLI tool
func PrintHelp(version string) {
	fmt.Printf("bump v%s - A utility to check and update dependencies\n\n", version)
	fmt.Println("Usage: bump [options]")
	fmt.Println("\nAuto-detects package.json or pubspec.yaml in current directory")
	fmt.Println("Automatically checks for outdated dependencies")
	fmt.Println("\nSupported files:")
	fmt.Println("  package.json  - npm dependencies")
	fmt.Println("  pubspec.yaml  - Dart/Flutter dependencies")
	fmt.Println("\nOptions:")
	fmt.Println("  --update, -u         Update dependencies to latest versions")
	fmt.Println("  --semver, -s         Respect semver constraints (^, ~) and skip hardcoded versions")
	fmt.Println("  --include-peers, -P  Include peer dependencies when updating")
	fmt.Println("  --verbose, -v        Enable verbose output")
	fmt.Println("  --no-cache, -C       Disable caching of registry lookups")
	fmt.Println("  --version, -V        Show version information")
	fmt.Println("  --help, -h           Show this help")
	fmt.Println("\nShorthand flags can be merged: -us is equivalent to -u -s")
	fmt.Println("\nExamples:")
	fmt.Println("  bump               # Check for outdated dependencies")
	fmt.Println("  bump --update      # Update dependencies to latest versions")
	fmt.Println("  bump -u            # Same as above (shorthand)")
	fmt.Println("  bump -uv           # Update with verbose output (merged shorthands)")
	fmt.Println("  bump -s            # Check with semver constraints")
	fmt.Println("  bump -us           # Update with semver constraints (merged)")
	fmt.Println("  bump -uP           # Update including peer dependencies")
}
