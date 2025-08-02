package npm

import (
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
	// Parse .npmrc configuration from both local and global files
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	npmrcPath := filepath.Join(cwd, ".npmrc")
	npmrcConfig, err := parseNpmrcFiles(npmrcPath)
	if err != nil {
		return "", fmt.Errorf("failed to parse .npmrc: %w", err)
	}

	// Get the appropriate registry for this package
	registryURL := getRegistryForPackage(packageName, npmrcConfig)
	url := fmt.Sprintf("%s/%s", registryURL, packageName)

	if verbose {
		fmt.Printf("Checking NPM package: %s (registry: %s)\n", packageName, registryURL)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication if available for this registry
	if authToken := getAuthTokenForRegistry(registryURL, npmrcConfig); authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
		if verbose {
			fmt.Printf("Using authentication for registry: %s\n", registryURL)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch package info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("registry returned status %d for %s", resp.StatusCode, packageName)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
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

// GetFileType returns the file type this registry client handles
func (c *RegistryClient) GetFileType() string {
	return "npm"
}

// Ensure RegistryClient implements the interface
var _ shared.RegistryClient = (*RegistryClient)(nil)
