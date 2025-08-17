// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later


package cache

import (
	"sync"
	"testing"
	"time"

	"github.com/repobird/repobird-cli/internal/models"
	"github.com/stretchr/testify/assert"
)

// initializeCacheForTesting creates a cache without persistent storage for testing
func initializeCacheForTesting() {
	globalCache = &GlobalCache{
		details:         make(map[string]*models.RunResponse),
		detailsAt:       make(map[string]time.Time),
		terminalDetails: make(map[string]*models.RunResponse),
		persistentCache: nil, // Disable persistent cache for tests
	}
}

func TestGetCachedList(t *testing.T) {
	// Reset global cache before each test
	ensureGlobalCache()

	t.Run("returns empty when cache is empty", func(t *testing.T) {
		runs, cached, _, _, _ := GetCachedList()
		assert.Empty(t, runs)
		assert.False(t, cached)
	})

	t.Run("returns cached list when available", func(t *testing.T) {
		expectedRuns := []models.RunResponse{
			{ID: "run-1", Status: models.StatusQueued},
			{ID: "run-2", Status: models.StatusProcessing},
		}

		SetCachedList(expectedRuns, nil)

		runs, cached, cachedAt, _, selectedIdx := GetCachedList()
		assert.Equal(t, expectedRuns, runs)
		assert.True(t, cached)
		assert.WithinDuration(t, time.Now(), cachedAt, 2*time.Second)
		assert.Equal(t, 0, selectedIdx)
	})

	t.Run("preserves selected index", func(t *testing.T) {
		runs := []models.RunResponse{
			{ID: "run-1", Status: models.StatusQueued},
		}

		globalCache.mu.Lock()
		globalCache.selectedIndex = 3
		globalCache.mu.Unlock()
		SetCachedList(runs, nil)

		_, _, _, _, selectedIdx := GetCachedList()
		assert.Equal(t, 3, selectedIdx)
	})
}

func TestSetCachedList(t *testing.T) {
	// Reset global cache before each test
	ensureGlobalCache()

	t.Run("sets cache with proper values", func(t *testing.T) {
		runs := []models.RunResponse{
			{ID: "run-1", Status: models.StatusQueued},
			{ID: "run-2", Status: models.StatusDone},
		}
		details := map[string]*models.RunResponse{
			"run-1": {ID: "run-1", Status: models.StatusQueued, Prompt: "Test"},
		}

		globalCache.mu.Lock()
		globalCache.selectedIndex = 1
		globalCache.mu.Unlock()
		SetCachedList(runs, details)

		globalCache.mu.RLock()
		defer globalCache.mu.RUnlock()

		assert.Equal(t, runs, globalCache.runs)
		assert.True(t, globalCache.cached)
		assert.NotNil(t, globalCache.details["run-1"])
		assert.Equal(t, 1, globalCache.selectedIndex)
	})

	t.Run("separates terminal and active runs in details", func(t *testing.T) {
		runs := []models.RunResponse{
			{ID: "run-1", Status: models.StatusDone},
			{ID: "run-2", Status: models.StatusProcessing},
			{ID: "run-3", Status: models.StatusFailed},
			{ID: "run-4", Status: models.StatusQueued},
		}
		details := map[string]*models.RunResponse{
			"run-1": {ID: "run-1", Status: models.StatusDone},
			"run-2": {ID: "run-2", Status: models.StatusProcessing},
			"run-3": {ID: "run-3", Status: models.StatusFailed},
			"run-4": {ID: "run-4", Status: models.StatusQueued},
		}

		SetCachedList(runs, details)

		globalCache.mu.RLock()
		defer globalCache.mu.RUnlock()

		// Terminal runs should be in terminalDetails
		assert.NotNil(t, globalCache.terminalDetails["run-1"])
		assert.NotNil(t, globalCache.terminalDetails["run-3"])

		// Active runs should be in details
		assert.NotNil(t, globalCache.details["run-2"])
		assert.NotNil(t, globalCache.details["run-4"])
	})
}

