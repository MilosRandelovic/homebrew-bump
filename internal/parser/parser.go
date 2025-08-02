package parser

import (
	"fmt"

	"github.com/MilosRandelovic/homebrew-bump/internal/dart"
	"github.com/MilosRandelovic/homebrew-bump/internal/npm"
	"github.com/MilosRandelovic/homebrew-bump/internal/shared"
)

// ParseDependencies parses dependencies from a file based on its type
func ParseDependencies(filePath, fileType string) ([]shared.Dependency, error) {
	parser, err := getParser(fileType)
	if err != nil {
		return nil, err
	}
	return parser.ParseDependencies(filePath)
}

// getParser returns the appropriate parser for the given file type
func getParser(fileType string) (shared.Parser, error) {
	switch fileType {
	case "npm":
		return npm.NewParser(), nil
	case "dart":
		return dart.NewParser(), nil
	default:
		return nil, fmt.Errorf("unsupported file type: %s", fileType)
	}
}
