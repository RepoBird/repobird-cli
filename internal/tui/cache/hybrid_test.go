// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package cache

import (
	"os"
	"testing"
	"time"

	"github.com/repobird/repobird-cli/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHybridCache_StatusTransition(t *testing.T) {
	// Setup test directory
	tmpDir := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")

	cache, err := NewHybridCache("test-user")
	require.NoError(t, err)
	defer func() { _ = cache.Close() }()

	// Start with active run
	run := models.RunResponse{
		ID:        "test-1",
		Status:    models.StatusProcessing,
		CreatedAt: time.Now(),
	}
	err = cache.SetRun(run)
	assert.NoError(t, err)

	// Should be retrievable and active
	cached, found := cache.GetRun("test-1")
	assert.True(t, found)
	assert.Equal(t, models.StatusProcessing, cached.Status)

	// Update to terminal status
	run.Status = models.StatusDone
	err = cache.SetRun(run)
	assert.NoError(t, err)

	// Should still be retrievable but now from permanent storage
	cached, found = cache.GetRun("test-1")
	assert.True(t, found)
	assert.Equal(t, models.StatusDone, cached.Status)

	// Should persist across cache recreation
	cache2, err := NewHybridCache("test-user")
	require.NoError(t, err)
	defer func() { _ = cache2.Close() }()

	cached, found = cache2.GetRun("test-1")
	assert.True(t, found, "terminal run should persist")
	assert.Equal(t, models.StatusDone, cached.Status)
}

func TestHybridCache_MixedRunStates(t *testing.T) {
	// Setup test directory
	tmpDir := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")

	cache, err := NewHybridCache("test-user")
	require.NoError(t, err)
	defer func() { _ = cache.Close() }()

	// Create runs with different states and unique times
	runs := []models.RunResponse{
		{ID: "active-1", Status: models.StatusQueued, CreatedAt: time.Now().Add(-90 * time.Minute)},     // Recent active
		{ID: "active-2", Status: models.StatusProcessing, CreatedAt: time.Now().Add(-60 * time.Minute)}, // Recent active
		{ID: "terminal-1", Status: models.StatusDone, CreatedAt: time.Now().Add(-120 * time.Minute)},    // 2 hours ago
		{ID: "terminal-2", Status: models.StatusFailed, CreatedAt: time.Now().Add(-30 * time.Minute)},   // 30 min ago
		{ID: "active-3", Status: models.StatusInitializing, CreatedAt: time.Now()},                      // Now
	}

	// Set all runs
	err = cache.SetRuns(runs)
	assert.NoError(t, err)

	// Get all runs should return both active and terminal
	cachedRuns, found := cache.GetRuns()
	assert.True(t, found)
	assert.Len(t, cachedRuns, 5, "should return all runs")

	// Verify runs are sorted by creation time (newest first)
	assert.Equal(t, "active-3", cachedRuns[0].ID)   // Now
	assert.Equal(t, "terminal-2", cachedRuns[1].ID) // -30 min

	// Invalidate active runs
	err = cache.InvalidateActiveRuns()
	assert.NoError(t, err)

	// Should still have terminal runs
	cachedRuns, found = cache.GetRuns()
	assert.True(t, found)
	assert.Len(t, cachedRuns, 2, "should only have terminal runs after invalidation")

	for _, run := range cachedRuns {
		assert.True(t, isTerminalState(run.Status), "only terminal runs should remain")
	}
}

func TestHybridCache_UserInfo(t *testing.T) {
	// Setup test directory
	tmpDir := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")

	cache, err := NewHybridCache("test-user")
	require.NoError(t, err)
	defer func() { _ = cache.Close() }()

	// Initially no user info
	_, found := cache.GetUserInfo()
	assert.False(t, found)

	// Set user info
	userInfo := &models.UserInfo{
		ID:    456,
		Email: "user@example.com",
		Name:  "Test User",
	}
	err = cache.SetUserInfo(userInfo)
	assert.NoError(t, err)

	// Retrieve user info
	cached, found := cache.GetUserInfo()
	assert.True(t, found)
	assert.Equal(t, userInfo.ID, cached.ID)
	assert.Equal(t, userInfo.Email, cached.Email)

	// Should persist
	cache2, err := NewHybridCache("test-user")
	require.NoError(t, err)
	defer func() { _ = cache2.Close() }()

	cached, found = cache2.GetUserInfo()
	assert.True(t, found)
	assert.Equal(t, userInfo.ID, cached.ID)
}

