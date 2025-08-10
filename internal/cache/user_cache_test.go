package cache

import (
	"testing"
	"time"

	"github.com/repobird/repobird-cli/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestUserSpecificCache(t *testing.T) {
	// Clean start
	ClearCache()
	defer ClearCache()

	// Test user 1
	user1ID := 123
	InitializeCacheForUser(&user1ID)

	// Add repository history for user 1
	err := AddRepositoryToHistory("user1/repo1")
	assert.NoError(t, err)
	err = AddRepositoryToHistory("user1/repo2")
	assert.NoError(t, err)

	// Verify user 1's history
	history, err := GetRepositoryHistory()
	assert.NoError(t, err)
	assert.Len(t, history, 2)
	assert.Equal(t, "user1/repo2", history[0]) // Most recent first
	assert.Equal(t, "user1/repo1", history[1])

	// Save form data for user 1
	SaveFormData(&FormData{
		Title:      "User 1 Task",
		Repository: "user1/repo1",
		Prompt:     "User 1 prompt",
	})

	formData := GetFormData()
	assert.NotNil(t, formData)
	assert.Equal(t, "User 1 Task", formData.Title)

	// Switch to user 2
	user2ID := 456
	InitializeCacheForUser(&user2ID)

	// User 2 should have empty history
	history, err = GetRepositoryHistory()
	assert.NoError(t, err)
	assert.Len(t, history, 0)

	// User 2 should have no form data
	formData = GetFormData()
	assert.Nil(t, formData)

	// Add repository history for user 2
	err = AddRepositoryToHistory("user2/repo1")
	assert.NoError(t, err)

	// Save form data for user 2
	SaveFormData(&FormData{
		Title:      "User 2 Task",
		Repository: "user2/repo1",
		Prompt:     "User 2 prompt",
	})

	// Verify user 2's data
	history, err = GetRepositoryHistory()
	assert.NoError(t, err)
	assert.Len(t, history, 1)
	assert.Equal(t, "user2/repo1", history[0])

	formData = GetFormData()
	assert.NotNil(t, formData)
	assert.Equal(t, "User 2 Task", formData.Title)

	// Switch back to user 1
	InitializeCacheForUser(&user1ID)

	// User 1's history should still be there
	history, err = GetRepositoryHistory()
	assert.NoError(t, err)
	assert.Len(t, history, 2)
	assert.Equal(t, "user1/repo2", history[0])
	assert.Equal(t, "user1/repo1", history[1])
}

func TestUserInfoCache(t *testing.T) {
	// Clean start
	ClearCache()
	defer ClearCache()

	// Initially no cached user info
	cachedInfo := GetCachedUserInfo()
	assert.Nil(t, cachedInfo)

	// Set user info
	userInfo := &models.UserInfo{
		ID:            123,
		Email:         "test@example.com",
		RemainingRuns: 10,
		TotalRuns:     100,
		Tier:          "premium",
	}
	SetCachedUserInfo(userInfo)

	// Should retrieve cached info
	cachedInfo = GetCachedUserInfo()
	assert.NotNil(t, cachedInfo)
	assert.Equal(t, 123, cachedInfo.ID)
	assert.Equal(t, "test@example.com", cachedInfo.Email)

	// Test that same user's info is preserved when reinitializing
	userID := 123
	InitializeCacheForUser(&userID)

	// User info should still be cached for same user
	cachedInfo = GetCachedUserInfo()
	assert.NotNil(t, cachedInfo)
	assert.Equal(t, 123, cachedInfo.ID)

	// Switch to different user
	user2ID := 456
	InitializeCacheForUser(&user2ID)

	// User info should be cleared for different user
	cachedInfo = GetCachedUserInfo()
	assert.Nil(t, cachedInfo)
}

func TestUserInfoCacheExpiry(t *testing.T) {
	// Clean start
	ClearCache()
	defer ClearCache()

	// Set user info
	userInfo := &models.UserInfo{
		ID:    123,
		Email: "test@example.com",
	}
	SetCachedUserInfo(userInfo)

	// Should retrieve cached info immediately
	cachedInfo := GetCachedUserInfo()
	assert.NotNil(t, cachedInfo)

	// Manually expire the cache (simulate 6 minutes passing)
	globalCache.mu.Lock()
	globalCache.userInfoTime = time.Now().Add(-6 * time.Minute)
	globalCache.mu.Unlock()

	// Should return nil for expired cache
	cachedInfo = GetCachedUserInfo()
	assert.Nil(t, cachedInfo)
}
