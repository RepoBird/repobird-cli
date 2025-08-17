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

// TestHybridCacheParallelFetchNoDeadlock tests that parallel GetRuns doesn't deadlock
func TestHybridCacheParallelFetchNoDeadlock(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	hybrid, err := NewHybridCache("test-user")
	require.NoError(t, err)
	defer func() { _ = hybrid.Close() }()

	// Add runs to both caches
	for i := 0; i < 50; i++ {
		status := models.StatusProcessing
		createdAt := time.Now()

		if i%2 == 0 {
			status = models.StatusDone // Terminal -> permanent
		} else if i%3 == 0 {
			createdAt = time.Now().Add(-3 * time.Hour) // Old -> permanent
		}

		run := models.RunResponse{
			ID:        fmt.Sprintf("run-%d", i),
			Status:    status,
			CreatedAt: createdAt,
			UpdatedAt: createdAt,
		}
		_ = hybrid.SetRun(run)
	}

	// Launch many concurrent GetRuns
	var wg sync.WaitGroup
	successCount := int32(0)

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runs, found := hybrid.GetRuns()
			if found && len(runs) > 0 {
				atomic.AddInt32(&successCount, 1)
			}
		}()
	}

	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		count := atomic.LoadInt32(&successCount)
		assert.Equal(t, int32(200), count, "All GetRuns should succeed")
	case <-time.After(3 * time.Second):
		t.Fatal("Parallel GetRuns deadlocked")
	}
}

// TestHybridCacheRouterStress tests routing under stress
func TestHybridCacheRouterStress(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	hybrid, err := NewHybridCache("test-user")
	require.NoError(t, err)
	defer func() { _ = hybrid.Close() }()

	var wg sync.WaitGroup
	numGoroutines := 100
	runsPerGoroutine := 10

	// Track routing decisions
	var terminalCount int32
	var activeCount int32
	var oldCount int32

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(gid int) {
			defer wg.Done()

			for j := 0; j < runsPerGoroutine; j++ {
				runID := fmt.Sprintf("g%d-r%d", gid, j)

				var run models.RunResponse
				switch j % 3 {
				case 0:
					// Terminal run
					run = models.RunResponse{
						ID:        runID,
						Status:    models.StatusDone,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}
					atomic.AddInt32(&terminalCount, 1)
				case 1:
					// Active run
					run = models.RunResponse{
						ID:        runID,
						Status:    models.StatusProcessing,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}
					atomic.AddInt32(&activeCount, 1)
				case 2:
					// Old stuck run
					run = models.RunResponse{
						ID:        runID,
						Status:    models.StatusProcessing,
						CreatedAt: time.Now().Add(-3 * time.Hour),
						UpdatedAt: time.Now().Add(-3 * time.Hour),
					}
					atomic.AddInt32(&oldCount, 1)
				}

				err := hybrid.SetRun(run)
				assert.NoError(t, err)

				// Immediately try to get it back
				retrieved, found := hybrid.GetRun(runID)
				assert.True(t, found, "Run %s should be found", runID)
				assert.Equal(t, runID, retrieved.ID)
			}
		}(i)
	}

	wg.Wait()

	t.Logf("Routing stats: Terminal=%d, Active=%d, Old=%d",
		terminalCount, activeCount, oldCount)

	// Verify totals
	totalExpected := numGoroutines * runsPerGoroutine
	totalRouted := int(terminalCount + activeCount + oldCount)
	assert.Equal(t, totalExpected, totalRouted)
}

// TestHybridCacheSetRunsParallelRouting tests parallel routing in SetRuns
func TestHybridCacheSetRunsParallelRouting(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	hybrid, err := NewHybridCache("test-user")
	require.NoError(t, err)
	defer func() { _ = hybrid.Close() }()

	// Create large batch of mixed runs
	runs := make([]models.RunResponse, 500)
	for i := 0; i < 500; i++ {
		status := models.StatusProcessing
		createdAt := time.Now()

		if i%3 == 0 {
			status = models.StatusDone
		} else if i%3 == 1 {
			status = models.StatusFailed
		} else if i%5 == 0 {
			createdAt = time.Now().Add(-3 * time.Hour)
		}

		runs[i] = models.RunResponse{
			ID:        fmt.Sprintf("batch-%d", i),
			Status:    status,
			CreatedAt: createdAt,
			UpdatedAt: createdAt,
		}
	}

	// Time the operation
	start := time.Now()
	err = hybrid.SetRuns(runs)
	duration := time.Since(start)

	assert.NoError(t, err)
	t.Logf("SetRuns with 500 runs took: %v", duration)

	// Verify all runs are accessible
	for i, run := range runs {
		retrieved, found := hybrid.GetRun(run.ID)
		assert.True(t, found, "Run %s at index %d should be found", run.ID, i)
		assert.Equal(t, run.ID, retrieved.ID)
	}
}