func TestHybridCache_FileHashes(t *testing.T) {
	// Setup test directory
	tmpDir := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")

	cache, err := NewHybridCache("test-user")
	require.NoError(t, err)
	defer func() { _ = cache.Close() }()

	// Set file hashes
	err = cache.SetFileHash("file1.go", "abc123")
	assert.NoError(t, err)
	err = cache.SetFileHash("file2.go", "def456")
	assert.NoError(t, err)

	// Get individual hash
	hash, found := cache.GetFileHash("file1.go")
	assert.True(t, found)
	assert.Equal(t, "abc123", hash)

	// Get all hashes
	allHashes := cache.GetAllFileHashes()
	assert.Len(t, allHashes, 2)
	assert.Equal(t, "abc123", allHashes["file1.go"])
	assert.Equal(t, "def456", allHashes["file2.go"])

	// Should persist
	cache2, err := NewHybridCache("test-user")
	require.NoError(t, err)
	defer func() { _ = cache2.Close() }()

	hash, found = cache2.GetFileHash("file1.go")
	assert.True(t, found)
	assert.Equal(t, "abc123", hash)
}

func TestHybridCache_RepositoryList(t *testing.T) {
	// Setup test directory
	tmpDir := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")

	cache, err := NewHybridCache("test-user")
	require.NoError(t, err)
	defer func() { _ = cache.Close() }()

	// Set repository list
	repos := []string{"org/repo1", "org/repo2", "user/repo3"}
	err = cache.SetRepositoryList(repos)
	assert.NoError(t, err)

	// Get repository list
	cached, found := cache.GetRepositoryList()
	assert.True(t, found)
	assert.Equal(t, repos, cached)

	// Should persist
	cache2, err := NewHybridCache("test-user")
	require.NoError(t, err)
	defer func() { _ = cache2.Close() }()

	cached, found = cache2.GetRepositoryList()
	assert.True(t, found)
	assert.Equal(t, repos, cached)
}

func TestHybridCache_DashboardData(t *testing.T) {
	// Setup test directory
	tmpDir := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")

	cache, err := NewHybridCache("test-user")
	require.NoError(t, err)
	defer func() { _ = cache.Close() }()

	// Create dashboard data
	dashData := &DashboardData{
		Runs: []models.RunResponse{
			{ID: "run-1", Status: models.StatusProcessing},
			{ID: "run-2", Status: models.StatusDone},
		},
		UserInfo: &models.UserInfo{
			ID: 789,
		},
		RepositoryList: []string{"repo1", "repo2"},
		LastUpdated:    time.Now(),
	}

	// Set dashboard data
	err = cache.SetDashboardData(dashData)
	assert.NoError(t, err)

	// Get dashboard data (session cache only)
	cached, found := cache.GetDashboardData()
	assert.True(t, found)
	assert.Len(t, cached.Runs, 2)

	// Dashboard data should NOT persist (it's session-only)
	cache2, err := NewHybridCache("test-user")
	require.NoError(t, err)
	defer func() { _ = cache2.Close() }()

	_, found = cache2.GetDashboardData()
	assert.False(t, found, "dashboard data should not persist")
}

func TestHybridCache_InvalidateRun(t *testing.T) {
	// Setup test directory
	tmpDir := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")

	cache, err := NewHybridCache("test-user")
	require.NoError(t, err)
	defer func() { _ = cache.Close() }()

	// Add both active and terminal runs
	activeRun := models.RunResponse{
		ID:     "active-run",
		Status: models.StatusProcessing,
	}
	terminalRun := models.RunResponse{
		ID:     "terminal-run",
		Status: models.StatusDone,
	}

	_ = cache.SetRun(activeRun)
	_ = cache.SetRun(terminalRun)

	// Verify both exist
	_, found := cache.GetRun("active-run")
	assert.True(t, found)
	_, found = cache.GetRun("terminal-run")
	assert.True(t, found)

	// Invalidate terminal run
	err = cache.InvalidateRun("terminal-run")
	assert.NoError(t, err)

	// Terminal run should be gone
	_, found = cache.GetRun("terminal-run")
	assert.False(t, found)

	// Active run should still exist
	_, found = cache.GetRun("active-run")
	assert.True(t, found)

	// Invalidate active run
	err = cache.InvalidateRun("active-run")
	assert.NoError(t, err)

	// Active run should be gone
	_, found = cache.GetRun("active-run")
	assert.False(t, found)
}

