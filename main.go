package main

import (
	"context"
	"fmt"
	"os"

	"github.com/MilosRandelovic/bump-core/parser"
	"github.com/MilosRandelovic/bump-core/shared"
	"github.com/MilosRandelovic/bump-core/updater"
	"github.com/MilosRandelovic/homebrew-bump/internal/output"
	"github.com/spf13/pflag"
)

var version = shared.Version

func main() {
	var (
		update                  = pflag.BoolP("update", "u", false, "Update dependencies to latest versions")
		verbose                 = pflag.BoolP("verbose", "v", false, "Enable verbose output")
		semver                  = pflag.BoolP("semver", "s", false, "Respect semver constraints (^, ~) and skip hardcoded versions")
		noCache                 = pflag.BoolP("no-cache", "C", false, "Disable caching of registry lookups")
		includePeerDependencies = pflag.BoolP("include-peers", "P", false, "Include peer dependencies when updating")
		monorepo                = pflag.BoolP("monorepo", "m", false, "Parse workspace packages in monorepo")
		showVersion             = pflag.BoolP("version", "V", false, "Show version information")
		help                    = pflag.BoolP("help", "h", false, "Show help information")
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

	// Create options struct from flags
	options := shared.Options{
		Verbose:                 *verbose,
		Update:                  *update,
		Semver:                  *semver,
		NoCache:                 *noCache,
		IncludePeerDependencies: *includePeerDependencies,
		Monorepo:                *monorepo,
	}

	// Verbose log function for bump-core callbacks
	var log shared.LogFunc
	if options.Verbose {
		log = func(format string, args ...any) {
			fmt.Printf(format, args...)
		}
	}

	// Auto-detect dependency file in current directory
	workingDirectory, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	filePath, registryType, err := parser.AutoDetectDependencyFile(workingDirectory, log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := updater.ValidateOptions(registryType, options); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Parse the file
	dependencies, err := parser.ParseDependencies(filePath, registryType, options)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing file: %v\n", err)
		os.Exit(1)
	}

	output.VerbosePrintf(options, "Found %d dependencies\n", len(dependencies))

	var progressCallback func(current, total int)
	if !options.Verbose {
		progressCallback = output.PrintProgressBar
	}

	// Check for outdated dependencies
	result, err := updater.CheckOutdated(context.Background(), dependencies, registryType, options, workingDirectory, progressCallback, log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking for updates: %v\n", err)
		os.Exit(1)
	}

	if len(result.Outdated) == 0 && len(result.Errors) == 0 && (!options.Semver || len(result.SemverSkipped) == 0) {
		fmt.Println("\nAll dependencies are up to date!")
		return
	}

	// Display results
	output.PrintOutdatedDependencies(result.Outdated, options)

	// Display semver skipped summary if in semver mode and there were skipped packages
	if options.Semver {
		output.PrintSemverSkipped(result.SemverSkipped, options)
	}

	// Display error summary if there were errors
	output.PrintErrors(result.Errors, options)

	// Update if requested
	if options.Update {
		if len(result.Outdated) > 0 {
			err := updater.UpdateDependencies(filePath, result.Outdated, registryType, options, workingDirectory, log)
			if err != nil {
				fmt.Fprintf(os.Stderr, "\nError updating dependencies: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("\nDependencies updated successfully!")
		} else {
			fmt.Println("\nNo dependencies to update.")
		}
	} else {
		output.PrintUpdatePrompt(len(result.Outdated) > 0, options.Semver)
	}
}
