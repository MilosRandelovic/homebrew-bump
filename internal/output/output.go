package output

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

// sortFilesByDepth sorts files by path depth (shortest first = root), then alphabetically
func sortFilesByDepth(files []string) {
	for i := 0; i < len(files)-1; i++ {
		for j := i + 1; j < len(files); j++ {
			depthI := strings.Count(files[i], string(filepath.Separator))
			depthJ := strings.Count(files[j], string(filepath.Separator))

			if depthI > depthJ {
				files[i], files[j] = files[j], files[i]
			} else if depthI == depthJ && files[i] > files[j] {
				files[i], files[j] = files[j], files[i]
			}
		}
	}
}

// sortStringsAlphabetically sorts a string slice alphabetically using bubble sort
func sortStringsAlphabetically(items []string) {
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[i] > items[j] {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

// getDisplayPath converts an absolute path to a relative path for display
func getDisplayPath(filePath string) string {
	cwd, _ := os.Getwd()
	if relPath, err := filepath.Rel(cwd, filePath); err == nil {
		return relPath
	}
	return filePath
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
func PrintOutdatedDependencies(outdated []shared.OutdatedDependency, options shared.Options) {
	if len(outdated) == 0 {
		return
	}

	grouped := make(map[string]map[shared.DependencyType][]shared.OutdatedDependency)
	files := []string{}
	for _, dependency := range outdated {
		if grouped[dependency.FilePath] == nil {
			grouped[dependency.FilePath] = make(map[shared.DependencyType][]shared.OutdatedDependency)
			files = append(files, dependency.FilePath)
		}
		grouped[dependency.FilePath][dependency.Type] = append(grouped[dependency.FilePath][dependency.Type], dependency)
	}

	sortFilesByDepth(files)
	showFilenames := len(files) > 1

	for _, file := range files {
		types := grouped[file]
		if showFilenames {
			fmt.Printf("\n%s:\n", getDisplayPath(file))
		}

		for _, depType := range []shared.DependencyType{shared.Dependencies, shared.DevDependencies, shared.PeerDependencies} {
			dependencies := types[depType]
			if len(dependencies) > 0 {
				if showFilenames {
					fmt.Printf("  %s:\n", depType.String())
				} else {
					fmt.Printf("\n%s:\n", depType.String())
				}
				printDependencyList(dependencies, showFilenames)
			}
		}
	}
}

func printDependencyList(outdated []shared.OutdatedDependency, indented bool) {
	// Sort alphabetically by name
	for i := 0; i < len(outdated)-1; i++ {
		for j := i + 1; j < len(outdated); j++ {
			if outdated[i].Name > outdated[j].Name {
				outdated[i], outdated[j] = outdated[j], outdated[i]
			}
		}
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

	indent := "    "
	if !indented {
		indent = "  "
	}

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
		fmt.Printf("%s%s%-*s%s  %*s  →  %s%s%s\n",
			indent,
			ColorCyan, maxNameWidth, dependency.Name, ColorReset,
			maxCurrentVersionWidth, currentVersion,
			color, latestVersion, ColorReset)
	}
}

// PrintSemverSkipped displays packages that were skipped due to semver constraints
func PrintSemverSkipped(semverSkipped []shared.SemverSkipped, options shared.Options) {
	if len(semverSkipped) == 0 {
		return
	}

	if options.Verbose {
		// Group by file and type, then deduplicate within each group
		grouped := make(map[string]map[shared.DependencyType]map[string]shared.SemverSkipped)
		files := []string{}
		for _, skip := range semverSkipped {
			if grouped[skip.FilePath] == nil {
				grouped[skip.FilePath] = make(map[shared.DependencyType]map[string]shared.SemverSkipped)
				files = append(files, skip.FilePath)
			}
			if grouped[skip.FilePath][skip.Type] == nil {
				grouped[skip.FilePath][skip.Type] = make(map[string]shared.SemverSkipped)
			}
			grouped[skip.FilePath][skip.Type][skip.Name] = skip
		}

		sortFilesByDepth(files)
		showFilenames := len(files) > 1

		fmt.Printf("\nPackages skipped due to semver constraints:\n")
		for _, file := range files {
			if showFilenames {
				fmt.Printf("\n%s:\n", getDisplayPath(file))
			}

			// Display by dependency type in the same order as outdated
			for _, depType := range []shared.DependencyType{shared.Dependencies, shared.DevDependencies, shared.PeerDependencies} {
				skippedByType := grouped[file][depType]
				if len(skippedByType) == 0 {
					continue
				}

				if showFilenames {
					fmt.Printf("  %s:\n", depType.String())
				} else {
					fmt.Printf("\n%s:\n", depType.String())
				}

				// Sort packages alphabetically within each type
				names := make([]string, 0, len(skippedByType))
				for name := range skippedByType {
					names = append(names, name)
				}
				sortStringsAlphabetically(names)

				indent := "    "
				if !showFilenames {
					indent = "  "
				}

				for _, name := range names {
					skipped := skippedByType[name]
					if skipped.LatestVersion != "" {
						fmt.Printf("%s%s%s%s: %s → %s (%s)\n", indent, ColorCyan, skipped.Name, ColorReset, skipped.OriginalVersion, skipped.LatestVersion, skipped.Reason)
					} else {
						fmt.Printf("%s%s%s%s: %s (%s)\n", indent, ColorCyan, skipped.Name, ColorReset, skipped.OriginalVersion, skipped.Reason)
					}
				}
			}
		}
	} else {
		fmt.Printf("\n%d packages were skipped due to updates not meeting semver constraints. Run 'bump --semver --verbose' to see the full output.\n", len(semverSkipped))
	}
}

// PrintErrors displays errors encountered during dependency checking
func PrintErrors(errors []shared.DependencyError, options shared.Options) {
	if len(errors) == 0 {
		return
	}

	// Sort alphabetically by name
	for i := 0; i < len(errors)-1; i++ {
		for j := i + 1; j < len(errors); j++ {
			if errors[i].Name > errors[j].Name {
				errors[i], errors[j] = errors[j], errors[i]
			}
		}
	}

	if options.Verbose {
		fmt.Printf("\nErrors encountered:\n")
		for _, dependencyError := range errors {
			fmt.Printf("  %s%s%s: %s\n", ColorCyan, dependencyError.Name, ColorReset, dependencyError.Error)
		}
	} else {
		if options.Semver {
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

// VerbosePrintf prints formatted output only if verbose mode is enabled
func VerbosePrintf(options shared.Options, format string, args ...any) {
	if options.Verbose {
		fmt.Printf(format, args...)
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
	fmt.Println("  --verbose, -v        Enable verbose output")
	fmt.Println("  --update, -u         Update dependencies to latest versions")
	fmt.Println("  --semver, -s         Respect semver constraints (^, ~) and skip hardcoded versions")
	fmt.Println("  --no-cache, -C       Disable caching of registry lookups")
	fmt.Println("  --include-peers, -P  Include peer dependencies when updating [npm only]")
	fmt.Println("  --monorepo, -m       Parse workspace packages in monorepo [npm only]")
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