// TestHybridCacheConcurrentInvalidation tests concurrent invalidation
func TestHybridCacheConcurrentInvalidation(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	hybrid, err := NewHybridCache("test-user")
	require.NoError(t, err)
	defer func() { _ = hybrid.Close() }()

	// Add runs
	numRuns := 100
	for i := 0; i < numRuns; i++ {
		status := models.StatusProcessing
		if i%2 == 0 {
			status = models.StatusDone
		}

		run := models.RunResponse{
			ID:        fmt.Sprintf("inv-%d", i),
			Status:    status,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		_ = hybrid.SetRun(run)
	}

	var wg sync.WaitGroup

	// Concurrent invalidations
	for i := 0; i < numRuns; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_ = hybrid.InvalidateRun(fmt.Sprintf("inv-%d", idx))
		}(i)
	}

	// Concurrent reads while invalidating
	for i := 0; i < numRuns; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, _ = hybrid.GetRun(fmt.Sprintf("inv-%d", idx))
		}(i)
	}

	wg.Wait()

	// All runs should be invalidated
	for i := 0; i < numRuns; i++ {
		_, found := hybrid.GetRun(fmt.Sprintf("inv-%d", i))
		assert.False(t, found, "Run inv-%d should be invalidated", i)
	}
}

// TestHybridCacheMixedOperationsStress tests all operations under stress
func TestHybridCacheMixedOperationsStress(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	hybrid, err := NewHybridCache("test-user")
	require.NoError(t, err)
	defer func() { _ = hybrid.Close() }()

	stopCh := make(chan struct{})
	var wg sync.WaitGroup

	// Operation counters
	var setRunOps int32
	var getRunOps int32
	var setRunsOps int32
	var getRunsOps int32
	var invalidateOps int32
	var userInfoOps int32
	var fileHashOps int32

	// SetRun worker
	wg.Add(1)
	go func() {
		defer wg.Done()
		i := 0
		for {
			select {
			case <-stopCh:
				return
			default:
				run := models.RunResponse{
					ID:        fmt.Sprintf("stress-set-%d", i),
					Status:    models.StatusProcessing,
					CreatedAt: time.Now(),
				}
				_ = hybrid.SetRun(run)
				atomic.AddInt32(&setRunOps, 1)
				i++
			}
		}
	}()

	// GetRun worker
	wg.Add(1)
	go func() {
		defer wg.Done()
		i := 0
		for {
			select {
			case <-stopCh:
				return
			default:
				_, _ = hybrid.GetRun(fmt.Sprintf("stress-set-%d", i))
				atomic.AddInt32(&getRunOps, 1)
				i++
			}
		}
	}()

	// SetRuns worker
	wg.Add(1)
	go func() {
		defer wg.Done()
		batch := 0
		for {
			select {
			case <-stopCh:
				return
			default:
				runs := []models.RunResponse{
					{ID: fmt.Sprintf("batch-%d-1", batch), Status: models.StatusDone},
					{ID: fmt.Sprintf("batch-%d-2", batch), Status: models.StatusProcessing},
				}
				_ = hybrid.SetRuns(runs)
				atomic.AddInt32(&setRunsOps, 1)
				batch++
			}
		}
	}()

	// GetRuns worker
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stopCh:
				return
			default:
				_, _ = hybrid.GetRuns()
				atomic.AddInt32(&getRunsOps, 1)
			}
		}
	}()

	// InvalidateRun worker
	wg.Add(1)
	go func() {
		defer wg.Done()
		i := 0
		for {
			select {
			case <-stopCh:
				return
			default:
				_ = hybrid.InvalidateRun(fmt.Sprintf("stress-set-%d", i))
				atomic.AddInt32(&invalidateOps, 1)
				i++
			}
		}
	}()

	// UserInfo worker
	wg.Add(1)
	go func() {
		defer wg.Done()
		i := 0
		for {
			select {
			case <-stopCh:
				return
			default:
				if i%2 == 0 {
					_ = hybrid.SetUserInfo(&models.UserInfo{ID: i})
				} else {
					_, _ = hybrid.GetUserInfo()
				}
				atomic.AddInt32(&userInfoOps, 1)
				i++
			}
		}
	}()

	// FileHash worker
	wg.Add(1)
	go func() {
		defer wg.Done()
		i := 0
		for {
			select {
			case <-stopCh:
				return
			default:
				path := fmt.Sprintf("/file%d.txt", i)
				if i%2 == 0 {
					_ = hybrid.SetFileHash(path, fmt.Sprintf("hash%d", i))
				} else {
					_, _ = hybrid.GetFileHash(path)
				}
				atomic.AddInt32(&fileHashOps, 1)
				i++
			}
		}
	}()

	// Run for 2 seconds
	time.Sleep(2 * time.Second)
	close(stopCh)
	wg.Wait()

	// Report statistics
	t.Logf("Stress test operations in 2 seconds:")
	t.Logf("  SetRun: %d", atomic.LoadInt32(&setRunOps))
	t.Logf("  GetRun: %d", atomic.LoadInt32(&getRunOps))
	t.Logf("  SetRuns: %d", atomic.LoadInt32(&setRunsOps))
	t.Logf("  GetRuns: %d", atomic.LoadInt32(&getRunsOps))
	t.Logf("  Invalidate: %d", atomic.LoadInt32(&invalidateOps))
	t.Logf("  UserInfo: %d", atomic.LoadInt32(&userInfoOps))
	t.Logf("  FileHash: %d", atomic.LoadInt32(&fileHashOps))

	totalOps := atomic.LoadInt32(&setRunOps) + atomic.LoadInt32(&getRunOps) +
		atomic.LoadInt32(&setRunsOps) + atomic.LoadInt32(&getRunsOps) +
		atomic.LoadInt32(&invalidateOps) + atomic.LoadInt32(&userInfoOps) +
		atomic.LoadInt32(&fileHashOps)

	assert.Greater(t, totalOps, int32(100), "Should complete many operations")
}
