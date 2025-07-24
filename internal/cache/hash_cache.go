// Package cache provides caching mechanisms for syncstation operations
package cache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// HashCacheEntry represents a cached hash entry
type HashCacheEntry struct {
	Hash     string    `json:"hash"`
	Size     int64     `json:"size"`
	ModTime  time.Time `json:"modTime"`
	CachedAt time.Time `json:"cachedAt"`
}

// HashCache provides thread-safe caching of file hashes
type HashCache struct {
	entries map[string]*HashCacheEntry
	mutex   sync.RWMutex
	maxAge  time.Duration
}

// NewHashCache creates a new hash cache with the specified maximum age
func NewHashCache(maxAge time.Duration) *HashCache {
	return &HashCache{
		entries: make(map[string]*HashCacheEntry),
		maxAge:  maxAge,
	}
}

// Get retrieves a cached hash if it's still valid
func (hc *HashCache) Get(filePath string) (string, bool) {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()

	entry, exists := hc.entries[filePath]
	if !exists {
		return "", false
	}

	// Check if cache entry is still valid
	if time.Since(entry.CachedAt) > hc.maxAge {
		return "", false
	}

	// Check if file has been modified since cache
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return "", false
	}

	// If size or mod time changed, cache is invalid
	if fileInfo.Size() != entry.Size || !fileInfo.ModTime().Equal(entry.ModTime) {
		return "", false
	}

	return entry.Hash, true
}

// Set stores a hash in the cache
func (hc *HashCache) Set(filePath, hash string, size int64, modTime time.Time) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	hc.entries[filePath] = &HashCacheEntry{
		Hash:     hash,
		Size:     size,
		ModTime:  modTime,
		CachedAt: time.Now(),
	}
}

// GetOrCalculate retrieves a cached hash or calculates it if not cached or invalid
func (hc *HashCache) GetOrCalculate(filePath string) (string, error) {
	// Try to get from cache first
	if cachedHash, found := hc.Get(filePath); found {
		return cachedHash, nil
	}

	// Calculate hash since it's not cached or invalid
	hash, err := hc.calculateFileHash(filePath)
	if err != nil {
		return "", err
	}

	// Get file info to cache along with hash
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return "", err
	}

	// Cache the result
	hc.Set(filePath, hash, fileInfo.Size(), fileInfo.ModTime())

	return hash, nil
}

// calculateFileHash calculates SHA256 hash of a file
func (hc *HashCache) calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("sha256:%x", hash.Sum(nil)), nil
}

// Clear removes all entries from the cache
func (hc *HashCache) Clear() {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	hc.entries = make(map[string]*HashCacheEntry)
}

// Remove removes a specific entry from the cache
func (hc *HashCache) Remove(filePath string) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	delete(hc.entries, filePath)
}

// Size returns the number of entries in the cache
func (hc *HashCache) Size() int {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()

	return len(hc.entries)
}

// CleanExpired removes expired entries from the cache
func (hc *HashCache) CleanExpired() int {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	now := time.Now()
	removed := 0

	for filePath, entry := range hc.entries {
		if now.Sub(entry.CachedAt) > hc.maxAge {
			delete(hc.entries, filePath)
			removed++
		}
	}

	return removed
}

// SaveToFile saves the cache to a file
func (hc *HashCache) SaveToFile(filePath string) error {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Marshal cache entries to JSON
	data, err := json.MarshalIndent(hc.entries, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(filePath, data, 0644)
}

// LoadFromFile loads the cache from a file
func (hc *HashCache) LoadFromFile(filePath string) error {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil // No cache file is not an error
	}

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Unmarshal JSON
	var entries map[string]*HashCacheEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return err
	}

	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	// Replace entries
	hc.entries = entries
	if hc.entries == nil {
		hc.entries = make(map[string]*HashCacheEntry)
	}

	return nil
}

// Stats returns cache statistics
type CacheStats struct {
	TotalEntries   int           `json:"totalEntries"`
	ExpiredEntries int           `json:"expiredEntries"`
	MaxAge         time.Duration `json:"maxAge"`
	OldestEntry    time.Time     `json:"oldestEntry"`
	NewestEntry    time.Time     `json:"newestEntry"`
}

func (hc *HashCache) Stats() CacheStats {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()

	stats := CacheStats{
		TotalEntries: len(hc.entries),
		MaxAge:       hc.maxAge,
	}

	now := time.Now()
	for _, entry := range hc.entries {
		if now.Sub(entry.CachedAt) > hc.maxAge {
			stats.ExpiredEntries++
		}

		if stats.OldestEntry.IsZero() || entry.CachedAt.Before(stats.OldestEntry) {
			stats.OldestEntry = entry.CachedAt
		}

		if stats.NewestEntry.IsZero() || entry.CachedAt.After(stats.NewestEntry) {
			stats.NewestEntry = entry.CachedAt
		}
	}

	return stats
}

// Invalidate marks entries as invalid if their files have changed
func (hc *HashCache) Invalidate(filePaths []string) int {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	invalidated := 0
	for _, filePath := range filePaths {
		if entry, exists := hc.entries[filePath]; exists {
			// Check if file still exists and has same attributes
			fileInfo, err := os.Stat(filePath)
			if err != nil || fileInfo.Size() != entry.Size || !fileInfo.ModTime().Equal(entry.ModTime) {
				delete(hc.entries, filePath)
				invalidated++
			}
		}
	}

	return invalidated
}

// GetKeys returns all file paths currently in the cache
func (hc *HashCache) GetKeys() []string {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()

	keys := make([]string, 0, len(hc.entries))
	for key := range hc.entries {
		keys = append(keys, key)
	}

	return keys
}

// Default cache instance with 1 hour max age
var defaultCache = NewHashCache(time.Hour)

// Global functions that use the default cache

// GetCachedHash retrieves a hash from the default cache
func GetCachedHash(filePath string) (string, bool) {
	return defaultCache.Get(filePath)
}

// SetCachedHash stores a hash in the default cache
func SetCachedHash(filePath, hash string, size int64, modTime time.Time) {
	defaultCache.Set(filePath, hash, size, modTime)
}

// GetOrCalculateHash retrieves or calculates a hash using the default cache
func GetOrCalculateHash(filePath string) (string, error) {
	return defaultCache.GetOrCalculate(filePath)
}

// ClearCache clears the default cache
func ClearCache() {
	defaultCache.Clear()
}

// SaveDefaultCache saves the default cache to a file
func SaveDefaultCache(filePath string) error {
	return defaultCache.SaveToFile(filePath)
}

// LoadDefaultCache loads the default cache from a file
func LoadDefaultCache(filePath string) error {
	return defaultCache.LoadFromFile(filePath)
}

// GetDefaultCacheStats returns statistics for the default cache
func GetDefaultCacheStats() CacheStats {
	return defaultCache.Stats()
}