package cache

import (
	"os"
	"path/filepath"
	"testing"

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