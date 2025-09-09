package shared

import "fmt"

// DependencyType represents the type of dependency
type DependencyType int

const (
	Dependencies DependencyType = iota
	DevDependencies
	PeerDependencies
)

// String returns the string representation of DependencyType
func (dependencyType DependencyType) String() string {
	switch dependencyType {
	case Dependencies:
		return "dependencies"
	case DevDependencies:
		return "devDependencies"
	case PeerDependencies:
		return "peerDependencies"
	default:
		panic(fmt.Sprintf("unknown DependencyType: %d", dependencyType))
	}
}

// Dependency represents a package dependency
type Dependency struct {
	Name            string
	Version         string         // Clean version for API calls (e.g., "1.2.3")
	OriginalVersion string         // Original version with prefixes (e.g., "^1.2.3")
	HostedURL       string         // For hosted packages, the registry URL (empty for pub.dev/npmjs.org)
	Type            DependencyType // Type of dependency (dependencies, devDependencies, peerDependencies)
	LineNumber      int            // Line number where this dependency is defined (1-based)
}

// OutdatedDependency represents a dependency that has a newer version available
type OutdatedDependency struct {
	Name            string
	CurrentVersion  string
	LatestVersion   string
	OriginalVersion string         // Original version with prefixes (e.g., "^1.2.3")
	HostedURL       string         // For hosted packages, the registry URL (empty for pub.dev/npmjs.org)
	Type            DependencyType // Type of dependency (dependencies, devDependencies, peerDependencies)
	LineNumber      int            // Line number where this dependency is defined (1-based)
}

// CheckResult contains the results of checking dependencies
type CheckResult struct {
	Outdated      []OutdatedDependency
	Errors        []DependencyError
	SemverSkipped []SemverSkipped
}

// DependencyError represents an error that occurred while checking a dependency
type DependencyError struct {
	Name  string
	Error string
}

// SemverSkipped represents a dependency that was skipped due to semver constraints
type SemverSkipped struct {
	Name            string
	CurrentVersion  string
	LatestVersion   string
	OriginalVersion string
	Reason          string
}

// SemverChange represents the type of version change
type SemverChange int

const (
	PatchChange SemverChange = iota
	MinorChange
	MajorChange
)

// Parser interface defines the contract for parsing dependencies from files
type Parser interface {
	ParseDependencies(filePath string) ([]Dependency, error)
	GetFileType() string
}

// Updater interface defines the contract for updating dependencies in files
type Updater interface {
	UpdateDependencies(filePath string, outdated []OutdatedDependency, verbose bool, semver bool, includePeerDependencies bool) error
	GetFileType() string
}

// RegistryClient interface defines the contract for fetching package information
type RegistryClient interface {
	GetLatestVersionFromRegistry(packageName, registryURL string, verbose bool, cache *Cache) (string, error)
	GetBothLatestVersions(packageName, constraint, registryURL string, verbose bool, cache *Cache) (absoluteLatest, constraintLatest string, err error)
	GetFileType() string
}
