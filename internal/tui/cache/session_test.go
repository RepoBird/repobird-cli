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

// TestSessionCacheNoMutex verifies SessionCache works without extra mutex
func TestSessionCacheNoMutex(t *testing.T) {
	session := NewSessionCache()
	defer session.Close()

	// Concurrent operations that would reveal mutex issues
	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Mix of operations
			run := models.RunResponse{
				ID:        fmt.Sprintf("sess-%d", id),
				Status:    models.StatusProcessing,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			// Set and get operations
			_ = session.SetRun(run)
			_, _ = session.GetRun(fmt.Sprintf("sess-%d", id))

			// Form data operations
			_ = session.SetFormData(fmt.Sprintf("form-%d", id), "data")
			_, _ = session.GetFormData(fmt.Sprintf("form-%d", id))
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
	case <-time.After(3 * time.Second):
		t.Fatal("Operations blocked - ttlcache should handle concurrency")
	}
}

// TestSessionCacheActiveRunsOnly verifies session cache only stores active runs
func TestSessionCacheActiveRunsOnly(t *testing.T) {
	session := NewSessionCache()
	defer session.Close()

	// Try to store terminal run
	terminalRun := models.RunResponse{
		ID:        "terminal-1",
		Status:    models.StatusDone,
		CreatedAt: time.Now(),
	}

	err := session.SetRun(terminalRun)
	assert.NoError(t, err)

	// Should not be retrievable (filtered out)
	retrieved, found := session.GetRun("terminal-1")
	assert.False(t, found)
	assert.Nil(t, retrieved)

	// Store active run
	activeRun := models.RunResponse{
		ID:        "active-1",
		Status:    models.StatusProcessing,
		CreatedAt: time.Now(),
	}

	err = session.SetRun(activeRun)
	assert.NoError(t, err)

	// Should be retrievable
	retrieved, found = session.GetRun("active-1")
	assert.True(t, found)
	assert.Equal(t, activeRun.ID, retrieved.ID)
}

// TestSessionCacheOldRunFiltering verifies old runs are filtered out
func TestSessionCacheOldRunFiltering(t *testing.T) {
	session := NewSessionCache()
	defer session.Close()

	// Old run (should be filtered)
	oldRun := models.RunResponse{
		ID:        "old-1",
		Status:    models.StatusProcessing,
		CreatedAt: time.Now().Add(-3 * time.Hour),
		UpdatedAt: time.Now().Add(-3 * time.Hour),
	}

	err := session.SetRun(oldRun)
	assert.NoError(t, err)

	// Should not be stored (old runs go to permanent)
	retrieved, found := session.GetRun("old-1")
	assert.False(t, found)
	assert.Nil(t, retrieved)
}

// TestSessionCacheTTL verifies TTL expiration
func TestSessionCacheTTL(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TTL test in short mode")
	}

	session := NewSessionCache()
	defer session.Close()

	// Store run with 5-minute TTL
	run := models.RunResponse{
		ID:        "ttl-test",
		Status:    models.StatusProcessing,
		CreatedAt: time.Now(),
	}

	err := session.SetRun(run)
	assert.NoError(t, err)

	// Should be retrievable immediately
	retrieved, found := session.GetRun("ttl-test")
	assert.True(t, found)
	assert.NotNil(t, retrieved)

	// Note: In production, items expire after 5 minutes
	// For testing, we'd need to mock time or use shorter TTL
}

