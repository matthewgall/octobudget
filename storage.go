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
	"time"
)

// Storage handles persistent storage of data
type Storage struct {
	basePath string
	cache    *Cache
	logger   *Logger
}

// NewStorage creates a new storage handler with caching
func NewStorage(basePath string, accountID string, logger *Logger) (*Storage, error) {
	// Ensure storage directory exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, &StorageError{
			Operation: "create_directory",
			Path:      basePath,
			Err:       err,
		}
	}

	// Initialize cache
	cache, err := NewCache(basePath, accountID, logger)
	if err != nil {
		return nil, &StorageError{
			Operation: "initialize_cache",
			Path:      basePath,
			Err:       err,
		}
	}

	// Clean expired cache entries on startup
	if err := cache.CleanExpired(); err != nil {
		logger.Warn("Failed to clean expired cache", "error", err)
	}

	logger.Debug("Storage initialized", "path", basePath)

	return &Storage{
		basePath: basePath,
		cache:    cache,
		logger:   logger,
	}, nil
}

// SaveAnalysisResult saves an analysis result
func (s *Storage) SaveAnalysisResult(result *AnalysisResult, accountID string) error {
	filename := fmt.Sprintf("%s_analysis_%s.json", accountID, result.GeneratedAt.Format("2006-01-02_15-04-05"))
	path := filepath.Join(s.basePath, filename)

	s.logger.LogStorageOperation("save_analysis", path)

	return s.saveJSON(path, result)
}

// LoadLatestAnalysis loads the most recent analysis result for the given account
func (s *Storage) LoadLatestAnalysis(accountID string) (*AnalysisResult, error) {
	pattern := filepath.Join(s.basePath, fmt.Sprintf("%s_analysis_*.json", accountID))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, &StorageError{
			Operation: "glob_analysis",
			Path:      pattern,
			Err:       err,
		}
	}

	if len(matches) == 0 {
		return nil, nil // No previous analysis found
	}

	// Get the most recent file (files are sorted by date in filename)
	latestFile := matches[len(matches)-1]

	s.logger.LogStorageOperation("load_latest_analysis", latestFile)

	var result AnalysisResult
	if err := s.loadJSON(latestFile, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// saveJSON saves data as JSON to a file
func (s *Storage) saveJSON(path string, data interface{}) error {
	file, err := os.Create(path)
	if err != nil {
		return &StorageError{
			Operation: "create_file",
			Path:      path,
			Err:       err,
		}
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(data); err != nil {
		return &StorageError{
			Operation: "encode_json",
			Path:      path,
			Err:       err,
		}
	}

	return nil
}

// loadJSON loads data from a JSON file
func (s *Storage) loadJSON(path string, target interface{}) error {
	file, err := os.Open(path)
	if err != nil {
		return &StorageError{
			Operation: "open_file",
			Path:      path,
			Err:       err,
		}
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(target); err != nil {
		return &StorageError{
			Operation: "decode_json",
			Path:      path,
			Err:       err,
		}
	}

	return nil
}

// ListStoredFiles lists all files in the storage directory
func (s *Storage) ListStoredFiles() ([]string, error) {
	entries, err := os.ReadDir(s.basePath)
	if err != nil {
		return nil, &StorageError{
			Operation: "list_directory",
			Path:      s.basePath,
			Err:       err,
		}
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}

	return files, nil
}

// SaveCache saves data to cache with a TTL (time-to-live)
func (s *Storage) SaveCache(key string, data interface{}, ttl time.Duration) error {
	return s.cache.Set(key, data, ttl)
}

// LoadCache loads data from cache if it exists and hasn't expired
func (s *Storage) LoadCache(key string, target interface{}) (bool, error) {
	return s.cache.Get(key, target)
}

// ClearCache clears all cache entries for the current account
func (s *Storage) ClearCache() error {
	return s.cache.Clear()
}

// CacheStats returns cache statistics for the current account
func (s *Storage) CacheStats() (total int, expired int, err error) {
	return s.cache.Stats()
}

// Close closes all storage resources
func (s *Storage) Close() error {
	if s.cache != nil {
		return s.cache.Close()
	}
	return nil
}
