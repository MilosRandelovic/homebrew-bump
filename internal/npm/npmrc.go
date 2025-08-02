package npm

import (
	"bufio"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
)

// NpmrcConfig holds the parsed .npmrc configuration
type NpmrcConfig struct {
	ScopeRegistries map[string]string // maps scope to registry URL
	AuthTokens      map[string]string // maps registry to auth token
}

// parseNpmrcFiles parses both local and global .npmrc files and merges their configurations
// Local .npmrc takes precedence for scope registries, global .npmrc provides auth tokens
func parseNpmrcFiles(localPath string) (*NpmrcConfig, error) {
	config := &NpmrcConfig{
		ScopeRegistries: make(map[string]string),
		AuthTokens:      make(map[string]string),
	}

	// Parse global .npmrc first (from home directory)
	homeDir, err := os.UserHomeDir()
	if err == nil {
		globalNpmrcPath := filepath.Join(homeDir, ".npmrc")
		globalConfig, err := parseNpmrcFile(globalNpmrcPath)
		if err == nil {
			// Copy global config
			maps.Copy(config.ScopeRegistries, globalConfig.ScopeRegistries)
			maps.Copy(config.AuthTokens, globalConfig.AuthTokens)
		}
	}

	// Parse local .npmrc (overrides global for scope registries)
	localConfig, err := parseNpmrcFile(localPath)
	if err != nil {
		// If local file doesn't exist, that's okay, we still have global config
		if !os.IsNotExist(err) {
			return nil, err
		}
	} else {
		// Local scope registries override global ones
		maps.Copy(config.ScopeRegistries, localConfig.ScopeRegistries)
		// Local auth tokens override global ones
		maps.Copy(config.AuthTokens, localConfig.AuthTokens)
	}

	return config, nil
}

func parseNpmrcFile(filePath string) (*NpmrcConfig, error) {
	config := &NpmrcConfig{
		ScopeRegistries: make(map[string]string),
		AuthTokens:      make(map[string]string),
	}

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty config if file doesn't exist
			return config, nil
		}
		return nil, fmt.Errorf("failed to open .npmrc file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// Parse scope registry: @scope:registry=https://registry.example.com
		if strings.Contains(line, ":registry=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				registry := strings.TrimSpace(parts[1])

				// Extract scope from @scope:registry
				if strings.HasSuffix(key, ":registry") {
					scope := strings.TrimSuffix(key, ":registry")
					config.ScopeRegistries[scope] = registry
				}
			}
		}

		// Parse auth tokens: //registry.example.com/:_authToken=token
		if strings.Contains(line, ":_authToken=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				token := strings.Trim(strings.TrimSpace(parts[1]), "\"'")

				// Extract registry from //registry.example.com/:_authToken
				if strings.HasSuffix(key, ":_authToken") {
					registry := strings.TrimSuffix(key, ":_authToken")
					registry = strings.TrimPrefix(registry, "//")
					registry = strings.TrimSuffix(registry, "/")
					config.AuthTokens[registry] = token
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading .npmrc file: %w", err)
	}

	return config, nil
}

// getRegistryForPackage determines the appropriate registry URL for a package
func getRegistryForPackage(packageName string, npmrcConfig *NpmrcConfig) string {
	// Check if it's a scoped package
	if strings.HasPrefix(packageName, "@") {
		if idx := strings.Index(packageName[1:], "/"); idx != -1 {
			scope := packageName[:idx+1] // Include @ but not the /
			if registry, exists := npmrcConfig.ScopeRegistries[scope]; exists {
				return registry
			}
		}
	}

	// Default to public npm registry
	return "https://registry.npmjs.org"
}

// getAuthTokenForRegistry finds the appropriate auth token for a registry URL
func getAuthTokenForRegistry(registryURL string, npmrcConfig *NpmrcConfig) string {
	// Extract hostname from registry URL for matching
	if after, ok := strings.CutPrefix(registryURL, "https://"); ok {
		hostname := after
		hostname = strings.TrimSuffix(hostname, "/")

		if token, exists := npmrcConfig.AuthTokens[hostname]; exists {
			return token
		}
	}

	return ""
}
