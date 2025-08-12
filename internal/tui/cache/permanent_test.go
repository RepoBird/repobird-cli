package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/repobird/repobird-cli/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPermanentCache_OnlyStoresTerminalRuns(t *testing.T) {
	// Setup test directory
	tmpDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")
	
	cache, err := NewPermanentCache("test-user-123")
	require.NoError(t, err)
	
	// Should not store active run
	activeRun := models.RunResponse{
		ID:     "test-1",
		Status: models.StatusProcessing,
	}
	err = cache.SetRun(activeRun)
	assert.NoError(t, err)
	
	_, found := cache.GetRun("test-1")
	assert.False(t, found, "active run should not be cached")
	
	// Should store terminal run (DONE)
	terminalRun := models.RunResponse{
		ID:        "test-2",
		Status:    models.StatusDone,
		CreatedAt: time.Now(),
	}
	err = cache.SetRun(terminalRun)
	assert.NoError(t, err)
	
	cached, found := cache.GetRun("test-2")
	assert.True(t, found, "terminal run should be cached")
	assert.Equal(t, terminalRun.ID, cached.ID)
	assert.Equal(t, terminalRun.Status, cached.Status)
	
	// Should store terminal run (FAILED)
	failedRun := models.RunResponse{
		ID:        "test-3",
		Status:    models.StatusFailed,
		CreatedAt: time.Now(),
	}
	err = cache.SetRun(failedRun)
	assert.NoError(t, err)
	
	cached, found = cache.GetRun("test-3")
	assert.True(t, found, "failed run should be cached")
	assert.Equal(t, failedRun.Status, cached.Status)
}

func TestPermanentCache_UserSeparation(t *testing.T) {
	// Setup test directory
	tmpDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")
	
	// Create caches for different users
	cache1, err := NewPermanentCache("user-1")
	require.NoError(t, err)
	
	cache2, err := NewPermanentCache("user-2")
	require.NoError(t, err)
	
	// Add run to user 1's cache
	run1 := models.RunResponse{
		ID:        "run-1",
		Status:    models.StatusDone,
		CreatedAt: time.Now(),
	}
	err = cache1.SetRun(run1)
	require.NoError(t, err)
	
	// User 2 should not see user 1's run
	_, found := cache2.GetRun("run-1")
	assert.False(t, found, "user 2 should not see user 1's run")
	
	// User 1 should see their own run
	cached, found := cache1.GetRun("run-1")
	assert.True(t, found, "user 1 should see their own run")
	assert.Equal(t, run1.ID, cached.ID)
}

func TestPermanentCache_FileHashStorage(t *testing.T) {
	// Setup test directory
	tmpDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")
	
	cache, err := NewPermanentCache("test-user")
	require.NoError(t, err)
	
	// Store file hashes
	err = cache.SetFileHash("file1.txt", "hash1")
	assert.NoError(t, err)
	
	err = cache.SetFileHash("file2.txt", "hash2")
	assert.NoError(t, err)
	
	// Retrieve individual hash
	hash, found := cache.GetFileHash("file1.txt")
	assert.True(t, found)
	assert.Equal(t, "hash1", hash)
	
	// Get all hashes
	allHashes := cache.GetAllFileHashes()
	assert.Len(t, allHashes, 2)
	assert.Equal(t, "hash1", allHashes["file1.txt"])
	assert.Equal(t, "hash2", allHashes["file2.txt"])
	
	// Test persistence - create new cache instance
	cache2, err := NewPermanentCache("test-user")
	require.NoError(t, err)
	
	hash, found = cache2.GetFileHash("file1.txt")
	assert.True(t, found, "hash should persist across cache instances")
	assert.Equal(t, "hash1", hash)
}

func TestPermanentCache_UserInfo(t *testing.T) {
	// Setup test directory
	tmpDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")
	
	cache, err := NewPermanentCache("test-user")
	require.NoError(t, err)
	
	// Store user info
	userInfo := &models.UserInfo{
		ID:    123,
		Email: "test@example.com",
		Name:  "Test User",
	}
	err = cache.SetUserInfo(userInfo)
	assert.NoError(t, err)
	
	// Retrieve user info
	cached, found := cache.GetUserInfo()
	assert.True(t, found)
	assert.Equal(t, userInfo.ID, cached.ID)
	assert.Equal(t, userInfo.Email, cached.Email)
	assert.Equal(t, userInfo.Name, cached.Name)
	
	// Test persistence
	cache2, err := NewPermanentCache("test-user")
	require.NoError(t, err)
	
	cached, found = cache2.GetUserInfo()
	assert.True(t, found, "user info should persist")
	assert.Equal(t, userInfo.ID, cached.ID)
}

