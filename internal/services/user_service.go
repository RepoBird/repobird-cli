// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package services

import (
	"sync"
	"time"

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

	// Cache initialization is now handled by individual TUI views with embedded cache
	// Each view has its own cache instance, no global initialization needed
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

// GetCurrentUserStringID returns the current user's string ID, or empty string if not available
func (us *UserService) GetCurrentUserStringID() string {
	user := us.GetCurrentUser()
	if user != nil && user.StringID != "" {
		return user.StringID
	}
	return ""
}

// ClearCurrentUser clears the cached user information
func (us *UserService) ClearCurrentUser() {
	us.mu.Lock()
	defer us.mu.Unlock()

	us.currentUser = nil
	us.cachedAt = time.Time{}

	// Cache is now embedded in TUI views, no global reset needed
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

// GetCurrentUserStringID is a convenience function to get the current user's string ID globally
func GetCurrentUserStringID() string {
	return globalUserService.GetCurrentUserStringID()
}

// ClearCurrentUser is a convenience function to clear the current user globally
func ClearCurrentUser() {
	globalUserService.ClearCurrentUser()
}
