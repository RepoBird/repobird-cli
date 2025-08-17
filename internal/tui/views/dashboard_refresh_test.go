// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package views

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestDashboardManualRefresh tests the 'r' key refresh functionality
func TestDashboardManualRefresh(t *testing.T) {
	// Set up temp directory for cache
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create mock client
	mockClient := new(mockAPIClient)

	// Create cache with some existing runs
	testCache := cache.NewSimpleCache()

	// Add both terminal and active runs to cache
	existingRuns := []models.RunResponse{
		{
			ID:        "terminal-run-1",
			Status:    models.StatusDone,
			CreatedAt: time.Now().Add(-2 * time.Hour),
		},
		{
			ID:        "active-run-1",
			Status:    models.StatusProcessing,
			CreatedAt: time.Now().Add(-10 * time.Minute),
		},
	}
	testCache.SetRuns(existingRuns)

	// Create dashboard
	dashboard := NewDashboardView(mockClient, testCache)
	dashboard.width = 120
	dashboard.height = 40

	// Mock API to return fresh data
	freshRuns := []models.RunResponse{
		{
			ID:        "fresh-run-1",
			Status:    models.StatusProcessing,
			CreatedAt: time.Now(),
		},
		{
			ID:        "fresh-run-2",
			Status:    models.StatusDone,
			CreatedAt: time.Now().Add(-5 * time.Minute),
		},
	}
	mockClient.On("GetRunsWithLimit", mock.Anything).Return(freshRuns, nil)
	mockClient.On("GetUserInfo").Return(&models.UserInfo{
		ID:    123,
		Email: "test@example.com",
	}, nil)

	// Simulate pressing 'r' key
	model, cmd := dashboard.Update(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'r'},
	})

	// Verify dashboard is loading
	updatedDash := model.(*DashboardView)
	assert.True(t, updatedDash.loading, "Dashboard should be loading after 'r' key")
	assert.NotNil(t, cmd, "Should have command to load data")

	// Verify cache was cleared (not just active runs)
	cachedRuns := testCache.GetRuns()
	assert.Empty(t, cachedRuns, "Cache should be completely cleared after manual refresh")

	// Note: The 'r' key calls cache.Clear() directly, not setting force_api_refresh flag
	// This is different from the navigation-triggered refresh which uses the flag
}

// TestDashboardAutoRefreshOnNavigation tests automatic refresh when returning from other views
func TestDashboardAutoRefreshOnNavigation(t *testing.T) {
	// Set up temp directory for cache
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create mock client
	mockClient := new(mockAPIClient)

	// Create cache
	testCache := cache.NewSimpleCache()

	// Set the refresh flag (as if returning from bulk or create)
	testCache.SetNavigationContext("dashboard_needs_refresh", true)

	// Add some old cached runs
	oldRuns := []models.RunResponse{
		{
			ID:        "old-run-1",
			Status:    models.StatusProcessing,
			CreatedAt: time.Now().Add(-1 * time.Hour),
		},
	}
	testCache.SetRuns(oldRuns)

	// Create dashboard
	dashboard := NewDashboardView(mockClient, testCache)

	// Mock API responses
	freshRuns := []models.RunResponse{
		{
			ID:        "new-run-after-create",
			Status:    models.StatusProcessing,
			CreatedAt: time.Now(),
		},
	}
	mockClient.On("GetRunsWithLimit", mock.Anything).Return(freshRuns, nil)
	mockClient.On("GetUserInfo").Return(&models.UserInfo{
		ID:    123,
		Email: "test@example.com",
	}, nil)

	// Send WindowSizeMsg which triggers refresh check
	model, cmd := dashboard.Update(tea.WindowSizeMsg{
		Width:  120,
		Height: 40,
	})

	// Verify dashboard is loading
	updatedDash := model.(*DashboardView)
	assert.True(t, updatedDash.loading, "Dashboard should be loading after detecting refresh flag")

	// Verify refresh flag was cleared
	refreshFlag := testCache.GetNavigationContext("dashboard_needs_refresh")
	assert.Nil(t, refreshFlag, "Refresh flag should be cleared after triggering refresh")

	// Verify command to load data was returned
	assert.NotNil(t, cmd, "Should have command to load dashboard data")
}

