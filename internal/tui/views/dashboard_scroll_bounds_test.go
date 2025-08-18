// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package views

import (
	"fmt"
	"testing"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/cache"
	"github.com/stretchr/testify/assert"
)

// Helper function to create a test dashboard (reuse from other test files)
func createTestDashboardForScrollBounds(t *testing.T) *DashboardView {
	t.Helper()
	
	// Create a mock API client (reuse existing mock from create_form_test.go)
	mockClient := &MockAPIClient{}
	
	// Create a simple cache for testing
	testCache := cache.NewSimpleCache()
	
	// Create dashboard
	dashboard := NewDashboardView(mockClient, testCache)
	
	// Set basic dimensions
	dashboard.width = 100
	dashboard.height = 30
	
	return dashboard
}

// TestScrollToSelectedAdvancedScenarios tests complex scrolling scenarios
func TestScrollToSelectedAdvancedScenarios(t *testing.T) {
	tests := []struct {
		name                string
		setupDashboard      func(*DashboardView)
		column              int
		expectedMaxYOffset  int
		shouldPreventPanic  bool
		description         string
	}{
		{
			name: "large run list with high selected index",
			setupDashboard: func(d *DashboardView) {
				// Create 500 runs (simulate real-world large dataset)
				d.filteredRuns = make([]*models.RunResponse, 500)
				for i := 0; i < 500; i++ {
					d.filteredRuns[i] = &models.RunResponse{
						ID:     fmt.Sprintf("%d", i+1),
						Title:  fmt.Sprintf("Run %d", i),
						Status: models.StatusDone,
					}
				}
				d.selectedRunIdx = 450 // High index like in the user's scenario
				d.runsViewport = viewport.New(40, 20)
			},
			column:              1,
			expectedMaxYOffset:  499, // Should not exceed total items - 1
			shouldPreventPanic:  true,
			description:         "Should handle scrolling to run 450 out of 500 without panic",
		},
		{
			name: "edge case - selected index equals total items",
			setupDashboard: func(d *DashboardView) {
				d.filteredRuns = make([]*models.RunResponse, 100)
				d.selectedRunIdx = 100 // Equals total count (invalid)
				d.runsViewport = viewport.New(40, 20)
			},
			column:              1,
			expectedMaxYOffset:  99,
			shouldPreventPanic:  true,
			description:         "Should clamp index when it equals total count",
		},
		{
			name: "repository column with realistic data",
			setupDashboard: func(d *DashboardView) {
				d.repositories = []models.Repository{
					{Name: "repo1"}, {Name: "repo2"}, {Name: "repo3"},
				}
				d.selectedRepoIdx = 10 // Beyond bounds
				d.repoViewport = viewport.New(40, 20)
			},
			column:              0,
			expectedMaxYOffset:  2, // Should clamp to last valid index
			shouldPreventPanic:  true,
			description:         "Should handle repository selection beyond bounds",
		},
		{
			name: "details column with long content",
			setupDashboard: func(d *DashboardView) {
				// Simulate long details (like the user sees)
				d.detailLines = make([]string, 50)
				for i := 0; i < 50; i++ {
					d.detailLines[i] = "Detail line " + string(rune(i))
				}
				d.selectedDetailLine = 45
				d.detailsViewport = viewport.New(40, 15)
			},
			column:              2,
			expectedMaxYOffset:  49,
			shouldPreventPanic:  true,
			description:         "Should handle long details content properly",
		},
		{
			name: "zero content scenario",
			setupDashboard: func(d *DashboardView) {
				d.filteredRuns = []*models.RunResponse{}
				d.selectedRunIdx = 50 // Any value
				d.runsViewport = viewport.New(40, 20)
			},
			column:              1,
			expectedMaxYOffset:  0,
			shouldPreventPanic:  true,
			description:         "Should handle empty content gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dashboard := createTestDashboardForScrollBounds(t)
			tt.setupDashboard(dashboard)

			// Ensure the test doesn't panic
			if tt.shouldPreventPanic {
				assert.NotPanics(t, func() {
					dashboard.scrollToSelected(tt.column)
				}, tt.description)
			}

			// Verify YOffset is within bounds
			var viewport *viewport.Model
			switch tt.column {
			case 0:
				viewport = &dashboard.repoViewport
			case 1:
				viewport = &dashboard.runsViewport
			case 2:
				viewport = &dashboard.detailsViewport
			}

			assert.LessOrEqual(t, viewport.YOffset, tt.expectedMaxYOffset,
				"YOffset should not exceed maximum valid offset")
			assert.GreaterOrEqual(t, viewport.YOffset, 0,
				"YOffset should not be negative")
		})
	}
}

// TestViewportContentUpdateRaceCondition tests scenarios where content updates might race with rendering
func TestViewportContentUpdateRaceCondition(t *testing.T) {
	dashboard := createTestDashboardForScrollBounds(t)

	// Simulate the user's scenario: returning from details view with high index
	dashboard.selectedRunIdx = 267 // From the debug logs
	dashboard.filteredRuns = make([]*models.RunResponse, 276) // From debug logs
	for i := 0; i < 276; i++ {
		dashboard.filteredRuns[i] = &models.RunResponse{
			ID:     fmt.Sprintf("%d", i+1),
			Title:  "Cached Run",
			Status: models.StatusDone,
		}
	}
	dashboard.selectedRepo = &models.Repository{Name: "test-repo"}

	// Initialize viewport with high YOffset (simulating the restored state)
	dashboard.runsViewport = viewport.New(30, 20)
	dashboard.runsViewport.YOffset = 250 // High offset from previous state

	// This should not panic and should handle the state gracefully
	assert.NotPanics(t, func() {
		dashboard.updateRunsViewportContent()
	}, "updateRunsViewportContent should handle high YOffset gracefully")

	// Verify the final state is valid
	assert.Equal(t, 267, dashboard.selectedRunIdx, "selectedRunIdx should be preserved when valid")
	assert.LessOrEqual(t, dashboard.runsViewport.YOffset, 275, "YOffset should be within content bounds")
}

