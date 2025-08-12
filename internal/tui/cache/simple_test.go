package cache

import (
	"testing"
	"time"

	"github.com/repobird/repobird-cli/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSimpleCache(t *testing.T) {
	// Set test environment to use temp directory
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cache := NewSimpleCache()
	assert.NotNil(t, cache)
	assert.NotNil(t, cache.hybrid)
	defer cache.Stop()
}

func TestCacheRuns(t *testing.T) {
	// Set test environment to use temp directory
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cache := NewSimpleCache()
	defer cache.Stop()

	// Test empty cache
	runs := cache.GetRuns()
	assert.Empty(t, runs)

	// Test setting and getting runs
	testRuns := []models.RunResponse{
		{ID: "run-1", Title: "Test Run 1", Status: "pending"},
		{ID: "run-2", Title: "Test Run 2", Status: "running"},
	}

	cache.SetRuns(testRuns)
	cachedRuns := cache.GetRuns()
	require.NotNil(t, cachedRuns)
	assert.Equal(t, testRuns, cachedRuns)
}

func TestCacheSingleRun(t *testing.T) {
	// Set test environment to use temp directory
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cache := NewSimpleCache()
	defer cache.Stop()

	// Test empty cache
	run := cache.GetRun("non-existent")
	assert.Nil(t, run)

	// Test setting and getting a single run
	testRun := models.RunResponse{
		ID:     "run-1",
		Title:  "Test Run",
		Status: "completed",
	}

	cache.SetRun(testRun)
	cachedRun := cache.GetRun("run-1")
	require.NotNil(t, cachedRun)
	assert.Equal(t, testRun, *cachedRun)

	// Test non-existent run after setting one
	nonExistent := cache.GetRun("run-2")
	assert.Nil(t, nonExistent)
}

func TestCacheUserInfo(t *testing.T) {
	// Set test environment to use temp directory
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cache := NewSimpleCache()
	defer cache.Stop()

	// Test empty cache
	info := cache.GetUserInfo()
	assert.Nil(t, info)

	// Test setting and getting user info
	testInfo := &models.UserInfo{
		ID:             1,
		GithubUsername: "testuser",
		Email:          "test@example.com",
	}

	cache.SetUserInfo(testInfo)
	cachedInfo := cache.GetUserInfo()
	require.NotNil(t, cachedInfo)
	assert.Equal(t, testInfo, cachedInfo)
}

func TestCacheFileHash(t *testing.T) {
	// Set test environment to use temp directory
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cache := NewSimpleCache()
	defer cache.Stop()

	// Test empty cache
	hash := cache.GetFileHash("/path/to/file")
	assert.Empty(t, hash)

	// Test setting and getting file hash
	testPath := "/path/to/test.json"
	testHash := "abc123def456"

	cache.SetFileHash(testPath, testHash)
	cachedHash := cache.GetFileHash(testPath)
	assert.Equal(t, testHash, cachedHash)

	// Test different path
	otherHash := cache.GetFileHash("/different/path")
	assert.Empty(t, otherHash)
}

func TestCacheDashboardData(t *testing.T) {
	// Set test environment to use temp directory
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cache := NewSimpleCache()
	defer cache.Stop()

	// Test empty cache
	data, exists := cache.GetDashboardCache()
	assert.Nil(t, data)
	assert.False(t, exists)

	// Test setting and getting dashboard data
	testData := &DashboardData{
		Runs: []models.RunResponse{
			{ID: "run-1", Title: "Dashboard Run"},
		},
		UserInfo: &models.UserInfo{
			ID:             1,
			GithubUsername: "dashuser",
		},
		RepositoryList: []string{"repo1", "repo2"},
		LastUpdated:    time.Now(),
	}

	cache.SetDashboardCache(testData)
	cachedData, exists := cache.GetDashboardCache()
	require.True(t, exists)
	require.NotNil(t, cachedData)
	assert.Equal(t, testData.Runs, cachedData.Runs)
	assert.Equal(t, testData.UserInfo, cachedData.UserInfo)
	assert.Equal(t, testData.RepositoryList, cachedData.RepositoryList)
}

func TestCacheClear(t *testing.T) {
	// Set test environment to use temp directory
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cache := NewSimpleCache()
	defer cache.Stop()

	// Add some data
	cache.SetRuns([]models.RunResponse{{ID: "run-1"}})
	cache.SetUserInfo(&models.UserInfo{ID: 1})
	cache.SetFileHash("/test/path", "hash123")

	// Verify data exists
	assert.NotNil(t, cache.GetRuns())
	assert.NotNil(t, cache.GetUserInfo())
	assert.NotEmpty(t, cache.GetFileHash("/test/path"))

	// Clear cache
	cache.Clear()

	// Verify data is gone
	assert.Empty(t, cache.GetRuns())
	assert.Nil(t, cache.GetUserInfo())
	assert.Empty(t, cache.GetFileHash("/test/path"))
}

func TestCacheConcurrency(t *testing.T) {
	// Set test environment to use temp directory
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cache := NewSimpleCache()
	defer cache.Stop()

	// Test concurrent reads and writes
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			cache.SetRuns([]models.RunResponse{{ID: "run-" + string(rune(i))}})
			cache.SetUserInfo(&models.UserInfo{ID: i})
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			_ = cache.GetRuns()
			_ = cache.GetUserInfo()
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Should not panic and should have some data
	assert.NotPanics(t, func() {
		cache.GetRuns()
		cache.GetUserInfo()
	})
}

func TestCacheExpiration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping expiration test in short mode")
	}

	// Set test environment to use temp directory
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create cache with very short TTL for testing
	cache := NewSimpleCache()
	defer cache.Stop()

	// Note: In real implementation, we'd need to modify NewSimpleCache
	// to accept TTL as parameter for testing. For now, this is a placeholder
	// to show the test structure.

	cache.SetRuns([]models.RunResponse{{ID: "expiring-run"}})
	assert.NotNil(t, cache.GetRuns())

	// In production, items expire after 5 minutes
	// For testing, we'd need a shorter TTL
}

// Benchmark tests
func BenchmarkCacheSetRuns(b *testing.B) {
	// Set test environment to use temp directory
	tmpDir := b.TempDir()
	b.Setenv("XDG_CONFIG_HOME", tmpDir)

	cache := NewSimpleCache()
	defer cache.Stop()

	runs := []models.RunResponse{
		{ID: "run-1", Title: "Test Run 1"},
		{ID: "run-2", Title: "Test Run 2"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.SetRuns(runs)
	}
}

func BenchmarkCacheGetRuns(b *testing.B) {
	// Set test environment to use temp directory
	tmpDir := b.TempDir()
	b.Setenv("XDG_CONFIG_HOME", tmpDir)

	cache := NewSimpleCache()
	defer cache.Stop()

	runs := []models.RunResponse{
		{ID: "run-1", Title: "Test Run 1"},
		{ID: "run-2", Title: "Test Run 2"},
	}
	cache.SetRuns(runs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cache.GetRuns()
	}
}

func BenchmarkCacheConcurrentAccess(b *testing.B) {
	// Set test environment to use temp directory
	tmpDir := b.TempDir()
	b.Setenv("XDG_CONFIG_HOME", tmpDir)

	cache := NewSimpleCache()
	defer cache.Stop()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%2 == 0 {
				cache.SetRuns([]models.RunResponse{{ID: "run-" + string(rune(i))}})
			} else {
				_ = cache.GetRuns()
			}
			i++
		}
	})
}
