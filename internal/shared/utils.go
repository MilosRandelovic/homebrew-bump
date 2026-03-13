package shared

import (
	"path/filepath"
	"slices"
	"strings"
)

// ExtractHostname extracts hostname from a URL for registry matching
func ExtractHostname(url string) string {
	// Remove protocol
	url, _ = strings.CutPrefix(url, "https://")
	url, _ = strings.CutPrefix(url, "http://")

	// Remove path
	if index := strings.Index(url, "/"); index != -1 {
		url = url[:index]
	}

	// Remove port
	if index := strings.Index(url, ":"); index != -1 {
		url = url[:index]
	}

	return strings.ToLower(url)
}

// SortFilesByDepth sorts file paths by path depth (shortest first = root), then alphabetically
func SortFilesByDepth(files []string) {
	slices.SortFunc(files, func(a, b string) int {
		depthA := strings.Count(a, string(filepath.Separator))
		depthB := strings.Count(b, string(filepath.Separator))
		if depthA != depthB {
			return depthA - depthB
		}
		return strings.Compare(a, b)
	})
}
