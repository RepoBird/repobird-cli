package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/repobird/repobird-cli/internal/models"
)

// FileHashCache manages the cache of file hashes for duplicate detection
type FileHashCache struct {
	mu        sync.RWMutex
	hashes    map[string]bool // Map of file hash to existence
	loaded    bool
	loadedAt  time.Time
	cacheFile string
	userID    *int        // User ID for user-specific caching
	apiClient interface{} // Will be set to the API client interface
}

// FileHashCacheData represents the persistent cache structure
type FileHashCacheData struct {
	Hashes    map[string]bool `json:"hashes"`
	LoadedAt  time.Time       `json:"loaded_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// NewFileHashCache creates a new file hash cache instance
func NewFileHashCache() *FileHashCache {
	return NewFileHashCacheForUser(nil)
}

// NewFileHashCacheForUser creates a new file hash cache instance for a specific user
func NewFileHashCacheForUser(userID *int) *FileHashCache {
	var cacheFile string

	// Use os.UserCacheDir for cross-platform compatibility
	baseDir, err := os.UserCacheDir()
	if err != nil {
		// Fallback to home directory if cache dir fails
		homeDir, _ := os.UserHomeDir()
		baseDir = filepath.Join(homeDir, ".cache")
	}

	if userID != nil {
		// User-specific cache directory
		cacheDir := filepath.Join(baseDir, "repobird", "users", fmt.Sprintf("user-%d", *userID))
		_ = os.MkdirAll(cacheDir, 0755)
		cacheFile = filepath.Join(cacheDir, "file_hashes.json")
	} else {
		// Fallback to shared cache directory
		cacheDir := filepath.Join(baseDir, "repobird", "shared")
		_ = os.MkdirAll(cacheDir, 0755)
		cacheFile = filepath.Join(cacheDir, "file_hashes.json")
	}

	return &FileHashCache{
		hashes:    make(map[string]bool),
		cacheFile: cacheFile,
		userID:    userID,
	}
}

// SetUserID updates the cache to use a user-specific directory
func (c *FileHashCache) SetUserID(userID *int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if userID == nil || c.userID != nil && *c.userID == *userID {
		return // No change needed
	}

	// Update to use user-specific cache directory
	baseDir, err := os.UserCacheDir()
	if err != nil {
		homeDir, _ := os.UserHomeDir()
		baseDir = filepath.Join(homeDir, ".cache")
	}

	cacheDir := filepath.Join(baseDir, "repobird", "users", fmt.Sprintf("user-%d", *userID))
	_ = os.MkdirAll(cacheDir, 0755)
	c.cacheFile = filepath.Join(cacheDir, "file_hashes.json")
	c.userID = userID

	// Reset loaded state to force reload from new location
	c.loaded = false
	c.hashes = make(map[string]bool)
}

// SetAPIClient sets the API client for fetching hashes from the server
func (c *FileHashCache) SetAPIClient(client interface{}) {
	c.apiClient = client
}

// LoadFromFile loads the cache from the persistent file
func (c *FileHashCache) LoadFromFile() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := os.ReadFile(c.cacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Cache file doesn't exist yet, that's ok
			return nil
		}
		return fmt.Errorf("failed to read cache file: %w", err)
	}

	var cacheData FileHashCacheData
	if err := json.Unmarshal(data, &cacheData); err != nil {
		return fmt.Errorf("failed to unmarshal cache data: %w", err)
	}

	c.hashes = cacheData.Hashes
	c.loadedAt = cacheData.LoadedAt
	c.loaded = true

	return nil
}

// SaveToFile saves the cache to the persistent file
func (c *FileHashCache) SaveToFile() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cacheData := FileHashCacheData{
		Hashes:    c.hashes,
		LoadedAt:  c.loadedAt,
		UpdatedAt: time.Now(),
	}

	data, err := json.MarshalIndent(cacheData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache data: %w", err)
	}

	if err := os.WriteFile(c.cacheFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// FetchFromAPI fetches all file hashes from the API and updates the cache
func (c *FileHashCache) FetchFromAPI(ctx context.Context, apiClient interface {
	GetFileHashes(context.Context) ([]models.FileHashEntry, error)
}) error {
	hashes, err := apiClient.GetFileHashes(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch file hashes from API: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Clear and rebuild the hash map
	c.hashes = make(map[string]bool)
	for _, entry := range hashes {
		if entry.FileHash != "" {
			c.hashes[entry.FileHash] = true
		}
	}

	c.loaded = true
	c.loadedAt = time.Now()

	// Save to file after fetching
	go func() {
		_ = c.SaveToFile()
	}()

	return nil
}

// EnsureLoaded ensures the cache is loaded, fetching from API if necessary
func (c *FileHashCache) EnsureLoaded(ctx context.Context, apiClient interface {
	GetFileHashes(context.Context) ([]models.FileHashEntry, error)
}) error {
	c.mu.RLock()
	if c.loaded {
		c.mu.RUnlock()
		return nil
	}
	c.mu.RUnlock()

	// Try to load from file first
	if err := c.LoadFromFile(); err == nil && c.loaded {
		return nil
	}

	// If not loaded from file or file doesn't exist, fetch from API
	return c.FetchFromAPI(ctx, apiClient)
}

// AddHash adds a new hash to the cache
func (c *FileHashCache) AddHash(hash string) {
	if hash == "" {
		return
	}

	c.mu.Lock()
	c.hashes[hash] = true
	c.mu.Unlock()

	// Save to file asynchronously
	go func() {
		_ = c.SaveToFile()
	}()
}

// HasHash checks if a hash exists in the cache
func (c *FileHashCache) HasHash(hash string) bool {
	if hash == "" {
		return false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hashes[hash]
}

// IsLoaded returns whether the cache has been loaded
func (c *FileHashCache) IsLoaded() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.loaded
}

// CalculateFileHash calculates the SHA-256 hash of a file's contents
// Works with any file type by hashing the raw file contents
func CalculateFileHash(filepath string) (string, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Calculate SHA-256 hash of raw file contents
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// CalculateConfigHash calculates the SHA-256 hash of a RunConfig
func CalculateConfigHash(config *models.RunConfig) (string, error) {
	if config == nil {
		return "", nil
	}

	// Create a normalized version for hashing (exclude volatile fields)
	normalized := map[string]interface{}{
		"prompt":     config.Prompt,
		"repository": config.Repository,
		"source":     config.Source,
		"target":     config.Target,
		"runType":    config.RunType,
		"title":      config.Title,
		"context":    config.Context,
		"files":      config.Files,
	}

	// Marshal to get normalized JSON
	data, err := json.Marshal(normalized)
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %w", err)
	}

	// Calculate SHA-256 hash
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// CalculateStringHash calculates the SHA-256 hash of a string
func CalculateStringHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// Set stores a hash in the cache (for tracking generated hashes)
func (c *FileHashCache) Set(key string, hash string) {
	if hash == "" {
		return
	}
	c.AddHash(hash)
}
