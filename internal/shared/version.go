package shared

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// CleanVersion removes prefix characters (^, ~, >=, etc.) from version strings
func CleanVersion(version string) string {
	prefixRegex := regexp.MustCompile(`^[\^~>=<]+`)
	return prefixRegex.ReplaceAllString(version, "")
}

// HasSemanticPrefix checks if version has semantic versioning prefix
func HasSemanticPrefix(version string) bool {
	if version == "" {
		return false
	}

	// Common semantic versioning prefixes
	prefixes := []string{"^", "~", ">=", ">", "<=", "<"}
	hasPrefix := false
	for _, prefix := range prefixes {
		if strings.HasPrefix(version, prefix) {
			hasPrefix = true
			break
		}
	}

	// If it doesn't start with a semantic prefix, it's not semantic
	if !hasPrefix {
		return false
	}

	// Check if it contains mixed semantic and non-semantic parts
	// Split by spaces and check if there are multiple parts
	parts := strings.Fields(version)
	if len(parts) > 1 {
		// For multiple parts, all parts should have semantic prefixes or be range operators
		for _, part := range parts {
			// Skip range operators
			if part == "&&" || part == "||" {
				continue
			}

			partHasPrefix := false
			for _, prefix := range prefixes {
				if strings.HasPrefix(part, prefix) {
					partHasPrefix = true
					break
				}
			}

			// If any part doesn't have a semantic prefix, it's mixed
			if !partHasPrefix {
				return false
			}
		}
	}

	return true
}

// GetVersionPrefix returns the prefix of a version string (^, ~, >=, etc.)
func GetVersionPrefix(version string) string {
	prefixRegex := regexp.MustCompile(`^([\^~>=<]+)`)
	matches := prefixRegex.FindStringSubmatch(version)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// IsSemverCompatible checks if a target version satisfies a semver constraint
func IsSemverCompatible(constraint, targetVersion string) bool {
	// Handle the case where constraint has no prefix (exact version)
	if GetVersionPrefix(constraint) == "" {
		cleanConstraint := CleanVersion(constraint)
		cleanTarget := CleanVersion(targetVersion)
		return cleanConstraint == cleanTarget
	}

	// Parse the constraint using Masterminds semver
	c, err := semver.NewConstraint(constraint)
	if err != nil {
		return false
	}

	// Parse the target version
	v, err := semver.NewVersion(targetVersion)
	if err != nil {
		return false
	}

	return c.Check(v)
}

// FindBothLatestVersions finds both the absolute latest version and the latest version satisfying a constraint
// Returns (absoluteLatest, constraintLatest, error)
func FindBothLatestVersions(versions []string, constraint string) (string, string, error) {
	if len(versions) == 0 {
		return "", "", fmt.Errorf("no versions provided")
	}

	// Check if the current version (from constraint) is a pre-release
	currentVersion := CleanVersion(constraint)

	// Parse versions using Masterminds semver
	var semverVersions []*semver.Version
	var versionMap = make(map[string]string) // semver string -> original string

	for _, v := range versions {
		sv, err := semver.NewVersion(v)
		if err != nil {
			// Skip invalid versions
			continue
		}
		semverVersions = append(semverVersions, sv)
		versionMap[sv.String()] = v
	}

	if len(semverVersions) == 0 {
		return "", "", fmt.Errorf("no valid semver versions found")
	}

	// Parse current version to check if it's pre-release
	currentSemver, err := semver.NewVersion(currentVersion)
	if err != nil {
		return "", "", fmt.Errorf("invalid current version: %s", currentVersion)
	}

	includePrerelease := currentSemver.Prerelease() != ""

	// Filter versions based on whether current version is pre-release
	var filteredVersions []*semver.Version
	for _, sv := range semverVersions {
		if includePrerelease {
			// If current is pre-release, include all versions
			filteredVersions = append(filteredVersions, sv)
		} else {
			// If current is stable, only include stable versions
			if sv.Prerelease() == "" {
				filteredVersions = append(filteredVersions, sv)
			}
		}
	}

	if len(filteredVersions) == 0 {
		if includePrerelease {
			return "", "", fmt.Errorf("no versions available")
		} else {
			return "", "", fmt.Errorf("no stable versions available")
		}
	}

	// Sort versions
	sort.Slice(filteredVersions, func(i, j int) bool {
		return filteredVersions[i].LessThan(filteredVersions[j])
	})

	// Absolute latest is the last in sorted filtered versions
	absoluteLatest := versionMap[filteredVersions[len(filteredVersions)-1].String()]

	// Find latest satisfying constraint
	var constraintLatest string

	// Parse the constraint using Masterminds semver
	constraintStr := constraint
	if GetVersionPrefix(constraint) == "" {
		// No prefix means exact version, we want newer versions
		constraintStr = ">" + currentVersion
	}

	c, err := semver.NewConstraint(constraintStr)
	if err != nil {
		return absoluteLatest, "", fmt.Errorf("invalid constraint: %s", constraintStr)
	}

	// Find the latest version that satisfies the constraint
	for i := len(filteredVersions) - 1; i >= 0; i-- {
		if c.Check(filteredVersions[i]) {
			constraintLatest = versionMap[filteredVersions[i].String()]
			break
		}
	}

	if constraintLatest == "" {
		return absoluteLatest, "", fmt.Errorf("no versions satisfy the constraint %s", constraint)
	}

	return absoluteLatest, constraintLatest, nil
}

// GetSemverChange determines the type of version change (major, minor, patch)
func GetSemverChange(currentVer, latestVer string) SemverChange {
	current, err1 := semver.NewVersion(CleanVersion(currentVer))
	latest, err2 := semver.NewVersion(CleanVersion(latestVer))

	if err1 != nil || err2 != nil {
		return PatchChange
	}

	// If the latest version is less than or equal to current, default to patch
	if !latest.GreaterThan(current) {
		return PatchChange
	}

	if latest.Major() != current.Major() {
		return MajorChange
	} else if latest.Minor() != current.Minor() {
		return MinorChange
	} else if latest.Patch() != current.Patch() {
		return PatchChange
	}

	return PatchChange
}