func TestPermanentCache_RepositoryList(t *testing.T) {
	// Setup test directory
	tmpDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")
	
	cache, err := NewPermanentCache("test-user")
	require.NoError(t, err)
	
	// Store repository list
	repos := []string{"repo1", "repo2", "repo3"}
	err = cache.SetRepositoryList(repos)
	assert.NoError(t, err)
	
	// Retrieve repository list
	cached, found := cache.GetRepositoryList()
	assert.True(t, found)
	assert.Equal(t, repos, cached)
	
	// Test persistence
	cache2, err := NewPermanentCache("test-user")
	require.NoError(t, err)
	
	cached, found = cache2.GetRepositoryList()
	assert.True(t, found, "repository list should persist")
	assert.Equal(t, repos, cached)
}

func TestPermanentCache_Clear(t *testing.T) {
	// Setup test directory
	tmpDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")
	
	cache, err := NewPermanentCache("test-user")
	require.NoError(t, err)
	
	// Add some data
	run := models.RunResponse{
		ID:     "test-run",
		Status: models.StatusDone,
	}
	_ = cache.SetRun(run)
	_ = cache.SetFileHash("file.txt", "hash")
	
	// Verify data exists
	_, found := cache.GetRun("test-run")
	assert.True(t, found)
	
	// Clear cache
	err = cache.Clear()
	assert.NoError(t, err)
	
	// Create new cache instance
	cache2, err := NewPermanentCache("test-user")
	require.NoError(t, err)
	
	// Verify data is gone
	_, found = cache2.GetRun("test-run")
	assert.False(t, found, "run should be cleared")
	
	hash, found := cache2.GetFileHash("file.txt")
	assert.False(t, found, "file hash should be cleared")
	assert.Empty(t, hash)
}

func TestPermanentCache_DirectoryStructure(t *testing.T) {
	// Setup test directory
	tmpDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")
	
	// Create cache for specific user
	cache, err := NewPermanentCache("user-123")
	require.NoError(t, err)
	
	// Add various data types
	run := models.RunResponse{
		ID:     "run-abc",
		Status: models.StatusDone,
	}
	_ = cache.SetRun(run)
	_ = cache.SetRepositoryList([]string{"repo1"})
	_ = cache.SetFileHash("file.txt", "hash123")
	
	// Verify directory structure
	userDir := filepath.Join(tmpDir, "repobird", "cache", "users")
	entries, err := os.ReadDir(userDir)
	require.NoError(t, err)
	assert.Len(t, entries, 1, "should have one user directory")
	
	// Check runs directory exists
	userHash := hashUserID("user-123")
	runFile := filepath.Join(userDir, userHash, "runs", "run-abc.json")
	_, err = os.Stat(runFile)
	assert.NoError(t, err, "run file should exist")
	
	// Check file hashes exist
	hashFile := filepath.Join(userDir, userHash, "file-hashes.json")
	_, err = os.Stat(hashFile)
	assert.NoError(t, err, "file hashes should exist")
}

func TestPermanentCache_AnonymousUser(t *testing.T) {
	// Setup test directory
	tmpDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")
	
	// Create cache for anonymous user
	cache, err := NewPermanentCache("")
	require.NoError(t, err)
	
	// Should still work for anonymous users
	run := models.RunResponse{
		ID:     "anon-run",
		Status: models.StatusDone,
	}
	err = cache.SetRun(run)
	assert.NoError(t, err)
	
	cached, found := cache.GetRun("anon-run")
	assert.True(t, found)
	assert.Equal(t, run.ID, cached.ID)
	
	// Check directory is created as "anonymous"
	anonDir := filepath.Join(tmpDir, "repobird", "cache", "users", "anonymous")
	_, err = os.Stat(anonDir)
	assert.NoError(t, err, "anonymous directory should exist")
}