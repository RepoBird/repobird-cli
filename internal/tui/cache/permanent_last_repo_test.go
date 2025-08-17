// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPermanentCache_LastRepository(t *testing.T) {
	// Create temp dir for test
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create cache for test user
	cache, err := NewPermanentCache("test-user")
	require.NoError(t, err)

	// Test setting and getting last repository
	testRepo := "myorg/myrepo"

	// Initially should not be found
	repo, found := cache.GetLastUsedRepository()
	assert.False(t, found)
	assert.Empty(t, repo)

	// Set last repository
	err = cache.SetLastUsedRepository(testRepo)
	require.NoError(t, err)

	// Should now be found
	repo, found = cache.GetLastUsedRepository()
	assert.True(t, found)
	assert.Equal(t, testRepo, repo)

	// Verify file exists
	expectedPath := filepath.Join(tmpDir, "repobird", "cache", "users", "test-user", "last-repository.json")
	_, err = os.Stat(expectedPath)
	assert.NoError(t, err)

	// Update to a different repository
	newRepo := "anotherorg/anotherrepo"
	err = cache.SetLastUsedRepository(newRepo)
	require.NoError(t, err)

	// Should return the new repository
	repo, found = cache.GetLastUsedRepository()
	assert.True(t, found)
	assert.Equal(t, newRepo, repo)
}

func TestPermanentCache_LastRepository_MultipleUsers(t *testing.T) {
	// Create temp dir for test
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create caches for different users
	cache1, err := NewPermanentCache("user1")
	require.NoError(t, err)

	cache2, err := NewPermanentCache("user2")
	require.NoError(t, err)

	// Set different repositories for each user
	err = cache1.SetLastUsedRepository("user1/repo1")
	require.NoError(t, err)

	err = cache2.SetLastUsedRepository("user2/repo2")
	require.NoError(t, err)

	// Each should retrieve their own repository
	repo1, found := cache1.GetLastUsedRepository()
	assert.True(t, found)
	assert.Equal(t, "user1/repo1", repo1)

	repo2, found := cache2.GetLastUsedRepository()
	assert.True(t, found)
	assert.Equal(t, "user2/repo2", repo2)
}

func TestPermanentCache_LastRepository_EmptyRepository(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cache, err := NewPermanentCache("test-user")
	require.NoError(t, err)

	// Setting empty repository should still work
	err = cache.SetLastUsedRepository("")
	require.NoError(t, err)

	// Should retrieve empty string
	repo, found := cache.GetLastUsedRepository()
	assert.True(t, found)
	assert.Equal(t, "", repo)
}

func TestPermanentCache_LastRepository_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create first cache instance and set repository
	cache1, err := NewPermanentCache("persist-user")
	require.NoError(t, err)

	testRepo := "persistent/repo"
	err = cache1.SetLastUsedRepository(testRepo)
	require.NoError(t, err)

	// Create new cache instance for same user
	cache2, err := NewPermanentCache("persist-user")
	require.NoError(t, err)

	// Should retrieve the same repository
	repo, found := cache2.GetLastUsedRepository()
	assert.True(t, found)
	assert.Equal(t, testRepo, repo)
}

func TestPermanentCache_LastRepository_FileFormat(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cache, err := NewPermanentCache("format-user")
	require.NoError(t, err)

	testRepo := "format/test-repo"
	beforeTime := time.Now()

	err = cache.SetLastUsedRepository(testRepo)
	require.NoError(t, err)

	afterTime := time.Now()

	// Read and verify the file format
	filePath := filepath.Join(tmpDir, "repobird", "cache", "users", "format-user", "last-repository.json")
	data, err := os.ReadFile(filePath)
	require.NoError(t, err)

	var fileContent struct {
		Repository string    `json:"repository"`
		UpdatedAt  time.Time `json:"updated_at"`
	}
	err = json.Unmarshal(data, &fileContent)
	require.NoError(t, err)

	assert.Equal(t, testRepo, fileContent.Repository)
	assert.True(t, fileContent.UpdatedAt.After(beforeTime) || fileContent.UpdatedAt.Equal(beforeTime))
	assert.True(t, fileContent.UpdatedAt.Before(afterTime) || fileContent.UpdatedAt.Equal(afterTime))
}

func TestPermanentCache_LastRepository_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cache, err := NewPermanentCache("concurrent-user")
	require.NoError(t, err)

	// Run concurrent writes
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			repo := fmt.Sprintf("repo-%d", n)
			_ = cache.SetLastUsedRepository(repo)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have some repository set (any of them is fine)
	repo, found := cache.GetLastUsedRepository()
	assert.True(t, found)
	assert.Contains(t, repo, "repo-")
}

func TestPermanentCache_LastRepository_AnonymousUser(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Test with anonymous user
	cache, err := NewPermanentCache("anonymous")
	require.NoError(t, err)

	testRepo := "anonymous/repo"
	err = cache.SetLastUsedRepository(testRepo)
	require.NoError(t, err)

	repo, found := cache.GetLastUsedRepository()
	assert.True(t, found)
	assert.Equal(t, testRepo, repo)

	// Verify file is in anonymous directory
	expectedPath := filepath.Join(tmpDir, "repobird", "cache", "anonymous", "last-repository.json")
	_, err = os.Stat(expectedPath)
	assert.NoError(t, err)
}

func TestPermanentCache_LastRepository_SpecialCharacters(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cache, err := NewPermanentCache("special-user")
	require.NoError(t, err)

	// Test with repository containing special characters
	specialRepos := []string{
		"user/repo-with-dash",
		"user/repo_with_underscore",
		"user/repo.with.dots",
		"UPPERCASE/REPO",
		"user/123-numeric",
		"@scoped/package",
	}

	for _, repo := range specialRepos {
		err = cache.SetLastUsedRepository(repo)
		require.NoError(t, err)

		retrieved, found := cache.GetLastUsedRepository()
		assert.True(t, found)
		assert.Equal(t, repo, retrieved, "Failed for repository: %s", repo)
	}
}
