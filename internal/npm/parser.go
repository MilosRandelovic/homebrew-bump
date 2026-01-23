package npm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MilosRandelovic/homebrew-bump/internal/shared"
)

// Parser handles npm package.json parsing
type Parser struct{}

// NewParser creates a new npm parser
func NewParser() *Parser {
	return &Parser{}
}

// ParseDependencies parses a package.json file and extracts dependencies
func (parser *Parser) ParseDependencies(filePath string, options shared.Options) ([]shared.Dependency, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	if options.Monorepo {
		var pkg struct {
			Workspaces []string `json:"workspaces"`
		}
		json.Unmarshal(data, &pkg)

		if len(pkg.Workspaces) > 0 {
			return parser.parseWorkspaces(filePath, pkg.Workspaces, options.IncludePeerDependencies)
		}
	}

	return parser.parseFile(filePath, options.IncludePeerDependencies)
}

func (parser *Parser) parseFile(filePath string, includePeerDependencies bool) ([]shared.Dependency, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse line by line to track line numbers and extract dependencies
	lines := strings.Split(string(data), "\n")
	var dependencies []shared.Dependency

	// Track which section we're in
	var currentSection shared.DependencyType
	var inSection bool

	for lineNumber, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Check if we're entering a dependency section
		if strings.Contains(trimmedLine, `"dependencies"`) && strings.Contains(trimmedLine, `:`) {
			currentSection = shared.Dependencies
			inSection = true
			continue
		} else if strings.Contains(trimmedLine, `"devDependencies"`) && strings.Contains(trimmedLine, `:`) {
			currentSection = shared.DevDependencies
			inSection = true
			continue
		} else if strings.Contains(trimmedLine, `"peerDependencies"`) && strings.Contains(trimmedLine, `:`) {
			if includePeerDependencies {
				currentSection = shared.PeerDependencies
				inSection = true
			}
			continue
		}

		// Check if we're leaving a section (closing brace or comma)
		if inSection && (trimmedLine == "}" || trimmedLine == "},") {
			inSection = false
			continue
		}

		// If we're in a section, look for dependency definitions
		if inSection {
			// Look for lines like: "package-name": "version",
			if strings.Contains(trimmedLine, `"`) && strings.Contains(trimmedLine, `:`) {
				// Parse the dependency name and version
				parts := strings.SplitN(trimmedLine, ":", 2)
				if len(parts) == 2 {
					// Extract package name (remove quotes and whitespace)
					nameStr := strings.TrimSpace(parts[0])
					nameStr = strings.Trim(nameStr, `"`)

					// Extract version (remove quotes, whitespace, and trailing comma)
					versionStr := strings.TrimSpace(parts[1])
					versionStr = strings.Trim(versionStr, `",`)
					versionStr = strings.Trim(versionStr, `"`)

					// Basic validation - skip empty names or versions
					if nameStr != "" && versionStr != "" {
						dependencies = append(dependencies, shared.Dependency{
							BaseDependency: shared.BaseDependency{
								Name:            nameStr,
								OriginalVersion: versionStr,
								Type:            currentSection,
								FilePath:        filePath,
								LineNumber:      lineNumber + 1, // Convert to 1-based
							},
							Version: shared.CleanVersion(versionStr),
						})
					}
				}
			}
		}
	}

	return dependencies, nil
}

func (parser *Parser) parseWorkspaces(rootPath string, patterns []string, includePeerDependencies bool) ([]shared.Dependency, error) {
	rootDir := filepath.Dir(rootPath)
	all := []shared.Dependency{}

	root, err := parser.parseFile(rootPath, includePeerDependencies)
	if err != nil {
		return nil, err
	}
	all = append(all, root...)

	for _, pattern := range patterns {
		if strings.HasPrefix(pattern, "!") {
			continue
		}

		matches, _ := filepath.Glob(filepath.Join(rootDir, pattern))
		for _, match := range matches {
			if info, err := os.Stat(match); err == nil && info.IsDir() {
				pkgPath := filepath.Join(match, "package.json")
				if _, err := os.Stat(pkgPath); err == nil {
					if deps, err := parser.parseFile(pkgPath, includePeerDependencies); err == nil {
						all = append(all, deps...)
					}
				}
			}
		}
	}

	return all, nil
}

// GetRegistryType returns the registry type this parser handles
func (parser *Parser) GetRegistryType() shared.RegistryType {
	return shared.Npm
}

// Ensure Parser implements the interface
var _ shared.Parser = (*Parser)(nil)
