package dart

import (
	"fmt"
	"os"
	"strings"

	"github.com/MilosRandelovic/homebrew-bump/internal/shared"
	"gopkg.in/yaml.v3"
)

// Parser handles Dart pubspec.yaml parsing
type Parser struct{}

// PubspecYaml represents the structure of a pubspec.yaml file
type PubspecYaml struct {
	Dependencies    map[string]any `yaml:"dependencies,omitempty"`
	DevDependencies map[string]any `yaml:"dev_dependencies,omitempty"`
}

// NewParser creates a new Dart parser
func NewParser() *Parser {
	return &Parser{}
}

// ParseDependencies parses a pubspec.yaml file and extracts dependencies
func (p *Parser) ParseDependencies(filePath string) ([]shared.Dependency, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var pubspec PubspecYaml
	if err := yaml.Unmarshal(data, &pubspec); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	var dependencies []shared.Dependency

	// Parse regular dependencies
	for name, versionInterface := range pubspec.Dependencies {
		// Skip flutter SDK dependency
		if name == "flutter" {
			continue
		}

		version := parseVersionFromInterface(versionInterface)
		// Skip SDK dependencies and empty versions
		if version == "" || strings.HasPrefix(version, "sdk:") {
			continue
		}

		dependencies = append(dependencies, shared.Dependency{
			Name:            name,
			Version:         shared.CleanVersion(version),
			OriginalVersion: version,
		})
	}

	// Parse dev dependencies
	for name, versionInterface := range pubspec.DevDependencies {
		version := parseVersionFromInterface(versionInterface)
		// Skip SDK dependencies and empty versions
		if version == "" || strings.HasPrefix(version, "sdk:") {
			continue
		}

		dependencies = append(dependencies, shared.Dependency{
			Name:            name,
			Version:         shared.CleanVersion(version),
			OriginalVersion: version,
		})
	}

	return dependencies, nil
}

// parseVersionFromInterface extracts version string from interface{}
// Handles both string versions ("^1.0.0") and map versions (git dependencies, etc.)
func parseVersionFromInterface(versionInterface interface{}) string {
	switch v := versionInterface.(type) {
	case string:
		return v
	case map[string]any:
		// Skip SDK dependencies
		if sdk, ok := v["sdk"]; ok {
			return fmt.Sprintf("sdk:%v", sdk)
		}
		// Handle git dependencies or other complex version specifications
		if path, ok := v["path"]; ok {
			return fmt.Sprintf("path:%v", path)
		}
		if git, ok := v["git"]; ok {
			return fmt.Sprintf("git:%v", git)
		}
		if hosted, ok := v["hosted"]; ok {
			return fmt.Sprintf("hosted:%v", hosted)
		}
		return "complex"
	default:
		return ""
	}
}

// GetFileType returns the file type this parser handles
func (p *Parser) GetFileType() string {
	return "dart"
}

// Ensure Parser implements the interface
var _ shared.Parser = (*Parser)(nil)