// TestDashboardRefreshPreservesTerminalRuns tests that auto-refresh preserves terminal runs
func TestDashboardRefreshPreservesTerminalRuns(t *testing.T) {
	// Set up temp directory for cache
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create cache
	testCache := cache.NewSimpleCache()

	// Add mix of terminal and active runs
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
	}
	testCache.SetRuns(runs)

	// Invalidate active runs (as done by bulk/create views)
	testCache.InvalidateActiveRuns()

	// Check remaining runs
	remainingRuns := testCache.GetRuns()

	// Should only have terminal runs
	assert.Len(t, remainingRuns, 2, "Should only have 2 terminal runs after InvalidateActiveRuns")

	// Verify specific runs remain
	runMap := make(map[string]bool)
	for _, run := range remainingRuns {
		runMap[run.ID] = true
	}

	assert.True(t, runMap["done-run-1"], "Done run should remain")
	assert.True(t, runMap["failed-run-1"], "Failed run should remain")
	assert.False(t, runMap["processing-run-1"], "Processing run should be cleared")
	assert.False(t, runMap["queued-run-1"], "Queued run should be cleared")
}

// TestDashboardRefreshAfterCreateFlow tests the complete CREATE -> DETAILS -> DASHBOARD flow
func TestDashboardRefreshAfterCreateFlow(t *testing.T) {
	// Set up temp directory for cache
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create cache
	testCache := cache.NewSimpleCache()

	// Simulate CREATE view setting from_create flag
	testCache.SetNavigationContext("from_create", true)

	// Simulate DETAILS view checking from_create and setting refresh flag
	fromCreate := testCache.GetNavigationContext("from_create")
	require.NotNil(t, fromCreate, "from_create flag should be set")

	if wasFromCreate, ok := fromCreate.(bool); ok && wasFromCreate {
		// This is what DETAILS view does when navigating to dashboard
		testCache.InvalidateActiveRuns()
		testCache.SetNavigationContext("dashboard_needs_refresh", true)
		testCache.SetNavigationContext("from_create", nil) // Clear the flag
	}

	// Verify flags are set correctly
	assert.Nil(t, testCache.GetNavigationContext("from_create"), "from_create should be cleared")
	assert.NotNil(t, testCache.GetNavigationContext("dashboard_needs_refresh"), "dashboard_needs_refresh should be set")

	// Create mock client and dashboard
	mockClient := new(mockAPIClient)
	dashboard := NewDashboardView(mockClient, testCache)

	// Mock API responses
	mockClient.On("GetRunsWithLimit", mock.Anything).Return([]models.RunResponse{
		{ID: "new-run-from-create", Status: models.StatusProcessing},
	}, nil)
	mockClient.On("GetUserInfo").Return(&models.UserInfo{ID: 1}, nil)

	// Trigger refresh check via WindowSizeMsg
	model, cmd := dashboard.Update(tea.WindowSizeMsg{
		Width:  120,
		Height: 40,
	})

	// Verify refresh was triggered
	updatedDash := model.(*DashboardView)
	assert.True(t, updatedDash.loading, "Dashboard should refresh after CREATE flow")
	assert.NotNil(t, cmd, "Should have command to load data")

	// Verify refresh flag was cleared
	assert.Nil(t, testCache.GetNavigationContext("dashboard_needs_refresh"), "Refresh flag should be cleared")
}

// TestDashboardNoRefreshWithoutFlag tests that dashboard doesn't refresh unnecessarily
func TestDashboardNoRefreshWithoutFlag(t *testing.T) {
	// Set up temp directory for cache
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create mock client
	mockClient := new(mockAPIClient)

	// Create cache WITHOUT setting refresh flag
	testCache := cache.NewSimpleCache()

	// Add cached runs
	cachedRuns := []models.RunResponse{
		{ID: "cached-run-1", Status: models.StatusDone},
	}
	testCache.SetRuns(cachedRuns)

	// Create dashboard
	dashboard := NewDashboardView(mockClient, testCache)

	// Simulate dashboard finishing its initial load by setting loading to false
	// (Dashboard starts with loading=true by default)
	dashboard.loading = false

	// Don't expect any API calls since no refresh flag is set
	// mockClient.On("GetRunsWithLimit", mock.Anything) - NOT called

	// Send WindowSizeMsg (this should NOT trigger a refresh without flags)
	model, _ := dashboard.Update(tea.WindowSizeMsg{
		Width:  120,
		Height: 40,
	})

	// Verify dashboard is still NOT loading (no unnecessary refresh)
	updatedDash := model.(*DashboardView)
	assert.False(t, updatedDash.loading, "Dashboard should not refresh without flag")

	// Verify cached runs are still there
	remainingRuns := testCache.GetRuns()
	assert.Len(t, remainingRuns, 1, "Cached runs should remain")
	assert.Equal(t, "cached-run-1", remainingRuns[0].ID)

	// Verify no API calls were made
	mockClient.AssertNotCalled(t, "GetRunsWithLimit", mock.Anything)
}
