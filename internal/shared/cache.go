package shared

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type CacheEntry struct {
	PackageName      string
	Type             string
	CurrentVersion   string
	Constraint       string
	AbsoluteLatest   string
	ConstraintLatest string
	Expiry           time.Time
}

type Cache struct {
	entries  map[string]CacheEntry
	filePath string
	mu       sync.Mutex
}

func NewCache() *Cache {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Warning: Could not determine home directory: %v\n", err)
		return nil
	}
	filePath := filepath.Join(homeDir, ".bump-cache")

	cache := &Cache{
		entries:  make(map[string]CacheEntry),
		filePath: filePath,
	}

	// Auto-load entries on creation
	cache.LoadEntries()

	return cache
}

func generateCacheKey(pkg, packageType, current, constraint string) string {
	return fmt.Sprintf("%s|%s|%s|%s", pkg, packageType, current, constraint)
}

// GenerateCacheKey is the exported version of generateCacheKey
func GenerateCacheKey(pkg, packageType, current, constraint string) string {
	return generateCacheKey(pkg, packageType, current, constraint)
}

func (c *Cache) LoadEntries() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	file, err := os.Open(c.filePath)
	if err != nil {
		// Only warn if it's not a "file not found" error (which is expected for new cache)
		if !os.IsNotExist(err) {
			fmt.Printf("Warning: Could not open cache file: %v\n", err)
		}
		return nil // treat as empty cache
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	entries := make(map[string]CacheEntry)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) != 7 {
			continue // skip malformed lines
		}

		expiry, err := time.Parse(time.RFC3339, parts[6])
		if err != nil {
			continue // skip entries with invalid expiry
		}

		entry := CacheEntry{
			PackageName:      parts[0],
			Type:             parts[1],
			CurrentVersion:   parts[2],
			Constraint:       parts[3],
			AbsoluteLatest:   parts[4],
			ConstraintLatest: parts[5],
			Expiry:           expiry,
		}

		key := generateCacheKey(entry.PackageName, entry.Type, entry.CurrentVersion, entry.Constraint)
		entries[key] = entry
	}

	c.entries = entries

	return nil
}

func (c *Cache) SaveEntries() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	file, err := os.Create(c.filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, entry := range c.entries {
		line := strings.Join([]string{
			entry.PackageName,
			entry.Type,
			entry.CurrentVersion,
			entry.Constraint,
			entry.AbsoluteLatest,
			entry.ConstraintLatest,
			entry.Expiry.Format(time.RFC3339),
		}, "|")

		if _, err := file.WriteString(line + "\n"); err != nil {
			return err
		}
	}

	return nil
}

func (c *Cache) Get(key string) (CacheEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok {
		return CacheEntry{}, false
	}
	if time.Now().After(entry.Expiry) {
		delete(c.entries, key)
		return CacheEntry{}, false
	}
	return entry, true
}

func (c *Cache) Set(entry CacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := generateCacheKey(entry.PackageName, entry.Type, entry.CurrentVersion, entry.Constraint)
	c.entries[key] = entry
}

func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]CacheEntry)
}

// CleanExpiredEntries removes all expired entries from the cache
func (c *Cache) CleanExpiredEntries() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.Expiry) {
			delete(c.entries, key)
		}
	}
}
