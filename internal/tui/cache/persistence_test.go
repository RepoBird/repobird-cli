package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/repobird/repobird-cli/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveAndLoadFromDisk(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Override XDG config home for testing
	oldConfigHome := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	defer os.Setenv("XDG_CONFIG_HOME", oldConfigHome)

	// Create first cache instance and populate it
	cache1 := NewSimpleCache()
	defer cache1.Stop()

	testRuns := []models.RunResponse{
		{ID: "run-1", Title: "Persisted Run 1"},
		{ID: "run-2", Title: "Persisted Run 2"},
	}
	testUserInfo := &models.UserInfo{
		ID:             1,
		GithubUsername: "persistuser",
		Email:          "persist@example.com",
	}
	testDashboard := &DashboardData{
		Runs:           testRuns,
		UserInfo:       testUserInfo,
		RepositoryList: []string{"repo1", "repo2", "repo3"},
		LastUpdated:    time.Now(),
	}

	cache1.SetRuns(testRuns)
	cache1.SetUserInfo(testUserInfo)
	cache1.SetDashboardCache(testDashboard)
	cache1.SetFileHash("/test/file.go", "hash123")

	// Save to disk
	err := cache1.SaveToDisk()
	require.NoError(t, err)

	// Verify file exists
	cacheFile := filepath.Join(tempDir, "repobird", "cache.json")
	assert.FileExists(t, cacheFile)

	// Create second cache instance and load from disk
	cache2 := NewSimpleCache()
	defer cache2.Stop()

	err = cache2.LoadFromDisk()
	require.NoError(t, err)

	// Verify loaded data
	loadedRuns := cache2.GetRuns()
	assert.Equal(t, testRuns, loadedRuns)

	loadedUserInfo := cache2.GetUserInfo()
	assert.Equal(t, testUserInfo, loadedUserInfo)

	loadedDashboard, exists := cache2.GetDashboardCache()
	assert.True(t, exists)
	assert.Equal(t, testDashboard.Runs, loadedDashboard.Runs)
	assert.Equal(t, testDashboard.UserInfo, loadedDashboard.UserInfo)
	assert.Equal(t, testDashboard.RepositoryList, loadedDashboard.RepositoryList)
}

func TestLoadFromDiskWithMissingFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Override XDG config home for testing
	oldConfigHome := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	defer os.Setenv("XDG_CONFIG_HOME", oldConfigHome)

	cache := NewSimpleCache()
	defer cache.Stop()

	// Loading from non-existent file should not error
	err := cache.LoadFromDisk()
	assert.NoError(t, err)

	// Cache should be empty
	assert.Nil(t, cache.GetRuns())
	assert.Nil(t, cache.GetUserInfo())
}

func TestLoadFromDiskWithCorruptedFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Override XDG config home for testing
	oldConfigHome := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	defer os.Setenv("XDG_CONFIG_HOME", oldConfigHome)

	// Create corrupted cache file
	cacheDir := filepath.Join(tempDir, "repobird")
	err := os.MkdirAll(cacheDir, 0700)
	require.NoError(t, err)

	cacheFile := filepath.Join(cacheDir, "cache.json")
	err = os.WriteFile(cacheFile, []byte("not valid json"), 0600)
	require.NoError(t, err)

	cache := NewSimpleCache()
	defer cache.Stop()

	// Loading corrupted file should error
	err = cache.LoadFromDisk()
	assert.Error(t, err)
}

func TestLoadFromDiskWithExpiredCache(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Override XDG config home for testing
	oldConfigHome := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	defer os.Setenv("XDG_CONFIG_HOME", oldConfigHome)

	// Create cache data with old timestamp
	oldCacheData := CacheData{
		Runs: []models.RunResponse{
			{ID: "old-run", Title: "Old Run"},
		},
		UserInfo: &models.UserInfo{
			ID:             2,
			GithubUsername: "olduser",
		},
		SavedAt: time.Now().Add(-2 * time.Hour), // 2 hours old
	}

	cacheDir := filepath.Join(tempDir, "repobird")
	err := os.MkdirAll(cacheDir, 0700)
	require.NoError(t, err)

	cacheFile := filepath.Join(cacheDir, "cache.json")
	jsonData, err := json.Marshal(oldCacheData)
	require.NoError(t, err)
	err = os.WriteFile(cacheFile, jsonData, 0600)
	require.NoError(t, err)

	cache := NewSimpleCache()
	defer cache.Stop()

	// Load from disk
	err = cache.LoadFromDisk()
	require.NoError(t, err)

	// Cache should be empty because data is too old (> 1 hour)
	assert.Nil(t, cache.GetRuns())
	assert.Nil(t, cache.GetUserInfo())
}

func TestLoadFromDiskWithFreshCache(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Override XDG config home for testing
	oldConfigHome := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	defer os.Setenv("XDG_CONFIG_HOME", oldConfigHome)

	// Create cache data with recent timestamp
	freshCacheData := CacheData{
		Runs: []models.RunResponse{
			{ID: "fresh-run", Title: "Fresh Run"},
		},
		UserInfo: &models.UserInfo{
			ID:             3,
			GithubUsername: "freshuser",
		},
		SavedAt: time.Now().Add(-30 * time.Minute), // 30 minutes old
	}

	cacheDir := filepath.Join(tempDir, "repobird")
	err := os.MkdirAll(cacheDir, 0700)
	require.NoError(t, err)

	cacheFile := filepath.Join(cacheDir, "cache.json")
	jsonData, err := json.Marshal(freshCacheData)
	require.NoError(t, err)
	err = os.WriteFile(cacheFile, jsonData, 0600)
	require.NoError(t, err)

	cache := NewSimpleCache()
	defer cache.Stop()

	// Load from disk
	err = cache.LoadFromDisk()
	require.NoError(t, err)

	// Cache should contain the fresh data
	runs := cache.GetRuns()
	require.NotNil(t, runs)
	assert.Len(t, runs, 1)
	assert.Equal(t, "fresh-run", runs[0].ID)

	userInfo := cache.GetUserInfo()
	require.NotNil(t, userInfo)
	assert.Equal(t, 3, userInfo.ID)
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

func TestSaveToDiskCreatesDirectory(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Override XDG config home for testing
	oldConfigHome := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	defer os.Setenv("XDG_CONFIG_HOME", oldConfigHome)

	cache := NewSimpleCache()
	defer cache.Stop()

	// Add some data
	cache.SetRuns([]models.RunResponse{{ID: "test-run"}})

	// Save to disk (directory should be created automatically)
	err := cache.SaveToDisk()
	require.NoError(t, err)

	// Verify directory and file exist
	cacheDir := filepath.Join(tempDir, "repobird")
	assert.DirExists(t, cacheDir)

	cacheFile := filepath.Join(cacheDir, "cache.json")
	assert.FileExists(t, cacheFile)

	// Verify file permissions
	info, err := os.Stat(cacheFile)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}