func TestHybridCache_Clear(t *testing.T) {
	// Setup test directory
	tmpDir := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")

	cache, err := NewHybridCache("test-user")
	require.NoError(t, err)
	defer func() { _ = cache.Close() }()

	// Add various data
	_ = cache.SetRun(models.RunResponse{
		ID:     "test-run",
		Status: models.StatusDone,
	})
	_ = cache.SetFileHash("file.txt", "hash")
	_ = cache.SetUserInfo(&models.UserInfo{ID: 999})
	_ = cache.SetRepositoryList([]string{"repo1"})

	// Clear cache
	err = cache.Clear()
	assert.NoError(t, err)

	// Create new cache instance
	cache2, err := NewHybridCache("test-user")
	require.NoError(t, err)
	defer func() { _ = cache2.Close() }()

	// All data should be gone
	_, found := cache2.GetRun("test-run")
	assert.False(t, found)

	hash, found := cache2.GetFileHash("file.txt")
	assert.False(t, found)
	assert.Empty(t, hash)

	_, found = cache2.GetUserInfo()
	assert.False(t, found)

	_, found = cache2.GetRepositoryList()
	assert.False(t, found)
}

func TestHybridCache_GetStats(t *testing.T) {
	// Setup test directory
	tmpDir := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")

	cache, err := NewHybridCache("test-user")
	require.NoError(t, err)
	defer func() { _ = cache.Close() }()

	// Add data
	runs := []models.RunResponse{
		{ID: "active-1", Status: models.StatusProcessing, CreatedAt: time.Now()},                 // Recent active
		{ID: "active-2", Status: models.StatusQueued, CreatedAt: time.Now().Add(-1 * time.Hour)}, // Recent active
		{ID: "terminal-1", Status: models.StatusDone, CreatedAt: time.Now()},
		{ID: "terminal-2", Status: models.StatusFailed, CreatedAt: time.Now()},
	}
	_ = cache.SetRuns(runs)
	_ = cache.SetRepositoryList([]string{"repo1", "repo2", "repo3"})

	// Get stats
	stats := cache.GetStats()
	assert.Equal(t, 2, stats.PermanentRuns, "should have 2 permanent runs")
	assert.Equal(t, 2, stats.ActiveRuns, "should have 2 active runs")
	assert.Equal(t, 3, stats.Repositories, "should have 3 repositories")
}

func TestHybridCache_FallbackToSessionOnly(t *testing.T) {
	// This simulates when permanent cache fails to initialize
	cache := &HybridCache{
		session:   NewSessionCache(),
		permanent: nil, // Simulate failed permanent cache
		userID:    "test-user",
	}
	defer func() { _ = cache.Close() }()

	// Should still work with session-only
	run := models.RunResponse{
		ID:        "session-only",
		Status:    models.StatusProcessing,
		CreatedAt: time.Now(),
	}
	err := cache.SetRun(run)
	assert.NoError(t, err)

	cached, found := cache.GetRun("session-only")
	assert.True(t, found)
	assert.Equal(t, run.ID, cached.ID)

	// Terminal runs won't persist without permanent cache
	terminalRun := models.RunResponse{
		ID:        "terminal",
		Status:    models.StatusDone,
		CreatedAt: time.Now(),
	}
	err = cache.SetRun(terminalRun)
	assert.NoError(t, err)

	// Won't be found because session cache doesn't store terminal runs
	_, found = cache.GetRun("terminal")
	assert.False(t, found, "terminal run can't be stored without permanent cache")
}

func TestHybridCache_OldStuckRunRouting(t *testing.T) {
	// Setup test directory
	tmpDir := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")

	cache, err := NewHybridCache("test-user")
	require.NoError(t, err)
	defer func() { _ = cache.Close() }()

	// Old stuck run (should go to permanent)
	oldRun := models.RunResponse{
		ID:        "old-stuck",
		Status:    models.StatusProcessing,
		CreatedAt: time.Now().Add(-3 * time.Hour),
	}

	// Recent active run (should go to session)
	recentRun := models.RunResponse{
		ID:        "recent-active",
		Status:    models.StatusProcessing,
		CreatedAt: time.Now().Add(-30 * time.Minute),
	}

	// Set both runs
	err = cache.SetRun(oldRun)
	assert.NoError(t, err)
	err = cache.SetRun(recentRun)
	assert.NoError(t, err)

	// Both should be retrievable
	cached, found := cache.GetRun("old-stuck")
	assert.True(t, found, "old stuck run should be found")
	assert.Equal(t, oldRun.ID, cached.ID)

	cached, found = cache.GetRun("recent-active")
	assert.True(t, found, "recent active run should be found")
	assert.Equal(t, recentRun.ID, cached.ID)

	// Create new cache instance - old run should persist
	cache2, err := NewHybridCache("test-user")
	require.NoError(t, err)
	defer func() { _ = cache2.Close() }()

	// Old run should persist (from permanent cache)
	cached, found = cache2.GetRun("old-stuck")
	assert.True(t, found, "old stuck run should persist")

	// Recent run should not persist (was only in session cache)
	_, found = cache2.GetRun("recent-active")
	assert.False(t, found, "recent active run should not persist")
}
