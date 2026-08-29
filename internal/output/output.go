package output

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/MilosRandelovic/bump-core/v2/shared"
)

// Color constants for terminal output
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m" // Major version changes
	colorYellow = "\033[33m" // Minor version changes
	colorGreen  = "\033[32m" // Patch version changes
	colorCyan   = "\033[36m" // Package names
)

// getChangeColor returns the appropriate color for the version change type
func getChangeColor(change shared.SemverChange) string {
	switch change {
	case shared.MajorChange:
		return colorRed
	case shared.MinorChange:
		return colorYellow
	case shared.PatchChange:
		return colorGreen
	default:
		return colorReset
	}
}

var workingDir, _ = os.Getwd()

// getDisplayPath converts an absolute path to a relative path for display
func getDisplayPath(filePath string) string {
	if relativePath, err := filepath.Rel(workingDir, filePath); err == nil {
		return relativePath
	}
	return filePath
}

// PrintProgressBar writes the current file's progress bar to standard error and ends the line when that file completes.
func PrintProgressBar(progressUpdate shared.Progress) {
	const barWidth = 20
	current := progressUpdate.FileCurrent
	total := progressUpdate.FileTotal
	if total == 0 {
		return
	}
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

	fmt.Fprintf(os.Stderr, "\r%s %s %d/%d %d%%", getDisplayPath(progressUpdate.FilePath), bar, current, total, int(progress*100))
	if current == total {
		fmt.Fprintln(os.Stderr) // New line when complete
	}
}

// PrintOutdatedDependencies writes color-coded updates to standard output, grouped by file and dependency type and sorted by package name.
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

	shared.SortFilesByDepth(files)
	showFilenames := len(files) > 1

	for _, file := range files {
		types := grouped[file]
		if showFilenames {
			fmt.Printf("\n%s:\n", getDisplayPath(file))
		}

		for _, dependencyType := range []shared.DependencyType{shared.Dependencies, shared.DevDependencies, shared.PeerDependencies} {
			dependencies := types[dependencyType]
			if len(dependencies) > 0 {
				if showFilenames {
					fmt.Printf("  %s:\n", dependencyType.String())
				} else {
					fmt.Printf("\n%s:\n", dependencyType.String())
				}
				printDependencyList(dependencies, showFilenames)
			}
		}
	}
}

func printDependencyList(outdated []shared.OutdatedDependency, indented bool) {

	// Sort alphabetically by name
	slices.SortFunc(outdated, func(first, second shared.OutdatedDependency) int {
		return strings.Compare(first.Name, second.Name)
	})

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
		color := getChangeColor(change)

		// Use the original version from the dependency struct
		currentVersion := dependency.OriginalVersion
		prefix := shared.GetVersionPrefix(currentVersion)
		latestVersion := prefix + dependency.LatestVersion

		// Apply color to output for better visibility
		fmt.Printf("%s%s%-*s%s  %*s  →  %s%s%s\n",
			indent,
			colorCyan, maxNameWidth, dependency.Name, colorReset,
			maxCurrentVersionWidth, currentVersion,
			color, latestVersion, colorReset)
	}
}

// PrintSemverSkipped writes skipped packages to standard output.
// Verbose mode prints sorted package details; otherwise it prints only a count and rerun hint.
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

		shared.SortFilesByDepth(files)
		showFilenames := len(files) > 1

		fmt.Printf("\nPackages skipped due to semver constraints:\n")
		for _, file := range files {
			if showFilenames {
				fmt.Printf("\n%s:\n", getDisplayPath(file))
			}

			// Display by dependency type in the same order as outdated
			for _, dependencyType := range []shared.DependencyType{shared.Dependencies, shared.DevDependencies, shared.PeerDependencies} {
				skippedByType := grouped[file][dependencyType]
				if len(skippedByType) == 0 {
					continue
				}

				if showFilenames {
					fmt.Printf("  %s:\n", dependencyType.String())
				} else {
					fmt.Printf("\n%s:\n", dependencyType.String())
				}

				// Sort packages alphabetically within each type
				names := make([]string, 0, len(skippedByType))
				for name := range skippedByType {
					names = append(names, name)
				}
				slices.Sort(names)

				indent := "    "
				if !showFilenames {
					indent = "  "
				}

				for _, name := range names {
					skipped := skippedByType[name]
					if skipped.LatestVersion != "" {
						fmt.Printf("%s%s%s%s: %s → %s (%s)\n", indent, colorCyan, skipped.Name, colorReset, skipped.OriginalVersion, skipped.LatestVersion, skipped.Reason)
					} else {
						fmt.Printf("%s%s%s%s: %s (%s)\n", indent, colorCyan, skipped.Name, colorReset, skipped.OriginalVersion, skipped.Reason)
					}
				}
			}
		}
	} else {
		fmt.Printf("\n%d packages were skipped due to updates not meeting semver constraints. Run 'bump --semver --verbose' to see the full output.\n", len(semverSkipped))
	}
}

// PrintErrors writes dependency-check failures to standard output.
// Verbose mode prints sorted error details; otherwise it prints only a count and rerun hint.
func PrintErrors(errors []shared.DependencyError, options shared.Options) {
	if len(errors) == 0 {
		return
	}

	// Sort alphabetically by name
	slices.SortFunc(errors, func(first, second shared.DependencyError) int {
		return strings.Compare(first.Name, second.Name)
	})

	if options.Verbose {
		fmt.Printf("\nErrors encountered:\n")
		for _, dependencyError := range errors {
			fmt.Printf("  %s%s%s: %s\n", colorCyan, dependencyError.Name, colorReset, dependencyError.Error)
		}
	} else {
		if options.Semver {
			fmt.Printf("\n%d packages could not be checked due to errors. Run 'bump --semver --verbose' to see the full output.\n", len(errors))
		} else {
			fmt.Printf("\n%d packages could not be checked due to errors. Run 'bump --verbose' to see the full output.\n", len(errors))
		}
	}
}

// PrintUpdatePrompt writes the update command required to preserve the active semantic-version and minimum-age options when updates exist.
func PrintUpdatePrompt(hasOutdated bool, options shared.Options) {
	if !hasOutdated {
		return
	}

	args := []string{"--update"}
	if options.Semver {
		args = append(args, "--semver")
	}
	if options.EnforceMinimumReleaseAge {
		args = append(args, "--minimum-age")
	}
	fmt.Printf("\nRun 'bump %s' to apply these dependency updates.\n", strings.Join(args, " "))
}

// VerbosePrintf writes formatted text to standard output only when verbose mode is enabled.
func VerbosePrintf(options shared.Options, format string, args ...any) {
	if options.Verbose {
		fmt.Printf(format, args...)
	}
}

// PrintHelp writes CLI usage and the supplied version to standard output.
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
	fmt.Println("  --minimum-age, -a    Only suggest versions published more than 24 hours ago")
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
	fmt.Println("  bump -ua           # Update using versions more than 24 hours old")
	fmt.Println("  bump -uP           # Update including peer dependencies")
}
