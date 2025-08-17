// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later


package cache

import (
	"testing"

	"github.com/repobird/repobird-cli/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimpleCache_LastRepository(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create simple cache (it will use current user from global service)
	cache := NewSimpleCache()
	require.NotNil(t, cache)

	// Initially should not be found
	repo, found := cache.GetLastUsedRepository()
	assert.False(t, found)
	assert.Empty(t, repo)

	// Set last repository
	testRepo := "simple/test-repo"
	err := cache.SetLastUsedRepository(testRepo)
	require.NoError(t, err)

	// Should now be found
	repo, found = cache.GetLastUsedRepository()
	assert.True(t, found)
	assert.Equal(t, testRepo, repo)

	// Update repository
	newRepo := "simple/new-repo"
	err = cache.SetLastUsedRepository(newRepo)
	require.NoError(t, err)

	// Should return updated repository
	repo, found = cache.GetLastUsedRepository()
	assert.True(t, found)
	assert.Equal(t, newRepo, repo)
}

func TestSimpleCache_LastRepository_FallbackToRecentRun(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cache := NewSimpleCache()

	// Add some runs to cache
	runs := []models.RunResponse{
		{
			ID:         "run1",
			Repository: "recent/repo1",
			Status:     "completed",
		},
		{
			ID:         "run2",
			Repository: "recent/repo2",
			Status:     "running",
		},
	}
	cache.SetRuns(runs)

	// Initially no last repository is set
	_, found := cache.GetLastUsedRepository()
	assert.False(t, found)

	// After setting a run, we should be able to get runs
	cachedRuns := cache.GetRuns()
	assert.Len(t, cachedRuns, 2)

	// The most recent run's repository can be used as fallback
	if len(cachedRuns) > 0 {
		assert.Equal(t, "recent/repo1", cachedRuns[0].Repository)
	}
}

func TestSimpleCache_LastRepository_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create first cache instance
	cache1 := NewSimpleCache()

	// Set repository
	testRepo := "persist/test-repo"
	err := cache1.SetLastUsedRepository(testRepo)
	require.NoError(t, err)

	// Stop first cache
	cache1.Stop()

	// Create new cache instance (same user due to global service)
	cache2 := NewSimpleCache()

	// Should retrieve the same repository
	repo, found := cache2.GetLastUsedRepository()
	assert.True(t, found)
	assert.Equal(t, testRepo, repo)

	// Clean up
	cache2.Stop()
}

func TestSimpleCache_LastRepository_WithAnonymousUser(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create cache (will use anonymous if no user is set)
	cache := NewSimpleCache()

	// Should still work with anonymous user
	testRepo := "anon/test-repo"
	err := cache.SetLastUsedRepository(testRepo)
	require.NoError(t, err)

	repo, found := cache.GetLastUsedRepository()
	assert.True(t, found)
	assert.Equal(t, testRepo, repo)

	cache.Stop()
}

func TestSimpleCache_LastRepository_EmptyValue(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cache := NewSimpleCache()

	// Set empty repository
	err := cache.SetLastUsedRepository("")
	require.NoError(t, err)

	// Should retrieve empty string
	repo, found := cache.GetLastUsedRepository()
	assert.True(t, found)
	assert.Equal(t, "", repo)

	cache.Stop()
}

func TestSimpleCache_LastRepository_SpecialCharacters(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cache := NewSimpleCache()

	specialRepos := []string{
		"user/repo-with-dash",
		"user/repo_with_underscore",
		"UPPERCASE/REPO",
		"@scoped/package",
		"user/123",
	}

	for _, testRepo := range specialRepos {
		err := cache.SetLastUsedRepository(testRepo)
		require.NoError(t, err)

		repo, found := cache.GetLastUsedRepository()
		assert.True(t, found)
		assert.Equal(t, testRepo, repo, "Failed for repository: %s", testRepo)
	}

	cache.Stop()
}
