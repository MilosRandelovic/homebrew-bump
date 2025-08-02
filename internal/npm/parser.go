package npm

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/MilosRandelovic/homebrew-bump/internal/shared"
)

// Parser handles NPM package.json parsing
type Parser struct{}

// PackageJson represents the structure of a package.json file
type PackageJson struct {
	Dependencies    map[string]string `json:"dependencies,omitempty"`
	DevDependencies map[string]string `json:"devDependencies,omitempty"`
}

// NewParser creates a new NPM parser
func NewParser() *Parser {
	return &Parser{}
}

// ParseDependencies parses a package.json file and extracts dependencies
func (p *Parser) ParseDependencies(filePath string) ([]shared.Dependency, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var pkg PackageJson
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	var dependencies []shared.Dependency

	// Parse regular dependencies
	for name, version := range pkg.Dependencies {
		dependencies = append(dependencies, shared.Dependency{
			Name:            name,
			Version:         shared.CleanVersion(version),
			OriginalVersion: version,
		})
	}

	// Parse dev dependencies
	for name, version := range pkg.DevDependencies {
		dependencies = append(dependencies, shared.Dependency{
			Name:            name,
			Version:         shared.CleanVersion(version),
			OriginalVersion: version,
		})
	}

	return dependencies, nil
}

// GetFileType returns the file type this parser handles
func (p *Parser) GetFileType() string {
	return "npm"
}

// Ensure Parser implements the interface
var _ shared.Parser = (*Parser)(nil)