func TestAddCachedDetail(t *testing.T) {
	// Reset global cache before each test
	ensureGlobalCache()

	t.Run("adds active run detail to cache", func(t *testing.T) {
		run := &models.RunResponse{
			ID:     "run-1",
			Status: models.StatusProcessing,
			Prompt: "Test prompt",
		}

		AddCachedDetail(run.ID, run)

		globalCache.mu.RLock()
		defer globalCache.mu.RUnlock()

		cached, exists := globalCache.details["run-1"]
		assert.True(t, exists)
		assert.Equal(t, run, cached)
	})

	t.Run("adds terminal run to terminal cache", func(t *testing.T) {
		run := &models.RunResponse{
			ID:     "run-2",
			Status: models.StatusDone,
			Prompt: "Completed prompt",
		}

		AddCachedDetail(run.ID, run)

		globalCache.mu.RLock()
		defer globalCache.mu.RUnlock()

		cached, exists := globalCache.terminalDetails["run-2"]
		assert.True(t, exists)
		assert.Equal(t, run, cached)

		// Should not be in active details
		_, existsInActive := globalCache.details["run-2"]
		assert.False(t, existsInActive)
	})

	t.Run("moves run from active to terminal when status changes", func(t *testing.T) {
		// Add as active first
		run1 := &models.RunResponse{
			ID:     "run-3",
			Status: models.StatusProcessing,
		}
		AddCachedDetail("run-3", run1)

		// Verify it's in active cache
		globalCache.mu.RLock()
		_, existsInActive := globalCache.details["run-3"]
		globalCache.mu.RUnlock()
		assert.True(t, existsInActive)

		// Update to terminal status
		run2 := &models.RunResponse{
			ID:     "run-3",
			Status: models.StatusDone,
		}
		AddCachedDetail("run-3", run2)

		// Verify it moved to terminal cache
		globalCache.mu.RLock()
		_, existsInActive = globalCache.details["run-3"]
		_, existsInTerminal := globalCache.terminalDetails["run-3"]
		globalCache.mu.RUnlock()

		assert.False(t, existsInActive)
		assert.True(t, existsInTerminal)
	})
}

func TestGetSelectedIndex(t *testing.T) {
	ensureGlobalCache()

	t.Run("returns selected index", func(t *testing.T) {
		// Set some cached data first
		runs := []models.RunResponse{
			{ID: "run-1", Status: models.StatusQueued},
		}
		SetCachedList(runs, nil)

		// Set selected index
		SetSelectedIndex(5)

		_, _, _, _, index := GetCachedList()
		assert.Equal(t, 5, index)
	})
}

func TestSetSelectedIndex(t *testing.T) {
	ensureGlobalCache()

	t.Run("sets selected index", func(t *testing.T) {
		SetSelectedIndex(7)

		globalCache.mu.RLock()
		defer globalCache.mu.RUnlock()

		assert.Equal(t, 7, globalCache.selectedIndex)
	})
}

func TestFormDataOperations(t *testing.T) {
	ensureGlobalCache()

	t.Run("save and retrieve form data", func(t *testing.T) {
		formData := &FormData{
			Title:      "Test Title",
			Repository: "user/repo",
			Source:     "main",
			Target:     "feature",
			Prompt:     "Test prompt",
		}

		SaveFormData(formData)

		result := GetFormData()
		assert.Equal(t, formData, result)
	})

	t.Run("get nil form data initially", func(t *testing.T) {
		globalCache.mu.Lock()
		globalCache.formData = nil
		globalCache.mu.Unlock()

		result := GetFormData()
		assert.Nil(t, result)
	})

	t.Run("clear form data", func(t *testing.T) {
		// Set some data first
		SaveFormData(&FormData{Title: "Test"})

		// Clear it
		ClearFormData()

		// Verify it's cleared
		result := GetFormData()
		assert.Nil(t, result)
	})
}

func TestClearCache(t *testing.T) {
	ensureGlobalCache()

	t.Run("clears all cache data", func(t *testing.T) {
		// Setup cache with data
		runs := []models.RunResponse{
			{ID: "run-1", Status: models.StatusDone},
		}
		details := map[string]*models.RunResponse{
			"run-1": {ID: "run-1", Status: models.StatusDone},
		}
		globalCache.mu.Lock()
		globalCache.selectedIndex = 3
		globalCache.mu.Unlock()
		SetCachedList(runs, details)
		SaveFormData(&FormData{Title: "Test"})

		// Clear cache
		ClearCache()

		globalCache.mu.RLock()
		defer globalCache.mu.RUnlock()

		assert.Empty(t, globalCache.runs)
		assert.False(t, globalCache.cached)
		assert.Empty(t, globalCache.details)
		assert.Empty(t, globalCache.detailsAt)
		assert.Equal(t, 0, globalCache.selectedIndex)
		assert.Nil(t, globalCache.formData)
		// Terminal details should be cleared too
		assert.Empty(t, globalCache.terminalDetails)
	})
}

func TestClearActiveCache(t *testing.T) {
	ensureGlobalCache()

	t.Run("removes only active runs from cache", func(t *testing.T) {
		// Add mixed runs
		runs := []models.RunResponse{
			{ID: "run-1", Status: models.StatusProcessing},
			{ID: "run-2", Status: models.StatusQueued},
			{ID: "run-3", Status: models.StatusDone},
			{ID: "run-4", Status: models.StatusFailed},
		}
		details := map[string]*models.RunResponse{
			"run-1": {ID: "run-1", Status: models.StatusProcessing},
			"run-2": {ID: "run-2", Status: models.StatusQueued},
			"run-3": {ID: "run-3", Status: models.StatusDone},
			"run-4": {ID: "run-4", Status: models.StatusFailed},
		}
		globalCache.mu.Lock()
		globalCache.selectedIndex = 2
		globalCache.mu.Unlock()
		SetCachedList(runs, details)

		// Clear active cache
		ClearActiveCache()

		globalCache.mu.RLock()
		defer globalCache.mu.RUnlock()

		// List should be cleared
		assert.Nil(t, globalCache.runs)
		assert.False(t, globalCache.cached)

		// Active details should be cleared
		assert.Empty(t, globalCache.details)
		assert.Empty(t, globalCache.detailsAt)

		// Terminal details should be preserved
		assert.NotEmpty(t, globalCache.terminalDetails)
	})
}

