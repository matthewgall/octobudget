// Copyright 2025 Matthew Gall <me@matthewgall.dev>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CacheEntry represents a single cached item with expiration
type CacheEntry struct {
	Data      json.RawMessage `json:"data"`
	CachedAt  time.Time       `json:"cached_at"`
	ExpiresAt time.Time       `json:"expires_at"`
}

// CacheStore holds all cache entries for an account
type CacheStore struct {
	Entries map[string]*CacheEntry `json:"entries"`
}

// Cache provides simple JSON file-based caching with per-account isolation
type Cache struct {
	filePath  string
	accountID string
	store     *CacheStore
	mutex     sync.RWMutex
	logger    *Logger
}

// NewCache creates a new JSON file cache instance
func NewCache(basePath string, accountID string, logger *Logger) (*Cache, error) {
	cacheFile := filepath.Join(basePath, fmt.Sprintf("cache_%s.json", accountID))

	cache := &Cache{
		filePath:  cacheFile,
		accountID: accountID,
		store:     &CacheStore{Entries: make(map[string]*CacheEntry)},
		logger:    logger,
	}

	// Load existing cache from file
	if err := cache.load(); err != nil {
		if !os.IsNotExist(err) {
			logger.Warn("Failed to load cache, starting fresh", "error", err)
		}
	}

	// Clean expired entries on startup
	cache.cleanExpired()

	logger.Debug("Cache initialized", "path", cacheFile, "account", accountID, "entries", len(cache.store.Entries))

	return cache, nil
}

// Set stores a value in cache with TTL (time-to-live)
func (c *Cache) Set(key string, value interface{}, ttl time.Duration) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Serialize value to JSON
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal cache value: %w", err)
	}

	c.store.Entries[key] = &CacheEntry{
		Data:      valueJSON,
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(ttl),
	}

	// Persist to disk
	if err := c.save(); err != nil {
		return err
	}

	c.logger.Debug("Cache set", "account", c.accountID, "key", key, "ttl", ttl)
	return nil
}

// Get retrieves a value from cache if it exists and hasn't expired
func (c *Cache) Get(key string, target interface{}) (bool, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.store.Entries[key]
	if !exists {
		c.logger.Debug("Cache miss", "account", c.accountID, "key", key)
		return false, nil // Cache miss
	}

	// Check if cache has expired
	if time.Now().After(entry.ExpiresAt) {
		c.logger.Debug("Cache expired", "account", c.accountID, "key", key)
		return false, nil
	}

	// Deserialize JSON into target
	if err := json.Unmarshal(entry.Data, target); err != nil {
		return false, fmt.Errorf("failed to unmarshal cache value: %w", err)
	}

	c.logger.Debug("Cache hit", "account", c.accountID, "key", key, "expires_in", time.Until(entry.ExpiresAt).Round(time.Second))
	return true, nil
}

// Delete removes a cache entry
func (c *Cache) Delete(key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.store.Entries, key)
	return c.save()
}

// CleanExpired removes all expired cache entries
func (c *Cache) CleanExpired() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.cleanExpired()
}

// cleanExpired removes expired entries (must be called with lock held)
func (c *Cache) cleanExpired() error {
	now := time.Now()
	removed := 0

	for key, entry := range c.store.Entries {
		if now.After(entry.ExpiresAt) {
			delete(c.store.Entries, key)
			removed++
		}
	}

	if removed > 0 {
		c.logger.Info("Cleaned expired cache entries", "count", removed)
		return c.save()
	}

	return nil
}

// Clear removes all cache entries for the current account
func (c *Cache) Clear() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	count := len(c.store.Entries)
	c.store.Entries = make(map[string]*CacheEntry)

	if err := c.save(); err != nil {
		return err
	}

	c.logger.Info("Cleared account cache", "account", c.accountID, "count", count)
	return nil
}

// Stats returns cache statistics for the current account
func (c *Cache) Stats() (total int, expired int, err error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	now := time.Now()
	total = len(c.store.Entries)

	for _, entry := range c.store.Entries {
		if now.After(entry.ExpiresAt) {
			expired++
		}
	}

	return total, expired, nil
}

// load reads the cache from disk
func (c *Cache) load() error {
	data, err := os.ReadFile(c.filePath)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, c.store); err != nil {
		return fmt.Errorf("failed to unmarshal cache file: %w", err)
	}

	return nil
}

// save writes the cache to disk
func (c *Cache) save() error {
	data, err := json.MarshalIndent(c.store, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	if err := os.WriteFile(c.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// Close closes the cache (no-op for JSON file cache, but kept for interface compatibility)
func (c *Cache) Close() error {
	// Final save and cleanup
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.cleanExpired()
}