// TestSessionCacheConcurrentSetRuns tests concurrent SetRuns operations
func TestSessionCacheConcurrentSetRuns(t *testing.T) {
	session := NewSessionCache()
	defer session.Close()

	var wg sync.WaitGroup
	numBatches := 50
	runsPerBatch := 10

	for i := 0; i < numBatches; i++ {
		wg.Add(1)
		go func(batchID int) {
			defer wg.Done()

			runs := make([]models.RunResponse, runsPerBatch)
			for j := 0; j < runsPerBatch; j++ {
				runs[j] = models.RunResponse{
					ID:        fmt.Sprintf("batch-%d-run-%d", batchID, j),
					Status:    models.StatusProcessing,
					CreatedAt: time.Now(),
				}
			}

			err := session.SetRuns(runs)
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()

	// Verify some runs are stored
	allRuns, found := session.GetRuns()
	assert.True(t, found)
	assert.NotEmpty(t, allRuns)
}

// TestSessionCacheInvalidateRun tests run invalidation
func TestSessionCacheInvalidateRun(t *testing.T) {
	session := NewSessionCache()
	defer session.Close()

	// Add run
	run := models.RunResponse{
		ID:        "invalidate-test",
		Status:    models.StatusProcessing,
		CreatedAt: time.Now(),
	}

	_ = session.SetRun(run)

	// Verify it exists
	_, found := session.GetRun("invalidate-test")
	assert.True(t, found)

	// Invalidate
	err := session.InvalidateRun("invalidate-test")
	assert.NoError(t, err)

	// Should be gone
	_, found = session.GetRun("invalidate-test")
	assert.False(t, found)
}

// TestSessionCacheInvalidateActiveRuns tests clearing all active runs
func TestSessionCacheInvalidateActiveRuns(t *testing.T) {
	session := NewSessionCache()
	defer session.Close()

	// Add multiple runs
	for i := 0; i < 10; i++ {
		run := models.RunResponse{
			ID:        fmt.Sprintf("active-%d", i),
			Status:    models.StatusProcessing,
			CreatedAt: time.Now(),
		}
		_ = session.SetRun(run)
	}

	// Verify they exist
	runs, found := session.GetRuns()
	assert.True(t, found)
	assert.Len(t, runs, 10)

	// Invalidate all active runs
	err := session.InvalidateActiveRuns()
	assert.NoError(t, err)

	// Should be empty
	runs, found = session.GetRuns()
	assert.False(t, found)
	assert.Empty(t, runs)
}

// TestSessionCacheFormData tests form data caching
func TestSessionCacheFormData(t *testing.T) {
	session := NewSessionCache()
	defer session.Close()

	// Store form data
	formData := map[string]string{
		"field1": "value1",
		"field2": "value2",
	}

	err := session.SetFormData("test-form", formData)
	assert.NoError(t, err)

	// Retrieve form data
	retrieved, found := session.GetFormData("test-form")
	assert.True(t, found)

	data, ok := retrieved.(map[string]string)
	require.True(t, ok)
	assert.Equal(t, formData, data)
}

// TestSessionCacheDashboardData tests dashboard data caching
func TestSessionCacheDashboardData(t *testing.T) {
	session := NewSessionCache()
	defer session.Close()

	// Store dashboard data
	dashData := &DashboardData{
		Runs: []models.RunResponse{
			{ID: "dash-1", Status: models.StatusProcessing},
		},
		UserInfo:    &models.UserInfo{ID: 123},
		LastUpdated: time.Now(),
	}

	err := session.SetDashboardData(dashData)
	assert.NoError(t, err)

	// Retrieve dashboard data
	retrieved, found := session.GetDashboardData()
	assert.True(t, found)
	assert.NotNil(t, retrieved)
	assert.Len(t, retrieved.Runs, 1)
	assert.Equal(t, 123, retrieved.UserInfo.ID)
}

// TestSessionCacheClear tests clearing all data
func TestSessionCacheClear(t *testing.T) {
	session := NewSessionCache()
	defer session.Close()

	// Add various data
	_ = session.SetRun(models.RunResponse{
		ID:        "clear-test",
		Status:    models.StatusProcessing,
		CreatedAt: time.Now(),
	})
	_ = session.SetFormData("form", "data")
	_ = session.SetDashboardData(&DashboardData{})

	// Clear all
	err := session.Clear()
	assert.NoError(t, err)

	// Verify everything is gone
	_, found := session.GetRun("clear-test")
	assert.False(t, found)

	_, found = session.GetFormData("form")
	assert.False(t, found)

	_, found = session.GetDashboardData()
	assert.False(t, found)
}

// TestSessionCacheConcurrentStress stress tests all operations
func TestSessionCacheConcurrentStress(t *testing.T) {
	session := NewSessionCache()
	defer session.Close()

	stopCh := make(chan struct{})
	var wg sync.WaitGroup
	var opsCount int32

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
					ID:        fmt.Sprintf("stress-%d", i),
					Status:    models.StatusProcessing,
					CreatedAt: time.Now(),
				}
				_ = session.SetRun(run)
				atomic.AddInt32(&opsCount, 1)
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
				_, _ = session.GetRun(fmt.Sprintf("stress-%d", i))
				atomic.AddInt32(&opsCount, 1)
				i++
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
				_, _ = session.GetRuns()
				atomic.AddInt32(&opsCount, 1)
			}
		}
	}()

	// FormData worker
	wg.Add(1)
	go func() {
		defer wg.Done()
		i := 0
		for {
			select {
			case <-stopCh:
				return
			default:
				key := fmt.Sprintf("form-%d", i)
				if i%2 == 0 {
					_ = session.SetFormData(key, fmt.Sprintf("data-%d", i))
				} else {
					_, _ = session.GetFormData(key)
				}
				atomic.AddInt32(&opsCount, 1)
				i++
			}
		}
	}()

	// Run for 1 second
	time.Sleep(1 * time.Second)
	close(stopCh)
	wg.Wait()

	totalOps := atomic.LoadInt32(&opsCount)
	t.Logf("SessionCache stress test: %d operations in 1 second", totalOps)
	assert.Greater(t, totalOps, int32(100), "Should complete many operations")
}

// BenchmarkSessionCacheSetRun benchmarks SetRun performance
func BenchmarkSessionCacheSetRun(b *testing.B) {
	session := NewSessionCache()
	defer session.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			run := models.RunResponse{
				ID:        fmt.Sprintf("bench-%d", i),
				Status:    models.StatusProcessing,
				CreatedAt: time.Now(),
			}
			_ = session.SetRun(run)
			i++
		}
	})
}

// BenchmarkSessionCacheGetRun benchmarks GetRun performance
func BenchmarkSessionCacheGetRun(b *testing.B) {
	session := NewSessionCache()
	defer session.Close()

	// Populate with runs
	for i := 0; i < 100; i++ {
		run := models.RunResponse{
			ID:        fmt.Sprintf("bench-%d", i),
			Status:    models.StatusProcessing,
			CreatedAt: time.Now(),
		}
		_ = session.SetRun(run)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_, _ = session.GetRun(fmt.Sprintf("bench-%d", i%100))
			i++
		}
	})
}
