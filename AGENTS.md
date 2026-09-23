# homebrew-bump

## Role

`homebrew-bump` is the Homebrew-first `bump` CLI and formula repository. It is a thin frontend over [bump-core](../bump-core): this repository owns flags, terminal presentation, CLI orchestration, smoke coverage, releases, and Homebrew installation; bump-core owns dependency behavior.

The repository name carries the tap prefix, but commands and built artifacts are named `bump`. The formula also installs bump-core's `bump-mcp` command.

## Package boundaries

```text
main.go          CLI flags and orchestration
internal/output/ Terminal presentation
Formula/bump.rb  Homebrew source formula
scripts/         Local and CI smoke validation
```

`main` depends on `internal/output` and the public bump-core packages. `internal/output` may use bump-core result types but must not perform parsing, registry access, or dependency updates. All ecosystem rules and dependency selection stay in bump-core.

`go.mod` pins a released `github.com/MilosRandelovic/bump-core/v2` version. Use an untracked `go.work` only when developing both repositories together; never commit a local replacement.

## Go and CLI conventions

- `commandOptions` owns the complete CLI flag set because `shared.Options` is deliberately limited to bump-core dependency behavior. Pass the full `commandOptions` value through its mapping methods, then pass the resulting `shared.Options` intact to bump-core parsing, checking, and updating and the resulting `output.Config` intact to terminal rendering.
- Name every callable parameter. Keep conventional abbreviations such as `ctx`, `err`, `ok`, `id`, `max`, `min`, `args`, `config`, and `info`; expand names that are not immediate in their scope.
- Keep stdout for normal command output and stderr for progress and failures. Verbose diagnostics are provided through `shared.LogFunc`; pass `nil` when verbose mode is off.
- Create a signal-aware context for registry checks and updates so interrupt and termination signals cancel in-flight work.
- Sort outdated, skipped, and error output by package name. Group dependency output by file, then by `dependencies`, `devDependencies`, and `peerDependencies`, and show file names only when more than one file has updates.
- Use relative display paths and semantic colors: red for major, yellow for minor, green for patch, and cyan for package names.
- Keep public comments as useful contracts. Other comments explain safety, platform constraints, or intent the code cannot express; they do not narrate statements or record changes.

## Safety invariants

- The CLI never implements registry, semver, cache, parsing, target-selection, or file-replacement rules. Change bump-core first when those contracts change.
- `--minimum-age` and `-a` select bump-core's fixed policy of releases published more than 24 hours ago. Bump-core never downgrades the current version under this policy; the age remains non-configurable.
- `--semver` preserves compatible constraints; npm-only peer and workspace options must be rejected for Pub through bump-core validation.
- File updates preserve constraints, formatting, hosted references, and unrelated content through bump-core. Monorepo results are grouped by each dependency's `FilePath`.
- The smoke test never contacts a package registry. It validates built command versions, help, combined shorthand parsing, MCP startup, and missing dependency-file failure.
- The formula installs both `bump` and `bump-mcp`, and its MCP module version must match the direct bump-core dependency in `go.mod`.

## Contracts and siblings

[bump-core](../bump-core) owns the Go API, sidecar protocol, MCP tools, version policies, and dependency target selectors. Merge and release bump-core contract changes before updating this repository to consume them.

[vscode-bump](../vscode-bump) bundles the sidecar and consumes bump-core's newline-delimited JSON protocol. CLI-only changes here do not alter the extension.

Users install from the tap with `brew tap MilosRandelovic/bump`, trust the formula with `brew trust --formula MilosRandelovic/bump/bump`, and then run `brew install bump`. Keep README installation commands and formula caveats aligned with the installed commands.

## Release

The bump-core release workflow opens the dependency update that starts a CLI release. Merging that update to `main` builds the commands, runs `make smoke` as the post-merge product gate before tagging, reads the version from `bump`, creates a matching tag and GitHub source release, and opens a formula update pull request.

The formula update changes only the release archive URL and checksum. Its merge is excluded from another release so the workflow cannot loop. Releases intentionally contain the repository source archives only; no binary assets or backfill workflow are required.

Automated formula update pull requests use `WORKFLOW_PAT` so their branches trigger CI before merge.

The formula uses the canonical `/archive/refs/tags/<tag>.tar.gz` URL. Workflow actions use their latest supported major tags, and every checkout step has an explicit name.

Validate workflow edits locally with `make workflow-lint`; Actionlint is intentionally not part of CI because CI should validate the product rather than validate its own workflow syntax.

The single macOS CI job is deliberate: formula installation, trust, style, and tests require Homebrew, and keeping all validation in one job preserves the repository's simple Homebrew-first workflow.

## Validation

Run the same commands as CI:

```sh
go mod download
test -z "$(gofmt -l .)"
go vet ./...
go test ./...
go test -race ./...
make smoke
brew install shellcheck
shellcheck scripts/*.sh
brew style Formula/bump.rb
brew tap MilosRandelovic/bump "$PWD"
brew trust --formula MilosRandelovic/bump/bump
brew install --build-from-source MilosRandelovic/bump/bump
brew test MilosRandelovic/bump/bump
```
