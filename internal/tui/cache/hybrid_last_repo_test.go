// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later


package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHybridCache_LastRepository(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create hybrid cache
	cache, err := NewHybridCache("hybrid-test-user")
	require.NoError(t, err)
	require.NotNil(t, cache)

	// Initially should not be found
	repo, found := cache.GetLastUsedRepository()
	assert.False(t, found)
	assert.Empty(t, repo)

	// Set last repository
	testRepo := "hybrid/test-repo"
	err = cache.SetLastUsedRepository(testRepo)
	require.NoError(t, err)

	// Should now be found
	repo, found = cache.GetLastUsedRepository()
	assert.True(t, found)
	assert.Equal(t, testRepo, repo)

	// Update repository
	newRepo := "hybrid/new-repo"
	err = cache.SetLastUsedRepository(newRepo)
	require.NoError(t, err)

	// Should return updated repository
	repo, found = cache.GetLastUsedRepository()
	assert.True(t, found)
	assert.Equal(t, newRepo, repo)
}

func TestHybridCache_LastRepository_WithoutPermanentCache(t *testing.T) {
	// Create hybrid cache without permanent cache (simulating error case)
	cache := &HybridCache{
		session: NewSessionCache(),
		userID:  "test-user",
		// permanent is nil
	}

	// Should handle gracefully when permanent cache is nil
	repo, found := cache.GetLastUsedRepository()
	assert.False(t, found)
	assert.Empty(t, repo)

	// Setting should not error when permanent cache is nil
	err := cache.SetLastUsedRepository("test/repo")
	assert.NoError(t, err)

	// Still should not find anything
	repo, found = cache.GetLastUsedRepository()
	assert.False(t, found)
	assert.Empty(t, repo)
}

func TestHybridCache_LastRepository_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create first hybrid cache instance
	cache1, err := NewHybridCache("persistence-user")
	require.NoError(t, err)

	// Set repository
	testRepo := "persistence/test-repo"
	err = cache1.SetLastUsedRepository(testRepo)
	require.NoError(t, err)

	// Close first cache
	err = cache1.Close()
	require.NoError(t, err)

	// Create new hybrid cache instance for same user
	cache2, err := NewHybridCache("persistence-user")
	require.NoError(t, err)

	// Should retrieve the same repository
	repo, found := cache2.GetLastUsedRepository()
	assert.True(t, found)
	assert.Equal(t, testRepo, repo)
}

func TestHybridCache_LastRepository_DifferentUsers(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create caches for different users
	cache1, err := NewHybridCache("user-a")
	require.NoError(t, err)

	cache2, err := NewHybridCache("user-b")
	require.NoError(t, err)

	// Set different repositories
	err = cache1.SetLastUsedRepository("user-a/repo")
	require.NoError(t, err)

	err = cache2.SetLastUsedRepository("user-b/repo")
	require.NoError(t, err)

	// Each should have their own repository
	repo1, found := cache1.GetLastUsedRepository()
	assert.True(t, found)
	assert.Equal(t, "user-a/repo", repo1)

	repo2, found := cache2.GetLastUsedRepository()
	assert.True(t, found)
	assert.Equal(t, "user-b/repo", repo2)
}

func TestHybridCache_LastRepository_AnonymousUser(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create cache with empty user ID (should become anonymous)
	cache, err := NewHybridCache("")
	require.NoError(t, err)

	// Set repository for anonymous user
	testRepo := "anonymous/test-repo"
	err = cache.SetLastUsedRepository(testRepo)
	require.NoError(t, err)

	// Should retrieve repository
	repo, found := cache.GetLastUsedRepository()
	assert.True(t, found)
	assert.Equal(t, testRepo, repo)
}
