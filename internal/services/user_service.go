package services

import (
	"sync"
	"time"

	"github.com/repobird/repobird-cli/internal/cache"
	"github.com/repobird/repobird-cli/internal/models"
)

// UserService manages user-specific operations and caching
type UserService struct {
	mu           sync.RWMutex
	currentUser  *models.UserInfo
	cachedAt     time.Time
	cacheTimeout time.Duration
}

// NewUserService creates a new user service instance
func NewUserService() *UserService {
	return &UserService{
		cacheTimeout: 5 * time.Minute, // Cache user info for 5 minutes
	}
}

// SetCurrentUser sets the current authenticated user and updates cache accordingly
func (us *UserService) SetCurrentUser(userInfo *models.UserInfo) {
	us.mu.Lock()
	defer us.mu.Unlock()

	us.currentUser = userInfo
	us.cachedAt = time.Now()

	// Initialize cache for this specific user if we have a valid ID
	if userInfo != nil && userInfo.ID > 0 {
		cache.InitializeCacheForUser(&userInfo.ID)
		cache.InitializeDashboardForUser(&userInfo.ID)
	} else {
		// Fall back to shared cache if no user ID
		cache.InitializeCacheForUser(nil)
		cache.InitializeDashboardForUser(nil)
	}
}

// GetCurrentUser returns the current authenticated user (cached)
func (us *UserService) GetCurrentUser() *models.UserInfo {
	us.mu.RLock()
	defer us.mu.RUnlock()

	// Check if cache is still valid
	if us.currentUser != nil && time.Since(us.cachedAt) < us.cacheTimeout {
		return us.currentUser
	}

	return nil
}

// GetCurrentUserID returns the current user's ID, or nil if not available
func (us *UserService) GetCurrentUserID() *int {
	user := us.GetCurrentUser()
	if user != nil && user.ID > 0 {
		return &user.ID
	}
	return nil
}

// ClearCurrentUser clears the cached user information
func (us *UserService) ClearCurrentUser() {
	us.mu.Lock()
	defer us.mu.Unlock()

	us.currentUser = nil
	us.cachedAt = time.Time{}

	// Reset cache to shared mode
	cache.InitializeCacheForUser(nil)
	cache.InitializeDashboardForUser(nil)
}

// IsUserCacheValid checks if the current user cache is still valid
func (us *UserService) IsUserCacheValid() bool {
	us.mu.RLock()
	defer us.mu.RUnlock()

	return us.currentUser != nil && time.Since(us.cachedAt) < us.cacheTimeout
}

// Global user service instance
var globalUserService *UserService

// Initialize the global user service
func init() {
	globalUserService = NewUserService()
}

// GetUserService returns the global user service instance
func GetUserService() *UserService {
	return globalUserService
}

// Helper functions for backward compatibility

// SetCurrentUser is a convenience function to set the current user globally
func SetCurrentUser(userInfo *models.UserInfo) {
	globalUserService.SetCurrentUser(userInfo)
}

// GetCurrentUser is a convenience function to get the current user globally
func GetCurrentUser() *models.UserInfo {
	return globalUserService.GetCurrentUser()
}

// GetCurrentUserID is a convenience function to get the current user ID globally
func GetCurrentUserID() *int {
	return globalUserService.GetCurrentUserID()
}

// ClearCurrentUser is a convenience function to clear the current user globally
func ClearCurrentUser() {
	globalUserService.ClearCurrentUser()
}
