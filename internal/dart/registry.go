package dart

import (
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

// GetLatestVersion fetches the latest version from pub.dev API or private registries
func (c *RegistryClient) GetLatestVersion(packageName string, verbose bool) (string, error) {
	return c.GetLatestVersionFromRegistry(packageName, "", verbose)
}

// GetLatestVersionFromRegistry fetches the latest version from a specific registry
func (c *RegistryClient) GetLatestVersionFromRegistry(packageName, registryURL string, verbose bool) (string, error) {
	// Parse pub configuration
	config, err := parsePubConfig()
	if err != nil {
		return "", fmt.Errorf("failed to parse pub config: %w", err)
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
		fmt.Printf("Checking Dart package: %s (registry: %s)\n", packageName, targetRegistry.URL)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication if available for this registry
	if targetRegistry.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+targetRegistry.AuthToken)
		if verbose {
			fmt.Printf("Using authentication for registry: %s\n", targetRegistry.URL)
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

	var packageInfo PubDevPackageInfo
	if err := json.Unmarshal(body, &packageInfo); err != nil {
		return "", fmt.Errorf("failed to parse pub.dev response: %w", err)
	}

	return packageInfo.Latest.Version, nil
}

// GetFileType returns the file type this registry client handles
func (c *RegistryClient) GetFileType() string {
	return "dart"
}

// Ensure RegistryClient implements the interface
var _ shared.RegistryClient = (*RegistryClient)(nil)
