// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package cache

import (
	"fmt"
	"sync"
	"sync/atomic"
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
		{ID: "run-1", Title: "Test Run 1", Status: "pending", CreatedAt: time.Now().Add(-2 * time.Hour)},
		{ID: "run-2", Title: "Test Run 2", Status: "running", CreatedAt: time.Now().Add(-1 * time.Hour)},
	}

	cache.SetRuns(testRuns)
	cachedRuns := cache.GetRuns()
	require.NotNil(t, cachedRuns)
	// Runs should be sorted by creation time (newest first)
	assert.Len(t, cachedRuns, 2)
	assert.Equal(t, "run-2", cachedRuns[0].ID) // Newer run first
	assert.Equal(t, "run-1", cachedRuns[1].ID) // Older run second
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
			cache.SetRuns([]models.RunResponse{{ID: fmt.Sprintf("run-%d", i)}})
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
				cache.SetRuns([]models.RunResponse{{ID: fmt.Sprintf("run-%d", i)}})
			} else {
				_ = cache.GetRuns()
			}
			i++
		}
	})
}

// TestSimpleCacheNoLockOnHybridCalls verifies SimpleCache doesn't hold locks when calling HybridCache
func TestSimpleCacheNoLockOnHybridCalls(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cache := NewSimpleCache()
	defer cache.Stop()

	// Track if operations complete without blocking
	completed := make(chan bool, 1)

	// Start concurrent operations that would deadlock if locks were held
	go func() {
		var wg sync.WaitGroup

		// Multiple goroutines calling different methods simultaneously
		for i := 0; i < 10; i++ {
			wg.Add(5)

			// GetRuns
			go func() {
				defer wg.Done()
				_ = cache.GetRuns()
			}()

			// SetRuns
			go func(idx int) {
				defer wg.Done()
				runs := []models.RunResponse{
					{ID: fmt.Sprintf("run-%d", idx), Status: models.StatusProcessing},
				}
				cache.SetRuns(runs)
			}(i)

			// GetRun
			go func(idx int) {
				defer wg.Done()
				_ = cache.GetRun(fmt.Sprintf("run-%d", idx))
			}(i)

			// SetRun
			go func(idx int) {
				defer wg.Done()
				run := models.RunResponse{ID: fmt.Sprintf("new-%d", idx), Status: models.StatusDone}
				cache.SetRun(run)
			}(i)

			// Mixed operations
			go func(idx int) {
				defer wg.Done()
				cache.SetUserInfo(&models.UserInfo{ID: idx})
				_ = cache.GetUserInfo()
			}(i)
		}

		wg.Wait()
		completed <- true
	}()

	select {
	case <-completed:
		// Success - no deadlock
	case <-time.After(2 * time.Second):
		t.Fatal("Operations blocked - possible lock held during HybridCache calls")
	}
}

// TestSimpleCacheGetRunsCopyOnRead verifies GetRuns returns a copy
func TestSimpleCacheGetRunsCopyOnRead(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cache := NewSimpleCache()
	defer cache.Stop()

	// Set initial runs
	originalRuns := []models.RunResponse{
		{ID: "run-1", Status: models.StatusProcessing},
		{ID: "run-2", Status: models.StatusDone},
	}
	cache.SetRuns(originalRuns)

	// Get runs and modify the returned slice
	runs := cache.GetRuns()
	require.Len(t, runs, 2)

	// Modify the returned slice
	runs[0].Status = models.StatusFailed
	runs[0].ID = "modified"

	// Get runs again and verify original data is unchanged
	runs2 := cache.GetRuns()
	assert.Equal(t, "run-1", runs2[0].ID)
	assert.Equal(t, models.StatusProcessing, runs2[0].Status)

	// Verify modification didn't affect cache
	assert.Equal(t, "modified", runs[0].ID)
	assert.Equal(t, models.StatusFailed, runs[0].Status)
}

