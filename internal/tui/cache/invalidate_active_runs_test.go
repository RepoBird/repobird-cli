package cache

import (
	"testing"
	"time"

	"github.com/repobird/repobird-cli/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSimpleCache_InvalidateActiveRuns tests that InvalidateActiveRuns only clears active runs
func TestSimpleCache_InvalidateActiveRuns(t *testing.T) {
	// Set up temp directory for cache
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cache := NewSimpleCache()

	// Add a mix of terminal and active runs
	runs := []models.RunResponse{
		{
			ID:        "done-run-1",
			Status:    models.StatusDone,
			CreatedAt: time.Now().Add(-3 * time.Hour),
		},
		{
			ID:        "failed-run-1",
			Status:    models.StatusFailed,
			CreatedAt: time.Now().Add(-2 * time.Hour),
		},
		{
			ID:        "processing-run-1",
			Status:    models.StatusProcessing,
			CreatedAt: time.Now().Add(-30 * time.Minute),
		},
		{
			ID:        "queued-run-1",
			Status:    models.StatusQueued,
			CreatedAt: time.Now().Add(-10 * time.Minute),
		},
		{
			ID:        "initializing-run-1",
			Status:    models.StatusInitializing,
			CreatedAt: time.Now().Add(-5 * time.Minute),
		},
	}

	// Set all runs in cache
	cache.SetRuns(runs)

	// Verify all runs are cached
	cachedRuns := cache.GetRuns()
	assert.Len(t, cachedRuns, 5, "Should have all 5 runs cached initially")

	// Invalidate active runs
	cache.InvalidateActiveRuns()

	// Check what remains in cache
	remainingRuns := cache.GetRuns()

	// Should only have terminal runs (DONE, FAILED)
	assert.Len(t, remainingRuns, 2, "Should only have 2 terminal runs remaining")

	// Verify only terminal runs remain
	for _, run := range remainingRuns {
		isTerminal := run.Status == models.StatusDone || run.Status == models.StatusFailed
		assert.True(t, isTerminal, "Only terminal runs should remain after InvalidateActiveRuns, got status: %s", run.Status)
	}

	// Verify specific runs remain
	runMap := make(map[string]bool)
	for _, run := range remainingRuns {
		runMap[run.ID] = true
	}

	assert.True(t, runMap["done-run-1"], "Done run should remain")
	assert.True(t, runMap["failed-run-1"], "Failed run should remain")
	assert.False(t, runMap["processing-run-1"], "Processing run should be cleared")
	assert.False(t, runMap["queued-run-1"], "Queued run should be cleared")
	assert.False(t, runMap["initializing-run-1"], "Initializing run should be cleared")
}

// TestHybridCache_InvalidateActiveRuns tests hybrid cache invalidation
func TestHybridCache_InvalidateActiveRuns(t *testing.T) {
	// Set up temp directory for cache
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	hybrid, err := NewHybridCache("test-user")
	require.NoError(t, err)

	// Add mix of runs - terminal runs will go to permanent cache, active to session
	terminalRun := models.RunResponse{
		ID:        "done-run",
		Status:    models.StatusDone,
		CreatedAt: time.Now().Add(-2 * time.Hour),
	}

	activeRun := models.RunResponse{
		ID:        "processing-run",
		Status:    models.StatusProcessing,
		CreatedAt: time.Now().Add(-10 * time.Minute),
	}

	// Set runs
	err = hybrid.SetRun(terminalRun)
	require.NoError(t, err)
	err = hybrid.SetRun(activeRun)
	require.NoError(t, err)

	// Verify both runs are retrievable
	doneRun, found := hybrid.GetRun("done-run")
	assert.True(t, found, "Done run should be found")
	assert.NotNil(t, doneRun)

	processingRun, found := hybrid.GetRun("processing-run")
	assert.True(t, found, "Processing run should be found")
	assert.NotNil(t, processingRun)

	// Invalidate active runs
	err = hybrid.InvalidateActiveRuns()
	require.NoError(t, err)

	// Terminal run should still be there (in permanent cache)
	doneRun, found = hybrid.GetRun("done-run")
	assert.True(t, found, "Done run should still be found after invalidation")
	assert.NotNil(t, doneRun)

	// Active run should be gone
	processingRun, found = hybrid.GetRun("processing-run")
	assert.False(t, found, "Processing run should be cleared after invalidation")
	assert.Nil(t, processingRun)
}

// TestSessionCache_InvalidateActiveRuns tests session cache invalidation
func TestSessionCache_InvalidateActiveRuns(t *testing.T) {
	session := NewSessionCache()

	// Add only active runs (session cache should only store active runs)
	runs := []models.RunResponse{
		{
			ID:        "running-run-1",
			Status:    models.StatusProcessing,
			CreatedAt: time.Now().Add(-5 * time.Minute),
		},
		{
			ID:        "running-run-2",
			Status:    models.StatusQueued,
			CreatedAt: time.Now().Add(-10 * time.Minute),
		},
	}

	err := session.SetRuns(runs)
	require.NoError(t, err)

	// Verify both are in session cache
	cachedRuns, found := session.GetRuns()
	assert.True(t, found)
	assert.Len(t, cachedRuns, 2, "Should have 2 active runs")

	// Invalidate active runs (should clear all since session cache only has active runs)
	err = session.InvalidateActiveRuns()
	require.NoError(t, err)

	// Check remaining runs (should be empty)
	cachedRuns, found = session.GetRuns()
	assert.False(t, found, "Should not find runs after invalidation")
	assert.Empty(t, cachedRuns, "Should have no runs remaining after invalidation")
}

// TestInvalidateActiveRuns_EmptyCache tests invalidation on empty cache
func TestInvalidateActiveRuns_EmptyCache(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cache := NewSimpleCache()

	// Invalidate on empty cache should not panic
	cache.InvalidateActiveRuns()

	// Cache should still be empty
	runs := cache.GetRuns()
	assert.Empty(t, runs)
}

// TestInvalidateActiveRuns_OnlyTerminalRuns tests invalidation when only terminal runs exist
func TestInvalidateActiveRuns_OnlyTerminalRuns(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cache := NewSimpleCache()

	// Add only terminal runs
	runs := []models.RunResponse{
		{
			ID:        "done-run-1",
			Status:    models.StatusDone,
			CreatedAt: time.Now().Add(-1 * time.Hour),
		},
		{
			ID:        "failed-run-1",
			Status:    models.StatusFailed,
			CreatedAt: time.Now().Add(-2 * time.Hour),
		},
	}

	cache.SetRuns(runs)

	// Invalidate active runs
	cache.InvalidateActiveRuns()

	// All runs should remain since they're terminal
	remainingRuns := cache.GetRuns()
	assert.Len(t, remainingRuns, 2, "Terminal runs should remain")
}

// TestInvalidateActiveRuns_OnlyActiveRuns tests invalidation when only active runs exist
func TestInvalidateActiveRuns_OnlyActiveRuns(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cache := NewSimpleCache()

	// Add only active runs
	runs := []models.RunResponse{
		{
			ID:        "processing-run-1",
			Status:    models.StatusProcessing,
			CreatedAt: time.Now().Add(-5 * time.Minute),
		},
		{
			ID:        "queued-run-1",
			Status:    models.StatusQueued,
			CreatedAt: time.Now().Add(-10 * time.Minute),
		},
	}

	cache.SetRuns(runs)

	// Invalidate active runs
	cache.InvalidateActiveRuns()

	// No runs should remain
	remainingRuns := cache.GetRuns()
	assert.Empty(t, remainingRuns, "All active runs should be cleared")
}
