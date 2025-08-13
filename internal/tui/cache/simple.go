package cache

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/services"
)

// SimpleCache now wraps HybridCache for backward compatibility
// while maintaining the same interface
type SimpleCache struct {
	hybrid      *HybridCache
	mu          sync.RWMutex // Additional safety for concurrent access
	contextData sync.Map     // Thread-safe map for navigation context
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
	// Get string user ID from the global user service (use original ID, not hash)
	stringID := services.GetCurrentUserStringID()
	if stringID != "" {
		return stringID
	}
	
	// Fallback to integer ID if string ID not available (backward compatibility)
	userIDPtr := services.GetCurrentUserID()
	if userIDPtr != nil && *userIDPtr > 0 {
		return fmt.Sprintf("%d", *userIDPtr)
	}

	// Fallback to anonymous if no user is authenticated
	return "anonymous"
}

// GetRuns retrieves cached runs from the hybrid cache
func (c *SimpleCache) GetRuns() []models.RunResponse {
	// No lock needed - HybridCache handles thread safety
	runs, _ := c.hybrid.GetRuns()

	// Return copy to avoid mutations
	result := make([]models.RunResponse, len(runs))
	copy(result, runs)
	return result
}

// SetRuns caches runs using the hybrid cache
func (c *SimpleCache) SetRuns(runs []models.RunResponse) {
	// No lock needed - HybridCache handles thread safety
	_ = c.hybrid.SetRuns(runs)
}

// GetRun retrieves a single cached run by ID
func (c *SimpleCache) GetRun(id string) *models.RunResponse {
	// No lock needed - HybridCache handles thread safety
	run, found := c.hybrid.GetRun(id)
	if !found {
		return nil
	}
	return run
}

// SetRun caches a single run
func (c *SimpleCache) SetRun(run models.RunResponse) {
	// No lock needed - HybridCache handles thread safety
	_ = c.hybrid.SetRun(run)
}

// GetUserInfo retrieves cached user info
func (c *SimpleCache) GetUserInfo() *models.UserInfo {
	// No lock needed - HybridCache handles thread safety
	info, found := c.hybrid.GetUserInfo()
	if !found {
		return nil
	}
	return info
}

// SetUserInfo caches user info
func (c *SimpleCache) SetUserInfo(info *models.UserInfo) {
	// No lock needed - HybridCache handles thread safety
	_ = c.hybrid.SetUserInfo(info)
}

// GetFileHash retrieves cached file hash
func (c *SimpleCache) GetFileHash(path string) string {
	// No lock needed - HybridCache handles thread safety
	hash, _ := c.hybrid.GetFileHash(path)
	return hash
}

// SetFileHash caches file hash
func (c *SimpleCache) SetFileHash(path string, hash string) {
	// No lock needed - HybridCache handles thread safety
	_ = c.hybrid.SetFileHash(path, hash)
}

// GetDashboardCache retrieves cached dashboard data
func (c *SimpleCache) GetDashboardCache() (*DashboardData, bool) {
	// No lock needed - HybridCache handles thread safety
	return c.hybrid.GetDashboardData()
}

// SetDashboardCache stores dashboard data
func (c *SimpleCache) SetDashboardCache(data *DashboardData) {
	// No lock needed - HybridCache handles thread safety
	_ = c.hybrid.SetDashboardData(data)
}

// Clear removes all cached items
func (c *SimpleCache) Clear() {
	// No lock needed - HybridCache handles thread safety
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

// Navigation Context Methods - Thread-safe using sync.Map

// SetContext stores a context value
func (c *SimpleCache) SetContext(key string, value interface{}) {
	c.contextData.Store(key, value)
}

// GetContext retrieves a context value
func (c *SimpleCache) GetContext(key string) interface{} {
	if val, ok := c.contextData.Load(key); ok {
		return val
	}
	return nil
}

// ClearContext removes a specific context value
func (c *SimpleCache) ClearContext(key string) {
	c.contextData.Delete(key)
}

// SetNavigationContext stores temporary navigation context
func (c *SimpleCache) SetNavigationContext(key string, value interface{}) {
	// Prefix with "nav:" to distinguish navigation context
	c.SetContext("nav:"+key, value)
}

// GetNavigationContext retrieves navigation context
func (c *SimpleCache) GetNavigationContext(key string) interface{} {
	return c.GetContext("nav:" + key)
}

// ClearAllNavigationContext removes all navigation context
func (c *SimpleCache) ClearAllNavigationContext() {
	// Iterate through all keys and delete those with "nav:" prefix
	c.contextData.Range(func(k, v interface{}) bool {
		if key, ok := k.(string); ok && strings.HasPrefix(key, "nav:") {
			c.contextData.Delete(k)
		}
		return true
	})
}

// GetAuthCache retrieves cached authentication info with timestamp
func (c *SimpleCache) GetAuthCache() (*AuthCache, bool) {
	return c.hybrid.GetAuthCache()
}

// SetAuthCache stores authentication info with timestamp
func (c *SimpleCache) SetAuthCache(userInfo *models.UserInfo) error {
	return c.hybrid.SetAuthCache(userInfo)
}

// IsAuthCacheValid checks if cached authentication is still valid
func (c *SimpleCache) IsAuthCacheValid() bool {
	return c.hybrid.IsAuthCacheValid()
}
