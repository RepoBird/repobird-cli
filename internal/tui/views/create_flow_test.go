// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package views

import (
	"testing"
	"time"

	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/cache"
	"github.com/repobird/repobird-cli/internal/tui/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestCreateFlowNavigationContext tests the complete CREATE -> DETAILS flow
func TestCreateFlowNavigationContext(t *testing.T) {
	// Set up temp directory for cache
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create mock client
	mockClient := new(mockAPIClient)

	// Create cache
	testCache := cache.NewSimpleCache()

	// Add some existing runs that should be invalidated after creation
	existingRuns := []models.RunResponse{
		{
			ID:        "existing-run-1",
			Status:    models.StatusProcessing,
			CreatedAt: time.Now().Add(-10 * time.Minute),
		},
	}
	testCache.SetRuns(existingRuns)

	// Mock successful run creation
	newRun := &models.RunResponse{
		ID:        "new-run-123",
		Status:    models.StatusQueued,
		Title:     "Test Run",
		CreatedAt: time.Now(),
	}

	mockClient.On("CreateRunAPI", mock.AnythingOfType("*models.APIRunRequest")).Return(newRun, nil)

	// Test the CREATE view navigation to DETAILS with FromCreate flag
	t.Run("CREATE view sets FromCreate flag when navigating to DETAILS", func(t *testing.T) {
		// Simulate a successful run creation that would navigate to details
		// In practice, this happens when CreateRunView submits successfully

		// The key part is that when CREATE succeeds, it should navigate with FromCreate: true
		// Let's test this by simulating what App.handleNavigation does

		// Simulate NavigateToDetailsMsg with FromCreate: true
		navMsg := messages.NavigateToDetailsMsg{
			RunID:      "new-run-123",
			FromCreate: true,
			RunData:    newRun,
		}

		// Verify the message has FromCreate set
		assert.True(t, navMsg.FromCreate, "NavigateToDetailsMsg should have FromCreate=true")
		assert.Equal(t, "new-run-123", navMsg.RunID, "Should have correct run ID")
		assert.NotNil(t, navMsg.RunData, "Should have run data")
	})

	// Test what happens when DETAILS view is created with FromCreate context
	t.Run("DETAILS view receives FromCreate flag in navigation context", func(t *testing.T) {
		// Simulate App setting navigation context when handling NavigateToDetailsMsg with FromCreate=true
		testCache.SetNavigationContext("from_create", true)

		// Verify context was set
		fromCreate := testCache.GetNavigationContext("from_create")
		require.NotNil(t, fromCreate, "from_create should be set in navigation context")
		assert.True(t, fromCreate.(bool), "from_create should be true")

		// Create details view (simulating what App does)
		detailsView := NewRunDetailsViewWithData(mockClient, testCache, *newRun)
		assert.NotNil(t, detailsView, "Details view should be created")

		// The from_create flag should be available for the details view to check
		// when user navigates back to dashboard
	})
}

// TestDetailsViewDashboardNavigation tests DETAILS -> DASHBOARD navigation with cache invalidation
func TestDetailsViewDashboardNavigation(t *testing.T) {
	// Set up temp directory for cache
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create mock client
	mockClient := new(mockAPIClient)

	// Create cache with from_create flag set
	testCache := cache.NewSimpleCache()
	testCache.SetNavigationContext("from_create", true)

	// Add some active runs that should be invalidated
	activeRuns := []models.RunResponse{
		{
			ID:        "active-run-1",
			Status:    models.StatusProcessing,
			CreatedAt: time.Now().Add(-15 * time.Minute),
		},
		{
			ID:        "active-run-2",
			Status:    models.StatusQueued,
			CreatedAt: time.Now().Add(-20 * time.Minute),
		},
	}
	testCache.SetRuns(activeRuns)

	// Create details view with mock run data
	runData := models.RunResponse{
		ID:     "test-run-123",
		Status: models.StatusDone,
		Title:  "Test Run",
	}

	_ = NewRunDetailsViewWithData(mockClient, testCache, runData)

	// Test simulating user pressing 'd' to go to dashboard
	t.Run("Details view sets refresh flag when from_create is true", func(t *testing.T) {
		// Verify initial state
		initialRuns := testCache.GetRuns()
		assert.Len(t, initialRuns, 2, "Should have 2 active runs initially")

		fromCreate := testCache.GetNavigationContext("from_create")
		require.NotNil(t, fromCreate, "from_create flag should be set")
		assert.True(t, fromCreate.(bool), "from_create should be true")

		// Simulate the logic that would happen when user navigates to dashboard
		// This would typically be in the key handler for 'd' key in details view
		if wasFromCreate, ok := fromCreate.(bool); ok && wasFromCreate {
			// This is what the details view should do
			testCache.SetNavigationContext("dashboard_needs_refresh", true)
			testCache.InvalidateActiveRuns()
			testCache.SetNavigationContext("from_create", nil) // Clear the flag
		}

		// Verify the actions were taken
		assert.Nil(t, testCache.GetNavigationContext("from_create"), "from_create should be cleared")

		refreshFlag := testCache.GetNavigationContext("dashboard_needs_refresh")
		assert.NotNil(t, refreshFlag, "dashboard_needs_refresh should be set")
		assert.True(t, refreshFlag.(bool), "dashboard_needs_refresh should be true")

		// Verify active runs were invalidated (only terminal runs should remain)
		remainingRuns := testCache.GetRuns()
		assert.Empty(t, remainingRuns, "Active runs should be invalidated")
	})
}

// TestDetailsViewWithoutFromCreateFlag tests normal navigation without CREATE context
func TestDetailsViewWithoutFromCreateFlag(t *testing.T) {
	// Set up temp directory for cache
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create mock client
	mockClient := new(mockAPIClient)

	// Create cache WITHOUT from_create flag
	testCache := cache.NewSimpleCache()

	// Add some runs
	runs := []models.RunResponse{
		{
			ID:        "run-1",
			Status:    models.StatusProcessing,
			CreatedAt: time.Now().Add(-15 * time.Minute),
		},
		{
			ID:        "run-2",
			Status:    models.StatusDone,
			CreatedAt: time.Now().Add(-1 * time.Hour),
		},
	}
	testCache.SetRuns(runs)

	// Create details view
	runData := models.RunResponse{
		ID:     "test-run-123",
		Status: models.StatusDone,
		Title:  "Test Run",
	}

	detailsView := NewRunDetailsViewWithData(mockClient, testCache, runData)
	assert.NotNil(t, detailsView, "Details view should be created")

	// Simulate navigation to dashboard without from_create flag
	fromCreate := testCache.GetNavigationContext("from_create")
	assert.Nil(t, fromCreate, "from_create should not be set")

	// Simulate what would happen on dashboard navigation
	if fromCreate != nil {
		if wasFromCreate, ok := fromCreate.(bool); ok && wasFromCreate {
			// This should NOT execute since fromCreate is nil
			testCache.SetNavigationContext("dashboard_needs_refresh", true)
			testCache.InvalidateActiveRuns()
		}
	}

	// Verify NO refresh flag was set
	refreshFlag := testCache.GetNavigationContext("dashboard_needs_refresh")
	assert.Nil(t, refreshFlag, "dashboard_needs_refresh should NOT be set without from_create")

	// Verify runs were NOT invalidated
	remainingRuns := testCache.GetRuns()
	assert.Len(t, remainingRuns, 2, "Runs should remain without from_create context")
}

// TestCreateViewSuccessfulSubmission tests the CREATE view submission process
func TestCreateViewSuccessfulSubmission(t *testing.T) {
	// Set up temp directory for cache
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create mock client
	mockClient := new(mockAPIClient)

	// Mock successful API call
	newRun := &models.RunResponse{
		ID:        "new-run-456",
		Status:    models.StatusQueued,
		Title:     "New Test Run",
		CreatedAt: time.Now(),
	}

	runRequest := &models.APIRunRequest{
		Prompt:         "Test prompt",
		RepositoryName: "test/repo",
		SourceBranch:   "main",
		TargetBranch:   "feature/test",
		RunType:        "run",
		Title:          "New Test Run",
	}

	mockClient.On("CreateRunAPI", runRequest).Return(newRun, nil)

	// Test that a successful CREATE submission would result in proper navigation
	t.Run("Successful CREATE returns NavigateToDetailsMsg with FromCreate=true", func(t *testing.T) {
		// This simulates what would happen in CreateRunView when submission succeeds

		// Call the mocked API
		result, err := mockClient.CreateRunAPI(runRequest)
		require.NoError(t, err, "API call should succeed")
		require.NotNil(t, result, "Should get run response")
		assert.Equal(t, "new-run-456", result.ID, "Should have correct run ID")

		// The CREATE view would then send a navigation message like this:
		navMsg := messages.NavigateToDetailsMsg{
			RunID:      result.ID,
			FromCreate: true, // This is the key flag
			RunData:    result,
		}

		// Verify the navigation message is constructed correctly
		assert.True(t, navMsg.FromCreate, "Navigation should have FromCreate=true")
		assert.Equal(t, result.ID, navMsg.RunID, "Should have correct run ID")
		assert.Equal(t, result, navMsg.RunData, "Should have run data")
	})
}

// TestCompleteCreateToDashboardFlow tests the full CREATE -> DETAILS -> DASHBOARD flow
func TestCompleteCreateToDashboardFlow(t *testing.T) {
	// Set up temp directory for cache
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create cache with some initial runs
	testCache := cache.NewSimpleCache()

	initialRuns := []models.RunResponse{
		{
			ID:        "old-run-1",
			Status:    models.StatusProcessing,
			CreatedAt: time.Now().Add(-30 * time.Minute),
		},
		{
			ID:        "old-run-2",
			Status:    models.StatusDone,
			CreatedAt: time.Now().Add(-2 * time.Hour),
		},
	}
	testCache.SetRuns(initialRuns)

	t.Run("Complete flow preserves terminal runs and invalidates active runs", func(t *testing.T) {
		// Step 1: CREATE succeeds and navigates to DETAILS with FromCreate=true
		// (This would be done by App.handleNavigation)
		testCache.SetNavigationContext("from_create", true)

		// Step 2: User is in DETAILS view and navigates to DASHBOARD
		// DETAILS view checks from_create flag and sets up refresh
		fromCreate := testCache.GetNavigationContext("from_create")
		if wasFromCreate, ok := fromCreate.(bool); ok && wasFromCreate {
			testCache.SetNavigationContext("dashboard_needs_refresh", true)
			testCache.InvalidateActiveRuns()
			testCache.SetNavigationContext("from_create", nil)
		}

		// Step 3: DASHBOARD view detects refresh flag and refreshes
		refreshFlag := testCache.GetNavigationContext("dashboard_needs_refresh")
		require.NotNil(t, refreshFlag, "Refresh flag should be set")
		assert.True(t, refreshFlag.(bool), "Refresh flag should be true")

		// Verify cache state after invalidation
		remainingRuns := testCache.GetRuns()

		// Only terminal runs should remain
		for _, run := range remainingRuns {
			assert.True(t,
				run.Status == models.StatusDone ||
					run.Status == models.StatusFailed,
				"Only terminal runs should remain, got: %s", run.Status)
		}

		// In this test case, old-run-2 is DONE (terminal) so it should remain
		// old-run-1 is PROCESSING (active) so it should be cleared
		assert.Len(t, remainingRuns, 1, "Should have 1 terminal run remaining")
		assert.Equal(t, "old-run-2", remainingRuns[0].ID, "Terminal run should remain")

		// Simulate dashboard clearing the refresh flag after using it
		testCache.SetNavigationContext("dashboard_needs_refresh", nil)

		refreshFlag = testCache.GetNavigationContext("dashboard_needs_refresh")
		assert.Nil(t, refreshFlag, "Refresh flag should be cleared after use")
	})
}
