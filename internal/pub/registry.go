package pub

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/MilosRandelovic/homebrew-bump/internal/shared"
)

// RegistryClient handles pub.dev and private registry operations
type RegistryClient struct{}

// PubDevPackageInfo represents the response from pub.dev API
type PubDevPackageInfo struct {
	Latest struct {
		Version string `json:"version"`
	} `json:"latest"`
}

// NewRegistryClient creates a new pub registry client
func NewRegistryClient() *RegistryClient {
	return &RegistryClient{}
}

// GetLatestVersionFromRegistry fetches the latest version from a specific registry
func (client *RegistryClient) GetLatestVersionFromRegistry(packageName, registryURL string, verbose bool, cache *shared.Cache) (string, error) {
	// Check cache first if enabled
	if cache != nil {
		key := shared.GenerateCacheKey(packageName, "pub", "", "*")
		if entry, ok := cache.Get(key); ok {
			if verbose {
				fmt.Printf("Cache hit: %s\n", packageName)
			}
			return entry.AbsoluteLatest, nil
		}
	}

	body, err := client.fetchPackageInfo(packageName, registryURL, verbose)
	if err != nil {
		return "", err
	}

	var packageInfo PubDevPackageInfo
	if err := json.Unmarshal(body, &packageInfo); err != nil {
		return "", fmt.Errorf("failed to parse pub.dev response: %w", err)
	}

	// Cache the result if cache is enabled
	if cache != nil {
		entry := shared.CacheEntry{
			PackageName:      packageName,
			Type:             "pub",
			CurrentVersion:   "",
			Constraint:       "*",
			AbsoluteLatest:   packageInfo.Latest.Version,
			ConstraintLatest: packageInfo.Latest.Version,
			Expiry:           time.Now().Add(10 * time.Minute),
		}
		cache.Set(entry)
	}
	return packageInfo.Latest.Version, nil
}

// GetBothLatestVersions fetches both the absolute latest version and the latest version satisfying a constraint
func (client *RegistryClient) GetBothLatestVersions(packageName, constraint, registryURL string, verbose bool, cache *shared.Cache) (string, string, error) {
	// Check cache first if enabled
	if cache != nil {
		key := shared.GenerateCacheKey(packageName, "pub", "", constraint)
		if entry, ok := cache.Get(key); ok {
			if verbose {
				fmt.Printf("Cache hit: %s\n", packageName)
			}
			return entry.AbsoluteLatest, entry.ConstraintLatest, nil
		}
	}

	body, err := client.fetchPackageInfo(packageName, registryURL, verbose)
	if err != nil {
		return "", "", err
	}

	var packageInfo struct {
		Versions []struct {
			Version string `json:"version"`
		} `json:"versions"`
	}

	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&packageInfo); err != nil {
		return "", "", fmt.Errorf("error decoding response: %w", err)
	}

	// Extract version strings
	var versions []string
	for _, versionInfo := range packageInfo.Versions {
		versions = append(versions, versionInfo.Version)
	}

	absoluteLatest, constraintLatest, err := shared.FindBothLatestVersions(versions, constraint)
	if err != nil {
		return "", "", err
	}

	// Cache the result if cache is enabled
	if cache != nil {
		entry := shared.CacheEntry{
			PackageName:      packageName,
			Type:             "pub",
			CurrentVersion:   "",
			Constraint:       constraint,
			AbsoluteLatest:   absoluteLatest,
			ConstraintLatest: constraintLatest,
			Expiry:           time.Now().Add(10 * time.Minute),
		}
		cache.Set(entry)
	}

	return absoluteLatest, constraintLatest, nil
}

// fetchPackageInfo is a shared method to fetch package information from registries
func (client *RegistryClient) fetchPackageInfo(packageName, registryURL string, verbose bool) ([]byte, error) {
	// Parse pub configuration
	config, err := parsePubConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to parse pub config: %w", err)
	}

	var targetRegistry *RegistryConfig
	var url string

	if registryURL != "" {
		// Use specified registry
		hostname := shared.ExtractHostname(registryURL)
		if regConfig, exists := config.Registries[hostname]; exists {
			targetRegistry = &regConfig
		} else {
			// Create temporary config for this registry
			targetRegistry = &RegistryConfig{
				URL: registryURL,
			}
		}
		url = fmt.Sprintf("%s/api/packages/%s", targetRegistry.URL, packageName)
	} else {
		// Use default pub.dev registry
		if pubDevConfig, exists := config.Registries["pub.dev"]; exists {
			targetRegistry = &pubDevConfig
		} else {
			targetRegistry = &RegistryConfig{
				URL: "https://pub.dev",
			}
		}
		url = fmt.Sprintf("%s/api/packages/%s", targetRegistry.URL, packageName)
	}

	if verbose {
		fmt.Printf("Checking PUB package: %s (registry: %s)\n", packageName, targetRegistry.URL)
	}

	httpClient := &http.Client{Timeout: 10 * time.Second}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication if available for this registry
	if targetRegistry.AuthToken != "" {
		request.Header.Set("Authorization", "Bearer "+targetRegistry.AuthToken)
		if verbose {
			fmt.Printf("Using authentication for registry: %s\n", targetRegistry.URL)
		}
	}

	response, err := httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch package info: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned status %d for %s", response.StatusCode, packageName)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return body, nil
}

// GetFileType returns the file type this registry client handles
func (client *RegistryClient) GetFileType() string {
	return "pub"
}

// Ensure RegistryClient implements the interface
var _ shared.RegistryClient = (*RegistryClient)(nil)
