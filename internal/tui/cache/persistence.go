package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/repobird/repobird-cli/internal/models"
)

// CacheData represents the structure of persisted cache data
type CacheData struct {
	Runs          []models.RunResponse `json:"runs"`
	UserInfo      *models.UserInfo     `json:"userInfo"`
	FileHashes    map[string]string    `json:"fileHashes"`
	DashboardData *DashboardData       `json:"dashboardData"`
	SavedAt       time.Time            `json:"savedAt"`
}

// SaveToDisk persists cache to disk (called on quit)
func (c *SimpleCache) SaveToDisk() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cacheFile := GetCacheFilePath()

	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(cacheFile), 0700); err != nil {
		return err
	}

	// Gather all cached data
	data := CacheData{
		Runs:          c.GetRuns(),
		UserInfo:      c.GetUserInfo(),
		FileHashes:    c.gatherFileHashes(),
		DashboardData: c.getDashboardDataUnsafe(),
		SavedAt:       time.Now(),
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFile, jsonData, 0600)
}

// LoadFromDisk restores cache from disk (called on start)
func (c *SimpleCache) LoadFromDisk() error {
	cacheFile := GetCacheFilePath()

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, that's OK
			return nil
		}
		return err
	}

	var cacheData CacheData
	if err := json.Unmarshal(data, &cacheData); err != nil {
		return err
	}

	// Only restore if cache is less than 1 hour old
	if time.Since(cacheData.SavedAt) < time.Hour {
		c.mu.Lock()
		defer c.mu.Unlock()

		// Validate runs are not test data before loading
		if cacheData.Runs != nil {
			isValidData := true
			for _, run := range cacheData.Runs {
				if strings.HasPrefix(run.ID, "test-") || run.Repository == "" {
					isValidData = false
					break
				}
			}
			if isValidData {
				c.cache.Set("runs", cacheData.Runs, 5*time.Minute)
			}
		}
		if cacheData.UserInfo != nil {
			c.cache.Set("userInfo", cacheData.UserInfo, 10*time.Minute)
		}
		if cacheData.FileHashes != nil {
			for path, hash := range cacheData.FileHashes {
				key := "fileHash:" + path
				c.cache.Set(key, hash, 30*time.Minute)
			}
		}
		if cacheData.DashboardData != nil {
			c.cache.Set("dashboard", cacheData.DashboardData, 5*time.Minute)
		}
	}

	return nil
}

// gatherFileHashes collects all file hashes from cache (internal use, assumes lock held)
func (c *SimpleCache) gatherFileHashes() map[string]string {
	hashes := make(map[string]string)

	// Since ttlcache v3 doesn't expose all items directly,
	// we'll need to track file hashes separately or skip this for now
	// This is a simplified version

	return hashes
}

// getDashboardDataUnsafe gets dashboard data without locking (internal use)
func (c *SimpleCache) getDashboardDataUnsafe() *DashboardData {
	if item := c.cache.Get("dashboard"); item != nil {
		if data, ok := item.Value().(*DashboardData); ok {
			return data
		}
	}
	return nil
}

// GetCacheFilePath returns the path where cache is stored
func GetCacheFilePath() string {
	// Respect XDG_CONFIG_HOME environment variable for testing
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		configDir = xdg.ConfigHome
	}
	return filepath.Join(configDir, "repobird", "cache.json")
}
