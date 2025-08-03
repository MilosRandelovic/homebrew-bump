package shared

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// CleanVersion removes prefix characters (^, ~, >=, etc.) from version strings
func CleanVersion(version string) string {
	prefixRegex := regexp.MustCompile(`^[\^~>=<]+`)
	return prefixRegex.ReplaceAllString(version, "")
}

// HasSemanticPrefix checks if a version string has a semantic versioning prefix like ^, ~, >=
func HasSemanticPrefix(version string) bool {
	return strings.HasPrefix(version, "^") ||
		strings.HasPrefix(version, "~") ||
		strings.HasPrefix(version, ">=")
}

// GetVersionPrefix extracts the version prefix (^, ~, >=, etc.) from a version string
func GetVersionPrefix(version string) string {
	prefixRegex := regexp.MustCompile(`^([\^~>=<]+)`)
	matches := prefixRegex.FindStringSubmatch(version)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// ParseSemanticVersion parses a semantic version string into components
func ParseSemanticVersion(version string) (*SemanticVersion, error) {
	// Handle pre-release and build metadata by splitting on '-' and '+'
	parts := strings.Split(version, "-")
	version = parts[0] // Take only the main version part

	parts = strings.Split(version, "+")
	version = parts[0] // Remove build metadata

	parts = strings.Split(version, ".")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid semantic version: %s", version)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid major version: %s", parts[0])
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid minor version: %s", parts[1])
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid patch version: %s", parts[2])
	}

	return &SemanticVersion{
		Major: major,
		Minor: minor,
		Patch: patch,
	}, nil
}

// IsSemverCompatible checks if the latest version is compatible with the original version constraint
func IsSemverCompatible(originalVersion, latestVersion string) bool {
	prefix := GetVersionPrefix(originalVersion)

	// If no prefix, it's a hardcoded version - not compatible with semver
	if prefix == "" {
		return false
	}

	// If the latest version contains pre-release identifiers, be conservative and skip it
	if strings.Contains(latestVersion, "-") {
		return false
	}

	// Parse current and latest versions
	currentVer, err := ParseSemanticVersion(CleanVersion(originalVersion))
	if err != nil {
		return false
	}

	latestVer, err := ParseSemanticVersion(latestVersion)
	if err != nil {
		return false
	}

	switch prefix {
	case "^":
		// Caret allows changes that do not modify the left-most non-zero digit
		if currentVer.Major == 0 {
			if currentVer.Minor == 0 {
				// ^0.0.x - only patch-level changes
				return latestVer.Major == 0 && latestVer.Minor == 0 && latestVer.Patch >= currentVer.Patch
			}
			// ^0.x.y - minor and patch-level changes
			return latestVer.Major == 0 && latestVer.Minor >= currentVer.Minor
		}
		// ^x.y.z - minor and patch-level changes
		return latestVer.Major == currentVer.Major &&
			(latestVer.Minor > currentVer.Minor ||
				(latestVer.Minor == currentVer.Minor && latestVer.Patch >= currentVer.Patch))

	case "~":
		// Tilde allows patch-level changes if a minor version is specified
		// ~1.2.3 := >=1.2.3 <1.3.0 (reasonably close to 1.2.3)
		return latestVer.Major == currentVer.Major &&
			latestVer.Minor == currentVer.Minor &&
			latestVer.Patch >= currentVer.Patch

	default:
		// For other prefixes like >=, >, <, <=, we'll be conservative and not update
		return false
	}
}

// FindLatestVersionSatisfyingConstraint finds the latest version from a list that satisfies a constraint
func FindLatestVersionSatisfyingConstraint(versions []string, constraint string) (string, error) {
	if len(versions) == 0 {
		return "", fmt.Errorf("no versions provided")
	}

	// Filter versions that satisfy the constraint
	var satisfyingVersions []string
	for _, version := range versions {
		if IsSemverCompatible(constraint, version) {
			satisfyingVersions = append(satisfyingVersions, version)
		}
	}

	if len(satisfyingVersions) == 0 {
		return "", fmt.Errorf("no versions satisfy the constraint %s", constraint)
	}

	// Sort versions to find the latest
	sort.Slice(satisfyingVersions, func(i, j int) bool {
		verI, errI := ParseSemanticVersion(satisfyingVersions[i])
		verJ, errJ := ParseSemanticVersion(satisfyingVersions[j])
		if errI != nil || errJ != nil {
			// Fallback to string comparison if parsing fails
			return satisfyingVersions[i] < satisfyingVersions[j]
		}
		return compareSemanticVersions(verI, verJ) < 0
	})

	// Return the latest (last in sorted array)
	return satisfyingVersions[len(satisfyingVersions)-1], nil
}

// FindBothLatestVersions finds both the absolute latest version and the latest version satisfying a constraint
// Returns (absoluteLatest, constraintLatest, error)
func FindBothLatestVersions(versions []string, constraint string) (string, string, error) {
	if len(versions) == 0 {
		return "", "", fmt.Errorf("no versions provided")
	}

	// Filter out pre-release versions (alpha, beta, etc.)
	var stableVersions []string
	for _, version := range versions {
		if !strings.Contains(version, "-") {
			stableVersions = append(stableVersions, version)
		}
	}

	if len(stableVersions) == 0 {
		return "", "", fmt.Errorf("no stable versions available")
	}

	// Sort stable versions
	sort.Slice(stableVersions, func(i, j int) bool {
		verI, errI := ParseSemanticVersion(stableVersions[i])
		verJ, errJ := ParseSemanticVersion(stableVersions[j])
		if errI != nil || errJ != nil {
			// Fallback to string comparison if parsing fails
			return stableVersions[i] < stableVersions[j]
		}
		return compareSemanticVersions(verI, verJ) < 0
	})

	// Absolute latest is the last in sorted stable versions
	absoluteLatest := stableVersions[len(stableVersions)-1]

	// Find latest satisfying constraint
	var constraintLatest string
	for i := len(stableVersions) - 1; i >= 0; i-- {
		if IsSemverCompatible(constraint, stableVersions[i]) {
			constraintLatest = stableVersions[i]
			break
		}
	}

	if constraintLatest == "" {
		return absoluteLatest, "", fmt.Errorf("no versions satisfy the constraint %s", constraint)
	}

	return absoluteLatest, constraintLatest, nil
}

// compareSemanticVersions compares two semantic versions
// Returns: -1 if a < b, 0 if a == b, 1 if a > b
func compareSemanticVersions(a, b *SemanticVersion) int {
	if a.Major != b.Major {
		if a.Major < b.Major {
			return -1
		}
		return 1
	}
	if a.Minor != b.Minor {
		if a.Minor < b.Minor {
			return -1
		}
		return 1
	}
	if a.Patch != b.Patch {
		if a.Patch < b.Patch {
			return -1
		}
		return 1
	}
	return 0
}