// TestRepositorySwitchingBounds tests bounds when switching between repositories
func TestRepositorySwitchingBounds(t *testing.T) {
	dashboard := createTestDashboardForScrollBounds(t)

	// Set up two repositories with different run counts
	dashboard.repositories = []models.Repository{
		{Name: "large-repo"},  // Will have many runs
		{Name: "small-repo"},  // Will have few runs
	}

	// Start with large repo selected and high run index
	dashboard.selectedRepoIdx = 0
	dashboard.selectedRunIdx = 245 // High index
	dashboard.selectedRepo = &dashboard.repositories[0]

	// Create many runs for first repo
	dashboard.filteredRuns = make([]*models.RunResponse, 270)
	for i := 0; i < 270; i++ {
		dashboard.filteredRuns[i] = &models.RunResponse{
			ID:         fmt.Sprintf("%d", i+1),
			Title:      "Large Repo Run",
			Status:     models.StatusDone,
			Repository: "large-repo",
		}
	}

	dashboard.runsViewport = viewport.New(30, 20)

	// Update content - should work fine with high index
	assert.NotPanics(t, func() {
		dashboard.updateRunsViewportContent()
	}, "Should handle high index with large repo")

	assert.Equal(t, 245, dashboard.selectedRunIdx, "Should preserve high index for large repo")

	// Now switch to small repo with only 6 runs (from debug logs)
	dashboard.selectedRepoIdx = 1
	dashboard.selectedRepo = &dashboard.repositories[1]
	dashboard.filteredRuns = make([]*models.RunResponse, 6) // Small repo
	for i := 0; i < 6; i++ {
		dashboard.filteredRuns[i] = &models.RunResponse{
			ID:         fmt.Sprintf("%d", i+1),
			Title:      "Small Repo Run",
			Status:     models.StatusDone,
			Repository: "small-repo",
		}
	}

	// This should clamp the index appropriately
	assert.NotPanics(t, func() {
		dashboard.updateRunsViewportContent()
	}, "Should handle index clamping when switching to small repo")

	// Should clamp to last valid index for small repo
	assert.Equal(t, 5, dashboard.selectedRunIdx, "Should clamp to last valid index for small repo")
	assert.LessOrEqual(t, dashboard.runsViewport.YOffset, 5, "YOffset should be clamped for small repo")
}

// TestEmergencyViewportReset tests the emergency reset functionality in render methods
func TestEmergencyViewportReset(t *testing.T) {
	dashboard := createTestDashboardForScrollBounds(t)
	dashboard.width = 100
	dashboard.height = 50

	// Set up viewport with severely invalid state (way beyond content)
	dashboard.runsViewport = viewport.New(30, 20)
	dashboard.runsViewport.SetContent("Line 1\nLine 2\n") // Only 2 lines
	dashboard.runsViewport.YOffset = 500                  // Way beyond content (500 >> 2)

	// The emergency reset should only trigger when offset is WAY beyond content (>= totalLines + 10)
	// In this case: 500 >= 2 + 10 = true, so should reset

	var renderOutput string
	assert.NotPanics(t, func() {
		renderOutput = dashboard.renderRunsColumn(30, 20)
	}, "renderRunsColumn should handle emergency viewport reset")

	assert.Greater(t, len(renderOutput), 0, "Should produce valid output after emergency reset")

	// Verify YOffset was reset to safe value
	assert.LessOrEqual(t, dashboard.runsViewport.YOffset, 1, "YOffset should be reset to safe value")
}

// TestViewportBoundsWithRealWorldData tests with data similar to what the user experienced
func TestViewportBoundsWithRealWorldData(t *testing.T) {
	dashboard := createTestDashboardForScrollBounds(t)

	// Simulate the exact scenario from debug logs
	dashboard.repositories = []models.Repository{
		{Name: "test-acc-254/youtube-music"},
		{Name: "test-acc-254/testy"},
	}

	// Simulate 276 runs (filtered count from logs)
	dashboard.filteredRuns = make([]*models.RunResponse, 276)
	for i := 0; i < 276; i++ {
		dashboard.filteredRuns[i] = &models.RunResponse{
			ID:     fmt.Sprintf("%d", i+300), // Start from 300 to match log patterns
			Title:  "Real Run",
			Status: models.StatusDone,
		}
	}

	// Test various selected indices that appeared in the logs
	testIndices := []int{267, 260, 255, 29, 21, 15, 247}

	for _, idx := range testIndices {
		t.Run(fmt.Sprintf("selectedRunIdx_%d", idx), func(t *testing.T) {
			dashboard.selectedRunIdx = idx
			dashboard.selectedRepo = &dashboard.repositories[0]
			dashboard.runsViewport = viewport.New(26, 26) // From logs: viewports=26/26/27

			assert.NotPanics(t, func() {
				dashboard.updateRunsViewportContent()
				dashboard.scrollToSelected(1)
			}, "Should handle real-world index %d without panic", idx)

			// Verify index is preserved (should be valid for 276 runs)
			assert.Equal(t, idx, dashboard.selectedRunIdx, 
				"selectedRunIdx %d should be preserved as it's within bounds", idx)

			// Verify viewport state is valid
			assert.GreaterOrEqual(t, dashboard.runsViewport.YOffset, 0, 
				"YOffset should not be negative")
			assert.LessOrEqual(t, dashboard.runsViewport.YOffset, 275, 
				"YOffset should not exceed max valid offset")
		})
	}
}