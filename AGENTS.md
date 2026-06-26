# Copilot Rules for Homebrew Bump

## Project Context

This is a thin CLI wrapper around [bump-core](../bump-core) — a Go library that manages dependency updates for npm (package.json) and Dart/Flutter pub (pubspec.yaml) projects. All core logic (parsing, registry communication, version checking, file updating) lives in bump-core. This repo provides only the CLI interface: flag parsing, terminal output formatting, progress bars, and colored output.

The repo has a `homebrew` prefix as the tool is available as a Homebrew tap (referenced in the `Formula` folder), but should always be built as `bump`.

## Architecture

- **bump-core** (`github.com/MilosRandelovic/bump-core`): All core logic — types, parsers, registry clients, updater, shared utilities. Referenced via `go.mod` replace directive pointing to `../bump-core`.
- **homebrew-bump** (this repo): CLI entry point (`main.go`) and terminal output formatting (`internal/output/`). No business logic here.

### Key Integration Points

- `parser.AutoDetectDependencyFile(directory, logFunc)` — takes a directory path and a `shared.LogFunc` callback (or nil)
- `parser.ParseDependencies(filePath, registryType, options)` — parses dependencies from a file
- `updater.CheckOutdated(deps, registryType, options, workingDirectory, progressCallback, logFunc)` — checks for outdated deps
- `updater.UpdateDependencies(filePath, outdated, registryType, options, workingDirectory, logFunc)` — updates dependency files
- The CLI creates a `shared.LogFunc` that wraps `fmt.Printf` when verbose mode is enabled, or passes nil otherwise

## Code Patterns to Follow

### Options Pattern

- ALL functions that accept configuration flags MUST use the `shared.Options` struct from bump-core
- Do NOT pass individual boolean parameters (verbose, semver, monorepo, etc.)

### LogFunc Pattern

- bump-core functions accept `log shared.LogFunc` callbacks for verbose output
- The CLI creates one when `--verbose` is set: `func(format string, args ...any) { fmt.Printf(format, args...) }`
- Pass nil when verbose is off — bump-core handles nil checks internally

### Output Formatting (this repo's responsibility)

- Sort ALL output lists (outdated, semverSkipped, errors) alphabetically by name within the print methods, not in main
- Group dependencies by file first, then by type (dependencies, devDependencies, peerDependencies)
- Only display file names when multiple files have outdated dependencies
- Use relative paths for file display
- Use semantic colors: red=major, yellow=minor, green=patch changes
- Show progress bars per file in non-verbose mode
- Provide detailed information in verbose mode

### File Updates

- Only update version fields, preserve all other original JSON/YAML content and formatting, including hosted references
- Keep version prefixes (^, ~, >=) when updating
- Internally store both clean and original versions for proper updates
- In monorepo mode, group updates by FilePath and update each file separately

### Semver Constraint Handling

- Support ^, ~, >=, >, <, <=, and compound constraints (>=1.0.0 <2.0.0)
- Filter out pre-release versions (alpha/beta/rc) unless explicitly requested
- Use shared.CleanVersion() to strip prefixes before comparisons

## Code Patterns to Avoid

- Don't make redundant API calls - use GetBothLatestVersions instead of separate calls
- Don't mix error types - constraint mismatches are semverSkipped, not errors
- Don't hardcode registry URLs - parse from configuration files
- Don't abbreviate function and variable names - use full descriptive names instead
- Don't add comments for the sake of documenting what you just changed

## Testing Patterns

- Always use MockRegistryClient for testing update logic
- Test with realistic package.json and pubspec.yaml content
- Include scoped packages (@company/package) and hosted packages in tests
- Verify both success and error paths

## File Structure Understanding

```text
main.go               # CLI entry point: flag parsing, orchestration
internal/
└── output/           # Terminal output formatting, progress bars, help text, colored output
```

All core logic lives in bump-core:

```text
bump-core/
├── shared/           # Common types, version utilities, interfaces
├── parser/           # Auto-detection and delegation
├── updater/          # Core update checking logic
├── npm/              # npm ecosystem (package.json, .npmrc, npm registry)
└── pub/              # Dart/Flutter pub ecosystem (pubspec.yaml, pub-tokens.json, pub registry)
```

When making changes, always consider the impact on both npm and pub ecosystems and ensure consistent behavior across both.
