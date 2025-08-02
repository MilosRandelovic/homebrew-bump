# Bump

A Go utility that parses `package.json` and `pubspec.yaml` files to check and update dependencies.

## Features

- Parse `package.json` files (NPM dependencies)
- Parse `pubspec.yaml` files (Dart/Flutter pub dependencies)
- Check for outdated dependencies
- Update dependencies to their latest versions
- Preserve version prefixes (^, ~, >=, etc.)
- Optionally check for updates while respecting semver constraints
- Verbose output option

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
bump -update
# or
bump -u
```

### Respect semver constraints

```bash
bump -semver
# or
bump -s
```

This mode will:

- Only show updates that are compatible with version constraints (`^` and `~`)
- Skip packages with hardcoded versions (no prefix)
- Skip updates that would violate semver rules

### Enable verbose output

```bash
bump -verbose
# or
bump -v
```

### Combine options

```bash
bump -update -verbose
# or
bump -u -v

# Update with semver constraints
bump -update -semver
# or
bump -u -s

# Check with semver constraints and verbose output
bump -semver -verbose
# or
bump -s -v
```

### Show version

```bash
bump -version
# or
bump -V
```

## Command Line Options

- `-update, -u`: Update dependencies to latest versions
- `-semver, -s`: Respect semver constraints (^, ~) and skip hardcoded versions
- `-verbose, -v`: Enable verbose output
- `-version, -V`: Show version information
- `-help, -h`: Show help information

## Supported File Types

### package.json (NPM)

- Regular dependencies
- Dev dependencies
- Fetches latest versions from NPM registry

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

## Architecture

The project is organized into the following packages:

- `main.go`: CLI interface and application entry point
- `internal/parser`: Handles parsing of package.json and pubspec.yaml files
- `internal/updater`: Handles checking for updates and updating dependency files

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the terms specified in the LICENSE file.
