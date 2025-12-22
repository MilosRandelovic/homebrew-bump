package shared

import "strings"

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
