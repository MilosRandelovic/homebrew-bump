package pub

import (
	"fmt"
	"os"
	"strings"

	"github.com/MilosRandelovic/homebrew-bump/internal/shared"
)

// Parser handles Dart pubspec.yaml parsing
type Parser struct{}

// NewParser creates a new Dart parser
func NewParser() *Parser {
	return &Parser{}
}

// ParseDependencies parses a pubspec.yaml file and extracts dependencies
func (parser *Parser) ParseDependencies(filePath string) ([]shared.Dependency, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	var dependencies []shared.Dependency

	// Track which section we're in
	var currentSection shared.DependencyType
	var inSection bool
	var currentPackage *packageInfo

	for lineNumber, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Check if we're entering a dependency section
		if strings.HasPrefix(trimmedLine, "dependencies:") {
			// Finalize any pending package from previous section
			if currentPackage != nil {
				if dependency := currentPackage.toDependency(currentSection); dependency != nil {
					dependencies = append(dependencies, *dependency)
				}
			}
			currentSection = shared.Dependencies
			inSection = true
			currentPackage = nil
			continue
		} else if strings.HasPrefix(trimmedLine, "dev_dependencies:") {
			// Finalize any pending package from previous section
			if currentPackage != nil {
				if dependency := currentPackage.toDependency(currentSection); dependency != nil {
					dependencies = append(dependencies, *dependency)
				}
			}
			currentSection = shared.DevDependencies
			inSection = true
			currentPackage = nil
			continue
		}

		// Check if we're leaving a section (non-indented line that's not a comment)
		if inSection && len(line) > 0 && line[0] != ' ' && line[0] != '\t' && trimmedLine != "" && !strings.HasPrefix(trimmedLine, "#") {
			// Finalize any pending package
			if currentPackage != nil {
				if dependency := currentPackage.toDependency(currentSection); dependency != nil {
					dependencies = append(dependencies, *dependency)
				}
				currentPackage = nil
			}
			inSection = false
		}

		// If we're in a section, look for dependency definitions
		if inSection && len(line) > 0 && (line[0] == ' ' || line[0] == '\t') && !strings.HasPrefix(trimmedLine, "#") && strings.Contains(trimmedLine, ":") {
			parts := strings.SplitN(trimmedLine, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				// Check if this is a top-level package name (2 spaces indentation)
				if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") {
					// Finalize previous package if any
					if currentPackage != nil {
						if dependency := currentPackage.toDependency(currentSection); dependency != nil {
							dependencies = append(dependencies, *dependency)
						}
					}

					// Start new package
					currentPackage = &packageInfo{
						name:       key,
						lineNumber: lineNumber + 1,
					}

					if value != "" {
						// Simple dependency (name: version)
						currentPackage.version = cleanQuotes(value)
					}
				} else if strings.HasPrefix(line, "    ") {
					// This is a sub-property of the current package (4+ spaces indentation)
					if currentPackage != nil {
						switch key {
						case "version":
							currentPackage.version = cleanQuotes(value)
							currentPackage.versionLineNumber = lineNumber + 1
						case "hosted":
							currentPackage.hostedURL = cleanQuotes(value)
						case "sdk":
							currentPackage.sdk = value
						}
					}
				}
			}
		}
	}

	// Finalize any pending package
	if currentPackage != nil {
		if dependency := currentPackage.toDependency(currentSection); dependency != nil {
			dependencies = append(dependencies, *dependency)
		}
	}

	return dependencies, nil
}

// packageInfo holds information about a package being parsed
type packageInfo struct {
	name              string
	version           string
	hostedURL         string
	sdk               string
	lineNumber        int
	versionLineNumber int
}

// toDependency converts packageInfo to shared.Dependency if it should be included
func (pkg *packageInfo) toDependency(section shared.DependencyType) *shared.Dependency {
	// Skip SDK dependencies
	if pkg.sdk != "" {
		return nil
	}

	if !shouldIncludeDependency(pkg.name, pkg.version) {
		return nil
	}

	// Use version line number if available, otherwise use package line number
	lineNum := pkg.lineNumber
	if pkg.versionLineNumber > 0 {
		lineNum = pkg.versionLineNumber
	}

	dependency := &shared.Dependency{
		Name:            pkg.name,
		Version:         shared.CleanVersion(pkg.version),
		OriginalVersion: pkg.version,
		Type:            section,
		LineNumber:      lineNum,
	}

	// Set hosted URL for non-pub.dev hosted packages
	if pkg.hostedURL != "" && !strings.Contains(pkg.hostedURL, "pub.dev") {
		dependency.HostedURL = pkg.hostedURL
	}

	return dependency
}

// cleanQuotes removes surrounding quotes from a string
func cleanQuotes(s string) string {
	s = strings.TrimSpace(s)
	if (strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`)) ||
		(strings.HasPrefix(s, `'`) && strings.HasSuffix(s, `'`)) {
		return s[1 : len(s)-1]
	}
	return s
}

// shouldIncludeDependency checks if a dependency should be included
func shouldIncludeDependency(name, version string) bool {
	// Skip flutter SDK dependency
	if name == "flutter" {
		return false
	}

	// Skip if no version
	if version == "" {
		return false
	}

	// Skip 'any' versions, SDK dependencies, path, git dependencies
	if version == "any" || strings.HasPrefix(version, "sdk:") || strings.HasPrefix(version, "path:") || strings.HasPrefix(version, "git:") {
		return false
	}

	return true
}

// GetFileType returns the file type this parser handles
func (parser *Parser) GetFileType() string {
	return "pub"
}

// Ensure Parser implements the interface
var _ shared.Parser = (*Parser)(nil)
