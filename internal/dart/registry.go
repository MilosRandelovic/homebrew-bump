package dart

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/MilosRandelovic/homebrew-bump/internal/shared"
)

// RegistryClient handles pub.dev registry operations
type RegistryClient struct{}

// PubDevPackageInfo represents the response from pub.dev API
type PubDevPackageInfo struct {
	Latest struct {
		Version string `json:"version"`
	} `json:"latest"`
}

// NewRegistryClient creates a new pub.dev registry client
func NewRegistryClient() *RegistryClient {
	return &RegistryClient{}
}

// GetLatestVersion fetches the latest version from pub.dev API
func (c *RegistryClient) GetLatestVersion(packageName string, verbose bool) (string, error) {
	url := fmt.Sprintf("https://pub.dev/api/packages/%s", packageName)

	if verbose {
		fmt.Printf("Checking pub.dev package: %s\n", packageName)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch package info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("pub.dev API returned status %d", resp.StatusCode)
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
