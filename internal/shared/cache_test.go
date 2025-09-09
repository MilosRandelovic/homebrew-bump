package shared

import (
	"os"
	"sync"
	"testing"
	"time"
)

func getTestCachePath() string {
	return "/tmp/.bump-cache-test"
}

func TestCacheBasicOps(t *testing.T) {
	cachePath := getTestCachePath()
	os.Remove(cachePath)

	// Create cache without auto-loading
	cache := &Cache{
		entries:  make(map[string]CacheEntry),
		filePath: cachePath,
		mu:       sync.Mutex{},
	}

	entry := CacheEntry{
		PackageName:      "test-package",
		Type:             "npm",
		CurrentVersion:   "1.0.0",
		Constraint:       "^1.0.0",
		AbsoluteLatest:   "2.0.0",
		ConstraintLatest: "2.0.0",
		Expiry:           time.Now().Add(10 * time.Minute),
	}
	cache.Set(entry)

	key := generateCacheKey(entry.PackageName, entry.Type, entry.CurrentVersion, entry.Constraint)
	got, ok := cache.Get(key)
	if !ok {
		t.Fatalf("expected cache hit")
	}
	if got.AbsoluteLatest != "2.0.0" {
		t.Errorf("expected latest version 2.0.0, got %s", got.AbsoluteLatest)
	}

	cache.SaveEntries()
	cache2 := &Cache{
		entries:  make(map[string]CacheEntry),
		filePath: cachePath,
		mu:       sync.Mutex{},
	}
	cache2.LoadEntries()
	got2, ok2 := cache2.Get(key)
	if !ok2 || got2.AbsoluteLatest != "2.0.0" {
		t.Errorf("expected persisted cache hit")
	}

	os.Remove(cachePath)
}

func TestCacheRegistryDifferentiation(t *testing.T) {
	cachePath := getTestCachePath()
	os.Remove(cachePath)
	cache := &Cache{
		entries:  make(map[string]CacheEntry),
		filePath: cachePath,
		mu:       sync.Mutex{},
	}

	entryNpm := CacheEntry{
		PackageName:      "foo",
		Type:             "npm",
		CurrentVersion:   "1.0.0",
		Constraint:       "*",
		AbsoluteLatest:   "2.0.0",
		ConstraintLatest: "2.0.0",
		Expiry:           time.Now().Add(10 * time.Minute),
	}
	entryPub := CacheEntry{
		PackageName:      "foo",
		Type:             "pub",
		CurrentVersion:   "1.0.0",
		Constraint:       "*",
		AbsoluteLatest:   "3.0.0",
		ConstraintLatest: "3.0.0",
		Expiry:           time.Now().Add(10 * time.Minute),
	}
	cache.Set(entryNpm)
	cache.Set(entryPub)

	keyNpm := generateCacheKey(entryNpm.PackageName, entryNpm.Type, entryNpm.CurrentVersion, entryNpm.Constraint)
	keyPub := generateCacheKey(entryPub.PackageName, entryPub.Type, entryPub.CurrentVersion, entryPub.Constraint)

	if got, ok := cache.Get(keyNpm); !ok || got.AbsoluteLatest != "2.0.0" {
		t.Errorf("expected npm cache hit")
	}
	if got, ok := cache.Get(keyPub); !ok || got.AbsoluteLatest != "3.0.0" {
		t.Errorf("expected pub cache hit")
	}

	os.Remove(cachePath)
}

func TestCacheExpiry(t *testing.T) {
	cachePath := getTestCachePath()
	os.Remove(cachePath)
	cache := &Cache{
		entries:  make(map[string]CacheEntry),
		filePath: cachePath,
		mu:       sync.Mutex{},
	}

	entry := CacheEntry{
		PackageName:      "foo",
		Type:             "npm",
		CurrentVersion:   "1.0.0",
		Constraint:       "*",
		AbsoluteLatest:   "2.0.0",
		ConstraintLatest: "2.0.0",
		Expiry:           time.Now().Add(-1 * time.Minute), // expired
	}
	cache.Set(entry)

	key := generateCacheKey(entry.PackageName, entry.Type, entry.CurrentVersion, entry.Constraint)
	if _, ok := cache.Get(key); ok {
		t.Errorf("expected cache miss due to expiry")
	}

	os.Remove(cachePath)
}

func TestCacheClear(t *testing.T) {
	cachePath := getTestCachePath()
	os.Remove(cachePath)
	cache := &Cache{
		entries:  make(map[string]CacheEntry),
		filePath: cachePath,
		mu:       sync.Mutex{},
	}

	entry := CacheEntry{
		PackageName:      "foo",
		Type:             "npm",
		CurrentVersion:   "1.0.0",
		Constraint:       "*",
		AbsoluteLatest:   "2.0.0",
		ConstraintLatest: "2.0.0",
		Expiry:           time.Now().Add(10 * time.Minute),
	}
	cache.Set(entry)
	cache.Clear()
	if len(cache.entries) != 0 {
		t.Errorf("expected cache to be empty after clear")
	}

	os.Remove(cachePath)
}

func TestCacheExpiredCleanup(t *testing.T) {
	cachePath := getTestCachePath()
	os.Remove(cachePath)

	// Create cache without auto-loading
	cache := &Cache{
		entries:  make(map[string]CacheEntry),
		filePath: cachePath,
		mu:       sync.Mutex{},
	}

	// Use fixed timestamps to avoid timing issues
	now := time.Now()
	pastTime := now.Add(-24 * time.Hour)  // Clearly expired
	futureTime := now.Add(24 * time.Hour) // Clearly valid

	// Add both expired and valid entries
	expiredEntry := CacheEntry{
		PackageName:      "expired-pkg",
		Type:             "npm",
		CurrentVersion:   "1.0.0",
		Constraint:       "*",
		AbsoluteLatest:   "2.0.0",
		ConstraintLatest: "2.0.0",
		Expiry:           pastTime, // clearly expired
	}
	validEntry := CacheEntry{
		PackageName:      "valid-pkg",
		Type:             "npm",
		CurrentVersion:   "1.0.0",
		Constraint:       "*",
		AbsoluteLatest:   "3.0.0",
		ConstraintLatest: "3.0.0",
		Expiry:           futureTime, // clearly valid
	}

	cache.Set(expiredEntry)
	cache.Set(validEntry)

	// Should have 2 entries initially
	if len(cache.entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(cache.entries))
	}

	// Clean expired entries
	cache.CleanExpiredEntries()

	// Should have only 1 entry after cleanup
	if len(cache.entries) != 1 {
		t.Errorf("expected 1 entry after cleanup, got %d", len(cache.entries))
	}

	// Valid entry should still be accessible
	validKey := generateCacheKey("valid-pkg", "npm", "1.0.0", "*")
	if _, ok := cache.Get(validKey); !ok {
		t.Errorf("expected valid entry to still be accessible")
	}

	// Expired entry should not be accessible
	expiredKey := generateCacheKey("expired-pkg", "npm", "1.0.0", "*")
	if _, ok := cache.Get(expiredKey); ok {
		t.Errorf("expected expired entry to not be accessible")
	}

	os.Remove(cachePath)
}
