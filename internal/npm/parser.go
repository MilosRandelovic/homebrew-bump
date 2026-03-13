package npm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/MilosRandelovic/homebrew-bump/internal/output"
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
		var packageData struct {
			Workspaces json.RawMessage `json:"workspaces"`
		}

		if err := json.Unmarshal(data, &packageData); err != nil {
			output.VerbosePrintf(options, "Warning: could not parse package.json for workspaces detection (%s): %v\n", filePath, err)
			return parser.parseFile(filePath, options)
		}

		workspacePatterns, err := extractWorkspacePatterns(packageData.Workspaces)
		if err != nil {
			output.VerbosePrintf(options, "Warning: invalid workspaces format in %s: %v\n", filePath, err)
			return parser.parseFile(filePath, options)
		}

		if len(workspacePatterns) > 0 {
			return parser.parseWorkspaces(filePath, workspacePatterns, options)
		}
	}

	return parser.parseFile(filePath, options)
}

func (parser *Parser) parseFile(filePath string, options shared.Options) ([]shared.Dependency, error) {
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
			if options.IncludePeerDependencies {
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

func (parser *Parser) parseWorkspaces(rootPath string, patterns []string, options shared.Options) ([]shared.Dependency, error) {
	rootDir := filepath.Dir(rootPath)
	all := []shared.Dependency{}

	root, err := parser.parseFile(rootPath, options)
	if err != nil {
		return nil, err
	}
	all = append(all, root...)

	for _, pattern := range patterns {
		if strings.HasPrefix(pattern, "!") {
			output.VerbosePrintf(options, "Warning: workspace negation pattern %q is not supported and was skipped\n", pattern)
			continue
		}

		matches, err := filepath.Glob(filepath.Join(rootDir, pattern))
		if err != nil {
			output.VerbosePrintf(options, "Warning: invalid workspace glob pattern %q: %v\n", pattern, err)
			continue
		}
		if len(matches) == 0 {
			output.VerbosePrintf(options, "Warning: workspace pattern %q matched no directories\n", pattern)
			continue
		}

		sort.Strings(matches)
		for _, match := range matches {
			if info, err := os.Stat(match); err == nil && info.IsDir() {
				pkgPath := filepath.Join(match, "package.json")
				if _, err := os.Stat(pkgPath); err == nil {
					if deps, err := parser.parseFile(pkgPath, options); err == nil {
						all = append(all, deps...)
					} else {
						output.VerbosePrintf(options, "Warning: failed to parse workspace package %s: %v\n", pkgPath, err)
					}
				} else {
					output.VerbosePrintf(options, "Warning: workspace directory %s has no package.json\n", match)
				}
			}
		}
	}

	return all, nil
}

func extractWorkspacePatterns(workspacesRaw json.RawMessage) ([]string, error) {
	if len(workspacesRaw) == 0 || string(workspacesRaw) == "null" {
		return nil, nil
	}

	var asArray []string
	if err := json.Unmarshal(workspacesRaw, &asArray); err == nil {
		return asArray, nil
	}

	var asObject struct {
		Packages []string `json:"packages"`
	}
	if err := json.Unmarshal(workspacesRaw, &asObject); err == nil {
		return asObject.Packages, nil
	}

	return nil, fmt.Errorf("expected an array of strings or object with packages field")
}

// GetRegistryType returns the registry type this parser handles
func (parser *Parser) GetRegistryType() shared.RegistryType {
	return shared.Npm
}

// Ensure Parser implements the interface
var _ shared.Parser = (*Parser)(nil)
