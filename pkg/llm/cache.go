package llm

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CacheEntry represents a single cached response.
type CacheEntry struct {
	Response  string    `json:"response"`
	Timestamp time.Time `json:"timestamp"`
}

// ResponseCache provides an in-memory cache with optional file persistence.
type ResponseCache struct {
	cacheFile string
	entries   map[string]CacheEntry
	ttl       time.Duration
}

// NewResponseCache initializes a new response cache.
// If cacheFile is provided, it tries to load existing entries.
func NewResponseCache(cacheFile string) *ResponseCache {
	c := &ResponseCache{
		cacheFile: cacheFile,
		entries:   make(map[string]CacheEntry),
		ttl:       24 * time.Hour,
	}
	_ = c.load()
	return c
}

// load reads the cache file from disk.
func (c *ResponseCache) load() error {
	if c.cacheFile == "" {
		return nil
	}
	// #nosec G304 - cacheFile is trusted internal path
	data, err := os.ReadFile(c.cacheFile)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &c.entries)
}

// Save persists the cache to disk.
func (c *ResponseCache) Save() error {
	if c.cacheFile == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(c.cacheFile), 0o750); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}
	data, err := json.MarshalIndent(c.entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}
	return os.WriteFile(c.cacheFile, data, 0o600)
}

// Get retrieves a response from the cache if it exists and hasn't expired.
func (c *ResponseCache) Get(key string) (string, bool) {
	entry, exists := c.entries[key]
	if !exists {
		return "", false
	}
	if time.Since(entry.Timestamp) > c.ttl {
		delete(c.entries, key)
		return "", false
	}
	return entry.Response, true
}

// Set stores a response in the cache.
func (c *ResponseCache) Set(key, response string) {
	c.entries[key] = CacheEntry{
		Response:  response,
		Timestamp: time.Now(),
	}
}

// NormalizeKey normalizes and hashes the user question for exact-match caching.
func NormalizeKey(question string) string {
	normalized := strings.ToLower(strings.TrimSpace(question))
	hash := sha256.Sum256([]byte(normalized))
	return fmt.Sprintf("%x", hash)
}
