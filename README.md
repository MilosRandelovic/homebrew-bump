# Bump

A Go utility that parses `package.json` and `pubspec.yaml` files to check and update dependencies.

## Features

- Parse `package.json` files (npm dependencies)
- Parse `pubspec.yaml` files (Dart/Flutter pub dependencies)
- Check for outdated dependencies
- Update dependencies to their latest versions
- Preserve version prefixes (^, ~, >=, etc.)
- Optionally check for updates while respecting semver constraints
- Optionally check for peer dependencies updates in `package.json`
- Optional monorepo support for npm workspaces
- Built in support for private registries and hosted packages
- Uses cache when running multiple `bump` commands in quick succession

## Installation

### Via Homebrew (Recommended)

```bash
# Add the tap
brew tap MilosRandelovic/homebrew-bump

# Install bump
brew install bump
```

### From source

```bash
go build -o bump
```

### Direct download

Download the latest release from the [GitHub releases page](https://github.com/MilosRandelovic/homebrew-bump/releases).

## Usage

### Check for outdated dependencies (automatic package file detection)

```bash
bump
```

### Update dependencies to latest versions

```bash
bump --update
# or
bump -u
```

### Respect semver constraints

```bash
bump --semver
# or
bump -s
```

This mode will:

- Only show updates that are compatible with version constraints (`^` and `~`)
- Skip packages with hardcoded versions (no prefix)
- Skip updates that would violate semver rules

### Parse monorepo workspaces

```bash
bump --monorepo
# or
bump -m
```

This mode will:

- Detect workspaces from root `package.json` file
- Parse all workspace packages matching glob patterns
- Check and update dependencies across all workspace packages
- Group output by file for clarity

### Enable verbose output

```bash
bump --verbose
# or
bump -v
```

### Combine options

You can merge shorthand flags for concise commands:

```bash
# Update with verbose output
bump -uv

# Update with semver constraints
bump --update --semver
# or
bump -us

# Check with semver constraints and verbose output
bump -sv
```

### Show version

```bash
bump --version
# or
bump -V
```

## Command Line Options

- `--update, -u`: Update dependencies to latest versions
- `--semver, -s`: Respect semver constraints (^, ~) and skip hardcoded versions
- `--verbose, -v`: Enable verbose output
- `--no-cache, -C`: Disable caching of registry lookups
- `--include-peers, -P`: Include peer dependencies when updating (npm only)
- `--monorepo, -m`: Parse workspace packages in monorepo (npm only)
- `--version, -V`: Show version information
- `--help, -h`: Show help information

**Note:** Long-form flags use double dashes (`--update`), shorthand flags use single dash (`-u`). Shorthand flags can be merged (e.g., `-us` for update with semver).

## Supported File Types

### package.json (npm)

- Regular dependencies
- Dev dependencies
- Fetches latest versions from npm registry

### pubspec.yaml (Dart/Flutter)

- Regular dependencies
- Dev dependencies
- Skips Flutter SDK dependency
- Handles complex dependencies (git, path, hosted)
- Fetches latest versions from pub.dev API

## Examples

### Example package.json

```json
{
  "dependencies": {
    "react": "^18.0.0",
    "axios": "~1.3.0"
  },
  "devDependencies": {
    "typescript": ">=4.9.0"
  }
}
```

### Example pubspec.yaml

```yaml
dependencies:
  flutter:
    sdk: flutter
  http: ^0.13.0
  shared_preferences: ^2.0.0

dev_dependencies:
  flutter_test:
    sdk: flutter
  mockito: ^5.3.0
```

### Example monorepo package.json

```json
{
  "name": "my-monorepo",
  "private": true,
  "workspaces": ["packages/*", "apps/*"],
  "dependencies": {
    "typescript": "^5.0.0"
  }
}
```

With workspace packages in `packages/package-a/package.json`, `packages/package-b/package.json`, etc.

## Architecture

The project is organized into the following packages:

- `main.go`: CLI interface and application entry point
- `internal/output`: Output formatting, progress bars, and help text
- `internal/parser`: Handles parsing and file detection for package.json and pubspec.yaml
- `internal/updater`: Handles checking for updates and updating dependency files
- `internal/shared`: Common types, utilities, and interfaces
- `internal/npm`: npm-specific registry client, parser, and configuration
- `internal/pub`: Dart/Flutter pub-specific registry client, parser, and configuration

The CLI uses [spf13/pflag](https://github.com/spf13/pflag) for POSIX-compliant flag parsing with support for both long-form (`--flag`) and shorthand (`-f`) options, including merged shorthands (`-us`).

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the terms specified in the LICENSE file.
