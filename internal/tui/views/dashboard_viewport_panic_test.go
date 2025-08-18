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

// TestScrollToSelectedBoundsValidation tests that scrollToSelected properly validates bounds
func TestScrollToSelectedBoundsValidation(t *testing.T) {
	tests := []struct {
		name              string
		column            int
		selectedIdx       int
		totalItems        int
		viewportHeight    int
		expectedYOffset   int
		expectOffsetReset bool
	}{
		{
			name:            "normal case - index within bounds",
			column:          1, // runs column
			selectedIdx:     10,
			totalItems:      100,
			viewportHeight:  20,
			expectedYOffset: 0, // Should not change from 0 since item is visible
		},
		{
			name:              "index beyond total items - should clamp",
			column:            1,
			selectedIdx:       150,
			totalItems:        100,
			viewportHeight:    20,
			expectedYOffset:   80, // 99 - 20 + 1 (to show last item)
			expectOffsetReset: true,
		},
		{
			name:            "empty items - should reset offset",
			column:          1,
			selectedIdx:     50,
			totalItems:      0,
			viewportHeight:  20,
			expectedYOffset: 0,
		},
		{
			name:            "single item - should not exceed bounds",
			column:          1,
			selectedIdx:     5,
			totalItems:      1,
			viewportHeight:  20,
			expectedYOffset: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create dashboard with mock data
			dashboard := createTestDashboard(t)

			// Set up test data based on column
			switch tt.column {
			case 0: // repositories
				dashboard.repositories = make([]models.Repository, tt.totalItems)
				dashboard.selectedRepoIdx = tt.selectedIdx
			case 1: // runs
				dashboard.filteredRuns = make([]*models.RunResponse, tt.totalItems)
				dashboard.selectedRunIdx = tt.selectedIdx
			case 2: // details
				dashboard.detailLines = make([]string, tt.totalItems)
				dashboard.selectedDetailLine = tt.selectedIdx
			}

			// Set up viewport
			var vp *viewport.Model
			switch tt.column {
			case 0:
				vp = &dashboard.repoViewport
			case 1:
				vp = &dashboard.runsViewport
			case 2:
				vp = &dashboard.detailsViewport
			}

			vp.Height = tt.viewportHeight
			vp.YOffset = 0

			// Call scrollToSelected
			dashboard.scrollToSelected(tt.column)

			// Verify YOffset is within valid bounds
			assert.GreaterOrEqual(t, vp.YOffset, 0, "YOffset should not be negative")
			if tt.totalItems > 0 {
				maxOffset := tt.totalItems - 1
				assert.LessOrEqual(t, vp.YOffset, maxOffset, "YOffset should not exceed max valid offset")
			} else {
				assert.Equal(t, 0, vp.YOffset, "YOffset should be 0 for empty content")
			}
		})
	}
}

// TestUpdateRunsViewportContentBoundsValidation tests the bounds validation in updateRunsViewportContent
func TestUpdateRunsViewportContentBoundsValidation(t *testing.T) {
	tests := []struct {
		name                string
		selectedRunIdx      int
		filteredRunsCount   int
		selectedRepo        *models.Repository
		expectIndexClamping bool
		expectedFinalIdx    int
	}{
		{
			name:              "normal case - index within bounds",
			selectedRunIdx:    10,
			filteredRunsCount: 100,
			selectedRepo:      &models.Repository{Name: "test-repo"},
			expectIndexClamping: false,
			expectedFinalIdx:  10,
		},
		{
			name:              "index beyond bounds - should clamp",
			selectedRunIdx:    150,
			filteredRunsCount: 100,
			selectedRepo:      &models.Repository{Name: "test-repo"},
			expectIndexClamping: true,
			expectedFinalIdx:  99,
		},
		{
			name:              "no repository selected - should not clamp",
			selectedRunIdx:    150,
			filteredRunsCount: 0,
			selectedRepo:      nil,
			expectIndexClamping: false,
			expectedFinalIdx:  150, // Should remain unchanged when no repo selected
		},
		{
			name:              "empty runs - should not clamp if no repo",
			selectedRunIdx:    50,
			filteredRunsCount: 0,
			selectedRepo:      nil,
			expectIndexClamping: false,
			expectedFinalIdx:  50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dashboard := createTestDashboard(t)
			dashboard.selectedRunIdx = tt.selectedRunIdx
			dashboard.selectedRepo = tt.selectedRepo
			
			// Create mock filtered runs
			dashboard.filteredRuns = make([]*models.RunResponse, tt.filteredRunsCount)
			for i := 0; i < tt.filteredRunsCount; i++ {
				dashboard.filteredRuns[i] = &models.RunResponse{
					ID:     fmt.Sprintf("%d", i+1),
					Title:  "Test Run",
					Status: models.StatusDone,
				}
			}

			// Initialize viewport
			dashboard.runsViewport = viewport.New(50, 20)
			dashboard.runsViewport.SetContent("Initial content\nLine 2\nLine 3")

			// Call updateRunsViewportContent
			dashboard.updateRunsViewportContent()

			// Verify index was clamped if expected
			if tt.expectIndexClamping {
				assert.Equal(t, tt.expectedFinalIdx, dashboard.selectedRunIdx, 
					"selectedRunIdx should be clamped to valid range")
			} else {
				assert.Equal(t, tt.expectedFinalIdx, dashboard.selectedRunIdx,
					"selectedRunIdx should remain unchanged")
			}

			// Verify viewport YOffset is valid
			assert.GreaterOrEqual(t, dashboard.runsViewport.YOffset, 0, 
				"Viewport YOffset should not be negative")
		})
	}
}

