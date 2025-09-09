package npm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/MilosRandelovic/homebrew-bump/internal/shared"
)

// RegistryClient handles NPM registry operations
type RegistryClient struct{}

// NpmPackageInfo represents the response from NPM registry
type NpmPackageInfo struct {
	DistTags map[string]string `json:"dist-tags"`
	Versions map[string]struct {
		Version    string `json:"version"`
		Deprecated any    `json:"deprecated,omitempty"`
	} `json:"versions"`
}

// NewRegistryClient creates a new NPM registry client
func NewRegistryClient() *RegistryClient {
	return &RegistryClient{}
}

// GetLatestVersionFromRegistry fetches the latest version from a specific registry
func (client *RegistryClient) GetLatestVersionFromRegistry(packageName, registryURL string, verbose bool, cache *shared.Cache) (string, error) {
	// Check cache first if enabled
	if cache != nil {
		key := shared.GenerateCacheKey(packageName, "npm", "", "*")
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

	var packageInfo NpmPackageInfo
	if err := json.Unmarshal(body, &packageInfo); err != nil {
		return "", fmt.Errorf("failed to parse NPM response: %w", err)
	}

	if latest, ok := packageInfo.DistTags["latest"]; ok {
		// Cache the result if cache is enabled
		if cache != nil {
			entry := shared.CacheEntry{
				PackageName:      packageName,
				Type:             "npm",
				CurrentVersion:   "",
				Constraint:       "*",
				AbsoluteLatest:   latest,
				ConstraintLatest: latest,
				Expiry:           time.Now().Add(10 * time.Minute),
			}
			cache.Set(entry)
		}
		return latest, nil
	}

	return "", fmt.Errorf("no latest version found for %s", packageName)
}

// GetBothLatestVersions fetches both the absolute latest version and the latest version satisfying a constraint
func (client *RegistryClient) GetBothLatestVersions(packageName, constraint, registryURL string, verbose bool, cache *shared.Cache) (string, string, error) {
	// Check cache first if enabled
	if cache != nil {
		key := shared.GenerateCacheKey(packageName, "npm", "", constraint)
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

	var packageInfo NpmPackageInfo
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&packageInfo); err != nil {
		return "", "", fmt.Errorf("failed to parse NPM response: %w", err)
	}

	// Get all non-deprecated versions
	versions := make([]string, 0, len(packageInfo.Versions))
	for version, versionInfo := range packageInfo.Versions {
		// Include only non-deprecated versions (deprecated field is null/missing for non-deprecated)
		if versionInfo.Deprecated == nil || versionInfo.Deprecated == "" {
			versions = append(versions, version)
		}
	}

	absoluteLatest, constraintLatest, err := shared.FindBothLatestVersions(versions, constraint)
	if err != nil {
		return "", "", err
	}

	// Cache the result if cache is enabled
	if cache != nil {
		entry := shared.CacheEntry{
			PackageName:      packageName,
			Type:             "npm",
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
	// Parse .npmrc configuration from both local and global files
	currentWorkingDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	npmrcPath := filepath.Join(currentWorkingDir, ".npmrc")
	npmrcConfig, err := parseNpmrcFiles(npmrcPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse .npmrc: %w", err)
	}

	var targetRegistryURL string
	if registryURL != "" {
		targetRegistryURL = registryURL
	} else {
		targetRegistryURL = getRegistryForPackage(packageName, npmrcConfig)
	}

	url := fmt.Sprintf("%s/%s", targetRegistryURL, packageName)

	if verbose {
		fmt.Printf("Checking NPM package: %s (registry: %s)\n", packageName, targetRegistryURL)
	}

	httpClient := &http.Client{Timeout: 10 * time.Second}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication if available for this registry
	if authToken := getAuthTokenForRegistry(targetRegistryURL, npmrcConfig); authToken != "" {
		request.Header.Set("Authorization", "Bearer "+authToken)
		if verbose {
			fmt.Printf("Using authentication for registry: %s\n", targetRegistryURL)
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
	return "npm"
}

// Ensure RegistryClient implements the interface
var _ shared.RegistryClient = (*RegistryClient)(nil)
