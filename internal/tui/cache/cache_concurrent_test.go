// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/repobird/repobird-cli/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNoCacheDeadlock tests that SetRepositoryData doesn't cause deadlocks
func TestNoCacheDeadlock(t *testing.T) {
	// Set up temporary config directory for test
	tmpDir := t.TempDir() // Automatically cleaned up
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cache := NewSimpleCache()
	defer cache.Stop()

	// Simulate dashboard loading pattern
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			repo := fmt.Sprintf("repo-%d", idx)
			runs := generateTestRuns(10)
			details := make(map[string]*models.RunResponse)
			for j := range runs {
				details[runs[j].ID] = &runs[j]
			}

			// This previously caused deadlock
			cache.SetRepositoryData(repo, convertToPointers(runs), details)
		}(i)
	}

	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success - no deadlock
	case <-time.After(5 * time.Second):
		t.Fatal("Deadlock detected - operations did not complete")
	}
}

// TestConcurrentGetSetRuns tests concurrent access to GetRuns and SetRuns
func TestConcurrentGetSetRuns(t *testing.T) {
	tmpDir := t.TempDir() // Automatically cleaned up
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cache := NewSimpleCache()
	defer cache.Stop()

	var wg sync.WaitGroup

	// Multiple writers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			runs := generateTestRuns(5)
			cache.SetRuns(runs)
		}(i)
	}

	// Multiple readers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = cache.GetRuns()
		}()
	}

	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success - no race conditions
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout - possible deadlock or race condition")
	}
}

// TestParallelCacheOperations tests multiple cache operations in parallel
func TestParallelCacheOperations(t *testing.T) {
	tmpDir := t.TempDir() // Automatically cleaned up
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cache := NewSimpleCache()
	defer cache.Stop()

	var wg sync.WaitGroup

	// Test all cache operations concurrently
	operations := []func(){
		// SetRun operations
		func() {
			run := generateTestRun("test-1", models.StatusProcessing)
			cache.SetRun(run)
		},
		// GetRun operations
		func() {
			_ = cache.GetRun("test-1")
		},
		// SetUserInfo operations
		func() {
			info := &models.UserInfo{
				ID:    1,
				Email: "test@example.com",
			}
			cache.SetUserInfo(info)
		},
		// GetUserInfo operations
		func() {
			_ = cache.GetUserInfo()
		},
		// SetFileHash operations
		func() {
			cache.SetFileHash("/path/to/file", "hash123")
		},
		// GetFileHash operations
		func() {
			_ = cache.GetFileHash("/path/to/file")
		},
		// SetDashboardCache operations
		func() {
			data := &DashboardData{
				Runs:        []models.RunResponse{generateTestRun("dash-1", models.StatusDone)},
				LastUpdated: time.Now(),
			}
			cache.SetDashboardCache(data)
		},
		// GetDashboardCache operations
		func() {
			_, _ = cache.GetDashboardCache()
		},
	}

	// Run each operation 100 times concurrently
	for _, op := range operations {
		for i := 0; i < 100; i++ {
			wg.Add(1)
			operation := op // Capture for closure
			go func() {
				defer wg.Done()
				operation()
			}()
		}
	}

	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success - all operations completed without deadlock
	case <-time.After(10 * time.Second):
		t.Fatal("Timeout - possible deadlock in parallel operations")
	}
}

// TestHybridCacheParallelFetch tests parallel fetching in HybridCache.GetRuns
func TestHybridCacheParallelFetch(t *testing.T) {
	tmpDir := t.TempDir() // Automatically cleaned up
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	hybrid, err := NewHybridCache("test-user")
	require.NoError(t, err)
	defer func() { _ = hybrid.Close() }()

	// Add some terminal runs to permanent cache
	for i := 0; i < 10; i++ {
		run := generateTestRun(fmt.Sprintf("perm-%d", i), models.StatusDone)
		err := hybrid.SetRun(run)
		assert.NoError(t, err)
	}

	// Add some active runs to session cache
	for i := 0; i < 10; i++ {
		run := generateTestRun(fmt.Sprintf("sess-%d", i), models.StatusProcessing)
		err := hybrid.SetRun(run)
		assert.NoError(t, err)
	}

	// Concurrent reads should not deadlock
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runs, found := hybrid.GetRuns()
			assert.True(t, found)
			assert.NotEmpty(t, runs)
		}()
	}

	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success - parallel fetch completed
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout - parallel fetch deadlocked")
	}
}

// TestAtomicFileWrites tests that PermanentCache handles concurrent writes safely
func TestAtomicFileWrites(t *testing.T) {
	tmpDir := t.TempDir() // Automatically cleaned up
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	perm, err := NewPermanentCache("test-user")
	require.NoError(t, err)
	defer func() { _ = perm.Close() }()

	var wg sync.WaitGroup

	// Concurrent writes to different files
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			run := generateTestRun(fmt.Sprintf("run-%d", idx), models.StatusDone)
			err := perm.SetRun(run)
			assert.NoError(t, err)
		}(i)
	}

	// Concurrent reads while writing
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, _ = perm.GetRun(fmt.Sprintf("run-%d", idx))
		}(i)
	}

	wg.Wait()

	// Verify all runs were written correctly
	allRuns, found := perm.GetAllRuns()
	assert.True(t, found)
	assert.Len(t, allRuns, 50)
}

// Helper functions

func generateTestRun(id string, status models.RunStatus) models.RunResponse {
	return models.RunResponse{
		ID:         id,
		Status:     status,
		Repository: "test/repo",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

func generateTestRuns(count int) []models.RunResponse {
	runs := make([]models.RunResponse, count)
	for i := 0; i < count; i++ {
		status := models.StatusProcessing
		switch i % 3 {
		case 0:
			status = models.StatusDone
		case 1:
			status = models.StatusFailed
		}
		runs[i] = generateTestRun(fmt.Sprintf("run-%d", i), status)
	}
	return runs
}

func convertToPointers(runs []models.RunResponse) []*models.RunResponse {
	pointers := make([]*models.RunResponse, len(runs))
	for i := range runs {
		pointers[i] = &runs[i]
	}
	return pointers
}
