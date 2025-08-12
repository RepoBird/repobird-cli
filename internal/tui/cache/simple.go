package cache

import (
	"fmt"
	"sync"
	"time"

	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/services"
)

// SimpleCache now wraps HybridCache for backward compatibility
// while maintaining the same interface
type SimpleCache struct {
	hybrid *HybridCache
	mu     sync.RWMutex // Additional safety for concurrent access
}

// NewSimpleCache creates a cache with the new hybrid architecture
func NewSimpleCache() *SimpleCache {
	// Get current user ID from auth context or config
	userID := getCurrentUserID()

	hybrid, err := NewHybridCache(userID)
	if err != nil {
		// If hybrid cache creation fails, create session-only cache
		hybrid = &HybridCache{
			session: NewSessionCache(),
			userID:  userID,
		}
	}

	return &SimpleCache{
		hybrid: hybrid,
	}
}

// getCurrentUserID retrieves the current user ID from context
func getCurrentUserID() string {
	// Get user ID from the global user service
	userIDPtr := services.GetCurrentUserID()
	if userIDPtr != nil && *userIDPtr > 0 {
		return fmt.Sprintf("%d", *userIDPtr)
	}

	// Fallback to anonymous if no user is authenticated
	return "anonymous"
}

// GetRuns retrieves cached runs from the hybrid cache
func (c *SimpleCache) GetRuns() []models.RunResponse {
	c.mu.RLock()
	defer c.mu.RUnlock()

	runs, _ := c.hybrid.GetRuns()
	return runs
}

// SetRuns caches runs using the hybrid cache
func (c *SimpleCache) SetRuns(runs []models.RunResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	_ = c.hybrid.SetRuns(runs)
}

// GetRun retrieves a single cached run by ID
func (c *SimpleCache) GetRun(id string) *models.RunResponse {
	c.mu.RLock()
	defer c.mu.RUnlock()

	run, found := c.hybrid.GetRun(id)
	if !found {
		return nil
	}
	return run
}

// SetRun caches a single run
func (c *SimpleCache) SetRun(run models.RunResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	_ = c.hybrid.SetRun(run)
}

// GetUserInfo retrieves cached user info
func (c *SimpleCache) GetUserInfo() *models.UserInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	info, found := c.hybrid.GetUserInfo()
	if !found {
		return nil
	}
	return info
}

// SetUserInfo caches user info
func (c *SimpleCache) SetUserInfo(info *models.UserInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	_ = c.hybrid.SetUserInfo(info)
}

// GetFileHash retrieves cached file hash
func (c *SimpleCache) GetFileHash(path string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	hash, _ := c.hybrid.GetFileHash(path)
	return hash
}

// SetFileHash caches file hash
func (c *SimpleCache) SetFileHash(path string, hash string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	_ = c.hybrid.SetFileHash(path, hash)
}

// GetDashboardCache retrieves cached dashboard data
func (c *SimpleCache) GetDashboardCache() (*DashboardData, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.hybrid.GetDashboardData()
}

// SetDashboardCache stores dashboard data
func (c *SimpleCache) SetDashboardCache(data *DashboardData) {
	c.mu.Lock()
	defer c.mu.Unlock()

	_ = c.hybrid.SetDashboardData(data)
}

// Clear removes all cached items
func (c *SimpleCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	_ = c.hybrid.Clear()
}

// Stop gracefully stops the cache's background goroutines
func (c *SimpleCache) Stop() {
	_ = c.hybrid.Close()
}

// DashboardData holds cached dashboard information
type DashboardData struct {
	Runs           []models.RunResponse
	UserInfo       *models.UserInfo
	RepositoryList []string
	LastUpdated    time.Time
}