// TestDashboardStatePreservation tests that dashboard state is properly preserved across navigation
func TestDashboardStatePreservation(t *testing.T) {
	dashboard := createTestDashboard(t)
	
	// Set up test state - will be used to verify restored dashboard
	_ = map[string]interface{}{
		"selectedRepoIdx":    5,
		"selectedRunIdx":     250,
		"selectedDetailLine": 3,
		"focusedColumn":      2,
		"width":              100,
		"height":             50,
	}

	// Create dashboard with restored state
	restoredDashboard := NewDashboardViewWithState(
		dashboard.client, 
		dashboard.cache,
		5,   // selectedRepoIdx
		250, // selectedRunIdx
		3,   // selectedDetailLine
		2,   // focusedColumn
	)

	// Set window size
	restoredDashboard.width = 100
	restoredDashboard.height = 50

	// Verify state was properly restored
	assert.Equal(t, 5, restoredDashboard.selectedRepoIdx, "selectedRepoIdx should be restored")
	assert.Equal(t, 250, restoredDashboard.selectedRunIdx, "selectedRunIdx should be restored")
	assert.Equal(t, 3, restoredDashboard.selectedDetailLine, "selectedDetailLine should be restored")
	assert.Equal(t, 2, restoredDashboard.focusedColumn, "focusedColumn should be restored")

	// Verify viewports were initialized safely
	assert.NotNil(t, restoredDashboard.repoViewport, "repoViewport should be initialized")
	assert.NotNil(t, restoredDashboard.runsViewport, "runsViewport should be initialized")
	assert.NotNil(t, restoredDashboard.detailsViewport, "detailsViewport should be initialized")

	// Verify viewports have safe initial content
	assert.Greater(t, len(restoredDashboard.repoViewport.View()), 0, "repoViewport should have content")
	assert.Greater(t, len(restoredDashboard.runsViewport.View()), 0, "runsViewport should have content")
	assert.Greater(t, len(restoredDashboard.detailsViewport.View()), 0, "detailsViewport should have content")
}

// TestViewportPanicPrevention tests that viewport operations don't panic with invalid states
func TestViewportPanicPrevention(t *testing.T) {
	dashboard := createTestDashboard(t)

	// Test high selectedRunIdx with empty content
	dashboard.selectedRunIdx = 500
	dashboard.filteredRuns = []*models.RunResponse{} // Empty
	dashboard.runsViewport = viewport.New(50, 20)
	dashboard.runsViewport.YOffset = 500 // Very high offset

	// This should not panic
	assert.NotPanics(t, func() {
		dashboard.updateRunsViewportContent()
	}, "updateRunsViewportContent should not panic with high YOffset and empty content")

	// Test with minimal content
	dashboard.filteredRuns = []*models.RunResponse{
		{ID: "1", Title: "Single Run", Status: models.StatusDone},
	}
	dashboard.selectedRepo = &models.Repository{Name: "test-repo"}
	dashboard.selectedRunIdx = 500 // Still high

	assert.NotPanics(t, func() {
		dashboard.updateRunsViewportContent()
	}, "updateRunsViewportContent should not panic and should clamp index")

	// Verify index was clamped
	assert.Equal(t, 0, dashboard.selectedRunIdx, "selectedRunIdx should be clamped to 0 for single item")
}

// TestRenderSafetyChecks tests that render functions handle invalid viewport states safely
func TestRenderSafetyChecks(t *testing.T) {
	dashboard := createTestDashboard(t)
	dashboard.width = 100
	dashboard.height = 50

	// Set up viewport with problematic state
	dashboard.runsViewport = viewport.New(50, 20)
	dashboard.runsViewport.SetContent("Line 1\n") // Minimal content
	dashboard.runsViewport.YOffset = 100         // Way beyond content

	// This should not panic and should return valid content
	assert.NotPanics(t, func() {
		content := dashboard.renderRunsColumn(30, 20)
		assert.Greater(t, len(content), 0, "renderRunsColumn should return non-empty content")
	}, "renderRunsColumn should handle invalid viewport state safely")

	// Test with zero dimensions
	dashboard.runsViewport = viewport.New(0, 0)
	assert.NotPanics(t, func() {
		content := dashboard.renderRunsColumn(30, 20)
		assert.Contains(t, content, "Loading", "Should show loading fallback for invalid viewport")
	}, "renderRunsColumn should handle zero-dimension viewport safely")
}

// Helper function to create a test dashboard
func createTestDashboard(t *testing.T) *DashboardView {
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