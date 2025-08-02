package parser

import (
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Dependency represents a package dependency
type Dependency struct {
	Name    string
	Version string
}

// PackageJson represents the structure of a package.json file
type PackageJson struct {
	Dependencies    map[string]string `json:"dependencies,omitempty"`
	DevDependencies map[string]string `json:"devDependencies,omitempty"`
}

// PubspecYaml represents the structure of a pubspec.yaml file
type PubspecYaml struct {
	Dependencies    map[string]interface{} `yaml:"dependencies,omitempty"`
	DevDependencies map[string]interface{} `yaml:"dev_dependencies,omitempty"`
}

// ParseDependencies parses dependencies from a file based on its type
func ParseDependencies(filePath, fileType string) ([]Dependency, error) {
	switch fileType {
	case "npm":
		return parsePackageJson(filePath)
	case "dart":
		return parsePubspecYaml(filePath)
	default:
		return nil, fmt.Errorf("unsupported file type: %s", fileType)
	}
}

// parsePackageJson parses a package.json file and extracts dependencies
func parsePackageJson(filePath string) ([]Dependency, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var pkg PackageJson
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	var dependencies []Dependency

	// Parse regular dependencies
	for name, version := range pkg.Dependencies {
		dependencies = append(dependencies, Dependency{
			Name:    name,
			Version: version,
		})
	}

	// Parse dev dependencies
	for name, version := range pkg.DevDependencies {
		dependencies = append(dependencies, Dependency{
			Name:    name,
			Version: version,
		})
	}

	return dependencies, nil
}

// parsePubspecYaml parses a pubspec.yaml file and extracts dependencies
func parsePubspecYaml(filePath string) ([]Dependency, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var pubspec PubspecYaml
	if err := yaml.Unmarshal(data, &pubspec); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	var dependencies []Dependency

	// Parse regular dependencies
	for name, versionInterface := range pubspec.Dependencies {
		// Skip flutter SDK dependency
		if name == "flutter" {
			continue
		}

		version := parseVersionFromInterface(versionInterface)
		if version != "" {
			dependencies = append(dependencies, Dependency{
				Name:    name,
				Version: version,
			})
		}
	}

	// Parse dev dependencies
	for name, versionInterface := range pubspec.DevDependencies {
		version := parseVersionFromInterface(versionInterface)
		if version != "" {
			dependencies = append(dependencies, Dependency{
				Name:    name,
				Version: version,
			})
		}
	}

	return dependencies, nil
}

// parseVersionFromInterface extracts version string from interface{}
// Handles both string versions ("^1.0.0") and map versions (git dependencies, etc.)
func parseVersionFromInterface(versionInterface interface{}) string {
	switch v := versionInterface.(type) {
	case string:
		return v
	case map[string]interface{}:
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
