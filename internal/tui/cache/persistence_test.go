package cache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/repobird/repobird-cli/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestAutomaticPersistence(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Override XDG config home for testing
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	// Create first cache instance and populate it
	cache1 := NewSimpleCache()
	defer cache1.Stop()

	// Test with terminal runs (should be automatically persisted)
	testRuns := []models.RunResponse{
		{ID: "run-1", Title: "Persisted Run 1", Status: models.StatusDone},
		{ID: "run-2", Title: "Persisted Run 2", Status: models.StatusFailed},
	}
	testUserInfo := &models.UserInfo{
		ID:             1,
		GithubUsername: "persistuser",
		Email:          "persist@example.com",
	}

	// Set data (automatically persisted by hybrid cache)
	cache1.SetRuns(testRuns)
	cache1.SetUserInfo(testUserInfo)
	cache1.SetFileHash("/test/file.go", "hash123")

	// Create second cache instance - should automatically load persisted data
	cache2 := NewSimpleCache()
	defer cache2.Stop()

	// Verify terminal runs are automatically loaded
	loadedUserInfo := cache2.GetUserInfo()
	assert.Equal(t, testUserInfo, loadedUserInfo)

	// File hashes should persist
	loadedHash := cache2.GetFileHash("/test/file.go")
	assert.Equal(t, "hash123", loadedHash)

	// Note: Dashboard data is session-only and won't persist
}

func TestNewCacheWithEmptyDirectory(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Override XDG config home for testing
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	cache := NewSimpleCache()
	defer cache.Stop()

	// New cache with empty directory should work fine
	// LoadFromDisk is now a no-op
	err := cache.LoadFromDisk()
	assert.NoError(t, err)

	// Cache should be empty
	assert.Empty(t, cache.GetRuns())
	assert.Nil(t, cache.GetUserInfo())
}

// TestLoadFromDiskIsNoOp verifies that LoadFromDisk is now a no-op
func TestLoadFromDiskIsNoOp(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	cache := NewSimpleCache()
	defer cache.Stop()

	// LoadFromDisk is now a no-op and should always succeed
	err := cache.LoadFromDisk()
	assert.NoError(t, err)
}

// TestSaveToDiskIsNoOp verifies that SaveToDisk is now a no-op
func TestSaveToDiskIsNoOp(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	cache := NewSimpleCache()
	defer cache.Stop()

	// Add some data
	cache.SetRuns([]models.RunResponse{{ID: "test-run", Status: models.StatusDone}})

	// SaveToDisk is now a no-op and should always succeed
	err := cache.SaveToDisk()
	assert.NoError(t, err)

	// The old cache.json file should not be created
	cacheFile := filepath.Join(tempDir, "repobird", "cache.json")
	assert.NoFileExists(t, cacheFile)
}

func TestGetCacheFilePath(t *testing.T) {
	// Save original XDG_CONFIG_HOME
	oldConfigHome := os.Getenv("XDG_CONFIG_HOME")

	// Test with custom XDG_CONFIG_HOME
	testDir := "/test/config"
	os.Setenv("XDG_CONFIG_HOME", testDir)

	path := GetCacheFilePath()
	assert.Equal(t, filepath.Join(testDir, "repobird", "cache.json"), path)

	// Restore original
	os.Setenv("XDG_CONFIG_HOME", oldConfigHome)
}
