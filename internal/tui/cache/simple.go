package cache

import (
	"sync"
	"time"

	"github.com/jellydator/ttlcache/v3"
	"github.com/repobird/repobird-cli/internal/models"
)

// SimpleCache wraps ttlcache for RepoBird's needs
type SimpleCache struct {
	cache *ttlcache.Cache[string, any]
	mu    sync.RWMutex // Additional safety for concurrent access
}

// NewSimpleCache creates a cache with sensible defaults
func NewSimpleCache() *SimpleCache {
	cache := ttlcache.New[string, any](
		ttlcache.WithTTL[string, any](5*time.Minute),
		ttlcache.WithCapacity[string, any](10000),
	)

	// Start automatic cleanup
	go cache.Start()

	return &SimpleCache{cache: cache}
}

// GetRuns retrieves cached runs
func (c *SimpleCache) GetRuns() []models.RunResponse {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if item := c.cache.Get("runs"); item != nil {
		if runs, ok := item.Value().([]models.RunResponse); ok {
			return runs
		}
	}
	return nil
}

// SetRuns caches runs with TTL
func (c *SimpleCache) SetRuns(runs []models.RunResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache.Set("runs", runs, ttlcache.DefaultTTL)
}

// GetRun retrieves a single cached run by ID
func (c *SimpleCache) GetRun(id string) *models.RunResponse {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := "run:" + id
	if item := c.cache.Get(key); item != nil {
		if run, ok := item.Value().(models.RunResponse); ok {
			return &run
		}
	}
	return nil
}

// SetRun caches a single run
func (c *SimpleCache) SetRun(run models.RunResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := "run:" + run.ID
	c.cache.Set(key, run, ttlcache.DefaultTTL)
}

// GetUserInfo retrieves cached user info
func (c *SimpleCache) GetUserInfo() *models.UserInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if item := c.cache.Get("userInfo"); item != nil {
		if info, ok := item.Value().(*models.UserInfo); ok {
			return info
		}
	}
	return nil
}

// SetUserInfo caches user info
func (c *SimpleCache) SetUserInfo(info *models.UserInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache.Set("userInfo", info, 10*time.Minute)
}

// GetFileHash retrieves cached file hash
func (c *SimpleCache) GetFileHash(path string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := "fileHash:" + path
	if item := c.cache.Get(key); item != nil {
		if hash, ok := item.Value().(string); ok {
			return hash
		}
	}
	return ""
}

// SetFileHash caches file hash
func (c *SimpleCache) SetFileHash(path string, hash string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := "fileHash:" + path
	c.cache.Set(key, hash, 30*time.Minute)
}

// GetDashboardCache retrieves cached dashboard data
func (c *SimpleCache) GetDashboardCache() (*DashboardData, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if item := c.cache.Get("dashboard"); item != nil {
		if data, ok := item.Value().(*DashboardData); ok {
			return data, true
		}
	}
	return nil, false
}

// SetDashboardCache stores dashboard data
func (c *SimpleCache) SetDashboardCache(data *DashboardData) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache.Set("dashboard", data, 5*time.Minute)
}

// Clear removes all cached items
func (c *SimpleCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache.DeleteAll()
}

// Stop gracefully stops the cache's background goroutines
func (c *SimpleCache) Stop() {
	c.cache.Stop()
}

// DashboardData holds cached dashboard information
type DashboardData struct {
	Runs           []models.RunResponse
	UserInfo       *models.UserInfo
	RepositoryList []string
	LastUpdated    time.Time
}
