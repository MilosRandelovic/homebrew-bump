package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/MilosRandelovic/homebrew-bump/internal/output"
	"github.com/MilosRandelovic/homebrew-bump/internal/parser"
	"github.com/MilosRandelovic/homebrew-bump/internal/updater"
	"github.com/spf13/pflag"
)

const version = "1.2.0"

func main() {
	var (
		update                  = pflag.BoolP("update", "u", false, "Update dependencies to latest versions")
		verbose                 = pflag.BoolP("verbose", "v", false, "Enable verbose output")
		showVersion             = pflag.BoolP("version", "V", false, "Show version information")
		help                    = pflag.BoolP("help", "h", false, "Show help information")
		semver                  = pflag.BoolP("semver", "s", false, "Respect semver constraints (^, ~) and skip hardcoded versions")
		includePeerDependencies = pflag.BoolP("include-peers", "P", false, "Include peer dependencies when updating")
		noCache                 = pflag.BoolP("no-cache", "C", false, "Disable caching of registry lookups")
	)
	pflag.Parse()

	// Check for any remaining arguments that weren't parsed as flags
	if pflag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "Error: Unknown arguments: %v\nRun 'bump --help' for usage information.\n", pflag.Args())
		os.Exit(1)
	}

	if *showVersion {
		fmt.Printf("bump version %s\n", version)
		os.Exit(0)
	}

	if *help {
		output.PrintHelp(version)
		os.Exit(0)
	}

	// Auto-detect dependency file in current directory
	filePath, fileType, err := parser.AutoDetectDependencyFile()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if *verbose {
		fmt.Printf("Found %s file: %s\n", fileType, filePath)
	}

	// Parse the file
	dependencies, err := parser.ParseDependencies(filePath, fileType, *includePeerDependencies)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing file: %v\n", err)
		os.Exit(1)
	}

	if *verbose {
		fmt.Printf("Found %d dependencies\n", len(dependencies))
	}

	// Always check for outdated dependencies (and extract filename from path for display)
	fmt.Printf("Checking %d dependencies from %s...\n", len(dependencies), filepath.Base(filePath))

	var progressCallback func(current, total int)
	if !*verbose {
		progressCallback = output.PrintProgressBar
	}

	result, err := updater.CheckOutdatedWithProgress(dependencies, fileType, *verbose, *semver, *noCache, progressCallback)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking for updates: %v\n", err)
		os.Exit(1)
	}

	outdated := result.Outdated
	errors := result.Errors
	semverSkipped := result.SemverSkipped

	// Sort outdated dependencies alphabetically by name
	sort.Slice(outdated, func(i, j int) bool {
		return outdated[i].Name < outdated[j].Name
	})

	// Sort errors alphabetically by name
	sort.Slice(errors, func(i, j int) bool {
		return errors[i].Name < errors[j].Name
	})

	// Sort semver skipped packages alphabetically by name
	sort.Slice(semverSkipped, func(i, j int) bool {
		return semverSkipped[i].Name < semverSkipped[j].Name
	})

	if len(outdated) == 0 && len(errors) == 0 && (!*semver || len(semverSkipped) == 0) {
		fmt.Println("\nAll dependencies are up to date!")
		return
	}

	if !*verbose {
		fmt.Printf("\n") // Add new line after progress bar only in non-verbose mode
	}

	// Display results
	output.PrintOutdatedDependencies(outdated, *verbose)

	// Display semver skipped summary if in semver mode and there were skipped packages
	if *semver {
		output.PrintSemverSkipped(semverSkipped, *verbose)
	}

	// Display error summary if there were errors
	output.PrintErrors(errors, *verbose, *semver)

	// Update if requested
	if *update {
		if len(outdated) > 0 {
			err := updater.UpdateDependencies(filePath, outdated, fileType, *verbose, *semver, *includePeerDependencies)
			if err != nil {
				fmt.Fprintf(os.Stderr, "\nError updating dependencies: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("\nDependencies updated successfully!")
		} else {
			fmt.Println("\nNo dependencies to update.")
		}
	} else {
		output.PrintUpdatePrompt(len(outdated) > 0, *semver)
	}
}
