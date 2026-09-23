package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/MilosRandelovic/bump-core/v2/parser"
	"github.com/MilosRandelovic/bump-core/v2/shared"
	"github.com/MilosRandelovic/bump-core/v2/updater"
	"github.com/MilosRandelovic/homebrew-bump/internal/output"
	"github.com/spf13/pflag"
)

var version = shared.Version
var checkOutdated = updater.CheckOutdated
var updateDependencies = updater.UpdateDependencies

type commandOptions struct {
	Update                  bool
	Verbose                 bool
	Semver                  bool
	MinimumAge              bool
	NoCache                 bool
	IncludePeerDependencies bool
	Monorepo                bool
	ShowVersion             bool
	Help                    bool
}

func (options commandOptions) dependencyOptions() shared.Options {
	return shared.Options{
		Semver:                   options.Semver,
		NoCache:                  options.NoCache,
		IncludePeerDependencies:  options.IncludePeerDependencies,
		Monorepo:                 options.Monorepo,
		EnforceMinimumReleaseAge: options.MinimumAge,
	}
}

func (options commandOptions) outputConfig() output.Config {
	return output.Config{
		Verbose:    options.Verbose,
		Semver:     options.Semver,
		MinimumAge: options.MinimumAge,
	}
}

func main() {
	var options commandOptions
	pflag.BoolVarP(&options.Update, "update", "u", false, "Update dependencies to latest versions")
	pflag.BoolVarP(&options.Verbose, "verbose", "v", false, "Enable verbose output")
	pflag.BoolVarP(&options.Semver, "semver", "s", false, "Respect semver constraints (^, ~) and skip hardcoded versions")
	pflag.BoolVarP(&options.MinimumAge, "minimum-age", "a", false, "Only suggest versions published more than 24 hours ago")
	pflag.BoolVarP(&options.NoCache, "no-cache", "C", false, "Disable caching of registry lookups")
	pflag.BoolVarP(&options.IncludePeerDependencies, "include-peers", "P", false, "Include peer dependencies when updating")
	pflag.BoolVarP(&options.Monorepo, "monorepo", "m", false, "Parse workspace packages in monorepo")
	pflag.BoolVarP(&options.ShowVersion, "version", "V", false, "Show version information")
	pflag.BoolVarP(&options.Help, "help", "h", false, "Show help information")
	pflag.Parse()

	if pflag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "Error: Unknown arguments: %v\nRun 'bump --help' for usage information.\n", pflag.Args())
		os.Exit(1)
	}

	if options.ShowVersion {
		fmt.Printf("bump version %s\n", version)
		os.Exit(0)
	}

	if options.Help {
		output.PrintHelp(version)
		os.Exit(0)
	}

	dependencyOptions := options.dependencyOptions()
	outputConfig := options.outputConfig()

	var log shared.LogFunc
	if options.Verbose {
		log = func(format string, args ...any) {
			fmt.Printf(format, args...)
		}
	}

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

	if err := updater.ValidateOptions(registryType, dependencyOptions); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	dependencies, err := parser.ParseDependencies(filePath, registryType, dependencyOptions)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing file: %v\n", err)
		os.Exit(1)
	}

	output.VerbosePrintf(outputConfig, "Found %d dependencies\n", len(dependencies))

	var progressCallback shared.ProgressFunc
	if !options.Verbose {
		progressCallback = output.PrintProgressBar
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	result, err := checkOutdated(ctx, dependencies, registryType, dependencyOptions, workingDirectory, progressCallback, log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking for updates: %v\n", err)
		os.Exit(1)
	}

	if len(result.Outdated) == 0 && len(result.Errors) == 0 && (!options.Semver || len(result.SemverSkipped) == 0) {
		fmt.Println("\nAll dependencies are up to date!")
		return
	}

	output.PrintOutdatedDependencies(result.Outdated)

	if options.Semver {
		output.PrintSemverSkipped(result.SemverSkipped, outputConfig)
	}

	output.PrintErrors(result.Errors, outputConfig)

	if options.Update {
		if len(result.Outdated) > 0 {
			err := updateDependencies(ctx, filePath, result.Outdated, registryType, dependencyOptions, workingDirectory, log)
			if err != nil {
				fmt.Fprintf(os.Stderr, "\nError updating dependencies: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("\nDependencies updated successfully!")
		} else {
			fmt.Println("\nNo dependencies to update.")
		}
	} else {
		output.PrintUpdatePrompt(len(result.Outdated) > 0, outputConfig)
	}
}
