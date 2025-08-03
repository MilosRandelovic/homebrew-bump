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
		Version string `json:"version"`
	} `json:"versions"`
}

// NewRegistryClient creates a new NPM registry client
func NewRegistryClient() *RegistryClient {
	return &RegistryClient{}
}

// GetLatestVersion fetches the latest version from NPM registry
func (client *RegistryClient) GetLatestVersion(packageName string, verbose bool) (string, error) {
	return client.GetLatestVersionFromRegistry(packageName, "", verbose)
}

// GetLatestVersionFromRegistry fetches the latest version from a specific registry
func (client *RegistryClient) GetLatestVersionFromRegistry(packageName, registryURL string, verbose bool) (string, error) {
	body, err := client.fetchPackageInfo(packageName, registryURL, verbose)
	if err != nil {
		return "", err
	}

	var packageInfo NpmPackageInfo
	if err := json.Unmarshal(body, &packageInfo); err != nil {
		return "", fmt.Errorf("failed to parse NPM response: %w", err)
	}

	if latest, ok := packageInfo.DistTags["latest"]; ok {
		return latest, nil
	}

	return "", fmt.Errorf("no latest version found for %s", packageName)
}

// GetBothLatestVersions fetches both the absolute latest version and the latest version satisfying a constraint
func (client *RegistryClient) GetBothLatestVersions(packageName, constraint string, verbose bool) (string, string, error) {
	body, err := client.fetchPackageInfo(packageName, "", verbose)
	if err != nil {
		return "", "", err
	}

	var packageInfo NpmPackageInfo
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&packageInfo); err != nil {
		return "", "", fmt.Errorf("failed to parse NPM response: %w", err)
	}

	// Get all versions
	versions := make([]string, 0, len(packageInfo.Versions))
	for version := range packageInfo.Versions {
		versions = append(versions, version)
	}

	return shared.FindBothLatestVersions(versions, constraint)
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