func TestCacheConcurrency(t *testing.T) {
	ensureGlobalCache()

	t.Run("concurrent reads and writes to list", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 100

		// Start multiple goroutines writing to cache
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				runs := []models.RunResponse{
					{ID: string(rune(id)), Status: models.StatusProcessing},
				}
				SetCachedList(runs, nil)
			}(i)
		}

		// Start multiple goroutines reading from cache
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _, _, _, _ = GetCachedList()
			}()
		}

		// Wait for all goroutines to complete
		wg.Wait()

		// Should not have panicked
		assert.True(t, true, "Concurrent operations completed without panic")
	})

	t.Run("concurrent detail operations", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 100

		// Start multiple goroutines adding details
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				run := &models.RunResponse{
					ID:     string(rune(id)),
					Status: models.StatusProcessing,
				}
				AddCachedDetail(run.ID, run)
			}(i)
		}

		// Start multiple goroutines clearing cache
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				time.Sleep(time.Millisecond * 10)
				ClearCache()
			}()
		}

		// Wait for all goroutines to complete
		wg.Wait()

		// Should not have panicked
		assert.True(t, true, "Concurrent detail operations completed without panic")
	})

	t.Run("concurrent form data operations", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 50

		// Start multiple goroutines saving form data
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				data := &FormData{
					Title: string(rune(id)),
				}
				SaveFormData(data)
			}(i)
		}

		// Start multiple goroutines reading form data
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = GetFormData()
			}()
		}

		// Start multiple goroutines clearing form data
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				time.Sleep(time.Millisecond * 5)
				ClearFormData()
			}()
		}

		// Wait for all goroutines to complete
		wg.Wait()

		// Should not have panicked
		assert.True(t, true, "Concurrent form data operations completed without panic")
	})
}

func TestIsTerminalStatus(t *testing.T) {
	tests := []struct {
		status     models.RunStatus
		isTerminal bool
	}{
		{models.StatusDone, true},
		{models.StatusFailed, true},
		{models.StatusProcessing, false},
		{models.StatusQueued, false},
		{models.StatusInitializing, false},
		{models.StatusPostProcess, false},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := isTerminalStatus(tt.status)
			assert.Equal(t, tt.isTerminal, result)
		})
	}
}

func TestRepositoryHistory(t *testing.T) {
	t.Run("returns empty history when cache is empty", func(t *testing.T) {
		// Use test-specific cache initialization
		initializeCacheForTesting()
		history, err := GetRepositoryHistory()
		assert.NoError(t, err)
		assert.Empty(t, history)
	})

	t.Run("adds repository to history", func(t *testing.T) {
		initializeCacheForTesting()
		err := AddRepositoryToHistory("owner/repo1")
		assert.NoError(t, err)

		history, err := GetRepositoryHistory()
		assert.NoError(t, err)
		assert.Equal(t, []string{"owner/repo1"}, history)
	})

	t.Run("moves existing repository to front", func(t *testing.T) {
		initializeCacheForTesting()
		// Add multiple repositories
		_ = AddRepositoryToHistory("owner/repo1")
		_ = AddRepositoryToHistory("owner/repo2")
		_ = AddRepositoryToHistory("owner/repo3")

		// Add repo1 again - should move to front
		_ = AddRepositoryToHistory("owner/repo1")

		history, err := GetRepositoryHistory()
		assert.NoError(t, err)
		assert.Equal(t, []string{"owner/repo1", "owner/repo3", "owner/repo2"}, history)
	})

	t.Run("gets most recent repository", func(t *testing.T) {
		initializeCacheForTesting()
		_ = AddRepositoryToHistory("owner/repo1")
		_ = AddRepositoryToHistory("owner/repo2")

		mostRecent, err := GetMostRecentRepository()
		assert.NoError(t, err)
		assert.Equal(t, "owner/repo2", mostRecent)
	})

	t.Run("handles empty string repositories", func(t *testing.T) {
		initializeCacheForTesting()
		err := AddRepositoryToHistory("")
		assert.NoError(t, err) // Should not error, but also should not add

		err = AddRepositoryToHistory("owner/repo1")
		assert.NoError(t, err)

		history, err := GetRepositoryHistory()
		assert.NoError(t, err)
		assert.Equal(t, []string{"owner/repo1"}, history)
	})
}