// TestSimpleCacheThreadSafety tests thread safety of all SimpleCache operations
func TestSimpleCacheThreadSafety(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cache := NewSimpleCache()
	defer cache.Stop()

	// Use atomic counters to track successful operations
	var getRunsCount int32
	var setRunsCount int32
	var getRunCount int32
	var setRunCount int32
	var getUserInfoCount int32
	var setUserInfoCount int32

	numGoroutines := 100
	opsPerGoroutine := 50

	var wg sync.WaitGroup

	// Launch goroutines performing random operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < opsPerGoroutine; j++ {
				switch j % 6 {
				case 0:
					_ = cache.GetRuns()
					atomic.AddInt32(&getRunsCount, 1)
				case 1:
					runs := []models.RunResponse{
						{ID: fmt.Sprintf("run-%d-%d", id, j), Status: models.StatusProcessing},
					}
					cache.SetRuns(runs)
					atomic.AddInt32(&setRunsCount, 1)
				case 2:
					_ = cache.GetRun(fmt.Sprintf("run-%d", id))
					atomic.AddInt32(&getRunCount, 1)
				case 3:
					run := models.RunResponse{ID: fmt.Sprintf("single-%d-%d", id, j), Status: models.StatusDone}
					cache.SetRun(run)
					atomic.AddInt32(&setRunCount, 1)
				case 4:
					_ = cache.GetUserInfo()
					atomic.AddInt32(&getUserInfoCount, 1)
				case 5:
					cache.SetUserInfo(&models.UserInfo{ID: id, Email: fmt.Sprintf("user%d@test.com", id)})
					atomic.AddInt32(&setUserInfoCount, 1)
				}
			}
		}(i)
	}

	// Wait for completion
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Verify all operations completed
		totalOps := atomic.LoadInt32(&getRunsCount) + atomic.LoadInt32(&setRunsCount) +
			atomic.LoadInt32(&getRunCount) + atomic.LoadInt32(&setRunCount) +
			atomic.LoadInt32(&getUserInfoCount) + atomic.LoadInt32(&setUserInfoCount)

		expectedOps := int32(numGoroutines * opsPerGoroutine)
		assert.Equal(t, expectedOps, totalOps, "Not all operations completed")

		t.Logf("Operations completed: GetRuns=%d, SetRuns=%d, GetRun=%d, SetRun=%d, GetUserInfo=%d, SetUserInfo=%d",
			atomic.LoadInt32(&getRunsCount), atomic.LoadInt32(&setRunsCount),
			atomic.LoadInt32(&getRunCount), atomic.LoadInt32(&setRunCount),
			atomic.LoadInt32(&getUserInfoCount), atomic.LoadInt32(&setUserInfoCount))

	case <-time.After(10 * time.Second):
		t.Fatal("Timeout - operations did not complete, possible deadlock")
	}
}

// TestSimpleCacheNavigationContext tests the navigation context methods
func TestSimpleCacheNavigationContext(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cache := NewSimpleCache()
	defer cache.Stop()

	// Test setting and getting navigation context
	cache.SetNavigationContext("selectedRepo", "test/repo")
	cache.SetNavigationContext("selectedRun", "run-123")

	assert.Equal(t, "test/repo", cache.GetNavigationContext("selectedRepo"))
	assert.Equal(t, "run-123", cache.GetNavigationContext("selectedRun"))
	assert.Nil(t, cache.GetNavigationContext("nonexistent"))

	// Test clearing all navigation context
	cache.ClearAllNavigationContext()
	assert.Nil(t, cache.GetNavigationContext("selectedRepo"))
	assert.Nil(t, cache.GetNavigationContext("selectedRun"))

	// Test concurrent navigation context operations
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func(idx int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", idx)
			value := fmt.Sprintf("value-%d", idx)
			cache.SetNavigationContext(key, value)
		}(i)

		go func(idx int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", idx)
			_ = cache.GetNavigationContext(key)
		}(i)
	}
	wg.Wait()
}
