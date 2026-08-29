# Copilot Rules for Homebrew Bump

## Project Context

This is a thin CLI wrapper around [bump-core](../bump-core) — a Go library that manages dependency updates for npm (package.json) and Dart/Flutter pub (pubspec.yaml) projects. All core logic (parsing, registry communication, version checking, file updating) lives in bump-core. This repo provides only the CLI interface: flag parsing, terminal output formatting, progress bars, and colored output.

The repo has a `homebrew` prefix as the tool is available as a Homebrew tap (referenced in the `Formula` folder), but should always be built as `bump`.

## Architecture

- **bump-core** (`github.com/MilosRandelovic/bump-core/v2`): All core logic — types, parsers, registry clients, updater, shared utilities. `go.mod` pins a published release; use a local `go.work` file when developing both repositories together.
- **homebrew-bump** (this repo): CLI entry point (`main.go`) and terminal output formatting (`internal/output/`). No business logic here.
- **bump-mcp** (`github.com/MilosRandelovic/bump-core/v2/cmd/bump-mcp`): MCP server owned by bump-core and installed by this repository's Homebrew formula.

### Key Integration Points

- `parser.AutoDetectDependencyFile(directory, logFunc)` — takes a directory path and a `shared.LogFunc` callback (or nil)
- `parser.ParseDependencies(filePath, registryType, options)` — parses dependencies from a file
- `updater.CheckOutdated(ctx, dependencies, registryType, options, workingDirectory, progressFunc, logFunc)` — checks for outdated dependencies
- `updater.UpdateDependencies(ctx, filePath, outdated, registryType, options, workingDirectory, logFunc)` — updates dependency files
- The CLI creates a `shared.LogFunc` that wraps `fmt.Printf` when verbose mode is enabled, or passes nil otherwise

## Code Patterns to Follow

### Options Pattern

- ALL functions that accept configuration flags MUST use the `shared.Options` struct from bump-core
- Do NOT pass individual boolean parameters (verbose, semver, monorepo, etc.)

### LogFunc Pattern

- bump-core functions accept `log shared.LogFunc` callbacks for verbose output
- The CLI creates one when `--verbose` is set: `func(format string, args ...any) { fmt.Printf(format, args...) }`
- Pass nil when verbose is off — bump-core handles nil checks internally
- `--minimum-age` / `-a` sets `shared.Options.EnforceMinimumReleaseAge`; the fixed policy only suggests releases published more than 24 hours ago and never downgrades the current version

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
- Use concise, idiomatic Go names. Keep conventional or immediately obvious abbreviations such as `ctx`, `err`, `ok`, `i`, `t`, `max`, `min`, `args`, `config`, and `info`; expand abbreviations whose meaning is unclear from their scope and domain.
- Do not add comments merely to narrate the code or document what changed. Keep comments for public API contracts and non-obvious intent.

## Testing Patterns

- Run `go test ./...` for CLI orchestration and output changes.
- Run `make smoke` to build the CLI and verify version output, help output, and missing dependency-file handling.
- Keep registry and dependency-update tests in `bump-core`; this repository should test CLI orchestration and terminal output only.
- Formula changes must install successfully from source and pass `brew test bump` on macOS.
- The formula must install both `bump` and `bump-mcp`; build the MCP command from its explicit bump-core module version rather than duplicating its implementation here.
- MCP version policies and package, dependency-type, and file targeting belong in bump-core and must remain identical to the sidecar protocol.
- Formula update pull requests must run CI; do not exempt their branches from validation.
- Validate workflow changes with Actionlint.

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
