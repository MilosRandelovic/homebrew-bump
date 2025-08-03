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
func (c *RegistryClient) GetLatestVersion(packageName string, verbose bool) (string, error) {
	return c.GetLatestVersionFromRegistry(packageName, "", verbose)
}

// GetLatestVersionFromRegistry fetches the latest version from a specific registry
func (c *RegistryClient) GetLatestVersionFromRegistry(packageName, registryURL string, verbose bool) (string, error) {
	body, err := c.fetchPackageInfo(packageName, registryURL, verbose)
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
func (c *RegistryClient) GetBothLatestVersions(packageName, constraint string, verbose bool) (string, string, error) {
	body, err := c.fetchPackageInfo(packageName, "", verbose)
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
func (c *RegistryClient) fetchPackageInfo(packageName, registryURL string, verbose bool) ([]byte, error) {
	// Parse .npmrc configuration from both local and global files
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	npmrcPath := filepath.Join(cwd, ".npmrc")
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

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication if available for this registry
	if authToken := getAuthTokenForRegistry(targetRegistryURL, npmrcConfig); authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
		if verbose {
			fmt.Printf("Using authentication for registry: %s\n", targetRegistryURL)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch package info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned status %d for %s", resp.StatusCode, packageName)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return body, nil
}

// GetFileType returns the file type this registry client handles
func (c *RegistryClient) GetFileType() string {
	return "npm"
}

// Ensure RegistryClient implements the interface
var _ shared.RegistryClient = (*RegistryClient)(nil)
