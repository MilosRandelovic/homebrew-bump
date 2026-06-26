# Changelog

Starting with v2.0.0, all core logic lives in [bump-core](https://github.com/MilosRandelovic/bump-core). See the [bump-core changelog](https://github.com/MilosRandelovic/bump-core/blob/main/CHANGELOG.md) for details on version checking, parsing, and update logic changes.

This file tracks CLI-specific changes (flags, output formatting, terminal UI).

## [2.0.0]

- Extracted all core logic into [bump-core](https://github.com/MilosRandelovic/bump-core)
- homebrew-bump is now a thin CLI wrapper that imports bump-core as a Go module
- Version is sourced from bump-core (`shared.Version`)

## [1.3.0]

- Add monorepo support to npm parsing

## [1.2.0]

- Ability to merge shorthand flags when calling bump, changed long flags to double-dash

## [1.1.1]

- Fixed peer dependency handling

## [1.1.0]

- Use cache when running multiple bump commands in quick succession
- Improved semver version handling
- Fixed handling of pre-release versions
- Console formatting improvements

## [1.0.1]

- Fixed a number of semver version resolution issues
- Fixed not taking npm deprecated versions into account

## [1.0.0]

- Initial release
