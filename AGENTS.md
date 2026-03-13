# Copilot Rules for Homebrew Bump

## Project Context

This is a Go CLI tool called "bump" that manages dependency updates for npm (package.json) and Dart/Flutter pub (pubspec.yaml) projects. It supports semver constraints, private registries, hosted packages, monorepo workspaces, and provides both checking and updating capabilities.

The repo has a `homebrew` prefix as the tool is available as a Homebrew tap (referenced in the `Formula` folder), but should always be built as `bump`.

### Monorepo Support

- npm monorepo support via `--monorepo` flag
- Detects workspaces from root package.json `workspaces` field
- Supports glob patterns for workspace detection (e.g., `packages/*`, `apps/*`)
- Each dependency tracks its FilePath for proper file-by-file updates
- Output groups dependencies by file when multiple files contain outdated packages

## Architecture Principles

- Keep npm/ and pub/ packages separate - no cross-dependencies
- Place common functionality in internal/shared/
- Follow single responsibility principle per package
- Use shared.Options struct for passing configuration flags instead of individual boolean parameters

## Code Patterns to Follow

### Options Pattern

- ALL functions that accept configuration flags MUST use the shared.Options struct
- Do NOT pass individual boolean parameters (verbose, semver, monorepo, etc.)
- Options struct contains: Verbose, Update, Semver, NoCache, IncludePeerDependencies, Monorepo
- Example: `func ParseDependencies(filePath string, options shared.Options)` instead of `func ParseDependencies(filePath string, includePeerDependencies bool, monorepo bool)`
- Access options with `options.Verbose`, `options.Semver`, etc.
- In tests, initialize with `shared.Options{}` for defaults or `shared.Options{Verbose: true, Semver: true}` for specific flags

### Error Handling

- Categorize constraint mismatches as `semverSkipped`, not `errors`
- When `GetBothLatestVersions` returns "no versions satisfy the constraint" error, add to semverSkipped with the absoluteLatest version
- Wrap errors with context: `fmt.Errorf("context: %w", err)`
- Distinguish between network errors, parse errors, and constraint mismatches

### Registry Client Implementation

- Always implement the shared.RegistryClient interface
- Pass registryURL parameter through the entire chain for hosted package support
- Use GetBothLatestVersions to get both absolute latest and constraint-satisfying versions in one call
- Support authentication for hosted packages via .npmrc and pub-tokens.json parsing

### Testing Standards

- Use t.TempDir() for temporary test files
- Create MockRegistryClient implementing shared.RegistryClient interface
- Include real-world test data (scoped packages, hosted packages, complex constraints)
- Add regression tests for bug fixes
- Test edge cases: invalid versions, network errors, authentication failures
- Test monorepo scenarios: workspace detection, glob patterns, FilePath assignment, multiple package.json files

### Output Formatting

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
internal/
├── shared/           # Common types, version utilities, interfaces
├── parser/           # Auto-detection and delegation
├── updater/          # Core update checking logic
├── output/           # Terminal output formatting, progress bars, help text
├── npm/              # npm ecosystem (package.json, .npmrc, npm registry)
└── pub/              # Dart/Flutter pub ecosystem (pubspec.yaml, pub-tokens.json, pub registry)
```

When making changes, always consider the impact on both npm and pub ecosystems and ensure consistent behavior across both.
