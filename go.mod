module github.com/MilosRandelovic/homebrew-bump

go 1.26.1

require (
	github.com/MilosRandelovic/bump-core v0.0.0
	github.com/spf13/pflag v1.0.10
)

require github.com/Masterminds/semver/v3 v3.4.0 // indirect

replace github.com/MilosRandelovic/bump-core => ../bump-core
