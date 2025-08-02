package shared

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// CleanVersion removes prefix characters (^, ~, >=, etc.) from version strings
func CleanVersion(version string) string {
	re := regexp.MustCompile(`^[\^~>=<]+`)
	return re.ReplaceAllString(version, "")
}

// GetVersionPrefix extracts the version prefix (^, ~, >=, etc.) from a version string
func GetVersionPrefix(version string) string {
	re := regexp.MustCompile(`^([\^~>=<]+)`)
	matches := re.FindStringSubmatch(version)
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
