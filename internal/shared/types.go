package shared

// Dependency represents a package dependency
type Dependency struct {
	Name            string
	Version         string // Clean version for API calls (e.g., "1.2.3")
	OriginalVersion string // Original version with prefixes (e.g., "^1.2.3")
}

// OutdatedDependency represents a dependency that has a newer version available
type OutdatedDependency struct {
	Name            string
	CurrentVersion  string
	LatestVersion   string
	OriginalVersion string // Original version with prefixes (e.g., "^1.2.3")
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

// SemanticVersion represents a parsed semantic version
type SemanticVersion struct {
	Major int
	Minor int
	Patch int
}

// Parser interface defines the contract for parsing dependencies from files
type Parser interface {
	ParseDependencies(filePath string) ([]Dependency, error)
	GetFileType() string
}

// Updater interface defines the contract for updating dependencies in files
type Updater interface {
	UpdateDependencies(filePath string, outdated []OutdatedDependency, verbose bool, semver bool) error
	GetFileType() string
}

// RegistryClient interface defines the contract for fetching package information
type RegistryClient interface {
	GetLatestVersion(packageName string, verbose bool) (string, error)
	GetFileType() string
}
