package llm

import (
	"path/filepath"
	"testing"
	"time"
)

func TestCacheHit(t *testing.T) {
	cache := NewResponseCache("")
	key := NormalizeKey("What is the capital of France?")
	cache.Set(key, "Paris")

	val, found := cache.Get(key)
	if !found {
		t.Error("Expected to find value in cache")
	}
	if val != "Paris" {
		t.Errorf("Expected 'Paris', got '%s'", val)
	}
}

func TestCacheMiss(t *testing.T) {
	cache := NewResponseCache("")
	key := NormalizeKey("Unknown question")
	_, found := cache.Get(key)
	if found {
		t.Error("Did not expect to find value in cache")
	}
}

func TestCachePersistence(t *testing.T) {
	tempDir := t.TempDir()
	cacheFile := filepath.Join(tempDir, "test.cache.json")

	// Create and save cache
	cache1 := NewResponseCache(cacheFile)
	key := NormalizeKey("Persist this")
	cache1.Set(key, "Persisted")
	if err := cache1.Save(); err != nil {
		t.Fatalf("Failed to save cache: %v", err)
	}

	// Load new cache
	cache2 := NewResponseCache(cacheFile)
	val, found := cache2.Get(key)
	if !found {
		t.Error("Expected to find value in loaded cache")
	}
	if val != "Persisted" {
		t.Errorf("Expected 'Persisted', got '%s'", val)
	}
}

func TestCacheTTLExpiry(t *testing.T) {
	cache := NewResponseCache("")
	key := NormalizeKey("Expire me")

	// Set entry with old timestamp
	cache.entries[key] = CacheEntry{
		Response:  "Old",
		Timestamp: time.Now().Add(-25 * time.Hour), // Older than 24h default TTL
	}

	_, found := cache.Get(key)
	if found {
		t.Error("Expected expired entry to not be found")
	}
	if _, stillExists := cache.entries[key]; stillExists {
		t.Error("Expected expired entry to be deleted from map")
	}
}

func TestNormalizeKey(t *testing.T) {
	key1 := NormalizeKey("Hello World")
	key2 := NormalizeKey("  hello world  ")
	key3 := NormalizeKey("HELLO WORLD")

	if key1 != key2 || key1 != key3 {
		t.Errorf("Expected identical hashes: %s, %s, %s", key1, key2, key3)
	}
}
